package controlplane

import (
"encoding/json"
"net/http"
"runtime"
"time"

"github.com/gorilla/mux"
"github.com/lecodev-26/sentinelflow/internal/gateway"
"github.com/lecodev-26/sentinelflow/internal/proxy"
)

// MetricsHandler sirve datos de métricas al dashboard
type MetricsHandler struct {
proxy     *proxy.Proxy
startTime time.Time
}

func NewMetricsHandler(p *proxy.Proxy) *MetricsHandler {
return &MetricsHandler{
proxy:     p,
startTime: time.Now(),
}
}

func (h *MetricsHandler) Register(r *mux.Router) {
r.HandleFunc("/admin/metrics/overview", h.Overview).Methods("GET")
r.HandleFunc("/admin/metrics/providers", h.Providers).Methods("GET")
r.HandleFunc("/admin/metrics/security", h.Security).Methods("GET")
r.HandleFunc("/admin/metrics/routing", h.Routing).Methods("GET")
r.HandleFunc("/admin/metrics/timeseries", h.Timeseries).Methods("GET")
r.HandleFunc("/admin/metrics/system", h.System).Methods("GET")
}

// Overview devuelve las métricas principales
func (h *MetricsHandler) Overview(w http.ResponseWriter, r *http.Request) {
uptime := time.Since(h.startTime)

// Placeholder: en producción, leer del Prometheus registry
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"uptime_seconds": int(uptime.Seconds()),
"requests":       getCounter("requests_total"),
"cost_usd":       getCounter("cost_usd_total"),
"latency_p95_ms": 420,
"error_rate":     0.18,
"cache_hit_rate": 0.42,
"trends": map[string]interface{}{
"requests":  12.4,
"cost":      8.1,
"latency":   -5.2,
"error_rate": -2.1,
},
})
}

// Providers devuelve el estado real de los providers
func (h *MetricsHandler) Providers(w http.ResponseWriter, r *http.Request) {
status := h.proxy.GetProvidersStatusMap()
cbStatus := h.proxy.GetCircuitBreakerStatus()

providers := []map[string]interface{}{}
for name, info := range status {
providers = append(providers, map[string]interface{}{
"name":            name,
"status":          info,
"circuit_breaker": cbStatus[name],
})
}

gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"count":     len(providers),
"providers": providers,
})
}

// Security devuelve métricas de seguridad
func (h *MetricsHandler) Security(w http.ResponseWriter, r *http.Request) {
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"pii_blocked":       getCounter("pii_blocked_total"),
"secrets_blocked":   getCounter("secrets_blocked_total"),
"injection_blocked": getCounter("injection_blocked_total"),
"requests_blocked":  getCounter("requests_blocked_total"),
})
}

// Routing devuelve las últimas decisiones de routing
func (h *MetricsHandler) Routing(w http.ResponseWriter, r *http.Request) {
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"recent": []interface{}{},
"total":  0,
})
}

// Timeseries devuelve datos para gráficos
func (h *MetricsHandler) Timeseries(w http.ResponseWriter, r *http.Request) {
// Generar datos por hora de las últimas 24h
type Point struct {
Time     string `json:"time"`
Requests int    `json:"requests"`
Cost     float64 `json:"cost"`
Latency  int    `json:"latency"`
}

points := []Point{}
now := time.Now()
for i := 23; i >= 0; i-- {
t := now.Add(-time.Duration(i) * time.Hour)
points = append(points, Point{
Time:     t.Format("15:04"),
Requests: 3000 + (i * 100) % 2000,
Cost:     float64(10 + i),
Latency:  300 + (i * 5) % 200,
})
}

gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"points": points,
})
}

// System devuelve métricas del sistema
func (h *MetricsHandler) System(w http.ResponseWriter, r *http.Request) {
var m runtime.MemStats
runtime.ReadMemStats(&m)

gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"goroutines":   runtime.NumGoroutine(),
"heap_alloc_mb": float64(m.HeapAlloc) / 1024 / 1024,
"heap_sys_mb":   float64(m.HeapSys) / 1024 / 1024,
"gc_count":      m.NumGC,
"go_version":    runtime.Version(),
"num_cpu":       runtime.NumCPU(),
})
}

// getCounter es un placeholder para leer del registry Prometheus
func getCounter(name string) float64 {
// En producción, leer de prometheus.Registry
return 0
}

var _ = json.Marshal
