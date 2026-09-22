package v3

import (
"context"
"regexp"
)

var injectionPatterns = []struct {
name    string
pattern *regexp.Regexp
risk    RiskLevel
}{
{"jailbreak_ignore", regexp.MustCompile(`(?i)ignore\s+(?:all\s+)?(?:previous|prior|above)\s+instructions`), RiskCritical},
{"jailbreak_you_are_now", regexp.MustCompile(`(?i)you\s+are\s+now\s+(?:free|unleashed|jailbroken|DAN)`), RiskCritical},
{"system_override", regexp.MustCompile(`(?i)(?:new|override)\s+system\s+prompt`), RiskCritical},
{"reveal_prompt", regexp.MustCompile(`(?i)(?:show|reveal|output|print)\s+(?:me\s+)?(?:your|the)\s+(?:system\s+)?(?:prompt|instructions)`), RiskHigh},
{"role_play", regexp.MustCompile(`(?i)pretend\s+(?:you\s+are|to\s+be)`), RiskMedium},
{"disregard", regexp.MustCompile(`(?i)disregard\s+(?:all\s+)?(?:previous|prior|safety)\s+(?:rules|guidelines|restrictions)`), RiskHigh},
{"token_theft", regexp.MustCompile(`(?i)(?:what|tell\s+me)\s+(?:is|are)\s+your\s+(?:system\s+)?(?:prompt|instructions|rules)`), RiskHigh},
}

// PromptInjectionScanner detecta intentos de injection
type PromptInjectionScanner struct{}

// NewPromptInjectionScanner crea un scanner
func NewPromptInjectionScanner() *PromptInjectionScanner {
return &PromptInjectionScanner{}
}

func (s *PromptInjectionScanner) Name() string { return "prompt_injection" }

func (s *PromptInjectionScanner) Scan(ctx context.Context, text string) []Finding {
var findings []Finding

for _, p := range injectionPatterns {
matches := p.pattern.FindAllStringIndex(text, -1)
for _, loc := range matches {
findings = append(findings, Finding{
Category:    CategoryPromptInjection,
Rule:        p.name,
Match:       text[loc[0]:loc[1]],
Start:       loc[0],
End:         loc[1],
Risk:        p.risk,
Description: "Prompt injection: " + p.name,
})
}
}

return findings
}
