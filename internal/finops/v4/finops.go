package finopsv4

type Action string

const (
	Warn     Action = "warn"
	Block    Action = "block"
	Fallback Action = "fallback_to_cheaper_model"
)

type Forecast struct {
	Spent, DailyAverage, Projected, Budget float64
	DaysRemaining                          int
	OverBudget                             bool
}
type BudgetPolicy struct {
	Limit  float64
	WarnAt []int
	Action Action
}

func ForecastSpend(spent, budget float64, daysElapsed, daysInPeriod int) Forecast {
	if daysElapsed <= 0 {
		return Forecast{Spent: spent, Budget: budget, DaysRemaining: daysInPeriod, Projected: spent, OverBudget: spent > budget}
	}
	avg := spent / float64(daysElapsed)
	remaining := daysInPeriod - daysElapsed
	projected := spent + avg*float64(remaining)
	return Forecast{Spent: spent, Budget: budget, DailyAverage: avg, Projected: projected, DaysRemaining: remaining, OverBudget: budget > 0 && projected > budget}
}
func (p BudgetPolicy) ActionFor(spent float64) Action {
	if p.Limit <= 0 {
		return ""
	}
	ratio := spent / p.Limit
	for _, pct := range p.WarnAt {
		if ratio >= float64(pct)/100 {
			if p.Action != "" {
				return p.Action
			}
			return Warn
		}
	}
	return ""
}
