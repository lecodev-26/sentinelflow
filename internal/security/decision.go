package security

import (
"time"
)

// Action representa la acción a tomar
type Action string

const (
ActionAllow  Action = "allow"
ActionRedact Action = "redact"
ActionWarn   Action = "warn"
ActionBlock  Action = "block"
ActionReview Action = "review"
)

// RiskLevel representa el nivel de riesgo
type RiskLevel string

const (
RiskNone     RiskLevel = "none"
RiskLow      RiskLevel = "low"
RiskMedium   RiskLevel = "medium"
RiskHigh     RiskLevel = "high"
RiskCritical RiskLevel = "critical"
)

// Category representa la categoría de un hallazgo
type Category string

const (
CategoryPII             Category = "pii"
CategorySecret          Category = "secret"
CategoryPromptInjection Category = "prompt_injection"
CategorySSRF            Category = "ssrf"
CategoryContentPolicy   Category = "content_policy"
CategoryOversized       Category = "oversized"
)

// Finding representa un hallazgo de seguridad
type Finding struct {
Category    Category   `json:"category"`
Rule        string     `json:"rule"`
Match       string     `json:"match,omitempty"`
Start       int        `json:"start,omitempty"`
End         int        `json:"end,omitempty"`
Risk        RiskLevel  `json:"risk"`
Description string     `json:"description,omitempty"`
Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Decision representa el resultado de evaluar una petición
type Decision struct {
// Acción a tomar
Action Action `json:"action"`

// Riesgo global
Risk RiskLevel `json:"risk"`

// Hallazgos individuales
Findings []Finding `json:"findings"`

// Texto procesado (si se aplicó redacción)
ProcessedText string `json:"-"`

// Motivo resumido
Reason string `json:"reason,omitempty"`

// Timestamp
Timestamp time.Time `json:"timestamp"`

// Metadata adicional
Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewDecision crea una decisión "allow" por defecto
func NewDecision() *Decision {
return &Decision{
Action:    ActionAllow,
Risk:      RiskNone,
Findings:  []Finding{},
Timestamp: time.Now(),
Metadata:  make(map[string]interface{}),
}
}

// AddFinding añade un hallazgo y actualiza el riesgo
func (d *Decision) AddFinding(f Finding) {
d.Findings = append(d.Findings, f)

// Actualizar riesgo al máximo
if riskOrder(f.Risk) > riskOrder(d.Risk) {
d.Risk = f.Risk
}

// Actualizar acción según el riesgo
if riskOrder(f.Risk) >= riskOrder(RiskHigh) {
if riskOrder(RiskHigh) > riskOrder(d.Risk) || d.Action == ActionAllow {
d.Action = ActionBlock
}
} else if f.Risk == RiskMedium && d.Action == ActionAllow {
d.Action = ActionRedact
} else if f.Risk == RiskLow && d.Action == ActionAllow {
d.Action = ActionWarn
}
}

// IsAllowed verifica si la acción permite continuar
func (d *Decision) IsAllowed() bool {
return d.Action == ActionAllow || d.Action == ActionRedact || d.Action == ActionWarn
}

// IsBlocked verifica si la acción bloquea la petición
func (d *Decision) IsBlocked() bool {
return d.Action == ActionBlock
}

// ShouldRedact verifica si hay que redactar
func (d *Decision) ShouldRedact() bool {
return d.Action == ActionRedact && d.ProcessedText != ""
}

// SetReason establece el motivo
func (d *Decision) SetReason(reason string) {
d.Reason = reason
}

// SetMetadata establece metadata
func (d *Decision) SetMetadata(key string, value interface{}) {
if d.Metadata == nil {
d.Metadata = make(map[string]interface{})
}
d.Metadata[key] = value
}

// Merge combina dos decisiones
func (d *Decision) Merge(other *Decision) {
if other == nil {
return
}

d.Findings = append(d.Findings, other.Findings...)

if riskOrder(other.Risk) > riskOrder(d.Risk) {
d.Risk = other.Risk
}

// La acción más restrictiva gana
if actionOrder(other.Action) > actionOrder(d.Action) {
d.Action = other.Action
}

if other.ProcessedText != "" {
d.ProcessedText = other.ProcessedText
}
}

func riskOrder(r RiskLevel) int {
switch r {
case RiskNone:
return 0
case RiskLow:
return 1
case RiskMedium:
return 2
case RiskHigh:
return 3
case RiskCritical:
return 4
}
return 0
}

func actionOrder(a Action) int {
switch a {
case ActionAllow:
return 0
case ActionWarn:
return 1
case ActionRedact:
return 2
case ActionReview:
return 3
case ActionBlock:
return 4
}
return 0
}
