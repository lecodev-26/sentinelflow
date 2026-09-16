package gateway

import (
"net/http"
"strings"

gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/rbac"
)

// AuthMiddleware valida la API key usando el UserManager real
type AuthMiddleware struct {
userMgr *rbac.UserManager
enabled bool
}

func NewAuthMiddleware(userMgr *rbac.UserManager, enabled bool) *AuthMiddleware {
return &AuthMiddleware{
userMgr: userMgr,
enabled: enabled,
}
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
rc, ok := gwcontext.FromContext(r.Context())
if !ok {
WriteError(w, NewInternalError("missing request context", nil))
return
}

if !m.enabled || m.userMgr == nil {
// Auth desactivado → identidad por defecto
rc.TenantID = "default"
rc.UserID = "anonymous"
rc.Role = "admin"
next.ServeHTTP(w, r)
return
}

// Extraer API key
authHeader := r.Header.Get("Authorization")
if authHeader == "" {
WriteError(w, NewAuthenticationError("missing Authorization header"))
return
}

parts := strings.SplitN(authHeader, " ", 2)
if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
WriteError(w, NewAuthenticationError("invalid Authorization format, expected 'Bearer <key>'"))
return
}

rawKey := parts[1]

// Validar contra el UserManager real
apiKey, user, valid := m.userMgr.ValidateAPIKey(rawKey)
if !valid {
logger.Warnf("🔒 Auth fallida desde %s", rc.IP)
WriteError(w, NewAuthenticationError("invalid or expired API key"))
return
}

// Rellenar el RequestContext
rc.APIKeyID = apiKey.ID
rc.UserID = user.ID
rc.TenantID = user.OrgID
rc.Role = string(user.Role)
if apiKey.ProjectID != "" {
rc.ProjectID = apiKey.ProjectID
}

logger.Infof("✅ Auth OK: user=%s tenant=%s role=%s", user.ID, user.OrgID, user.Role)

next.ServeHTTP(w, r)
})
}
