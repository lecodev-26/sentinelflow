package core

import "testing"

func TestAnalyzerClassifiesCodeAndTools(t *testing.T) {
	a := NewAnalyzer().Analyze("write code and call a tool", 1)
	if a.Requirements.Intent != IntentToolUse || !a.Requirements.NeedsTools {
		t.Fatal(a)
	}
}
func TestAnalyzerDetectsSensitiveInput(t *testing.T) {
	a := NewAnalyzer().Analyze("use this API key secret", 0)
	if !a.Requirements.Sensitive {
		t.Fatal(a)
	}
}
