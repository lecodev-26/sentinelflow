// Package audit persiste un log inmutable de acciones importantes.
//
// Modelo:
//   - El gateway (o cualquier productor) emite eventos audit.* al Outbox.
//   - El worker consume esos eventos desde el bus y hace INSERT en audit_log.
//   - audit_log es append-only: nunca UPDATE, nunca DELETE.
//
// El consumer es idempotente: índice único en event_id evita duplicados
// cuando el bus reentrega (at-least-once).
package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/idgen"
)

// Consumer persiste eventos audit.* en la tabla audit_log.
type Consumer struct {
	pool *pgxpool.Pool
}

// NewConsumer crea el consumer de audit.
func NewConsumer(pool *pgxpool.Pool) *Consumer {
	return &Consumer{pool: pool}
}

// Register suscribe el consumer a todos los eventos audit.*.
func (c *Consumer) Register(bus events.EventBus) error {
	types := []string{
		events.EventAPIKeyCreated,
		events.EventAPIKeyRevoked,
		events.EventPolicyChanged,
		events.EventUserCreated,
		events.EventUserDeleted,
	}
	for _, t := range types {
		if err := bus.Subscribe(t, events.NewConsumer("audit."+t, c.handle)); err != nil {
			return err
		}
	}
	return nil
}

// handle procesa cualquier evento audit.* y lo persiste en audit_log.
func (c *Consumer) handle(ctx context.Context, ev events.Event) {
	resourceType, resourceID := extractResource(ev)
	payloadJSON, _ := json.Marshal(ev.Payload)

	const q = `
INSERT INTO audit_log (
id, event_id, action,
actor_id, actor_email,
tenant_id, project_id,
resource_type, resource_id,
ip, user_agent,
request_id, trace_id,
payload, occurred_at, recorded_at
) VALUES (
$1, $2, $3,
$4, $5,
$6, $7,
$8, $9,
$10, $11,
$12, $13,
$14, $15, NOW()
)
ON CONFLICT (event_id) DO NOTHING
`

	_, err := c.pool.Exec(ctx, q,
		"audit_"+idgen.RandomHex(8),
		ev.ID,
		ev.Type,
		ev.UserID,
		str(ev.Payload["actor_email"]),
		ev.TenantID,
		ev.ProjectID,
		resourceType,
		resourceID,
		str(ev.Payload["ip"]),
		str(ev.Payload["user_agent"]),
		ev.RequestID,
		ev.TraceID,
		payloadJSON,
		ev.Timestamp,
	)
	if err != nil {
		logrus.Errorf("audit: insert failed event=%s action=%s: %v", ev.ID, ev.Type, err)
		return
	}
	logrus.WithFields(logrus.Fields{
		"event_id": ev.ID,
		"action":   ev.Type,
		"tenant":   ev.TenantID,
	}).Debug("📜 audit: evento persistido")
}

// extractResource deduce tipo e ID del recurso a partir del payload.
func extractResource(ev events.Event) (string, string) {
	switch ev.Type {
	case events.EventAPIKeyCreated, events.EventAPIKeyRevoked:
		return "api_key", str(ev.Payload["api_key_id"])
	case events.EventPolicyChanged:
		return "policy", str(ev.Payload["policy_id"])
	case events.EventUserCreated, events.EventUserDeleted:
		return "user", str(ev.Payload["user_id"])
	}
	return "unknown", ""
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
