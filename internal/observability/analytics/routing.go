package analytics

import (
	"sync"
	"time"
)

// RoutingDecision representa una decisión de routing
type RoutingDecision struct {
	RequestID  string                 `json:"request_id"`
	TenantID   string                 `json:"tenant_id,omitempty"`
	Model      string                 `json:"model"`
	Selected   string                 `json:"selected_provider"`
	Candidates []CandidateScore       `json:"candidates"`
	Reason     string                 `json:"reason,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CandidateScore es la puntuación de un candidato
type CandidateScore struct {
	Provider     string  `json:"provider"`
	Total        float64 `json:"total"`
	LatencyScore float64 `json:"latency_score"`
	CostScore    float64 `json:"cost_score"`
	HealthScore  float64 `json:"health_score"`
	QualityScore float64 `json:"quality_score"`
	Selected     bool    `json:"selected"`
}

// RoutingAnalytics analiza las decisiones de routing
type RoutingAnalytics struct {
	mu        sync.RWMutex
	decisions []*RoutingDecision
	maxSize   int

	// Contadores por provider
	selectionCount map[string]int64
}

// NewRoutingAnalytics crea un nuevo analizador
func NewRoutingAnalytics(maxSize int) *RoutingAnalytics {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &RoutingAnalytics{
		decisions:      make([]*RoutingDecision, 0, maxSize),
		maxSize:        maxSize,
		selectionCount: make(map[string]int64),
	}
}

// Record registra una decisión
func (r *RoutingAnalytics) Record(decision *RoutingDecision) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if decision.Timestamp.IsZero() {
		decision.Timestamp = time.Now()
	}

	r.decisions = append(r.decisions, decision)
	if len(r.decisions) > r.maxSize {
		r.decisions = r.decisions[len(r.decisions)-r.maxSize:]
	}

	r.selectionCount[decision.Selected]++
}

// ListRecent lista las decisiones recientes
func (r *RoutingAnalytics) ListRecent(limit int) []*RoutingDecision {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.decisions) {
		limit = len(r.decisions)
	}

	result := make([]*RoutingDecision, 0, limit)
	for i := len(r.decisions) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, r.decisions[i])
	}
	return result
}

// SelectionCounts devuelve el conteo por provider
func (r *RoutingAnalytics) SelectionCounts() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range r.selectionCount {
		result[k] = v
	}
	return result
}

// Size devuelve el número de decisiones
func (r *RoutingAnalytics) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.decisions)
}
