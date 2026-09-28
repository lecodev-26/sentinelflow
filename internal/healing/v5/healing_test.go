package healingv5

import "testing"

func TestHealingController(t *testing.T) {
	c := NewController()
	x := c.Observe("1", "provider", true, "timeout")
	if x.State != Degraded || x.Attempts != 1 {
		t.Fatal(x)
	}
	if !c.Recover("1") {
		t.Fatal("recover failed")
	}
}
