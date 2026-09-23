package middleware

import (
"net/http"
"sync"
"time"
)

// Idempotency guarda respuestas por Idempotency-Key
type Idempotency struct {
mu      sync.RWMutex
entries map[string]*idempotencyEntry
maxAge  time.Duration
}

type idempotencyEntry struct {
body       []byte
statusCode int
createdAt  time.Time
}

// NewIdempotency crea un nuevo middleware
func NewIdempotency(maxAge time.Duration) *Idempotency {
if maxAge == 0 {
maxAge = 24 * time.Hour
}
m := &Idempotency{
entries: make(map[string]*idempotencyEntry),
maxAge:  maxAge,
}
go m.cleanup()
return m
}

// Handler envuelve un handler con idempotencia
func (m *Idempotency) Handler(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// Solo para POST/PUT/DELETE
if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
next.ServeHTTP(w, r)
return
}

key := r.Header.Get("Idempotency-Key")
if key == "" {
next.ServeHTTP(w, r)
return
}

// Buscar respuesta previa
m.mu.RLock()
entry, exists := m.entries[key]
m.mu.RUnlock()

if exists {
w.Header().Set("X-Idempotency-Replayed", "true")
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(entry.statusCode)
w.Write(entry.body)
return
}

// Usar wrapper unificado (preserva Flusher)
recorder := NewResponseWriterWrapper(w, true)

next.ServeHTTP(recorder, r)

// Guardar si es exitoso o error de cliente (4xx)
if recorder.StatusCode() < 500 {
m.mu.Lock()
m.entries[key] = &idempotencyEntry{
body:       recorder.Buffer(),
statusCode: recorder.StatusCode(),
createdAt:  time.Now(),
}
m.mu.Unlock()
}
})
}

func (m *Idempotency) cleanup() {
ticker := time.NewTicker(1 * time.Hour)
defer ticker.Stop()

for range ticker.C {
m.mu.Lock()
cutoff := time.Now().Add(-m.maxAge)
for k, v := range m.entries {
if v.createdAt.Before(cutoff) {
delete(m.entries, k)
}
}
m.mu.Unlock()
}
}
