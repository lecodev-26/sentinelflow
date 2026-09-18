package sqlite

import (
"context"
"database/sql"
"time"

"github.com/lecodev-26/sentinelflow/internal/storage"
)

type usageRepo struct {
db *sql.DB
}

func (r *usageRepo) Create(ctx context.Context, u *storage.UsageRecord) error {
if u.Timestamp.IsZero() {
u.Timestamp = time.Now()
}

_, err := r.db.ExecContext(ctx, `
INSERT INTO usage_records (id, request_id, tenant_id, project_id, provider, model,
input_tokens, output_tokens, total_tokens, cost_usd, latency_ms, status, timestamp)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
u.ID, u.RequestID, u.TenantID, u.ProjectID, u.Provider, u.Model,
u.InputTokens, u.OutputTokens, u.TotalTokens, u.CostUSD, u.LatencyMs, u.Status, u.Timestamp)
return err
}

func (r *usageRepo) ListByTenant(ctx context.Context, tenantID string, since time.Time) ([]*storage.UsageRecord, error) {
rows, err := r.db.QueryContext(ctx, `
SELECT id, request_id, tenant_id, project_id, provider, model,
input_tokens, output_tokens, total_tokens, cost_usd, latency_ms, status, timestamp
FROM usage_records
WHERE tenant_id = ? AND timestamp >= ?
ORDER BY timestamp DESC LIMIT 1000`, tenantID, since)
if err != nil {
return nil, err
}
defer rows.Close()

var records []*storage.UsageRecord
for rows.Next() {
var u storage.UsageRecord
err := rows.Scan(&u.ID, &u.RequestID, &u.TenantID, &u.ProjectID, &u.Provider,
&u.Model, &u.InputTokens, &u.OutputTokens, &u.TotalTokens, &u.CostUSD,
&u.LatencyMs, &u.Status, &u.Timestamp)
if err != nil {
return nil, err
}
records = append(records, &u)
}
return records, rows.Err()
}

func (r *usageRepo) SumCostByTenant(ctx context.Context, tenantID string, since time.Time) (float64, error) {
var total sql.NullFloat64
err := r.db.QueryRowContext(ctx, `
SELECT SUM(cost_usd) FROM usage_records
WHERE tenant_id = ? AND timestamp >= ?`, tenantID, since).Scan(&total)
if err != nil {
return 0, err
}
return total.Float64, nil
}

func (r *usageRepo) SumCostByProvider(ctx context.Context, tenantID string, since time.Time) (map[string]float64, error) {
rows, err := r.db.QueryContext(ctx, `
SELECT provider, SUM(cost_usd) FROM usage_records
WHERE tenant_id = ? AND timestamp >= ?
GROUP BY provider`, tenantID, since)
if err != nil {
return nil, err
}
defer rows.Close()

result := make(map[string]float64)
for rows.Next() {
var provider string
var cost sql.NullFloat64
if err := rows.Scan(&provider, &cost); err != nil {
return nil, err
}
result[provider] = cost.Float64
}
return result, rows.Err()
}

func (r *usageRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
res, err := r.db.ExecContext(ctx, `DELETE FROM usage_records WHERE timestamp < ?`, before)
if err != nil {
return 0, err
}
return res.RowsAffected()
}
