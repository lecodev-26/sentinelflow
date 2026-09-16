package context

import (
"context"
"time"
)

// RequestContext contiene toda la información de una petición
type RequestContext struct {
// Identificadores
RequestID string `json:"request_id"`
TraceID   string `json:"trace_id"`

// Identidad
TenantID  string `json:"tenant_id"`
ProjectID string `json:"project_id"`
UserID    string `json:"user_id"`
APIKeyID  string `json:"api_key_id"`
Role      string `json:"role"`

// Petición
Method string `json:"method"`
Path   string `json:"path"`
Model  string `json:"model"`
Stream bool   `json:"stream"`

// Metadatos
IP        string    `json:"ip"`
UserAgent string    `json:"user_agent"`
StartTime time.Time `json:"start_time"`

// Resultado
Provider   string        `json:"provider,omitempty"`
StatusCode int           `json:"status_code,omitempty"`
Latency    time.Duration `json:"latency,omitempty"`
Tokens     int           `json:"tokens,omitempty"`
Cost       float64       `json:"cost,omitempty"`
CacheHit   bool          `json:"cache_hit"`
Error      string        `json:"error,omitempty"`

// Metadata adicional
Metadata map[string]interface{} `json:"metadata,omitempty"`

// Contexto de Go
ctx context.Context
}

type contextKey string

const (
requestContextKey contextKey = "request_context"
)

// New crea un nuevo RequestContext
func New(ctx context.Context) *RequestContext {
rc := &RequestContext{
RequestID: generateID(),
TraceID:   generateID(),
StartTime: time.Now(),
Metadata:  make(map[string]interface{}),
ctx:       ctx,
}
return rc
}

// WithContext añade el RequestContext al contexto de Go
func WithContext(ctx context.Context, rc *RequestContext) context.Context {
return context.WithValue(ctx, requestContextKey, rc)
}

// FromContext extrae el RequestContext del contexto de Go
func FromContext(ctx context.Context) (*RequestContext, bool) {
rc, ok := ctx.Value(requestContextKey).(*RequestContext)
return rc, ok
}

// GetContext devuelve el contexto de Go
func (rc *RequestContext) GetContext() context.Context {
if rc.ctx != nil {
return rc.ctx
}
return context.Background()
}

// SetMetadata añade un valor a la metadata
func (rc *RequestContext) SetMetadata(key string, value interface{}) {
if rc.Metadata == nil {
rc.Metadata = make(map[string]interface{})
}
rc.Metadata[key] = value
}

// GetMetadata obtiene un valor de la metadata
func (rc *RequestContext) GetMetadata(key string) (interface{}, bool) {
if rc.Metadata == nil {
return nil, false
}
v, ok := rc.Metadata[key]
return v, ok
}

// Finish marca la petición como completada
func (rc *RequestContext) Finish() {
rc.Latency = time.Since(rc.StartTime)
}

func generateID() string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, 16)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
time.Sleep(1)
}
return string(b)
}
