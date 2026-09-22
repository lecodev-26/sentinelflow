package gateway

import (
	"net/http"
	"os"
	"strings"

	gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/rbac"
)

// Environment representa el entorno de ejecución
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// AuthMiddleware valida la API key usando el UserManager real
type AuthMiddleware struct {
	userMgr *rbac.UserManager
	mode    Environment
}

// NewAuthMiddleware crea el middleware en modo dev (auth opcional)
func NewAuthMiddleware(userMgr *rbac.UserManager, mode Environment) *AuthMiddleware {
	return &AuthMiddleware{
		userMgr: userMgr,
		mode:    mode,
	}
}

// NewAuthMiddlewareFromEnv crea el middleware leyendo SENTINELFLOW_ENV
func NewAuthMiddlewareFromEnv(userMgr *rbac.UserManager) (*AuthMiddleware, error) {
	env := os.Getenv("SENTINELFLOW_ENV")
	if env == "" {
		env = "development"
	}

	mode := Environment(env)

	// Validación: en producción no se permite auth desactivada
	if mode == EnvProduction && userMgr == nil {
		return nil, ErrProductionAuthRequired
	}

	return &AuthMiddleware{
		userMgr: userMgr,
		mode:    mode,
	}, nil
}

// ErrProductionAuthRequired se devuelve si producción no tiene auth configurada
var ErrProductionAuthRequired = &GatewayError{
	Type:       ErrTypeInternal,
	Message:    "production environment requires auth to be enabled",
	StatusCode: http.StatusInternalServerError,
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc, ok := gwcontext.FromContext(r.Context())
		if !ok {
			WriteError(w, NewInternalError("missing request context", nil))
			return
		}

		// En desarrollo, si no hay userMgr, usar identidad por defecto
		if m.mode == EnvDevelopment && m.userMgr == nil {
			rc.TenantID = "default"
			rc.UserID = "anonymous"
			rc.Role = string(rbac.RoleAdmin)
			next.ServeHTTP(w, r)
			return
		}

		// En staging/producción o con userMgr: validar API key
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

		if m.userMgr == nil {
			WriteError(w, NewAuthenticationError("authentication not configured"))
			return
		}

		apiKey, user, valid := m.userMgr.ValidateAPIKey(rawKey)
		if !valid {
			logger.Warnf("🔒 Auth fallida desde %s", rc.IP)
			WriteError(w, NewAuthenticationError("invalid or expired API key"))
			return
		}

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
