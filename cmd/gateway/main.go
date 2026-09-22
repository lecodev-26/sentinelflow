package main

import (
"context"
"encoding/json"
"fmt"
"io"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gorilla/mux"
accountingv3 "github.com/lecodev-26/sentinelflow/internal/accounting/v3"
"github.com/lecodev-26/sentinelflow/internal/gateway/v3/middleware"
"github.com/lecodev-26/sentinelflow/internal/gateway/v3/normalizer"
"github.com/lecodev-26/sentinelflow/internal/identity"
"github.com/lecodev-26/sentinelflow/internal/logger"
policyv3 "github.com/lecodev-26/sentinelflow/internal/policy/v3"
providers "github.com/lecodev-26/sentinelflow/internal/providers/v3"
"github.com/lecodev-26/sentinelflow/internal/providers/v3/adapters"
routing "github.com/lecodev-26/sentinelflow/internal/routing/v3"
"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
log.Printf("🛡️ SentinelFlow Gateway v%s", version.Full())

logger.Init(&struct {
Level  string
Format string
Output string
}{
Level:  "info",
Format: "json",
Output: "stdout",
})

dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
if dbURL == "" {
log.Fatalf("❌ SENTINELFLOW_DATABASE_URL is required")
}

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

pgClient, err := postgres.New(ctx, postgres.DefaultConfig(dbURL))
if err != nil {
log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
}
defer pgClient.Close()
log.Printf("✅ PostgreSQL conectado")

// Migraciones (aplica las nuevas tablas FinOps)
if err := pgClient.Migrate(ctx); err != nil {
log.Printf("⚠️ Error aplicando migraciones: %v", err)
} else {
log.Printf("✅ Migraciones aplicadas")
}

// Provider Registry
registry := providers.NewRegistry()

if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
_ = registry.Register(adapters.NewOpenAIAdapter(apiKey, ""))
log.Printf("✅ Provider registrado: openai")
}
if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
_ = registry.Register(adapters.NewAnthropicAdapter(apiKey, ""))
log.Printf("✅ Provider registrado: anthropic")
}
_ = registry.Register(adapters.NewOllamaAdapter(""))
log.Printf("✅ Provider registrado: ollama")

manager := providers.NewManager(registry)
manager.Start(context.Background())
defer manager.Stop()

// Model Registry + Routing Engine
modelRegistry := routing.NewModelRegistry()
routingEngine := routing.NewEngine(modelRegistry, manager, routing.DefaultWeights())

log.Printf("📊 Providers: %d, Models: %d", registry.Count(), len(modelRegistry.List()))

// Policy Engine
policyEvaluator := policyv3.NewEvaluator()
defaultPolicy := policyv3.NewDefaultPolicy("default")
compiled, err := policyv3.NewCompiler().Compile(defaultPolicy)
if err != nil {
log.Fatalf("❌ Error compilando policy: %v", err)
}
policyEvaluator.Register(compiled)
log.Printf("📋 Policy Engine: %d políticas", len(policyEvaluator.List()))

// Accounting Service V3.6
accountingSvc := accountingv3.NewService(pgClient.Usage(), pgClient.Budgets(), modelRegistry)
accountingSvc.OnBudgetAlert(func(alert accountingv3.BudgetAlert) {
logger.Warnf("🚨 BUDGET ALERT: tenant=%s threshold=%d%% spent=$%.2f/%.2f",
alert.TenantID, alert.Threshold, alert.Spent, alert.Limit)
})
log.Printf("💰 Accounting service iniciado")

identitySvc := identity.NewService(pgClient)
authEnabled := os.Getenv("SENTINELFLOW_ENV") == "production"

authMw := middleware.NewAuth(identitySvc, authEnabled)
idempotencyMw := middleware.NewIdempotency(24 * time.Hour)
policyMw := middleware.NewPolicyMiddleware(policyEvaluator, true)
accountingMw := middleware.NewAccountingMiddleware(accountingSvc, true)
normalizerSvc := normalizer.New()

r := mux.NewRouter()

r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(fmt.Sprintf(`{"status":"ok","service":"sentinelflow-gateway","version":"%s","auth_enabled":%v,"providers":%d,"models":%d}`,
version.String(), authEnabled, registry.Count(), len(modelRegistry.List()))))
}).Methods("GET")

