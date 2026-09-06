package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Total de peticiones
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_requests_total",
			Help: "Total number of requests processed",
		},
		[]string{"method", "path", "provider", "status"},
	)

	// Latencia de peticiones
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sentinel_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "provider"},
	)

	// Fallos de proveedores
	ProviderFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_provider_failures_total",
			Help: "Total number of provider failures",
		},
		[]string{"provider"},
	)

	// Fallbacks ejecutados
	FallbacksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_fallbacks_total",
			Help: "Total number of fallbacks executed",
		},
		[]string{"from", "to"},
	)

	// Caché hits y misses
	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sentinel_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sentinel_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	// Rate Limit hits
	RateLimitHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sentinel_ratelimit_hits_total",
			Help: "Total number of rate limit hits",
		},
	)

	// Smart Routing decisiones
	SmartRoutingDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_smart_routing_total",
			Help: "Total number of smart routing decisions",
		},
		[]string{"model", "provider"},
	)
)

// MetricsMiddleware es un middleware que registra métricas
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		provider := r.Header.Get("X-Provider")
		if provider == "" {
			provider = "unknown"
		}

		RequestsTotal.WithLabelValues(r.Method, r.URL.Path, provider, http.StatusText(rw.statusCode)).Inc()
		RequestDuration.WithLabelValues(r.Method, r.URL.Path, provider).Observe(duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Handler devuelve el handler para Prometheus
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordProviderFailure registra un fallo de proveedor
func RecordProviderFailure(provider string) {
	ProviderFailures.WithLabelValues(provider).Inc()
}

// RecordFallback registra un fallback
func RecordFallback(from, to string) {
	FallbacksTotal.WithLabelValues(from, to).Inc()
}

// RecordCacheHit registra un acierto de caché
func RecordCacheHit() {
	CacheHits.Inc()
}

// RecordCacheMiss registra un fallo de caché
func RecordCacheMiss() {
	CacheMisses.Inc()
}

// RecordRateLimit registra un rate limit hit
func RecordRateLimit() {
	RateLimitHits.Inc()
}

// RecordSmartRouting registra una decisión de smart routing
func RecordSmartRouting(model, provider string) {
	SmartRoutingDecisions.WithLabelValues(model, provider).Inc()
}