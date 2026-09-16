package context

import (
"context"
"sync"
"time"
)

// RequestContext contiene toda la información de una petición
type RequestContext struct {
mu sync.RWMutex

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
TTFT       time.Duration `json:"ttft,omitempty"`

// Tokens
InputTokens  int `json:"input_tokens"`
OutputTokens int `json:"output_tokens"`
TotalTokens  int `json:"total_tokens"`
CachedTokens int `json:"cached_tokens"`

// Coste
EstimatedCost float64 `json:"estimated_cost"`
ActualCost    float64 `json:"actual_cost"`

// Cache
CacheHit   bool   `json:"cache_hit"`
CacheLayer string `json:"cache_layer,omitempty"`

// Errores
Error string `json:"error,omitempty"`

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
rc.mu.RLock()
defer rc.mu.RUnlock()
if rc.ctx != nil {
return rc.ctx
}
return context.Background()
}

// SetMetadata añade un valor a la metadata
func (rc *RequestContext) SetMetadata(key string, value interface{}) {
rc.mu.Lock()
defer rc.mu.Unlock()
if rc.Metadata == nil {
rc.Metadata = make(map[string]interface{})
}
rc.Metadata[key] = value
}

// GetMetadata obtiene un valor de la metadata
func (rc *RequestContext) GetMetadata(key string) (interface{}, bool) {
rc.mu.RLock()
defer rc.mu.RUnlock()
if rc.Metadata == nil {
return nil, false
}
v, ok := rc.Metadata[key]
return v, ok
}

// SetProvider establece el provider seleccionado
func (rc *RequestContext) SetProvider(p string) {
rc.mu.Lock()
defer rc.mu.Unlock()
rc.Provider = p
}

// SetTokens establece el uso de tokens
func (rc *RequestContext) SetTokens(input, output, cached int) {
rc.mu.Lock()
defer rc.mu.Unlock()
rc.InputTokens = input
rc.OutputTokens = output
rc.CachedTokens = cached
rc.TotalTokens = input + output
}

// SetCost establece el coste real
func (rc *RequestContext) SetCost(cost float64) {
rc.mu.Lock()
defer rc.mu.Unlock()
rc.ActualCost = cost
}

// SetCacheHit marca la petición como cache HIT
func (rc *RequestContext) SetCacheHit(layer string) {
rc.mu.Lock()
defer rc.mu.Unlock()
rc.CacheHit = true
rc.CacheLayer = layer
}

// SetError establece el error de la petición
func (rc *RequestContext) SetError(err string) {
rc.mu.Lock()
defer rc.mu.Unlock()
rc.Error = err
}

// Finish marca la petición como completada
func (rc *RequestContext) Finish() {
rc.mu.Lock()
defer rc.mu.Unlock()
rc.Latency = time.Since(rc.StartTime)
}

// Snapshot devuelve una copia inmutable del contexto
func (rc *RequestContext) Snapshot() RequestContext {
rc.mu.RLock()
defer rc.mu.RUnlock()
snapshot := *rc
snapshot.mu = sync.RWMutex{}
snapshot.Metadata = make(map[string]interface{})
for k, v := range rc.Metadata {
snapshot.Metadata[k] = v
}
return snapshot
}

// ToMap devuelve el contexto como mapa para logs/métricas
func (rc *RequestContext) ToMap() map[string]interface{} {
rc.mu.RLock()
defer rc.mu.RUnlock()
return map[string]interface{}{
"request_id":    rc.RequestID,
"trace_id":      rc.TraceID,
"tenant_id":     rc.TenantID,
"project_id":    rc.ProjectID,
"user_id":       rc.UserID,
"api_key_id":    rc.APIKeyID,
"role":          rc.Role,
"method":        rc.Method,
"path":          rc.Path,
"model":         rc.Model,
"provider":      rc.Provider,
"stream":        rc.Stream,
"status_code":   rc.StatusCode,
"latency_ms":    rc.Latency.Milliseconds(),
"ttft_ms":       rc.TTFT.Milliseconds(),
"input_tokens":  rc.InputTokens,
"output_tokens": rc.OutputTokens,
"total_tokens":  rc.TotalTokens,
"cached_tokens": rc.CachedTokens,
"cost_usd":      rc.ActualCost,
"cache_hit":     rc.CacheHit,
"cache_layer":   rc.CacheLayer,
"error":         rc.Error,
"ip":            rc.IP,
}
}

func generateID() string {
return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, n)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
}
return string(b)
}
