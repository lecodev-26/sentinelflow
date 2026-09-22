package postgres

import (
	"context"
	"database/sql"
	"time"
)

// Budget representa un presupuesto persistido
type Budget struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Scope        string    `json:"scope"` // tenant, project, model
	ScopeID      string    `json:"scope_id,omitempty"`
	MonthlyLimit float64   `json:"monthly_limit_usd"`
	DailyLimit   float64   `json:"daily_limit_usd"`
	SpentMonth   float64   `json:"spent_month_usd"`
	SpentDay     float64   `json:"spent_day_usd"`
	MonthStart   time.Time `json:"month_start"`
	DayStart     time.Time `json:"day_start"`
	Alert80Sent  bool      `json:"alert_80_sent"`
	Alert90Sent  bool      `json:"alert_90_sent"`
	Alert100Sent bool      `json:"alert_100_sent"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PercentMonth devuelve el % usado del mes
func (b *Budget) PercentMonth() float64 {
	if b.MonthlyLimit <= 0 {
		return 0
	}
	return b.SpentMonth / b.MonthlyLimit * 100
}

// PercentDay devuelve el % usado del día
func (b *Budget) PercentDay() float64 {
	if b.DailyLimit <= 0 {
		return 0
	}
	return b.SpentDay / b.DailyLimit * 100
}

// BudgetRepo gestiona budgets
type BudgetRepo struct {
	client *Client
}

func (c *Client) Budgets() *BudgetRepo {
	return &BudgetRepo{client: c}
}

// Upsert crea o actualiza un budget
func (r *BudgetRepo) Upsert(ctx context.Context, b *Budget) error {
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now()
	}
	if b.MonthStart.IsZero() {
		b.MonthStart = time.Now()
	}
	if b.DayStart.IsZero() {
		b.DayStart = time.Now()
	}
	if b.Scope == "" {
		b.Scope = "tenant"
	}

	_, err := r.client.Exec(ctx, `
INSERT INTO budgets (id, tenant_id, scope, scope_id, monthly_limit_usd, daily_limit_usd,
spent_month_usd, spent_day_usd, month_start, day_start, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (tenant_id, scope, scope_id) DO UPDATE SET
monthly_limit_usd = EXCLUDED.monthly_limit_usd,
daily_limit_usd = EXCLUDED.daily_limit_usd,
updated_at = EXCLUDED.updated_at`,
		b.ID, b.TenantID, b.Scope, b.ScopeID, b.MonthlyLimit, b.DailyLimit,
		b.SpentMonth, b.SpentDay, b.MonthStart, b.DayStart, b.UpdatedAt)
	return err
}

// GetByTenant devuelve el budget tenant-wide
func (r *BudgetRepo) GetByTenant(ctx context.Context, tenantID string) (*Budget, error) {
	row := r.client.QueryRow(ctx, `
SELECT id, tenant_id, scope, COALESCE(scope_id, ''), monthly_limit_usd, daily_limit_usd,
spent_month_usd, spent_day_usd, month_start, day_start,
alert_80_sent, alert_90_sent, alert_100_sent, updated_at
FROM budgets
WHERE tenant_id = $1 AND scope = 'tenant'`, tenantID)

	return scanBudget(row)
}

// RecordSpend incrementa el gasto y actualiza alertas
func (r *BudgetRepo) RecordSpend(ctx context.Context, tenantID string, cost float64) (*Budget, error) {
	// Asegurar que existe
	b, err := r.GetByTenant(ctx, tenantID)
	if err == sql.ErrNoRows || b == nil {
		// Crear budget por defecto
		b = &Budget{
			ID:           "budget_" + tenantID,
			TenantID:     tenantID,
			Scope:        "tenant",
			MonthlyLimit: 100.0,
			DailyLimit:   10.0,
			MonthStart:   time.Now(),
			DayStart:     time.Now(),
		}
		if err := r.Upsert(ctx, b); err != nil {
			return nil, err
		}
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Reset mensual si pasó el mes
	now := time.Now()
	if now.Month() != b.MonthStart.Month() || now.Year() != b.MonthStart.Year() {
		b.SpentMonth = 0
		b.MonthStart = now
		b.Alert80Sent = false
		b.Alert90Sent = false
		b.Alert100Sent = false
	}

	// Reset diario si pasó el día
	if now.Day() != b.DayStart.Day() {
		b.SpentDay = 0
		b.DayStart = now
	}

	// Incrementar
	b.SpentMonth += cost
	b.SpentDay += cost
	b.UpdatedAt = now

	// Actualizar en DB
	_, err = r.client.Exec(ctx, `
UPDATE budgets SET
spent_month_usd = $1,
spent_day_usd = $2,
month_start = $3,
day_start = $4,
alert_80_sent = $5,
alert_90_sent = $6,
alert_100_sent = $7,
updated_at = $8
WHERE tenant_id = $9 AND scope = 'tenant'`,
		b.SpentMonth, b.SpentDay, b.MonthStart, b.DayStart,
		b.Alert80Sent, b.Alert90Sent, b.Alert100Sent, b.UpdatedAt, tenantID)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// MarkAlert marca una alerta como enviada
func (r *BudgetRepo) MarkAlert(ctx context.Context, tenantID string, threshold int) error {
	column := "alert_80_sent"
	switch threshold {
	case 90:
		column = "alert_90_sent"
	case 100:
		column = "alert_100_sent"
	}

	_, err := r.client.Exec(ctx, `
UPDATE budgets SET `+column+` = TRUE, updated_at = NOW()
WHERE tenant_id = $1 AND scope = 'tenant'`, tenantID)
	return err
}

func scanBudget(row interface {
	Scan(dest ...interface{}) error
}) (*Budget, error) {
	var b Budget
	var scopeID string
	err := row.Scan(
		&b.ID, &b.TenantID, &b.Scope, &scopeID,
		&b.MonthlyLimit, &b.DailyLimit,
		&b.SpentMonth, &b.SpentDay,
		&b.MonthStart, &b.DayStart,
		&b.Alert80Sent, &b.Alert90Sent, &b.Alert100Sent,
		&b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	b.ScopeID = scopeID
	return &b, nil
}
