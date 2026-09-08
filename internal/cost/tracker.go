package cost

import (
"sync"
"time"
)

// CostEntry registra el coste de una petición
type CostEntry struct {
RequestID    string    `json:"request_id"`
TenantID     string    `json:"tenant_id"`
Provider     string    `json:"provider"`
Model        string    `json:"model"`
InputTokens  int       `json:"input_tokens"`
OutputTokens int       `json:"output_tokens"`
TotalTokens  int       `json:"total_tokens"`
CostUSD      float64   `json:"cost_usd"`
Timestamp    time.Time `json:"timestamp"`
}

// CostTracker rastrea el coste por tenant
type CostTracker struct {
mu      sync.RWMutex
entries []CostEntry
limits  map[string]*Budget
}

// Budget define un presupuesto
type Budget struct {
TenantID     string    `json:"tenant_id"`
MonthlyLimit float64   `json:"monthly_limit"`
Used         float64   `json:"used"`
ResetDate    time.Time `json:"reset_date"`
}

// NewCostTracker crea un nuevo tracker
func NewCostTracker() *CostTracker {
return &CostTracker{
entries: make([]CostEntry, 0),
limits:  make(map[string]*Budget),
}
}

// Record registra una petición
func (ct *CostTracker) Record(tenantID, provider, model string, inputTokens, outputTokens int, cost float64) {
ct.mu.Lock()
defer ct.mu.Unlock()

entry := CostEntry{
RequestID:    generateRequestID(),
TenantID:     tenantID,
Provider:     provider,
Model:        model,
InputTokens:  inputTokens,
OutputTokens: outputTokens,
TotalTokens:  inputTokens + outputTokens,
CostUSD:      cost,
Timestamp:    time.Now(),
}

ct.entries = append(ct.entries, entry)

// Actualizar budget
if budget, exists := ct.limits[tenantID]; exists {
budget.Used += cost
}
}

// SetBudget establece un presupuesto para un tenant
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

// GetBudget devuelve el presupuesto de un tenant
func (ct *CostTracker) GetBudget(tenantID string) (*Budget, bool) {
ct.mu.RLock()
defer ct.mu.RUnlock()

budget, exists := ct.limits[tenantID]
if !exists {
return nil, false
}

// Resetear si pasó el mes
if time.Now().After(budget.ResetDate) {
budget.Used = 0
budget.ResetDate = time.Now().Add(30 * 24 * time.Hour)
}

return budget, true
}

// CheckBudget verifica si un tenant tiene presupuesto
func (ct *CostTracker) CheckBudget(tenantID string, estimatedCost float64) (bool, float64) {
budget, exists := ct.GetBudget(tenantID)
if !exists {
return true, 0 // Sin límite
}

remaining := budget.MonthlyLimit - budget.Used
if remaining < estimatedCost {
return false, remaining
}

return true, remaining
}

// GetUsage devuelve el uso de un tenant
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

// GetTotalCost devuelve el coste total de un tenant
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
