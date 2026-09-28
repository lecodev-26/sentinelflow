package sentinelflow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChat(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Error("missing auth")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"r1","object":"chat.completion","model":"m","choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer s.Close()
	c := New(s.URL, "k")
	got, e := c.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}})
	if e != nil || got.ID != "r1" {
		t.Fatalf("%v %#v", e, got)
	}
}
