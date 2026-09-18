package security

import (
"context"
"strings"

"github.com/lecodev-26/sentinelflow/internal/security/pii"
"github.com/lecodev-26/sentinelflow/internal/security/prompt"
)

type Scanner interface {
Name() string
Scan(ctx context.Context, text string) []Finding
}

type Pipeline struct {
scanners []Scanner
}

// NewPipeline crea un pipeline con los scanners por defecto
func NewPipeline() *Pipeline {
return &Pipeline{
scanners: []Scanner{
NewPIIScanner(),
NewSecretScanner(),
NewPromptInjectionScanner(),
NewSSRFScanner(),
},
}
}

func (p *Pipeline) Register(s Scanner) {
p.scanners = append(p.scanners, s)
}

func (p *Pipeline) Evaluate(ctx context.Context, text string) *Decision {
decision := NewDecision()

for _, scanner := range p.scanners {
findings := scanner.Scan(ctx, text)
for _, f := range findings {
decision.AddFinding(f)
}
}

if decision.Action == ActionRedact {
decision.ProcessedText = redactText(text, decision.Findings)
}

return decision
}

func redactText(text string, findings []Finding) string {
type indexed struct {
start, end int
}
var indexes []indexed
for _, f := range findings {
if f.Start < f.End && f.End <= len(text) {
indexes = append(indexes, indexed{f.Start, f.End})
}
}

for i := 0; i < len(indexes); i++ {
for j := i + 1; j < len(indexes); j++ {
if indexes[j].start > indexes[i].start {
indexes[i], indexes[j] = indexes[j], indexes[i]
}
}
}

result := text
for _, idx := range indexes {
redacted := strings.Repeat("*", idx.end-idx.start)
result = result[:idx.start] + redacted + result[idx.end:]
}

return result
}

// === PII SCANNER ===

type PIIScanner struct {
detector *pii.Detector
}

func NewPIIScanner() *PIIScanner {
return &PIIScanner{detector: pii.NewDetector()}
}

func (s *PIIScanner) Name() string { return "pii" }

func (s *PIIScanner) Scan(ctx context.Context, text string) []Finding {
matches := s.detector.Detect(text)
findings := make([]Finding, 0, len(matches))
for _, m := range matches {
risk := RiskMedium
switch m.Type {
case pii.CreditCard, pii.SSN:
risk = RiskCritical
case pii.Email, pii.Phone:
risk = RiskMedium
default:
risk = RiskLow
}
findings = append(findings, Finding{
Category:    CategoryPII,
Rule:        string(m.Type),
Match:       m.Value,
Start:       m.Start,
End:         m.End,
Risk:        risk,
Description: "PII detected: " + string(m.Type),
})
}
return findings
}

// === SECRET SCANNER ===

type SecretScanner struct {
detector *pii.SecretDetector
}

func NewSecretScanner() *SecretScanner {
return &SecretScanner{detector: pii.NewSecretDetector()}
}

func (s *SecretScanner) Name() string { return "secret" }

func (s *SecretScanner) Scan(ctx context.Context, text string) []Finding {
matches := s.detector.Detect(text)
findings := make([]Finding, 0, len(matches))
for _, m := range matches {
findings = append(findings, Finding{
Category:    CategorySecret,
Rule:        string(m.Type),
Match:       m.Value,
Start:       m.Start,
End:         m.End,
Risk:        RiskCritical,
Description: "Secret detected: " + string(m.Type),
})
}
return findings
}

// === PROMPT INJECTION SCANNER ===

type PromptInjectionScanner struct {
detector *prompt.Detector
}

func NewPromptInjectionScanner() *PromptInjectionScanner {
return &PromptInjectionScanner{detector: prompt.NewDetector()}
}

func (s *PromptInjectionScanner) Name() string { return "prompt_injection" }

func (s *PromptInjectionScanner) Scan(ctx context.Context, text string) []Finding {
matches := s.detector.Detect(text)
findings := make([]Finding, 0, len(matches))
for _, m := range matches {
risk := RiskHigh
if m.Type.GetSeverity() == "critical" {
risk = RiskCritical
}
findings = append(findings, Finding{
Category:    CategoryPromptInjection,
Rule:        string(m.Type),
Match:       m.Pattern,
Start:       m.Start,
End:         m.End,
Risk:        risk,
Description: "Prompt injection: " + string(m.Type),
})
}
return findings
}
