package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/lecodev-26/sentinelflow/internal/rbac"
)

// RequireScope devuelve un middleware que exige un scope específico
func RequireScope(scope rbac.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verificar si el contexto tiene el scope
			if !HasScope(r.Context(), string(scope)) {
				writeScopeError(w, string(scope))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyScope exige AL MENOS UNO de los scopes
func RequireAnyScope(scopes ...rbac.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, s := range scopes {
				if HasScope(r.Context(), string(s)) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeScopeError(w, "one of: "+scopesList(scopes))
		})
	}
}

func writeScopeError(w http.ResponseWriter, required string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"type":     "insufficient_scope",
			"message":  "missing required scope: " + required,
			"required": required,
		},
	})
}

func scopesList(scopes []rbac.Scope) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += ", "
		}
		result += string(s)
	}
	return result
}
