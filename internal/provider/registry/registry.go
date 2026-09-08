package registry

import (
"context"
"sync"

"github.com/lecodev-26/sentinelflow/internal/provider"
)

type Registry struct {
mu         sync.RWMutex
providers  map[string]provider.Provider
priorities []string
}

func NewRegistry() *Registry {
return &Registry{
providers:  make(map[string]provider.Provider),
priorities: []string{},
}
}

func (r *Registry) Register(p provider.Provider) {
r.mu.Lock()
defer r.mu.Unlock()
r.providers[p.Name()] = p
r.priorities = append(r.priorities, p.Name())
}

func (r *Registry) Get(name string) (provider.Provider, bool) {
r.mu.RLock()
defer r.mu.RUnlock()
p, ok := r.providers[name]
return p, ok
}

func (r *Registry) GetAll() []provider.Provider {
r.mu.RLock()
defer r.mu.RUnlock()
result := make([]provider.Provider, 0, len(r.priorities))
for _, name := range r.priorities {
if p, ok := r.providers[name]; ok {
result = append(result, p)
}
}
return result
}

func (r *Registry) List() []string {
r.mu.RLock()
defer r.mu.RUnlock()
names := make([]string, 0, len(r.providers))
for name := range r.providers {
names = append(names, name)
}
return names
}

func (r *Registry) HealthCheck() map[string]error {
r.mu.RLock()
defer r.mu.RUnlock()
result := make(map[string]error)
for name, p := range r.providers {
result[name] = p.Health(context.Background())
}
return result
}
