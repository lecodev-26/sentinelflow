package v3

import (
	"strings"
	"testing"
	"time"
)

func TestCredentialPoolRotationAndQuarantine(t *testing.T) {
	p := NewCredentialPool("openai")
	if err := p.Add(&Credential{ID: "k1", Secret: "secret", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	c, err := p.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if c.Secret != "secret" {
		t.Fatal("secret mismatch")
	}
	p.Release("k1", false)
	p.Release("k1", false)
	p.Release("k1", false)
	if _, err := p.Acquire(); err == nil || !strings.Contains(err.Error(), "no healthy") {
		t.Fatalf("expected quarantined pool, got %v", err)
	}
	if err := p.Rotate("k1", &Credential{ID: "k2", Secret: "replacement"}); err != nil {
		t.Fatal(err)
	}
	c, err = p.Acquire()
	if err != nil || c.ID != "k2" {
		t.Fatalf("expected replacement credential, got %#v %v", c, err)
	}
}
func TestCredentialPoolNeverExposesSecretInMetadata(t *testing.T) {
	p := NewCredentialPool("anthropic")
	_ = p.Add(&Credential{ID: "a1", Secret: "super-secret"})
	m := p.ListMetadata()
	if len(m) != 1 {
		t.Fatal("expected metadata")
	}
	for _, k := range m {
		if _, ok := k["secret"]; ok {
			t.Fatal("secret leaked")
		}
	}
}
func TestCredentialPoolExpiry(t *testing.T) {
	p := NewCredentialPool("x")
	_ = p.Add(&Credential{ID: "expired", Secret: "x", ExpiresAt: time.Now().Add(-time.Second)})
	if _, err := p.Acquire(); err == nil {
		t.Fatal("expected expiry to block credential")
	}
}
