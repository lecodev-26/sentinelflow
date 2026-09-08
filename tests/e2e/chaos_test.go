package e2e

import (
"bytes"
"encoding/json"
"net/http"
"testing"
"time"
)

func TestChaosProviderDead(t *testing.T) {
t.Skip("E2E tests require proxy running")

url := "http://localhost:8080/v1/chat/completions"

// Probar que el sistema sobrevive a un provider muerto
body := map[string]interface{}{
"model": "openai-dead",
"messages": []map[string]string{
{"role": "user", "content": "Hello"},
},
}
jsonBody, _ := json.Marshal(body)

start := time.Now()
resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
if err != nil {
t.Skip("Proxy not running, skipping test")
return
}
defer resp.Body.Close()

latency := time.Since(start)

// Debería fallar rápido o hacer failover
if latency > 5*time.Second {
t.Logf("WARNING: Slow failover: %v", latency)
}

t.Logf("Status: %d, Latency: %v", resp.StatusCode, latency)
}

func TestChaosProviderSlow(t *testing.T) {
t.Skip("E2E tests require proxy running")

url := "http://localhost:8080/v1/chat/completions"

body := map[string]interface{}{
"model": "openai-slow",
"messages": []map[string]string{
{"role": "user", "content": "Hello"},
},
}
jsonBody, _ := json.Marshal(body)

start := time.Now()
resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
if err != nil {
t.Skip("Proxy not running, skipping test")
return
}
defer resp.Body.Close()

latency := time.Since(start)

// Si el provider es lento, debería hacer failover
t.Logf("Status: %d, Latency: %v", resp.StatusCode, latency)
}
