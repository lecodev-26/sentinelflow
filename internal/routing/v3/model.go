package v3

import (
	"sync"
	"time"
)

// ModelInfo describe un modelo con metadata completa
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Family       string   `json:"family,omitempty"`
	Capabilities []string `json:"capabilities"`
	ContextSize  int      `json:"context_size"`
	MaxOutput    int      `json:"max_output,omitempty"`

	// Pricing (USD per 1M tokens)
	InputPer1M  float64 `json:"input_per_1m"`
	OutputPer1M float64 `json:"output_per_1m"`

	// Metadata
	Deprecated bool                   `json:"deprecated,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ModelRegistry gestiona los modelos disponibles
type ModelRegistry struct {
	mu     sync.RWMutex
	models map[string]*ModelInfo // key: "provider:model-id"
}

// NewModelRegistry crea un nuevo registry
func NewModelRegistry() *ModelRegistry {
	r := &ModelRegistry{
		models: make(map[string]*ModelInfo),
	}
	r.loadDefaults()
	return r
}

// Register registra un modelo
func (r *ModelRegistry) Register(m *ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[m.Provider+":"+m.ID] = m
}

// Get devuelve un modelo por provider+ID
func (r *ModelRegistry) Get(provider, modelID string) (*ModelInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[provider+":"+modelID]
	return m, ok
}

// GetByID devuelve todos los modelos con ese ID (pueden ser varios providers)
func (r *ModelRegistry) GetByID(modelID string) []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ModelInfo
	for _, m := range r.models {
		if m.ID == modelID {
			result = append(result, m)
		}
	}
	return result
}

// List devuelve todos los modelos
func (r *ModelRegistry) List() []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ModelInfo, 0, len(r.models))
	for _, m := range r.models {
		result = append(result, m)
	}
	return result
}

// ListByProvider lista modelos de un provider
func (r *ModelRegistry) ListByProvider(provider string) []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ModelInfo
	for _, m := range r.models {
		if m.Provider == provider {
			result = append(result, m)
		}
	}
	return result
}

// loadDefaults carga el catálogo por defecto con pricing real
func (r *ModelRegistry) loadDefaults() {
	now := time.Now()
	_ = now

	// === OpenAI ===
	r.Register(&ModelInfo{
		ID: "gpt-4", Name: "GPT-4", Provider: "openai", Family: "gpt-4",
		Capabilities: []string{"chat", "stream", "tools", "vision", "json", "function"},
		ContextSize:  8192, MaxOutput: 4096,
		InputPer1M: 30.00, OutputPer1M: 60.00,
	})

	r.Register(&ModelInfo{
		ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "openai", Family: "gpt-4",
		Capabilities: []string{"chat", "stream", "tools", "vision", "json", "function"},
		ContextSize:  128000, MaxOutput: 4096,
		InputPer1M: 10.00, OutputPer1M: 30.00,
	})

	r.Register(&ModelInfo{
		ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "openai", Family: "gpt-3.5",
		Capabilities: []string{"chat", "stream", "tools", "json", "function"},
		ContextSize:  16385, MaxOutput: 4096,
		InputPer1M: 0.50, OutputPer1M: 1.50,
	})

	// === Anthropic ===
	r.Register(&ModelInfo{
		ID: "claude-3-opus", Name: "Claude 3 Opus", Provider: "anthropic", Family: "claude-3",
		Capabilities: []string{"chat", "stream", "tools", "vision"},
		ContextSize:  200000, MaxOutput: 4096,
		InputPer1M: 15.00, OutputPer1M: 75.00,
	})

	r.Register(&ModelInfo{
		ID: "claude-3-sonnet", Name: "Claude 3 Sonnet", Provider: "anthropic", Family: "claude-3",
		Capabilities: []string{"chat", "stream", "tools", "vision"},
		ContextSize:  200000, MaxOutput: 4096,
		InputPer1M: 3.00, OutputPer1M: 15.00,
	})

	r.Register(&ModelInfo{
		ID: "claude-3-haiku", Name: "Claude 3 Haiku", Provider: "anthropic", Family: "claude-3",
		Capabilities: []string{"chat", "stream", "tools", "vision"},
		ContextSize:  200000, MaxOutput: 4096,
		InputPer1M: 0.25, OutputPer1M: 1.25,
	})

	// === Ollama (local, gratis) ===
	r.Register(&ModelInfo{
		ID: "llama3", Name: "Llama 3", Provider: "ollama", Family: "llama",
		Capabilities: []string{"chat", "stream"},
		ContextSize:  8192, MaxOutput: 4096,
		InputPer1M: 0.0, OutputPer1M: 0.0,
	})

	r.Register(&ModelInfo{
		ID: "llama3.1", Name: "Llama 3.1", Provider: "ollama", Family: "llama",
		Capabilities: []string{"chat", "stream", "tools"},
		ContextSize:  128000, MaxOutput: 4096,
		InputPer1M: 0.0, OutputPer1M: 0.0,
	})

	r.Register(&ModelInfo{
		ID: "mistral", Name: "Mistral", Provider: "ollama", Family: "mistral",
		Capabilities: []string{"chat", "stream"},
		ContextSize:  8192, MaxOutput: 4096,
		InputPer1M: 0.0, OutputPer1M: 0.0,
	})
}
