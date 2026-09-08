package cost

import (
"encoding/json"
"net/http"
)

// DashboardHandler maneja las peticiones del dashboard de costes
type DashboardHandler struct {
tracker *CostTracker
}

// NewDashboardHandler crea un nuevo handler
func NewDashboardHandler(tracker *CostTracker) *DashboardHandler {
return &DashboardHandler{tracker: tracker}
}

// GetSummary devuelve un resumen de costes
func (h *DashboardHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
tenantID := r.URL.Query().Get("tenant")
if tenantID == "" {
tenantID = "default"
}

days := 30
if d := r.URL.Query().Get("days"); d != "" {
// Parsear días si es necesario
}

entries := h.tracker.GetUsage(tenantID, days)
totalCost := h.tracker.GetTotalCost(tenantID, days)

// Agrupar por proveedor
byProvider := make(map[string]float64)
byModel := make(map[string]float64)

for _, entry := range entries {
byProvider[entry.Provider] += entry.CostUSD
byModel[entry.Model] += entry.CostUSD
}

// Obtener budget
budget, _ := h.tracker.GetBudget(tenantID)

response := map[string]interface{}{
"tenant":      tenantID,
"days":        days,
"total":       totalCost,
"by_provider": byProvider,
"by_model":    byModel,
"requests":    len(entries),
}

if budget != nil {
response["budget"] = map[string]interface{}{
"limit":     budget.MonthlyLimit,
"used":      budget.Used,
"remaining": budget.MonthlyLimit - budget.Used,
"reset":     budget.ResetDate,
}
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)
}

// GetDailyCost devuelve coste diario
func (h *DashboardHandler) GetDailyCost(w http.ResponseWriter, r *http.Request) {
tenantID := r.URL.Query().Get("tenant")
if tenantID == "" {
tenantID = "default"
}

entries := h.tracker.GetUsage(tenantID, 30)

// Agrupar por día
daily := make(map[string]float64)
for _, entry := range entries {
day := entry.Timestamp.Format("2006-01-02")
daily[day] += entry.CostUSD
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(daily)
}
