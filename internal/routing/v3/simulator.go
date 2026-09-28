package v3

import (
	regionsv3 "github.com/lecodev-26/sentinelflow/internal/enterprise/v3/regions"
	"time"
)

type SimulationCandidate struct {
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
	Eligible bool     `json:"eligible"`
	Score    *Score   `json:"score,omitempty"`
	Reasons  []string `json:"reasons"`
}

type RoutingSimulation struct {
	Request    Request               `json:"request"`
	Selected   *SimulationCandidate  `json:"selected,omitempty"`
	Candidates []SimulationCandidate `json:"candidates"`
	Filtered   int                   `json:"filtered"`
}

// Simulate evaluates routing filters and scoring without executing a provider.
func (e *Engine) Simulate(req *Request) (*RoutingSimulation, error) {
	available := e.providerMgr.AvailableProviders()
	result := &RoutingSimulation{Request: *req, Candidates: make([]SimulationCandidate, 0, len(available))}
	for _, p := range available {
		c := SimulationCandidate{Provider: p.ID(), Model: req.Model}
		if req.PreferredProvider != "" && p.ID() != req.PreferredProvider {
			c.Reasons = []string{"provider_not_preferred"}
			result.Candidates = append(result.Candidates, c)
			continue
		}
		if contains(req.ExcludedProviders, p.ID()) {
			c.Reasons = []string{"provider_excluded"}
			result.Candidates = append(result.Candidates, c)
			continue
		}
		level := regionsv3.ResidencyLevel(req.Residency)
		if level == "" {
			level = regionsv3.ResidencyGlobal
		}
		if err := e.regionResolver.ValidateProvider(level, p.ID()); err != nil {
			c.Reasons = []string{"residency_not_allowed"}
			result.Candidates = append(result.Candidates, c)
			continue
		}
		m, ok := e.modelRegistry.Get(p.ID(), req.Model)
		if !ok {
			c.Reasons = []string{"model_unavailable"}
			result.Candidates = append(result.Candidates, c)
			continue
		}
		if len(req.RequiredCapabilities) > 0 && !hasAllCapabilities(m, req.RequiredCapabilities) {
			c.Reasons = []string{"missing_capability"}
			result.Candidates = append(result.Candidates, c)
			continue
		}
		if req.MinContextSize > 0 && m.ContextSize < req.MinContextSize {
			c.Reasons = []string{"insufficient_context"}
			result.Candidates = append(result.Candidates, c)
			continue
		}
		if req.MaxCost > 0 && m.InputPer1M+m.OutputPer1M > req.MaxCost {
			c.Reasons = []string{"cost_limit"}
			result.Candidates = append(result.Candidates, c)
			continue
		}
		health := "unknown"
		var latency time.Duration
		if h, ok := e.providerMgr.HealthMonitor().Get(p.ID()); ok {
			health = string(h.Status)
			latency = h.AvgLatency
		}
		cb := e.providerMgr.GetBreaker(p.ID())
		sc := e.scorer.Score(Candidate{ProviderID: p.ID(), ProviderName: p.Name(), Model: m, HealthStatus: health, AvgLatency: latency, CircuitState: string(cb.State())})
		c.Eligible = true
		c.Score = &sc
		c.Reasons = []string{computeReason(sc)}
		result.Candidates = append(result.Candidates, c)
	}
	for i := range result.Candidates {
		c := &result.Candidates[i]
		if c.Eligible && (result.Selected == nil || c.Score.Total > result.Selected.Score.Total) {
			selected := *c
			result.Selected = &selected
		}
	}
	for _, c := range result.Candidates {
		if !c.Eligible {
			result.Filtered++
		}
	}
	return result, nil
}
