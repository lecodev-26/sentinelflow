package routingv4

import (
	routingv3 "github.com/lecodev-26/sentinelflow/internal/routing/v3"
	"testing"
)

func TestFromV3(t *testing.T) {
	d := &routingv3.Decision{Selected: routingv3.Candidate{ProviderID: "p", Model: &routingv3.ModelInfo{ID: "m"}, Region: "eu"}, Reason: "latency", Score: routingv3.Score{Total: .9}}
	r := FromV3(d, "17")
	if r.Provider != "p" || r.Model != "m" || r.Region != "eu" || r.PolicyVersion != "17" {
		t.Fatal(r)
	}
}
