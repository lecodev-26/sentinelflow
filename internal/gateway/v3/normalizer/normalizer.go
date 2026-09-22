package normalizer

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Request es el formato interno unificado
type Request struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Temperature *float32               `json:"temperature,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream"`
	Tools       []Tool                 `json:"tools,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Message representa un mensaje en formato interno
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool representa una herramienta (function calling)
type Tool struct {
	Type     string                 `json:"type"`
	Function map[string]interface{} `json:"function"`
}

// Format indica el formato de origen
type Format string

const (
	FormatOpenAI    Format = "openai"
	FormatAnthropic Format = "anthropic"
	FormatGemini    Format = "gemini"
	FormatInternal  Format = "internal"
	FormatUnknown   Format = "unknown"
)

// Normalizer convierte requests de diferentes APIs al formato interno
type Normalizer struct{}

// New crea un normalizador
func New() *Normalizer {
	return &Normalizer{}
}

// Normalize detecta el formato y convierte al formato interno
func (n *Normalizer) Normalize(raw []byte) (*Request, Format, error) {
	// Intentar parsear como JSON genérico
	var generic map[string]interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, FormatUnknown, fmt.Errorf("invalid JSON: %w", err)
	}

	// Detectar formato por campos característicos
	if _, hasContents := generic["contents"]; hasContents {
		return n.normalizeGemini(raw)
	}
	if _, hasSystem := generic["system"]; hasSystem {
		return n.normalizeAnthropic(raw)
	}
	// Por defecto OpenAI
	return n.normalizeOpenAI(raw)
}

// === OpenAI ===

func (n *Normalizer) normalizeOpenAI(raw []byte) (*Request, Format, error) {
	var req struct {
		Model       string                 `json:"model"`
		Messages    []Message              `json:"messages"`
		Temperature *float32               `json:"temperature,omitempty"`
		MaxTokens   *int                   `json:"max_tokens,omitempty"`
		Stream      bool                   `json:"stream"`
		Tools       []Tool                 `json:"tools,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, FormatOpenAI, fmt.Errorf("error parsing OpenAI format: %w", err)
	}

	if req.Model == "" {
		return nil, FormatOpenAI, errors.New("model is required")
	}
	if len(req.Messages) == 0 {
		return nil, FormatOpenAI, errors.New("messages are required")
	}

	return &Request{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
		Tools:       req.Tools,
		Metadata:    req.Metadata,
	}, FormatOpenAI, nil
}

// === Anthropic ===

func (n *Normalizer) normalizeAnthropic(raw []byte) (*Request, Format, error) {
	var req struct {
		Model       string    `json:"model"`
		System      string    `json:"system,omitempty"`
		Messages    []Message `json:"messages"`
		Temperature *float32  `json:"temperature,omitempty"`
		MaxTokens   *int      `json:"max_tokens,omitempty"`
		Stream      bool      `json:"stream"`
	}

	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, FormatAnthropic, fmt.Errorf("error parsing Anthropic format: %w", err)
	}

	// Anthropic usa system como campo separado; lo convertimos a mensaje
	messages := req.Messages
	if req.System != "" {
		messages = append([]Message{{Role: "system", Content: req.System}}, messages...)
	}

	return &Request{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream,
	}, FormatAnthropic, nil
}

// === Gemini ===

func (n *Normalizer) normalizeGemini(raw []byte) (*Request, Format, error) {
	var req struct {
		Model    string `json:"model"`
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
		GenerationConfig struct {
			Temperature     *float32 `json:"temperature,omitempty"`
			MaxOutputTokens *int     `json:"max_output_tokens,omitempty"`
		} `json:"generationConfig"`
	}

	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, FormatGemini, fmt.Errorf("error parsing Gemini format: %w", err)
	}

	// Convertir contents a messages
	var messages []Message
	for _, c := range req.Contents {
		content := ""
		for _, p := range c.Parts {
			content += p.Text
		}
		role := c.Role
		if role == "model" {
			role = "assistant"
		}
		messages = append(messages, Message{Role: role, Content: content})
	}

	return &Request{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.GenerationConfig.Temperature,
		MaxTokens:   req.GenerationConfig.MaxOutputTokens,
	}, FormatGemini, nil
}

// ToOpenAI convierte el formato interno a OpenAI (para llamar al provider)
func (n *Normalizer) ToOpenAI(req *Request) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      req.Stream,
		"tools":       req.Tools,
	})
}
