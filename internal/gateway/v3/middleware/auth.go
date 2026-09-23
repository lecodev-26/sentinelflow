package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lecodev-26/sentinelflow/internal/identity"
	"github.com/lecodev-26/sentinelflow/internal/logger"
)

// Context keys
type contextKey string

const (
	CtxTenantID  contextKey = "tenant_id"
	CtxOrgID     contextKey = "org_id"
	CtxProjectID contextKey = "project_id"
	CtxUserID    contextKey = "user_id"
	CtxAPIKeyID  contextKey = "api_key_id"
	CtxScopes    contextKey = "scopes"
	CtxResidency contextKey = "residency"
)

// Auth valida API keys contra PostgreSQL
type Auth struct {
	svc     *identity.Service
	enabled bool
}

// NewAuth crea un nuevo middleware de auth
func NewAuth(svc *identity.Service, enabled bool) *Auth {
	return &Auth{svc: svc, enabled: enabled}
}

// Handler envuelve un handler con validación de API key
func (a *Auth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth para health/version
		if r.URL.Path == "/health" || r.URL.Path == "/version" ||
			r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		if !a.enabled {
			// Dev mode: identidad por defecto
			ctx := context.WithValue(r.Context(), CtxTenantID, "default")
			ctx = context.WithValue(ctx, CtxOrgID, "default")
			ctx = context.WithValue(ctx, CtxUserID, "anonymous")
			ctx = context.WithValue(ctx, CtxResidency, "global")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Extraer API key
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAuthError(w, "missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeAuthError(w, "invalid Authorization format")
			return
		}

		rawKey := parts[1]

		// Validar contra Identity Service
		apiKey, user, err := a.svc.ValidateAPIKey(r.Context(), rawKey)
		if err != nil {
			logger.Warnf("🔒 Auth failed: %v", err)
			writeAuthError(w, "invalid or expired API key")
			return
		}

		// Cargar residencia del tenant (org)
		residency := "global"
		if org, err := a.svc.GetOrganization(r.Context(), user.OrgID); err == nil {
			if org.Residency != "" {
				residency = org.Residency
			}
		}

		logger.Infof("✅ Auth OK: user=%s org=%s residency=%s scopes=%v",
			user.ID, user.OrgID, residency, apiKey.Scopes)

		// Inyectar contexto
		ctx := context.WithValue(r.Context(), CtxTenantID, user.OrgID)
		ctx = context.WithValue(ctx, CtxOrgID, user.OrgID)
		ctx = context.WithValue(ctx, CtxProjectID, apiKey.ProjectID)
		ctx = context.WithValue(ctx, CtxUserID, user.ID)
		ctx = context.WithValue(ctx, CtxAPIKeyID, apiKey.ID)
		ctx = context.WithValue(ctx, CtxScopes, apiKey.Scopes)
		ctx = context.WithValue(ctx, CtxResidency, residency)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HasScope verifica si el contexto tiene un scope
func HasScope(ctx context.Context, scope string) bool {
	scopes, ok := ctx.Value(CtxScopes).([]string)
	if !ok {
		return false
	}
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// GetTenantID devuelve el tenant ID del contexto
func GetTenantID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxTenantID).(string); ok {
		return v
	}
	return ""
}

// GetOrgID devuelve el org ID del contexto
func GetOrgID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxOrgID).(string); ok {
		return v
	}
	return ""
}

// GetProjectID devuelve el project ID del contexto
func GetProjectID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxProjectID).(string); ok {
		return v
	}
	return ""
}

// GetUserID devuelve el user ID del contexto
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxUserID).(string); ok {
		return v
	}
	return ""
}

// GetAPIKeyID devuelve el API key ID del contexto
func GetAPIKeyID(ctx context.Context) string {
	if v, ok := ctx.Value(CtxAPIKeyID).(string); ok {
		return v
	}
	return ""
}

// GetResidency devuelve la residencia del contexto
func GetResidency(ctx context.Context) string {
	if v, ok := ctx.Value(CtxResidency).(string); ok {
		return v
	}
	return "global"
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"type":    "authentication_error",
			"message": message,
		},
	})
}
