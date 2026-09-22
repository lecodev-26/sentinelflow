package scim

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SCIMUser representa un usuario en formato SCIM
type SCIMUser struct {
	Schemas    []string `json:"schemas"`
	ID         string   `json:"id"`
	ExternalID string   `json:"externalId,omitempty"`
	UserName   string   `json:"userName"`
	Name       struct {
		GivenName  string `json:"givenName,omitempty"`
		FamilyName string `json:"familyName,omitempty"`
		Formatted  string `json:"formatted,omitempty"`
	} `json:"name,omitempty"`
	Emails []struct {
		Value   string `json:"value"`
		Primary bool   `json:"primary,omitempty"`
	} `json:"emails"`
	Active    bool      `json:"active"`
	OrgID     string    `json:"orgId,omitempty"`
	CreatedAt time.Time `json:"meta.created,omitempty"`
	UpdatedAt time.Time `json:"meta.lastModified,omitempty"`
}

// SCIMListResponse respuesta de lista SCIM
type SCIMListResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	StartIndex   int         `json:"startIndex"`
	ItemsPerPage int         `json:"itemsPerPage"`
	Resources    []*SCIMUser `json:"Resources"`
}

// Handler maneja los endpoints SCIM
type Handler struct {
	mu    sync.RWMutex
	users map[string]*SCIMUser
}

// NewHandler crea un nuevo handler SCIM
func NewHandler() *Handler {
	return &Handler{users: make(map[string]*SCIMUser)}
}

// Register registra las rutas SCIM
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/scim/v2/Users", h.handleUsers)
	mux.HandleFunc("/scim/v2/Users/", h.handleUser)
	mux.HandleFunc("/scim/v2/ServiceProviderConfig", h.handleServiceProviderConfig)
	mux.HandleFunc("/scim/v2/Schemas", h.handleSchemas)
}

// List lista usuarios
func (h *Handler) List() []*SCIMUser {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]*SCIMUser, 0, len(h.users))
	for _, u := range h.users {
		result = append(result, u)
	}
	return result
}

// Create crea un usuario
func (h *Handler) Create(user *SCIMUser) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	user.Active = true
	h.users[user.ID] = user
	return nil
}

// Delete elimina un usuario
func (h *Handler) Delete(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.users[id]; !exists {
		return false
	}
	delete(h.users, id)
	return true
}

func (h *Handler) handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")

	switch r.Method {
	case http.MethodGet:
		users := h.List()
		resp := SCIMListResponse{
			Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
			TotalResults: len(users),
			StartIndex:   1,
			ItemsPerPage: len(users),
			Resources:    users,
		}
		json.NewEncoder(w).Encode(resp)

	case http.MethodPost:
		var user SCIMUser
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if user.ID == "" {
			user.ID = generateSCIMID()
		}
		h.Create(&user)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleUser(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/scim/v2/Users/")
	w.Header().Set("Content-Type", "application/scim+json")

	switch r.Method {
	case http.MethodGet:
		h.mu.RLock()
		user, exists := h.users[id]
		h.mu.RUnlock()
		if !exists {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(user)

	case http.MethodDelete:
		if !h.Delete(id) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas":        []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch":          map[string]bool{"supported": true},
		"bulk":           map[string]bool{"supported": false},
		"filter":         map[string]bool{"supported": true},
		"changePassword": map[string]bool{"supported": false},
		"sort":           map[string]bool{"supported": true},
	})
}

func (h *Handler) handleSchemas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"Resources": []map[string]string{
			{"id": "urn:ietf:params:scim:schemas:core:2.0:User", "name": "User"},
		},
	})
}

func generateSCIMID() string {
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
