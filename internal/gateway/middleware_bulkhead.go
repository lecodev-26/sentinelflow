package gateway

import (
	"net/http"
	"time"
)

// BulkheadMiddleware limita la concurrencia
type BulkheadMiddleware struct {
	sem     chan struct{}
	timeout time.Duration
}

func NewBulkheadMiddleware(maxConcurrent int, timeout time.Duration) *BulkheadMiddleware {
	if maxConcurrent <= 0 {
		maxConcurrent = 100
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &BulkheadMiddleware{
		sem:     make(chan struct{}, maxConcurrent),
		timeout: timeout,
	}
}

func (m *BulkheadMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case m.sem <- struct{}{}:
			defer func() { <-m.sem }()
			next.ServeHTTP(w, r)
		case <-time.After(m.timeout):
			WriteError(w, NewRateLimitedError("server at capacity, try again later"))
		}
	})
}
