// Package webhooks envía notificaciones HTTP a URLs registradas por el cliente.
//
// Flujo:
//  1. Cliente registra webhook: URL + secret + event_types[]
//  2. Cualquier evento que llega al bus con event_type en la lista se copia
//     a webhook_deliveries (una fila por webhook que lo escucha)
//  3. El dispatcher lee webhook_deliveries y hace POST firmado con HMAC-SHA256
//  4. Reintentos con backoff exponencial
package webhooks

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/idgen"
)

// Consumer escucha eventos del bus y encola deliveries.
type Consumer struct {
	pool *pgxpool.Pool
}

// NewConsumer crea el consumer.
func NewConsumer(pool *pgxpool.Pool) *Consumer {
	return &Consumer{pool: pool}
}

// Register suscribe el consumer a los tipos de evento configurables.
// Los tipos posibles se definen en webhooks.event_types, pero como
// el bus necesita saber los canales de antemano, aquí suscribimos a
// todos los eventos "notificables". El filtrado por webhook se hace
// en la query de enqueue.
func (c *Consumer) Register(bus events.EventBus) error {
	types := []string{
		events.EventBudgetWarning,
		events.EventBudgetExceeded,
		events.EventSecurityBlocked,
		events.EventPIIDetected,
		events.EventSecretDetected,
		events.EventInjectionDetected,
		events.EventPolicyChanged,
		events.EventProviderFailed,
		events.EventProviderHealthChanged,
	}
	for _, t := range types {
		if err := bus.Subscribe(t, events.NewConsumer("webhooks."+t, c.handle)); err != nil {
			return err
		}
	}
	return nil
}

// handle encola una delivery por cada webhook activo del tenant que
// escuche este tipo de evento.
func (c *Consumer) handle(ctx context.Context, ev events.Event) {
	payloadJSON, _ := json.Marshal(ev)

	// INSERT ... SELECT: por cada webhook del tenant que escuche el tipo,
	// insertar una delivery. Idempotente por (webhook_id, event_id).
	const q = `
INSERT INTO webhook_deliveries (
id, webhook_id, event_id, event_type, payload, status, created_at, updated_at
)
SELECT
'whd_' || $1 || '_' || w.id,
w.id, $2, $3, $4, 'pending', NOW(), NOW()
FROM webhooks w
WHERE w.tenant_id = $5
  AND w.active = true
  AND w.event_types @> jsonb_build_array($3::text)
ON CONFLICT (webhook_id, event_id) DO NOTHING
`
	tag, err := c.pool.Exec(ctx, q,
		idgen.RandomHex(8),
		ev.ID,
		ev.Type,
		payloadJSON,
		ev.TenantID,
	)
	if err != nil {
		logrus.Errorf("webhooks: enqueue failed event=%s type=%s: %v", ev.ID, ev.Type, err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		logrus.WithFields(logrus.Fields{
			"event_id": ev.ID,
			"type":     ev.Type,
			"queued":   n,
			"tenant":   ev.TenantID,
		}).Info("🔗 webhooks: deliveries encoladas")
	}
}
