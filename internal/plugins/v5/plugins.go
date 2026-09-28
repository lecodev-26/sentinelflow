package pluginsv5

import (
	"fmt"
	"sync"
)

type Type string

const (
	Provider  Type = "provider"
	Tool      Type = "tool"
	Policy    Type = "policy"
	Evaluator Type = "evaluator"
	Storage   Type = "storage"
)

type Plugin struct {
	ID, Version string
	Type        Type
	Metadata    map[string]string
}
type Registry struct {
	mu    sync.RWMutex
	items map[string]Plugin
}

func NewRegistry() *Registry { return &Registry{items: map[string]Plugin{}} }
func (r *Registry) Register(p Plugin) error {
	if p.ID == "" || p.Version == "" {
		return fmt.Errorf("plugin id/version required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[p.ID] = p
	return nil
}
func (r *Registry) Get(id string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.items[id]
	return p, ok
}
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Plugin, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, p)
	}
	return out
}
