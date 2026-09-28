package promptsv5

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Version struct {
	Number    int
	Template  string
	State     string
	CreatedAt time.Time
}
type Registry struct {
	mu     sync.RWMutex
	items  map[string][]Version
	active map[string]int
}

func NewRegistry() *Registry {
	return &Registry{items: map[string][]Version{}, active: map[string]int{}}
}
func (r *Registry) Create(id, template string) (Version, error) {
	if id == "" || template == "" {
		return Version{}, errors.New("prompt id and template required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	v := Version{Number: len(r.items[id]) + 1, Template: template, State: "draft", CreatedAt: time.Now().UTC()}
	r.items[id] = append(r.items[id], v)
	return v, nil
}
func (r *Registry) Publish(id string, n int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	xs := r.items[id]
	if n <= 0 || n > len(xs) {
		return fmt.Errorf("prompt version not found")
	}
	for i := range xs {
		if xs[i].Number == n {
			xs[i].State = "published"
		} else if xs[i].State == "published" {
			xs[i].State = "archived"
		}
	}
	r.items[id] = xs
	r.active[id] = n
	return nil
}
func (r *Registry) Active(id string) (Version, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.active[id]
	if !ok {
		return Version{}, false
	}
	for _, v := range r.items[id] {
		if v.Number == n {
			return v, true
		}
	}
	return Version{}, false
}
