package v3

import (
"context"
"sync"
"time"
)

// ProviderHealth contiene el estado de salud de un provider
type ProviderHealth struct {
Provider          string        `json:"provider"`
Status            HealthStatus  `json:"status"`
LastCheck         time.Time     `json:"last_check"`
LastError         string        `json:"last_error,omitempty"`
ConsecutiveFails  int           `json:"consecutive_fails"`
ConsecutiveOK     int           `json:"consecutive_ok"`
AvgLatency        time.Duration `json:"avg_latency"`
LastLatency       time.Duration `json:"last_latency"`
}

// HealthMonitor monitoriza la salud de los providers
type HealthMonitor struct {
mu       sync.RWMutex
health   map[string]*ProviderHealth
registry *Registry
interval time.Duration
timeout  time.Duration
stopCh   chan struct{}
}

// NewHealthMonitor crea un nuevo monitor
func NewHealthMonitor(registry *Registry, interval, timeout time.Duration) *HealthMonitor {
if interval == 0 {
interval = 30 * time.Second
}
if timeout == 0 {
timeout = 5 * time.Second
}

return &HealthMonitor{
health:   make(map[string]*ProviderHealth),
registry: registry,
interval: interval,
timeout:  timeout,
stopCh:   make(chan struct{}),
}
}

// Start inicia el monitor
func (m *HealthMonitor) Start(ctx context.Context) {
// Inicializar entradas
for _, p := range m.registry.GetAll() {
m.health[p.ID()] = &ProviderHealth{
Provider: p.ID(),
Status:   HealthUnknown,
}
}

go m.loop(ctx)
}

// Stop detiene el monitor
func (m *HealthMonitor) Stop() {
close(m.stopCh)
}

func (m *HealthMonitor) loop(ctx context.Context) {
ticker := time.NewTicker(m.interval)
defer ticker.Stop()

// Primera comprobación inmediata
m.checkAll(ctx)

for {
select {
case <-ticker.C:
m.checkAll(ctx)
case <-m.stopCh:
return
case <-ctx.Done():
return
}
}
}

func (m *HealthMonitor) checkAll(ctx context.Context) {
providers := m.registry.GetAll()

var wg sync.WaitGroup
for _, p := range providers {
wg.Add(1)
go func(p Provider) {
defer wg.Done()
m.checkProvider(ctx, p)
}(p)
}

wg.Wait()
}

func (m *HealthMonitor) checkProvider(ctx context.Context, p Provider) {
checkCtx, cancel := context.WithTimeout(ctx, m.timeout)
defer cancel()

start := time.Now()
err := p.Health(checkCtx)
latency := time.Since(start)

m.mu.Lock()
defer m.mu.Unlock()

h, exists := m.health[p.ID()]
if !exists {
h = &ProviderHealth{Provider: p.ID()}
m.health[p.ID()] = h
}

h.LastCheck = time.Now()
h.LastLatency = latency

// Media móvil simple
if h.AvgLatency == 0 {
h.AvgLatency = latency
} else {
h.AvgLatency = (h.AvgLatency*4 + latency) / 5
}

if err != nil {
h.ConsecutiveFails++
h.ConsecutiveOK = 0
h.LastError = err.Error()

switch {
case h.ConsecutiveFails >= 5:
h.Status = HealthUnhealthy
case h.ConsecutiveFails >= 2:
h.Status = HealthDegraded
default:
h.Status = HealthDegraded
}
} else {
h.ConsecutiveOK++
h.ConsecutiveFails = 0
h.LastError = ""
h.Status = HealthHealthy
}
}

// Get devuelve el estado de un provider
func (m *HealthMonitor) Get(providerID string) (*ProviderHealth, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
h, exists := m.health[providerID]
return h, exists
}

// GetAll devuelve el estado de todos los providers
func (m *HealthMonitor) GetAll() []*ProviderHealth {
m.mu.RLock()
defer m.mu.RUnlock()
result := make([]*ProviderHealth, 0, len(m.health))
for _, h := range m.health {
result = append(result, h)
}
return result
}

// IsHealthy devuelve true si el provider está saludable
func (m *HealthMonitor) IsHealthy(providerID string) bool {
h, exists := m.Get(providerID)
if !exists {
return false
}
return h.Status == HealthHealthy
}