r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
info := version.Get()
w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `"}`))
}).Methods("GET")

r.HandleFunc("/v1/providers", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"providers": manager.HealthStatus(),
})
}).Methods("GET")

r.HandleFunc("/v1/models", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
models := modelRegistry.List()
data := make([]map[string]interface{}, 0, len(models))
for _, m := range models {
data = append(data, map[string]interface{}{
"id":       m.ID,
"object":   "model",
"provider": m.Provider,
"owned_by": m.Provider,
"context":  m.ContextSize,
"pricing":  map[string]float64{"input": m.InputPer1M, "output": m.OutputPer1M},
})
}
json.NewEncoder(w).Encode(map[string]interface{}{
"object": "list",
"data":   data,
})
}).Methods("GET")

// FinOps endpoints V3.6
r.HandleFunc("/v1/usage/stats", func(w http.ResponseWriter, req *http.Request) {
tenantID := req.URL.Query().Get("tenant")
if tenantID == "" {
tenantID = "default"
}

daysStr := req.URL.Query().Get("days")
days := 30
if daysStr != "" {
fmt.Sscanf(daysStr, "%d", &days)
}

stats, err := accountingSvc.Stats(req.Context(), tenantID, time.Duration(days)*24*time.Hour)
if err != nil {
writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
return
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(stats)
}).Methods("GET")

chatHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
body, err := io.ReadAll(req.Body)
if err != nil {
writeError(w, http.StatusBadRequest, "invalid_request", "error reading body")
return
}
defer req.Body.Close()

normReq, format, err := normalizerSvc.Normalize(body)
if err != nil {
writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
return
}

tenantID := middleware.GetTenantID(req.Context())
userID := middleware.GetUserID(req.Context())
apiKeyID := middleware.GetAPIKeyID(req.Context())

logger.Infof("📨 Request: tenant=%s user=%s format=%s model=%s stream=%v",
tenantID, userID, format, normReq.Model, normReq.Stream)

routeReq := &routing.Request{
RequestID: req.Header.Get("Idempotency-Key"),
Model:     normReq.Model,
}
if routeReq.RequestID == "" {
routeReq.RequestID = fmt.Sprintf("%d", time.Now().UnixNano())
}
if normReq.Stream {
routeReq.RequiredCapabilities = []string{"stream"}
}

if policyResult, ok := middleware.GetPolicyResult(req.Context()); ok && policyResult.Routing != nil {
routeReq.RequiredCapabilities = append(routeReq.RequiredCapabilities, policyResult.Routing.RequiredCapabilities...)
routeReq.PreferredProvider = policyResult.Routing.PreferredProvider
routeReq.ExcludedProviders = policyResult.Routing.BlockedProviders
routeReq.MaxCost = policyResult.Routing.MaxCostPer1M
}

decision, err := routingEngine.Route(routeReq)
if err != nil {
writeError(w, http.StatusServiceUnavailable, "no_provider", err.Error())
return
}

logger.Infof("🎯 Routing: %s → %s (score=%.3f, reason=%s, filtered=%d)",
normReq.Model, decision.Selected.ProviderID, decision.Score.Total,
decision.Reason, decision.Filtered)

provider, ok := registry.Get(decision.Selected.ProviderID)
if !ok {
writeError(w, http.StatusServiceUnavailable, "provider_not_found", "provider not found")
return
}

chatReq := &providers.ChatRequest{
Model:       normReq.Model,
Temperature: normReq.Temperature,
MaxTokens:   normReq.MaxTokens,
Stream:      normReq.Stream,
}
for _, m := range normReq.Messages {
chatReq.Messages = append(chatReq.Messages, providers.Message{
Role:    m.Role,
Content: m.Content,
})
}

cb := manager.GetBreaker(provider.ID())
if !cb.Allow() {
writeError(w, http.StatusServiceUnavailable, "circuit_open", "circuit breaker open")
return
}

if normReq.Stream {
handleStream(w, req, provider, chatReq, cb, tenantID, userID, apiKeyID, decision)
return
}

resp, err := provider.Chat(req.Context(), chatReq)
if err != nil {
cb.RecordFailure()
logger.Errorf("❌ Provider error: %v", err)
writeError(w, http.StatusBadGateway, "provider_error", err.Error())
return
}

cb.RecordSuccess()

w.Header().Set("Content-Type", "application/json")
w.Header().Set("X-Provider", provider.ID())
w.Header().Set("X-Routing-Score", fmt.Sprintf("%.3f", decision.Score.Total))
w.Header().Set("X-Routing-Reason", decision.Reason)
w.Header().Set("X-Tenant-Id", tenantID)
w.Header().Set("X-User-Id", userID)
w.Header().Set("X-Api-Key-Id", apiKeyID)
if decision.Experiment != "" {
w.Header().Set("X-Experiment", decision.Experiment)
}
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(resp)
})

