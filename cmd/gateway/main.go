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
"github.com/lecodev-26/sentinelflow/internal/gateway/v3/middleware"
"github.com/lecodev-26/sentinelflow/internal/gateway/v3/normalizer"
"github.com/lecodev-26/sentinelflow/internal/identity"
"github.com/lecodev-26/sentinelflow/internal/logger"
providers "github.com/lecodev-26/sentinelflow/internal/providers/v3"
"github.com/lecodev-26/sentinelflow/internal/providers/v3/adapters"
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

// PostgreSQL
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

// Provider Registry V3
registry := providers.NewRegistry()

if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
_ = registry.Register(adapters.NewOpenAIAdapter(apiKey, ""))
log.Printf("✅ Provider registrado: openai")
} else {
log.Printf("⚠️ OPENAI_API_KEY no configurada")
}

if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
_ = registry.Register(adapters.NewAnthropicAdapter(apiKey, ""))
log.Printf("✅ Provider registrado: anthropic")
} else {
log.Printf("⚠️ ANTHROPIC_API_KEY no configurada")
}

_ = registry.Register(adapters.NewOllamaAdapter(""))
log.Printf("✅ Provider registrado: ollama")

manager := providers.NewManager(registry)
manager.Start(context.Background())
defer manager.Stop()

log.Printf("📊 Providers registrados: %d", registry.Count())

identitySvc := identity.NewService(pgClient)
authEnabled := os.Getenv("SENTINELFLOW_ENV") == "production"

authMw := middleware.NewAuth(identitySvc, authEnabled)
idempotencyMw := middleware.NewIdempotency(24 * time.Hour)
normalizerSvc := normalizer.New()

r := mux.NewRouter()

// Health
r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(fmt.Sprintf(`{"status":"ok","service":"sentinelflow-gateway","version":"%s","auth_enabled":%v,"providers":%d}`,
version.String(), authEnabled, registry.Count())))
}).Methods("GET")

// Version
r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
info := version.Get()
w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `"}`))
}).Methods("GET")

// Providers status
r.HandleFunc("/v1/providers", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"providers": manager.HealthStatus(),
})
}).Methods("GET")

// Chat handler (con soporte streaming)
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

provider := selectProvider(manager, normReq.Model)
if provider == nil {
writeError(w, http.StatusServiceUnavailable, "no_provider", "no provider available")
return
}

cb := manager.GetBreaker(provider.ID())
if !cb.Allow() {
writeError(w, http.StatusServiceUnavailable, "circuit_open", "circuit breaker open")
return
}

// === STREAMING ===
if normReq.Stream {
handleStream(w, req, provider, chatReq, cb, tenantID, userID, apiKeyID)
return
}

// === NON-STREAMING ===
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
w.Header().Set("X-Tenant-Id", tenantID)
w.Header().Set("X-User-Id", userID)
w.Header().Set("X-Api-Key-Id", apiKeyID)
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(resp)
})

r.Handle("/v1/chat/completions",
authMw.Handler(idempotencyMw.Handler(chatHandler)),
).Methods("POST")

srv := &http.Server{
Addr:         ":8080",
Handler:      r,
ReadTimeout:  30 * time.Second,
WriteTimeout: 0, // 0 para streaming
IdleTimeout:  60 * time.Second,
}

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("✅ Gateway en http://localhost:8080")
log.Printf("   Auth: %v", authEnabled)
log.Printf("   Streaming: enabled (SSE)")
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

// handleStream maneja el streaming SSE real
func handleStream(
w http.ResponseWriter,
req *http.Request,
provider providers.Provider,
chatReq *providers.ChatRequest,
cb *providers.CircuitBreaker,
tenantID, userID, apiKeyID string,
) {
// Verificar que el ResponseWriter soporta Flusher
flusher, ok := w.(http.Flusher)
if !ok {
writeError(w, http.StatusInternalServerError, "streaming_not_supported", "streaming not supported")
return
}

// Iniciar stream del provider
start := time.Now()
chunks, err := provider.Stream(req.Context(), chatReq)
if err != nil {
cb.RecordFailure()
logger.Errorf("❌ Stream error: %v", err)
writeError(w, http.StatusBadGateway, "provider_error", err.Error())
return
}

// Configurar headers SSE
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")
w.Header().Set("X-Accel-Buffering", "no")
w.Header().Set("X-Provider", provider.ID())
w.Header().Set("X-Tenant-Id", tenantID)
w.Header().Set("X-User-Id", userID)
w.Header().Set("X-Api-Key-Id", apiKeyID)
w.WriteHeader(http.StatusOK)
flusher.Flush()

var ttft time.Duration
totalChunks := 0

// Procesar chunks
for chunk := range chunks {
// Verificar contexto del cliente
select {
case <-req.Context().Done():
logger.Warnf("⚠️ Client disconnected after %d chunks", totalChunks)
cb.RecordSuccess() // parcialmente OK
return
default:
}

// Primer chunk → calcular TTFT
if totalChunks == 0 {
ttft = time.Since(start)
logger.Infof("⚡ TTFT: %dms", ttft.Milliseconds())
}

// Serializar como SSE
data, err := json.Marshal(map[string]interface{}{
"id":      "chatcmpl-stream",
"object":  "chat.completion.chunk",
"created": time.Now().Unix(),
"model":   chatReq.Model,
"choices": []map[string]interface{}{
{
"index": chunk.Index,
"delta": map[string]string{
"content": chunk.Delta,
},
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

// Enviar [DONE]
fmt.Fprintf(w, "data: [DONE]\n\n")
flusher.Flush()

cb.RecordSuccess()
latency := time.Since(start)
logger.Infof("✅ Stream completed: %d chunks, TTFT=%dms, total=%dms",
totalChunks, ttft.Milliseconds(), latency.Milliseconds())
}

// selectProvider elige un provider según el modelo
func selectProvider(manager *providers.Manager, model string) providers.Provider {
available := manager.AvailableProviders()
if len(available) == 0 {
return nil
}

if len(model) >= 3 && model[:3] == "gpt" {
for _, p := range available {
if p.ID() == "openai" {
return p
}
}
}

if len(model) >= 6 && model[:6] == "claude" {
for _, p := range available {
if p.ID() == "anthropic" {
return p
}
}
}

localPrefixes := []string{"llama", "mistral", "qwen", "phi", "gemma", "llava"}
for _, prefix := range localPrefixes {
if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
for _, p := range available {
if p.ID() == "ollama" {
return p
}
}
}
}

return available[0]
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
