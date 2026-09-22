package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/accounting"
	gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
	"github.com/lecodev-26/sentinelflow/internal/logger"
)

// AccountingMiddleware registra el uso real usando accounting.Service
type AccountingMiddleware struct {
	service *accounting.Service
	enabled bool
}

func NewAccountingMiddleware(service *accounting.Service, enabled bool) *AccountingMiddleware {
	return &AccountingMiddleware{
		service: service,
		enabled: enabled,
	}
}

func (m *AccountingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled || m.service == nil {
			next.ServeHTTP(w, r)
			return
		}

		rc, ok := gwcontext.FromContext(r.Context())
		if !ok {
			WriteError(w, NewInternalError("missing request context", nil))
			return
		}

		recorder := &accountingRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			buffer:         &bytes.Buffer{},
		}

		start := time.Now()
		next.ServeHTTP(recorder, r)
		latency := time.Since(start)

		// Construir UsageEvent
		event := accounting.NewUsageEvent(rc.RequestID, rc.TenantID, rc.Provider, rc.Model)
		event.UserID = rc.UserID
		event.APIKeyID = rc.APIKeyID
		event.ProjectID = rc.ProjectID
		event.Latency = latency
		event.CacheHit = rc.CacheHit
		event.CacheLayer = rc.CacheLayer

		if recorder.statusCode >= 400 {
			event.Status = "error"
		} else if rc.CacheHit {
			event.Status = "cached"
		} else {
			event.Status = "success"
		}

		// Extraer usage de la respuesta
		if recorder.statusCode == http.StatusOK {
			var respBody map[string]interface{}
			if err := json.Unmarshal(recorder.buffer.Bytes(), &respBody); err == nil {
				if usage, ok := respBody["usage"].(map[string]interface{}); ok {
					if pt, ok := usage["prompt_tokens"].(float64); ok {
						event.InputTokens = int(pt)
					}
					if ct, ok := usage["completion_tokens"].(float64); ok {
						event.OutputTokens = int(ct)
					}
				}
				if p, ok := respBody["provider"].(string); ok {
					event.Provider = p
				}
				if mdl, ok := respBody["model"].(string); ok {
					event.Model = mdl
				}
			}
		}

		event.CalculateTotals()

		// Registrar
		m.service.Record(r.Context(), event)

		// Actualizar RequestContext
		rc.SetCost(event.CostUSD)
		if event.Provider != "" {
			rc.SetProvider(event.Provider)
		}

		logger.Infof("💰 Accounting: tenant=%s provider=%s model=%s tokens=%d cost=$%.6f status=%s",
			event.TenantID, event.Provider, event.Model, event.TotalTokens, event.CostUSD, event.Status)
	})
}

type accountingRecorder struct {
	http.ResponseWriter
	statusCode int
	buffer     *bytes.Buffer
}

func (r *accountingRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *accountingRecorder) Write(b []byte) (int, error) {
	r.buffer.Write(b)
	return r.ResponseWriter.Write(b)
}
