package accounting

import (
"sync"
"time"
)

// BudgetState representa el estado de un budget
type BudgetState struct {
TenantID     string    `json:"tenant_id"`
MonthlyLimit float64   `json:"monthly_limit"`
Spent        float64   `json:"spent"`
MonthStart   time.Time `json:"month_start"`
ResetDate    time.Time `json:"reset_date"`

// Alerts
Alert80Sent  bool `json:"alert_80_sent"`
Alert90Sent  bool `json:"alert_90_sent"`
Alert100Sent bool `json:"alert_100_sent"`

// Forecast
Forecast float64 `json:"forecast"`
}

// PercentUsed devuelve el % usado
func (b *BudgetState) PercentUsed() float64 {
if b.MonthlyLimit <= 0 {
return 0
}
return b.Spent / b.MonthlyLimit * 100
}

// Remaining devuelve el dinero restante
func (b *BudgetState) Remaining() float64 {
rem := b.MonthlyLimit - b.Spent
if rem < 0 {
return 0
}
return rem
}

// BudgetAlert es una alerta de presupuesto
type BudgetAlert struct {
TenantID  string    `json:"tenant_id"`
Threshold int       `json:"threshold"` // 80, 90, 100
Spent     float64   `json:"spent"`
Limit     float64   `json:"limit"`
Percent   float64   `json:"percent"`
Forecast  float64   `json:"forecast"`
Timestamp time.Time `json:"timestamp"`
}

// BudgetAlertCallback se ejecuta cuando se supera un threshold
type BudgetAlertCallback func(alert BudgetAlert)

// BudgetEngine gestiona presupuestos
type BudgetEngine struct {
mu        sync.RWMutex
budgets   map[string]*BudgetState // tenantID → state
onAlert   BudgetAlertCallback
}

// NewBudgetEngine crea un nuevo engine
func NewBudgetEngine(onAlert BudgetAlertCallback) *BudgetEngine {
return &BudgetEngine{
budgets: make(map[string]*BudgetState),
onAlert: onAlert,
}
}

// SetBudget configura un presupuesto
func (b *BudgetEngine) SetBudget(tenantID string, monthlyLimit float64) {
b.mu.Lock()
defer b.mu.Unlock()

b.budgets[tenantID] = &BudgetState{
TenantID:     tenantID,
MonthlyLimit: monthlyLimit,
Spent:        0,
MonthStart:   time.Now(),
ResetDate:    time.Now().Add(30 * 24 * time.Hour),
}
}

// Record registra un gasto
func (b *BudgetEngine) Record(tenantID string, cost float64) *BudgetAlert {
b.mu.Lock()
defer b.mu.Unlock()

state, exists := b.budgets[tenantID]
if !exists {
return nil
}

// Resetear si pasó el mes
if time.Now().After(state.ResetDate) {
state.Spent = 0
state.MonthStart = time.Now()
state.ResetDate = time.Now().Add(30 * 24 * time.Hour)
state.Alert80Sent = false
state.Alert90Sent = false
state.Alert100Sent = false
}

state.Spent += cost

// Calcular forecast
state.Forecast = b.forecast(state)

// Verificar thresholds
pct := state.PercentUsed()
var alert *BudgetAlert

if pct >= 100 && !state.Alert100Sent {
state.Alert100Sent = true
alert = b.makeAlert(state, 100)
} else if pct >= 90 && !state.Alert90Sent {
state.Alert90Sent = true
alert = b.makeAlert(state, 90)
} else if pct >= 80 && !state.Alert80Sent {
state.Alert80Sent = true
alert = b.makeAlert(state, 80)
}

if alert != nil && b.onAlert != nil {
go b.onAlert(*alert)
}

return alert
}

// GetState devuelve el estado de un budget
func (b *BudgetEngine) GetState(tenantID string) (*BudgetState, bool) {
b.mu.RLock()
defer b.mu.RUnlock()
state, exists := b.budgets[tenantID]
return state, exists
}

// CheckBudget verifica si hay presupuesto
func (b *BudgetEngine) CheckBudget(tenantID string, estimatedCost float64) (bool, float64) {
b.mu.RLock()
defer b.mu.RUnlock()

state, exists := b.budgets[tenantID]
if !exists {
return true, 0
}

remaining := state.Remaining()
if remaining < estimatedCost {
return false, remaining
}
return true, remaining
}

// forecast estima el gasto mensual basado en el ritmo actual
func (b *BudgetEngine) forecast(state *BudgetState) float64 {
elapsed := time.Since(state.MonthStart)
if elapsed.Hours() < 1 {
return state.Spent
}

// Gasto por hora * 24 * 30
perHour := state.Spent / elapsed.Hours()
return perHour * 24 * 30
}

func (b *BudgetEngine) makeAlert(state *BudgetState, threshold int) *BudgetAlert {
return &BudgetAlert{
TenantID:  state.TenantID,
Threshold: threshold,
Spent:     state.Spent,
Limit:     state.MonthlyLimit,
Percent:   state.PercentUsed(),
Forecast:  state.Forecast,
Timestamp: time.Now(),
}
}

// ListBudgets lista todos los presupuestos
func (b *BudgetEngine) ListBudgets() []*BudgetState {
b.mu.RLock()
defer b.mu.RUnlock()

result := make([]*BudgetState, 0, len(b.budgets))
for _, state := range b.budgets {
result = append(result, state)
}
return result
}
