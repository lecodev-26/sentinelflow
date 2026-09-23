// Package metrics expone métricas Prometheus para SentinelFlow.
//
// Todo el código registra aquí sus métricas y el gateway expone /metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// === Outbox ===
	OutboxEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinelflow_outbox_events_total",
			Help: "Total de eventos del outbox por tipo y estado final",
		},
		[]string{"type", "status"},
	)

	OutboxPending = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sentinelflow_outbox_pending",
			Help: "Eventos pendientes o fallidos actualmente en el outbox",
		},
	)

	OutboxPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sentinelflow_outbox_publish_duration_seconds",
			Help:    "Duración de publicación de un evento del outbox al bus",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	OutboxRecovered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sentinelflow_outbox_recovered_total",
			Help: "Eventos recuperados de estado 'publishing' colgado",
		},
	)

	// === Gateway HTTP ===
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinelflow_http_requests_total",
			Help: "Total de requests HTTP por método, ruta y estado",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sentinelflow_http_request_duration_seconds",
			Help:    "Duración de requests HTTP",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// === Accounting ===
	UsageRecorded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinelflow_usage_recorded_total",
			Help: "Registros de uso por tenant y provider",
		},
		[]string{"tenant", "provider", "status"},
	)

	CostUSD = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinelflow_cost_usd_total",
			Help: "Coste acumulado en USD por tenant",
		},
		[]string{"tenant"},
	)

	// === Analytics ===
	AnalyticsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinelflow_analytics_processed_total",
			Help: "Eventos procesados por el consumer analytics",
		},
		[]string{"type", "status"},
	)

	// === Audit ===
	AuditPersisted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinelflow_audit_persisted_total",
			Help: "Eventos de audit persistidos",
		},
		[]string{"action"},
	)
)
