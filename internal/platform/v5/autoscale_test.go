package platformv5

import "testing"

func TestAutoscaler(t *testing.T) {
	a := Autoscaler{Min: 2, Max: 10, TargetCPU: .5, TargetQueue: 10}
	if n := a.Desired(Capacity{Replicas: 2, CPU: 1, Queue: 30}); n < 6 {
		t.Fatal(n)
	}
}
