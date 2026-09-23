package postgres

import (
	"context"
	"time"
)

// SecurityEvent es una fila de security_events.
type SecurityEvent struct {
	ID         string                 `json:"id"`
	Kind       string                 `json:"kind"`
	Severity   string                 `json:"severity"`
	TenantID   string                 `json:"tenant_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Detector   string                 `json:"detector,omitempty"`
	Rule       string                 `json:"rule,omitempty"`
	Snippet    string                 `json:"snippet,omitempty"`
	Action     string                 `json:"action,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
	RecordedAt time.Time              `json:"recorded_at"`
}

// SecurityRepo consulta security_events.
type SecurityRepo struct {
	client *Client
}

func (c *Client) Security() *SecurityRepo {
	return &SecurityRepo{client: c}
}

// ListByTenant devuelve los últimos N eventos de un tenant.
func (r *SecurityRepo) ListByTenant(ctx context.Context, tenantID string, limit int, kind string) ([]SecurityEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	var (
		rows interface {
			Next() bool
			Scan(dest ...any) error
			Close()
			Err() error
		}
		err error
	)
	if kind != "" {
		rows, err = r.client.Query(ctx, `
SELECT id, kind, severity,
       COALESCE(tenant_id,''), COALESCE(user_id,''), COALESCE(request_id,''),
       COALESCE(detector,''), COALESCE(rule,''), COALESCE(snippet,''), COALESCE(action,''),
       payload, occurred_at, recorded_at
FROM security_events
WHERE tenant_id = $1 AND kind = $2
ORDER BY occurred_at DESC
LIMIT $3
`, tenantID, kind, limit)
	} else {
		rows, err = r.client.Query(ctx, `
SELECT id, kind, severity,
       COALESCE(tenant_id,''), COALESCE(user_id,''), COALESCE(request_id,''),
       COALESCE(detector,''), COALESCE(rule,''), COALESCE(snippet,''), COALESCE(action,''),
       payload, occurred_at, recorded_at
FROM security_events
WHERE tenant_id = $1
ORDER BY occurred_at DESC
LIMIT $2
`, tenantID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SecurityEvent{}
	for rows.Next() {
		var e SecurityEvent
		if err := rows.Scan(
			&e.ID, &e.Kind, &e.Severity,
			&e.TenantID, &e.UserID, &e.RequestID,
			&e.Detector, &e.Rule, &e.Snippet, &e.Action,
			&e.Payload, &e.OccurredAt, &e.RecordedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
