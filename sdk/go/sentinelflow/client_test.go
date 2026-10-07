package sentinelflow

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestNewConfiguresClient(t *testing.T) {
	c := New("https://example.com/", "k")
	if c.BaseURL != "https://example.com" {
		t.Fatalf("base URL = %q", c.BaseURL)
	}
	if c.HTTPClient == nil || c.HTTPClient.Timeout != defaultHTTPTimeout {
		t.Fatalf("unexpected HTTP client timeout: %#v", c.HTTPClient)
	}
}

func TestChat(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Error("missing auth")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"r1","object":"chat.completion","model":"m","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer s.Close()

	c := New(s.URL, "k")
	got, err := c.Chat(context.Background(), ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil || got.ID != "r1" {
		t.Fatalf("%v %#v", err, got)
	}
}

func TestChatRejectsStreaming(t *testing.T) {
	c := New("https://example.com", "k")
	_, err := c.Chat(context.Background(), ChatRequest{Stream: true})
	if !errors.Is(err, ErrStreamingRequiresChatStream) {
		t.Fatalf("expected streaming error, got %v", err)
	}
}

func TestAPIError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"auth_error","message":"bad key"}}`))
	}))
	defer s.Close()

	_, err := New(s.URL, "bad").Chat(context.Background(), ChatRequest{Model: "m"})
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if apiErr.Status != http.StatusUnauthorized || apiErr.Type != "auth_error" || apiErr.Message != "bad key" {
		t.Fatalf("unexpected API error: %#v", apiErr)
	}
}

func TestModelsError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"type":"forbidden","message":"denied"}}`))
	}))
	defer s.Close()

	_, err := New(s.URL, "k").Models(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Fatalf("unexpected error: %T %v", err, err)
	}
}

func TestModels(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"m1","provider":"p","context":8192}]}`))
	}))
	defer s.Close()

	models, err := New(s.URL, "k").Models(context.Background())
	if err != nil || len(models) != 1 || models[0].ID != "m1" {
		t.Fatalf("unexpected models: %v %#v", err, models)
	}
}

func TestChatStream(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Error("missing stream accept header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: first\n\ndata: second\n\ndata: [DONE]\n\n"))
	}))
	defer s.Close()

	var got []string
	err := New(s.URL, "k").ChatStream(context.Background(), ChatRequest{Model: "m"}, func(event StreamEvent) error {
		got = append(got, event.Data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "first,second" {
		t.Fatalf("events = %#v", got)
	}
}

func TestChatStreamCallbackError(t *testing.T) {
	want := errors.New("stop")
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: first\n\n"))
	}))
	defer s.Close()

	err := New(s.URL, "k").ChatStream(context.Background(), ChatRequest{Model: "m"}, func(StreamEvent) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected callback error, got %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	c := New("https://example.com", "k")
	c.HTTPClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := c.Chat(ctx, ChatRequest{Model: "m"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline, got %v", err)
	}
}
