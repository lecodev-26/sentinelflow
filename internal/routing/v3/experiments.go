package v3

import (
"hash/fnv"
"sync"
)

// Experiment representa un experimento de routing
type Experiment struct {
ID          string             `json:"id"`
Name        string             `json:"name"`
Enabled     bool               `json:"enabled"`
Traffic     map[string]float64 `json:"traffic"` // providerID → % (0.0-1.0)
Description string             `json:"description,omitempty"`
}

// ExperimentManager gestiona los experimentos de routing
type ExperimentManager struct {
mu          sync.RWMutex
experiments map[string]*Experiment // key: model or "*"
}

// NewExperimentManager crea un nuevo manager
func NewExperimentManager() *ExperimentManager {
return &ExperimentManager{
experiments: make(map[string]*Experiment),
}
}

// Add registra un experimento para un modelo
func (m *ExperimentManager) Add(model string, exp *Experiment) {
m.mu.Lock()
defer m.mu.Unlock()
m.experiments[model] = exp
}

// Get devuelve el experimento activo para un modelo
func (m *ExperimentManager) Get(model string) (*Experiment, bool) {
m.mu.RLock()
defer m.mu.RUnlock()

// Buscar específico
if exp, ok := m.experiments[model]; ok && exp.Enabled {
return exp, true
}
// Buscar wildcard
if exp, ok := m.experiments["*"]; ok && exp.Enabled {
return exp, true
}
return nil, false
}

// SelectProvider elige un provider según el experimento
// Usa hash consistente para que el mismo request-id siempre vaya al mismo
func (e *Experiment) SelectProvider(requestID string) string {
if !e.Enabled || len(e.Traffic) == 0 {
return ""
}

// Hash del requestID → 0.0-1.0
h := fnv.New32a()
h.Write([]byte(requestID))
hash := float64(h.Sum32()) / float64(^uint32(0))

// Recorrer providers por % acumulado
cumulative := 0.0
for provider, pct := range e.Traffic {
cumulative += pct
if hash <= cumulative {
return provider
}
}

// Fallback: último provider
for provider := range e.Traffic {
return provider
}
return ""
}

// List devuelve todos los experimentos
func (m *ExperimentManager) List() []*Experiment {
m.mu.RLock()
defer m.mu.RUnlock()
result := make([]*Experiment, 0, len(m.experiments))
for _, exp := range m.experiments {
result = append(result, exp)
}
return result
}

// Remove elimina un experimento
func (m *ExperimentManager) Remove(model string) bool {
m.mu.Lock()
defer m.mu.Unlock()
if _, exists := m.experiments[model]; !exists {
return false
}
delete(m.experiments, model)
return true
}
