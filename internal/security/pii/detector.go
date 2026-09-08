package pii

import (
"regexp"
"strings"
)

// PIIType representa un tipo de PII
type PIIType string

const (
Email      PIIType = "email"
Phone      PIIType = "phone"
CreditCard PIIType = "credit_card"
SSN        PIIType = "ssn"
IPAddress  PIIType = "ip_address"
URL        PIIType = "url"
Name       PIIType = "name"
Location   PIIType = "location"
)

// PIIMatch representa una coincidencia de PII
type PIIMatch struct {
Type    PIIType `json:"type"`
Value   string  `json:"value"`
Start   int     `json:"start"`
End     int     `json:"end"`
Context string  `json:"context"`
}

// Detector detecta PII en texto
type Detector struct {
patterns map[PIIType]*regexp.Regexp
}

// NewDetector crea un nuevo detector de PII
func NewDetector() *Detector {
return &Detector{
patterns: map[PIIType]*regexp.Regexp{
Email:      regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
Phone:      regexp.MustCompile(`\+?[0-9]{1,3}?[-\s.]?\(?[0-9]{3}\)?[-\s.]?[0-9]{3}[-\s.]?[0-9]{4}`),
CreditCard: regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`),
SSN:        regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
IPAddress:  regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
URL:        regexp.MustCompile(`https?://[^\s]+`),
},
}
}

// Detect detecta PII en un texto
func (d *Detector) Detect(text string) []PIIMatch {
var matches []PIIMatch

for piiType, pattern := range d.patterns {
locations := pattern.FindAllStringIndex(text, -1)
for _, loc := range locations {
matches = append(matches, PIIMatch{
Type:    piiType,
Value:   text[loc[0]:loc[1]],
Start:   loc[0],
End:     loc[1],
Context: getContext(text, loc[0], loc[1]),
})
}
}

return matches
}

// HasPII verifica si un texto contiene PII
func (d *Detector) HasPII(text string) bool {
return len(d.Detect(text)) > 0
}

// Redact redacta PII en un texto
func (d *Detector) Redact(text string) string {
matches := d.Detect(text)
if len(matches) == 0 {
return text
}

// Reemplazar de derecha a izquierda para no afectar índices
result := []rune(text)
for i := len(matches) - 1; i >= 0; i-- {
match := matches[i]
redacted := strings.Repeat("*", len(match.Value))
start := match.Start
end := match.End
if start < len(result) && end <= len(result) {
result = append(result[:start], append([]rune(redacted), result[end:]...)...)
}
}

return string(result)
}

func getContext(text string, start, end int) string {
// Obtener 20 caracteres alrededor
ctxStart := start - 20
if ctxStart < 0 {
ctxStart = 0
}
ctxEnd := end + 20
if ctxEnd > len(text) {
ctxEnd = len(text)
}
return text[ctxStart:ctxEnd]
}
