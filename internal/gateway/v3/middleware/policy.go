package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/lecodev-26/sentinelflow/internal/logger"
	policyv3 "github.com/lecodev-26/sentinelflow/internal/policy/v3"
	securityv3 "github.com/lecodev-26/sentinelflow/internal/security/v3"
)

// PolicyEvalContextKey es la clave para el resultado de la política
type policyContextKey string

const (
	CtxPolicyResult   policyContextKey = "policy_result"
	CtxSecurityResult policyContextKey = "security_result"
)

// PolicyMiddleware aplica policy + security a las peticiones
type PolicyMiddleware struct {
	evaluator *policyv3.Evaluator
	security  *securityv3.Pipeline
	enabled   bool
}

// NewPolicyMiddleware crea un nuevo middleware
func NewPolicyMiddleware(evaluator *policyv3.Evaluator, enabled bool) *PolicyMiddleware {
	return &PolicyMiddleware{
		evaluator: evaluator,
		security:  securityv3.NewPipeline(),
		enabled:   enabled,
	}
}

// Handler envuelve un handler con policy + security
func (m *PolicyMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Solo procesar POST de chat
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			next.ServeHTTP(w, r)
			return
		}

		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Leer body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid_request", "error reading body")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Parsear
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid_request", "invalid JSON")
			return
		}

		model, _ := reqBody["model"].(string)

		// === 1. Evaluar política ===
		tenantID := getTenantFromCtx(r.Context())
		evalCtx := &policyv3.EvalContext{
			TenantID: tenantID,
			Model:    model,
		}

		policyResult := m.evaluator.Evaluate(r.Context(), evalCtx)

		if !policyResult.Allowed {
			logger.Warnf("🚫 Policy DENY: tenant=%s model=%s reason=%s",
				tenantID, model, policyResult.Reason)
			writeErrorJSON(w, http.StatusForbidden, "policy_denied", policyResult.Reason)
			return
		}

		// === 2. Security scan del input ===
		messages, _ := reqBody["messages"].([]interface{})
		securityDecision := securityv3.NewDecision()

		for i, msgRaw := range messages {
			msg, ok := msgRaw.(map[string]interface{})
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role != "user" {
				continue
			}

			content, _ := msg["content"].(string)
			if content == "" {
				continue
			}

			// Aplicar security policy si existe
			if policyResult.Security != nil {
				// PII
				if policyResult.Security.PIIDetection {
					piiScanner := securityv3.NewPIIScanner()
					for _, f := range piiScanner.Scan(r.Context(), content) {
						securityDecision.AddFinding(f)
					}
				}
				// Secrets
				if policyResult.Security.SecretDetection {
					secretScanner := securityv3.NewSecretScanner()
					for _, f := range secretScanner.Scan(r.Context(), content) {
						securityDecision.AddFinding(f)
					}
				}
				// Prompt injection
				if policyResult.Security.PromptInjection {
					injectionScanner := securityv3.NewPromptInjectionScanner()
					for _, f := range injectionScanner.Scan(r.Context(), content) {
						securityDecision.AddFinding(f)
					}
				}
				// SSRF
				if policyResult.Security.SSRFProtection {
					ssrfScanner := securityv3.NewSSRFScanner()
					for _, f := range ssrfScanner.Scan(r.Context(), content) {
						securityDecision.AddFinding(f)
					}
				}
			}

			// Si hay redacción, actualizar contenido
			if securityDecision.ShouldRedact() {
				messages[i].(map[string]interface{})["content"] = securityDecision.ProcessedText
			}
		}

		// Si bloqueado, devolver error
		if securityDecision.IsBlocked() {
			logger.Warnf("🚫 Security BLOCK: tenant=%s model=%s risk=%s findings=%d",
				tenantID, model, securityDecision.Risk, len(securityDecision.Findings))
			writeErrorJSON(w, http.StatusBadRequest, "security_blocked",
				"request blocked by security policy: "+string(securityDecision.Risk))
			return
		}

		// Reconstruir body si hubo redacción
		if securityDecision.ShouldRedact() {
			reqBody["messages"] = messages
			newBody, _ := json.Marshal(reqBody)
			r.Body = io.NopCloser(bytes.NewReader(newBody))
			r.ContentLength = int64(len(newBody))
			logger.Warnf("⚠️ PII redacted: %d findings", len(securityDecision.Findings))
		}

		// Inyectar en contexto
		ctx := context.WithValue(r.Context(), CtxPolicyResult, policyResult)
		ctx = context.WithValue(ctx, CtxSecurityResult, securityDecision)

		// Log de éxito
		if len(securityDecision.Findings) > 0 {
			logger.Infof("🛡️ Security scan: risk=%s findings=%d action=%s",
				securityDecision.Risk, len(securityDecision.Findings), securityDecision.Action)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetPolicyResult devuelve el resultado de la política
func GetPolicyResult(ctx context.Context) (*policyv3.EvalResult, bool) {
	v, ok := ctx.Value(CtxPolicyResult).(*policyv3.EvalResult)
	return v, ok
}

// GetSecurityResult devuelve el resultado de seguridad
func GetSecurityResult(ctx context.Context) (*securityv3.Decision, bool) {
	v, ok := ctx.Value(CtxSecurityResult).(*securityv3.Decision)
	return v, ok
}

func getTenantFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(CtxTenantID).(string); ok {
		return v
	}
	return "default"
}

func writeErrorJSON(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"type":    errType,
			"message": message,
		},
	})
}
