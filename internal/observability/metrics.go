package observability

import (
"time"

"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
// TTFT - Time To First Token
TTFT = promauto.NewHistogram(
prometheus.HistogramOpts{
Name:    "sentinelflow_ttft_seconds",
Help:    "Time to first token in seconds",
Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
},
)

// Tokens por modelo
TokensTotal = promauto.NewCounterVec(
prometheus.CounterOpts{
Name: "sentinelflow_tokens_total",
Help: "Total tokens processed",
},
[]string{"provider", "model", "type"},
)

// Coste por proveedor
CostTotal = promauto.NewCounterVec(
prometheus.CounterOpts{
Name: "sentinelflow_cost_usd_total",
Help: "Total cost in USD",
},
[]string{"provider", "model"},
)

// Peticiones activas
ActiveRequests = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "sentinelflow_active_requests",
Help: "Number of active requests",
},
)

// Errores por tipo
ErrorsTotal = promauto.NewCounterVec(
prometheus.CounterOpts{
Name: "sentinelflow_errors_total",
Help: "Total errors by type",
},
[]string{"type", "provider"},
)

// Proveedores offline
ProviderStatus = promauto.NewGaugeVec(
prometheus.GaugeOpts{
Name: "sentinelflow_provider_status",
Help: "Provider status (1=online, 0=offline)",
},
[]string{"provider"},
)
)

// RecordTTFT registra el tiempo hasta el primer token
func RecordTTFT(duration time.Duration) {
TTFT.Observe(duration.Seconds())
}

// RecordTokens registra el uso de tokens
func RecordTokens(provider, model, tokenType string, count int) {
TokensTotal.WithLabelValues(provider, model, tokenType).Add(float64(count))
}

// RecordCost registra el coste
func RecordCost(provider, model string, cost float64) {
CostTotal.WithLabelValues(provider, model).Add(cost)
}

// RecordActiveRequest incrementa/decrementa peticiones activas
func RecordActiveRequest(inc bool) {
if inc {
ActiveRequests.Inc()
} else {
ActiveRequests.Dec()
}
}

// RecordError registra un error
func RecordError(errorType, provider string) {
ErrorsTotal.WithLabelValues(errorType, provider).Inc()
}

// SetProviderStatus establece el estado de un proveedor
func SetProviderStatus(provider string, online bool) {
if online {
ProviderStatus.WithLabelValues(provider).Set(1)
} else {
ProviderStatus.WithLabelValues(provider).Set(0)
}
}
