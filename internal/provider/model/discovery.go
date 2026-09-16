package model

import (
"context"
"encoding/json"
"fmt"
"io"
"net/http"
"time"
)

// Discovery descubre modelos de un provider
type Discovery struct {
client *http.Client
}

// NewDiscovery crea un nuevo discovery service
func NewDiscovery() *Discovery {
return &Discovery{
client: &http.Client{Timeout: 10 * time.Second},
}
}

// DiscoverOpenAI descubre modelos de OpenAI
func (d *Discovery) DiscoverOpenAI(ctx context.Context, apiKey, baseURL string) ([]*Model, error) {
if baseURL == "" {
baseURL = "https://api.openai.com/v1"
}

req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
if err != nil {
return nil, err
}
req.Header.Set("Authorization", "Bearer "+apiKey)

resp, err := d.client.Do(req)
if err != nil {
return nil, fmt.Errorf("error fetching models: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("API error: %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var result struct {
Data []struct {
ID      string `json:"id"`
Created int64  `json:"created"`
OwnedBy string `json:"owned_by"`
} `json:"data"`
}

if err := json.Unmarshal(body, &result); err != nil {
return nil, err
}

var models []*Model
for _, m := range result.Data {
model := &Model{
ID:           m.ID,
Name:         m.ID,
Provider:     "openai",
Capabilities: DefaultCapabilities("openai"),
Pricing:      Pricing{}, // se rellena del catálogo estático
Limits:       Limits{},
DiscoveredAt: time.Now(),
}
models = append(models, model)
}

return models, nil
}

// DiscoverAnthropic descubre modelos de Anthropic
// Nota: Anthropic no tiene endpoint /models público, usamos catálogo estático
func (d *Discovery) DiscoverAnthropic(ctx context.Context, apiKey string) ([]*Model, error) {
// Retornar modelos conocidos del catálogo por defecto
var result []*Model
for _, m := range DefaultCatalog() {
if m.Provider == "anthropic" {
result = append(result, m)
}
}
return result, nil
}

// DiscoverOllama descubre modelos de Ollama local
func (d *Discovery) DiscoverOllama(ctx context.Context, baseURL string) ([]*Model, error) {
if baseURL == "" {
baseURL = "http://localhost:11434"
}

req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
if err != nil {
return nil, err
}

resp, err := d.client.Do(req)
if err != nil {
return nil, fmt.Errorf("ollama not available: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
return nil, fmt.Errorf("ollama API error: %d", resp.StatusCode)
}

body, err := io.ReadAll(resp.Body)
if err != nil {
return nil, err
}

var result struct {
Models []struct {
Name       string `json:"name"`
ModifiedAt string `json:"modified_at"`
Size       int64  `json:"size"`
} `json:"models"`
}

if err := json.Unmarshal(body, &result); err != nil {
return nil, err
}

var models []*Model
for _, m := range result.Models {
model := &Model{
ID:           m.Name,
Name:         m.Name,
Provider:     "local-llama",
Capabilities: DefaultCapabilities("local-llama"),
Pricing:      Pricing{InputPer1M: 0, OutputPer1M: 0},
Limits:       Limits{MaxContextTokens: 8192, MaxOutputTokens: 4096},
DiscoveredAt: time.Now(),
Metadata: map[string]interface{}{
"size_bytes":  m.Size,
"modified_at": m.ModifiedAt,
},
}
models = append(models, model)
}

return models, nil
}

// DiscoverAll descubre modelos de todos los providers
func (d *Discovery) DiscoverAll(ctx context.Context, openAIKey, anthropicKey, ollamaURL string) []*Model {
var all []*Model

if openAIKey != "" {
if models, err := d.DiscoverOpenAI(ctx, openAIKey, ""); err == nil {
all = append(all, models...)
}
}

if anthropicKey != "" {
if models, err := d.DiscoverAnthropic(ctx, anthropicKey); err == nil {
all = append(all, models...)
}
}

if ollamaURL != "" {
if models, err := d.DiscoverOllama(ctx, ollamaURL); err == nil {
all = append(all, models...)
}
}

return all
}
