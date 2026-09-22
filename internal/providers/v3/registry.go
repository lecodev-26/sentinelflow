package v3

import (
	"errors"
	"sync"
)

// Registry gestiona los providers registrados
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	order     []string
}

// NewRegistry crea un nuevo registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		order:     make([]string, 0),
	}
}

// Register añade un provider
func (r *Registry) Register(p Provider) error {
	if p == nil {
		return errors.New("cannot register nil provider")
	}
	id := p.ID()
	if id == "" {
		return errors.New("provider must have an ID")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[id]; exists {
		return errors.New("provider already registered: " + id)
	}

	r.providers[id] = p
	r.order = append(r.order, id)
	return nil
}

// Unregister elimina un provider
func (r *Registry) Unregister(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[id]; !exists {
		return false
	}

	delete(r.providers, id)

	// Remover del order
	newOrder := make([]string, 0, len(r.order)-1)
	for _, pid := range r.order {
		if pid != id {
			newOrder = append(newOrder, pid)
		}
	}
	r.order = newOrder

	return true
}

// Get devuelve un provider por ID
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, exists := r.providers[id]
	return p, exists
}

// List devuelve los IDs de todos los providers
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

// GetAll devuelve todos los providers en orden
func (r *Registry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Provider, 0, len(r.order))
	for _, id := range r.order {
		if p, exists := r.providers[id]; exists {
			result = append(result, p)
		}
	}
	return result
}

// Count devuelve el número de providers
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}

// Clear vacía el registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = make(map[string]Provider)
	r.order = make([]string, 0)
}
