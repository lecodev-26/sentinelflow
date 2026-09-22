package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sync"
	"time"
)

// DedupMiddleware deduplica peticiones idénticas concurrentes
type DedupMiddleware struct {
	mu       sync.Mutex
	inflight map[string]*dedupCall
	enabled  bool
}

type dedupCall struct {
	done chan struct{}
	body []byte
	code int
}

func NewDedupMiddleware(enabled bool) *DedupMiddleware {
	return &DedupMiddleware{
		inflight: make(map[string]*dedupCall),
		enabled:  enabled,
	}
}

func (m *DedupMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled || r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		// Leer body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			WriteError(w, NewInvalidRequestError("error reading body"))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Hash del body
		hash := sha256.Sum256(body)
		key := hex.EncodeToString(hash[:])

		m.mu.Lock()
		existing, exists := m.inflight[key]
		if exists {
			// Esperar al que ya está en vuelo
			m.mu.Unlock()
			select {
			case <-existing.done:
				w.Header().Set("X-Dedup", "HIT")
				w.WriteHeader(existing.code)
				w.Write(existing.body)
				return
			case <-time.After(30 * time.Second):
				WriteError(w, NewInternalError("dedup timeout", nil))
				return
			}
		}

		// Somos los primeros
		call := &dedupCall{done: make(chan struct{})}
		m.inflight[key] = call
		m.mu.Unlock()

		// Capturar la respuesta
		recorder := &dedupRecorder{ResponseWriter: w, statusCode: 200, buffer: &bytes.Buffer{}}

		next.ServeHTTP(recorder, r)

		// Guardar y notificar
		call.body = recorder.buffer.Bytes()
		call.code = recorder.statusCode
		close(call.done)

		m.mu.Lock()
		delete(m.inflight, key)
		m.mu.Unlock()
	})
}

type dedupRecorder struct {
	http.ResponseWriter
	statusCode int
	buffer     *bytes.Buffer
}

func (r *dedupRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *dedupRecorder) Write(b []byte) (int, error) {
	r.buffer.Write(b)
	return r.ResponseWriter.Write(b)
}

var _ = context.Background
