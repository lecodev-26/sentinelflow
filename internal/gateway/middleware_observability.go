package gateway

import (
"net/http"
"time"

gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/observability"
"go.opentelemetry.io/otel/codes"
)

// ObservabilityMiddleware integra OpenTelemetry en el pipeline
type ObservabilityMiddleware struct {
enabled bool
}

func NewObservabilityMiddleware(enabled bool) *ObservabilityMiddleware {
return &ObservabilityMiddleware{enabled: enabled}
}

func (m *ObservabilityMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
rc, ok := gwcontext.FromContext(r.Context())
if !ok {
WriteError(w, NewInternalError("missing request context", nil))
return
}

// Iniciar span
var span *observability.SpanWrapper
if m.enabled {
span = observability.StartSpan(r.Context(), "gateway.request")
defer span.End()

span.SetAttribute("request_id", rc.RequestID)
span.SetAttribute("trace_id", rc.TraceID)
span.SetAttribute("tenant_id", rc.TenantID)
span.SetAttribute("user_id", rc.UserID)
span.SetAttribute("method", r.Method)
span.SetAttribute("path", r.URL.Path)
span.SetAttribute("model", rc.Model)

r = r.WithContext(span.Context())
}

// Capturar la respuesta
recorder := &obsRecorder{
ResponseWriter: w,
statusCode:     http.StatusOK,
}

start := time.Now()
next.ServeHTTP(recorder, r)
latency := time.Since(start)

// Actualizar RequestContext
rc.StatusCode = recorder.statusCode
rc.Latency = latency

// Añadir atributos al span
if span != nil {
span.SetAttribute("status_code", recorder.statusCode)
span.SetAttribute("latency_ms", latency.Milliseconds())
span.SetAttribute("provider", rc.Provider)
span.SetAttribute("tokens", rc.TotalTokens)
span.SetAttribute("cost_usd", rc.ActualCost)
span.SetAttribute("cache_hit", rc.CacheHit)

if recorder.statusCode >= 500 {
span.SetStatus(codes.Error, http.StatusText(recorder.statusCode))
} else {
span.SetStatus(codes.Ok, "")
}
}

// Log estructurado SIN PII
logger.WithFields(logger.Get().WithFields(nil).Data).Info("")
logger.WithFields(map[string]interface{}{
"request_id": rc.RequestID,
"trace_id":   rc.TraceID,
"tenant_id":  rc.TenantID,
"user_id":    rc.UserID,
"method":     r.Method,
"path":       r.URL.Path,
"status":     recorder.statusCode,
"latency_ms": latency.Milliseconds(),
"provider":   rc.Provider,
"model":      rc.Model,
"tokens":     rc.TotalTokens,
"cost_usd":   rc.ActualCost,
"cache_hit":  rc.CacheHit,
"ip":         rc.IP,
}).Info("gateway request completed")
})
}

type obsRecorder struct {
http.ResponseWriter
statusCode int
}

func (r *obsRecorder) WriteHeader(code int) {
r.statusCode = code
r.ResponseWriter.WriteHeader(code)
}
