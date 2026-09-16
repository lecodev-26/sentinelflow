package gateway

import (
"bytes"
"encoding/json"
"net/http"

gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
"github.com/lecodev-26/sentinelflow/internal/cost"
"github.com/lecodev-26/sentinelflow/internal/logger"
)

// CostMiddleware captura la respuesta del provider y registra el coste real
type CostMiddleware struct {
tracker *cost.CostTracker
enabled bool
}

// NewCostMiddleware crea un nuevo middleware de coste
func NewCostMiddleware(tracker *cost.CostTracker, enabled bool) *CostMiddleware {
return &CostMiddleware{
tracker: tracker,
enabled: enabled,
}
}

func (m *CostMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if !m.enabled || m.tracker == nil {
next.ServeHTTP(w, r)
return
}

rc, ok := gwcontext.FromContext(r.Context())
if !ok {
WriteError(w, NewInternalError("missing request context", nil))
return
}

// Capturar la respuesta
recorder := &costRecorder{
ResponseWriter: w,
statusCode:     http.StatusOK,
buffer:         &bytes.Buffer{},
}

next.ServeHTTP(recorder, r)

// Solo registrar si fue exitoso
if recorder.statusCode == http.StatusOK {
// Extraer usage de la respuesta
var respBody map[string]interface{}
if err := json.Unmarshal(recorder.buffer.Bytes(), &respBody); err == nil {
var inputTokens, outputTokens int
var provider, model string

if usage, ok := respBody["usage"].(map[string]interface{}); ok {
if pt, ok := usage["prompt_tokens"].(float64); ok {
inputTokens = int(pt)
}
if ct, ok := usage["completion_tokens"].(float64); ok {
outputTokens = int(ct)
}
}
if p, ok := respBody["provider"].(string); ok {
provider = p
}
if mdl, ok := respBody["model"].(string); ok {
model = mdl
}

// Calcular coste
costUSD := cost.CalculateCost(provider, model, inputTokens, outputTokens)

// Registrar
m.tracker.Record(
rc.TenantID,
rc.ProjectID,
provider,
model,
inputTokens,
outputTokens,
costUSD,
)

rc.Provider = provider
rc.Tokens = inputTokens + outputTokens
rc.Cost = costUSD

logger.Infof("💰 Coste registrado: tenant=%s provider=%s tokens=%d cost=$%.6f",
rc.TenantID, provider, inputTokens+outputTokens, costUSD)
}
}
})
}

type costRecorder struct {
http.ResponseWriter
statusCode int
buffer     *bytes.Buffer
}

func (r *costRecorder) WriteHeader(code int) {
r.statusCode = code
r.ResponseWriter.WriteHeader(code)
}

func (r *costRecorder) Write(b []byte) (int, error) {
r.buffer.Write(b)
return r.ResponseWriter.Write(b)
}
