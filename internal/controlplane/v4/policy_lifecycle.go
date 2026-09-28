package controlplanev4

import (
	"fmt"
	"sync"
	"time"
)

type State string

const (
	Draft      State = "draft"
	Validated  State = "validated"
	Published  State = "published"
	RolledBack State = "rolled_back"
)

type Policy struct {
	ID            string
	Version       int
	State         State
	CanaryPercent int
	CreatedAt     time.Time
	Rules         map[string]any
}
type PolicyManager struct {
	mu       sync.RWMutex
	policies map[string][]Policy
	active   map[string]Policy
}

func NewPolicyManager() *PolicyManager {
	return &PolicyManager{policies: map[string][]Policy{}, active: map[string]Policy{}}
}
func (m *PolicyManager) Create(id string, rules map[string]any) (Policy, error) {
	if id == "" {
		return Policy{}, fmt.Errorf("policy id required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p := Policy{ID: id, Version: len(m.policies[id]) + 1, State: Draft, CreatedAt: time.Now().UTC(), Rules: rules}
	m.policies[id] = append(m.policies[id], p)
	return p, nil
}
func (m *PolicyManager) Validate(id string, v int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.find(id, v)
	if !ok {
		return fmt.Errorf("policy not found")
	}
	if p.Rules == nil {
		return fmt.Errorf("rules required")
	}
	p.State = Validated
	m.replace(p)
	return nil
}
func (m *PolicyManager) Publish(id string, v, canary int) error {
	if canary != 10 && canary != 50 && canary != 100 {
		return fmt.Errorf("canary must be 10, 50 or 100")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.find(id, v)
	if !ok || p.State != Validated {
		return fmt.Errorf("policy must be validated before publish")
	}
	p.State = Published
	p.CanaryPercent = canary
	m.replace(p)
	if canary == 100 {
		m.active[id] = p
	}
	return nil
}
func (m *PolicyManager) Rollback(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.active[id]
	if !ok {
		return fmt.Errorf("no active policy")
	}
	p.State = RolledBack
	m.replace(p)
	delete(m.active, id)
	return nil
}
func (m *PolicyManager) Active(id string) (Policy, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.active[id]
	return p, ok
}
func (m *PolicyManager) find(id string, v int) (Policy, bool) {
	for _, p := range m.policies[id] {
		if p.Version == v {
			return p, true
		}
	}
	return Policy{}, false
}
func (m *PolicyManager) replace(p Policy) {
	for i := range m.policies[p.ID] {
		if m.policies[p.ID][i].Version == p.Version {
			m.policies[p.ID][i] = p
			return
		}
	}
}
