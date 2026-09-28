package middleware

import (
	securityv3 "github.com/lecodev-26/sentinelflow/internal/security/v3"
	"net/http"
	"time"
)

type AnomalyMiddleware struct{ detector *securityv3.AnomalyDetector }

func NewAnomalyMiddleware(d *securityv3.AnomalyDetector) *AnomalyMiddleware {
	return &AnomalyMiddleware{detector: d}
}

func (m *AnomalyMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.detector == nil {
			next.ServeHTTP(w, r)
			return
		}
		rw := NewResponseWriterWrapper(w, false)
		next.ServeHTTP(rw, r)
		tenant := GetTenantID(r.Context())
		key := GetAPIKeyID(r.Context())
		_ = m.detector.Record(tenant, key, rw.StatusCode(), rw.StatusCode() == http.StatusForbidden || rw.StatusCode() == http.StatusBadRequest, time.Now())
	})
}
