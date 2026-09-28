package globalv5

import (
	"testing"
	"time"
)

func TestGlobalRouter(t *testing.T) {
	r := NewRouter([]Region{{ID: "us", Latency: 200 * time.Millisecond, Capacity: 1, Healthy: true}, {ID: "eu", Latency: 20 * time.Millisecond, Capacity: 1, Healthy: true}})
	x := r.Select(nil)
	if x[0].ID != "eu" {
		t.Fatal(x)
	}
}
