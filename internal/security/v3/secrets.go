package v3

import (
"context"
"regexp"
)

var secretPatterns = map[string]*regexp.Regexp{
"openai_key":   regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
"anthropic_key": regexp.MustCompile(`sk-ant-[a-zA-Z0-9\-_]{20,}`),
"aws_key":      regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
"aws_secret":   regexp.MustCompile(`(?i)aws[_\-]?secret[_\-]?access[_\-]?key['"\s:=]+([a-zA-Z0-9/+=]{40})`),
"github_token": regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
"private_key":  regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
"bearer":       regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-_\.]{20,}`),
"password":     regexp.MustCompile(`(?i)(?:password|passwd|pwd)['"\s:=]+([^\s'"]{8,})`),
}

// SecretScanner detecta secretos
type SecretScanner struct{}

// NewSecretScanner crea un secret scanner
func NewSecretScanner() *SecretScanner {
return &SecretScanner{}
}

func (s *SecretScanner) Name() string { return "secret" }

func (s *SecretScanner) Scan(ctx context.Context, text string) []Finding {
var findings []Finding

for name, pattern := range secretPatterns {
matches := pattern.FindAllStringIndex(text, -1)
for _, loc := range matches {
findings = append(findings, Finding{
Category:    CategorySecret,
Rule:        name,
Match:       text[loc[0]:loc[1]],
Start:       loc[0],
End:         loc[1],
Risk:        RiskCritical,
Description: "Secret detected: " + name,
})
}
}

return findings
}
