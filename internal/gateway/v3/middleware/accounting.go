package middleware

import (
"bytes"
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

recorder := &accountingRecorder{
ResponseWriter: w,
statusCode:     http.StatusOK,
buffer:         &bytes.Buffer{},
}

start := time.Now()
next.ServeHTTP(recorder, r)
latency := time.Since(start)

// No registrar si fue bloqueado por seguridad
if recorder.statusCode == http.StatusBadRequest ||
recorder.statusCode == http.StatusForbidden {
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

// Extraer tokens del body de respuesta (no-streaming)
var inputTokens, outputTokens int
var model string

if recorder.statusCode == http.StatusOK {
var respBody map[string]interface{}
if err := json.Unmarshal(recorder.buffer.Bytes(), &respBody); err == nil {
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

// Si no hay tokens (por ej. error o provider sin usage), no registrar
if inputTokens == 0 && outputTokens == 0 && recorder.statusCode != http.StatusOK {
return
}

// Determinar status
status := "success"
if recorder.statusCode >= 400 {
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

// accountingRecorder captura la respuesta
type accountingRecorder struct {
http.ResponseWriter
statusCode int
buffer     *bytes.Buffer
}

func (r *accountingRecorder) WriteHeader(code int) {
r.statusCode = code
r.ResponseWriter.WriteHeader(code)
}

func (r *accountingRecorder) Write(b []byte) (int, error) {
r.buffer.Write(b)
return r.ResponseWriter.Write(b)
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
