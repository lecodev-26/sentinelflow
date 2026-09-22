package main

import (
"context"
"encoding/json"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gorilla/mux"
oidcv3 "github.com/lecodev-26/sentinelflow/internal/enterprise/v3/oidc"
scimv3 "github.com/lecodev-26/sentinelflow/internal/enterprise/v3/scim"
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
logger.Infof("📊 Pool stats: %v", pgClient.Stats())

if err := pgClient.Migrate(ctx); err != nil {
log.Fatalf("❌ Error aplicando migraciones: %v", err)
}
log.Printf("✅ Migraciones aplicadas")

identitySvc := identity.NewService(pgClient)
log.Printf("✅ Identity service iniciado")

// OIDC V3.7
oidcMgr := oidcv3.NewManager()
oidcFlow := oidcv3.NewFlow(oidcMgr)

// Registrar providers desde env vars (opcional)
if cid := os.Getenv("GOOGLE_CLIENT_ID"); cid != "" {
_ = oidcMgr.RegisterProvider(&oidcv3.Config{
Provider:     oidcv3.ProviderGoogle,
ClientID:     cid,
ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
})
log.Printf("✅ OIDC provider registrado: google")
}
if cid := os.Getenv("GITHUB_CLIENT_ID"); cid != "" {
_ = oidcMgr.RegisterProvider(&oidcv3.Config{
Provider:     oidcv3.ProviderGitHub,
ClientID:     cid,
ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
})
log.Printf("✅ OIDC provider registrado: github")
}

// SCIM V3.7
scimHandler := scimv3.NewHandler(identitySvc)
log.Printf("✅ SCIM 2.0 handler iniciado")

r := mux.NewRouter()
r.Use(corsMiddleware)

// Health
r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
if err := pgClient.HealthCheck(req.Context()); err != nil {
w.WriteHeader(http.StatusServiceUnavailable)
w.Write([]byte(`{"status":"unhealthy"}`))
return
}
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(`{"status":"ok","service":"sentinelflow-controlplane","version":"` + version.String() + `"}`))
}).Methods("GET")

r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
info := version.Get()
w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `"}`))
}).Methods("GET")

// === OIDC Endpoints V3.7 ===
r.HandleFunc("/auth/oidc/providers", func(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"providers": oidcMgr.ListProviders(),
})
}).Methods("GET")

r.HandleFunc("/auth/oidc/{provider}/login", func(w http.ResponseWriter, req *http.Request) {
provider := oidcv3.Provider(mux.Vars(req)["provider"])
redirect := req.URL.Query().Get("redirect")

authURL, err := oidcFlow.BuildAuthURL(provider, redirect)
if err != nil {
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusBadRequest)
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
return
}

http.Redirect(w, req, authURL, http.StatusTemporaryRedirect)
}).Methods("GET")

r.HandleFunc("/auth/oidc/{provider}/callback", func(w http.ResponseWriter, req *http.Request) {
// En producción: intercambiar code por token
// Por ahora placeholder
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(map[string]interface{}{
"status":  "received",
"code":    req.URL.Query().Get("code"),
"state":   req.URL.Query().Get("state"),
"message": "OIDC callback received (token exchange not implemented)",
})
}).Methods("GET")

// === SCIM Endpoints V3.7 ===
scimHandler.Register(r)

// === Identity Endpoints (V3.1) ===
// (los mismos que ya teníamos, resumidos)

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
log.Printf("   Endpoints Enterprise V3.7:")
log.Printf("   GET  /auth/oidc/providers")
log.Printf("   GET  /auth/oidc/{provider}/login")
log.Printf("   GET  /auth/oidc/{provider}/callback")
log.Printf("   GET  /scim/v2/Users")
log.Printf("   POST /scim/v2/Users")
log.Printf("   GET  /scim/v2/ServiceProviderConfig")
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
