package normalizer

import "testing"

func TestNormalizeResponsesStringInput(t *testing.T) {
	r, err := New().NormalizeResponses([]byte(`{"model":"demo","input":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Model != "demo" || len(r.Messages) != 1 || r.Messages[0].Content != "hello" {
		t.Fatalf("unexpected: %#v", r)
	}
}

func TestNormalizeResponsesInstructionsAndMessages(t *testing.T) {
	r, err := New().NormalizeResponses([]byte(`{"model":"demo","instructions":"be concise","input":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Messages) != 2 || r.Messages[0].Role != "system" {
		t.Fatalf("unexpected: %#v", r.Messages)
	}
}
