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
"github.com/lecodev-26/sentinelflow/internal/cost"
"github.com/lecodev-26/sentinelflow/internal/gateway"
"github.com/lecodev-26/sentinelflow/internal/logger"
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

p, err := proxy.NewProxy(*configFile)
if err != nil {
log.Fatalf("❌ Error creando proxy: %v", err)
}

// RBAC
orgMgr := rbac.NewOrganizationManager()
userMgr := rbac.NewUserManager(orgMgr)

defaultOrg := orgMgr.CreateOrganization("default", "Default Organization")
defaultUser, err := userMgr.CreateUser("admin@local", "Admin", rbac.RoleAdmin, defaultOrg.ID)
if err != nil {
log.Fatalf("❌ Error creando usuario: %v", err)
}

rawKey, _, err := userMgr.CreateAPIKey(defaultUser.ID, "default-key", "")
if err != nil {
log.Fatalf("❌ Error creando API key: %v", err)
}
log.Printf("🔑 API key demo: %s", rawKey)

// Cost tracker
costTracker := cost.NewCostTracker()
costTracker.SetBudget("default", 100.0) // $100/mes para el tenant default
costTracker.SetAlertCallback(func(tenantID string, threshold int, budget *cost.Budget) {
logger.Warnf("🚨 BUDGET ALERT: tenant=%s threshold=%d%% used=$%.2f/%.2f",
tenantID, threshold, budget.Used, budget.MonthlyLimit)
})

// Middlewares
limiter := ratelimit.NewLimiter(100, time.Minute)
limits := gateway.NewLimitMiddleware(gateway.DefaultLimits())
auth := gateway.NewAuthMiddleware(userMgr, false)
security := gateway.NewSecurityMiddleware(false)
cacheMw := gateway.NewCacheMiddleware(true, 5*time.Minute)
quotaMw := gateway.NewQuotaMiddleware(true)
quotaMw.SetQuota("default", gateway.DefaultQuota())
costMw := gateway.NewCostMiddleware(costTracker, true)

// Pipeline completo
pipeline := gateway.NewPipeline().
Use(gateway.ContextMiddleware()).
Use(metrics.MetricsMiddleware).
Use(limits.Handler).
Use(auth.Handler).
Use(security.Handler).
Use(func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if !limiter.Allow(r.RemoteAddr) {
gateway.WriteError(w, gateway.NewRateLimitedError("too many requests"))
return
}
next.ServeHTTP(w, r)
})
}).
Use(quotaMw.Handler).
Use(cacheMw.Handler).
Use(costMw.Handler)

mainHandler := pipeline.Then(p.Handler())

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
log.Printf("🔗 Pipeline: context → metrics → limits → auth → security → ratelimit → quota → cache → cost → engine")
log.Printf("💰 Budget: $100/mes para tenant 'default'")
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
