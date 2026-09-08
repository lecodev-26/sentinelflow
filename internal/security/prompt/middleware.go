package prompt

import (
"bytes"
"encoding/json"
"io"
"net/http"
"strings"
)

// Middleware es un middleware para detección de inyección de prompts
type Middleware struct {
detector *Detector
policy   Policy
enabled  bool
}

// NewMiddleware crea un nuevo middleware
func NewMiddleware(detector *Detector, policy Policy) *Middleware {
return &Middleware{
detector: detector,
policy:   policy,
enabled:  true,
}
}

// Handler envuelve un handler con detección de inyección
func (m *Middleware) Handler(next http.HandlerFunc) http.HandlerFunc {
return func(w http.ResponseWriter, r *http.Request) {
// Solo para POST a /v1/chat/completions
if r.Method != "POST" || !strings.Contains(r.URL.Path, "/chat/completions") {
next(w, r)
return
}

// Leer body
body, err := io.ReadAll(r.Body)
if err != nil {
http.Error(w, "Error reading body", http.StatusBadRequest)
return
}
r.Body = io.NopCloser(bytes.NewReader(body))

// Parsear request para extraer mensajes
var req struct {
Messages []struct {
Role    string `json:"role"`
Content string `json:"content"`
} `json:"messages"`
}

if err := json.Unmarshal(body, &req); err != nil {
next(w, r)
return
}

// Verificar cada mensaje del usuario
for i, msg := range req.Messages {
if msg.Role == "user" {
processed, matches, action := m.policy.Process(msg.Content, m.detector)
if len(matches) > 0 {
// Registrar detección (en producción, aquí se loguearía)
_ = matches

if action == ActionBlock {
http.Error(w, "Prompt injection detected and blocked", http.StatusBadRequest)
return
}
// Si es warn o allow, actualizar el mensaje procesado (posiblemente sanitizado)
req.Messages[i].Content = processed
}
}
}

// Reconstruir body con mensajes procesados
newBody, err := json.Marshal(req)
if err != nil {
next(w, r)
return
}
r.Body = io.NopCloser(bytes.NewReader(newBody))

next(w, r)
}
}
