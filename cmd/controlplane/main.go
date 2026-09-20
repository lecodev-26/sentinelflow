package main

import (
"context"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gorilla/mux"
"github.com/lecodev-26/sentinelflow/internal/controlplane/v3"
"github.com/lecodev-26/sentinelflow/internal/identity"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
log.Printf("🛡️ SentinelFlow Control Plane v%s", version.Full())

logger.Init(&struct {
Level  string
Format string
Output string
}{
Level:  "info",
Format: "json",
Output: "stdout",
})

// Cargar DATABASE_URL
dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
if dbURL == "" {
log.Fatalf("❌ SENTINELFLOW_DATABASE_URL is required")
}

// Conectar a PostgreSQL
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

pgClient, err := postgres.New(ctx, postgres.DefaultConfig(dbURL))
if err != nil {
log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
}
defer pgClient.Close()

log.Printf("✅ PostgreSQL conectado")
logger.Infof("📊 Pool stats: %v", pgClient.Stats())

// Aplicar migraciones
if err := pgClient.Migrate(ctx); err != nil {
log.Fatalf("❌ Error aplicando migraciones: %v", err)
}
log.Printf("✅ Migraciones aplicadas")

// Iniciar servicio de Identity
identitySvc := identity.NewService(pgClient)
log.Printf("✅ Identity service iniciado")

// Router
r := mux.NewRouter()
r.Use(corsMiddleware)

// Health
r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
if err := pgClient.HealthCheck(req.Context()); err != nil {
w.WriteHeader(http.StatusServiceUnavailable)
w.Write([]byte(`{"status":"unhealthy","error":"` + err.Error() + `"}`))
return
}
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(`{"status":"ok","service":"sentinelflow-controlplane","version":"` + version.String() + `"}`))
}).Methods("GET")

// Version
r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
info := version.Get()
w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `"}`))
}).Methods("GET")

// Registrar handlers de identity
handlers := v3.New(identitySvc)
handlers.Register(r)

// Servidor
srv := &http.Server{
Addr:         ":8081",
Handler:      r,
ReadTimeout:  30 * time.Second,
WriteTimeout: 30 * time.Second,
IdleTimeout:  60 * time.Second,
}

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("✅ Control Plane en http://localhost:8081")
log.Printf("")
log.Printf("📋 Endpoints disponibles:")
log.Printf("   GET    /health")
log.Printf("   GET    /version")
log.Printf("   GET    /v1/organizations")
log.Printf("   POST   /v1/organizations")
log.Printf("   GET    /v1/organizations/{id}")
log.Printf("   DELETE /v1/organizations/{id}")
log.Printf("   GET    /v1/organizations/{id}/projects")
log.Printf("   POST   /v1/organizations/{id}/projects")
log.Printf("   GET    /v1/projects/{id}")
log.Printf("   DELETE /v1/projects/{id}")
log.Printf("   GET    /v1/organizations/{id}/users")
log.Printf("   POST   /v1/users")
log.Printf("   GET    /v1/users/{id}")
log.Printf("   DELETE /v1/users/{id}")
log.Printf("   GET    /v1/users/{id}/api-keys")
log.Printf("   POST   /v1/users/{id}/api-keys")
log.Printf("   DELETE /v1/api-keys/{id}")
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("❌ Control Plane error: %v", err)
}
}()

<-stop
log.Println("🔄 Apagando...")

shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()

srv.Shutdown(shutdownCtx)
log.Println("✅ Control Plane detenido")
}

func corsMiddleware(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
if r.Method == "OPTIONS" {
w.WriteHeader(http.StatusOK)
return
}
next.ServeHTTP(w, r)
})
}
