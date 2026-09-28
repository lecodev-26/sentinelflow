package finopsv4

import "testing"

func TestForecast(t *testing.T) {
	f := ForecastSpend(50, 100, 10, 30)
	if f.Projected != 150 || !f.OverBudget {
		t.Fatal(f)
	}
}
func TestBudgetAction(t *testing.T) {
	p := BudgetPolicy{Limit: 100, WarnAt: []int{50, 75, 90}, Action: Block}
	if p.ActionFor(80) != Block {
		t.Fatal("expected block")
	}
}
