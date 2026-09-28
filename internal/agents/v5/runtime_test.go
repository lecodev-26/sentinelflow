package agentsv5

import (
	"context"
	"testing"
)

type exec struct{}

func (exec) Execute(_ context.Context, s Step) (any, error) { return s.Tool + "-ok", nil }

type approval struct{ ok bool }

func (a approval) Request(context.Context, Run, Step) (bool, error) { return a.ok, nil }
func TestRuntimeExecutesAndApproves(t *testing.T) {
	r := NewRuntime(exec{}, approval{true})
	run, err := r.Start(context.Background(), &Run{ID: "r1", AgentID: "a1", Steps: []Step{{ID: "s1", Tool: "search", RequiresApproval: true}}})
	if err != nil || run.Status != StatusCompleted || len(run.Results) != 1 {
		t.Fatal(run, err)
	}
}
func TestRuntimeDeniesApproval(t *testing.T) {
	r := NewRuntime(exec{}, approval{false})
	run, err := r.Start(context.Background(), &Run{ID: "r2", AgentID: "a1", Steps: []Step{{ID: "s1", Tool: "delete", RequiresApproval: true}}})
	if err == nil || run.Status != StatusCancelled {
		t.Fatal(run, err)
	}
}
