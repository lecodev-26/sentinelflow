package routingv5

import (
	"sort"
	"sync"
	"time"
)

type Profile struct {
	Provider    string
	Model       string
	Quality     float64
	Latency     time.Duration
	SuccessRate float64
	CostUSD     float64
	Samples     int
	UpdatedAt   time.Time
}
type Outcome struct {
	Provider, Model  string
	Quality, CostUSD float64
	Latency          time.Duration
	Success          bool
	At               time.Time
}
type AdaptiveEngine struct {
	mu       sync.RWMutex
	profiles map[string]Profile
}

func NewAdaptiveEngine() *AdaptiveEngine { return &AdaptiveEngine{profiles: map[string]Profile{}} }
func key(p, m string) string             { return p + "/" + m }
func (e *AdaptiveEngine) Record(o Outcome) {
	e.mu.Lock()
	defer e.mu.Unlock()
	k := key(o.Provider, o.Model)
	p := e.profiles[k]
	p.Provider = o.Provider
	p.Model = o.Model
	p.Samples++
	n := float64(p.Samples)
	p.Quality += (o.Quality - p.Quality) / n
	p.CostUSD += (o.CostUSD - p.CostUSD) / n
	p.Latency += (o.Latency - p.Latency) / time.Duration(p.Samples)
	success := 0.
	if o.Success {
		success = 1
	}
	p.SuccessRate += (success - p.SuccessRate) / n
	p.UpdatedAt = o.At
	if p.Quality == 0 {
		p.Quality = .5
	}
	e.profiles[k] = p
}
func (e *AdaptiveEngine) Profile(provider, model string) (Profile, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.profiles[key(provider, model)]
	return p, ok
}

type Candidate struct {
	Provider, Model string
	BaseScore       float64
	Profile         Profile
}

func (e *AdaptiveEngine) Rank(cs []Candidate) []Candidate {
	out := append([]Candidate(nil), cs...)
	e.mu.RLock()
	defer e.mu.RUnlock()
	for i := range out {
		if p, ok := e.profiles[key(out[i].Provider, out[i].Model)]; ok {
			out[i].Profile = p
			out[i].BaseScore = .35*p.Quality + .30*p.SuccessRate + .20*(1/(1+p.Latency.Seconds())) + .15*(1/(1+p.CostUSD))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].BaseScore > out[j].BaseScore })
	return out
}
