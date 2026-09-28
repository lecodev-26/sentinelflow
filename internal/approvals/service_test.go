package approvals

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEnvironmentAndStatusValidation(t *testing.T) {
	for _, v := range []string{"development", "staging", "production"} {
		if !validEnv(v) {
			t.Fatalf("expected valid environment %q", v)
		}
	}
	if validEnv("qa") {
		t.Fatal("qa must not be accepted")
	}
	for _, v := range []string{StatusPending, StatusApproved, StatusRejected, StatusExpired, StatusCancelled} {
		if !validStatus(v) {
			t.Fatalf("expected valid status %q", v)
		}
	}
	if validStatus("running") {
		t.Fatal("running must not be accepted")
	}
}

func TestRequestJSONDoesNotExposeRawPayload(t *testing.T) {
	now := time.Now().UTC()
	r := Request{
		ID: "apr_test", TenantID: "tenant", ProjectID: "project", Environment: "staging",
		RequesterID: "user", Action: "tool.execute", TargetType: "tool", TargetID: "shell",
		PayloadSHA256: "abc", Reason: "needs approval", Status: StatusPending,
		ExpiresAt: &now, CreatedAt: now, UpdatedAt: &now,
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"id", "tenant_id", "environment", "payload_sha256"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing JSON field %s: %s", want, s)
		}
	}
	if strings.Contains(s, "raw_payload") {
		t.Fatal("raw payload must not be exposed")
	}
}
