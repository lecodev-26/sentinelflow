package sentinelflow

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrStreamingRequiresChatStream = errors.New("sentinelflow: use ChatStream for streaming requests")

const defaultHTTPTimeout = 60 * time.Second

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

type StreamEvent struct {
	Data string
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
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     key,
		HTTPClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) request(ctx context.Context, method, path string, in interface{}) (*http.Request, error) {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("sentinelflow: encode request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func decodeAPIError(status int, raw []byte) error {
	var payload struct {
		Error Error `json:"error"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil {
		if payload.Error.Type != "" || payload.Error.Message != "" {
			payload.Error.Status = status
			return &payload.Error
		}
	}

	message := strings.TrimSpace(string(raw))
	if message == "" {
		message = http.StatusText(status)
	}
	return &Error{Status: status, Type: "http_error", Message: message}
}

func (c *Client) doJSON(ctx context.Context, method, path string, in, out interface{}) error {
	req, err := c.request(ctx, method, path, in)
	if err != nil {
		return err
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("sentinelflow: read response: %w", err)
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(resp.StatusCode, raw)
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("sentinelflow: decode response: %w", err)
	}
	return nil
}

func (c *Client) Chat(ctx context.Context, r ChatRequest) (Response, error) {
	var out Response
	if r.Stream {
		return out, ErrStreamingRequiresChatStream
	}
	return out, c.doJSON(ctx, http.MethodPost, "/v1/chat/completions", r, &out)
}

func (c *Client) Responses(ctx context.Context, model, input string) (Response, error) {
	var out Response
	payload := struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}{Model: model, Input: input}
	return out, c.doJSON(ctx, http.MethodPost, "/v1/responses", payload, &out)
}

func (c *Client) Models(ctx context.Context) ([]Model, error) {
	var out struct {
		Data []Model `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/v1/models", nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) ChatStream(ctx context.Context, r ChatRequest, onEvent func(StreamEvent) error) error {
	if onEvent == nil {
		return errors.New("sentinelflow: ChatStream callback is nil")
	}
	r.Stream = true

	req, err := c.request(ctx, http.MethodPost, "/v1/chat/completions", r)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		raw, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("sentinelflow: read error response: %w", readErr)
		}
		return decodeAPIError(resp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var data strings.Builder
	flush := func() error {
		if data.Len() == 0 {
			return nil
		}
		payload := strings.TrimSuffix(data.String(), "\n")
		data.Reset()
		if payload == "[DONE]" {
			return nil
		}
		return onEvent(StreamEvent{Data: payload})
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			value := strings.TrimPrefix(line, "data:")
			value = strings.TrimPrefix(value, " ")
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("sentinelflow: read stream: %w", err)
	}
	return flush()
}
