package router

import (
"context"
"fmt"

"github.com/lecodev-26/sentinelflow/internal/provider"
"github.com/lecodev-26/sentinelflow/internal/provider/registry"
)

type Router struct {
registry *registry.Registry
}

func NewRouter(reg *registry.Registry) *Router {
return &Router{registry: reg}
}

func (r *Router) Route(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
providers := r.registry.GetAll()
if len(providers) == 0 {
return nil, fmt.Errorf("no providers available")
}

var lastErr error
for _, p := range providers {
select {
case <-ctx.Done():
return nil, ctx.Err()
default:
}

resp, err := p.Chat(ctx, req)
if err == nil {
return resp, nil
}
lastErr = err
}

return nil, fmt.Errorf("all providers failed: %v", lastErr)
}

func (r *Router) GetProvider(name string) (provider.Provider, bool) {
return r.registry.Get(name)
}

func (r *Router) ListProviders() []string {
return r.registry.List()
}

func (r *Router) HealthCheck() map[string]error {
return r.registry.HealthCheck()
}
