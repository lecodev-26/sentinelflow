package v3

import (
	"sync"
	"time"
)

// CircuitState representa el estado del circuit breaker
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

// CircuitBreaker protege contra fallos repetidos de un provider
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failures         int
	successes        int
	lastFailureTime  time.Time
	maxFailures      int
	cooldownPeriod   time.Duration
	successThreshold int
}

// NewCircuitBreaker crea un nuevo circuit breaker
func NewCircuitBreaker(maxFailures int, cooldown time.Duration) *CircuitBreaker {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		state:            CircuitClosed,
		maxFailures:      maxFailures,
		cooldownPeriod:   cooldown,
		successThreshold: 2,
	}
}

// Allow verifica si se permite una petición
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) > cb.cooldownPeriod {
			cb.state = CircuitHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return true
}

// RecordSuccess registra un éxito
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
		}
	case CircuitClosed:
		cb.failures = 0
	}
}

// RecordFailure registra un fallo
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitOpen
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.failures = cb.maxFailures
	}
}

// State devuelve el estado actual
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats devuelve estadísticas
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return map[string]interface{}{
		"state":            string(cb.state),
		"failures":         cb.failures,
		"successes":        cb.successes,
		"max_failures":     cb.maxFailures,
		"cooldown_seconds": cb.cooldownPeriod.Seconds(),
		"last_failure":     cb.lastFailureTime,
	}
}
