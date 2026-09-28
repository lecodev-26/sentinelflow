package v3

import (
	"testing"
	"time"
)

func TestSLOMonitorEvaluation(t *testing.T) {
	m := NewSLOMonitor()
	m.SetTarget("openai", SLOTarget{Availability: .99, LatencyP95: 200 * time.Millisecond, Window: time.Hour})
	for i := 0; i < 99; i++ {
		m.Record("openai", 100*time.Millisecond, true)
	}
	m.Record("openai", 500*time.Millisecond, false)
	s, ok := m.Evaluate("openai")
	if !ok {
		t.Fatal("missing SLO")
	}
	if s.Requests != 100 || s.Errors != 1 {
		t.Fatalf("unexpected counters: %#v", s)
	}
	if s.P95Latency != 100*time.Millisecond {
		t.Fatalf("unexpected p95: %v", s.P95Latency)
	}
	if s.BurnRate <= 0 || s.ErrorBudgetRemaining <= 0 || !s.Compliant {
		t.Fatalf("unexpected SLO: %#v", s)
	}
}

func TestSLOWindowExpiresSamples(t *testing.T) {
	m := NewSLOMonitor()
	m.SetTarget("p", SLOTarget{Availability: .99, LatencyP95: time.Second, Window: time.Millisecond})
	m.Record("p", time.Second, false)
	time.Sleep(3 * time.Millisecond)
	m.Record("p", time.Millisecond, true)
	s, _ := m.Evaluate("p")
	if s.Requests != 1 || s.Errors != 0 {
		t.Fatalf("expired sample retained: %#v", s)
	}
}
