package platformv5

import (
	"context"
	"testing"
)

func TestPlatformPlan(t *testing.T) {
	p := New()
	x, err := p.Plan(context.Background(), Request{TenantID: "t", ProjectID: "p", Input: "write code and use a tool"})
	if err != nil || x.Intent != "tool_use" || x.Security == "block" {
		t.Fatal(x, err)
	}
}
func TestPlatformRejectsMissingTenant(t *testing.T) {
	_, err := New().Plan(context.Background(), Request{Input: "hello"})
	if err == nil {
		t.Fatal("expected tenant requirement")
	}
}
