package benchmark

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"sync"
"time"
)

// OverheadBenchmark mide el overhead de SentinelFlow
type OverheadBenchmark struct {
DirectURL   string
ProxyURL    string
Concurrency int
Requests    int
}

// NewOverheadBenchmark crea un nuevo benchmark
func NewOverheadBenchmark(directURL, proxyURL string, concurrency, requests int) *OverheadBenchmark {
return &OverheadBenchmark{
DirectURL:   directURL,
ProxyURL:    proxyURL,
Concurrency: concurrency,
Requests:    requests,
}
}

// Run ejecuta el benchmark
func (b *OverheadBenchmark) Run() (Stats, Stats, error) {
// Benchmark directo
directStats, err := b.runDirect()
if err != nil {
return Stats{}, Stats{}, err
}

// Benchmark con proxy
proxyStats, err := b.runProxy()
if err != nil {
return Stats{}, Stats{}, err
}

return directStats, proxyStats, nil
}

func (b *OverheadBenchmark) runDirect() (Stats, error) {
metrics := NewMetrics()
var wg sync.WaitGroup
sem := make(chan struct{}, b.Concurrency)

for i := 0; i < b.Requests; i++ {
wg.Add(1)
sem <- struct{}{}
go func(id int) {
defer wg.Done()
defer func() { <-sem }()

start := time.Now()
success := b.makeDirectRequest(id)
metrics.Record(success, time.Since(start))
}(i)
}

wg.Wait()
metrics.Finish()
return metrics.GetStats(), nil
}

func (b *OverheadBenchmark) runProxy() (Stats, error) {
metrics := NewMetrics()
var wg sync.WaitGroup
sem := make(chan struct{}, b.Concurrency)

for i := 0; i < b.Requests; i++ {
wg.Add(1)
sem <- struct{}{}
go func(id int) {
defer wg.Done()
defer func() { <-sem }()

start := time.Now()
success := b.makeProxyRequest(id)
metrics.Record(success, time.Since(start))
}(i)
}

wg.Wait()
metrics.Finish()
return metrics.GetStats(), nil
}

func (b *OverheadBenchmark) makeDirectRequest(id int) bool {
body := map[string]interface{}{
"model": "gpt-3.5-turbo",
"messages": []map[string]string{
{"role": "user", "content": fmt.Sprintf("Hello %d", id)},
},
}
jsonBody, _ := json.Marshal(body)

resp, err := http.Post(b.DirectURL, "application/json", bytes.NewReader(jsonBody))
if err != nil {
return false
}
defer resp.Body.Close()

return resp.StatusCode == 200
}

func (b *OverheadBenchmark) makeProxyRequest(id int) bool {
body := map[string]interface{}{
"model": "gpt-3.5-turbo",
"messages": []map[string]string{
{"role": "user", "content": fmt.Sprintf("Hello %d", id)},
},
}
jsonBody, _ := json.Marshal(body)

resp, err := http.Post(b.ProxyURL, "application/json", bytes.NewReader(jsonBody))
if err != nil {
return false
}
defer resp.Body.Close()

return resp.StatusCode == 200
}
