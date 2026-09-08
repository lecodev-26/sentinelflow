package policy

import (
"bytes"
"context"
"encoding/json"
"io"
"net/http"
)

type Middleware struct {
engine *Engine
}

func NewMiddleware(engine *Engine) *Middleware {
return &Middleware{engine: engine}
}

func (m *Middleware) Handler(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
tenantID := r.Header.Get("X-Tenant-ID")
if tenantID == "" {
tenantID = "default"
}

userID := r.Header.Get("X-User-ID")
if userID == "" {
userID = "anonymous"
}

provider := r.Header.Get("X-Provider")

body, err := io.ReadAll(r.Body)
if err != nil {
http.Error(w, "Error reading body", http.StatusBadRequest)
return
}
r.Body = io.NopCloser(bytes.NewReader(body))

requestID := r.Header.Get("X-Request-ID")
if requestID == "" {
requestID = generateID()
}

evalCtx := &EvaluationContext{
RequestID: requestID,
TenantID:  tenantID,
UserID:    userID,
Provider:  provider,
Method:    r.Method,
Path:      r.URL.Path,
Metadata:  make(map[string]interface{}),
}

var reqBody map[string]interface{}
if err := json.Unmarshal(body, &reqBody); err == nil {
if model, ok := reqBody["model"].(string); ok {
evalCtx.Model = model
}
evalCtx.Input = reqBody
}

result := m.engine.Evaluate(r.Context(), evalCtx)

if !result.Allowed {
http.Error(w, result.Reason, http.StatusForbidden)
return
}

for _, action := range result.Actions {
if action.Type == "redact" {
_ = action
}
}

ctx := context.WithValue(r.Context(), "policy_result", result)
r = r.WithContext(ctx)

next(w, r)
}
}
