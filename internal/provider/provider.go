package provider

import (
"context"
"time"
)

// Message representa un mensaje en una conversación
type Message struct {
Role    string `json:"role"`    // system, user, assistant, tool
Content string `json:"content"` // Contenido del mensaje
}

// ChatRequest es la petición normalizada interna
type ChatRequest struct {
Model       string    `json:"model"`
Messages    []Message `json:"messages"`
Temperature *float32  `json:"temperature,omitempty"`
MaxTokens   *int      `json:"max_tokens,omitempty"`
Stream      bool      `json:"stream"`
}

// ChatResponse es la respuesta normalizada interna
type ChatResponse struct {
ID        string    `json:"id"`
Model     string    `json:"model"`
Provider  string    `json:"provider"`
Choices   []Choice  `json:"choices"`
Usage     Usage     `json:"usage"`
Latency   time.Duration `json:"latency"`
Timestamp time.Time `json:"timestamp"`
}

// Choice representa una opción de respuesta
type Choice struct {
Index        int     `json:"index"`
Message      Message `json:"message"`
FinishReason string  `json:"finish_reason,omitempty"`
}

// Usage representa el consumo de tokens
type Usage struct {
PromptTokens     int `json:"prompt_tokens"`
CompletionTokens int `json:"completion_tokens"`
TotalTokens      int `json:"total_tokens"`
}

// Event representa un evento de streaming
type Event struct {
Type    string      `json:"type"`    // "chunk", "done", "error"
Content string      `json:"content"` // Contenido del chunk
Data    interface{} `json:"data,omitempty"`
}

// Provider es la interfaz que deben implementar todos los proveedores
type Provider interface {
// Name devuelve el nombre del proveedor
Name() string

// Chat realiza una petición de chat completa (no streaming)
Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

// Stream realiza una petición de chat con streaming
Stream(ctx context.Context, req *ChatRequest) (<-chan Event, error)

// Health verifica si el proveedor está operativo
Health(ctx context.Context) error

// Models devuelve la lista de modelos disponibles
Models(ctx context.Context) ([]string, error)
}
