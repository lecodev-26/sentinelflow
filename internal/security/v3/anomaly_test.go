package v3

import (
	"testing"
	"time"
)

func TestAnomalyDetectorBlocks(t *testing.T) {
	d := NewAnomalyDetector(time.Minute, 4, 10)
	now := time.Now()
	if d.Record("t", "k", 200, false, now) != nil {
		t.Fatal("unexpected anomaly")
	}
	if d.Record("t", "k", 400, true, now.Add(time.Second)) == nil {
		t.Fatal("expected anomaly")
	}
}
