package saml

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"

	crewsaml "github.com/crewjam/saml"
	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/identity"
)

type Handler struct {
	service  *Service
	identity *identity.Service
}

func NewHandler(service *Service, ids *identity.Service) *Handler {
	return &Handler{service: service, identity: ids}
}
func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/auth/saml/metadata", h.metadata).Methods(http.MethodGet)
	r.HandleFunc("/auth/saml/login", h.login).Methods(http.MethodGet)
	r.HandleFunc("/auth/saml/acs", h.acs).Methods(http.MethodPost)
}
func (h *Handler) metadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	b, err := xml.Marshal(h.service.Metadata())
	if err == nil {
		_, err = w.Write(append([]byte(xml.Header), b...))
	}
	if err != nil {
		http.Error(w, "metadata generation failed", 500)
	}
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	u, _, err := h.service.LoginURL()
	if err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	http.Redirect(w, r, u, http.StatusFound)
}
func (h *Handler) acs(w http.ResponseWriter, r *http.Request) {
	assertion, err := h.service.ParseResponse(r)
	if err != nil {
		http.Error(w, "invalid SAML response", http.StatusUnauthorized)
		return
	}
	email := attribute(assertion, "email", "mail", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress")
	name := attribute(assertion, "name", "displayName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name")
	if email == "" && assertion.Subject != nil && assertion.Subject.NameID != nil {
		email = assertion.Subject.NameID.Value
	}
	if name == "" {
		name = email
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		http.Error(w, "SAML assertion has no usable identity", http.StatusUnauthorized)
		return
	}
	user, err := h.identity.GetUserByEmail(r.Context(), email)
	if err != nil {
		orgs, e := h.identity.ListOrganizations(r.Context())
		if e != nil || len(orgs) == 0 {
			http.Error(w, "no organization available", 500)
			return
		}
		user, err = h.identity.CreateUser(r.Context(), orgs[0].ID, email, name, "viewer")
		if err != nil {
			http.Error(w, "user provisioning failed", 500)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"authenticated": true, "user_id": user.ID, "email": user.Email, "name": user.Name, "provider": "saml"})
}
func attribute(a *crewsaml.Assertion, names ...string) string {
	for _, st := range a.AttributeStatements {
		for _, v := range st.Attributes {
			for _, want := range names {
				if strings.EqualFold(v.Name, want) || strings.EqualFold(v.FriendlyName, want) {
					if len(v.Values) > 0 {
						return v.Values[0].Value
					}
				}
			}
		}
	}
	return ""
}
