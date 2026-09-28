package toolsv5

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

type Spec struct {
	ID            string
	Version       string
	Description   string
	Risk          Risk
	Permissions   []string
	Timeout       time.Duration
	MaxInputBytes int
}
type Handler func(context.Context, map[string]any) (any, error)
type Registry struct {
	mu       sync.RWMutex
	specs    map[string]Spec
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{specs: map[string]Spec{}, handlers: map[string]Handler{}}
}
func (r *Registry) Register(s Spec, h Handler) error {
	if s.ID == "" || h == nil {
		return errors.New("tool id and handler required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.specs[s.ID]; ok {
		return fmt.Errorf("tool %s already registered", s.ID)
	}
	r.specs[s.ID] = s
	r.handlers[s.ID] = h
	return nil
}
func (r *Registry) Get(id string) (Spec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.specs[id]
	return s, ok
}
func (r *Registry) List() []Spec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Spec, 0, len(r.specs))
	for _, s := range r.specs {
		out = append(out, s)
	}
	return out
}

type Policy struct{ Allowed, Denied, Approval []string }

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
func (p Policy) AllowedTool(id string) bool {
	if contains(p.Denied, id) {
		return false
	}
	if len(p.Allowed) == 0 {
		return true
	}
	return contains(p.Allowed, id)
}
func (p Policy) RequiresApproval(id string) bool { return contains(p.Approval, id) }

type Gateway struct{ registry *Registry }

func NewGateway(r *Registry) *Gateway { return &Gateway{registry: r} }
func (g *Gateway) Call(ctx context.Context, id string, input map[string]any, p Policy) (any, error) {
	if !p.AllowedTool(id) {
		return nil, fmt.Errorf("tool %s denied", id)
	}
	s, ok := g.registry.Get(id)
	if !ok {
		return nil, fmt.Errorf("tool %s not found", id)
	}
	if s.MaxInputBytes > 0 {
		if len(fmt.Sprint(input)) > s.MaxInputBytes {
			return nil, fmt.Errorf("tool input exceeds limit")
		}
	}
	if s.Timeout <= 0 {
		s.Timeout = 30 * time.Second
	}
	c, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	g.registry.mu.RLock()
	h := g.registry.handlers[id]
	g.registry.mu.RUnlock()
	return h(c, input)
}
