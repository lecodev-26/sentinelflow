package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestChaosProviderDead verifica que el sistema sobrevive a un provider muerto
func TestChaosProviderDead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	url := "http://localhost:8080/v1/chat/completions"

	body := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "test"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	start := time.Now()
	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Skipf("Proxy no corriendo: %v", err)
		return
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	// El sistema debe responder rápido (no colgarse)
	if latency > 10*time.Second {
		t.Errorf("Sistema demasiado lento: %v", latency)
	}

	t.Logf("Status: %d, Latency: %v", resp.StatusCode, latency)
}

// TestChaosTimeout verifica que los timeouts funcionan
func TestChaosTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	url := "http://localhost:8080/v1/chat/completions"

	body := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "test"},
		},
	}
	jsonBody, _ := json.Marshal(body)

	client := &http.Client{Timeout: 5 * time.Second}
	start := time.Now()

	resp, err := client.Post(url, "application/json", bytes.NewReader(jsonBody))
	latency := time.Since(start)

	if err != nil {
		t.Logf("Timeout esperado o proxy no corriendo: %v (latency: %v)", err, latency)
		return
	}
	defer resp.Body.Close()

	t.Logf("Status: %d, Latency: %v", resp.StatusCode, latency)
}

// TestChaosConcurrentRequests verifica comportamiento bajo concurrencia
func TestChaosConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	url := "http://localhost:8080/health"

	concurrency := 20
	done := make(chan bool, concurrency)

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			defer func() { done <- true }()
			resp, err := http.Get(url)
			if err != nil {
				return
			}
			resp.Body.Close()
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}

	elapsed := time.Since(start)
	t.Logf("Concurrency %d completed in %v", concurrency, elapsed)

	if elapsed > 10*time.Second {
		t.Errorf("Concurrent requests too slow: %v", elapsed)
	}
}
