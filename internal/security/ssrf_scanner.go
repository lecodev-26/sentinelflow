package security

import (
	"context"
	"regexp"

	"github.com/lecodev-26/sentinelflow/internal/security/ssrf"
)

// SSRFScanner detecta URLs potencialmente peligrosas (SSRF)
type SSRFScanner struct {
	validator *ssrf.Validator
	urlRegex  *regexp.Regexp
}

func NewSSRFScanner() *SSRFScanner {
	return &SSRFScanner{
		validator: ssrf.NewValidator(),
		urlRegex:  regexp.MustCompile(`https?://[^\s"'<>]+`),
	}
}

func (s *SSRFScanner) Name() string { return "ssrf" }

func (s *SSRFScanner) Scan(ctx context.Context, text string) []Finding {
	urls := s.urlRegex.FindAllStringIndex(text, -1)
	findings := make([]Finding, 0)

	for _, loc := range urls {
		url := text[loc[0]:loc[1]]

		if err := s.validator.Validate(url); err != nil {
			findings = append(findings, Finding{
				Category:    CategorySSRF,
				Rule:        "url_blocked",
				Match:       url,
				Start:       loc[0],
				End:         loc[1],
				Risk:        RiskHigh,
				Description: "SSRF blocked: " + err.Error(),
			})
		}
	}

	return findings
}
