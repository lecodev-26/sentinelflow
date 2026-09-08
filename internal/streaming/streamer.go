package streaming

import (
"context"
"encoding/json"
"fmt"
"net/http"
"time"
)

// Event es un evento de streaming
type Event struct {
Type    string      `json:"type"`    // "chunk", "done", "error"
Content string      `json:"content"` // Contenido del chunk
Data    interface{} `json:"data,omitempty"`
}

// SSEWriter escribe eventos en formato Server-Sent Events
type SSEWriter struct {
w       http.ResponseWriter
flusher http.Flusher
}

// NewSSEWriter crea un nuevo escritor SSE
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")

flusher, ok := w.(http.Flusher)
if !ok {
return nil
}

return &SSEWriter{
w:       w,
flusher: flusher,
}
}

// Write envía un evento SSE
func (s *SSEWriter) Write(event Event) error {
data, err := json.Marshal(event)
if err != nil {
return err
}

_, err = s.w.Write([]byte("data: " + string(data) + "\n\n"))
if err != nil {
return err
}

s.flusher.Flush()
return nil
}

// StreamProviderCallback es una función que recibe eventos de streaming
type StreamProviderCallback func(ctx context.Context) (<-chan Event, error)

// StreamHandler maneja streaming desde un proveedor
func StreamHandler(ctx context.Context, w http.ResponseWriter, callback StreamProviderCallback, timeout time.Duration) error {
sse := NewSSEWriter(w)
if sse == nil {
return fmt.Errorf("streaming not supported")
}

ctx, cancel := context.WithTimeout(ctx, timeout)
defer cancel()

events, err := callback(ctx)
if err != nil {
return err
}

for {
select {
case <-ctx.Done():
return ctx.Err()
case event, ok := <-events:
if !ok {
return nil
}
if err := sse.Write(event); err != nil {
return err
}
}
}
}

// MockStreamProvider crea un mock de streaming para pruebas
func MockStreamProvider(ctx context.Context) (<-chan Event, error) {
events := make(chan Event)

go func() {
defer close(events)

chunks := []string{
"Hello",
" world",
"! This is",
" a streaming",
" response.",
}

for _, chunk := range chunks {
select {
case <-ctx.Done():
return
default:
events <- Event{
Type:    "chunk",
Content: chunk,
}
time.Sleep(100 * time.Millisecond)
}
}

events <- Event{
Type:    "done",
Content: "",
}
}()

return events, nil
}
