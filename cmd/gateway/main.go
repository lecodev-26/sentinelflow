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

// Conectar a PostgreSQL para auth
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
w.Write([]byte(`{"status":"ok","service":"sentinelflow-gateway","version":"` + version.String() + `","auth_enabled":` + boolToStr(authEnabled) + `}`))
}).Methods("GET")

// Version
r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
info := version.Get()
w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `"}`))
}).Methods("GET")

// Chat completions con auth + idempotency + normalizer
chatHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
// Leer body
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

// Extraer tenant del contexto (inyectado por auth middleware)
tenantID := middleware.GetTenantID(req.Context())
userID := middleware.GetUserID(req.Context())
apiKeyID := middleware.GetAPIKeyID(req.Context())

logger.Infof("📨 Request: tenant=%s user=%s format=%s model=%s stream=%v",
tenantID, userID, format, normReq.Model, normReq.Stream)

// TODO V3.3: routing + provider execution
// Por ahora placeholder
_ = normalizerSvc

w.Header().Set("Content-Type", "application/json")
w.Header().Set("X-Request-Format", string(format))
w.Header().Set("X-Tenant-ID", tenantID)
w.Header().Set("X-User-ID", userID)
w.Header().Set("X-API-Key-ID", apiKeyID)
w.WriteHeader(http.StatusOK)

json.NewEncoder(w).Encode(map[string]interface{}{
"error": map[string]string{
"type":    "not_implemented",
"message": "Provider execution coming in V3.3",
},
"received": map[string]interface{}{
"model":          normReq.Model,
"messages_count": len(normReq.Messages),
"stream":         normReq.Stream,
"format":         string(format),
},
})
})

r.Handle("/v1/chat/completions",
authMw.Handler(idempotencyMw.Handler(chatHandler)),
).Methods("POST")

// Servidor
srv := &http.Server{
Addr:         ":8080",
Handler:      r,
ReadTimeout:  30 * time.Second,
WriteTimeout: 30 * time.Second,
IdleTimeout:  60 * time.Second,
}

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("✅ Gateway en http://localhost:8080")
log.Printf("   Auth: %v", authEnabled)
log.Printf("   Pipeline: auth → idempotency → normalizer → [routing → provider]")
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
