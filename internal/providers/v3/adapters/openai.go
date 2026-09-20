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

// OpenAIAdapter implementa el provider para OpenAI
type OpenAIAdapter struct {
apiKey  string
baseURL string
client  *http.Client
}

// NewOpenAIAdapter crea un nuevo adapter de OpenAI
func NewOpenAIAdapter(apiKey, baseURL string) *OpenAIAdapter {
if baseURL == "" {
baseURL = "https://api.openai.com/v1"
}
return &OpenAIAdapter{
apiKey:  apiKey,
baseURL: baseURL,
client:  &http.Client{Timeout: 60 * time.Second},
}
}

func (a *OpenAIAdapter) ID() string   { return "openai" }
func (a *OpenAIAdapter) Name() string { return "OpenAI" }

func (a *OpenAIAdapter) Capabilities() []v3.Capability {
return []v3.Capability{v3.CapChat, v3.CapStream, v3.CapTools, v3.CapVision, v3.CapJSON, v3.CapFunction}
}

func (a *OpenAIAdapter) Models(ctx context.Context) ([]v3.ModelInfo, error) {
return []v3.ModelInfo{
{ID: "gpt-4", Name: "GPT-4", Provider: "openai", Capabilities: a.Capabilities(), ContextSize: 8192},
{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "openai", Capabilities: a.Capabilities(), ContextSize: 128000},
{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "openai", Capabilities: a.Capabilities(), ContextSize: 16385},
}, nil
}

func (a *OpenAIAdapter) Chat(ctx context.Context, req *v3.ChatRequest) (*v3.ChatResponse, error) {
start := time.Now()

body, err := json.Marshal(map[string]interface{}{
"model":       req.Model,
"messages":    req.Messages,
"temperature": req.Temperature,
"max_tokens":  req.MaxTokens,
"stream":      false,
})
if err != nil {
return nil, err
}

httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
if err != nil {
return nil, err
}
httpReq.Header.Set("Content-Type", "application/json")
httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

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
return nil, fmt.Errorf("openai error %d: %s", resp.StatusCode, string(respBody))
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
return nil, err
}

result := &v3.ChatResponse{
ID:        apiResp.ID,
Model:     apiResp.Model,
Provider:  a.ID(),
Latency:   time.Since(start),
Timestamp: time.Now(),
}

for _, c := range apiResp.Choices {
result.Choices = append(result.Choices, v3.Choice{
Index:        c.Index,
Message:      v3.Message{Role: c.Message.Role, Content: c.Message.Content},
FinishReason: c.FinishReason,
})
}

result.Usage = v3.Usage{
PromptTokens:     apiResp.Usage.PromptTokens,
CompletionTokens: apiResp.Usage.CompletionTokens,
TotalTokens:      apiResp.Usage.TotalTokens,
}

return result, nil
}

func (a *OpenAIAdapter) Stream(ctx context.Context, req *v3.ChatRequest) (<-chan v3.StreamChunk, error) {
body, err := json.Marshal(map[string]interface{}{
"model":       req.Model,
"messages":    req.Messages,
"temperature": req.Temperature,
"max_tokens":  req.MaxTokens,
"stream":      true,
})
if err != nil {
return nil, err
}

httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(body))
if err != nil {
return nil, err
}
httpReq.Header.Set("Content-Type", "application/json")
httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

resp, err := a.client.Do(httpReq)
if err != nil {
return nil, err
}

if resp.StatusCode != http.StatusOK {
resp.Body.Close()
return nil, fmt.Errorf("openai stream error %d", resp.StatusCode)
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
if data == "[DONE]" {
return
}

var event struct {
Choices []struct {
Index int `json:"index"`
Delta struct {
Content string `json:"content"`
} `json:"delta"`
FinishReason string `json:"finish_reason"`
} `json:"choices"`
}

if err := json.Unmarshal([]byte(data), &event); err != nil {
continue
}

for _, c := range event.Choices {
select {
case <-ctx.Done():
return
case chunks <- v3.StreamChunk{
Delta:        c.Delta.Content,
FinishReason: c.FinishReason,
Index:        c.Index,
Timestamp:    time.Now(),
}:
}
}
}
}()

return chunks, nil
}

func (a *OpenAIAdapter) Health(ctx context.Context) error {
httpReq, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+"/models", nil)
if err != nil {
return err
}
httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

resp, err := a.client.Do(httpReq)
if err != nil {
return err
}
defer resp.Body.Close()

if resp.StatusCode >= 500 {
return fmt.Errorf("provider unhealthy: %d", resp.StatusCode)
}
return nil
}
