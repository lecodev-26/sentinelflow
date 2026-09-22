package router

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/provider"
	"github.com/lecodev-26/sentinelflow/internal/provider/registry"
	"github.com/lecodev-26/sentinelflow/internal/resilience"
	"github.com/lecodev-26/sentinelflow/internal/router/strategies"
)

// ResilientRouter envuelve el SmartRouter con circuit breaker por provider
type ResilientRouter struct {
	registry        *registry.Registry
	metrics         *strategies.MetricsStore
	strategy        strategies.RoutingStrategy
	circuitBreakers map[string]*resilience.CircuitBreaker
	cbMu            sync.RWMutex
	retryConfig     resilience.RetryConfig
	cbConfig        CBConfig
}

// CBConfig configuración del circuit breaker
type CBConfig struct {
	MaxFailures    int
	CooldownPeriod time.Duration
}

// DefaultCBConfig devuelve la configuración por defecto
func DefaultCBConfig() CBConfig {
	return CBConfig{
		MaxFailures:    5,
		CooldownPeriod: 30 * time.Second,
	}
}

// NewResilientRouter crea un nuevo router resiliente
func NewResilientRouter(reg *registry.Registry, cbCfg CBConfig) *ResilientRouter {
	composite := strategies.NewCompositeStrategy(
		&strategies.HealthStrategy{},
		&strategies.LatencyStrategy{},
		&strategies.CostStrategy{},
	)

	r := &ResilientRouter{
		registry:        reg,
		metrics:         strategies.NewMetricsStore(),
		strategy:        composite,
		circuitBreakers: make(map[string]*resilience.CircuitBreaker),
		retryConfig:     resilience.DefaultRetryConfig(),
		cbConfig:        cbCfg,
	}

	// Crear circuit breakers para todos los providers registrados
	for _, p := range reg.GetAll() {
		r.circuitBreakers[p.Name()] = resilience.NewCircuitBreaker(
			cbCfg.MaxFailures,
			cbCfg.CooldownPeriod,
		)
	}

	return r
}

// getBreaker devuelve el circuit breaker de un provider
func (r *ResilientRouter) getBreaker(name string) *resilience.CircuitBreaker {
	r.cbMu.RLock()
	cb, exists := r.circuitBreakers[name]
	r.cbMu.RUnlock()

	if exists {
		return cb
	}

	// Crear si no existe
	r.cbMu.Lock()
	defer r.cbMu.Unlock()
	if cb, exists := r.circuitBreakers[name]; exists {
		return cb
	}
	cb = resilience.NewCircuitBreaker(r.cbConfig.MaxFailures, r.cbConfig.CooldownPeriod)
	r.circuitBreakers[name] = cb
	return cb
}

// Route selecciona un provider y ejecuta con resiliencia
func (r *ResilientRouter) Route(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	providers := r.registry.GetAll()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Filtrar providers con circuit breaker abierto
	var availableProviders []provider.Provider
	for _, p := range providers {
		cb := r.getBreaker(p.Name())
		if cb.Allow() {
			availableProviders = append(availableProviders, p)
		} else {
			logger.Warnf("🚫 Circuit breaker OPEN para %s, saltando", p.Name())
		}
	}

	if len(availableProviders) == 0 {
		return nil, fmt.Errorf("all providers have circuit breakers open")
	}

	// Seleccionar mejor provider
	selCtx := &strategies.SelectionContext{
		Request:   req,
		Providers: availableProviders,
		Metrics:   r.metrics,
	}

	selectedName, err := r.strategy.Select(selCtx)
	if err != nil {
		selectedName = availableProviders[0].Name()
	}

	// Intentar todos en orden de preferencia hasta que uno funcione
	orderedProviders := r.orderProviders(availableProviders, selectedName)

	var lastErr error
	for _, p := range orderedProviders {
		cb := r.getBreaker(p.Name())
		if !cb.Allow() {
			continue
		}

		startTime := time.Now()
		logger.Infof("🔄 Intentando: %s", p.Name())

		resp, err := p.Chat(ctx, req)
		latency := time.Since(startTime)

		if err == nil {
			cb.RecordSuccess()
			r.metrics.RecordRequest(p.Name(), latency, true, resp.Usage.TotalTokens, 0)
			logger.Infof("✅ Éxito con %s (%.2fms)", p.Name(), float64(latency.Microseconds())/1000.0)
			return resp, nil
		}

		cb.RecordFailure()
		r.metrics.RecordRequest(p.Name(), latency, false, 0, 0)
		logger.Warnf("❌ Falló %s: %v", p.Name(), err)
		lastErr = err
	}

	return nil, fmt.Errorf("all providers failed: %v", lastErr)
}

// orderProviders ordena los providers poniendo el seleccionado primero
func (r *ResilientRouter) orderProviders(providers []provider.Provider, preferred string) []provider.Provider {
	ordered := make([]provider.Provider, 0, len(providers))
	for _, p := range providers {
		if p.Name() == preferred {
			ordered = append(ordered, p)
		}
	}
	for _, p := range providers {
		if p.Name() != preferred {
			ordered = append(ordered, p)
		}
	}
	return ordered
}

// GetMetrics devuelve las métricas de routing
func (r *ResilientRouter) GetMetrics() map[string]*strategies.ProviderMetrics {
	return r.metrics.GetMetrics()
}

// GetCircuitBreakerStatus devuelve el estado de los circuit breakers
func (r *ResilientRouter) GetCircuitBreakerStatus() map[string]string {
	r.cbMu.RLock()
	defer r.cbMu.RUnlock()

	status := make(map[string]string)
	for name, cb := range r.circuitBreakers {
		status[name] = cb.StateString()
	}
	return status
}
