package v3

import "testing"

func TestEstimateCostAndPolicy(t *testing.T) {
	m := &ModelInfo{InputPer1M: 10, OutputPer1M: 30}
	c := EstimateCost(m, 100000, 50000)
	if c.USD != 2.5 {
		t.Fatalf("unexpected cost: %v", c.USD)
	}
	if !(CostPolicy{MaxRequestUSD: 3}).Allows(c) {
		t.Fatal("cost should be allowed")
	}
	if (CostPolicy{MaxRequestUSD: 2}).Allows(c) {
		t.Fatal("cost should be blocked")
	}
}
