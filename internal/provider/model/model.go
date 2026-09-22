package model

import (
	"sync"
	"time"
)

// Capability representa una capacidad de un modelo
type Capability string

const (
	CapabilityChat     Capability = "chat"
	CapabilityStream   Capability = "stream"
	CapabilityTools    Capability = "tools"
	CapabilityVision   Capability = "vision"
	CapabilityAudio    Capability = "audio"
	CapabilityEmbed    Capability = "embedding"
	CapabilityJSON     Capability = "json_mode"
	CapabilityFunction Capability = "function_calling"
)

// Pricing contiene los precios por modelo
type Pricing struct {
	InputPer1M  float64 `json:"input_per_1m_usd"`
	OutputPer1M float64 `json:"output_per_1m_usd"`
	CachedPer1M float64 `json:"cached_per_1m_usd,omitempty"`
}

// Limits contiene los límites de un modelo
type Limits struct {
	MaxContextTokens  int `json:"max_context_tokens"`
	MaxOutputTokens   int `json:"max_output_tokens"`
	MaxRequestsPerMin int `json:"max_requests_per_min,omitempty"`
}

// Model representa un modelo de IA
type Model struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Provider     string                 `json:"provider"`
	Family       string                 `json:"family,omitempty"`
	Capabilities []Capability           `json:"capabilities"`
	Pricing      Pricing                `json:"pricing"`
	Limits       Limits                 `json:"limits"`
	Deprecated   bool                   `json:"deprecated,omitempty"`
	DiscoveredAt time.Time              `json:"discovered_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// HasCapability verifica si el modelo tiene una capacidad
func (m *Model) HasCapability(c Capability) bool {
	for _, cap := range m.Capabilities {
		if cap == c {
			return true
		}
	}
	return false
}

// Registry es el registro central de modelos
type Registry struct {
	mu     sync.RWMutex
	models map[string]*Model // key: "provider:model-id"
}

// NewRegistry crea un nuevo registry
func NewRegistry() *Registry {
	return &Registry{
		models: make(map[string]*Model),
	}
}

// Register registra un modelo
func (r *Registry) Register(m *Model) {
	if m.DiscoveredAt.IsZero() {
		m.DiscoveredAt = time.Now()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := m.Provider + ":" + m.ID
	r.models[key] = m
}

// Get devuelve un modelo por provider+id
func (r *Registry) Get(provider, modelID string) (*Model, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[provider+":"+modelID]
	return m, ok
}

// GetByID devuelve modelos con ese ID (de cualquier provider)
func (r *Registry) GetByID(modelID string) []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Model
	for _, m := range r.models {
		if m.ID == modelID {
			result = append(result, m)
		}
	}
	return result
}

// FindByCapability devuelve modelos con una capacidad
func (r *Registry) FindByCapability(c Capability) []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*Model
	for _, m := range r.models {
		if m.HasCapability(c) {
			result = append(result, m)
		}
	}
	return result
}

// List devuelve todos los modelos
func (r *Registry) List() []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Model, 0, len(r.models))
	for _, m := range r.models {
		result = append(result, m)
	}
	return result
}

// Delete elimina un modelo
func (r *Registry) Delete(provider, modelID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := provider + ":" + modelID
	if _, exists := r.models[key]; !exists {
		return false
	}
	delete(r.models, key)
	return true
}

// Clear vacía el registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models = make(map[string]*Model)
}
