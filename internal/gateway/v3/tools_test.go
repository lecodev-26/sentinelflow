package v3

import "testing"

func TestToolGatewayPolicy(t *testing.T) {
	g := NewToolGateway()
	if err := g.Register(ToolDefinition{Name: "search"}); err != nil {
		t.Fatal(err)
	}
	d := g.Evaluate("search", ToolPolicy{Allowed: []string{"search"}})
	if !d.Allowed {
		t.Fatal("expected allowed tool")
	}
	if _, err := g.Authorize([]string{"search", "other"}, ToolPolicy{Allowed: []string{"search"}}); err == nil {
		t.Fatal("expected deny")
	}
}

func TestToolGatewayApproval(t *testing.T) {
	g := NewToolGateway()
	_ = g.Register(ToolDefinition{Name: "dangerous", RequiresApproval: true})
	d := g.Evaluate("dangerous", ToolPolicy{Allowed: []string{"dangerous"}})
	if !d.Allowed || !d.RequiresApproval {
		t.Fatalf("unexpected decision: %#v", d)
	}
}
