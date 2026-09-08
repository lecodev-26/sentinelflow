package observability

import (
"net/http"
"time"

"go.opentelemetry.io/otel/codes"
)

// ObservabilityMiddleware es un middleware que añade tracing y métricas
type ObservabilityMiddleware struct{}

// NewObservabilityMiddleware crea un nuevo middleware
func NewObservabilityMiddleware() *ObservabilityMiddleware {
return &ObservabilityMiddleware{}
}

// Handler envuelve un handler con observabilidad
func (m *ObservabilityMiddleware) Handler(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
// Iniciar span
span := StartSpan(r.Context(), r.URL.Path)
defer span.End()

// Añadir atributos
span.SetAttribute("http.method", r.Method)
span.SetAttribute("http.path", r.URL.Path)
span.SetAttribute("http.user_agent", r.UserAgent())

// Registrar petición activa
RecordActiveRequest(true)
defer RecordActiveRequest(false)

// Crear ResponseWriter wrapper para capturar status
rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

startTime := time.Now()

// Ejecutar handler
next.ServeHTTP(rw, r.WithContext(span.Context()))

// Registrar métricas
duration := time.Since(startTime)
span.SetAttribute("http.status_code", rw.statusCode)
span.SetAttribute("http.duration_ms", duration.Milliseconds())

if rw.statusCode >= 500 {
span.SetStatus(codes.Error, http.StatusText(rw.statusCode))
RecordError("server_error", "unknown")
} else {
span.SetStatus(codes.Ok, "")
}

// Set provider status en el span si existe
if provider := r.Header.Get("X-Provider"); provider != "" {
span.SetAttribute("provider", provider)
}
}
}

// responseWriter wrapper para capturar el código de estado
type responseWriter struct {
http.ResponseWriter
statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
rw.statusCode = code
rw.ResponseWriter.WriteHeader(code)
}
