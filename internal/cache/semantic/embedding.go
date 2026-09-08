package semantic

import (
"bytes"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"sync"
"time"
)

// Embedder genera embeddings para texto
type Embedder interface {
Embed(text string) ([]float64, error)
}

// OpenAIEmbedder usa la API de OpenAI para embeddings
type OpenAIEmbedder struct {
apiKey  string
baseURL string
client  *http.Client
mu      sync.Mutex
}

// NewOpenAIEmbedder crea un nuevo embedder de OpenAI
func NewOpenAIEmbedder() *OpenAIEmbedder {
apiKey := os.Getenv("OPENAI_API_KEY")
if apiKey == "" {
apiKey = "sk-dummy-key"
}

return &OpenAIEmbedder{
apiKey:  apiKey,
baseURL: "https://api.openai.com/v1",
client:  &http.Client{Timeout: 10 * time.Second},
}
}

// Embed genera un embedding para un texto
func (e *OpenAIEmbedder) Embed(text string) ([]float64, error) {
e.mu.Lock()
defer e.mu.Unlock()

// Construir request
reqBody := map[string]interface{}{
"input": text,
"model": "text-embedding-ada-002",
}

jsonBody, err := json.Marshal(reqBody)
if err != nil {
return nil, fmt.Errorf("error marshaling request: %w", err)
}

req, err := http.NewRequest("POST", e.baseURL+"/embeddings", bytes.NewReader(jsonBody))
if err != nil {
return nil, fmt.Errorf("error creating request: %w", err)
}

req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+e.apiKey)

resp, err := e.client.Do(req)
if err != nil {
return nil, fmt.Errorf("error executing request: %w", err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, fmt.Errorf("error reading response: %w", err)
}

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
}

var result struct {
Data []struct {
Embedding []float64 `json:"embedding"`
} `json:"data"`
}

if err := json.Unmarshal(body, &result); err != nil {
return nil, fmt.Errorf("error parsing response: %w", err)
}

if len(result.Data) == 0 {
return nil, fmt.Errorf("no embedding returned")
}

return result.Data[0].Embedding, nil
}

// MockEmbedder es un embedder de prueba que genera vectores aleatorios
type MockEmbedder struct{}

// Embed genera un embedding mock
func (m *MockEmbedder) Embed(text string) ([]float64, error) {
// Generar un vector pseudo-aleatorio basado en el texto
vec := make([]float64, 1536)
hash := int64(0)
for _, c := range text {
hash = hash*31 + int64(c)
}
for i := range vec {
vec[i] = float64((hash+int64(i))%100) / 100.0
}
return vec, nil
}
