package platformv5

type Capacity struct {
	Replicas  int
	CPU       float64
	Queue     float64
	LatencyMS float64
}
type Autoscaler struct {
	Min, Max               int
	TargetCPU, TargetQueue float64
}

func (a Autoscaler) Desired(c Capacity) int {
	if a.Min <= 0 {
		a.Min = 1
	}
	if a.Max < a.Min {
		a.Max = a.Min
	}
	n := c.Replicas
	if n < a.Min {
		n = a.Min
	}
	if n == 0 {
		n = a.Min
	}
	cpu := 1
	if a.TargetCPU > 0 {
		cpu = int((c.CPU/a.TargetCPU)*float64(n) + .999)
	}
	queue := n
	if a.TargetQueue > 0 {
		queue = int((c.Queue/a.TargetQueue)*float64(n) + .999)
	}
	if cpu > n {
		n = cpu
	}
	if queue > n {
		n = queue
	}
	if c.LatencyMS > 0 && c.LatencyMS > 2000 {
		n++
	}
	if n > a.Max {
		n = a.Max
	}
	return n
}
