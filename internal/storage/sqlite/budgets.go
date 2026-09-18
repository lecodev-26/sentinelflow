package sqlite

import (
"context"
"database/sql"
"time"

"github.com/lecodev-26/sentinelflow/internal/storage"
)

type budgetRepo struct {
db *sql.DB
}

func (r *budgetRepo) Upsert(ctx context.Context, b *storage.Budget) error {
b.UpdatedAt = time.Now()
_, err := r.db.ExecContext(ctx, `
INSERT INTO budgets (id, tenant_id, monthly_limit, spent, month_start, reset_date, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(tenant_id) DO UPDATE SET
monthly_limit=excluded.monthly_limit,
spent=excluded.spent,
month_start=excluded.month_start,
reset_date=excluded.reset_date,
updated_at=excluded.updated_at`,
b.ID, b.TenantID, b.MonthlyLimit, b.Spent, b.MonthStart, b.ResetDate, b.UpdatedAt)
return err
}

func (r *budgetRepo) GetByTenant(ctx context.Context, tenantID string) (*storage.Budget, error) {
row := r.db.QueryRowContext(ctx, `
SELECT id, tenant_id, monthly_limit, spent, month_start, reset_date, updated_at
FROM budgets WHERE tenant_id = ?`, tenantID)

var b storage.Budget
err := row.Scan(&b.ID, &b.TenantID, &b.MonthlyLimit, &b.Spent, &b.MonthStart, &b.ResetDate, &b.UpdatedAt)
if err == sql.ErrNoRows {
return nil, storage.ErrNotFound
}
if err != nil {
return nil, err
}
return &b, nil
}

func (r *budgetRepo) List(ctx context.Context) ([]*storage.Budget, error) {
rows, err := r.db.QueryContext(ctx, `
SELECT id, tenant_id, monthly_limit, spent, month_start, reset_date, updated_at
FROM budgets ORDER BY tenant_id`)
if err != nil {
return nil, err
}
defer rows.Close()

var budgets []*storage.Budget
for rows.Next() {
var b storage.Budget
if err := rows.Scan(&b.ID, &b.TenantID, &b.MonthlyLimit, &b.Spent, &b.MonthStart, &b.ResetDate, &b.UpdatedAt); err != nil {
return nil, err
}
budgets = append(budgets, &b)
}
return budgets, rows.Err()
}

func (r *budgetRepo) UpdateSpent(ctx context.Context, tenantID string, spent float64) error {
_, err := r.db.ExecContext(ctx, `
UPDATE budgets SET spent = ?, updated_at = ? WHERE tenant_id = ?`,
spent, time.Now(), tenantID)
return err
}

func (r *budgetRepo) Delete(ctx context.Context, tenantID string) error {
_, err := r.db.ExecContext(ctx, `DELETE FROM budgets WHERE tenant_id = ?`, tenantID)
return err
}
