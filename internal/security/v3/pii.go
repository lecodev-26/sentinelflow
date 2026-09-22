package v3

import (
"context"
"regexp"
)

var piiPatterns = map[string]*regexp.Regexp{
"email":       regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
"phone":       regexp.MustCompile(`\+?[0-9]{1,3}[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`),
"credit_card": regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
"ssn":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
"ipv4":        regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
}

// PIIScanner detecta PII
type PIIScanner struct{}

// NewPIIScanner crea un PII scanner
func NewPIIScanner() *PIIScanner {
return &PIIScanner{}
}

func (s *PIIScanner) Name() string { return "pii" }

func (s *PIIScanner) Scan(ctx context.Context, text string) []Finding {
var findings []Finding

for name, pattern := range piiPatterns {
matches := pattern.FindAllStringIndex(text, -1)
for _, loc := range matches {
risk := RiskMedium
switch name {
case "credit_card", "ssn":
risk = RiskCritical
case "email", "phone":
risk = RiskMedium
case "ipv4":
risk = RiskLow
}

findings = append(findings, Finding{
Category:    CategoryPII,
Rule:        name,
Match:       text[loc[0]:loc[1]],
Start:       loc[0],
End:         loc[1],
Risk:        risk,
Description: "PII detected: " + name,
})
}
}

return findings
}
