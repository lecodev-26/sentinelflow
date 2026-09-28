package memoryv5

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

type Scope struct{ TenantID, ProjectID, UserID string }
type Item struct {
	ID                   string
	Scope                Scope
	Text                 string
	Metadata             map[string]string
	CreatedAt, ExpiresAt time.Time
}
type Store struct {
	mu    sync.RWMutex
	items map[string]Item
}

func NewStore() *Store   { return &Store{items: map[string]Item{}} }
func valid(s Scope) bool { return s.TenantID != "" }
func (s *Store) Put(_ context.Context, i Item) error {
	if i.ID == "" || !valid(i.Scope) || i.Text == "" {
		return errors.New("id, tenant and text required")
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[i.ID] = i
	return nil
}
func (s *Store) Get(_ context.Context, scope Scope, id string) (Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i, ok := s.items[id]
	if !ok || i.Scope.TenantID != scope.TenantID || i.Scope.ProjectID != scope.ProjectID || i.Scope.UserID != scope.UserID {
		return Item{}, false
	}
	if !i.ExpiresAt.IsZero() && time.Now().After(i.ExpiresAt) {
		return Item{}, false
	}
	return i, true
}
func (s *Store) Search(_ context.Context, scope Scope, q string, limit int) []Item {
	if limit <= 0 {
		limit = 10
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	type hit struct {
		i     Item
		score int
	}
	var hs []hit
	for _, i := range s.items {
		if i.Scope != scope {
			continue
		}
		if !i.ExpiresAt.IsZero() && time.Now().After(i.ExpiresAt) {
			continue
		}
		score := 0
		for _, w := range tokenize(q) {
			if contains(tokenize(i.Text), w) {
				score++
			}
		}
		if score > 0 {
			hs = append(hs, hit{i, score})
		}
	}
	sort.Slice(hs, func(a, b int) bool { return hs[a].score > hs[b].score })
	out := []Item{}
	for i := 0; i < len(hs) && i < limit; i++ {
		out = append(out, hs[i].i)
	}
	return out
}
func tokenize(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r += 32
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			cur += string(r)
		} else if cur != "" {
			out = append(out, cur)
			cur = ""
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
