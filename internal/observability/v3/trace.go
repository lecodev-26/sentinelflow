package v3

import (
	"sync"
	"time"
)

// Span representa una etapa del pipeline
type Span struct {
	Name       string                 `json:"name"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	DurationMs int64                  `json:"duration_ms"`
	Status     string                 `json:"status"` // ok, error, skipped
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Trace representa el ciclo completo de una petición
type Trace struct {
	TraceID     string                 `json:"trace_id"`
	RequestID   string                 `json:"request_id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	APIKeyID    string                 `json:"api_key_id,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	StatusCode  int                    `json:"status_code"`
	TotalMs     int64                  `json:"total_ms"`
	Spans       []*Span                `json:"spans"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at"`
	Status      string                 `json:"status"` // ok, error, blocked
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SpanBuilder construye un span
type SpanBuilder struct {
	span *Span
}

// StartSpan inicia un nuevo span
func StartSpan(name string) *SpanBuilder {
	return &SpanBuilder{
		span: &Span{
			Name:       name,
			StartTime:  time.Now(),
			Status:     "ok",
			Attributes: make(map[string]interface{}),
		},
	}
}

// WithAttr añade un atributo
func (b *SpanBuilder) WithAttr(key string, value interface{}) *SpanBuilder {
	b.span.Attributes[key] = value
	return b
}

// WithError marca el span como error
func (b *SpanBuilder) WithError(err string) *SpanBuilder {
	b.span.Status = "error"
	b.span.Attributes["error"] = err
	return b
}

// End finaliza el span y devuelve el Span
func (b *SpanBuilder) End() *Span {
	b.span.EndTime = time.Now()
	b.span.DurationMs = b.span.EndTime.Sub(b.span.StartTime).Milliseconds()
	return b.span
}

// Store almacena traces en memoria con rotación
type Store struct {
	mu        sync.RWMutex
	traces    map[string]*Trace
	order     []string
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

// Start inicia un nuevo trace
func (s *Store) Start(traceID, requestID string) *Trace {
	t := &Trace{
		TraceID:   traceID,
		RequestID: requestID,
		StartedAt: time.Now(),
		Spans:     []*Span{},
		Status:    "ok",
		Metadata:  make(map[string]interface{}),
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

// AddSpan añade un span a un trace
func (s *Store) AddSpan(traceID string, span *Span) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.traces[traceID]
	if !ok {
		return
	}
	t.Spans = append(t.Spans, span)
}

// End finaliza un trace
func (s *Store) End(traceID string, statusCode int, status string, err string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.traces[traceID]
	if !ok {
		return
	}

	t.CompletedAt = time.Now()
	t.TotalMs = t.CompletedAt.Sub(t.StartedAt).Milliseconds()
	t.StatusCode = statusCode
	t.Status = status
	t.Error = err
}

// Get devuelve un trace
func (s *Store) Get(traceID string) (*Trace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.traces[traceID]
	return t, ok
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
		if t, ok := s.traces[s.order[i]]; ok {
			if tenantID == "" || t.TenantID == tenantID {
				result = append(result, t)
			}
		}
	}
	return result
}

// Size devuelve el número de traces
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.traces)
}
