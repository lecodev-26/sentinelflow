package router

import (
"context"
"fmt"
"time"

"github.com/lecodev-26/sentinelflow/internal/provider"
"github.com/lecodev-26/sentinelflow/internal/provider/registry"
"github.com/lecodev-26/sentinelflow/internal/router/strategies"
)

type SmartRouter struct {
registry      *registry.Registry
metrics       *strategies.MetricsStore
strategy      strategies.RoutingStrategy
latencyTarget time.Duration
costTarget    float64
}

func NewSmartRouter(reg *registry.Registry) *SmartRouter {
// Crear estrategia compuesta con las estrategias disponibles
composite := strategies.NewCompositeStrategy(
&strategies.HealthStrategy{},
&strategies.LatencyStrategy{},
&strategies.CostStrategy{},
)

return &SmartRouter{
registry:      reg,
metrics:       strategies.NewMetricsStore(),
strategy:      composite,
latencyTarget: 2 * time.Second,
costTarget:    0.0001,
}
}

func (r *SmartRouter) SetStrategy(strategy strategies.RoutingStrategy) {
r.strategy = strategy
}

func (r *SmartRouter) Route(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
providers := r.registry.GetAll()
if len(providers) == 0 {
return nil, fmt.Errorf("no providers available")
}

selCtx := &strategies.SelectionContext{
Request:       req,
Providers:     providers,
Metrics:       r.metrics,
LatencyTarget: r.latencyTarget,
CostTarget:    r.costTarget,
}

selectedProvider, err := r.strategy.Select(selCtx)
if err != nil {
selectedProvider = providers[0].Name()
}

p, exists := r.registry.Get(selectedProvider)
if !exists {
return nil, fmt.Errorf("selected provider not found: %s", selectedProvider)
}

startTime := time.Now()

resp, err := p.Chat(ctx, req)
if err != nil {
r.metrics.RecordRequest(selectedProvider, time.Since(startTime), false, 0, 0)
return nil, err
}

r.metrics.RecordRequest(
selectedProvider,
time.Since(startTime),
true,
resp.Usage.TotalTokens,
r.costTarget*float64(resp.Usage.TotalTokens),
)

return resp, nil
}

func (r *SmartRouter) GetMetrics() map[string]*strategies.ProviderMetrics {
return r.metrics.GetMetrics()
}
