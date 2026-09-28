package security

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/idgen"
	"github.com/sirupsen/logrus"
)

type Consumer struct{ pool *pgxpool.Pool }

func NewConsumer(pool *pgxpool.Pool) *Consumer { return &Consumer{pool: pool} }
func (c *Consumer) Register(bus events.EventBus) error {
	for _, t := range []string{events.EventSecurityBlocked, events.EventPIIDetected, events.EventSecretDetected, events.EventInjectionDetected} {
		if err := bus.Subscribe(t, events.NewConsumer("security."+t, c.handle)); err != nil {
			return err
		}
	}
	return nil
}
func (c *Consumer) handle(ctx context.Context, ev events.Event) {
	kind := kindFromEvent(ev.Type)
	environment := str(ev.Payload["environment"])
	if environment == "" {
		environment = "production"
	}
	severity := str(ev.Payload["severity"])
	if severity == "" {
		severity = "medium"
	}
	payloadJSON, _ := json.Marshal(ev.Payload)
	const q = `
INSERT INTO security_events (
id, event_id, kind, severity,
tenant_id, environment, project_id, user_id, request_id, trace_id,
detector, rule, snippet, action, payload, occurred_at, recorded_at
) VALUES (
$1, $2, $3, $4,
$5, $6, $7, $8, $9, $10,
$11, $12, $13, $14, $15, $16, NOW()
)
ON CONFLICT (event_id) DO NOTHING`
	_, err := c.pool.Exec(ctx, q, "sec_"+idgen.RandomHex(8), ev.ID, kind, severity, ev.TenantID, environment, ev.ProjectID, ev.UserID, ev.RequestID, ev.TraceID, str(ev.Payload["detector"]), str(ev.Payload["rule"]), str(ev.Payload["snippet"]), str(ev.Payload["action"]), payloadJSON, ev.Timestamp)
	if err != nil {
		logrus.Errorf("security: insert failed event=%s kind=%s: %v", ev.ID, kind, err)
		return
	}
	logrus.WithFields(logrus.Fields{"event_id": ev.ID, "kind": kind, "severity": severity, "tenant": ev.TenantID, "environment": environment}).Info("🛡️ security event persistido")
}
func kindFromEvent(t string) string {
	switch t {
	case events.EventSecurityBlocked:
		return "blocked"
	case events.EventPIIDetected:
		return "pii"
	case events.EventSecretDetected:
		return "secret"
	case events.EventInjectionDetected:
		return "prompt_injection"
	}
	return "unknown"
}
func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
