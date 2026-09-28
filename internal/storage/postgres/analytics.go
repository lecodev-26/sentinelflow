package postgres

import (
	"context"
	"time"
)

type AnalyticsDaily struct {
	TenantID     string    `json:"tenant_id"`
	Environment  string    `json:"environment"`
	Day          string    `json:"day"`
	Requests     int64     `json:"requests"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	Errors       int64     `json:"errors"`
	LastUpdated  time.Time `json:"last_updated"`
}
type AnalyticsRepo struct{ client *Client }

func (c *Client) Analytics() *AnalyticsRepo { return &AnalyticsRepo{client: c} }
func (r *AnalyticsRepo) ListByTenant(ctx context.Context, tenantID, environment string, days int) ([]AnalyticsDaily, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	const q = `SELECT tenant_id,environment,to_char(day,'YYYY-MM-DD'),requests,input_tokens,output_tokens,total_tokens,cost_usd::float8,errors,last_updated FROM analytics_daily WHERE tenant_id=$1 AND environment=$2 ORDER BY day DESC LIMIT $3`
	rows, err := r.client.Query(ctx, q, tenantID, environment, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AnalyticsDaily{}
	for rows.Next() {
		var a AnalyticsDaily
		if err := rows.Scan(&a.TenantID, &a.Environment, &a.Day, &a.Requests, &a.InputTokens, &a.OutputTokens, &a.TotalTokens, &a.CostUSD, &a.Errors, &a.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
