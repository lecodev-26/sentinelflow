package evaluationv5

import (
	"context"
	"strings"
)

type Case struct{ ID, Input, Expected string }
type Result struct {
	CaseID string
	Output string
	Score  float64
	Passed bool
}
type Evaluator func(context.Context, Case) (string, error)
type Runner struct{ Eval Evaluator }

func (r Runner) Run(ctx context.Context, cases []Case) []Result {
	out := make([]Result, 0, len(cases))
	for _, c := range cases {
		o, err := r.Eval(ctx, c)
		score := 0.
		if err == nil {
			score = similarity(o, c.Expected)
		}
		out = append(out, Result{CaseID: c.ID, Output: o, Score: score, Passed: score >= .8})
	}
	return out
}
func similarity(a, b string) float64 {
	aa := strings.Fields(strings.ToLower(a))
	bb := strings.Fields(strings.ToLower(b))
	if len(bb) == 0 {
		return 0
	}
	m := map[string]bool{}
	for _, x := range aa {
		m[x] = true
	}
	n := 0
	for _, x := range bb {
		if m[x] {
			n++
		}
	}
	return float64(n) / float64(len(bb))
}
