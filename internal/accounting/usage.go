package accounting

import (
	"time"
)

// UsageEvent representa el uso completo de una petición
type UsageEvent struct {
	// Identificadores
	RequestID string `json:"request_id"`
	TraceID   string `json:"trace_id,omitempty"`

	// Tenant
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	APIKeyID  string `json:"api_key_id,omitempty"`

	// Provider
	Provider string `json:"provider"`
	Model    string `json:"model"`

	// Tokens
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	CachedTokens int `json:"cached_tokens,omitempty"`

	// Timing
	Latency time.Duration `json:"latency"`
	TTFT    time.Duration `json:"ttft,omitempty"`

	// Coste
	CostUSD       float64 `json:"cost_usd"`
	InputCostUSD  float64 `json:"input_cost_usd"`
	OutputCostUSD float64 `json:"output_cost_usd"`

	// Estado
	Status     string `json:"status"` // success, error, cached
	CacheHit   bool   `json:"cache_hit"`
	CacheLayer string `json:"cache_layer,omitempty"`
	Fallbacks  int    `json:"fallbacks,omitempty"`
	Retries    int    `json:"retries,omitempty"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewUsageEvent crea un nuevo evento
func NewUsageEvent(requestID, tenantID, provider, model string) *UsageEvent {
	return &UsageEvent{
		RequestID: requestID,
		TenantID:  tenantID,
		Provider:  provider,
		Model:     model,
		Timestamp: time.Now(),
		Status:    "success",
	}
}

// CalculateTotals recalcula totales
func (e *UsageEvent) CalculateTotals() {
	e.TotalTokens = e.InputTokens + e.OutputTokens
	e.CostUSD = e.InputCostUSD + e.OutputCostUSD
}

// IsSuccess verifica si fue exitoso
func (e *UsageEvent) IsSuccess() bool {
	return e.Status == "success" || e.Status == "cached"
}

// IsCached verifica si fue cache hit
func (e *UsageEvent) IsCached() bool {
	return e.CacheHit
}
