package postgres

import (
	"context"
	"time"
)

// AuditEntry es una fila de audit_log leída desde la DB.
type AuditEntry struct {
	ID           string                 `json:"id"`
	EventID      string                 `json:"event_id"`
	Action       string                 `json:"action"`
	ActorID      string                 `json:"actor_id,omitempty"`
	ActorEmail   string                 `json:"actor_email,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	IP           string                 `json:"ip,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	OccurredAt   time.Time              `json:"occurred_at"`
	RecordedAt   time.Time              `json:"recorded_at"`
}

// AuditRepo consulta el audit_log.
type AuditRepo struct {
	client *Client
}

func (c *Client) Audit() *AuditRepo {
	return &AuditRepo{client: c}
}

// ListByTenant devuelve las entradas de un tenant, más recientes primero.
func (r *AuditRepo) ListByTenant(ctx context.Context, tenantID string, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	const q = `
SELECT id, event_id, action,
       COALESCE(actor_id,''), COALESCE(actor_email,''),
       COALESCE(tenant_id,''), COALESCE(project_id,''),
       COALESCE(resource_type,''), COALESCE(resource_id,''),
       COALESCE(ip,''), COALESCE(request_id,''),
       payload, occurred_at, recorded_at
FROM audit_log
WHERE tenant_id = $1
ORDER BY occurred_at DESC
LIMIT $2
`
	rows, err := r.client.Query(ctx, q, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(
			&e.ID, &e.EventID, &e.Action,
			&e.ActorID, &e.ActorEmail,
			&e.TenantID, &e.ProjectID,
			&e.ResourceType, &e.ResourceID,
			&e.IP, &e.RequestID,
			&e.Payload, &e.OccurredAt, &e.RecordedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
