package accounting

import (
"fmt"
"sync"

"github.com/lecodev-26/sentinelflow/internal/provider/model"
)

// PricingEngine calcula costes usando el Model Registry
type PricingEngine struct {
mu            sync.RWMutex
registry      *model.Registry
overrideCosts map[string]model.Pricing // overrides por provider:model
}

// NewPricingEngine crea un nuevo engine
func NewPricingEngine(registry *model.Registry) *PricingEngine {
return &PricingEngine{
registry:      registry,
overrideCosts: make(map[string]model.Pricing),
}
}

// Calculate calcula el coste de una petición
func (p *PricingEngine) Calculate(provider, modelID string, inputTokens, outputTokens, cachedTokens int) (float64, float64, float64, error) {
pricing, err := p.GetPricing(provider, modelID)
if err != nil {
return 0, 0, 0, err
}

// Calcular coste (precio por 1M tokens)
inputCost := (float64(inputTokens) / 1_000_000.0) * pricing.InputPer1M
outputCost := (float64(outputTokens) / 1_000_000.0) * pricing.OutputPer1M

// Descuento para cached (si aplica)
if cachedTokens > 0 && pricing.CachedPer1M > 0 {
cachedCost := (float64(cachedTokens) / 1_000_000.0) * pricing.CachedPer1M
// El cached reemplaza parte del input
inputCost = (float64(inputTokens-cachedTokens) / 1_000_000.0) * pricing.InputPer1M
inputCost += cachedCost
}

total := inputCost + outputCost
return total, inputCost, outputCost, nil
}

// GetPricing devuelve el pricing de un modelo
func (p *PricingEngine) GetPricing(provider, modelID string) (model.Pricing, error) {
p.mu.RLock()
override, hasOverride := p.overrideCosts[provider+":"+modelID]
p.mu.RUnlock()

if hasOverride {
return override, nil
}

if p.registry != nil {
if m, ok := p.registry.Get(provider, modelID); ok {
return m.Pricing, nil
}
}

return model.Pricing{}, fmt.Errorf("no pricing for %s:%s", provider, modelID)
}

// SetOverride permite sobrescribir el pricing
func (p *PricingEngine) SetOverride(provider, modelID string, pricing model.Pricing) {
p.mu.Lock()
defer p.mu.Unlock()
p.overrideCosts[provider+":"+modelID] = pricing
}

// SetRegistry actualiza el registry
func (p *PricingEngine) SetRegistry(reg *model.Registry) {
p.mu.Lock()
defer p.mu.Unlock()
p.registry = reg
}

// EstimateCost estima el coste sin tokens reales
func (p *PricingEngine) EstimateCost(provider, modelID string, estimatedInputTokens, estimatedOutputTokens int) float64 {
total, _, _, err := p.Calculate(provider, modelID, estimatedInputTokens, estimatedOutputTokens, 0)
if err != nil {
return 0
}
return total
}
