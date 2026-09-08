package chaos

import (
"math/rand"
"time"
)

// FailureType representa un tipo de fallo
type FailureType string

const (
FailureTimeout    FailureType = "timeout"
FailureError      FailureType = "error"
FailureSlow       FailureType = "slow"
FailurePanic      FailureType = "panic"
FailureDisconnect FailureType = "disconnect"
)

// Injector inyecta fallos en el sistema
type Injector struct {
enabled     bool
failureRate float64
failures    []FailureType
}

// NewInjector crea un nuevo inyector de fallos
func NewInjector() *Injector {
return &Injector{
enabled:     false,
failureRate: 0.0,
failures:    []FailureType{},
}
}

// Enable activa el chaos engineering
func (i *Injector) Enable(rate float64, failures []FailureType) {
i.enabled = true
i.failureRate = rate
i.failures = failures
}

// Disable desactiva el chaos engineering
func (i *Injector) Disable() {
i.enabled = false
}

// ShouldFail determina si debe ocurrir un fallo
func (i *Injector) ShouldFail() bool {
if !i.enabled {
return false
}
return rand.Float64() < i.failureRate
}

// GetFailure devuelve un tipo de fallo aleatorio
func (i *Injector) GetFailure() FailureType {
if len(i.failures) == 0 {
return FailureError
}
return i.failures[rand.Intn(len(i.failures))]
}

// SimulateFailure simula un fallo
func (i *Injector) SimulateFailure(failure FailureType) error {
switch failure {
case FailureTimeout:
time.Sleep(10 * time.Second)
return nil
case FailureError:
return &ChaosError{Message: "chaos injected error"}
case FailureSlow:
time.Sleep(3 * time.Second)
return nil
case FailurePanic:
panic("chaos injected panic")
case FailureDisconnect:
return &ChaosError{Message: "connection reset by peer"}
default:
return nil
}
}

// ChaosError es un error de chaos
type ChaosError struct {
Message string
}

func (e *ChaosError) Error() string {
return e.Message
}
