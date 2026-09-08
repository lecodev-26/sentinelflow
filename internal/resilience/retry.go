package resilience

import (
"context"
"math/rand"
"time"
)

// RetryConfig configuración para reintentos
type RetryConfig struct {
MaxAttempts int
InitialBackoff time.Duration
MaxBackoff    time.Duration
Jitter        bool
}

// DefaultRetryConfig devuelve la configuración por defecto
func DefaultRetryConfig() RetryConfig {
return RetryConfig{
MaxAttempts:     3,
InitialBackoff:  100 * time.Millisecond,
MaxBackoff:      2 * time.Second,
Jitter:          true,
}
}

// DoWithRetry ejecuta una función con reintentos
func DoWithRetry(ctx context.Context, fn func() error, config RetryConfig) error {
var lastErr error

for attempt := 0; attempt < config.MaxAttempts; attempt++ {
select {
case <-ctx.Done():
return ctx.Err()
default:
}

// Ejecutar la función
err := fn()
if err == nil {
return nil
}
lastErr = err

// Si es el último intento, no esperar
if attempt == config.MaxAttempts-1 {
break
}

// Calcular backoff
backoff := config.InitialBackoff * time.Duration(attempt+1)
if backoff > config.MaxBackoff {
backoff = config.MaxBackoff
}

// Añadir jitter
if config.Jitter {
jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
backoff = backoff/2 + jitter
}

// Esperar
select {
case <-ctx.Done():
return ctx.Err()
case <-time.After(backoff):
}
}

return lastErr
}

// RetryableFunc es una función que puede reintentarse
type RetryableFunc func() error

// RetryableProvider es un wrapper para proveedores con retry
type RetryableProvider struct {
fn     RetryableFunc
config RetryConfig
}

// NewRetryableProvider crea un nuevo wrapper
func NewRetryableProvider(fn RetryableFunc, config RetryConfig) *RetryableProvider {
return &RetryableProvider{
fn:     fn,
config: config,
}
}

// Execute ejecuta la función con reintentos
func (rp *RetryableProvider) Execute(ctx context.Context) error {
return DoWithRetry(ctx, rp.fn, rp.config)
}
