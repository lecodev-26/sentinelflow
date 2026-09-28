package learningv5

import (
	"testing"
	"time"
)

func TestLearningAggregate(t *testing.T) {
	l := New()
	l.Record(Signal{Route: "r", Reward: 1, At: time.Now()})
	l.Record(Signal{Route: "r", Reward: 0, At: time.Now()})
	a := l.Aggregate()
	if len(a) != 1 || a[0].Samples != 2 || a[0].AverageReward != .5 {
		t.Fatal(a)
	}
}
