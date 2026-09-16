package controlplane

import (
"net/http"

"github.com/gorilla/mux"
"github.com/lecodev-26/sentinelflow/internal/gateway"
"github.com/lecodev-26/sentinelflow/internal/proxy"
)

type ProviderHandler struct {
proxy *proxy.Proxy
}

func NewProviderHandler(p *proxy.Proxy) *ProviderHandler {
return &ProviderHandler{proxy: p}
}

func (h *ProviderHandler) Register(r *mux.Router) {
r.HandleFunc("/admin/providers", h.List).Methods("GET")
r.HandleFunc("/admin/circuit-breakers", h.CircuitBreakers).Methods("GET")
r.HandleFunc("/admin/budgets", h.ListBudgets).Methods("GET")
}

// List devuelve el estado de todos los providers
func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
status := h.proxy.GetProvidersStatusMap()
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"providers": status,
})
}

// CircuitBreakers devuelve el estado de los circuit breakers
func (h *ProviderHandler) CircuitBreakers(w http.ResponseWriter, r *http.Request) {
status := h.proxy.GetCircuitBreakerStatus()
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"circuit_breakers": status,
})
}

// ListBudgets devuelve los budgets por tenant
func (h *ProviderHandler) ListBudgets(w http.ResponseWriter, r *http.Request) {
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"budgets": []interface{}{},
})
}
