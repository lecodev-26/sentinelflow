package sentinelflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL, APIKey string
	HTTPClient      *http.Client
}
type Error struct {
	Status        int
	Type, Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("sentinelflow: %d %s: %s", e.Status, e.Type, e.Message)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}
type Response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model,omitempty"`
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices,omitempty"`
	Output []map[string]interface{} `json:"output,omitempty"`
}
type Model struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Context  int    `json:"context"`
}

func New(baseURL, key string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: key, HTTPClient: http.DefaultClient}
}
func (c *Client) do(ctx context.Context, path string, in, out interface{}) error {
	b, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var x struct {
			Error Error `json:"error"`
		}
		_ = json.Unmarshal(raw, &x)
		x.Error.Status = resp.StatusCode
		return &x.Error
	}
	return json.Unmarshal(raw, out)
}
func (c *Client) Chat(ctx context.Context, r ChatRequest) (Response, error) {
	var out Response
	return out, c.do(ctx, "/v1/chat/completions", r, &out)
}
func (c *Client) Responses(ctx context.Context, model, input string) (Response, error) {
	var out Response
	return out, c.do(ctx, "/v1/responses", map[string]interface{}{"model": model, "input": input}, &out)
}
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/models", nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, e := c.HTTPClient.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	var out struct {
		Data []Model `json:"data"`
	}
	if e = json.NewDecoder(resp.Body).Decode(&out); e != nil {
		return nil, e
	}
	return out.Data, nil
}
