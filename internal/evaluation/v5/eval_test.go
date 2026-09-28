package evaluationv5

import (
	"context"
	"testing"
)

func TestRunner(t *testing.T) {
	r := Runner{Eval: func(_ context.Context, c Case) (string, error) { return c.Expected, nil }}
	x := r.Run(context.Background(), []Case{{ID: "1", Expected: "hello world"}})
	if !x[0].Passed {
		t.Fatal(x)
	}
}
