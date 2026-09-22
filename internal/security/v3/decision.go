package v3

import (
	"time"
)

// Action es la acción a tomar
type Action string

const (
	ActionAllow  Action = "allow"
	ActionRedact Action = "redact"
	ActionWarn   Action = "warn"
	ActionBlock  Action = "block"
)

// RiskLevel es el nivel de riesgo
type RiskLevel string

const (
	RiskNone     RiskLevel = "none"
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// Category es la categoría de un hallazgo
type Category string

const (
	CategoryPII             Category = "pii"
	CategorySecret          Category = "secret"
	CategoryPromptInjection Category = "prompt_injection"
	CategorySSRF            Category = "ssrf"
	CategoryOversized       Category = "oversized"
)

// Finding representa un hallazgo
type Finding struct {
	Category    Category  `json:"category"`
	Rule        string    `json:"rule"`
	Match       string    `json:"match,omitempty"`
	Start       int       `json:"start,omitempty"`
	End         int       `json:"end,omitempty"`
	Risk        RiskLevel `json:"risk"`
	Description string    `json:"description,omitempty"`
}

// Decision es el resultado del escaneo
type Decision struct {
	Action        Action    `json:"action"`
	Risk          RiskLevel `json:"risk"`
	Findings      []Finding `json:"findings"`
	ProcessedText string    `json:"-"`
	Reason        string    `json:"reason,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewDecision crea una decisión por defecto (allow)
func NewDecision() *Decision {
	return &Decision{
		Action:    ActionAllow,
		Risk:      RiskNone,
		Findings:  []Finding{},
		Timestamp: time.Now(),
	}
}

// AddFinding añade un hallazgo y escala la decisión
func (d *Decision) AddFinding(f Finding) {
	d.Findings = append(d.Findings, f)

	if riskOrder(f.Risk) > riskOrder(d.Risk) {
		d.Risk = f.Risk
	}

	// Escalar acción
	switch f.Risk {
	case RiskCritical:
		d.Action = ActionBlock
	case RiskHigh:
		if d.Action != ActionBlock {
			d.Action = ActionBlock
		}
	case RiskMedium:
		if d.Action == ActionAllow {
			d.Action = ActionRedact
		}
	case RiskLow:
		if d.Action == ActionAllow {
			d.Action = ActionWarn
		}
	}
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
	if actionOrder(other.Action) > actionOrder(d.Action) {
		d.Action = other.Action
	}
	if other.ProcessedText != "" {
		d.ProcessedText = other.ProcessedText
	}
}

// IsBlocked verifica si la acción bloquea
func (d *Decision) IsBlocked() bool {
	return d.Action == ActionBlock
}

// ShouldRedact verifica si hay que redactar
func (d *Decision) ShouldRedact() bool {
	return d.Action == ActionRedact && d.ProcessedText != ""
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
	case ActionBlock:
		return 3
	}
	return 0
}
