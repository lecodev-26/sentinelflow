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

limiter := ratelimit.NewLimiter(100, time.Minute)
limits := gateway.NewLimitMiddleware(gateway.DefaultLimits())

r := mux.NewRouter()

r.Use(metrics.MetricsMiddleware)

r.Use(func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
ip := r.RemoteAddr
if !limiter.Allow(ip) {
http.Error(w, "Too many requests", http.StatusTooManyRequests)
return
}
next.ServeHTTP(w, r)
})
})

// ✅ CORREGIDO: usar p.Handler().ServeHTTP
r.Handle("/", limits.Handler(p.Handler().ServeHTTP))
r.HandleFunc("/health", p.HealthCheck)

r.PathPrefix("/dashboard").Handler(
http.StripPrefix("/dashboard", http.FileServer(http.Dir("./web/dashboard"))),
)
r.PathPrefix("/demo").Handler(
http.StripPrefix("/demo", http.FileServer(http.Dir("./web/demo"))),
)

r.HandleFunc("/api/providers", p.GetProvidersStatus).Methods("GET")
r.HandleFunc("/api/logs/stream", p.StreamLogs).Methods("GET")

srv := &http.Server{
Addr:         ":" + *port,
Handler:      r,
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
log.Printf("🔒 Rate Limiting activo: 100 req/min por IP")
log.Printf("🚧 Request limits: 1MB body, 16KB headers")
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
