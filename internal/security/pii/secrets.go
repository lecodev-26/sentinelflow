package pii

import (
"regexp"
)

// SecretType representa un tipo de secreto
type SecretType string

const (
APIKey      SecretType = "api_key"
AWSKey      SecretType = "aws_key"
PrivateKey  SecretType = "private_key"
Password    SecretType = "password"
Token       SecretType = "token"
BearerToken SecretType = "bearer_token"
)

// SecretMatch representa una coincidencia de secreto
type SecretMatch struct {
Type   SecretType `json:"type"`
Value  string     `json:"value"`
Start  int        `json:"start"`
End    int        `json:"end"`
}

// SecretDetector detecta secretos en texto
type SecretDetector struct {
patterns map[SecretType]*regexp.Regexp
}

// NewSecretDetector crea un nuevo detector de secretos
func NewSecretDetector() *SecretDetector {
return &SecretDetector{
patterns: map[SecretType]*regexp.Regexp{
APIKey:      regexp.MustCompile(`(?i)(?:api[_-]?key|apikey)[\s:=]+['"]?([a-zA-Z0-9_\-]{20,})['"]?`),
AWSKey:      regexp.MustCompile(`(?i)(AKIA|ASIA)[A-Z0-9]{16,}`),
PrivateKey:  regexp.MustCompile(`-----BEGIN (?:RSA|DSA|EC|OPENSSH) PRIVATE KEY-----`),
Password:    regexp.MustCompile(`(?i)(?:password|passwd|pwd)[\s:=]+['"]?([^\s'"]{6,})['"]?`),
BearerToken: regexp.MustCompile(`(?i)bearer[\s]+([a-zA-Z0-9_\-\.]+)`),
},
}
}

// Detect detecta secretos en un texto
func (sd *SecretDetector) Detect(text string) []SecretMatch {
var matches []SecretMatch

for secretType, pattern := range sd.patterns {
locations := pattern.FindAllStringIndex(text, -1)
for _, loc := range locations {
matches = append(matches, SecretMatch{
Type:   secretType,
Value:  text[loc[0]:loc[1]],
Start:  loc[0],
End:    loc[1],
})
}
}

return matches
}

// HasSecrets verifica si un texto contiene secretos
func (sd *SecretDetector) HasSecrets(text string) bool {
return len(sd.Detect(text)) > 0
}
