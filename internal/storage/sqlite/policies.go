package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/storage"
)

type policyRepo struct {
	db *sql.DB
}

func (r *policyRepo) Create(ctx context.Context, p *storage.Policy) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	p.UpdatedAt = time.Now()
	if p.Rules == "" {
		p.Rules = "[]"
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO policies (id, name, description, tenant_id, priority, enabled, rules, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Description, p.TenantID, p.Priority, p.Enabled, p.Rules, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *policyRepo) GetByID(ctx context.Context, id string) (*storage.Policy, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, description, tenant_id, priority, enabled, rules, created_at, updated_at
FROM policies WHERE id = ?`, id)

	var p storage.Policy
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.TenantID, &p.Priority, &p.Enabled, &p.Rules, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *policyRepo) ListByTenant(ctx context.Context, tenantID string) ([]*storage.Policy, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, description, tenant_id, priority, enabled, rules, created_at, updated_at
FROM policies WHERE tenant_id = ? ORDER BY priority ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*storage.Policy
	for rows.Next() {
		var p storage.Policy
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.TenantID, &p.Priority, &p.Enabled, &p.Rules, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, &p)
	}
	return policies, rows.Err()
}

func (r *policyRepo) Update(ctx context.Context, p *storage.Policy) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
UPDATE policies SET name=?, description=?, priority=?, enabled=?, rules=?, updated_at=?
WHERE id=?`,
		p.Name, p.Description, p.Priority, p.Enabled, p.Rules, p.UpdatedAt, p.ID)
	return err
}

func (r *policyRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM policies WHERE id = ?`, id)
	return err
}
