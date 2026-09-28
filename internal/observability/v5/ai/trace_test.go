package aiobs

import (
	"testing"
	"time"
)

func TestTrace(t *testing.T) {
	s := NewStore()
	s.Start("t", "org", "p")
	s.AddSpan("t", Span{ID: "1", Name: "model", Start: time.Now(), End: time.Now().Add(time.Millisecond)})
	tr, ok := s.Get("t")
	if !ok || len(tr.Spans) != 1 {
		t.Fatal(tr)
	}
}
