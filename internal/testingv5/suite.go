package testingv5

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type Case struct {
	ID  string
	Run func(context.Context) error
}
type Report struct {
	Passed, Failed int
	Duration       time.Duration
	Failures       []string
}

func Run(ctx context.Context, cases []Case) Report {
	start := time.Now()
	r := Report{}
	for _, c := range cases {
		if err := c.Run(ctx); err != nil {
			r.Failed++
			r.Failures = append(r.Failures, fmt.Sprintf("%s: %v", c.ID, err))
		} else {
			r.Passed++
		}
	}
	r.Duration = time.Since(start)
	return r
}
func AssertNoFailures(t *testing.T, r Report) {
	t.Helper()
	if r.Failed > 0 {
		t.Fatalf("V5 regression failures: %v", r.Failures)
	}
}
