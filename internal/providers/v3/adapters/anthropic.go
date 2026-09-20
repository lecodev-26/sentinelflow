package adapters

import (
"bufio"
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"strings"
"time"

"github.com/lecodev-26/sentinelflow/internal/providers/v3"
)

type AnthropicAdapter struct {
apiKey  string
baseURL string
client  *http.Client
}

func NewAnthropicAdapter(apiKey, baseURL string) *AnthropicAdapter {
if baseURL == "" {
baseURL = "https://api.anthropic.com/v1"
}
return &AnthropicAdapter{
apiKey:  apiKey,
baseURL: baseURL,
client:  &http.Client{Timeout: 60 * time.Second},
}
}

func (a *AnthropicAdapter) ID() string   { return "anthropic" }
func (a *AnthropicAdapter) Name() string { return "Anthropic" }

func (a *AnthropicAdapter) Capabilities() []v3.Capability {
return []v3.Capability{v3.CapChat, v3.CapStream, v3.CapTools, v3.CapVision}
}

func (a *AnthropicAdapter) Models(ctx context.Context) ([]v3.ModelInfo, error) {
return []v3.ModelInfo{
{ID: "claude-3-opus", Name: "Claude 3 Opus", Provider: "anthropic", Capabilities: a.Capabilities(), ContextSize: 200000},
{ID: "claude-3-sonnet", Name: "Claude 3 Sonnet", Provider: "anthropic", Capabilities: a.Capabilities(), ContextSize: 200000},
{ID: "claude-3-haiku", Name: "Claude 3 Haiku", Provider: "anthropic", Capabilities: a.Capabilities(), ContextSize: 200000},
}, nil
}

func (a *AnthropicAdapter) Chat(ctx context.Context, req *v3.ChatRequest) (*v3.ChatResponse, error) {
start := time.Now()

// Separar system de messages
var system string
var messages []map[string]string
for _, m := range req.Messages {
if m.Role == "system" {
system = m.Content
continue
}
messages = append(messages, map[string]string{"role": m.Role, "content": m.Content})
}

payload := map[string]interface{}{
"model":      req.Model,
"messages":   messages,
"max_tokens": 4096,
}
if system != "" {
payload["system"] = system
}
if req.Temperature != nil {
payload["temperature"] = *req.Temperature
}
if req.MaxTokens != nil {
payload["max_tokens"] = *req.MaxTokens
}

body, err := json.Marshal(payload)
if err != nil {
return nil, err
}

httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(body))
if err != nil {
return nil, err
}
httpReq.Header.Set("Content-Type", "application/json")
httpReq.Header.Set("x-api-key", a.apiKey)
httpReq.Header.Set("anthropic-version", "2023-06-01")

resp, err := a.client.Do(httpReq)
if err != nil {
return nil, err
}
defer resp.Body.Close()

respBody, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

if resp.StatusCode != http.StatusOK {
if resp.StatusCode == http.StatusUnauthorized {
return nil, v3.ErrUnauthorized
}
if resp.StatusCode == http.StatusTooManyRequests {
return nil, v3.ErrRateLimited
}
return nil, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, string(respBody))
}

var apiResp struct {
ID      string `json:"id"`
Model   string `json:"model"`
Content []struct {
Type string `json:"type"`
Text string `json:"text"`
} `json:"content"`
StopReason string `json:"stop_reason"`
Usage      struct {
InputTokens  int `json:"input_tokens"`
OutputTokens int `json:"output_tokens"`
} `json:"usage"`
}

if err := json.Unmarshal(respBody, &apiResp); err != nil {
return nil, err
}

var text string
for _, c := range apiResp.Content {
if c.Type == "text" {
text += c.Text
}
}

return &v3.ChatResponse{
ID:       apiResp.ID,
Model:    apiResp.Model,
Provider: a.ID(),
Choices: []v3.Choice{{
Index:        0,
Message:      v3.Message{Role: "assistant", Content: text},
FinishReason: apiResp.StopReason,
}},
Usage: v3.Usage{
PromptTokens:     apiResp.Usage.InputTokens,
CompletionTokens: apiResp.Usage.OutputTokens,
TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
},
Latency:   time.Since(start),
Timestamp: time.Now(),
}, nil
}

func (a *AnthropicAdapter) Stream(ctx context.Context, req *v3.ChatRequest) (<-chan v3.StreamChunk, error) {
var system string
var messages []map[string]string
for _, m := range req.Messages {
if m.Role == "system" {
system = m.Content
continue
}
messages = append(messages, map[string]string{"role": m.Role, "content": m.Content})
}

payload := map[string]interface{}{
"model":      req.Model,
"messages":   messages,
"max_tokens": 4096,
"stream":     true,
}
if system != "" {
payload["system"] = system
}

body, _ := json.Marshal(payload)

httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/messages", bytes.NewReader(body))
if err != nil {
return nil, err
}
httpReq.Header.Set("Content-Type", "application/json")
httpReq.Header.Set("x-api-key", a.apiKey)
httpReq.Header.Set("anthropic-version", "2023-06-01")

resp, err := a.client.Do(httpReq)
if err != nil {
return nil, err
}

if resp.StatusCode != http.StatusOK {
resp.Body.Close()
return nil, fmt.Errorf("anthropic stream error %d", resp.StatusCode)
}

chunks := make(chan v3.StreamChunk, 10)

go func() {
defer close(chunks)
defer resp.Body.Close()

scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
line := scanner.Text()
if !strings.HasPrefix(line, "data: ") {
continue
}

data := strings.TrimPrefix(line, "data: ")
var event struct {
Type  string `json:"type"`
Delta struct {
Type string `json:"type"`
Text string `json:"text"`
} `json:"delta"`
}
if err := json.Unmarshal([]byte(data), &event); err != nil {
continue
}

if event.Type == "content_block_delta" && event.Delta.Text != "" {
select {
case <-ctx.Done():
return
case chunks <- v3.StreamChunk{
Delta:     event.Delta.Text,
Timestamp: time.Now(),
}:
}
}
}
}()

return chunks, nil
}

func (a *AnthropicAdapter) Health(ctx context.Context) error {
// Anthropic no tiene endpoint de health público, hacemos un ping
httpReq, err := http.NewRequestWithContext(ctx, "GET", "https://status.anthropic.com/api/v2/status.json", nil)
if err != nil {
return err
}

resp, err := a.client.Do(httpReq)
if err != nil {
return err
}
defer resp.Body.Close()

if resp.StatusCode >= 500 {
return fmt.Errorf("anthropic status API error: %d", resp.StatusCode)
}
return nil
}
