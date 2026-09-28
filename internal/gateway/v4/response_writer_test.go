package gateway

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseWriterPreservesStreamingInterfaces(t *testing.T) {
	w := NewResponseWriter(httptest.NewRecorder())
	if w.Status() != 200 || w.Written() {
		t.Fatal("bad initial state")
	}
	w.WriteHeader(201)
	w.Write([]byte("ok"))
	if w.Status() != 201 || !w.Written() {
		t.Fatal("state not captured")
	}
	if _, ok := w.(interface{ Flush() }); !ok {
		t.Fatal("missing flusher")
	}
	_, _ = io.Copy(w, strings.NewReader("x"))
}
