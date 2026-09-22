package analytics

import (
	"sync"
	"time"
)

// CostAnalytics analiza costes por múltiples dimensiones
type CostAnalytics struct {
	mu sync.RWMutex

	// Por tenant
	byTenant map[string]float64

	// Por provider
	byProvider map[string]float64

	// Por modelo
	byModel map[string]float64

	// Por día
	byDay map[string]float64

	// Total
	total float64
}

// NewCostAnalytics crea un nuevo analizador
func NewCostAnalytics() *CostAnalytics {
	return &CostAnalytics{
		byTenant:   make(map[string]float64),
		byProvider: make(map[string]float64),
		byModel:    make(map[string]float64),
		byDay:      make(map[string]float64),
	}
}

// Record registra un coste
func (c *CostAnalytics) Record(tenantID, provider, model string, cost float64, t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.total += cost
	c.byTenant[tenantID] += cost
	c.byProvider[provider] += cost
	c.byModel[model] += cost
	c.byDay[t.Format("2006-01-02")] += cost
}

// Snapshot devuelve una foto de los datos
type CostSnapshot struct {
	Total      float64            `json:"total"`
	ByTenant   map[string]float64 `json:"by_tenant"`
	ByProvider map[string]float64 `json:"by_provider"`
	ByModel    map[string]float64 `json:"by_model"`
	ByDay      map[string]float64 `json:"by_day"`
}

// Snapshot devuelve una foto de los datos
func (c *CostAnalytics) Snapshot() CostSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CostSnapshot{
		Total:      c.total,
		ByTenant:   copyMap(c.byTenant),
		ByProvider: copyMap(c.byProvider),
		ByModel:    copyMap(c.byModel),
		ByDay:      copyMap(c.byDay),
	}
}

// GetByTenant devuelve el coste de un tenant
func (c *CostAnalytics) GetByTenant(tenantID string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.byTenant[tenantID]
}

func copyMap(m map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
