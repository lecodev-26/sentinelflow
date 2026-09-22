package v3

import (
	"context"
	"sync"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/logger"
)

// ManagedProvider combina un provider con su health y circuit breaker
type ManagedProvider struct {
	Provider       Provider
	Health         *ProviderHealth
	CircuitBreaker *CircuitBreaker
}

// Manager gestiona todos los providers con health + CB
type Manager struct {
	registry        *Registry
	healthMonitor   *HealthMonitor
	circuitBreakers map[string]*CircuitBreaker
	mu              sync.RWMutex
}

// NewManager crea un nuevo manager
func NewManager(registry *Registry) *Manager {
	m := &Manager{
		registry:        registry,
		circuitBreakers: make(map[string]*CircuitBreaker),
	}

	// Crear health monitor
	m.healthMonitor = NewHealthMonitor(registry, 30*time.Second, 5*time.Second)

	// Crear circuit breakers para todos los providers
	for _, p := range registry.GetAll() {
		m.circuitBreakers[p.ID()] = NewCircuitBreaker(5, 30*time.Second)
	}

	return m
}

// Start inicia el health monitor
func (m *Manager) Start(ctx context.Context) {
	m.healthMonitor.Start(ctx)
	logger.Info("❤️ Health monitor iniciado")
}

// Stop detiene todo
func (m *Manager) Stop() {
	m.healthMonitor.Stop()
}

// GetBreaker devuelve el circuit breaker de un provider
func (m *Manager) GetBreaker(providerID string) *CircuitBreaker {
	m.mu.RLock()
	cb, exists := m.circuitBreakers[providerID]
	m.mu.RUnlock()

	if exists {
		return cb
	}

	// Crear si no existe
	m.mu.Lock()
	defer m.mu.Unlock()
	cb = NewCircuitBreaker(5, 30*time.Second)
	m.circuitBreakers[providerID] = cb
	return cb
}

// GetManaged devuelve un ManagedProvider
func (m *Manager) GetManaged(providerID string) (*ManagedProvider, bool) {
	p, exists := m.registry.Get(providerID)
	if !exists {
		return nil, false
	}

	health, _ := m.healthMonitor.Get(providerID)
	cb := m.GetBreaker(providerID)

	return &ManagedProvider{
		Provider:       p,
		Health:         health,
		CircuitBreaker: cb,
	}, true
}

// AvailableProviders devuelve los providers disponibles (healthy + CB closed)
func (m *Manager) AvailableProviders() []Provider {
	var result []Provider

	for _, p := range m.registry.GetAll() {
		// Verificar health
		health, exists := m.healthMonitor.Get(p.ID())
		if exists && (health.Status == HealthUnhealthy || health.Status == HealthDisabled) {
			continue
		}

		// Verificar circuit breaker
		cb := m.GetBreaker(p.ID())
		if !cb.Allow() {
			continue
		}

		result = append(result, p)
	}

	return result
}

// HealthStatus devuelve el estado completo de todos los providers
func (m *Manager) HealthStatus() map[string]interface{} {
	result := make(map[string]interface{})

	for _, p := range m.registry.GetAll() {
		health, _ := m.healthMonitor.Get(p.ID())
		cb := m.GetBreaker(p.ID())

		info := map[string]interface{}{
			"provider_id":     p.ID(),
			"provider_name":   p.Name(),
			"circuit_breaker": cb.Stats(),
		}

		if health != nil {
			info["health"] = map[string]interface{}{
				"status":            string(health.Status),
				"last_check":        health.LastCheck,
				"last_error":        health.LastError,
				"consecutive_fails": health.ConsecutiveFails,
				"avg_latency_ms":    health.AvgLatency.Milliseconds(),
				"last_latency_ms":   health.LastLatency.Milliseconds(),
			}
		}

		result[p.ID()] = info
	}

	return result
}

// CircuitBreakerStatus devuelve el estado de los CB
func (m *Manager) CircuitBreakerStatus() map[string]string {
	result := make(map[string]string)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for id, cb := range m.circuitBreakers {
		result[id] = string(cb.State())
	}
	return result
}

// HealthMonitor devuelve el health monitor
func (m *Manager) HealthMonitor() *HealthMonitor {
	return m.healthMonitor
}
