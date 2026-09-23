package middleware

import (
"bufio"
"bytes"
"errors"
"net"
"net/http"
)

// responseWriterWrapper envuelve un http.ResponseWriter preservando
// todas las interfaces opcionales: Flusher, Hijacker, Pusher, ReaderFrom.
//
// Sin esto, el streaming SSE se rompe porque accountingRecorder y
// idempotencyRecorder no implementaban http.Flusher.
type responseWriterWrapper struct {
http.ResponseWriter
statusCode int
written    bool
buffer     *bytes.Buffer // opcional: solo si captureBody = true
captureBody bool
}

// NewResponseWriterWrapper crea un nuevo wrapper
func NewResponseWriterWrapper(w http.ResponseWriter, captureBody bool) *responseWriterWrapper {
var buf *bytes.Buffer
if captureBody {
buf = &bytes.Buffer{}
}
return &responseWriterWrapper{
ResponseWriter: w,
statusCode:     http.StatusOK,
buffer:         buf,
captureBody:    captureBody,
}
}

// StatusCode devuelve el código HTTP capturado
func (w *responseWriterWrapper) StatusCode() int {
return w.statusCode
}

// Written devuelve si ya se escribió algo
func (w *responseWriterWrapper) Written() bool {
return w.written
}

// Buffer devuelve el body capturado (solo si captureBody=true)
func (w *responseWriterWrapper) Buffer() []byte {
if w.buffer == nil {
return nil
}
return w.buffer.Bytes()
}

// Header delega al ResponseWriter original
func (w *responseWriterWrapper) Header() http.Header {
return w.ResponseWriter.Header()
}

// WriteHeader captura el status y delega
func (w *responseWriterWrapper) WriteHeader(code int) {
if w.written {
return
}
w.statusCode = code
w.written = true
w.ResponseWriter.WriteHeader(code)
}

// Write captura el body (si aplica) y delega
func (w *responseWriterWrapper) Write(b []byte) (int, error) {
if !w.written {
w.written = true
}
if w.captureBody && w.buffer != nil {
w.buffer.Write(b)
}
return w.ResponseWriter.Write(b)
}

// === Flusher (crítico para SSE) ===

// Flush implementa http.Flusher si el writer original lo soporta
func (w *responseWriterWrapper) Flush() {
if f, ok := w.ResponseWriter.(http.Flusher); ok {
f.Flush()
}
}

// === Hijacker (para WebSocket / conexiones raw) ===

// Hijack implementa http.Hijacker si el writer original lo soporta
func (w *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
if h, ok := w.ResponseWriter.(http.Hijacker); ok {
return h.Hijack()
}
return nil, nil, errors.New("hijack not supported")
}

// === Pusher (HTTP/2 server push) ===

// Push implementa http.Pusher si el writer original lo soporta
func (w *responseWriterWrapper) Push(target string, opts *http.PushOptions) error {
if p, ok := w.ResponseWriter.(http.Pusher); ok {
return p.Push(target, opts)
}
return http.ErrNotSupported
}

// === ReaderFrom (optimización de io.Copy) ===

// ReadFrom implementa io.ReaderFrom si el writer original lo soporta.
// Esto permite que io.Copy use sendfile/splice en Linux.
func (w *responseWriterWrapper) ReadFrom(src interface{ Read([]byte) (int, error) }) (int64, error) {
if rf, ok := w.ResponseWriter.(interface{ ReadFrom(r interface{ Read([]byte) (int, error) }) (int64, error) }); ok {
return rf.ReadFrom(src)
}
// Fallback: copia manual
return copyBuffer(w, src)
}

// Unwrap devuelve el writer original (útil para debugging)
func (w *responseWriterWrapper) Unwrap() http.ResponseWriter {
return w.ResponseWriter
}

// copyBuffer es un fallback simple para ReadFrom
func copyBuffer(dst interface{ Write([]byte) (int, error) }, src interface{ Read([]byte) (int, error) }) (int64, error) {
buf := make([]byte, 32*1024)
var total int64
for {
nr, er := src.Read(buf)
if nr > 0 {
nw, ew := dst.Write(buf[:nr])
total += int64(nw)
if ew != nil {
return total, ew
}
if nw < nr {
return total, errors.New("short write")
}
}
if er != nil {
if er.Error() == "EOF" {
return total, nil
}
return total, er
}
}
}
