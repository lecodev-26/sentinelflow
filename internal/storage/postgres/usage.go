package postgres

import (
	"context"
	"time"
)

// UsageRecord representa un registro de uso
type UsageRecord struct {
	ID            string    `json:"id"`
	RequestID     string    `json:"request_id"`
	TenantID      string    `json:"tenant_id"`
	ProjectID     string    `json:"project_id,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	APIKeyID      string    `json:"api_key_id,omitempty"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	InputCostUSD  float64   `json:"input_cost_usd"`
	OutputCostUSD float64   `json:"output_cost_usd"`
	CostUSD       float64   `json:"cost_usd"`
	LatencyMs     int       `json:"latency_ms"`
	TTFTMs        int       `json:"ttft_ms"`
	Status        string    `json:"status"`
	CacheHit      bool      `json:"cache_hit"`
	Fallback      bool      `json:"fallback"`
	Timestamp     time.Time `json:"timestamp"`
}

// UsageRepo gestiona registros de uso
type UsageRepo struct {
	client *Client
}

func (c *Client) Usage() *UsageRepo {
	return &UsageRepo{client: c}
}

// Create inserta un nuevo registro
func (r *UsageRepo) Create(ctx context.Context, rec *UsageRecord) error {
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	rec.TotalTokens = rec.InputTokens + rec.OutputTokens
	rec.CostUSD = rec.InputCostUSD + rec.OutputCostUSD

	_, err := r.client.Exec(ctx, `
INSERT INTO usage_records (
id, request_id, tenant_id, project_id, user_id, api_key_id,
provider, model, input_tokens, output_tokens, total_tokens,
input_cost_usd, output_cost_usd, cost_usd, latency_ms, ttft_ms,
status, cache_hit, fallback, timestamp
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
		rec.ID, rec.RequestID, rec.TenantID, rec.ProjectID, rec.UserID, rec.APIKeyID,
		rec.Provider, rec.Model, rec.InputTokens, rec.OutputTokens, rec.TotalTokens,
		rec.InputCostUSD, rec.OutputCostUSD, rec.CostUSD, rec.LatencyMs, rec.TTFTMs,
		rec.Status, rec.CacheHit, rec.Fallback, rec.Timestamp)
	return err
}

// SumCostByTenant suma el coste de un tenant desde una fecha
func (r *UsageRepo) SumCostByTenant(ctx context.Context, tenantID string, since time.Time) (float64, error) {
	var total float64
	err := r.client.QueryRow(ctx, `
SELECT COALESCE(SUM(cost_usd), 0) FROM usage_records
WHERE tenant_id = $1 AND timestamp >= $2`, tenantID, since).Scan(&total)
	return total, err
}

// SumCostByProvider agrupa el coste por provider
func (r *UsageRepo) SumCostByProvider(ctx context.Context, tenantID string, since time.Time) (map[string]float64, error) {
	rows, err := r.client.Query(ctx, `
SELECT provider, COALESCE(SUM(cost_usd), 0) FROM usage_records
WHERE tenant_id = $1 AND timestamp >= $2
GROUP BY provider`, tenantID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var provider string
		var cost float64
		if err := rows.Scan(&provider, &cost); err != nil {
			return nil, err
		}
		result[provider] = cost
	}
	return result, rows.Err()
}

// SumCostByModel agrupa el coste por modelo
func (r *UsageRepo) SumCostByModel(ctx context.Context, tenantID string, since time.Time) (map[string]float64, error) {
	rows, err := r.client.Query(ctx, `
SELECT model, COALESCE(SUM(cost_usd), 0) FROM usage_records
WHERE tenant_id = $1 AND timestamp >= $2
GROUP BY model`, tenantID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var model string
		var cost float64
		if err := rows.Scan(&model, &cost); err != nil {
			return nil, err
		}
		result[model] = cost
	}
	return result, rows.Err()
}

// SumCostByDay agrupa el coste por día
func (r *UsageRepo) SumCostByDay(ctx context.Context, tenantID string, since time.Time) (map[string]float64, error) {
	rows, err := r.client.Query(ctx, `
SELECT DATE(timestamp)::text, COALESCE(SUM(cost_usd), 0) FROM usage_records
WHERE tenant_id = $1 AND timestamp >= $2
GROUP BY DATE(timestamp)
ORDER BY DATE(timestamp)`, tenantID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var day string
		var cost float64
		if err := rows.Scan(&day, &cost); err != nil {
			return nil, err
		}
		result[day] = cost
	}
	return result, rows.Err()
}

// CountByTenant cuenta las peticiones de un tenant
func (r *UsageRepo) CountByTenant(ctx context.Context, tenantID string, since time.Time) (int64, error) {
	var count int64
	err := r.client.QueryRow(ctx, `
SELECT COUNT(*) FROM usage_records
WHERE tenant_id = $1 AND timestamp >= $2`, tenantID, since).Scan(&count)
	return count, err
}
