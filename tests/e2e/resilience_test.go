package e2e

import (
"bytes"
"encoding/json"
"net/http"
"testing"
"time"
)

func TestCircuitBreaker(t *testing.T) {
t.Skip("E2E tests require proxy running")

url := "http://localhost:8080/v1/chat/completions"

// Provocar múltiples fallos para abrir el circuito
for i := 0; i < 10; i++ {
body := map[string]interface{}{
"model": "fail-model",
"messages": []map[string]string{
{"role": "user", "content": "Hello"},
},
}
jsonBody, _ := json.Marshal(body)

resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
if err != nil {
t.Skip("Proxy not running, skipping test")
return
}
resp.Body.Close()

// Después de varios fallos, debería abrir el circuito
if resp.StatusCode == http.StatusServiceUnavailable {
t.Logf("Circuit breaker opened after %d attempts", i+1)
return
}
}
}
