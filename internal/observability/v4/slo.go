package observabilityv4

import "time"

type SLO struct {
	TargetAvailability float64
	Window             time.Duration
}
type Status struct {
	Requests, Errors                    int64
	Availability, ErrorBudget, BurnRate float64
	Compliant                           bool
}

func Evaluate(s SLO, requests, errors int64) Status {
	if requests <= 0 {
		return Status{Requests: requests, Errors: errors, Availability: 1, ErrorBudget: 1, Compliant: true}
	}
	availability := float64(requests-errors) / float64(requests)
	allowed := 1 - s.TargetAvailability
	used := 1 - availability
	budget := 1.0
	if allowed > 0 {
		budget = 1 - used/allowed
	}
	burn := 0.0
	if allowed > 0 {
		burn = used / allowed
	}
	return Status{Requests: requests, Errors: errors, Availability: availability, ErrorBudget: budget, BurnRate: burn, Compliant: availability >= s.TargetAvailability}
}
