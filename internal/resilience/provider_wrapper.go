package resilience

import (
"context"

"github.com/lecodev-26/sentinelflow/internal/provider"
)

type ResilientProvider struct {
provider       provider.Provider
circuitBreaker *CircuitBreaker
retryConfig    RetryConfig
}

func NewResilientProvider(p provider.Provider, cb *CircuitBreaker, config RetryConfig) *ResilientProvider {
return &ResilientProvider{
provider:       p,
circuitBreaker: cb,
retryConfig:    config,
}
}

func (rp *ResilientProvider) Name() string {
return rp.provider.Name()
}

func (rp *ResilientProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
if !rp.circuitBreaker.Allow() {
return nil, &CircuitOpenError{Provider: rp.provider.Name()}
}

var resp *provider.ChatResponse

err := DoWithRetry(ctx, func() error {
var err error
resp, err = rp.provider.Chat(ctx, req)
return err
}, rp.retryConfig)

if err != nil {
rp.circuitBreaker.RecordFailure()
return nil, err
}

rp.circuitBreaker.RecordSuccess()
return resp, nil
}

func (rp *ResilientProvider) Stream(ctx context.Context, req *provider.ChatRequest) (<-chan provider.Event, error) {
if !rp.circuitBreaker.Allow() {
return nil, &CircuitOpenError{Provider: rp.provider.Name()}
}

events, err := rp.provider.Stream(ctx, req)
if err != nil {
rp.circuitBreaker.RecordFailure()
return nil, err
}

rp.circuitBreaker.RecordSuccess()
return events, nil
}

func (rp *ResilientProvider) Health(ctx context.Context) error {
return rp.provider.Health(ctx)
}

func (rp *ResilientProvider) Models(ctx context.Context) ([]string, error) {
return rp.provider.Models(ctx)
}

type CircuitOpenError struct {
Provider string
}

func (e *CircuitOpenError) Error() string {
return "circuit breaker open for provider: " + e.Provider
}
