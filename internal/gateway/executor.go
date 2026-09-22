package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/events"
	gwcontext "github.com/lecodev-26/sentinelflow/internal/gateway/context"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/provider"
	"github.com/lecodev-26/sentinelflow/internal/router"
)

// Executor es el corazón del gateway.
type Executor struct {
	router *router.IntelligentRouter
}

// NewExecutor crea un nuevo Executor
func NewExecutor(r *router.IntelligentRouter) *Executor {
	return &Executor{router: r}
}

// Execute procesa una petición completa
func (e *Executor) Execute(ctx context.Context, rc *gwcontext.RequestContext, body []byte) ([]byte, int, error) {
	start := time.Now()

	// Parsear body a ChatRequest interno
	chatReq, err := parseChatRequest(body)
	if err != nil {
		rc.SetError(err.Error())
		events.Publish(ctx, events.Event{
			Type:      events.EventRouteFailed,
			RequestID: rc.RequestID,
			TenantID:  rc.TenantID,
			Payload:   map[string]interface{}{"error": err.Error()},
		})
		return nil, 400, err
	}

	// Actualizar el contexto con el modelo
	rc.Model = chatReq.Model
	rc.Stream = chatReq.Stream

	// Ejecutar con el router inteligente
	events.Publish(ctx, events.Event{
		Type:      events.EventProviderStarted,
		RequestID: rc.RequestID,
		TenantID:  rc.TenantID,
		Payload:   map[string]interface{}{"model": chatReq.Model},
	})

	resp, err := e.router.Route(ctx, chatReq)
	latency := time.Since(start)

	if err != nil {
		rc.SetError(err.Error())
		rc.StatusCode = 503
		events.Publish(ctx, events.Event{
			Type:      events.EventProviderFailed,
			RequestID: rc.RequestID,
			TenantID:  rc.TenantID,
			Payload:   map[string]interface{}{"error": err.Error()},
		})
		logger.Errorf("❌ Executor falló: %v", err)
		return nil, 503, err
	}

	// Actualizar contexto con el resultado
	rc.SetProvider(resp.Provider)
	rc.SetTokens(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, 0)
	rc.Latency = latency
	rc.StatusCode = 200

	events.Publish(ctx, events.Event{
		Type:      events.EventProviderCompleted,
		RequestID: rc.RequestID,
		TenantID:  rc.TenantID,
		Payload: map[string]interface{}{
			"provider": resp.Provider,
			"latency":  latency.Milliseconds(),
			"tokens":   rc.TotalTokens,
		},
	})

	// Serializar respuesta
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return nil, 500, fmt.Errorf("error marshaling response: %w", err)
	}

	return respJSON, 200, nil
}

// parseChatRequest convierte el body JSON a ChatRequest interno
func parseChatRequest(body []byte) (*provider.ChatRequest, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	req := &provider.ChatRequest{}

	if model, ok := raw["model"].(string); ok {
		req.Model = model
	}
	if stream, ok := raw["stream"].(bool); ok {
		req.Stream = stream
	}
	if temp, ok := raw["temperature"].(float64); ok {
		t := float32(temp)
		req.Temperature = &t
	}
	if maxTok, ok := raw["max_tokens"].(float64); ok {
		mt := int(maxTok)
		req.MaxTokens = &mt
	}

	if msgs, ok := raw["messages"].([]interface{}); ok {
		for _, m := range msgs {
			if msg, ok := m.(map[string]interface{}); ok {
				role, _ := msg["role"].(string)
				content, _ := msg["content"].(string)
				req.Messages = append(req.Messages, provider.Message{
					Role:    role,
					Content: content,
				})
			}
		}
	}

	return req, nil
}
