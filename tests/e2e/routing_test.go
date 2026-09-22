package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRouting(t *testing.T) {
	t.Skip("E2E tests require proxy running")

	url := "http://localhost:8080/v1/chat/completions"

	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"GPT-3.5 → OpenAI", "gpt-3.5-turbo", "openai"},
		{"GPT-4 → OpenAI", "gpt-4", "openai"},
		{"Claude → Anthropic", "claude-3", "anthropic"},
		{"Default → OpenAI", "unknown-model", "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]interface{}{
				"model": tt.model,
				"messages": []map[string]string{
					{"role": "user", "content": "Hello"},
				},
			}
			jsonBody, _ := json.Marshal(body)

			resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
			if err != nil {
				t.Skip("Proxy not running, skipping test")
				return
			}
			defer resp.Body.Close()

			provider := resp.Header.Get("X-Provider")
			if provider == "" {
				var respBody map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&respBody)
				if p, ok := respBody["provider"].(string); ok {
					provider = p
				}
			}

			t.Logf("Provider: %s", provider)
		})
	}
}

func TestFailover(t *testing.T) {
	t.Skip("E2E tests require proxy running")

	url := "http://localhost:8080/v1/chat/completions"

	body := map[string]interface{}{
		"model": "openai-fail",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Skip("Proxy not running, skipping test")
		return
	}
	defer resp.Body.Close()

	t.Logf("Status: %d", resp.StatusCode)
}
