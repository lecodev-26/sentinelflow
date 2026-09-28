package governancev5

import "testing"

func TestGovernanceRule(t *testing.T) {
	r := Rule{Models: []string{"m"}, Regions: []string{"eu"}, MaxRisk: 5}
	if !r.Allows("m", "p", "eu", 4) || r.Allows("m", "p", "us", 4) {
		t.Fatal("bad governance decision")
	}
}
