package rbac

import (
"context"
"net/http"
"strings"
)

// Middleware es un middleware de RBAC
type Middleware struct {
userMgr *UserManager
}

// NewMiddleware crea un nuevo middleware de RBAC
func NewMiddleware(userMgr *UserManager) *Middleware {
return &Middleware{userMgr: userMgr}
}

// Handler envuelve un handler con autenticación y autorización
func (m *Middleware) Handler(next http.HandlerFunc, requiredPermission Permission) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
// Extraer API key del header
authHeader := r.Header.Get("Authorization")
if authHeader == "" {
http.Error(w, "Authorization header required", http.StatusUnauthorized)
return
}

parts := strings.Split(authHeader, " ")
if len(parts) != 2 || parts[0] != "Bearer" {
http.Error(w, "Invalid Authorization format", http.StatusUnauthorized)
return
}

key := parts[1]

// Validar API key
apiKey, user, valid := m.userMgr.ValidateAPIKey(key)
if !valid {
http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
return
}

// Verificar permiso
if !CheckAccess(user.Role, r.URL.Path, requiredPermission) {
http.Error(w, "Insufficient permissions", http.StatusForbidden)
return
}

// Añadir información al contexto
ctx := context.WithValue(r.Context(), "user_id", user.ID)
ctx = context.WithValue(ctx, "api_key_id", apiKey.ID)
ctx = context.WithValue(ctx, "org_id", user.OrgID)
ctx = context.WithValue(ctx, "role", user.Role)

next(w, r.WithContext(ctx))
}
}
