package controlplane

import (
"net/http"

"github.com/gorilla/mux"
)

// Router agrupa todos los handlers del control plane
type Router struct {
orgs      *OrganizationHandler
users     *UserHandler
providers *ProviderHandler
metrics   *MetricsHandler
}

// NewRouter crea el router completo del control plane
func NewRouter(
orgs *OrganizationHandler,
users *UserHandler,
providers *ProviderHandler,
metrics *MetricsHandler,
) *Router {
return &Router{
orgs:      orgs,
users:     users,
providers: providers,
metrics:   metrics,
}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(`{"status":"ok","service":"sentinelflow-control-plane","version":"1.0.0"}`))
}

// Register registra todas las rutas con versionado v1
func (r *Router) Register(router *mux.Router) {
// Version 1 del API
v1 := router.PathPrefix("/v1").Subrouter()
v1.HandleFunc("/organizations", r.orgs.List).Methods("GET")
v1.HandleFunc("/organizations", r.orgs.Create).Methods("POST")
v1.HandleFunc("/organizations/{id}", r.orgs.Get).Methods("GET")
v1.HandleFunc("/organizations/{id}/projects", r.orgs.ListProjects).Methods("GET")
v1.HandleFunc("/organizations/{id}/projects", r.orgs.CreateProject).Methods("POST")
v1.HandleFunc("/users", r.users.Create).Methods("POST")
v1.HandleFunc("/users/{id}", r.users.Get).Methods("GET")
v1.HandleFunc("/users/{id}/api-keys", r.users.CreateAPIKey).Methods("POST")
v1.HandleFunc("/api-keys/{key}/revoke", r.users.RevokeAPIKey).Methods("POST")
v1.HandleFunc("/providers", r.providers.List).Methods("GET")
v1.HandleFunc("/circuit-breakers", r.providers.CircuitBreakers).Methods("GET")
v1.HandleFunc("/budgets", r.providers.ListBudgets).Methods("GET")
v1.HandleFunc("/metrics/overview", r.metrics.Overview).Methods("GET")
v1.HandleFunc("/metrics/providers", r.metrics.Providers).Methods("GET")
v1.HandleFunc("/metrics/security", r.metrics.Security).Methods("GET")
v1.HandleFunc("/metrics/routing", r.metrics.Routing).Methods("GET")
v1.HandleFunc("/metrics/timeseries", r.metrics.Timeseries).Methods("GET")
v1.HandleFunc("/metrics/system", r.metrics.System).Methods("GET")

// Health (sin versión)
router.HandleFunc("/health", healthHandler).Methods("GET")

// Alias legacy /admin/* (deprecated, será removido en v2.0)
admin := router.PathPrefix("/admin").Subrouter()
admin.HandleFunc("/organizations", r.orgs.List).Methods("GET")
admin.HandleFunc("/organizations", r.orgs.Create).Methods("POST")
admin.HandleFunc("/organizations/{id}", r.orgs.Get).Methods("GET")
admin.HandleFunc("/organizations/{id}/projects", r.orgs.ListProjects).Methods("GET")
admin.HandleFunc("/organizations/{id}/projects", r.orgs.CreateProject).Methods("POST")
admin.HandleFunc("/users", r.users.Create).Methods("POST")
admin.HandleFunc("/users/{id}", r.users.Get).Methods("GET")
admin.HandleFunc("/users/{id}/api-keys", r.users.CreateAPIKey).Methods("POST")
admin.HandleFunc("/api-keys/{key}/revoke", r.users.RevokeAPIKey).Methods("POST")
admin.HandleFunc("/providers", r.providers.List).Methods("GET")
admin.HandleFunc("/circuit-breakers", r.providers.CircuitBreakers).Methods("GET")
admin.HandleFunc("/budgets", r.providers.ListBudgets).Methods("GET")
admin.HandleFunc("/metrics/overview", r.metrics.Overview).Methods("GET")
admin.HandleFunc("/metrics/providers", r.metrics.Providers).Methods("GET")
admin.HandleFunc("/metrics/security", r.metrics.Security).Methods("GET")
admin.HandleFunc("/metrics/routing", r.metrics.Routing).Methods("GET")
admin.HandleFunc("/metrics/timeseries", r.metrics.Timeseries).Methods("GET")
admin.HandleFunc("/metrics/system", r.metrics.System).Methods("GET")
admin.HandleFunc("/health", healthHandler).Methods("GET")
}
