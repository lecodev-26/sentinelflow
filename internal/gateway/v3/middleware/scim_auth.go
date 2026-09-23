package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// SCIMAuthMiddleware valida un Bearer token estático para endpoints SCIM.
// Este token es independiente de las API keys del gateway y se configura
// via la variable de entorno SCIM_TOKEN.
//
// En producción, el token debería venir de un Secret Manager.
type SCIMAuthMiddleware struct {
	token string
}

// NewSCIMAuthMiddleware lee SCIM_TOKEN del entorno
func NewSCIMAuthMiddleware() *SCIMAuthMiddleware {
	return &SCIMAuthMiddleware{
		token: os.Getenv("SCIM_TOKEN"),
	}
}

// Enabled devuelve true si SCIM_TOKEN está configurado
func (m *SCIMAuthMiddleware) Enabled() bool {
	return m.token != ""
}

// Handler envuelve un handler con validación de Bearer token
func (m *SCIMAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Si no hay SCIM_TOKEN configurado, en dev permitimos (pero avisamos)
		if m.token == "" {
			// En producción, esto debería ser un error fatal en startup.
			// Por ahora, dejamos pasar con un warning.
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeSCIMError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeSCIMError(w, http.StatusUnauthorized, "invalid Authorization format, expected 'Bearer <token>'")
			return
		}

		if parts[1] != m.token {
			writeSCIMError(w, http.StatusUnauthorized, "invalid SCIM token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeSCIMError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		"detail":  message,
		"status":  status,
	})
}
