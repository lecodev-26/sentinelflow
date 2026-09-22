package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/storage"
)

type auditRepo struct {
	db *sql.DB
}

func (r *auditRepo) Create(ctx context.Context, e *storage.AuditEvent) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	if e.Metadata == "" {
		e.Metadata = "{}"
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO audit_events (id, timestamp, type, tenant_id, user_id, request_id, action, result, reason, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Timestamp, e.Type, e.TenantID, e.UserID, e.RequestID,
		e.Action, e.Result, e.Reason, e.Metadata)
	return err
}

func (r *auditRepo) List(ctx context.Context, tenantID string, limit int) ([]*storage.AuditEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, timestamp, type, tenant_id, user_id, request_id, action, result, reason, metadata
FROM audit_events
WHERE tenant_id = ? OR ? = ''
ORDER BY timestamp DESC LIMIT ?`, tenantID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditEvents(rows)
}

func (r *auditRepo) ListByRequest(ctx context.Context, requestID string) ([]*storage.AuditEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, timestamp, type, tenant_id, user_id, request_id, action, result, reason, metadata
FROM audit_events WHERE request_id = ? ORDER BY timestamp ASC`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditEvents(rows)
}

func (r *auditRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM audit_events WHERE timestamp < ?`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanAuditEvents(rows *sql.Rows) ([]*storage.AuditEvent, error) {
	var events []*storage.AuditEvent
	for rows.Next() {
		var e storage.AuditEvent
		err := rows.Scan(&e.ID, &e.Timestamp, &e.Type, &e.TenantID, &e.UserID,
			&e.RequestID, &e.Action, &e.Result, &e.Reason, &e.Metadata)
		if err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}
