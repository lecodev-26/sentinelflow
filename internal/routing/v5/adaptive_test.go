package routingv5

import (
	"testing"
	"time"
)

func TestAdaptiveRankingLearnsOutcome(t *testing.T) {
	e := NewAdaptiveEngine()
	e.Record(Outcome{Provider: "p2", Model: "m", Quality: 1, Success: true, Latency: time.Millisecond, CostUSD: .1, At: time.Now()})
	e.Record(Outcome{Provider: "p1", Model: "m", Quality: .3, Success: false, Latency: time.Second, CostUSD: 2, At: time.Now()})
	r := e.Rank([]Candidate{{Provider: "p1", Model: "m"}, {Provider: "p2", Model: "m"}})
	if r[0].Provider != "p2" {
		t.Fatal(r)
	}
}
