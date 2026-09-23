// Package security consumer persiste los eventos security.* en security_events.
package security

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/idgen"
)

// Consumer persiste eventos security.* en la DB.
type Consumer struct {
	pool *pgxpool.Pool
}

// NewConsumer crea el consumer.
func NewConsumer(pool *pgxpool.Pool) *Consumer {
	return &Consumer{pool: pool}
}

// Register suscribe el consumer a todos los eventos security.*.
func (c *Consumer) Register(bus events.EventBus) error {
	types := []string{
		events.EventSecurityBlocked,
		events.EventPIIDetected,
		events.EventSecretDetected,
		events.EventInjectionDetected,
	}
	for _, t := range types {
		if err := bus.Subscribe(t, events.NewConsumer("security."+t, c.handle)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Consumer) handle(ctx context.Context, ev events.Event) {
	kind := kindFromEvent(ev.Type)
	severity := str(ev.Payload["severity"])
	if severity == "" {
		severity = "medium"
	}
	detector := str(ev.Payload["detector"])
	rule := str(ev.Payload["rule"])
	snippet := str(ev.Payload["snippet"])
	action := str(ev.Payload["action"])

	payloadJSON, _ := json.Marshal(ev.Payload)

	const q = `
INSERT INTO security_events (
id, event_id, kind, severity,
tenant_id, project_id, user_id, request_id, trace_id,
detector, rule, snippet, action,
payload, occurred_at, recorded_at
) VALUES (
$1, $2, $3, $4,
$5, $6, $7, $8, $9,
$10, $11, $12, $13,
$14, $15, NOW()
)
ON CONFLICT (event_id) DO NOTHING
`
	_, err := c.pool.Exec(ctx, q,
		"sec_"+idgen.RandomHex(8),
		ev.ID, kind, severity,
		ev.TenantID, ev.ProjectID, ev.UserID, ev.RequestID, ev.TraceID,
		detector, rule, snippet, action,
		payloadJSON, ev.Timestamp,
	)
	if err != nil {
		logrus.Errorf("security: insert failed event=%s kind=%s: %v", ev.ID, kind, err)
		return
	}
	logrus.WithFields(logrus.Fields{
		"event_id": ev.ID,
		"kind":     kind,
		"severity": severity,
		"tenant":   ev.TenantID,
	}).Info("🛡️ security event persistido")
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
