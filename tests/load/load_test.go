package load

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"sync"
"testing"
"time"
)

type result struct {
success bool
latency time.Duration
err     error
}

func makeRequest(url string, id int) result {
start := time.Now()
body := map[string]interface{}{
"model": "gpt-3.5-turbo",
"messages": []map[string]string{
{"role": "user", "content": fmt.Sprintf("Hello from request %d", id)},
},
}
jsonBody, _ := json.Marshal(body)

resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
if err != nil {
return result{success: false, latency: time.Since(start), err: err}
}
defer resp.Body.Close()

return result{success: resp.StatusCode == 200, latency: time.Since(start)}
}

func TestLoadSimulation(t *testing.T) {
t.Skip("Solo para pruebas manuales con el proxy corriendo")

url := "http://localhost:8080/v1/chat/completions"
requests := 500

var wg sync.WaitGroup
results := make(chan result, requests)

start := time.Now()

for i := 0; i < requests; i++ {
wg.Add(1)
go func(id int) {
defer wg.Done()
results <- makeRequest(url, id)
}(i)
}

wg.Wait()
close(results)

duration := time.Since(start)

var total, success, failed int
var totalLatency time.Duration

for res := range results {
total++
if res.success {
success++
} else {
failed++
}
totalLatency += res.latency
}

avgLatency := totalLatency / time.Duration(total)
successRate := float64(success) / float64(total) * 100

fmt.Printf("\n📊 Load Test Results:\n")
fmt.Printf("   Total requests: %d\n", total)
fmt.Printf("   Success: %d\n", success)
fmt.Printf("   Failed: %d\n", failed)
fmt.Printf("   Success rate: %.2f%%\n", successRate)
fmt.Printf("   Avg latency: %v\n", avgLatency)
fmt.Printf("   Duration: %v\n", duration)
fmt.Printf("   Throughput: %.2f req/s\n", float64(total)/duration.Seconds())

if successRate < 99.0 {
t.Errorf("Success rate too low: %.2f%%", successRate)
}
}
