package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// HMAC signature
// =============================================================================

func TestSign_DeterministicWithSameInputs(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	sig1 := sign(payload, "secret-abc")
	sig2 := sign(payload, "secret-abc")
	if sig1 != sig2 {
		t.Errorf("signature not deterministic: %q vs %q", sig1, sig2)
	}
}

func TestSign_DifferentSecretDifferentSignature(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	s1 := sign(payload, "secret-a")
	s2 := sign(payload, "secret-b")
	if s1 == s2 {
		t.Errorf("different secrets produced same signature")
	}
}

func TestSign_DifferentPayloadDifferentSignature(t *testing.T) {
	s1 := sign([]byte(`{"a":1}`), "secret")
	s2 := sign([]byte(`{"a":2}`), "secret")
	if s1 == s2 {
		t.Errorf("different payloads produced same signature")
	}
}

func TestSign_HasPrefix(t *testing.T) {
	s := sign([]byte("x"), "secret")
	if !strings.HasPrefix(s, "sha256=") {
		t.Errorf("signature should start with sha256=, got %q", s)
	}
}

func TestSign_MatchesManualHMAC(t *testing.T) {
	// Verifica que sign() es exactamente HMAC-SHA256(payload, secret) en hex.
	payload := []byte(`{"event":"verify"}`)
	secret := "my-secret"
	expected := hmac.New(sha256.New, []byte(secret))
	expected.Write(payload)
	want := "sha256=" + hex.EncodeToString(expected.Sum(nil))

	got := sign(payload, secret)
	if got != want {
		t.Errorf("sign mismatch:\n  got:  %q\n  want: %q", got, want)
	}
}

// =============================================================================
// Backoff
// =============================================================================

func TestComputeBackoff_ExponentialWithoutJitterBounds(t *testing.T) {
	// Con jitter ±20%, verificamos que los valores caen en los rangos esperados.
	d := &Dispatcher{
		baseBackoff: 5 * time.Second,
		maxBackoff:  60 * time.Second,
	}
	cases := []struct {
		attempts int
		min, max time.Duration
	}{
		{1, 4 * time.Second, 6 * time.Second},   // 5s ±20%
		{2, 8 * time.Second, 12 * time.Second},  // 10s ±20%
		{3, 16 * time.Second, 24 * time.Second}, // 20s ±20%
		{4, 32 * time.Second, 48 * time.Second}, // 40s ±20%
		{5, 48 * time.Second, 72 * time.Second}, // capped a 60s ±20%
		{99, 48 * time.Second, 72 * time.Second},
	}
	for _, c := range cases {
		for i := 0; i < 100; i++ {
			got := d.computeBackoff(c.attempts)
			if got < c.min || got > c.max {
				t.Errorf("attempts=%d: got %v, want between %v and %v", c.attempts, got, c.min, c.max)
				break
			}
		}
	}
}

func TestComputeBackoff_ZeroOrNegativeAttempts(t *testing.T) {
	d := &Dispatcher{
		baseBackoff: 5 * time.Second,
		maxBackoff:  60 * time.Second,
	}
	// attempts=0 debe comportarse como attempts=1
	for i := 0; i < 50; i++ {
		got := d.computeBackoff(0)
		if got < 4*time.Second || got > 6*time.Second {
			t.Errorf("attempts=0: got %v, want ~5s±20%%", got)
		}
	}
}

func TestComputeBackoff_CappedAtMax(t *testing.T) {
	d := &Dispatcher{
		baseBackoff: 1 * time.Second,
		maxBackoff:  5 * time.Second,
	}
	// Con attempts=20, base*2^19 overflow-able; debe devolver ~5s ±20%
	for i := 0; i < 50; i++ {
		got := d.computeBackoff(20)
		if got < 4*time.Second || got > 6*time.Second {
			t.Errorf("attempts=20: got %v, want ~5s±20%%", got)
		}
	}
}

func TestComputeBackoff_NonNegative(t *testing.T) {
	// El jitter con delta negativo nunca debe producir backoff < 0
	d := &Dispatcher{
		baseBackoff: 100 * time.Millisecond,
		maxBackoff:  1 * time.Second,
	}
	for i := 0; i < 500; i++ {
		if got := d.computeBackoff(1); got < 0 {
			t.Fatalf("computeBackoff returned negative: %v", got)
		}
	}
}

// =============================================================================
// Default config sanity
// =============================================================================

func TestDefaultDispatcherConfig_Reasonable(t *testing.T) {
	c := DefaultDispatcherConfig()
	if c.PollInterval <= 0 {
		t.Errorf("PollInterval should be > 0, got %v", c.PollInterval)
	}
	if c.BatchSize <= 0 {
		t.Errorf("BatchSize should be > 0, got %d", c.BatchSize)
	}
	if c.MaxAttempts <= 0 {
		t.Errorf("MaxAttempts should be > 0, got %d", c.MaxAttempts)
	}
	if c.BaseBackoff <= 0 {
		t.Errorf("BaseBackoff should be > 0, got %v", c.BaseBackoff)
	}
	if c.MaxBackoff < c.BaseBackoff {
		t.Errorf("MaxBackoff (%v) should be >= BaseBackoff (%v)", c.MaxBackoff, c.BaseBackoff)
	}
	if c.HTTPTimeout <= 0 {
		t.Errorf("HTTPTimeout should be > 0, got %v", c.HTTPTimeout)
	}
}
