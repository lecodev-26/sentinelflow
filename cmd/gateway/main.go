package main

import (
"context"
"encoding/json"
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

// Registrar providers si tienen credenciales
if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
if err := registry.Register(adapters.NewOpenAIAdapter(apiKey, "")); err != nil {
log.Printf("⚠️ Error registrando OpenAI: %v", err)
} else {
log.Printf("✅ Provider registrado: openai")
}
} else {
log.Printf("⚠️ OPENAI_API_KEY no configurada, OpenAI no disponible")
}

if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
if err := registry.Register(adapters.NewAnthropicAdapter(apiKey, "")); err != nil {
log.Printf("⚠️ Error registrando Anthropic: %v", err)
} else {
log.Printf("✅ Provider registrado: anthropic")
}
} else {
log.Printf("⚠️ ANTHROPIC_API_KEY no configurada, Anthropic no disponible")
}

// Ollama siempre intenta (aunque no esté corriendo, fallará en health)
if err := registry.Register(adapters.NewOllamaAdapter("")); err != nil {
log.Printf("⚠️ Error registrando Ollama: %v", err)
} else {
log.Printf("✅ Provider registrado: ollama (puede no estar disponible)")
}

// Manager con health + circuit breaker
manager := providers.NewManager(registry)
manager.Start(context.Background())
defer manager.Stop()

log.Printf("📊 Providers registrados: %d", registry.Count())

// Services
identitySvc := identity.NewService(pgClient)
authEnabled := os.Getenv("SENTINELFLOW_ENV") == "production"

// Middleware
authMw := middleware.NewAuth(identitySvc, authEnabled)
idempotencyMw := middleware.NewIdempotency(24 * time.Hour)
normalizerSvc := normalizer.New()

// Router
r := mux.NewRouter()

// Health
r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(`{"status":"ok","service":"sentinelflow-gateway","version":"` + version.String() + `","auth_enabled":` + boolToStr(authEnabled) + `,"providers":` + intToStr(registry.Count()) + `}`))
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

// Chat completions
chatHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
body, err := io.ReadAll(req.Body)
if err != nil {
writeError(w, http.StatusBadRequest, "invalid_request", "error reading body")
return
}
defer req.Body.Close()

// Normalizar
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

// Convertir a ChatRequest V3
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

// Seleccionar provider (simple: por modelo)
provider := selectProvider(manager, normReq.Model)
if provider == nil {
writeError(w, http.StatusServiceUnavailable, "no_provider", "no provider available for this model")
return
}

logger.Infof("🎯 Provider seleccionado: %s", provider.ID())

// Health/CB check
cb := manager.GetBreaker(provider.ID())
if !cb.Allow() {
writeError(w, http.StatusServiceUnavailable, "circuit_open", "circuit breaker open")
return
}

// Ejecutar (no streaming por ahora)
if normReq.Stream {
writeError(w, http.StatusNotImplemented, "not_implemented", "streaming coming in V3.3 step 8")
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
WriteTimeout: 60 * time.Second,
IdleTimeout:  60 * time.Second,
}

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("✅ Gateway en http://localhost:8080")
log.Printf("   Auth: %v", authEnabled)
log.Printf("   Providers: %d", registry.Count())
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

// selectProvider elige un provider según el modelo
func selectProvider(manager *providers.Manager, model string) providers.Provider {
available := manager.AvailableProviders()

// Preferencias por modelo
if len(available) == 0 {
return nil
}

// gpt-* → OpenAI
if len(model) >= 3 && model[:3] == "gpt" {
for _, p := range available {
if p.ID() == "openai" {
return p
}
}
}

// claude-* → Anthropic
if len(model) >= 6 && model[:6] == "claude" {
for _, p := range available {
if p.ID() == "anthropic" {
return p
}
}
}

// llava, llama, mistral, qwen → Ollama
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

// Fallback: primer provider disponible
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

func boolToStr(b bool) string {
if b {
return "true"
}
return "false"
}

func intToStr(n int) string {
return json.Number(json.Number(time.Now().Format("05"))).String()[:0] + intToString(n)
}

func intToString(n int) string {
if n == 0 {
return "0"
}
var result []byte
for n > 0 {
result = append([]byte{byte('0' + n%10)}, result...)
n /= 10
}
return string(result)
}
