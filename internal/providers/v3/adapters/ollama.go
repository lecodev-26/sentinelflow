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

type OllamaAdapter struct {
baseURL string
client  *http.Client
}

func NewOllamaAdapter(baseURL string) *OllamaAdapter {
if baseURL == "" {
baseURL = "http://localhost:11434"
}
return &OllamaAdapter{
baseURL: baseURL,
client:  &http.Client{Timeout: 120 * time.Second},
}
}

func (a *OllamaAdapter) ID() string   { return "ollama" }
func (a *OllamaAdapter) Name() string { return "Ollama (Local)" }

func (a *OllamaAdapter) Capabilities() []v3.Capability {
return []v3.Capability{v3.CapChat, v3.CapStream}
}

func (a *OllamaAdapter) Models(ctx context.Context) ([]v3.ModelInfo, error) {
resp, err := a.client.Get(a.baseURL + "/api/tags")
if err != nil {
return nil, err
}
defer resp.Body.Close()

var result struct {
Models []struct {
Name string `json:"name"`
} `json:"models"`
}

if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, err
}

var models []v3.ModelInfo
for _, m := range result.Models {
models = append(models, v3.ModelInfo{
ID:           m.Name,
Name:         m.Name,
Provider:     "ollama",
Capabilities: a.Capabilities(),
ContextSize:  8192,
})
}
return models, nil
}

func (a *OllamaAdapter) Chat(ctx context.Context, req *v3.ChatRequest) (*v3.ChatResponse, error) {
start := time.Now()

body, _ := json.Marshal(map[string]interface{}{
"model":    req.Model,
"messages": req.Messages,
"stream":   false,
})

httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/chat", bytes.NewReader(body))
if err != nil {
return nil, err
}
httpReq.Header.Set("Content-Type", "application/json")

resp, err := a.client.Do(httpReq)
if err != nil {
return nil, err
}
defer resp.Body.Close()

respBody, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var apiResp struct {
Model   string `json:"model"`
Message struct {
Role    string `json:"role"`
Content string `json:"content"`
} `json:"message"`
Done            bool `json:"done"`
PromptEvalCount int  `json:"prompt_eval_count"`
EvalCount       int  `json:"eval_count"`
}

if err := json.Unmarshal(respBody, &apiResp); err != nil {
return nil, err
}

return &v3.ChatResponse{
Model:    apiResp.Model,
Provider: a.ID(),
Choices: []v3.Choice{{
Index:   0,
Message: v3.Message{Role: "assistant", Content: apiResp.Message.Content},
}},
Usage: v3.Usage{
PromptTokens:     apiResp.PromptEvalCount,
CompletionTokens: apiResp.EvalCount,
TotalTokens:      apiResp.PromptEvalCount + apiResp.EvalCount,
},
Latency:   time.Since(start),
Timestamp: time.Now(),
}, nil
}

func (a *OllamaAdapter) Stream(ctx context.Context, req *v3.ChatRequest) (<-chan v3.StreamChunk, error) {
body, _ := json.Marshal(map[string]interface{}{
"model":    req.Model,
"messages": req.Messages,
"stream":   true,
})

httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/api/chat", bytes.NewReader(body))
if err != nil {
return nil, err
}
httpReq.Header.Set("Content-Type", "application/json")

resp, err := a.client.Do(httpReq)
if err != nil {
return nil, err
}

chunks := make(chan v3.StreamChunk, 10)

go func() {
defer close(chunks)
defer resp.Body.Close()

scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
line := strings.TrimSpace(scanner.Text())
if line == "" {
continue
}

var event struct {
Message struct {
Content string `json:"content"`
} `json:"message"`
Done bool `json:"done"`
}
if err := json.Unmarshal([]byte(line), &event); err != nil {
continue
}

if event.Message.Content != "" {
select {
case <-ctx.Done():
return
case chunks <- v3.StreamChunk{
Delta:     event.Message.Content,
Timestamp: time.Now(),
}:
}
}

if event.Done {
return
}
}
}()

return chunks, nil
}

func (a *OllamaAdapter) Health(ctx context.Context) error {
resp, err := a.client.Get(a.baseURL + "/api/tags")
if err != nil {
return err
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return fmt.Errorf("ollama unhealthy: %d", resp.StatusCode)
}
return nil
}
