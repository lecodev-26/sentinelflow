package learningv5

import (
	"sort"
	"sync"
	"time"
)

type Signal struct {
	Route, Model string
	Reward       float64
	At           time.Time
}
type Loop struct {
	mu      sync.RWMutex
	signals []Signal
}

func New() *Loop                { return &Loop{} }
func (l *Loop) Record(s Signal) { l.mu.Lock(); l.signals = append(l.signals, s); l.mu.Unlock() }

type Aggregate struct {
	Route         string
	Samples       int
	AverageReward float64
}

func (l *Loop) Aggregate() []Aggregate {
	l.mu.RLock()
	defer l.mu.RUnlock()
	m := map[string]*Aggregate{}
	for _, s := range l.signals {
		x := m[s.Route]
		if x == nil {
			x = &Aggregate{Route: s.Route}
			m[s.Route] = x
		}
		x.Samples++
		x.AverageReward += (s.Reward - x.AverageReward) / float64(x.Samples)
	}
	out := make([]Aggregate, 0, len(m))
	for _, x := range m {
		out = append(out, *x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AverageReward > out[j].AverageReward })
	return out
}
