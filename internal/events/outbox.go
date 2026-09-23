package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Outbox implementa el patrón Transactional Outbox sobre PostgreSQL.
//
// Separación de responsabilidades:
//   - Writer:    lo usa el código de negocio DENTRO de su transacción.
//   - Publisher: worker que lee la tabla y publica al EventBus.
//   - Consumer:  idempotente, del otro lado del bus (no está aquí).
type Outbox struct {
	pool *pgxpool.Pool
	bus  EventBus

	// Configuración del publisher
	pollInterval time.Duration
	batchSize    int
	maxAttempts  int
	baseBackoff  time.Duration
	maxBackoff   time.Duration
}

// OutboxConfig agrupa la configuración del publisher.
type OutboxConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
}

// DefaultOutboxConfig devuelve valores razonables para producción.
func DefaultOutboxConfig() OutboxConfig {
	return OutboxConfig{
		PollInterval: 500 * time.Millisecond,
		BatchSize:    100,
		MaxAttempts:  10,
		BaseBackoff:  2 * time.Second,
		MaxBackoff:   5 * time.Minute,
	}
}

// NewOutbox crea una instancia de Outbox.
func NewOutbox(pool *pgxpool.Pool, bus EventBus, cfg OutboxConfig) *Outbox {
	return &Outbox{
		pool:         pool,
		bus:          bus,
		pollInterval: cfg.PollInterval,
		batchSize:    cfg.BatchSize,
		maxAttempts:  cfg.MaxAttempts,
		baseBackoff:  cfg.BaseBackoff,
		maxBackoff:   cfg.MaxBackoff,
	}
}

// =============================================================================
// WRITER — se llama DENTRO de la transacción de negocio
// =============================================================================

// EnqueueTx inserta un evento en el outbox usando la TX del llamante.
//
// IMPORTANTE: debe invocarse en la MISMA transacción que el cambio de negocio.
// Si la TX falla, el evento desaparece con ella (correcto).
func (o *Outbox) EnqueueTx(ctx context.Context, tx pgx.Tx, ev Event) error {
	payloadJSON, err := json.Marshal(ev.Payload)
	if err != nil {
		return fmt.Errorf("outbox: marshal payload: %w", err)
	}

	const q = `
INSERT INTO outbox_events (
event_id, event_type,
tenant_id, project_id, user_id, request_id, trace_id,
payload,
status, attempts, next_attempt_at,
occurred_at, created_at
) VALUES (
$1, $2,
NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''),
$8,
'pending', 0, NOW(),
$9, NOW()
)
ON CONFLICT (event_id) DO NOTHING
`

	_, err = tx.Exec(ctx, q,
		ev.ID,
		ev.Type,
		ev.TenantID,
		ev.ProjectID,
		ev.UserID,
		ev.RequestID,
		ev.TraceID,
		payloadJSON,
		ev.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("outbox: enqueue: %w", err)
	}
	return nil
}

// =============================================================================
// PUBLISHER — worker loop
// =============================================================================

// Run arranca el loop del publisher. Bloquea hasta que el ctx se cancele.
func (o *Outbox) Run(ctx context.Context) error {
	logrus.Info("📤 OutboxPublisher iniciado",
		"poll_interval", o.pollInterval,
		"batch_size", o.batchSize)

	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("📤 OutboxPublisher detenido")
			return ctx.Err()
		case <-ticker.C:
			if err := o.drain(ctx); err != nil {
				logrus.Errorf("outbox drain error: %v", err)
			}
		}
	}
}

// drain procesa lotes hasta que no queden pendientes elegibles.
func (o *Outbox) drain(ctx context.Context) error {
	for {
		n, err := o.processBatch(ctx)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
	}
}

// processBatch reclama y publica un lote. Devuelve cuántos procesó.
func (o *Outbox) processBatch(ctx context.Context) (int, error) {
	tx, err := o.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("outbox: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := o.claimBatch(ctx, tx)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, tx.Commit(ctx)
	}

	// Publicar fuera de la TX para no retener el lock durante el I/O del bus.
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("outbox: commit claim: %w", err)
	}

	for _, r := range rows {
		o.publishOne(ctx, r)
	}
	return len(rows), nil
}

