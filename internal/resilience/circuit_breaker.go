package resilience

import (
"sync"
"time"
)

// State representa el estado del circuit breaker
type State int

const (
StateClosed State = iota
StateOpen
StateHalfOpen
)

// CircuitBreaker protege contra fallos repetidos
type CircuitBreaker struct {
mu              sync.RWMutex
state           State
failures        int
lastFailureTime time.Time

// Configuración
maxFailures    int           // Fallos antes de abrir
cooldownPeriod time.Duration // Tiempo en open antes de half-open
successes      int           // Éxitos en half-open para cerrar
maxSuccesses   int           // Éxitos necesarios para cerrar
}

// NewCircuitBreaker crea un nuevo circuit breaker
func NewCircuitBreaker(maxFailures int, cooldownPeriod time.Duration) *CircuitBreaker {
return &CircuitBreaker{
state:          StateClosed,
maxFailures:    maxFailures,
cooldownPeriod: cooldownPeriod,
maxSuccesses:   2,
successes:      0,
}
}

// Allow verifica si se permite una petición
func (cb *CircuitBreaker) Allow() bool {
cb.mu.Lock()
defer cb.mu.Unlock()

switch cb.state {
case StateClosed:
return true
case StateOpen:
if time.Since(cb.lastFailureTime) > cb.cooldownPeriod {
cb.state = StateHalfOpen
cb.successes = 0
return true
}
return false
case StateHalfOpen:
return true
default:
return true
}
}

// RecordSuccess registra una petición exitosa
func (cb *CircuitBreaker) RecordSuccess() {
cb.mu.Lock()
defer cb.mu.Unlock()

switch cb.state {
case StateHalfOpen:
cb.successes++
if cb.successes >= cb.maxSuccesses {
cb.state = StateClosed
cb.failures = 0
cb.successes = 0
}
case StateClosed:
cb.failures = 0
}
}

// RecordFailure registra una petición fallida
func (cb *CircuitBreaker) RecordFailure() {
cb.mu.Lock()
defer cb.mu.Unlock()

switch cb.state {
case StateClosed:
cb.failures++
cb.lastFailureTime = time.Now()
if cb.failures >= cb.maxFailures {
cb.state = StateOpen
}
case StateHalfOpen:
cb.state = StateOpen
cb.lastFailureTime = time.Now()
cb.failures = cb.maxFailures
}
}

// State devuelve el estado actual
func (cb *CircuitBreaker) State() State {
cb.mu.RLock()
defer cb.mu.RUnlock()
return cb.state
}

// StateString devuelve el estado como string
func (cb *CircuitBreaker) StateString() string {
switch cb.State() {
case StateClosed:
return "closed"
case StateOpen:
return "open"
case StateHalfOpen:
return "half-open"
default:
return "unknown"
}
}
