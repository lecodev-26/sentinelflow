package v3

import "testing"

func TestModelEquivalenceResolve(t *testing.T) {
	catalog := NewModelRegistry()
	r := NewModelEquivalenceRegistry()
	r.Register(&ModelEquivalence{ID: "balanced-chat", Name: "Balanced Chat", Members: []string{"openai:gpt-4-turbo", "anthropic:claude-3-sonnet", "ollama:llama3.1"}})
	got := r.Resolve("balanced-chat", catalog, []string{"chat"}, 32000)
	if len(got) != 3 {
		t.Fatalf("expected 3 compatible models, got %d", len(got))
	}
	if got[0].Model.ContextSize < 32000 {
		t.Fatalf("returned model below context requirement")
	}
}

func TestModelEquivalenceDoesNotInventMembers(t *testing.T) {
	catalog := NewModelRegistry()
	r := NewModelEquivalenceRegistry()
	r.Register(&ModelEquivalence{ID: "strict", Name: "Strict", Members: []string{"openai:not-real", "not-a-provider:model"}})
	if got := r.Resolve("strict", catalog, nil, 0); len(got) != 0 {
		t.Fatalf("expected no matches, got %d", len(got))
	}
}
