package gateway

import (
"net/http"

gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
"github.com/lecodev-26/sentinelflow/internal/logger"
)

// AuthMiddleware valida la API key y extrae la identidad
type AuthMiddleware struct {
enabled bool
}

func NewAuthMiddleware(enabled bool) *AuthMiddleware {
return &AuthMiddleware{enabled: enabled}
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
rc, ok := gwcontext.FromContext(r.Context())
if !ok {
WriteError(w, NewInternalError("missing request context", nil))
return
}

if !m.enabled {
// Auth desactivado → identidad por defecto
rc.TenantID = "default"
rc.UserID = "anonymous"
rc.Role = "viewer"
next.ServeHTTP(w, r)
return
}

// Extraer API key del header
authHeader := r.Header.Get("Authorization")
if authHeader == "" {
WriteError(w, NewAuthenticationError("missing Authorization header"))
return
}

// TODO: integrar con rbac.UserManager.ValidateAPIKey
// Por ahora aceptamos cualquier Bearer como demo
logger.Infof("🔐 Auth header presente para %s", rc.RequestID)
rc.TenantID = "default"
rc.UserID = "user"
rc.Role = "admin"

next.ServeHTTP(w, r)
})
}
