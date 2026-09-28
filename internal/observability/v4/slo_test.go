package observabilityv4

import (
	"testing"
	"time"
)

func TestSLO(t *testing.T) {
	s := Evaluate(SLO{TargetAvailability: .99, Window: 30 * time.Minute}, 1000, 5)
	if !s.Compliant || s.BurnRate <= 0 {
		t.Fatal(s)
	}
}
