package v3

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/gateway/v3/middleware"
	"github.com/lecodev-26/sentinelflow/internal/identity"
	"github.com/lecodev-26/sentinelflow/internal/rbac"
)

// Handlers agrupa todos los handlers del Control Plane V3
type Handlers struct {
	svc    *identity.Service
	authMw *middleware.Auth
}

// New crea nuevos handlers (sin auth, para tests)
func New(svc *identity.Service) *Handlers {
	return &Handlers{svc: svc}
}

// NewWithAuth crea handlers con auth middleware (para producción)
func NewWithAuth(svc *identity.Service, authMw *middleware.Auth) *Handlers {
	return &Handlers{
		svc:    svc,
		authMw: authMw,
	}
}

// Register registra todas las rutas con auth + scopes
func (h *Handlers) Register(r *mux.Router) {
	api := r.PathPrefix("/v1").Subrouter()

	// Si hay authMw, aplicarlo a todo /v1/*
	if h.authMw != nil {
		api.Use(h.authMw.Handler)
	}

	// === READ endpoints (admin:read) ===
	read := api.PathPrefix("").Subrouter()
	if h.authMw != nil {
		read.Use(middleware.RequireScope(rbac.ScopeReadAdmin))
	}
	read.HandleFunc("/organizations", h.ListOrganizations).Methods("GET")
	read.HandleFunc("/organizations/{id}", h.GetOrganization).Methods("GET")
	read.HandleFunc("/organizations/{id}/projects", h.ListProjects).Methods("GET")
	read.HandleFunc("/projects/{id}", h.GetProject).Methods("GET")
	read.HandleFunc("/organizations/{id}/users", h.ListUsers).Methods("GET")
	read.HandleFunc("/users/{id}", h.GetUser).Methods("GET")
	read.HandleFunc("/users/{id}/api-keys", h.ListAPIKeys).Methods("GET")

	// === WRITE endpoints (admin:write) ===
	write := api.PathPrefix("").Subrouter()
	if h.authMw != nil {
		write.Use(middleware.RequireScope(rbac.ScopeWriteAdmin))
	}
	write.HandleFunc("/organizations", h.CreateOrganization).Methods("POST")
	write.HandleFunc("/organizations/{id}", h.DeleteOrganization).Methods("DELETE")
	write.HandleFunc("/organizations/{id}/projects", h.CreateProject).Methods("POST")
	write.HandleFunc("/projects/{id}", h.DeleteProject).Methods("DELETE")
	write.HandleFunc("/users", h.CreateUser).Methods("POST")
	write.HandleFunc("/users/{id}", h.DeleteUser).Methods("DELETE")
	write.HandleFunc("/users/{id}/api-keys", h.CreateAPIKey).Methods("POST")
	write.HandleFunc("/api-keys/{id}", h.RevokeAPIKey).Methods("DELETE")

	// === Webhooks ===
	h.RegisterWebhooks(api)
}

// === ORGANIZATIONS ===

func (h *Handlers) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.svc.ListOrganizations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  orgs,
		"total": len(orgs),
	})
}

func (h *Handlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}

	org, err := h.svc.CreateOrganization(r.Context(), req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, org)
}

func (h *Handlers) GetOrganization(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	org, err := h.svc.GetOrganization(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "organization not found")
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h *Handlers) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteOrganization(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "organization not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// === PROJECTS ===

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["id"]
	projects, err := h.svc.ListProjects(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  projects,
		"total": len(projects),
	})
}

func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["id"]
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}

	project, err := h.svc.CreateProject(r.Context(), orgID, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (h *Handlers) GetProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	project, err := h.svc.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "project not found")
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (h *Handlers) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteProject(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "project not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// === USERS ===

func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	orgID := mux.Vars(r)["id"]
	users, err := h.svc.ListUsers(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  users,
		"total": len(users),
	})
}

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrgID string `json:"org_id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}

	user, err := h.svc.CreateUser(r.Context(), req.OrgID, req.Email, req.Name, req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	user, err := h.svc.GetUser(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.svc.DeleteUser(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// === API KEYS ===

func (h *Handlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]
	keys, err := h.svc.ListAPIKeys(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  keys,
		"total": len(keys),
	})
}

func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]

	var req struct {
		Name      string   `json:"name"`
		ProjectID string   `json:"project_id"`
		Scopes    []string `json:"scopes"`
		TTLHours  int      `json:"ttl_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
		return
	}

	ttl := time.Duration(req.TTLHours) * time.Hour
	if ttl == 0 {
		ttl = 365 * 24 * time.Hour
	}

	rawKey, key, err := h.svc.CreateAPIKey(r.Context(), userID, req.ProjectID, req.Name, req.Scopes, ttl)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"api_key": key,
		"key":     rawKey,
		"warning": "Esta clave solo se muestra una vez. Guárdala de forma segura.",
	})
}

func (h *Handlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.svc.RevokeAPIKey(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "api key not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// === HELPERS ===

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"type":    errType,
			"message": message,
		},
	})
}
