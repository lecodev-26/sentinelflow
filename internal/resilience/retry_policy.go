package resilience

import (
"math/rand"
"time"
)

// RetryPolicy define cuándo y cómo reintentar
type RetryPolicy struct {
MaxAttempts     int
InitialBackoff  time.Duration
MaxBackoff      time.Duration
Jitter          bool
RetryableErrors []string
}

// DefaultRetryPolicy devuelve la política por defecto
func DefaultRetryPolicy() RetryPolicy {
return RetryPolicy{
MaxAttempts:    3,
InitialBackoff: 100 * time.Millisecond,
MaxBackoff:     5 * time.Second,
Jitter:         true,
RetryableErrors: []string{
"429", "500", "502", "503", "504",
"timeout", "connection refused", "connection reset",
},
}
}

// ShouldRetry verifica si un error es retryable
func (p *RetryPolicy) ShouldRetry(err error) bool {
if err == nil {
return false
}
errStr := err.Error()
for _, retryable := range p.RetryableErrors {
if containsStr(errStr, retryable) {
return true
}
}
return false
}

// GetBackoff calcula el tiempo de espera
func (p *RetryPolicy) GetBackoff(attempt int) time.Duration {
backoff := p.InitialBackoff * time.Duration(attempt+1)
if backoff > p.MaxBackoff {
backoff = p.MaxBackoff
}
if p.Jitter {
jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
backoff = backoff/2 + jitter
}
return backoff
}

func containsStr(s, substr string) bool {
for i := 0; i <= len(s)-len(substr); i++ {
if s[i:i+len(substr)] == substr {
return true
}
}
return false
}
