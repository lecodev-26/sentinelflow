package controlplane

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/gateway"
	"github.com/lecodev-26/sentinelflow/internal/rbac"
)

// UserHandler gestiona usuarios y API keys
type UserHandler struct {
	mgr *rbac.UserManager
}

func NewUserHandler(mgr *rbac.UserManager) *UserHandler {
	return &UserHandler{mgr: mgr}
}

func (h *UserHandler) Register(r *mux.Router) {
	r.HandleFunc("/admin/users", h.Create).Methods("POST")
	r.HandleFunc("/admin/users/{id}", h.Get).Methods("GET")
	r.HandleFunc("/admin/users/{id}/api-keys", h.CreateAPIKey).Methods("POST")
	r.HandleFunc("/admin/api-keys/{key}/revoke", h.RevokeAPIKey).Methods("POST")
}

// Create crea un nuevo usuario
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
		OrgID string `json:"org_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		gateway.WriteError(w, gateway.NewInvalidRequestError("invalid JSON"))
		return
	}

	if req.Email == "" || req.OrgID == "" {
		gateway.WriteError(w, gateway.NewInvalidRequestError("email and org_id are required"))
		return
	}

	role := rbac.Role(req.Role)
	if role == "" {
		role = rbac.RoleViewer
	}

	user, err := h.mgr.CreateUser(req.Email, req.Name, role, req.OrgID)
	if err != nil {
		gateway.WriteError(w, gateway.NewInvalidRequestError(err.Error()))
		return
	}

	gateway.WriteJSON(w, http.StatusCreated, user)
}

// Get devuelve un usuario por ID
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, exists := h.mgr.GetUser(id)
	if !exists {
		gateway.WriteError(w, gateway.NewInvalidRequestError("user not found"))
		return
	}

	gateway.WriteJSON(w, http.StatusOK, user)
}

// CreateAPIKey crea una nueva API key para un usuario
func (h *UserHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	var req struct {
		Name      string   `json:"name"`
		ProjectID string   `json:"project_id"`
		Scopes    []string `json:"scopes"`
		TTLHours  int      `json:"ttl_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		gateway.WriteError(w, gateway.NewInvalidRequestError("invalid JSON"))
		return
	}

	// Convertir scopes
	var scopes []rbac.Scope
	for _, s := range req.Scopes {
		scopes = append(scopes, rbac.Scope(s))
	}

	// TTL
	ttl := time.Duration(req.TTLHours) * time.Hour
	if ttl == 0 {
		ttl = 365 * 24 * time.Hour
	}

	rawKey, apiKey, err := h.mgr.CreateAPIKey(userID, req.Name, req.ProjectID, scopes, ttl)
	if err != nil {
		gateway.WriteError(w, gateway.NewInvalidRequestError(err.Error()))
		return
	}

	// Devolver la key en claro UNA SOLA VEZ
	gateway.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"api_key": apiKey,
		"key":     rawKey,
		"warning": "Esta clave solo se muestra una vez. Guárdala de forma segura.",
	})
}

// RevokeAPIKey revoca una API key
func (h *UserHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	if !h.mgr.RevokeAPIKey(key) {
		gateway.WriteError(w, gateway.NewInvalidRequestError("api key not found"))
		return
	}

	gateway.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": "revoked",
	})
}
