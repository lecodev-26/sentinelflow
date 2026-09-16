package cost

import (
"sync"
"time"
)

type CostEntry struct {
RequestID    string    `json:"request_id"`
TenantID     string    `json:"tenant_id"`
ProjectID    string    `json:"project_id"`
Provider     string    `json:"provider"`
Model        string    `json:"model"`
InputTokens  int       `json:"input_tokens"`
OutputTokens int       `json:"output_tokens"`
TotalTokens  int       `json:"total_tokens"`
CostUSD      float64   `json:"cost_usd"`
Timestamp    time.Time `json:"timestamp"`
}

type Budget struct {
TenantID      string    `json:"tenant_id"`
MonthlyLimit  float64   `json:"monthly_limit"`
Used          float64   `json:"used"`
ResetDate     time.Time `json:"reset_date"`
Alert80Sent   bool      `json:"alert_80_sent"`
Alert90Sent   bool      `json:"alert_90_sent"`
Alert100Sent  bool      `json:"alert_100_sent"`
}

// BudgetAlertCallback se llama cuando se alcanza un threshold
type BudgetAlertCallback func(tenantID string, threshold int, budget *Budget)

type CostTracker struct {
mu         sync.RWMutex
entries    []CostEntry
limits     map[string]*Budget
alertCb    BudgetAlertCallback
}

func NewCostTracker() *CostTracker {
return &CostTracker{
entries: make([]CostEntry, 0),
limits:  make(map[string]*Budget),
}
}

// SetAlertCallback registra el callback de alertas
func (ct *CostTracker) SetAlertCallback(cb BudgetAlertCallback) {
ct.alertCb = cb
}

// Record registra una petición con tokens reales
func (ct *CostTracker) Record(tenantID, projectID, provider, model string, inputTokens, outputTokens int, cost float64) {
ct.mu.Lock()
defer ct.mu.Unlock()

entry := CostEntry{
RequestID:    generateRequestID(),
TenantID:     tenantID,
ProjectID:    projectID,
Provider:     provider,
Model:        model,
InputTokens:  inputTokens,
OutputTokens: outputTokens,
TotalTokens:  inputTokens + outputTokens,
CostUSD:      cost,
Timestamp:    time.Now(),
}

ct.entries = append(ct.entries, entry)

// Actualizar budget + alerts
if budget, exists := ct.limits[tenantID]; exists {
budget.Used += cost

if budget.MonthlyLimit > 0 {
pct := budget.Used / budget.MonthlyLimit * 100
cb := ct.alertCb

if pct >= 100 && !budget.Alert100Sent {
budget.Alert100Sent = true
if cb != nil {
go cb(tenantID, 100, budget)
}
} else if pct >= 90 && !budget.Alert90Sent {
budget.Alert90Sent = true
if cb != nil {
go cb(tenantID, 90, budget)
}
} else if pct >= 80 && !budget.Alert80Sent {
budget.Alert80Sent = true
if cb != nil {
go cb(tenantID, 80, budget)
}
}
}
}
}

func (ct *CostTracker) SetBudget(tenantID string, monthlyLimit float64) {
ct.mu.Lock()
defer ct.mu.Unlock()

ct.limits[tenantID] = &Budget{
TenantID:     tenantID,
MonthlyLimit: monthlyLimit,
Used:         0,
ResetDate:    time.Now().Add(30 * 24 * time.Hour),
}
}

func (ct *CostTracker) GetBudget(tenantID string) (*Budget, bool) {
ct.mu.RLock()
budget, exists := ct.limits[tenantID]
ct.mu.RUnlock()

if !exists {
return nil, false
}

// Resetear si pasó el mes
if time.Now().After(budget.ResetDate) {
ct.mu.Lock()
budget.Used = 0
budget.ResetDate = time.Now().Add(30 * 24 * time.Hour)
budget.Alert80Sent = false
budget.Alert90Sent = false
budget.Alert100Sent = false
ct.mu.Unlock()
}

return budget, true
}

func (ct *CostTracker) CheckBudget(tenantID string, estimatedCost float64) (bool, float64) {
budget, exists := ct.GetBudget(tenantID)
if !exists {
return true, 0
}

remaining := budget.MonthlyLimit - budget.Used
if remaining < estimatedCost {
return false, remaining
}
return true, remaining
}

func (ct *CostTracker) GetUsage(tenantID string, days int) []CostEntry {
ct.mu.RLock()
defer ct.mu.RUnlock()

var result []CostEntry
since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
for _, entry := range ct.entries {
if entry.TenantID == tenantID && entry.Timestamp.After(since) {
result = append(result, entry)
}
}
return result
}

func (ct *CostTracker) GetTotalCost(tenantID string, days int) float64 {
entries := ct.GetUsage(tenantID, days)
var total float64
for _, entry := range entries {
total += entry.CostUSD
}
return total
}

func generateRequestID() string {
return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, n)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
}
return string(b)
}
