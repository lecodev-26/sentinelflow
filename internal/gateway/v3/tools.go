package v3

import (
	"errors"
	"fmt"
	"sync"
)

type ToolDefinition struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	InputSchema      map[string]interface{} `json:"input_schema,omitempty"`
	RequiresApproval bool                   `json:"requires_approval,omitempty"`
}
type ToolPolicy struct {
	Allowed         []string
	Denied          []string
	RequireApproval []string
}
type ToolDecision struct {
	Name             string `json:"name"`
	Allowed          bool   `json:"allowed"`
	RequiresApproval bool   `json:"requires_approval"`
	Reason           string `json:"reason"`
}
type ToolGateway struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

func NewToolGateway() *ToolGateway { return &ToolGateway{tools: map[string]ToolDefinition{}} }
func (g *ToolGateway) Register(t ToolDefinition) error {
	if t.Name == "" {
		return errors.New("tool name is required")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tools[t.Name] = t
	return nil
}
func (g *ToolGateway) Get(n string) (ToolDefinition, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	t, ok := g.tools[n]
	return t, ok
}
func toolContains(a []string, n string) bool {
	for _, x := range a {
		if x == n {
			return true
		}
	}
	return false
}
func (g *ToolGateway) Evaluate(n string, p ToolPolicy) ToolDecision {
	t, ok := g.Get(n)
	if !ok {
		return ToolDecision{Name: n, Reason: "tool_not_registered"}
	}
	if toolContains(p.Denied, n) {
		return ToolDecision{Name: n, Reason: "tool_denied"}
	}
	if len(p.Allowed) > 0 && !toolContains(p.Allowed, n) {
		return ToolDecision{Name: n, Reason: "tool_not_allowlisted"}
	}
	approval := toolContains(p.RequireApproval, n) || t.RequiresApproval
	return ToolDecision{Name: n, Allowed: true, RequiresApproval: approval, Reason: fmt.Sprintf("tool_allowed:%s", n)}
}
func (g *ToolGateway) Authorize(names []string, p ToolPolicy) ([]ToolDecision, error) {
	out := make([]ToolDecision, 0, len(names))
	for _, n := range names {
		d := g.Evaluate(n, p)
		out = append(out, d)
		if !d.Allowed {
			return out, fmt.Errorf("tool %q denied: %s", n, d.Reason)
		}
	}
	return out, nil
}
