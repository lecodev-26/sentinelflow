package audit

import (
"bytes"
"io"
"net/http"
"time"
)

// Middleware es un middleware de auditoría
type Middleware struct {
logger *Logger
}

// NewMiddleware crea un nuevo middleware de auditoría
func NewMiddleware(logger *Logger) *Middleware {
return &Middleware{logger: logger}
}

// Handler envuelve un handler con auditoría
func (m *Middleware) Handler(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
start := time.Now()

// Extraer información de la petición
tenantID := getTenantID(r)
requestID := getRequestID(r)

// Leer body
body, err := io.ReadAll(r.Body)
if err != nil {
http.Error(w, "Error reading body", http.StatusBadRequest)
return
}
r.Body = io.NopCloser(bytes.NewReader(body))

// Crear ResponseWriter wrapper
rw := &responseWriter{ResponseWriter: w}

// Ejecutar handler
next.ServeHTTP(rw, r)

// Registrar auditoría
latency := time.Since(start)

entry := Entry{
TenantID:  tenantID,
RequestID: requestID,
Method:    r.Method,
Path:      r.URL.Path,
Status:    rw.statusCode,
Latency:   latency,
IP:        r.RemoteAddr,
UserAgent: r.UserAgent(),
Level:     LevelInfo,
}

if rw.statusCode >= 400 {
entry.Level = LevelError
}

_ = m.logger.Log(entry)
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

func getTenantID(r *http.Request) string {
if tenant := r.Header.Get("X-Tenant-ID"); tenant != "" {
return tenant
}
if tenant := r.URL.Query().Get("tenant"); tenant != "" {
return tenant
}
return "default"
}

func getRequestID(r *http.Request) string {
if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
return reqID
}
return generateID()
}
