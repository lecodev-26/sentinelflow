package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	accountingv3 "github.com/lecodev-26/sentinelflow/internal/accounting/v3"
	"github.com/lecodev-26/sentinelflow/internal/idgen"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

// AccountingMiddleware registra el uso después del handler
type AccountingMiddleware struct {
	service *accountingv3.Service
	enabled bool
}

// NewAccountingMiddleware crea un nuevo middleware
func NewAccountingMiddleware(service *accountingv3.Service, enabled bool) *AccountingMiddleware {
	return &AccountingMiddleware{
		service: service,
		enabled: enabled,
	}
}

// Handler envuelve un handler con accounting
func (m *AccountingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled || m.service == nil || r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			next.ServeHTTP(w, r)
			return
		}

		isStream := isStreamingRequest(r)
		recorder := NewResponseWriterWrapper(w, !isStream)

		start := time.Now()
		next.ServeHTTP(recorder, r)
		latency := time.Since(start)

		if recorder.StatusCode() == http.StatusBadRequest ||
			recorder.StatusCode() == http.StatusForbidden {
			return
		}

		tenantID := getTenantFromCtx(r.Context())
		userID := getUserFromCtx(r.Context())
		apiKeyID := getAPIKeyFromCtx(r.Context())

		providerID := recorder.Header().Get("X-Provider")
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = r.Header.Get("Idempotency-Key")
		}

		var inputTokens, outputTokens int
		var model string

		if !isStream && recorder.StatusCode() == http.StatusOK {
			var respBody map[string]interface{}
			if err := json.Unmarshal(recorder.Buffer(), &respBody); err == nil {
				if usage, ok := respBody["usage"].(map[string]interface{}); ok {
					if v, ok := usage["prompt_tokens"].(float64); ok {
						inputTokens = int(v)
					}
					if v, ok := usage["completion_tokens"].(float64); ok {
						outputTokens = int(v)
					}
				}
				if m, ok := respBody["model"].(string); ok {
					model = m
				}
			}
		}

		status := "success"
		if recorder.StatusCode() >= 400 {
			status = "error"
		}

		rec := &postgres.UsageRecord{
			ID:           "usage_" + idgen.RandomHex(6),
			RequestID:    requestID,
			TenantID:     tenantID,
			UserID:       userID,
			APIKeyID:     apiKeyID,
			Provider:     providerID,
			Model:        model,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			LatencyMs:    int(latency.Milliseconds()),
			Status:       status,
			Timestamp:    time.Now(),
		}

		go func() {
			_ = m.service.Record(r.Context(), rec)
		}()
	})
}

func isStreamingRequest(r *http.Request) bool {
	if r.Header.Get("Accept") == "text/event-stream" {
		return true
	}
	return false
}

func getUserFromCtx(ctx interface{ Value(interface{}) interface{} }) string {
	if v, ok := ctx.Value(CtxUserID).(string); ok {
		return v
	}
	return ""
}

func getAPIKeyFromCtx(ctx interface{ Value(interface{}) interface{} }) string {
	if v, ok := ctx.Value(CtxAPIKeyID).(string); ok {
		return v
	}
	return ""
}
