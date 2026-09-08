package security

import (
"context"
"net/http"
"strings"
)

// Context keys
type contextKey string

const (
TenantIDKey contextKey = "tenant_id"
APIKeyIDKey  contextKey = "api_key_id"
)

// AuthMiddleware es un middleware de autenticación
type AuthMiddleware struct {
apiKeyManager *APIKeyManager
tenantManager *TenantManager
quotaManager  *QuotaManager
}

// NewAuthMiddleware crea un nuevo middleware
func NewAuthMiddleware(apiKeyMgr *APIKeyManager, tenantMgr *TenantManager, quotaMgr *QuotaManager) *AuthMiddleware {
return &AuthMiddleware{
apiKeyManager: apiKeyMgr,
tenantManager: tenantMgr,
quotaManager:  quotaMgr,
}
}

// Handler envuelve un handler con autenticación
func (m *AuthMiddleware) Handler(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
// Extraer API key del header
authHeader := r.Header.Get("Authorization")
if authHeader == "" {
http.Error(w, "Authorization header required", http.StatusUnauthorized)
return
}

// Verificar formato Bearer
parts := strings.Split(authHeader, " ")
if len(parts) != 2 || parts[0] != "Bearer" {
http.Error(w, "Invalid Authorization format. Use Bearer <key>", http.StatusUnauthorized)
return
}

key := parts[1]

// Validar clave
apiKey, valid := m.apiKeyManager.ValidateKey(key)
if !valid {
http.Error(w, "Invalid or inactive API key", http.StatusUnauthorized)
return
}

// Verificar tenant
tenant, exists := m.tenantManager.GetTenant(apiKey.TenantID)
if !exists {
http.Error(w, "Tenant not found", http.StatusUnauthorized)
return
}

// Verificar cuota (solo para requests de chat)
if strings.Contains(r.URL.Path, "/chat/completions") {
allowed, err := m.quotaManager.CheckAndRecord(tenant.ID, tenant.Quotas, 100) // 100 tokens estimados
if err != nil || !allowed {
http.Error(w, "Quota exceeded", http.StatusTooManyRequests)
return
}
}

// Añadir información al contexto
ctx := context.WithValue(r.Context(), TenantIDKey, tenant.ID)
ctx = context.WithValue(ctx, APIKeyIDKey, apiKey.ID)
r = r.WithContext(ctx)

next(w, r)
}
}
