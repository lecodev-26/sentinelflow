package routingv4

import routingv3 "github.com/lecodev-26/sentinelflow/internal/routing/v3"

type DecisionReason struct {
	Factor string
	Detail string
}
type Candidate struct {
	Provider, Model, Region string
	Score                   float64
	Eligible                bool
	Reasons                 []DecisionReason
}
type RoutingDecision struct {
	Provider, Model, Region, PolicyVersion string
	Score                                  float64
	Reasons                                []DecisionReason
	Candidates                             []Candidate
}

func FromV3(d *routingv3.Decision, policyVersion string) RoutingDecision {
	out := RoutingDecision{PolicyVersion: policyVersion, Provider: d.Selected.ProviderID, Model: d.Selected.Model.ID, Region: d.Selected.Region, Score: d.Score.Total}
	out.Reasons = []DecisionReason{{Factor: "routing", Detail: d.Reason}}
	for _, s := range d.AllCandidates {
		out.Candidates = append(out.Candidates, Candidate{Provider: s.ProviderID, Model: s.ModelID, Score: s.Total, Eligible: true})
	}
	return out
}
