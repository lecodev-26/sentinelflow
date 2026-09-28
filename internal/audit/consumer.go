package audit

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
	for _, t := range []string{events.EventAPIKeyCreated, events.EventAPIKeyRevoked, events.EventPolicyChanged, events.EventUserCreated, events.EventUserDeleted} {
		if err := bus.Subscribe(t, events.NewConsumer("audit."+t, c.handle)); err != nil {
			return err
		}
	}
	return nil
}
func (c *Consumer) handle(ctx context.Context, ev events.Event) {
	resourceType, resourceID := extractResource(ev)
	environment := str(ev.Payload["environment"])
	if environment == "" {
		environment = "production"
	}
	payloadJSON, _ := json.Marshal(ev.Payload)
	const q = `
INSERT INTO audit_log (
id, event_id, action, actor_id, actor_email,
tenant_id, environment, project_id, resource_type, resource_id,
ip, user_agent, request_id, trace_id, payload, occurred_at, recorded_at
) VALUES (
$1, $2, $3, $4, $5,
$6, $7, $8, $9, $10,
$11, $12, $13, $14, $15, $16, NOW()
)
ON CONFLICT (event_id) DO NOTHING`
	_, err := c.pool.Exec(ctx, q, "audit_"+idgen.RandomHex(8), ev.ID, ev.Type, ev.UserID, str(ev.Payload["actor_email"]), ev.TenantID, environment, ev.ProjectID, resourceType, resourceID, str(ev.Payload["ip"]), str(ev.Payload["user_agent"]), ev.RequestID, ev.TraceID, payloadJSON, ev.Timestamp)
	if err != nil {
		logrus.Errorf("audit: insert failed event=%s action=%s: %v", ev.ID, ev.Type, err)
		return
	}
	logrus.WithFields(logrus.Fields{"event_id": ev.ID, "action": ev.Type, "tenant": ev.TenantID, "environment": environment}).Debug("📜 audit: evento persistido")
}
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
