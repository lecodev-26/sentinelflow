package events

import (
	"testing"
	"time"
)

func TestComputeBackoff_NoJitter_Exponential(t *testing.T) {
	o := &Outbox{
		baseBackoff:    2 * time.Second,
		maxBackoff:     60 * time.Second,
		jitterFraction: 0,
	}
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{0, 2 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 32 * time.Second},
		{6, 60 * time.Second},
		{99, 60 * time.Second},
	}
	for _, c := range cases {
		got := o.computeBackoff(c.attempts)
		if got != c.want {
			t.Errorf("attempts=%d: got %v, want %v", c.attempts, got, c.want)
		}
	}
}

func TestComputeBackoff_WithJitter_StaysInBounds(t *testing.T) {
	o := &Outbox{
		baseBackoff:    2 * time.Second,
		maxBackoff:     60 * time.Second,
		jitterFraction: 0.2,
	}
	for i := 0; i < 1000; i++ {
		got := o.computeBackoff(1)
		if got < 1600*time.Millisecond || got > 2400*time.Millisecond {
			t.Fatalf("attempts=1 jitter out of bounds: %v", got)
		}
	}
	for i := 0; i < 1000; i++ {
		got := o.computeBackoff(10)
		if got < 48*time.Second || got > 72*time.Second {
			t.Fatalf("attempts=10 jitter out of bounds: %v", got)
		}
	}
}

func TestComputeBackoff_JitterProducesVariety(t *testing.T) {
	o := &Outbox{
		baseBackoff:    10 * time.Second,
		maxBackoff:     60 * time.Second,
		jitterFraction: 0.2,
	}
	seen := make(map[time.Duration]struct{})
	for i := 0; i < 100; i++ {
		seen[o.computeBackoff(1)] = struct{}{}
	}
	if len(seen) < 50 {
		t.Errorf("jitter no produce variedad suficiente: %d valores unicos en 100 intentos", len(seen))
	}
}
