package testingv5

import (
	"context"
	"errors"
	"testing"
)

func TestSuite(t *testing.T) {
	r := Run(context.Background(), []Case{{ID: "ok", Run: func(context.Context) error { return nil }}, {ID: "bad", Run: func(context.Context) error { return errors.New("boom") }}})
	if r.Passed != 1 || r.Failed != 1 {
		t.Fatal(r)
	}
}
