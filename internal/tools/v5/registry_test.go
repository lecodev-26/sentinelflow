package toolsv5

import (
	"context"
	"testing"
	"time"
)

func TestGatewayEnforcesPolicy(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(Spec{ID: "safe", Version: "1", Timeout: time.Second}, func(context.Context, map[string]any) (any, error) { return "ok", nil }); err != nil {
		t.Fatal(err)
	}
	g := NewGateway(r)
	if _, err := g.Call(context.Background(), "safe", nil, Policy{Denied: []string{"safe"}}); err == nil {
		t.Fatal("expected denial")
	}
}
func TestGatewayCallsTool(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(Spec{ID: "x", Version: "1"}, func(context.Context, map[string]any) (any, error) { return 42, nil })
	v, err := NewGateway(r).Call(context.Background(), "x", nil, Policy{})
	if err != nil || v.(int) != 42 {
		t.Fatal(v, err)
	}
}
