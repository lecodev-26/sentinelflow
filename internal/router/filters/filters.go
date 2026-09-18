package filters

import (
"github.com/lecodev-26/sentinelflow/internal/provider"
"github.com/lecodev-26/sentinelflow/internal/provider/model"
"github.com/lecodev-26/sentinelflow/internal/router/scoring"
)

// Request describe los requisitos de una petición
type Request struct {
Model        string
Capabilities []model.Capability
MaxCost      float64       // USD por 1M tokens
MaxLatency   int           // ms
MinHealth    float64       // 0-1
RequiredProvider string    // forzar provider específico
ExcludeProviders []string  // excluir providers
}

// Filter filtra candidatos basado en requisitos
type Filter struct {
modelRegistry *model.Registry
}

// NewFilter crea un nuevo filtro
func NewFilter(reg *model.Registry) *Filter {
return &Filter{modelRegistry: reg}
}

// Apply filtra los candidatos
func (f *Filter) Apply(
providers []provider.Provider,
healthStatus map[string]string,
circuitStatus map[string]string,
req Request,
) []scoring.Candidate {
var candidates []scoring.Candidate

for _, p := range providers {
// 1. Filtro por provider específico
if req.RequiredProvider != "" && p.Name() != req.RequiredProvider {
continue
}

// 2. Filtro por providers excluidos
if contains(req.ExcludeProviders, p.Name()) {
continue
}

// 3. Filtro por circuit breaker
if circuitStatus[p.Name()] == "open" {
continue
}

// 4. Filtro por health
health := healthStatus[p.Name()]
if health == "disabled" || health == "unhealthy" {
continue
}

// 5. Buscar modelo
var m *model.Model
if req.Model != "" {
if found, ok := f.modelRegistry.Get(p.Name(), req.Model); ok {
m = found
} else {
// El provider no tiene ese modelo
continue
}
}

// 6. Filtro por capabilities
if len(req.Capabilities) > 0 && m != nil {
if !hasAllCapabilities(m, req.Capabilities) {
continue
}
}

// 7. Filtro por coste máximo
if req.MaxCost > 0 && m != nil {
cost := m.Pricing.InputPer1M + m.Pricing.OutputPer1M
if cost > req.MaxCost {
continue
}
}

candidates = append(candidates, scoring.Candidate{
ProviderName: p.Name(),
Model:        m,
HealthStatus: health,
CircuitState: circuitStatus[p.Name()],
})
}

return candidates
}

func contains(list []string, s string) bool {
for _, item := range list {
if item == s {
return true
}
}
return false
}

func hasAllCapabilities(m *model.Model, caps []model.Capability) bool {
for _, c := range caps {
if !m.HasCapability(c) {
return false
}
}
return true
}
