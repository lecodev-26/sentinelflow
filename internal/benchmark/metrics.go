package benchmark

import (
"sync"
"time"
)

// Metrics almacena métricas de rendimiento
type Metrics struct {
mu            sync.Mutex
TotalRequests int
SuccessCount  int
FailCount     int
TotalLatency  time.Duration
MinLatency    time.Duration
MaxLatency    time.Duration
StartTime     time.Time
EndTime       time.Time
}

// NewMetrics crea un nuevo collector de métricas
func NewMetrics() *Metrics {
return &Metrics{
MinLatency: time.Hour,
StartTime:  time.Now(),
}
}

// Record registra una petición
func (m *Metrics) Record(success bool, latency time.Duration) {
m.mu.Lock()
defer m.mu.Unlock()

m.TotalRequests++
if success {
m.SuccessCount++
} else {
m.FailCount++
}
m.TotalLatency += latency
if latency < m.MinLatency {
m.MinLatency = latency
}
if latency > m.MaxLatency {
m.MaxLatency = latency
}
}

// Finish finaliza la medición
func (m *Metrics) Finish() {
m.mu.Lock()
defer m.mu.Unlock()
m.EndTime = time.Now()
}

// GetStats devuelve estadísticas
func (m *Metrics) GetStats() Stats {
m.mu.Lock()
defer m.mu.Unlock()

avg := time.Duration(0)
if m.TotalRequests > 0 {
avg = m.TotalLatency / time.Duration(m.TotalRequests)
}

duration := m.EndTime.Sub(m.StartTime)
throughput := float64(m.TotalRequests) / duration.Seconds()

return Stats{
TotalRequests: m.TotalRequests,
SuccessCount:  m.SuccessCount,
FailCount:     m.FailCount,
AvgLatency:    avg,
MinLatency:    m.MinLatency,
MaxLatency:    m.MaxLatency,
Duration:      duration,
Throughput:    throughput,
SuccessRate:   float64(m.SuccessCount) / float64(m.TotalRequests) * 100,
}
}

// Stats contiene estadísticas de rendimiento
type Stats struct {
TotalRequests int
SuccessCount  int
FailCount     int
AvgLatency    time.Duration
MinLatency    time.Duration
MaxLatency    time.Duration
Duration      time.Duration
Throughput    float64
SuccessRate   float64
}