// outboxRow es la proyección que lee el publisher.
type outboxRow struct {
	eventID    string
	eventType  string
	tenantID   *string
	projectID  *string
	userID     *string
	requestID  *string
	traceID    *string
	payload    []byte
	attempts   int
	occurredAt time.Time
}

// claimBatch reclama hasta batchSize filas con FOR UPDATE SKIP LOCKED.
// Marca 'publishing' dentro de la TX, así el resto de workers no las tocan.
func (o *Outbox) claimBatch(ctx context.Context, tx pgx.Tx) ([]outboxRow, error) {
	const q = `
WITH claimed AS (
SELECT event_id
FROM outbox_events
WHERE status IN ('pending','failed')
  AND next_attempt_at <= NOW()
ORDER BY next_attempt_at
LIMIT $1
FOR UPDATE SKIP LOCKED
)
UPDATE outbox_events o
SET status    = 'publishing',
    attempts  = o.attempts + 1
FROM claimed
WHERE o.event_id = claimed.event_id
RETURNING
o.event_id, o.event_type,
o.tenant_id, o.project_id, o.user_id, o.request_id, o.trace_id,
o.payload, o.attempts, o.occurred_at
`
	rows, err := tx.Query(ctx, q, o.batchSize)
	if err != nil {
		return nil, fmt.Errorf("outbox: claim: %w", err)
	}
	defer rows.Close()

	var out []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(
			&r.eventID, &r.eventType,
			&r.tenantID, &r.projectID, &r.userID, &r.requestID, &r.traceID,
			&r.payload, &r.attempts, &r.occurredAt,
		); err != nil {
			return nil, fmt.Errorf("outbox: scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// publishOne publica un evento al bus y actualiza su estado.
func (o *Outbox) publishOne(ctx context.Context, r outboxRow) {
	ev := Event{
		ID:        r.eventID,
		Type:      r.eventType,
		Timestamp: r.occurredAt,
	}
	if r.tenantID != nil {
		ev.TenantID = *r.tenantID
	}
	if r.projectID != nil {
		ev.ProjectID = *r.projectID
	}
	if r.userID != nil {
		ev.UserID = *r.userID
	}
	if r.requestID != nil {
		ev.RequestID = *r.requestID
	}
	if r.traceID != nil {
		ev.TraceID = *r.traceID
	}
	if len(r.payload) > 0 {
		_ = json.Unmarshal(r.payload, &ev.Payload)
	}

	// Intento de publicación
	if err := o.bus.Publish(ctx, ev); err != nil {
		o.markFailed(ctx, r.eventID, r.attempts, err)
		return
	}
	o.markPublished(ctx, r.eventID)
}

func (o *Outbox) markPublished(ctx context.Context, id string) {
	const q = `
UPDATE outbox_events
SET status = 'published', published_at = NOW()
WHERE event_id = $1
`
	if _, err := o.pool.Exec(ctx, q, id); err != nil {
		logrus.Errorf("outbox: markPublished %s: %v", id, err)
	}
}

func (o *Outbox) markFailed(ctx context.Context, id string, attempts int, publishErr error) {
	// Si hemos superado maxAttempts -> dead letter
	status := "failed"
	if attempts >= o.maxAttempts {
		status = "dead"
	}

	backoff := o.computeBackoff(attempts)
	nextAttempt := time.Now().Add(backoff)

	const q = `
UPDATE outbox_events
SET status = $2,
    last_error = $3,
    next_attempt_at = $4
WHERE event_id = $1
`
	if _, err := o.pool.Exec(ctx, q, id, status, publishErr.Error(), nextAttempt); err != nil {
		logrus.Errorf("outbox: markFailed %s: %v", id, err)
		return
	}

	if status == "dead" {
		logrus.Errorf("☠️ outbox: evento %s marcado dead tras %d intentos: %v",
			id, attempts, publishErr)
	}
}

// computeBackoff: exponencial con jitter, capped a maxBackoff.
func (o *Outbox) computeBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return o.baseBackoff
	}
	backoff := o.baseBackoff
	for i := 1; i < attempts; i++ {
		backoff *= 2
		if backoff >= o.maxBackoff {
			return o.maxBackoff
		}
	}
	return backoff
}
