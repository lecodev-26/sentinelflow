package v3

import (
"context"
"time"

"github.com/lecodev-26/sentinelflow/internal/logger"
routing "github.com/lecodev-26/sentinelflow/internal/routing/v3"
"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

// Service es el servicio de accounting
type Service struct {
usageRepo   *postgres.UsageRepo
budgetRepo  *postgres.BudgetRepo
modelReg    *routing.ModelRegistry
onBudgetAlert func(alert BudgetAlert)
}

// BudgetAlert representa una alerta de presupuesto
type BudgetAlert struct {
TenantID  string    `json:"tenant_id"`
Threshold int       `json:"threshold"`
Spent     float64   `json:"spent_usd"`
Limit     float64   `json:"limit_usd"`
Percent   float64   `json:"percent"`
Timestamp time.Time `json:"timestamp"`
}

// NewService crea un nuevo servicio
func NewService(usageRepo *postgres.UsageRepo, budgetRepo *postgres.BudgetRepo, modelReg *routing.ModelRegistry) *Service {
return &Service{
usageRepo:  usageRepo,
budgetRepo: budgetRepo,
modelReg:   modelReg,
}
}

// OnBudgetAlert registra un callback de alerta
func (s *Service) OnBudgetAlert(cb func(BudgetAlert)) {
s.onBudgetAlert = cb
}

// Record registra un uso y calcula coste
func (s *Service) Record(ctx context.Context, rec *postgres.UsageRecord) error {
// Calcular coste con pricing real del Model Registry
if s.modelReg != nil && rec.Provider != "" && rec.Model != "" {
if model, ok := s.modelReg.Get(rec.Provider, rec.Model); ok {
rec.InputCostUSD = (float64(rec.InputTokens) / 1_000_000.0) * model.InputPer1M
rec.OutputCostUSD = (float64(rec.OutputTokens) / 1_000_000.0) * model.OutputPer1M
rec.CostUSD = rec.InputCostUSD + rec.OutputCostUSD
}
}

// Persistir usage record
if err := s.usageRepo.Create(ctx, rec); err != nil {
logger.Errorf("❌ Error guardando usage record: %v", err)
return err
}

logger.Infof("💰 Usage: tenant=%s provider=%s model=%s tokens=%d cost=$%.6f",
rec.TenantID, rec.Provider, rec.Model, rec.TotalTokens, rec.CostUSD)

// Actualizar budget
if rec.CostUSD > 0 {
budget, err := s.budgetRepo.RecordSpend(ctx, rec.TenantID, rec.CostUSD)
if err != nil {
logger.Errorf("❌ Error actualizando budget: %v", err)
} else {
s.checkAlerts(ctx, budget)
}
}

return nil
}

// checkAlerts verifica si hay que disparar alertas
func (s *Service) checkAlerts(ctx context.Context, b *postgres.Budget) {
if b.MonthlyLimit <= 0 {
return
}

pct := b.PercentMonth()
var threshold int

if pct >= 100 && !b.Alert100Sent {
threshold = 100
b.Alert100Sent = true
} else if pct >= 90 && !b.Alert90Sent {
threshold = 90
b.Alert90Sent = true
} else if pct >= 80 && !b.Alert80Sent {
threshold = 80
b.Alert80Sent = true
}

if threshold == 0 {
return
}

// Marcar como enviada
_ = s.budgetRepo.MarkAlert(ctx, b.TenantID, threshold)

logger.Warnf("🚨 BUDGET ALERT: tenant=%s threshold=%d%% spent=$%.2f/%.2f",
b.TenantID, threshold, b.SpentMonth, b.MonthlyLimit)

if s.onBudgetAlert != nil {
go s.onBudgetAlert(BudgetAlert{
TenantID:  b.TenantID,
Threshold: threshold,
Spent:     b.SpentMonth,
Limit:     b.MonthlyLimit,
Percent:   pct,
Timestamp: time.Now(),
})
}
}

// Stats agrega estadísticas por tenant
func (s *Service) Stats(ctx context.Context, tenantID string, since time.Duration) (map[string]interface{}, error) {
sinceTime := time.Now().Add(-since)

totalCost, _ := s.usageRepo.SumCostByTenant(ctx, tenantID, sinceTime)
byProvider, _ := s.usageRepo.SumCostByProvider(ctx, tenantID, sinceTime)
byModel, _ := s.usageRepo.SumCostByModel(ctx, tenantID, sinceTime)
byDay, _ := s.usageRepo.SumCostByDay(ctx, tenantID, sinceTime)
count, _ := s.usageRepo.CountByTenant(ctx, tenantID, sinceTime)

budget, _ := s.budgetRepo.GetByTenant(ctx, tenantID)

return map[string]interface{}{
"tenant_id":    tenantID,
"total_cost":   totalCost,
"by_provider":  byProvider,
"by_model":     byModel,
"by_day":       byDay,
"requests":     count,
"budget":       budget,
}, nil
}
