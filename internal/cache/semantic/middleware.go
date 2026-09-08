package semantic

import (
"bytes"
"encoding/json"
"io"
"net/http"
"strings"
)

// Middleware es un middleware para caché semántica
type Middleware struct {
cache   *Cache
enabled bool
}

// NewMiddleware crea un nuevo middleware
func NewMiddleware(cache *Cache) *Middleware {
return &Middleware{
cache:   cache,
enabled: true,
}
}

// Handler envuelve un handler con caché semántica
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

// Parsear request para obtener el prompt y modelo
var req struct {
Model    string `json:"model"`
Messages []struct {
Role    string `json:"role"`
Content string `json:"content"`
} `json:"messages"`
Stream bool `json:"stream"`
}

if err := json.Unmarshal(body, &req); err != nil {
next(w, r)
return
}

// Si es streaming, no cachear
if req.Stream {
next(w, r)
return
}

// Extraer el último mensaje del usuario
var prompt string
for i := len(req.Messages) - 1; i >= 0; i-- {
if req.Messages[i].Role == "user" {
prompt = req.Messages[i].Content
break
}
}

if prompt == "" {
next(w, r)
return
}

// Buscar en caché
if cached, found := m.cache.Get(prompt, req.Model); found {
// Devolver respuesta cachead
w.Header().Set("Content-Type", "application/json")
w.Header().Set("X-Cache", "HIT (semantic)")
w.WriteHeader(http.StatusOK)
w.Write(cached)
return
}

// Si no está en caché, crear un ResponseWriter wrapper
rw := &responseWriter{ResponseWriter: w}

// Ejecutar el handler
next.ServeHTTP(rw, r)

// Si la respuesta fue exitosa, guardar en caché
if rw.statusCode == http.StatusOK {
// Guardar la respuesta en caché
go func() {
// Extraer el contenido de la respuesta para guardar
_ = m.cache.Set(prompt, rw.body.Bytes(), req.Model, "unknown")
}()
}
}
}

// responseWriter wrapper para capturar la respuesta
type responseWriter struct {
http.ResponseWriter
body       bytes.Buffer
statusCode int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
rw.body.Write(b)
return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
rw.statusCode = statusCode
rw.ResponseWriter.WriteHeader(statusCode)
}
