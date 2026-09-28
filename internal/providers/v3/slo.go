package v3

import (
	"sort"
	"sync"
	"time"
)

type SLOTarget struct {
	Availability float64       `json:"availability"`
	LatencyP95   time.Duration `json:"latency_p95"`
	Window       time.Duration `json:"window"`
}

type ProviderSLO struct {
	Provider             string        `json:"provider"`
	Target               SLOTarget     `json:"target"`
	Requests             int64         `json:"requests"`
	Errors               int64         `json:"errors"`
	Availability         float64       `json:"availability"`
	P95Latency           time.Duration `json:"p95_latency"`
	ErrorBudgetRemaining float64       `json:"error_budget_remaining"`
	BurnRate             float64       `json:"burn_rate"`
	Compliant            bool          `json:"compliant"`
	WindowStart          time.Time     `json:"window_start"`
}

type sloSample struct {
	at      time.Time
	latency time.Duration
	success bool
}

type SLOMonitor struct {
	mu      sync.RWMutex
	targets map[string]SLOTarget
	samples map[string][]sloSample
}

func NewSLOMonitor() *SLOMonitor {
	return &SLOMonitor{targets: map[string]SLOTarget{}, samples: map[string][]sloSample{}}
}

func (m *SLOMonitor) SetTarget(provider string, target SLOTarget) {
	if target.Availability <= 0 || target.Availability > 1 {
		target.Availability = .99
	}
	if target.LatencyP95 <= 0 {
		target.LatencyP95 = 2 * time.Second
	}
	if target.Window <= 0 {
		target.Window = 30 * time.Minute
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.targets[provider] = target
}

func (m *SLOMonitor) Record(provider string, latency time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := m.targets[provider]
	if target.Window <= 0 {
		target = SLOTarget{Availability: .99, LatencyP95: 2 * time.Second, Window: 30 * time.Minute}
		m.targets[provider] = target
	}
	now := time.Now()
	a := append(m.samples[provider], sloSample{at: now, latency: latency, success: success})
	cutoff := now.Add(-target.Window)
	i := 0
	for i < len(a) && a[i].at.Before(cutoff) {
		i++
	}
	if i > 0 {
		a = append([]sloSample(nil), a[i:]...)
	}
	m.samples[provider] = a
}

func (m *SLOMonitor) Evaluate(provider string) (ProviderSLO, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	target, ok := m.targets[provider]
	if !ok {
		return ProviderSLO{}, false
	}
	a := m.samples[provider]
	out := ProviderSLO{Provider: provider, Target: target, WindowStart: time.Now().Add(-target.Window)}
	if len(a) == 0 {
		out.Availability = 1
		out.ErrorBudgetRemaining = 1
		out.Compliant = true
		return out, true
	}
	var errors int64
	lat := make([]time.Duration, 0, len(a))
	for _, s := range a {
		if !s.success {
			errors++
		}
		lat = append(lat, s.latency)
	}
	out.Requests = int64(len(a))
	out.Errors = errors
	out.Availability = float64(len(a)-int(errors)) / float64(len(a))
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	idx := int(float64(len(lat)-1) * .95)
	out.P95Latency = lat[idx]
	allowedErrors := float64(len(a)) * (1 - target.Availability)
	if allowedErrors <= 0 {
		if errors == 0 {
			out.ErrorBudgetRemaining = 1
		} else {
			out.ErrorBudgetRemaining = 0
		}
	} else {
		out.ErrorBudgetRemaining = 1 - float64(errors)/allowedErrors
		if out.ErrorBudgetRemaining < 0 {
			out.ErrorBudgetRemaining = 0
		}
	}
	if allowedErrors > 0 {
		out.BurnRate = float64(errors) / allowedErrors
	}
	out.Compliant = out.Availability >= target.Availability && out.P95Latency <= target.LatencyP95
	return out, true
}

func (m *SLOMonitor) All() []ProviderSLO {
	m.mu.RLock()
	ids := make([]string, 0, len(m.targets))
	for id := range m.targets {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	sort.Strings(ids)
	out := make([]ProviderSLO, 0, len(ids))
	for _, id := range ids {
		if s, ok := m.Evaluate(id); ok {
			out = append(out, s)
		}
	}
	return out
}
