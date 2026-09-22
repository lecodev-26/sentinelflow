package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/security"
)

// SecurityMiddleware escanea las peticiones con el pipeline unificado
type SecurityMiddleware struct {
	pipeline *security.Pipeline
	enabled  bool
}

func NewSecurityMiddleware(enabled bool) *SecurityMiddleware {
	return &SecurityMiddleware{
		pipeline: security.NewPipeline(),
		enabled:  enabled,
	}
}

func (m *SecurityMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		rc, ok := gwcontext.FromContext(r.Context())
		if !ok {
			WriteError(w, NewInternalError("missing request context", nil))
			return
		}

		// Leer body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			WriteError(w, NewInvalidRequestError("error reading body"))
			return
		}
		defer r.Body.Close()

		// Parsear JSON
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			// No es JSON, no escanear
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
			return
		}

		// Extraer mensajes
		var messages []map[string]interface{}
		if msgs, ok := reqBody["messages"].([]interface{}); ok {
			for _, m := range msgs {
				if msg, ok := m.(map[string]interface{}); ok {
					messages = append(messages, msg)
				}
			}
		}

		// Decisión global (se mergea con cada mensaje)
		globalDecision := security.NewDecision()

		// Escanear cada mensaje del usuario
		for i, msg := range messages {
			role, _ := msg["role"].(string)
			if role != "user" {
				continue
			}

			content, _ := msg["content"].(string)
			if content == "" {
				continue
			}

			// Evaluar con el pipeline
			decision := m.pipeline.Evaluate(r.Context(), content)
			globalDecision.Merge(decision)

			// Si hay redacción, usar el texto procesado
			if decision.ShouldRedact() {
				messages[i]["content"] = decision.ProcessedText
				logger.Warnf("⚠️ PII redactada: %d hallazgos", len(decision.Findings))
			}
		}

		// Guardar decisión en el contexto
		rc.SetMetadata("security_decision", globalDecision)
		rc.SetMetadata("security_findings", len(globalDecision.Findings))
		rc.SetMetadata("security_risk", string(globalDecision.Risk))

		// Si está bloqueado, devolver error
		if globalDecision.IsBlocked() {
			logger.Warnf("🚫 Request bloqueado por seguridad: %s (risk=%s, findings=%d)",
				globalDecision.Reason, globalDecision.Risk, len(globalDecision.Findings))
			WriteError(w, NewPolicyDeniedError("request blocked by security policy"))
			return
		}

		// Reconstruir body con cambios (si hubo redacción)
		if globalDecision.ShouldRedact() {
			reqBody["messages"] = messages
			newBody, err := json.Marshal(reqBody)
			if err != nil {
				WriteError(w, NewInternalError("error rebuilding body", err))
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(newBody))
			r.ContentLength = int64(len(newBody))
		} else {
			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		next.ServeHTTP(w, r)
	})
}
