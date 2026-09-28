package gateway

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	http.Hijacker
	io.ReaderFrom
	Status() int
	Written() bool
}

type writer struct {
	http.ResponseWriter
	status  int
	written bool
}

func NewResponseWriter(w http.ResponseWriter) ResponseWriter {
	return &writer{ResponseWriter: w, status: http.StatusOK}
}
func (w *writer) WriteHeader(code int) {
	if w.written {
		return
	}
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}
func (w *writer) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}
func (w *writer) Status() int   { return w.status }
func (w *writer) Written() bool { return w.written }
func (w *writer) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
func (w *writer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}
func (w *writer) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	return io.Copy(w.ResponseWriter, r)
}
