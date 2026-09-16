package controlplane

import (
"encoding/json"
"net/http"

"github.com/gorilla/mux"
"github.com/lecodev-26/sentinelflow/internal/gateway"
"github.com/lecodev-26/sentinelflow/internal/rbac"
)

// OrganizationHandler gestiona organizaciones
type OrganizationHandler struct {
mgr *rbac.OrganizationManager
}

func NewOrganizationHandler(mgr *rbac.OrganizationManager) *OrganizationHandler {
return &OrganizationHandler{mgr: mgr}
}

// Register registra las rutas
func (h *OrganizationHandler) Register(r *mux.Router) {
r.HandleFunc("/admin/organizations", h.List).Methods("GET")
r.HandleFunc("/admin/organizations", h.Create).Methods("POST")
r.HandleFunc("/admin/organizations/{id}", h.Get).Methods("GET")
r.HandleFunc("/admin/organizations/{id}/projects", h.ListProjects).Methods("GET")
r.HandleFunc("/admin/organizations/{id}/projects", h.CreateProject).Methods("POST")
}

// List devuelve todas las organizaciones
func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
// El manager no tiene List() explícito, iteramos por ID (placeholder)
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"organizations": []interface{}{},
})
}

// Create crea una nueva organización
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
var req struct {
Name        string `json:"name"`
Description string `json:"description"`
}
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
gateway.WriteError(w, gateway.NewInvalidRequestError("invalid JSON"))
return
}

if req.Name == "" {
gateway.WriteError(w, gateway.NewInvalidRequestError("name is required"))
return
}

org := h.mgr.CreateOrganization(req.Name, req.Description)
gateway.WriteJSON(w, http.StatusCreated, org)
}

// Get devuelve una organización por ID
func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
id := vars["id"]

org, exists := h.mgr.GetOrganization(id)
if !exists {
gateway.WriteError(w, gateway.NewInvalidRequestError("organization not found"))
return
}

gateway.WriteJSON(w, http.StatusOK, org)
}

// ListProjects lista los proyectos de una organización
func (h *OrganizationHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
orgID := vars["id"]

projects := h.mgr.ListProjects(orgID)
gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
"projects": projects,
})
}

// CreateProject crea un proyecto en una organización
func (h *OrganizationHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
vars := mux.Vars(r)
orgID := vars["id"]

var req struct {
Name        string `json:"name"`
Description string `json:"description"`
}
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
gateway.WriteError(w, gateway.NewInvalidRequestError("invalid JSON"))
return
}

project, err := h.mgr.CreateProject(orgID, req.Name, req.Description)
if err != nil {
gateway.WriteError(w, gateway.NewInvalidRequestError(err.Error()))
return
}

gateway.WriteJSON(w, http.StatusCreated, project)
}
