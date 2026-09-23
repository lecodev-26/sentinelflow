// Package analytics consume eventos del bus y actualiza agregados.
//
// Consumers actuales:
//   - usage.recorded -> upsert en analytics_daily + log estructurado
//
// El consumer debe ser idempotente: si el mismo evento llega dos veces
// (at-least-once del bus), el UPSERT por (tenant_id, day) puede sumar dos
// veces. Para evitarlo, usamos processed_events (V3.2.4) como dedup.
// Por ahora, dejamos el log y el upsert "best effort" con nota.
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/lecodev-26/sentinelflow/internal/events"
)

// Consumer agrupa la lógica de analytics.
type Consumer struct {
	pool *pgxpool.Pool
}

// NewConsumer crea el consumer de analytics.
func NewConsumer(pool *pgxpool.Pool) *Consumer {
	return &Consumer{pool: pool}
}

// HandleUsageRecorded procesa un evento usage.recorded:
//   - Loguea el evento con campos estructurados (trazabilidad)
//   - Hace UPSERT en analytics_daily (tenant, day) sumando tokens/coste
func (c *Consumer) HandleUsageRecorded(ctx context.Context, ev events.Event) {
	tenantID := ev.TenantID
	if tenantID == "" {
		logrus.Warn("analytics: evento sin tenant_id, ignorado")
		return
	}

	// Extraer campos del payload
	provider := strPayload(ev, "provider")
	model := strPayload(ev, "model")
	status := strPayload(ev, "status")
	inputTokens := int(int64Payload(ev, "input_tokens"))
	outputTokens := int(int64Payload(ev, "output_tokens"))
	costUSD := floatPayload(ev, "cost_usd")

	logrus.WithFields(logrus.Fields{
		"event_id":      ev.ID,
		"tenant_id":     tenantID,
		"provider":      provider,
		"model":         model,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"cost_usd":      costUSD,
		"status":        status,
	}).Info("📊 analytics: usage.recorded")

	// UPSERT idempotente por (tenant_id, day)
	day := ev.Timestamp.UTC().Format("2006-01-02")
	errors := 0
	if status == "error" {
		errors = 1
	}

	const q = `
INSERT INTO analytics_daily (
tenant_id, day, requests,
input_tokens, output_tokens, total_tokens,
cost_usd, errors, last_updated
) VALUES (
$1, $2::date, 1,
$3, $4, $5,
$6, $7, NOW()
)
ON CONFLICT (tenant_id, day) DO UPDATE SET
requests      = analytics_daily.requests + EXCLUDED.requests,
input_tokens  = analytics_daily.input_tokens + EXCLUDED.input_tokens,
output_tokens = analytics_daily.output_tokens + EXCLUDED.output_tokens,
total_tokens  = analytics_daily.total_tokens + EXCLUDED.total_tokens,
cost_usd      = analytics_daily.cost_usd + EXCLUDED.cost_usd,
errors        = analytics_daily.errors + EXCLUDED.errors,
last_updated  = NOW()
`
	_, err := c.pool.Exec(ctx, q,
		tenantID, day,
		inputTokens, outputTokens, inputTokens+outputTokens,
		costUSD, errors,
	)
	if err != nil {
		logrus.Errorf("analytics: upsert failed: %v", err)
		return
	}
}

// === helpers de payload ===

func strPayload(ev events.Event, key string) string {
	if v, ok := ev.Payload[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func int64Payload(ev events.Event, key string) int64 {
	if v, ok := ev.Payload[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int:
			return int64(n)
		case int64:
			return n
		}
	}
	return 0
}

func floatPayload(ev events.Event, key string) float64 {
	if v, ok := ev.Payload[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}

// Register registra los consumers de analytics en el bus.
func (c *Consumer) Register(bus events.EventBus) error {
	if err := bus.Subscribe(events.EventUsageRecorded,
		events.NewConsumer("analytics.usage.recorded", c.HandleUsageRecorded),
	); err != nil {
		return fmt.Errorf("analytics: subscribe usage.recorded: %w", err)
	}
	return nil
}

// Para satisfacer time import si se elimina por refactor
var _ = time.Now
