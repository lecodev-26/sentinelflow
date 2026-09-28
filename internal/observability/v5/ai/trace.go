package aiobs

import (
	"sync"
	"time"
)

type Span struct {
	ID, ParentID, Name, Tenant, Project, Provider, Model string
	Attributes                                           map[string]string
	Start, End                                           time.Time
	Error                                                string
}
type Trace struct {
	ID        string
	Spans     []Span
	StartedAt time.Time
}
type Store struct {
	mu     sync.RWMutex
	traces map[string]Trace
}

func NewStore() *Store { return &Store{traces: map[string]Trace{}} }
func (s *Store) Start(id, tenant, project string) Trace {
	t := Trace{ID: id, StartedAt: time.Now().UTC()}
	s.mu.Lock()
	s.traces[id] = t
	s.mu.Unlock()
	return t
}
func (s *Store) AddSpan(tid string, sp Span) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.traces[tid]; ok {
		if sp.Start.IsZero() {
			sp.Start = time.Now().UTC()
		}
		if sp.End.IsZero() {
			sp.End = time.Now().UTC()
		}
		t.Spans = append(t.Spans, sp)
		s.traces[tid] = t
	}
}
func (s *Store) Get(id string) (Trace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.traces[id]
	return t, ok
}
func Duration(sp Span) time.Duration { return sp.End.Sub(sp.Start) }
