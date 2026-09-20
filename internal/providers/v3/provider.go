package v3

import (
"context"
"errors"
"time"
)

// Capability representa una capacidad de un provider o modelo
type Capability string

const (
CapChat     Capability = "chat"
CapStream   Capability = "stream"
CapTools    Capability = "tools"
CapVision   Capability = "vision"
CapEmbed    Capability = "embedding"
CapFunction Capability = "function_calling"
CapJSON     Capability = "json_mode"
)

// Message representa un mensaje normalizado
type Message struct {
Role    string `json:"role"`
Content string `json:"content"`
}

// ChatRequest es la petición normalizada interna
type ChatRequest struct {
Model       string    `json:"model"`
Messages    []Message `json:"messages"`
Temperature *float32  `json:"temperature,omitempty"`
MaxTokens   *int      `json:"max_tokens,omitempty"`
Stream      bool      `json:"stream"`
}

// Choice es una opción de respuesta
type Choice struct {
Index        int     `json:"index"`
Message      Message `json:"message"`
FinishReason string  `json:"finish_reason,omitempty"`
}

// Usage contiene el uso de tokens
type Usage struct {
PromptTokens     int `json:"prompt_tokens"`
CompletionTokens int `json:"completion_tokens"`
TotalTokens      int `json:"total_tokens"`
}

// ChatResponse es la respuesta normalizada
type ChatResponse struct {
ID        string        `json:"id"`
Model     string        `json:"model"`
Provider  string        `json:"provider"`
Choices   []Choice      `json:"choices"`
Usage     Usage         `json:"usage"`
Latency   time.Duration `json:"latency"`
Timestamp time.Time     `json:"timestamp"`
}

// StreamChunk es un fragmento de respuesta en streaming
type StreamChunk struct {
Delta        string    `json:"delta"`
FinishReason string    `json:"finish_reason,omitempty"`
Index        int       `json:"index"`
Timestamp    time.Time `json:"timestamp"`
}

// ModelInfo describe un modelo disponible
type ModelInfo struct {
ID           string       `json:"id"`
Name         string       `json:"name"`
Provider     string       `json:"provider"`
Capabilities []Capability `json:"capabilities"`
ContextSize  int          `json:"context_size"`
MaxOutput    int          `json:"max_output,omitempty"`
}

// HealthStatus representa el estado de un provider
type HealthStatus string

const (
HealthUnknown   HealthStatus = "unknown"
HealthHealthy   HealthStatus = "healthy"
HealthDegraded  HealthStatus = "degraded"
HealthUnhealthy HealthStatus = "unhealthy"
HealthDisabled  HealthStatus = "disabled"
)

// Provider es la interface que todos los providers deben implementar
type Provider interface {
// ID devuelve el identificador único del provider
ID() string

// Name devuelve el nombre legible
Name() string

// Capabilities devuelve las capacidades soportadas
Capabilities() []Capability

// Models devuelve los modelos disponibles
Models(ctx context.Context) ([]ModelInfo, error)

// Chat ejecuta una petición de chat completa
Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

// Stream ejecuta una petición de chat con streaming
Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)

// Health verifica el estado del provider
Health(ctx context.Context) error
}

// Errors comunes
var (
ErrNotSupported    = errors.New("not supported")
ErrUnauthorized    = errors.New("unauthorized")
ErrRateLimited     = errors.New("rate limited")
ErrProviderDown    = errors.New("provider down")
ErrInvalidRequest  = errors.New("invalid request")
ErrContextTooLarge = errors.New("context too large")
)
