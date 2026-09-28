package core

import "strings"

type Modality string

const (
	ModalityText     Modality = "text"
	ModalityImage    Modality = "image"
	ModalityAudio    Modality = "audio"
	ModalityVideo    Modality = "video"
	ModalityDocument Modality = "document"
)

type Intent string

const (
	IntentChat       Intent = "chat"
	IntentCode       Intent = "code"
	IntentReasoning  Intent = "reasoning"
	IntentExtraction Intent = "extraction"
	IntentToolUse    Intent = "tool_use"
	IntentGeneration Intent = "generation"
)

type Requirements struct {
	Intent                Intent
	Modalities            []Modality
	Complexity            float64
	MinContext            int
	NeedsTools            bool
	NeedsStreaming        bool
	NeedsStructuredOutput bool
	MaxLatencyMS          int
	MaxCostUSD            float64
	Residency             string
	Sensitive             bool
}
type Analysis struct {
	Requirements Requirements
	Signals      []string
}
type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }
func (a *Analyzer) Analyze(input string, toolCount int) Analysis {
	s := strings.ToLower(strings.TrimSpace(input))
	r := Requirements{Intent: IntentChat, Modalities: []Modality{ModalityText}, Complexity: .2}
	if strings.Contains(s, "code") || strings.Contains(s, "golang") || strings.Contains(s, "python") {
		r.Intent = IntentCode
		r.Complexity = .7
	}
	if strings.Contains(s, "reason") || strings.Contains(s, "analyse") || strings.Contains(s, "analyze") {
		r.Intent = IntentReasoning
		r.Complexity = .85
	}
	if strings.Contains(s, "extract") || strings.Contains(s, "json") {
		r.Intent = IntentExtraction
		r.NeedsStructuredOutput = true
		r.Complexity = .45
	}
	if toolCount > 0 || strings.Contains(s, "tool") || strings.Contains(s, "function") {
		r.Intent = IntentToolUse
		r.NeedsTools = true
		r.Complexity = .7
	}
	if len(s) > 12000 {
		r.MinContext = 16000
		r.Complexity += .1
	} else if len(s) > 6000 {
		r.MinContext = 8192
	}
	sensitive := []string{"password", "secret", "token", "api key", "credit card", "ssn"}
	for _, x := range sensitive {
		if strings.Contains(s, x) {
			r.Sensitive = true
			break
		}
	}
	return Analysis{Requirements: r, Signals: []string{string(r.Intent)}}
}
