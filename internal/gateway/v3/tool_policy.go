package v3

import "fmt"

type RequestToolPolicy struct {
	Allowed         []string
	Denied          []string
	RequireApproval []string
}

func (g *ToolGateway) AuthorizeDefinitions(defs []ToolDefinition, p RequestToolPolicy) ([]ToolDecision, error) {
	out := make([]ToolDecision, 0, len(defs))
	for _, d := range defs {
		if err := g.Register(d); err != nil {
			return out, err
		}
		decision := g.Evaluate(d.Name, ToolPolicy{Allowed: p.Allowed, Denied: p.Denied, RequireApproval: p.RequireApproval})
		out = append(out, decision)
		if !decision.Allowed {
			return out, fmt.Errorf("tool %q denied: %s", d.Name, decision.Reason)
		}
	}
	return out, nil
}
