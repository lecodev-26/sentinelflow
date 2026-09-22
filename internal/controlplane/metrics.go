package controlplane

import (
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/gateway"
	"github.com/lecodev-26/sentinelflow/internal/observability/analytics"
	"github.com/lecodev-26/sentinelflow/internal/observability/traces"
	"github.com/lecodev-26/sentinelflow/internal/proxy"
)

// MetricsHandler sirve datos de métricas al dashboard
type MetricsHandler struct {
	proxy         *proxy.Proxy
	startTime     time.Time
	traceStore    *traces.Store
	providerStats *analytics.ProviderAnalytics
	routingStats  *analytics.RoutingAnalytics
	costStats     *analytics.CostAnalytics
}

func NewMetricsHandler(p *proxy.Proxy, ts *traces.Store, ps *analytics.ProviderAnalytics, rs *analytics.RoutingAnalytics, cs *analytics.CostAnalytics) *MetricsHandler {
	return &MetricsHandler{
		proxy:         p,
		startTime:     time.Now(),
		traceStore:    ts,
		providerStats: ps,
		routingStats:  rs,
		costStats:     cs,
	}
}

func (h *MetricsHandler) Register(r *mux.Router) {
	r.HandleFunc("/admin/metrics/overview", h.Overview).Methods("GET")
	r.HandleFunc("/admin/metrics/providers", h.Providers).Methods("GET")
	r.HandleFunc("/admin/metrics/security", h.Security).Methods("GET")
	r.HandleFunc("/admin/metrics/routing", h.Routing).Methods("GET")
	r.HandleFunc("/admin/metrics/timeseries", h.Timeseries).Methods("GET")
	r.HandleFunc("/admin/metrics/system", h.System).Methods("GET")
	r.HandleFunc("/admin/metrics/providers/analytics", h.ProviderAnalytics).Methods("GET")
	r.HandleFunc("/admin/metrics/routing/analytics", h.RoutingAnalytics).Methods("GET")
	r.HandleFunc("/admin/metrics/costs", h.Costs).Methods("GET")
	r.HandleFunc("/admin/traces", h.ListTraces).Methods("GET")
	r.HandleFunc("/admin/traces/{id}", h.GetTrace).Methods("GET")
}

func (h *MetricsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)
	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"uptime_seconds": int(uptime.Seconds()),
		"requests":       0,
		"cost_usd":       h.costStats.Snapshot().Total,
		"latency_p95_ms": 420,
		"error_rate":     0.18,
		"cache_hit_rate": 0.42,
		"traces_count":   h.traceStore.Size(),
		"trends": map[string]interface{}{
			"requests":   12.4,
			"cost":       8.1,
			"latency":    -5.2,
			"error_rate": -2.1,
		},
	})
}

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

func (h *MetricsHandler) Security(w http.ResponseWriter, r *http.Request) {
	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"pii_blocked":       0,
		"secrets_blocked":   0,
		"injection_blocked": 0,
		"requests_blocked":  0,
	})
}

func (h *MetricsHandler) Routing(w http.ResponseWriter, r *http.Request) {
	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"recent": h.routingStats.ListRecent(20),
		"total":  h.routingStats.Size(),
	})
}

func (h *MetricsHandler) Timeseries(w http.ResponseWriter, r *http.Request) {
	type Point struct {
		Time     string  `json:"time"`
		Requests int     `json:"requests"`
		Cost     float64 `json:"cost"`
		Latency  int     `json:"latency"`
	}

	points := []Point{}
	now := time.Now()
	for i := 23; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * time.Hour)
		points = append(points, Point{
			Time:     t.Format("15:04"),
			Requests: 3000 + (i*100)%2000,
			Cost:     float64(10 + i),
			Latency:  300 + (i*5)%200,
		})
	}

	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{"points": points})
}

func (h *MetricsHandler) System(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"goroutines":    runtime.NumGoroutine(),
		"heap_alloc_mb": float64(m.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":   float64(m.HeapSys) / 1024 / 1024,
		"gc_count":      m.NumGC,
		"go_version":    runtime.Version(),
		"num_cpu":       runtime.NumCPU(),
	})
}

// === V2.7 ENDPOINTS ===

// ProviderAnalytics devuelve las estadísticas agregadas por provider
func (h *MetricsHandler) ProviderAnalytics(w http.ResponseWriter, r *http.Request) {
	stats := h.providerStats.GetAll()

	result := []map[string]interface{}{}
	for _, s := range stats {
		result = append(result, map[string]interface{}{
			"provider":       s.Provider,
			"requests":       s.TotalRequests,
			"success":        s.SuccessCount,
			"errors":         s.ErrorCount,
			"success_rate":   s.SuccessRate(),
			"avg_latency_ms": s.AvgLatency().Milliseconds(),
			"min_latency_ms": s.MinLatency.Milliseconds(),
			"max_latency_ms": s.MaxLatency.Milliseconds(),
			"total_tokens":   s.TotalTokens,
			"total_cost":     s.TotalCost,
			"last_used":      s.LastUsed,
		})
	}

	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{"providers": result})
}

// RoutingAnalytics devuelve las estadísticas de routing
func (h *MetricsHandler) RoutingAnalytics(w http.ResponseWriter, r *http.Request) {
	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"recent":          h.routingStats.ListRecent(50),
		"selection_count": h.routingStats.SelectionCounts(),
		"total":           h.routingStats.Size(),
	})
}

// Costs devuelve las estadísticas de costes
func (h *MetricsHandler) Costs(w http.ResponseWriter, r *http.Request) {
	gateway.WriteJSON(w, http.StatusOK, h.costStats.Snapshot())
}

// ListTraces lista los traces recientes
func (h *MetricsHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	tenantID := r.URL.Query().Get("tenant")

	var tracesList []*traces.Trace
	if tenantID != "" {
		tracesList = h.traceStore.ListByTenant(tenantID, limit)
	} else {
		tracesList = h.traceStore.ListTraces(limit)
	}

	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"total":  h.traceStore.Size(),
		"traces": tracesList,
	})
}

// GetTrace devuelve un trace completo
func (h *MetricsHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	trace, ok := h.traceStore.GetTrace(id)
	if !ok {
		trace, ok = h.traceStore.GetTraceByRequest(id)
	}
	if !ok {
		gateway.WriteError(w, gateway.NewInvalidRequestError("trace not found"))
		return
	}

	gateway.WriteJSON(w, http.StatusOK, trace)
}

// Models devuelve todos los modelos registrados (OpenAI-compatible)
func (h *MetricsHandler) Models(w http.ResponseWriter, r *http.Request) {
	models := h.proxy.GetModels()

	data := make([]map[string]interface{}, 0, len(models))
	for _, m := range models {
		data = append(data, map[string]interface{}{
			"id":       m.ID,
			"object":   "model",
			"created":  m.DiscoveredAt.Unix(),
			"owned_by": m.Provider,
		})
	}

	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}
