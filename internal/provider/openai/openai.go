package openai

import (
"bytes"
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"time"

"github.com/lecodev-26/sentinelflow/internal/provider"
)

type Client struct {
apiKey  string
baseURL string
client  *http.Client
}

func NewClient() *Client {
apiKey := os.Getenv("OPENAI_API_KEY")
if apiKey == "" {
apiKey = "sk-dummy-key"
}
return &Client{
apiKey:  apiKey,
baseURL: "https://api.openai.com/v1",
client:  &http.Client{Timeout: 30 * time.Second},
}
}

func (c *Client) Name() string { return "openai" }

type openAIRequest struct {
Model       string          `json:"model"`
Messages    []openAIMessage `json:"messages"`
Temperature *float32        `json:"temperature,omitempty"`
MaxTokens   *int            `json:"max_tokens,omitempty"`
Stream      bool            `json:"stream"`
}

type openAIMessage struct {
Role    string `json:"role"`
Content string `json:"content"`
}

type openAIResponse struct {
ID      string `json:"id"`
Model   string `json:"model"`
Choices []struct {
Index        int `json:"index"`
Message      struct {
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

func (c *Client) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
startTime := time.Now()

messages := make([]openAIMessage, len(req.Messages))
for i, msg := range req.Messages {
messages[i] = openAIMessage{Role: msg.Role, Content: msg.Content}
}

openAIReq := openAIRequest{
Model:       req.Model,
Messages:    messages,
Temperature: req.Temperature,
MaxTokens:   req.MaxTokens,
Stream:      false,
}

body, err := json.Marshal(openAIReq)
if err != nil {
return nil, fmt.Errorf("error marshaling request: %w", err)
}

httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
if err != nil {
return nil, fmt.Errorf("error creating request: %w", err)
}
httpReq.Header.Set("Content-Type", "application/json")
httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

resp, err := c.client.Do(httpReq)
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

var openAIResp openAIResponse
if err := json.Unmarshal(respBody, &openAIResp); err != nil {
return nil, fmt.Errorf("error parsing response: %w", err)
}

response := &provider.ChatResponse{
ID:        openAIResp.ID,
Model:     openAIResp.Model,
Provider:  c.Name(),
Latency:   time.Since(startTime),
Timestamp: time.Now(),
}

if len(openAIResp.Choices) > 0 {
response.Choices = []provider.Choice{
{
Index: openAIResp.Choices[0].Index,
Message: provider.Message{
Role:    openAIResp.Choices[0].Message.Role,
Content: openAIResp.Choices[0].Message.Content,
},
FinishReason: openAIResp.Choices[0].FinishReason,
},
}
}

response.Usage = provider.Usage{
PromptTokens:     openAIResp.Usage.PromptTokens,
CompletionTokens: openAIResp.Usage.CompletionTokens,
TotalTokens:      openAIResp.Usage.TotalTokens,
}

return response, nil
}

func (c *Client) Stream(ctx context.Context, req *provider.ChatRequest) (<-chan provider.Event, error) {
events := make(chan provider.Event)
go func() {
defer close(events)
response, err := c.Chat(ctx, req)
if err != nil {
events <- provider.Event{Type: "error", Content: err.Error()}
return
}
if len(response.Choices) > 0 {
events <- provider.Event{Type: "chunk", Content: response.Choices[0].Message.Content}
}
events <- provider.Event{Type: "done", Content: ""}
}()
return events, nil
}

func (c *Client) Health(ctx context.Context) error {
req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
if err != nil {
return fmt.Errorf("error creating health check: %w", err)
}
req.Header.Set("Authorization", "Bearer "+c.apiKey)

resp, err := c.client.Do(req)
if err != nil {
return fmt.Errorf("health check failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
}
return nil
}

func (c *Client) Models(ctx context.Context) ([]string, error) {
return []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"}, nil
}
