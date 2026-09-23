package v3

import (
	"context"
	"testing"
)

// =============================================================================
// PII Scanner
// =============================================================================

func TestPIIScanner_Email(t *testing.T) {
	s := NewPIIScanner()
	fs := s.Scan(context.Background(), "contact me at john.doe@example.com please")
	if len(fs) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(fs), fs)
	}
	if fs[0].Rule != "email" {
		t.Errorf("rule: got %q want email", fs[0].Rule)
	}
	if fs[0].Risk != RiskMedium {
		t.Errorf("risk: got %q want medium", fs[0].Risk)
	}
}

func TestPIIScanner_CreditCard_IsCritical(t *testing.T) {
	s := NewPIIScanner()
	fs := s.Scan(context.Background(), "card 4111 1111 1111 1111 end")
	found := false
	for _, f := range fs {
		if f.Rule == "credit_card" {
			found = true
			if f.Risk != RiskCritical {
				t.Errorf("credit_card risk: got %q want critical", f.Risk)
			}
		}
	}
	if !found {
		t.Errorf("expected credit_card finding, got: %+v", fs)
	}
}

func TestPIIScanner_SSN_IsCritical(t *testing.T) {
	s := NewPIIScanner()
	fs := s.Scan(context.Background(), "ssn is 123-45-6789")
	found := false
	for _, f := range fs {
		if f.Rule == "ssn" {
			found = true
			if f.Risk != RiskCritical {
				t.Errorf("ssn risk: got %q want critical", f.Risk)
			}
		}
	}
	if !found {
		t.Errorf("expected ssn finding, got: %+v", fs)
	}
}

func TestPIIScanner_CleanText(t *testing.T) {
	s := NewPIIScanner()
	fs := s.Scan(context.Background(), "hello world, this is a clean sentence")
	if len(fs) != 0 {
		t.Errorf("expected 0 findings on clean text, got %d: %+v", len(fs), fs)
	}
}

func TestPIIScanner_MultipleFindings(t *testing.T) {
	s := NewPIIScanner()
	fs := s.Scan(context.Background(), "email a@b.com and another c@d.org and ssn 111-22-3333")
	if len(fs) < 3 {
		t.Errorf("expected >=3 findings, got %d: %+v", len(fs), fs)
	}
}

// =============================================================================
// Secret Scanner
// =============================================================================

func TestSecretScanner_OpenAIKey_IsCritical(t *testing.T) {
	s := NewSecretScanner()
	fs := s.Scan(context.Background(), "token: sk-abcdef1234567890abcdef1234567890abcdef")
	if len(fs) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(fs), fs)
	}
	if fs[0].Rule != "openai_key" {
		t.Errorf("rule: got %q want openai_key", fs[0].Rule)
	}
	if fs[0].Risk != RiskCritical {
		t.Errorf("risk: got %q want critical", fs[0].Risk)
	}
}

func TestSecretScanner_AWSKey(t *testing.T) {
	s := NewSecretScanner()
	fs := s.Scan(context.Background(), "AKIAIOSFODNN7EXAMPLE is the key")
	found := false
	for _, f := range fs {
		if f.Rule == "aws_key" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected aws_key finding, got: %+v", fs)
	}
}

func TestSecretScanner_GitHubToken(t *testing.T) {
	s := NewSecretScanner()
	fs := s.Scan(context.Background(), "ghp_abcdefghijklmnopqrstuvwxyz0123456789")
	found := false
	for _, f := range fs {
		if f.Rule == "github_token" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected github_token finding, got: %+v", fs)
	}
}

func TestSecretScanner_PrivateKey(t *testing.T) {
	s := NewSecretScanner()
	fs := s.Scan(context.Background(), "-----BEGIN RSA PRIVATE KEY-----\nMIIE...")
	found := false
	for _, f := range fs {
		if f.Rule == "private_key" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected private_key finding, got: %+v", fs)
	}
}

func TestSecretScanner_CleanText(t *testing.T) {
	s := NewSecretScanner()
	fs := s.Scan(context.Background(), "nothing to see here, just normal text")
	if len(fs) != 0 {
		t.Errorf("expected 0 findings, got %d: %+v", len(fs), fs)
	}
}

// =============================================================================
// Decision
// =============================================================================

func TestDecision_AddFinding_EscalatesToBlock_OnCritical(t *testing.T) {
	d := NewDecision()
	if d.Action != ActionAllow {
		t.Fatalf("initial action should be allow, got %q", d.Action)
	}
	d.AddFinding(Finding{Category: CategorySecret, Rule: "openai_key", Risk: RiskCritical})
	if d.Action != ActionBlock {
		t.Errorf("critical finding should set ActionBlock, got %q", d.Action)
	}
	if !d.IsBlocked() {
		t.Errorf("IsBlocked should be true")
	}
}

func TestDecision_AddFinding_Redact_OnMedium(t *testing.T) {
	d := NewDecision()
	d.AddFinding(Finding{Category: CategoryPII, Rule: "email", Risk: RiskMedium})
	if d.Action != ActionRedact {
		t.Errorf("medium finding should set ActionRedact, got %q", d.Action)
	}
}

func TestDecision_AddFinding_Warn_OnLow(t *testing.T) {
	d := NewDecision()
	d.AddFinding(Finding{Category: CategoryPII, Rule: "ipv4", Risk: RiskLow})
	if d.Action != ActionWarn {
		t.Errorf("low finding should set ActionWarn, got %q", d.Action)
	}
}

func TestDecision_Merge_TakesHighestRisk(t *testing.T) {
	a := NewDecision()
	a.AddFinding(Finding{Category: CategoryPII, Rule: "email", Risk: RiskMedium})

	b := NewDecision()
	b.AddFinding(Finding{Category: CategorySecret, Rule: "openai_key", Risk: RiskCritical})

	a.Merge(b)
	if a.Risk != RiskCritical {
		t.Errorf("merged risk: got %q want critical", a.Risk)
	}
	if a.Action != ActionBlock {
		t.Errorf("merged action: got %q want block", a.Action)
	}
}
