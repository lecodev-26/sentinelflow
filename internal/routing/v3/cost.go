package v3

// CostEstimate representa el coste previsto de una petición antes de ejecutarla.
type CostEstimate struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	USD          float64 `json:"usd"`
}

func EstimateCost(m *ModelInfo, inputTokens, outputTokens int) CostEstimate {
	if m == nil {
		return CostEstimate{InputTokens: inputTokens, OutputTokens: outputTokens}
	}
	return CostEstimate{InputTokens: inputTokens, OutputTokens: outputTokens, USD: (float64(inputTokens)/1e6)*m.InputPer1M + (float64(outputTokens)/1e6)*m.OutputPer1M}
}

type CostPolicy struct {
	MaxRequestUSD      float64 `json:"max_request_usd"`
	BudgetRemainingUSD float64 `json:"budget_remaining_usd"`
	PreferCheaper      bool    `json:"prefer_cheaper"`
}

func (p CostPolicy) Allows(c CostEstimate) bool {
	if p.MaxRequestUSD > 0 && c.USD > p.MaxRequestUSD {
		return false
	}
	if p.BudgetRemainingUSD > 0 && c.USD > p.BudgetRemainingUSD {
		return false
	}
	return true
}
