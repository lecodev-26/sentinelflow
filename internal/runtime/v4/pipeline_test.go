package runtimev4

import (
	"context"
	"testing"
)

func TestPipelineOrderAndDecision(t *testing.T) {
	var got []string
	p := New(StageFunc{"auth", func(c *Context) error { got = append(got, "auth"); c.Decision.Allowed = true; return nil }}, StageFunc{"route", func(c *Context) error { got = append(got, "route"); c.Decision.Provider = "openai"; return nil }})
	r, e := p.Run(&Context{Context: context.Background()})
	if e != nil || r.Decision.Provider != "openai" {
		t.Fatalf("%v %#v", e, r)
	}
	if len(got) != 2 || got[0] != "auth" || got[1] != "route" {
		t.Fatal(got)
	}
}
