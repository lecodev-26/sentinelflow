package e2e

import (
"bufio"
"bytes"
"encoding/json"
"net/http"
"strings"
"testing"
"time"
)

func TestStreaming(t *testing.T) {
t.Skip("E2E tests require proxy running")

url := "http://localhost:8080/v1/chat/completions"

body := map[string]interface{}{
"model": "gpt-3.5-turbo",
"messages": []map[string]string{
{"role": "user", "content": "Tell me a short story"},
},
"stream": true,
}
jsonBody, _ := json.Marshal(body)

resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
if err != nil {
t.Skip("Proxy not running, skipping test")
return
}
defer resp.Body.Close()

// Verificar que es SSE
contentType := resp.Header.Get("Content-Type")
if !strings.Contains(contentType, "text/event-stream") {
t.Errorf("Expected text/event-stream, got %s", contentType)
}

// Leer eventos
scanner := bufio.NewScanner(resp.Body)
var chunks int
timeout := time.After(10 * time.Second)

for {
select {
case <-timeout:
t.Fatal("Timeout waiting for events")
default:
if !scanner.Scan() {
return
}
line := scanner.Text()
if strings.HasPrefix(line, "data:") {
chunks++
}
if chunks > 0 {
break
}
}
}

t.Logf("Received %d chunks", chunks)
}
