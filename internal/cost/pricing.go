package cost

// Pricing contiene los precios por modelo
var Pricing = map[string]map[string]Price{
"openai": {
"gpt-3.5-turbo":    {Input: 0.0000015, Output: 0.000002},
"gpt-4":            {Input: 0.00003, Output: 0.00006},
"gpt-4-turbo":      {Input: 0.00001, Output: 0.00003},
},
"anthropic": {
"claude-3":         {Input: 0.000003, Output: 0.000015},
"claude-3-sonnet":  {Input: 0.000003, Output: 0.000015},
"claude-3-opus":    {Input: 0.000015, Output: 0.000075},
},
"local-llama": {
"llama3":           {Input: 0.0, Output: 0.0},
"llama3.1":         {Input: 0.0, Output: 0.0},
},
}

// Price representa el precio por token
type Price struct {
Input  float64 `json:"input"`  // Precio por 1k tokens input
Output float64 `json:"output"` // Precio por 1k tokens output
}

// GetPricing devuelve el precio para un provider y modelo
func GetPricing(provider, model string) (Price, bool) {
if providerPrices, ok := Pricing[provider]; ok {
if price, ok := providerPrices[model]; ok {
return price, true
}
}
return Price{}, false
}

// CalculateCost calcula el coste de una petición
func CalculateCost(provider, model string, inputTokens, outputTokens int) float64 {
price, ok := GetPricing(provider, model)
if !ok {
return 0.0
}

inputCost := price.Input * float64(inputTokens) / 1000.0
outputCost := price.Output * float64(outputTokens) / 1000.0

return inputCost + outputCost
}
