package audit

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/lecodev-26/sentinelflow/internal/events"
)

// Log emite un evento audit.* al outbox dentro de la TX del llamante.
//
// Uso desde el gateway (o desde cualquier sitio que tenga una TX de negocio):
//
//	err := client.WithTx(ctx, func(tx pgx.Tx) error {
//	   if err := audit.Log(ctx, tx, outbox, audit.Entry{
//	       Action:       events.EventAPIKeyCreated,
//	       TenantID:     tenantID,
//	       ActorID:      userID,
//	       ResourceType: "api_key",
//	       ResourceID:   apiKeyID,
//	       Payload:      map[string]any{"name": name},
//	   }); err != nil {
//	       return err
//	   }
//	   return createAPIKeyTx(ctx, tx, ...)
//	})
//
// Garantía: si la TX hace rollback, el evento audit.* no se persiste.
// No hay auditoría fantasma de acciones que nunca pasaron.
type Entry struct {
	Action       string
	TenantID     string
	ProjectID    string
	ActorID      string
	ActorEmail   string
	ResourceType string
	ResourceID   string
	IP           string
	UserAgent    string
	RequestID    string
	TraceID      string
	Payload      map[string]any
}

// Log inserta el evento audit en el outbox dentro de la TX del llamante.
func Log(ctx context.Context, tx pgx.Tx, outbox *events.Outbox, e Entry) error {
	if e.Payload == nil {
		e.Payload = make(map[string]any)
	}
	// Duplicar campos clave en payload para que el consumer pueda extraerlos.
	if e.ActorEmail != "" {
		e.Payload["actor_email"] = e.ActorEmail
	}
	if e.IP != "" {
		e.Payload["ip"] = e.IP
	}
	if e.UserAgent != "" {
		e.Payload["user_agent"] = e.UserAgent
	}
	switch e.ResourceType {
	case "api_key":
		e.Payload["api_key_id"] = e.ResourceID
	case "policy":
		e.Payload["policy_id"] = e.ResourceID
	case "user":
		e.Payload["user_id"] = e.ResourceID
	}

	ev := events.NewEvent(e.Action).
		WithTenant(e.TenantID).
		WithProject(e.ProjectID).
		WithUser(e.ActorID).
		WithRequest(e.RequestID).
		WithTrace(e.TraceID)

	for k, v := range e.Payload {
		ev = ev.WithPayload(k, v)
	}

	return outbox.EnqueueTx(ctx, tx, ev)
}
