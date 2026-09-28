package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/lecodev-26/sentinelflow/internal/events"
)

type Consumer struct{ pool *pgxpool.Pool }

func NewConsumer(pool *pgxpool.Pool) *Consumer { return &Consumer{pool: pool} }

func (c *Consumer) HandleUsageRecorded(ctx context.Context, ev events.Event) {
	tenantID := ev.TenantID
	if tenantID == "" {
		logrus.Warn("analytics: evento sin tenant_id, ignorado")
		return
	}
	environment := strPayload(ev, "environment")
	if environment == "" {
		environment = "production"
	}
	provider := strPayload(ev, "provider")
	model := strPayload(ev, "model")
	status := strPayload(ev, "status")
	inputTokens := int(int64Payload(ev, "input_tokens"))
	outputTokens := int(int64Payload(ev, "output_tokens"))
	costUSD := floatPayload(ev, "cost_usd")

	logrus.WithFields(logrus.Fields{"event_id": ev.ID, "tenant_id": tenantID, "environment": environment, "provider": provider, "model": model, "input_tokens": inputTokens, "output_tokens": outputTokens, "cost_usd": costUSD, "status": status}).Info("📊 analytics: usage.recorded")

	day := ev.Timestamp.UTC().Format("2006-01-02")
	errors := 0
	if status == "error" {
		errors = 1
	}

	const q = `
INSERT INTO analytics_daily (
tenant_id, environment, day, requests,
input_tokens, output_tokens, total_tokens,
cost_usd, errors, last_updated
) VALUES (
$1, $2, $3::date, 1,
$4, $5, $6,
$7, $8, NOW()
)
ON CONFLICT (tenant_id, environment, day) DO UPDATE SET
requests      = analytics_daily.requests + EXCLUDED.requests,
input_tokens  = analytics_daily.input_tokens + EXCLUDED.input_tokens,
output_tokens = analytics_daily.output_tokens + EXCLUDED.output_tokens,
total_tokens  = analytics_daily.total_tokens + EXCLUDED.total_tokens,
cost_usd      = analytics_daily.cost_usd + EXCLUDED.cost_usd,
errors        = analytics_daily.errors + EXCLUDED.errors,
last_updated  = NOW()
`
	_, err := c.pool.Exec(ctx, q, tenantID, environment, day, inputTokens, outputTokens, inputTokens+outputTokens, costUSD, errors)
	if err != nil {
		logrus.Errorf("analytics: upsert failed: %v", err)
	}
}

func strPayload(ev events.Event, key string) string {
	if v, ok := ev.Payload[key].(string); ok {
		return v
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

func (c *Consumer) Register(bus events.EventBus) error {
	if err := bus.Subscribe(events.EventUsageRecorded, events.NewConsumer("analytics.usage.recorded", c.HandleUsageRecorded)); err != nil {
		return fmt.Errorf("analytics: subscribe usage.recorded: %w", err)
	}
	return nil
}

var _ = time.Now
