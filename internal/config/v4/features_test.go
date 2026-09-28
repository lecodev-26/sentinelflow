package configv4

import "testing"

func TestLoadDefaults(t *testing.T) {
	f := LoadFlags()
	if !f.RoutingV4 || !f.FinOps || !f.EventBus || !f.Enterprise || !f.SecurityCenter {
		t.Fatal("V4 defaults must be enabled")
	}
}
