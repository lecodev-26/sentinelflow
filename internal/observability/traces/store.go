package traces

import (
"sync"
"time"
)

// Span representa un paso en el trace
type Span struct {
SpanID    string                 `json:"span_id"`
ParentID  string                 `json:"parent_id,omitempty"`
Name      string                 `json:"name"`
StartTime time.Time              `json:"start_time"`
EndTime   time.Time              `json:"end_time"`
Duration  time.Duration          `json:"duration"`
Status    string                 `json:"status"` // ok, error
Attributes map[string]interface{} `json:"attributes,omitempty"`
Events    []SpanEvent            `json:"events,omitempty"`
}

// SpanEvent es un evento dentro de un span
type SpanEvent struct {
Timestamp time.Time              `json:"timestamp"`
Name      string                 `json:"name"`
Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Trace es un trace completo
type Trace struct {
TraceID     string        `json:"trace_id"`
RequestID   string        `json:"request_id"`
TenantID    string        `json:"tenant_id,omitempty"`
UserID      string        `json:"user_id,omitempty"`
StartTime   time.Time     `json:"start_time"`
EndTime     time.Time     `json:"end_time"`
Duration    time.Duration `json:"duration"`
Status      string        `json:"status"` // ok, error, blocked
Provider    string        `json:"provider,omitempty"`
Model       string        `json:"model,omitempty"`
Spans       []*Span       `json:"spans"`
Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Store almacena traces en memoria (con rotación)
type Store struct {
mu       sync.RWMutex
traces   map[string]*Trace
order    []string
maxTraces int
}

// NewStore crea un nuevo store
func NewStore(maxTraces int) *Store {
if maxTraces <= 0 {
maxTraces = 10000
}
return &Store{
traces:    make(map[string]*Trace),
order:     make([]string, 0, maxTraces),
maxTraces: maxTraces,
}
}

// StartTrace inicia un nuevo trace
func (s *Store) StartTrace(traceID, requestID, tenantID, userID string) *Trace {
t := &Trace{
TraceID:   traceID,
RequestID: requestID,
TenantID:  tenantID,
UserID:    userID,
StartTime: time.Now(),
Spans:     []*Span{},
Metadata:  make(map[string]interface{}),
Status:    "ok",
}

s.mu.Lock()
s.traces[traceID] = t
s.order = append(s.order, traceID)

// Rotación
if len(s.order) > s.maxTraces {
oldest := s.order[0]
s.order = s.order[1:]
delete(s.traces, oldest)
}
s.mu.Unlock()

return t
}

// GetTrace devuelve un trace por ID
func (s *Store) GetTrace(traceID string) (*Trace, bool) {
s.mu.RLock()
defer s.mu.RUnlock()
t, ok := s.traces[traceID]
return t, ok
}

// GetTraceByRequest devuelve un trace por request ID
func (s *Store) GetTraceByRequest(requestID string) (*Trace, bool) {
s.mu.RLock()
defer s.mu.RUnlock()
for _, t := range s.traces {
if t.RequestID == requestID {
return t, true
}
}
return nil, false
}

// EndTrace finaliza un trace
func (s *Store) EndTrace(traceID, status, provider, model string) {
s.mu.Lock()
defer s.mu.Unlock()

t, ok := s.traces[traceID]
if !ok {
return
}

t.EndTime = time.Now()
t.Duration = t.EndTime.Sub(t.StartTime)
t.Status = status
t.Provider = provider
t.Model = model
}

// AddSpan añade un span a un trace
func (s *Store) AddSpan(traceID string, span *Span) {
s.mu.Lock()
defer s.mu.Unlock()

t, ok := s.traces[traceID]
if !ok {
return
}

if span.EndTime.IsZero() {
span.EndTime = time.Now()
}
span.Duration = span.EndTime.Sub(span.StartTime)

t.Spans = append(t.Spans, span)
}

// ListTraces lista los traces recientes
func (s *Store) ListTraces(limit int) []*Trace {
s.mu.RLock()
defer s.mu.RUnlock()

if limit <= 0 || limit > len(s.order) {
limit = len(s.order)
}

result := make([]*Trace, 0, limit)
// Más recientes primero
for i := len(s.order) - 1; i >= 0 && len(result) < limit; i-- {
if t, ok := s.traces[s.order[i]]; ok {
result = append(result, t)
}
}
return result
}

// ListByTenant lista traces de un tenant
func (s *Store) ListByTenant(tenantID string, limit int) []*Trace {
s.mu.RLock()
defer s.mu.RUnlock()

if limit <= 0 {
limit = 100
}

result := make([]*Trace, 0, limit)
for i := len(s.order) - 1; i >= 0 && len(result) < limit; i-- {
if t, ok := s.traces[s.order[i]]; ok && t.TenantID == tenantID {
result = append(result, t)
}
}
return result
}

// Size devuelve el número de traces almacenados
func (s *Store) Size() int {
s.mu.RLock()
defer s.mu.RUnlock()
return len(s.traces)
}

// Clear vacía el store
func (s *Store) Clear() {
s.mu.Lock()
defer s.mu.Unlock()
s.traces = make(map[string]*Trace)
s.order = []string{}
}
