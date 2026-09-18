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
"github.com/lecodev-26/sentinelflow/internal/audit"
"github.com/lecodev-26/sentinelflow/internal/controlplane"
"github.com/lecodev-26/sentinelflow/internal/cost"
"github.com/lecodev-26/sentinelflow/internal/gateway"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/metrics"
"github.com/lecodev-26/sentinelflow/internal/observability"
"github.com/lecodev-26/sentinelflow/internal/proxy"
"github.com/lecodev-26/sentinelflow/internal/ratelimit"
"github.com/lecodev-26/sentinelflow/internal/rbac"
)

const Version = "2.3.0"

func main() {
port := flag.String("port", "8080", "Puerto del proxy")
adminPort := flag.String("admin-port", "8081", "Puerto del control plane")
metricsPort := flag.String("metrics-port", "9090", "Puerto para métricas")
configFile := flag.String("config", "configs/rules.yaml", "Archivo de configuración")
flag.Parse()

log.Printf("🛡️ SentinelFlow v%s", Version)
log.Printf("   Gateway:       :%s", *port)
log.Printf("   Control Plane: :%s", *adminPort)
log.Printf("   Métricas:      :%s", *metricsPort)

if err := observability.InitTracing("sentinelflow", ""); err != nil {
log.Printf("⚠️ Tracing no disponible: %v", err)
}

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

rawKey, _, err := userMgr.CreateAPIKey(defaultUser.ID, "default-key", "", nil, 0)
if err != nil {
log.Fatalf("❌ Error creando API key: %v", err)
}
log.Printf("🔑 API key demo: %s", rawKey)

// Cost tracker
costTracker := cost.NewCostTracker()
costTracker.SetBudget("default", 100.0)
costTracker.SetAlertCallback(func(tenantID string, threshold int, budget *cost.Budget) {
logger.Warnf("🚨 BUDGET ALERT: tenant=%s threshold=%d%% used=$%.2f/%.2f",
tenantID, threshold, budget.Used, budget.MonthlyLimit)
})

// Audit: suscribir al bus de eventos
audit.SubscribeAll(&audit.LoggerWriter{})
logger.Info("📝 Audit system iniciado")

// Middlewares
limiter := ratelimit.NewLimiter(100, time.Minute)
limits := gateway.NewLimitMiddleware(gateway.DefaultLimits())

auth, err := gateway.NewAuthMiddlewareFromEnv(userMgr)
if err != nil {
log.Fatalf("❌ Error configurando auth: %v", err)
}

security := gateway.NewSecurityMiddleware(false)
cacheMw := gateway.NewCacheMiddleware(true, 5*time.Minute)
quotaMw := gateway.NewQuotaMiddleware(true)
quotaMw.SetQuota("default", gateway.DefaultQuota())
costMw := gateway.NewCostMiddleware(costTracker, true)
obsMw := gateway.NewObservabilityMiddleware(false)

// Pipeline completo
pipeline := gateway.NewPipeline().
Use(gateway.ContextMiddleware()).
Use(obsMw.Handler).
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

// === GATEWAY ROUTER ===
gatewayRouter := mux.NewRouter()
gatewayRouter.HandleFunc("/health", p.HealthCheck)
gatewayRouter.PathPrefix("/dashboard").Handler(
http.StripPrefix("/dashboard", http.FileServer(http.Dir("./web/dashboard"))),
)
gatewayRouter.PathPrefix("/demo").Handler(
http.StripPrefix("/demo", http.FileServer(http.Dir("./web/demo"))),
)
gatewayRouter.PathPrefix("/").Handler(mainHandler)

// === CONTROL PLANE ROUTER ===
adminRouter := mux.NewRouter()
adminRouter.Use(corsMiddleware)

orgHandler := controlplane.NewOrganizationHandler(orgMgr)
userHandler := controlplane.NewUserHandler(userMgr)
providerHandler := controlplane.NewProviderHandler(p)
metricsHandler := controlplane.NewMetricsHandler(p)

cpRouter := controlplane.NewRouter(orgHandler, userHandler, providerHandler, metricsHandler)
cpRouter.Register(adminRouter)

// Servidores
gatewaySrv := &http.Server{
Addr:         ":" + *port,
Handler:      gatewayRouter,
ReadTimeout:  30 * time.Second,
WriteTimeout: 30 * time.Second,
IdleTimeout:  60 * time.Second,
}

adminSrv := &http.Server{
Addr:         ":" + *adminPort,
Handler:      adminRouter,
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
log.Printf("✅ Gateway en http://localhost:%s", *port)
if err := gatewaySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("❌ Gateway error: %v", err)
}
}()

go func() {
log.Printf("✅ Control Plane en http://localhost:%s", *adminPort)
log.Printf("   API v1: http://localhost:%s/v1/...", *adminPort)
log.Printf("   Legacy: http://localhost:%s/admin/... (deprecated)", *adminPort)
if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Fatalf("❌ Admin error: %v", err)
}
}()

go func() {
log.Printf("✅ Métricas en http://localhost:%s/metrics", *metricsPort)
if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
log.Printf("⚠️ Error en métricas: %v", err)
}
}()

<-stop
log.Println("🔄 Apagando...")

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

gatewaySrv.Shutdown(ctx)
adminSrv.Shutdown(ctx)
metricsSrv.Shutdown(ctx)
p.Stop()

log.Println("✅ SentinelFlow detenido correctamente")
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
