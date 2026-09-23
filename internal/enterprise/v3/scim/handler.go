package scim

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/identity"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

// User representa un usuario SCIM
type User struct {
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
		Type    string `json:"type,omitempty"`
	} `json:"emails"`
	Active bool   `json:"active"`
	OrgID  string `json:"orgId,omitempty"`
	Meta   struct {
		ResourceType string `json:"resourceType,omitempty"`
		Location     string `json:"location,omitempty"`
	} `json:"meta,omitempty"`
}

// Handler maneja SCIM 2.0 endpoints
type Handler struct {
	identitySvc *identity.Service
}

// NewHandler crea un nuevo handler SCIM
func NewHandler(svc *identity.Service) *Handler {
	return &Handler{identitySvc: svc}
}

// Register registra las rutas
func (h *Handler) Register(r *mux.Router) {
	scim := r.PathPrefix("/scim/v2").Subrouter()
	scim.HandleFunc("/Users", h.handleUsers).Methods("GET", "POST")
	scim.HandleFunc("/Users/{id}", h.handleUser).Methods("GET", "PATCH", "PUT", "DELETE")
	scim.HandleFunc("/ServiceProviderConfig", h.handleConfig).Methods("GET")
	scim.HandleFunc("/Schemas", h.handleSchemas).Methods("GET")
}

func (h *Handler) handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")

	switch r.Method {
	case http.MethodGet:
		h.listUsers(w, r)
	case http.MethodPost:
		h.createUser(w, r)
	}
}

func (h *Handler) handleUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	id := mux.Vars(r)["id"]

	switch r.Method {
	case http.MethodGet:
		h.getUser(w, r, id)
	case http.MethodDelete:
		h.deleteUser(w, r, id)
	case http.MethodPatch, http.MethodPut:
		h.updateUser(w, r, id)
	}
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
			"totalResults": 0,
			"Resources":    []interface{}{},
		})
		return
	}

	users, err := h.identitySvc.ListUsers(r.Context(), orgID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resources := make([]User, 0, len(users))
	for _, u := range users {
		resources = append(resources, scimUserFromIdentity(u))
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": len(resources),
		"startIndex":   1,
		"itemsPerPage": len(resources),
		"Resources":    resources,
	})
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var scimUser User
	if err := json.NewDecoder(r.Body).Decode(&scimUser); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if scimUser.UserName == "" || scimUser.OrgID == "" || len(scimUser.Emails) == 0 {
		h.writeError(w, http.StatusBadRequest, "userName, orgId, and email required")
		return
	}

	email := scimUser.Emails[0].Value
	name := scimUser.Name.Formatted
	if name == "" {
		name = scimUser.UserName
	}

	user, err := h.identitySvc.CreateUser(r.Context(), scimUser.OrgID, email, name, "viewer")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	logger.Infof("👥 SCIM user created: %s (%s)", user.Email, user.ID)

	result := scimUserFromIdentity(user)
	result.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	result.Meta.ResourceType = "User"
	result.Meta.Location = "/scim/v2/Users/" + user.ID

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request, id string) {
	user, err := h.identitySvc.GetUser(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "user not found")
		return
	}

	result := scimUserFromIdentity(user)
	result.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	result.Meta.ResourceType = "User"

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.identitySvc.DeleteUser(r.Context(), id); err != nil {
		h.writeError(w, http.StatusNotFound, "user not found")
		return
	}
	logger.Infof("👥 SCIM user deleted: %s", id)
	w.WriteHeader(http.StatusNoContent)
}

// updateUser aplica cambios reales en la base de datos
func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request, id string) {
	// 1. Verificar que el usuario existe
	_, err := h.identitySvc.GetUser(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "user not found")
		return
	}

	// 2. Parsear el SCIM User recibido
	var scimUser User
	if err := json.NewDecoder(r.Body).Decode(&scimUser); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// 3. Convertir a UserUpdates (solo campos definidos)
	updates := identity.UserUpdates{}

	if scimUser.Name.Formatted != "" {
		name := scimUser.Name.Formatted
		updates.Name = &name
	} else if scimUser.Name.GivenName != "" || scimUser.Name.FamilyName != "" {
		name := strings.TrimSpace(scimUser.Name.GivenName + " " + scimUser.Name.FamilyName)
		updates.Name = &name
	}

	if len(scimUser.Emails) > 0 && scimUser.Emails[0].Value != "" {
		email := scimUser.Emails[0].Value
		updates.Email = &email
	}

	// Aplicar active solo si el body traía el campo.
	// Como json.Decoder no distingue "no enviado" de "false",
	// siempre lo aplicamos (comportamiento SCIM estándar para PUT).
	updates.Active = &scimUser.Active

	// 4. Aplicar update en la DB
	updated, err := h.identitySvc.UpdateUser(r.Context(), id, updates)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	logger.Infof("👥 SCIM user updated: %s (%s)", updated.Email, updated.ID)

	// 5. Devolver el usuario actualizado en formato SCIM
	result := scimUserFromIdentity(updated)
	result.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	result.Meta.ResourceType = "User"
	result.Meta.Location = "/scim/v2/Users/" + updated.ID

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas":          []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"documentationUri": "https://github.com/lecodev-26/sentinelflow",
		"patch":            map[string]bool{"supported": true},
		"bulk":             map[string]bool{"supported": false},
		"filter":           map[string]interface{}{"supported": true, "maxResults": 100},
		"changePassword":   map[string]bool{"supported": false},
		"sort":             map[string]bool{"supported": true},
		"etag":             map[string]bool{"supported": false},
	})
}

func (h *Handler) handleSchemas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/scim+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": 1,
		"Resources": []map[string]string{
			{"id": "urn:ietf:params:scim:schemas:core:2.0:User", "name": "User"},
		},
	})
}

func scimUserFromIdentity(u *postgres.User) User {
	result := User{
		ID:       u.ID,
		UserName: u.Email,
		Active:   u.Active,
		OrgID:    u.OrgID,
	}
	result.Name.Formatted = u.Name
	result.Emails = []struct {
		Value   string `json:"value"`
		Primary bool   `json:"primary,omitempty"`
		Type    string `json:"type,omitempty"`
	}{
		{Value: u.Email, Primary: true, Type: "work"},
	}
	return result
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		"detail":  message,
		"status":  status,
	})
}
