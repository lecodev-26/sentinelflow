package middleware

import (
"encoding/json"
"net/http"
"time"

accountingv3 "github.com/lecodev-26/sentinelflow/internal/accounting/v3"
"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

// AccountingMiddleware registra el uso después del handler
type AccountingMiddleware struct {
service *accountingv3.Service
enabled bool
}

// NewAccountingMiddleware crea un nuevo middleware
func NewAccountingMiddleware(service *accountingv3.Service, enabled bool) *AccountingMiddleware {
return &AccountingMiddleware{
service: service,
enabled: enabled,
}
}

// Handler envuelve un handler con accounting
func (m *AccountingMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if !m.enabled || m.service == nil || r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
next.ServeHTTP(w, r)
return
}

// NO capturamos body si es streaming, porque el body son chunks SSE
// y no queremos bufferear todo el stream.
isStream := isStreamingRequest(r)

// Usar el wrapper unificado (preserva Flusher, Hijacker, Pusher, ReaderFrom)
recorder := NewResponseWriterWrapper(w, !isStream)

start := time.Now()
next.ServeHTTP(recorder, r)
latency := time.Since(start)

// No registrar si fue bloqueado por seguridad
if recorder.StatusCode() == http.StatusBadRequest ||
recorder.StatusCode() == http.StatusForbidden {
return
}

// Extraer contexto
tenantID := getTenantFromCtx(r.Context())
userID := getUserFromCtx(r.Context())
apiKeyID := getAPIKeyFromCtx(r.Context())

providerID := recorder.Header().Get("X-Provider")
requestID := r.Header.Get("X-Request-Id")
if requestID == "" {
requestID = r.Header.Get("Idempotency-Key")
}

var inputTokens, outputTokens int
var model string

// Solo parsear body en no-streaming
if !isStream && recorder.StatusCode() == http.StatusOK {
var respBody map[string]interface{}
if err := json.Unmarshal(recorder.Buffer(), &respBody); err == nil {
if usage, ok := respBody["usage"].(map[string]interface{}); ok {
if v, ok := usage["prompt_tokens"].(float64); ok {
inputTokens = int(v)
}
if v, ok := usage["completion_tokens"].(float64); ok {
outputTokens = int(v)
}
}
if m, ok := respBody["model"].(string); ok {
model = m
}
}
}

// Determinar status
status := "success"
if recorder.StatusCode() >= 400 {
status = "error"
}

rec := &postgres.UsageRecord{
ID:           "usage_" + generateShortID(),
RequestID:    requestID,
TenantID:     tenantID,
UserID:       userID,
APIKeyID:     apiKeyID,
Provider:     providerID,
Model:        model,
InputTokens:  inputTokens,
OutputTokens: outputTokens,
LatencyMs:    int(latency.Milliseconds()),
Status:       status,
Timestamp:    time.Now(),
}

// Registrar en background para no bloquear la respuesta
go func() {
_ = m.service.Record(r.Context(), rec)
}()
})
}

// isStreamingRequest verifica si la petición es de streaming
func isStreamingRequest(r *http.Request) bool {
// Si el header Accept indica SSE
if r.Header.Get("Accept") == "text/event-stream" {
return true
}
// Nota: en el body se indica "stream":true, pero necesitaríamos
// leerlo (y romper el body). Por eso usamos el header Content-Type
// o un header custom X-Stream: true que el cliente puede poner.
// Alternativa: el handler de chat marca en el contexto.
return false
}

func getUserFromCtx(ctx interface{ Value(interface{}) interface{} }) string {
if v, ok := ctx.Value(CtxUserID).(string); ok {
return v
}
return ""
}

func getAPIKeyFromCtx(ctx interface{ Value(interface{}) interface{} }) string {
if v, ok := ctx.Value(CtxAPIKeyID).(string); ok {
return v
}
return ""
}

func generateShortID() string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, 12)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
}
return string(b)
}
