package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/idgen"
	observabilityv3 "github.com/lecodev-26/sentinelflow/internal/observability/v3"
)

// TracingMiddleware crea un trace por request
type TracingMiddleware struct {
	store   *observabilityv3.Store
	enabled bool
}

// NewTracingMiddleware crea un nuevo middleware
func NewTracingMiddleware(store *observabilityv3.Store, enabled bool) *TracingMiddleware {
	return &TracingMiddleware{
		store:   store,
		enabled: enabled,
	}
}

// Handler envuelve un handler con tracing
func (m *TracingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled || m.store == nil {
			next.ServeHTTP(w, r)
			return
		}

		traceID := idgen.RandomHex(8)
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = r.Header.Get("Idempotency-Key")
		}
		if requestID == "" {
			requestID = traceID
		}

		trace := m.store.Start(traceID, requestID)

		ctx := context.WithValue(r.Context(), CtxTraceID, traceID)
		ctx = context.WithValue(ctx, CtxTrace, trace)

		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))
		latency := time.Since(start)

		if tenantID := getTenantFromCtx(ctx); tenantID != "" {
			trace.TenantID = tenantID
		}
		if userID := getUserFromCtx(ctx); userID != "" {
			trace.UserID = userID
		}
		if apiKeyID := getAPIKeyFromCtx(ctx); apiKeyID != "" {
			trace.APIKeyID = apiKeyID
		}

		status := "ok"
		if trace.StatusCode >= 400 {
			status = "error"
		}
		if trace.StatusCode == 403 || trace.StatusCode == 400 {
			if trace.Error != "" {
				status = "blocked"
			}
		}

		m.store.AddSpan(traceID, observabilityv3.StartSpan("middleware.total").
			WithAttr("latency_ms", latency.Milliseconds()).
			End())

		m.store.End(traceID, 200, status, "")

		w.Header().Set("X-Trace-Id", traceID)
	})
}

// Context keys
type traceContextKey string

const (
	CtxTraceID traceContextKey = "trace_id"
	CtxTrace   traceContextKey = "trace"
)
