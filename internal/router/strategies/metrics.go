package strategies

import (
"sync"
"time"
)

// ProviderMetrics almacena métricas de un proveedor
type ProviderMetrics struct {
mu               sync.RWMutex
SuccessRate      float64       `json:"success_rate"`
AvgLatency       time.Duration `json:"avg_latency"`
P95Latency       time.Duration `json:"p95_latency"`
CostPerToken     float64       `json:"cost_per_token"`
LastChecked      time.Time     `json:"last_checked"`
TotalRequests    int           `json:"total_requests"`
TotalFailures    int           `json:"total_failures"`
TotalTokens      int           `json:"total_tokens"`
}

// MetricsStore almacena métricas de todos los proveedores
type MetricsStore struct {
mu        sync.RWMutex
metrics   map[string]*ProviderMetrics
}

// NewMetricsStore crea un nuevo store
func NewMetricsStore() *MetricsStore {
return &MetricsStore{
metrics: make(map[string]*ProviderMetrics),
}
}

// GetOrCreate obtiene o crea métricas para un proveedor
func (s *MetricsStore) GetOrCreate(provider string) *ProviderMetrics {
s.mu.Lock()
defer s.mu.Unlock()

if m, exists := s.metrics[provider]; exists {
return m
}

m := &ProviderMetrics{
CostPerToken: 0.00001, // valor por defecto
LastChecked:  time.Now(),
}
s.metrics[provider] = m
return m
}

// RecordRequest registra una petición
func (s *MetricsStore) RecordRequest(provider string, latency time.Duration, success bool, tokens int, cost float64) {
m := s.GetOrCreate(provider)
m.mu.Lock()
defer m.mu.Unlock()

m.TotalRequests++
m.TotalTokens += tokens
m.CostPerToken = (m.CostPerToken*float64(m.TotalRequests-1) + cost) / float64(m.TotalRequests)

if !success {
m.TotalFailures++
}

// Calcular success rate
if m.TotalRequests > 0 {
m.SuccessRate = float64(m.TotalRequests-m.TotalFailures) / float64(m.TotalRequests) * 100
}

// Actualizar latencias
m.AvgLatency = time.Duration((int64(m.AvgLatency)*int64(m.TotalRequests-1) + int64(latency)) / int64(m.TotalRequests))
if latency > m.P95Latency {
m.P95Latency = latency
}
m.LastChecked = time.Now()
}

// GetMetrics devuelve todas las métricas
func (s *MetricsStore) GetMetrics() map[string]*ProviderMetrics {
s.mu.RLock()
defer s.mu.RUnlock()

result := make(map[string]*ProviderMetrics)
for k, v := range s.metrics {
result[k] = v
}
return result
}
