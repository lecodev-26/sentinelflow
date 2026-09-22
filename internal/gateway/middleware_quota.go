package gateway

import (
	"sync"
	"time"

	gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
	"net/http"

	"github.com/lecodev-26/sentinelflow/internal/logger"
)

// QuotaMiddleware aplica quotas por tenant
type QuotaMiddleware struct {
	mu      sync.Mutex
	usage   map[string]*Usage // tenantID -> Usage
	limits  map[string]*Quota
	enabled bool
}

// Quota define los límites
type Quota struct {
	RequestsPerMinute int
	TokensPerMinute   int
	MonthlyTokens     int
}

// Usage registra el uso actual
type Usage struct {
	Requests    int
	Tokens      int
	MonthTokens int
	WindowStart time.Time
	MonthStart  time.Time
}

// DefaultQuota devuelve una quota estándar
func DefaultQuota() *Quota {
	return &Quota{
		RequestsPerMinute: 1000,
		TokensPerMinute:   100000,
		MonthlyTokens:     10000000,
	}
}

// NewQuotaMiddleware crea un nuevo middleware
func NewQuotaMiddleware(enabled bool) *QuotaMiddleware {
	return &QuotaMiddleware{
		usage:   make(map[string]*Usage),
		limits:  make(map[string]*Quota),
		enabled: enabled,
	}
}

// SetQuota configura la quota para un tenant
func (m *QuotaMiddleware) SetQuota(tenantID string, quota *Quota) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limits[tenantID] = quota
}

func (m *QuotaMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		rc, ok := gwcontext.FromContext(r.Context())
		if !ok {
			WriteError(w, NewInternalError("missing request context", nil))
			return
		}

		// Obtener o crear quota
		m.mu.Lock()
		quota, exists := m.limits[rc.TenantID]
		if !exists {
			quota = DefaultQuota()
			m.limits[rc.TenantID] = quota
		}

		usage, exists := m.usage[rc.TenantID]
		if !exists {
			usage = &Usage{
				WindowStart: time.Now(),
				MonthStart:  time.Now(),
			}
			m.usage[rc.TenantID] = usage
		}

		now := time.Now()

		// Resetear ventana de minuto
		if now.Sub(usage.WindowStart) > time.Minute {
			usage.Requests = 0
			usage.Tokens = 0
			usage.WindowStart = now
		}

		// Resetear mes
		if now.Sub(usage.MonthStart) > 30*24*time.Hour {
			usage.MonthTokens = 0
			usage.MonthStart = now
		}

		// Verificar límites
		if usage.Requests >= quota.RequestsPerMinute {
			m.mu.Unlock()
			logger.Warnf("🚫 Quota exceeded: tenant=%s requests=%d/%d", rc.TenantID, usage.Requests, quota.RequestsPerMinute)
			WriteError(w, NewQuotaExceededError("requests per minute exceeded"))
			return
		}

		if usage.MonthTokens >= quota.MonthlyTokens {
			m.mu.Unlock()
			logger.Warnf("🚫 Monthly quota exceeded: tenant=%s tokens=%d/%d", rc.TenantID, usage.MonthTokens, quota.MonthlyTokens)
			WriteError(w, NewQuotaExceededError("monthly token quota exceeded"))
			return
		}

		// Registrar uso (estimado; el settlement real se hace en accounting)
		estimatedTokens := 100
		usage.Requests++
		usage.Tokens += estimatedTokens
		usage.MonthTokens += estimatedTokens

		m.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
