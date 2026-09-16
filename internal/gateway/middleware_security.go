package gateway

import (
"bytes"
"encoding/json"
"io"
"net/http"

"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/security/pii"
"github.com/lecodev-26/sentinelflow/internal/security/prompt"
)

type SecurityMiddleware struct {
piiDetector    *pii.Detector
piiPolicy      pii.Policy
secretDetector *pii.SecretDetector
promptDetector *prompt.Detector
promptPolicy   prompt.Policy
enabled        bool
}

func NewSecurityMiddleware(enabled bool) *SecurityMiddleware {
return &SecurityMiddleware{
piiDetector:    pii.NewDetector(),
piiPolicy:      pii.DefaultPolicy(),
secretDetector: pii.NewSecretDetector(),
promptDetector: prompt.NewDetector(),
promptPolicy:   prompt.DefaultPolicy(),
enabled:        enabled,
}
}

func (m *SecurityMiddleware) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
if !m.enabled {
next.ServeHTTP(w, r)
return
}

// Leer body
body, err := io.ReadAll(r.Body)
if err != nil {
WriteError(w, NewInvalidRequestError("error reading body"))
return
}
defer r.Body.Close()

// Parsear JSON
var reqBody map[string]interface{}
if err := json.Unmarshal(body, &reqBody); err != nil {
// No es JSON, no escanear
r.Body = io.NopCloser(bytes.NewReader(body))
next.ServeHTTP(w, r)
return
}

// Extraer mensajes
var messages []map[string]interface{}
if msgs, ok := reqBody["messages"].([]interface{}); ok {
for _, m := range msgs {
if msg, ok := m.(map[string]interface{}); ok {
messages = append(messages, msg)
}
}
}

// Escanear cada mensaje del usuario
for i, msg := range messages {
role, _ := msg["role"].(string)
if role != "user" {
continue
}

content, _ := msg["content"].(string)
if content == "" {
continue
}

// 1. Prompt injection (Process devuelve 3 valores)
content, matches, action := m.promptPolicy.Process(content, m.promptDetector)
if len(matches) > 0 {
logger.Warnf("⚠️ Prompt injection detectado: %d coincidencias, acción: %s", len(matches), action)
if action == prompt.ActionBlock {
WriteError(w, NewPolicyDeniedError("prompt injection detected"))
return
}
}

// 2. PII (Process devuelve 3 valores)
content, piiMatches, piiAction := m.piiPolicy.Process(content, m.piiDetector)
if len(piiMatches) > 0 {
logger.Warnf("⚠️ PII detectada: %d coincidencias, acción: %s", len(piiMatches), piiAction)
if piiAction == pii.ActionBlock {
WriteError(w, NewPolicyDeniedError("PII detected and blocked"))
return
}
}

// 3. Secretos
if m.secretDetector.HasSecrets(content) {
logger.Warnf("🚨 Secretos detectados, bloqueando petición")
WriteError(w, NewPolicyDeniedError("secrets detected in request"))
return
}

// Actualizar el mensaje con el contenido potencialmente redactado
messages[i]["content"] = content
}

// Reconstruir body
reqBody["messages"] = messages
newBody, err := json.Marshal(reqBody)
if err != nil {
WriteError(w, NewInternalError("error rebuilding body", err))
return
}

r.Body = io.NopCloser(bytes.NewReader(newBody))
r.ContentLength = int64(len(newBody))

next.ServeHTTP(w, r)
})
}
