package v3

import (
"time"
)

// Action representa la acción de una regla de política
type Action string

const (
ActionAllow  Action = "allow"
ActionDeny   Action = "deny"
ActionRedact Action = "redact"
ActionWarn   Action = "warn"
ActionRoute  Action = "route"
ActionModify Action = "modify"
)

// Policy es una política versionada
type Policy struct {
ID          string    `json:"id"`
Name        string    `json:"name"`
Description string    `json:"description,omitempty"`
Version     int       `json:"version"`
TenantID    string    `json:"tenant_id,omitempty"` // vacío = global
Priority    int       `json:"priority"`
Enabled     bool      `json:"enabled"`

// Reglas
Deny    []Rule `json:"deny,omitempty"`
Allow   []Rule `json:"allow,omitempty"`
Limits  *Limits `json:"limits,omitempty"`
Routing *RoutingPolicy `json:"routing,omitempty"`
Security *SecurityPolicy `json:"security,omitempty"`

// Metadata
CreatedAt time.Time `json:"created_at"`
CreatedBy string    `json:"created_by,omitempty"`
}

// Rule es una regla condicional
type Rule struct {
Model       string                 `json:"model,omitempty"`
Provider    string                 `json:"provider,omitempty"`
When        map[string]interface{} `json:"when,omitempty"`
Action      Action                 `json:"action"`
Reason      string                 `json:"reason,omitempty"`
}

// Limits define los límites de una política
type Limits struct {
MaxTokens      int `json:"max_tokens,omitempty"`
MaxMessages    int `json:"max_messages,omitempty"`
MaxBodySize    int `json:"max_body_size,omitempty"`
MaxConcurrency int `json:"max_concurrency,omitempty"`
RequestsPerMin int `json:"requests_per_minute,omitempty"`
}

// RoutingPolicy define las preferencias de routing
type RoutingPolicy struct {
AllowedProviders   []string `json:"allowed_providers,omitempty"`
BlockedProviders   []string `json:"blocked_providers,omitempty"`
AllowedModels      []string `json:"allowed_models,omitempty"`
RequiredCapabilities []string `json:"required_capabilities,omitempty"`
MaxCostPer1M       float64  `json:"max_cost_per_1m,omitempty"`
PreferredProvider  string   `json:"preferred_provider,omitempty"`
}

// SecurityPolicy define las reglas de seguridad
type SecurityPolicy struct {
PIIDetection       bool   `json:"pii_detection"`
SecretDetection    bool   `json:"secret_detection"`
PromptInjection    bool   `json:"prompt_injection"`
SSRFProtection     bool   `json:"ssrf_protection"`
PIIDefaultAction   Action `json:"pii_default_action,omitempty"`      // redact o block
SecretDefaultAction Action `json:"secret_default_action,omitempty"`  // block
InjectionDefaultAction Action `json:"injection_default_action,omitempty"` // block
}

// DefaultSecurityPolicy devuelve la política de seguridad por defecto
func DefaultSecurityPolicy() *SecurityPolicy {
return &SecurityPolicy{
PIIDetection:           true,
SecretDetection:        true,
PromptInjection:        true,
SSRFProtection:         true,
PIIDefaultAction:       ActionRedact,
SecretDefaultAction:    ActionDeny,
InjectionDefaultAction: ActionDeny,
}
}

// DefaultLimits devuelve los límites por defecto
func DefaultLimits() *Limits {
return &Limits{
MaxTokens:      128000,
MaxMessages:    100,
MaxBodySize:    1024 * 1024, // 1MB
MaxConcurrency: 100,
RequestsPerMin: 100,
}
}

// NewDefaultPolicy crea una política por defecto
func NewDefaultPolicy(tenantID string) *Policy {
return &Policy{
ID:       "policy_default",
Name:     "Default Policy",
Version:  1,
TenantID: tenantID,
Priority: 100,
Enabled:  true,
Limits:   DefaultLimits(),
Security: DefaultSecurityPolicy(),
CreatedAt: time.Now(),
}
}
