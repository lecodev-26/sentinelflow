package asecurity

import (
	"regexp"
	"strings"
)

type Action string

const (
	ActionAllow  Action = "allow"
	ActionReview Action = "review"
	ActionBlock  Action = "block"
)

type Finding struct {
	Kind, Detail string
	Severity     int
	Action       Action
}
type Detector struct{ secret, pii, inj *regexp.Regexp }

func NewDetector() *Detector {
	return &Detector{secret: regexp.MustCompile(`(?i)(api[_ -]?key|secret|password|bearer)\s*[:=]\s*[^\s]+`), pii: regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`), inj: regexp.MustCompile(`(?i)(ignore previous instructions|system prompt|jailbreak|bypass (?:the )?(?:policy|safety))`)}
}
func (d *Detector) Scan(text string) []Finding {
	var f []Finding
	if d.secret.MatchString(text) {
		f = append(f, Finding{"secret", "credential-like material detected", 10, ActionBlock})
	}
	if d.pii.MatchString(text) {
		f = append(f, Finding{"pii", "payment-card-like identifier detected", 9, ActionBlock})
	}
	if d.inj.MatchString(text) {
		f = append(f, Finding{"prompt_injection", "instruction override signal detected", 8, ActionReview})
	}
	return f
}
func Highest(f []Finding) Action {
	a := ActionAllow
	for _, x := range f {
		if x.Action == ActionBlock {
			return ActionBlock
		}
		if x.Action == ActionReview {
			a = ActionReview
		}
	}
	return a
}
func Redact(text string) string {
	re := regexp.MustCompile(`(?i)(api[_ -]?key|secret|password|bearer)\s*[:=]\s*[^\s]+`)
	return re.ReplaceAllStringFunc(text, func(s string) string {
		idx := strings.IndexAny(s, ":=")
		if idx < 0 {
			return "[REDACTED]"
		}
		return s[:idx+1] + " [REDACTED]"
	})
}
