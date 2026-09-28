package finopsv5

import (
	"sort"
	"sync"
	"time"
)

type Usage struct {
	Tenant, Project, Model, Provider string
	Requests                         int
	Tokens                           int
	CostUSD                          float64
	At                               time.Time
}
type Ledger struct {
	mu    sync.RWMutex
	items []Usage
}

func NewLedger() *Ledger         { return &Ledger{} }
func (l *Ledger) Record(u Usage) { l.mu.Lock(); defer l.mu.Unlock(); l.items = append(l.items, u) }
func (l *Ledger) Total(tenant string, since time.Time) float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n := 0.
	for _, u := range l.items {
		if (tenant == "" || u.Tenant == tenant) && !u.At.Before(since) {
			n += u.CostUSD
		}
	}
	return n
}

type Forecast struct {
	CurrentUSD   float64
	ProjectedUSD float64
	DailyBurnUSD float64
	Days         int
}

func (l *Ledger) Forecast(tenant string, since time.Time, days int) Forecast {
	if days <= 0 {
		days = 30
	}
	now := time.Now()
	total := l.Total(tenant, since)
	elapsed := now.Sub(since).Hours() / 24
	if elapsed < 1 {
		elapsed = 1
	}
	daily := total / elapsed
	return Forecast{CurrentUSD: total, ProjectedUSD: daily * float64(days), DailyBurnUSD: daily, Days: days}
}

type Opportunity struct {
	Provider, Model                          string
	CurrentCost, AlternativeCost, SavingsUSD float64
	Reason                                   string
}

func Optimize(usages []Usage, alternatives map[string]float64) []Opportunity {
	m := map[string]float64{}
	for _, u := range usages {
		m[u.Model] += u.CostUSD
	}
	var out []Opportunity
	for model, cost := range m {
		if alt, ok := alternatives[model]; ok && alt < cost {
			out = append(out, Opportunity{Model: model, CurrentCost: cost, AlternativeCost: alt, SavingsUSD: cost - alt, Reason: "lower observed equivalent cost"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SavingsUSD > out[j].SavingsUSD })
	return out
}
