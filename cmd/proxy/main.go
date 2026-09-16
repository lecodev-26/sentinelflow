package main

import (
"context"
"flag"
"log"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/gorilla/mux"
"github.com/lecodev-26/sentinelflow/internal/gateway"
"github.com/lecodev-26/sentinelflow/internal/metrics"
"github.com/lecodev-26/sentinelflow/internal/proxy"
"github.com/lecodev-26/sentinelflow/internal/ratelimit"
"github.com/lecodev-26/sentinelflow/internal/rbac"
)

func main() {
port := flag.String("port", "8080", "Puerto del proxy")
metricsPort := flag.String("metrics-port", "9090", "Puerto para métricas")
configFile := flag.String("config", "configs/rules.yaml", "Archivo de configuración")
flag.Parse()

log.Printf("🛡️ SentinelFlow iniciando en puerto %s", *port)
log.Printf("📋 Configuración: %s", *configFile)

p, err := proxy.NewProxy(*configFile)
if err != nil {
log.Fatalf("❌ Error creando proxy: %v", err)
}

// Configurar RBAC
orgMgr := rbac.NewOrganizationManager()
userMgr := rbac.NewUserManager(orgMgr)

// Crear organización y usuario por defecto para demo
defaultOrg := orgMgr.CreateOrganization("default", "Default Organization")
defaultUser, err := userMgr.CreateUser("admin@local", "Admin", rbac.RoleAdmin, defaultOrg.ID)
if err != nil {
log.Fatalf("❌ Error creando usuario: %v", err)
}

// Crear API key para demo
rawKey, _, err := userMgr.CreateAPIKey(defaultUser.ID, "default-key", "")
if err != nil {
log.Fatalf("❌ Error creando API key: %v", err)
}
log.Printf("🔑 API key demo: %s", rawKey)

limiter := ratelimit.NewLimiter(100, time.Minute)
limits := gateway.NewLimitMiddleware(gateway.DefaultLimits())
auth := gateway.NewAuthMiddleware(userMgr, false) // false = auth desactivado para demo

// Pipeline de middlewares
pipeline := gateway.NewPipeline().
Use(gateway.ContextMiddleware()).
Use(metrics.MetricsMiddleware).
Use(limits.Handler).
Use(auth.Handler).
Use(func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
ip := r.RemoteAddr
if !limiter.Allow(ip) {
gateway.WriteError(w, gateway.NewRateLimitedError("too many requests"))
return
}
next.ServeHTTP(w, r)
})
})

mainHandler := pipeline.Then(p.Handler())

// Router final
finalRouter := mux.NewRouter()
finalRouter.HandleFunc("/health", p.HealthCheck)
finalRouter.PathPrefix("/dashboard").Handler(
http.StripPrefix("/dashboard", http.FileServer(http.Dir("./web/dashboard"))),
)
finalRouter.PathPrefix("/demo").Handler(
http.StripPrefix("/demo", http.FileServer(http.Dir("./web/demo"))),
)
finalRouter.HandleFunc("/api/providers", p.GetProvidersStatus).Methods("GET")
finalRouter.HandleFunc("/api/logs/stream", p.StreamLogs).Methods("GET")
finalRouter.PathPrefix("/").Handler(mainHandler)

srv := &http.Server{
Addr:         ":" + *port,
Handler:      finalRouter,
ReadTimeout:  30 * time.Second,
WriteTimeout: 30 * time.Second,
IdleTimeout:  60 * time.Second,
}

metricsSrv := &http.Server{
Addr:    ":" + *metricsPort,
Handler: metrics.Handler(),
}

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

go func() {
log.Printf("✅ Proxy en http://localhost:%s", *port)
log.Printf("📊 Dashboard en http://localhost:%s/dashboard", *port)
log.Printf("🎨 Demo en http://localhost:%s/demo", *port)
log.Printf("📈 Métricas en http://localhost:%s/metrics", *metricsPort)
log.Printf("🔒 Rate Limiting: 100 req/min por IP")
log.Printf("🚧 Request limits: 1MB body, 16KB headers")
log.Printf("🔐 RBAC: organización + usuario + API key configurados")
if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("❌ Error: %v", err)
}
}()

go func() {
if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Printf("⚠️ Error en métricas: %v", err)
}
}()

<-stop
log.Println("🔄 Apagando...")

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

srv.Shutdown(ctx)
metricsSrv.Shutdown(ctx)
p.Stop()

log.Println("✅ SentinelFlow detenido correctamente")
}
