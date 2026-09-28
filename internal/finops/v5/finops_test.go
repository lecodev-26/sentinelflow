package finopsv5

import (
	"testing"
	"time"
)

func TestForecast(t *testing.T) {
	l := NewLedger()
	now := time.Now().Add(-24 * time.Hour)
	l.Record(Usage{Tenant: "t", CostUSD: 10, At: now})
	f := l.Forecast("t", now.Add(-time.Hour), 30)
	if f.ProjectedUSD < 250 {
		t.Fatal(f)
	}
}
func TestOptimize(t *testing.T) {
	o := Optimize([]Usage{{Model: "a", CostUSD: 10}}, map[string]float64{"a": 4})
	if len(o) != 1 || o[0].SavingsUSD != 6 {
		t.Fatal(o)
	}
}
