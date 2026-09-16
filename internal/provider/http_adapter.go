package provider

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"time"
)

// HTTPProvider es un adapter genérico para cualquier API HTTP compatible
type HTTPProvider struct {
name     string
baseURL  string
headers  map[string]string
client   *http.Client
timeout  time.Duration
models   []string
}

// NewHTTPProvider crea un adapter HTTP genérico
func NewHTTPProvider(name, baseURL string, headers map[string]string, timeout time.Duration, models []string) *HTTPProvider {
if timeout == 0 {
timeout = 30 * time.Second
}
return &HTTPProvider{
name:    name,
baseURL: baseURL,
headers: headers,
client:  &http.Client{Timeout: timeout},
timeout: timeout,
models:  models,
}
}

func (p *HTTPProvider) Name() string { return p.name }

func (p *HTTPProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
startTime := time.Now()

body, err := json.Marshal(req)
if err != nil {
return nil, fmt.Errorf("error marshaling request: %w", err)
}

httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
if err != nil {
return nil, fmt.Errorf("error creating request: %w", err)
}
httpReq.Header.Set("Content-Type", "application/json")
for k, v := range p.headers {
httpReq.Header.Set(k, v)
}

resp, err := p.client.Do(httpReq)
if err != nil {
return nil, fmt.Errorf("error executing request: %w", err)
}
defer resp.Body.Close()

respBody, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("error reading response: %w", err)
}

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
}

var apiResp struct {
ID      string `json:"id"`
Model   string `json:"model"`
Choices []struct {
Index   int `json:"index"`
Message struct {
Role    string `json:"role"`
Content string `json:"content"`
} `json:"message"`
FinishReason string `json:"finish_reason"`
} `json:"choices"`
Usage struct {
PromptTokens     int `json:"prompt_tokens"`
CompletionTokens int `json:"completion_tokens"`
TotalTokens      int `json:"total_tokens"`
} `json:"usage"`
}

if err := json.Unmarshal(respBody, &apiResp); err != nil {
return nil, fmt.Errorf("error parsing response: %w", err)
}

response := &ChatResponse{
ID:        apiResp.ID,
Model:     apiResp.Model,
Provider:  p.name,
Latency:   time.Since(startTime),
Timestamp: time.Now(),
}

if len(apiResp.Choices) > 0 {
response.Choices = []Choice{
{
Index: apiResp.Choices[0].Index,
Message: Message{
Role:    apiResp.Choices[0].Message.Role,
Content: apiResp.Choices[0].Message.Content,
},
FinishReason: apiResp.Choices[0].FinishReason,
},
}
}

response.Usage = Usage{
PromptTokens:     apiResp.Usage.PromptTokens,
CompletionTokens: apiResp.Usage.CompletionTokens,
TotalTokens:      apiResp.Usage.TotalTokens,
}

return response, nil
}

func (p *HTTPProvider) Stream(ctx context.Context, req *ChatRequest) (<-chan Event, error) {
events := make(chan Event)
go func() {
defer close(events)
resp, err := p.Chat(ctx, req)
if err != nil {
events <- Event{Type: "error", Content: err.Error()}
return
}
if len(resp.Choices) > 0 {
events <- Event{Type: "chunk", Content: resp.Choices[0].Message.Content}
}
events <- Event{Type: "done", Content: ""}
}()
return events, nil
}

func (p *HTTPProvider) Health(ctx context.Context) error {
req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
if err != nil {
return fmt.Errorf("error creating health check: %w", err)
}
for k, v := range p.headers {
req.Header.Set(k, v)
}

resp, err := p.client.Do(req)
if err != nil {
return fmt.Errorf("health check failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode >= 500 {
return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
}
return nil
}

func (p *HTTPProvider) Models(ctx context.Context) ([]string, error) {
if len(p.models) > 0 {
return p.models, nil
}
return []string{}, nil
}
