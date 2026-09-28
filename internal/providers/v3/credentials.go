package v3

import (
	"errors"
	"sync"
	"time"
)

type Credential struct {
	ID          string
	Secret      string
	ExpiresAt   time.Time
	Active      bool
	Quarantined bool
	RateLimit   int
	Requests    int64
	Failures    int
}

type CredentialPool struct {
	mu       sync.Mutex
	provider string
	items    []*Credential
	next     uint64
}

func NewCredentialPool(provider string) *CredentialPool { return &CredentialPool{provider: provider} }
func (p *CredentialPool) Add(c *Credential) error {
	if c == nil || c.ID == "" || c.Secret == "" {
		return errors.New("credential id and secret are required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := *c
	if cp.ExpiresAt.IsZero() {
		cp.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	if !cp.Active {
		cp.Active = true
	}
	p.items = append(p.items, &cp)
	return nil
}
func (p *CredentialPool) Acquire() (*Credential, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	n := len(p.items)
	if n == 0 {
		return nil, errors.New("credential pool is empty")
	}
	for i := 0; i < n; i++ {
		idx := int((p.next + uint64(i)) % uint64(n))
		c := p.items[idx]
		if !c.Active || c.Quarantined || (!c.ExpiresAt.IsZero() && now.After(c.ExpiresAt)) {
			continue
		}
		if c.RateLimit > 0 && c.Requests >= int64(c.RateLimit) {
			continue
		}
		c.Requests++
		p.next = uint64(idx + 1)
		cp := *c
		return &cp, nil
	}
	return nil, errors.New("no healthy credential available")
}
func (p *CredentialPool) Release(id string, success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.items {
		if c.ID != id {
			continue
		}
		if !success {
			c.Failures++
			if c.Failures >= 3 {
				c.Quarantined = true
			}
		} else if c.Failures > 0 {
			c.Failures--
		}
		return
	}
}
func (p *CredentialPool) Rotate(id string, replacement *Credential) error {
	if replacement == nil || replacement.ID == "" || replacement.Secret == "" {
		return errors.New("replacement credential is required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, c := range p.items {
		if c.ID == id {
			cp := *replacement
			if cp.ExpiresAt.IsZero() {
				cp.ExpiresAt = time.Now().Add(24 * time.Hour)
			}
			cp.Active = true
			p.items[i] = &cp
			return nil
		}
	}
	return errors.New("credential not found")
}
func (p *CredentialPool) Quarantine(id string, quarantined bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.items {
		if c.ID == id {
			c.Quarantined = quarantined
			return true
		}
	}
	return false
}
func (p *CredentialPool) ListMetadata() []map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]map[string]interface{}, 0, len(p.items))
	for _, c := range p.items {
		out = append(out, map[string]interface{}{"id": c.ID, "expires_at": c.ExpiresAt, "active": c.Active, "quarantined": c.Quarantined, "requests": c.Requests, "failures": c.Failures})
	}
	return out
}
