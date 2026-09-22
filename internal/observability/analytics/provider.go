package analytics

import (
	"sync"
	"time"
)

// ProviderStats contiene estadísticas de un provider
type ProviderStats struct {
	Provider      string        `json:"provider"`
	TotalRequests int64         `json:"total_requests"`
	SuccessCount  int64         `json:"success_count"`
	ErrorCount    int64         `json:"error_count"`
	TotalLatency  time.Duration `json:"total_latency"`
	MinLatency    time.Duration `json:"min_latency"`
	MaxLatency    time.Duration `json:"max_latency"`
	TotalTokens   int64         `json:"total_tokens"`
	TotalCost     float64       `json:"total_cost"`
	LastUsed      time.Time     `json:"last_used"`
}

// AvgLatency devuelve la latencia media
func (s *ProviderStats) AvgLatency() time.Duration {
	if s.TotalRequests == 0 {
		return 0
	}
	return s.TotalLatency / time.Duration(s.TotalRequests)
}

// SuccessRate devuelve la tasa de éxito
func (s *ProviderStats) SuccessRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.SuccessCount) / float64(s.TotalRequests) * 100
}

// ProviderAnalytics analiza el uso por provider
type ProviderAnalytics struct {
	mu    sync.RWMutex
	stats map[string]*ProviderStats
}

// NewProviderAnalytics crea un nuevo analizador
func NewProviderAnalytics() *ProviderAnalytics {
	return &ProviderAnalytics{
		stats: make(map[string]*ProviderStats),
	}
}

// Record registra una petición
func (p *ProviderAnalytics) Record(provider string, latency time.Duration, success bool, tokens int, cost float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	stat, exists := p.stats[provider]
	if !exists {
		stat = &ProviderStats{
			Provider:   provider,
			MinLatency: time.Hour,
		}
		p.stats[provider] = stat
	}

	stat.TotalRequests++
	if success {
		stat.SuccessCount++
	} else {
		stat.ErrorCount++
	}
	stat.TotalLatency += latency
	if latency < stat.MinLatency {
		stat.MinLatency = latency
	}
	if latency > stat.MaxLatency {
		stat.MaxLatency = latency
	}
	stat.TotalTokens += int64(tokens)
	stat.TotalCost += cost
	stat.LastUsed = time.Now()
}

// GetAll devuelve todas las estadísticas
func (p *ProviderAnalytics) GetAll() []*ProviderStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*ProviderStats, 0, len(p.stats))
	for _, s := range p.stats {
		result = append(result, s)
	}
	return result
}

// Get devuelve las estadísticas de un provider
func (p *ProviderAnalytics) Get(provider string) (*ProviderStats, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, ok := p.stats[provider]
	return s, ok
}