// Pipeline: auth → idempotency → policy → accounting → chat
r.Handle("/v1/chat/completions",
authMw.Handler(
idempotencyMw.Handler(
policyMw.Handler(
accountingMw.Handler(chatHandler),
),
),
),
).Methods("POST")

srv := &http.Server{
Addr:         ":8080",
Handler:      r,
ReadTimeout:  30 * time.Second,
WriteTimeout: 0,
IdleTimeout:  60 * time.Second,
}

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("✅ Gateway en http://localhost:8080")
log.Printf("   Auth: %v", authEnabled)
log.Printf("   Routing: intelligent")
log.Printf("   Policy: enabled")
log.Printf("   Security: PII + Secrets + Prompt + SSRF")
log.Printf("   Accounting: enabled")
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("❌ Gateway error: %v", err)
}
}()

<-stop
log.Println("🔄 Apagando...")

shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()

srv.Shutdown(shutdownCtx)
log.Println("✅ Gateway detenido")
}

func handleStream(
w http.ResponseWriter,
req *http.Request,
provider providers.Provider,
chatReq *providers.ChatRequest,
cb *providers.CircuitBreaker,
tenantID, userID, apiKeyID string,
decision *routing.Decision,
) {
flusher, ok := w.(http.Flusher)
if !ok {
writeError(w, http.StatusInternalServerError, "streaming_not_supported", "streaming not supported")
return
}

start := time.Now()
chunks, err := provider.Stream(req.Context(), chatReq)
if err != nil {
cb.RecordFailure()
writeError(w, http.StatusBadGateway, "provider_error", err.Error())
return
}

w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")
w.Header().Set("X-Accel-Buffering", "no")
w.Header().Set("X-Provider", provider.ID())
w.Header().Set("X-Routing-Score", fmt.Sprintf("%.3f", decision.Score.Total))
w.Header().Set("X-Routing-Reason", decision.Reason)
w.Header().Set("X-Tenant-Id", tenantID)
w.Header().Set("X-User-Id", userID)
w.Header().Set("X-Api-Key-Id", apiKeyID)
w.WriteHeader(http.StatusOK)
flusher.Flush()

var ttft time.Duration
totalChunks := 0

for chunk := range chunks {
select {
case <-req.Context().Done():
cb.RecordSuccess()
return
default:
}

if totalChunks == 0 {
ttft = time.Since(start)
logger.Infof("⚡ TTFT: %dms", ttft.Milliseconds())
}

data, err := json.Marshal(map[string]interface{}{
"id":      "chatcmpl-stream",
"object":  "chat.completion.chunk",
"created": time.Now().Unix(),
"model":   chatReq.Model,
"choices": []map[string]interface{}{
{
"index":         chunk.Index,
"delta":         map[string]string{"content": chunk.Delta},
"finish_reason": chunk.FinishReason,
},
},
})
if err != nil {
continue
}

fmt.Fprintf(w, "data: %s\n\n", string(data))
flusher.Flush()
totalChunks++

if chunk.FinishReason != "" {
break
}
}

fmt.Fprintf(w, "data: [DONE]\n\n")
flusher.Flush()

cb.RecordSuccess()
logger.Infof("✅ Stream completed: %d chunks, TTFT=%dms, total=%dms",
totalChunks, ttft.Milliseconds(), time.Since(start).Milliseconds())
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(status)
json.NewEncoder(w).Encode(map[string]interface{}{
"error": map[string]string{
"type":    errType,
"message": message,
},
})
}
