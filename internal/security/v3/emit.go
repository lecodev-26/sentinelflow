package v3

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/lecodev-26/sentinelflow/internal/events"
)

// Emitter publica eventos security.* al outbox.
//
// En V3.3 se llama desde el gateway cuando el pipeline detecta findings.
// Como el pipeline corre antes de la TX de accounting, estos eventos se
// emiten en su propia TX corta (una por evento agrupado).
type Emitter struct {
	outbox *events.Outbox
	pool   interface {
		BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
	}
}

// NewEmitter crea un emitter.
func NewEmitter(outbox *events.Outbox, pool interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}) *Emitter {
	return &Emitter{outbox: outbox, pool: pool}
}

// Finding representa un hallazgo a emitir.
type EmitFinding struct {
	Kind     string // pii | secret | prompt_injection | ssrf | blocked
	Severity string // low | medium | high | critical
	Detector string
	Rule     string
	Snippet  string
	Action   string // allow | redact | warn | block
}

// Emit publica todos los findings en una sola TX.
// Si findings está vacío, no hace nada.
func (e *Emitter) Emit(ctx context.Context, tenantID, projectID, userID, requestID, traceID string, findings []EmitFinding) error {
	if len(findings) == 0 {
		return nil
	}
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, f := range findings {
		evType := eventTypeFor(f.Kind)
		if evType == "" {
			continue
		}
		ev := events.NewEvent(evType).
			WithTenant(tenantID).
			WithProject(projectID).
			WithUser(userID).
			WithRequest(requestID).
			WithTrace(traceID).
			WithPayload("severity", f.Severity).
			WithPayload("detector", f.Detector).
			WithPayload("rule", f.Rule).
			WithPayload("snippet", f.Snippet).
			WithPayload("action", f.Action).
			WithPayload("kind", f.Kind)
		if err := e.outbox.EnqueueTx(ctx, tx, ev); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func eventTypeFor(kind string) string {
	switch kind {
	case "pii":
		return events.EventPIIDetected
	case "secret":
		return events.EventSecretDetected
	case "prompt_injection":
		return events.EventInjectionDetected
	case "blocked":
		return events.EventSecurityBlocked
	}
	return ""
}
