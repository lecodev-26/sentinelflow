package v3

import (
"context"
"net/url"
"regexp"
"strings"
)

var urlPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

var blockedHosts = []string{
"localhost",
"127.0.0.1",
"0.0.0.0",
"::1",
"169.254.169.254", // AWS metadata
"metadata.google.internal",
}

var blockedSchemes = []string{
"file",
"gopher",
"dict",
"ftp",
}

// SSRFScanner detecta URLs potencialmente peligrosas
type SSRFScanner struct{}

// NewSSRFScanner crea un SSRF scanner
func NewSSRFScanner() *SSRFScanner {
return &SSRFScanner{}
}

func (s *SSRFScanner) Name() string { return "ssrf" }

func (s *SSRFScanner) Scan(ctx context.Context, text string) []Finding {
var findings []Finding

urls := urlPattern.FindAllStringIndex(text, -1)
for _, loc := range urls {
rawURL := text[loc[0]:loc[1]]

u, err := url.Parse(rawURL)
if err != nil {
continue
}

// Verificar scheme
blocked := false
for _, scheme := range blockedSchemes {
if strings.EqualFold(u.Scheme, scheme) {
blocked = true
break
}
}

// Verificar host
if !blocked {
host := strings.ToLower(u.Hostname())
for _, h := range blockedHosts {
if host == h {
blocked = true
break
}
}
// IPs privadas
if strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "172.16.") {
blocked = true
}
}

if blocked {
findings = append(findings, Finding{
Category:    CategorySSRF,
Rule:        "blocked_url",
Match:       rawURL,
Start:       loc[0],
End:         loc[1],
Risk:        RiskHigh,
Description: "SSRF: blocked URL " + rawURL,
})
}
}

return findings
}
