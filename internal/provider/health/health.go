package health

import (
"context"
"sync"
"time"

"github.com/lecodev-26/sentinelflow/internal/provider"
)

// Status representa el estado de un proveedor
type Status string

const (
StatusUnknown    Status = "unknown"
StatusHealthy    Status = "healthy"
StatusDegraded   Status = "degraded"
StatusUnhealthy  Status = "unhealthy"
StatusDisabled   Status = "disabled"
)

// ProviderHealth contiene el estado de un proveedor
type ProviderHealth struct {
Name            string        `json:"name"`
Status          Status        `json:"status"`
LastCheck       time.Time     `json:"last_check"`
LastError       string        `json:"last_error,omitempty"`
ConsecutiveFails int          `json:"consecutive_fails"`
Latency         time.Duration `json:"latency"`
}

// Monitor monitorea la salud de los proveedores
type Monitor struct {
mu        sync.RWMutex
health    map[string]*ProviderHealth
providers map[string]provider.Provider
interval  time.Duration
timeout   time.Duration
stopCh    chan struct{}
}

// NewMonitor crea un nuevo monitor
func NewMonitor(interval, timeout time.Duration) *Monitor {
return &Monitor{
health:    make(map[string]*ProviderHealth),
providers: make(map[string]provider.Provider),
interval:  interval,
timeout:   timeout,
stopCh:    make(chan struct{}),
}
}

// Register registra un proveedor para monitorear
func (m *Monitor) Register(p provider.Provider) {
m.mu.Lock()
defer m.mu.Unlock()

m.providers[p.Name()] = p
m.health[p.Name()] = &ProviderHealth{
Name:   p.Name(),
Status: StatusUnknown,
}
}

// Start inicia el monitoreo en background
func (m *Monitor) Start(ctx context.Context) {
go func() {
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
}()
}

// Stop detiene el monitoreo
func (m *Monitor) Stop() {
close(m.stopCh)
}

func (m *Monitor) checkAll(ctx context.Context) {
m.mu.RLock()
providers := make([]provider.Provider, 0, len(m.providers))
for _, p := range m.providers {
providers = append(providers, p)
}
m.mu.RUnlock()

var wg sync.WaitGroup
for _, p := range providers {
wg.Add(1)
go func(p provider.Provider) {
defer wg.Done()
m.checkProvider(ctx, p)
}(p)
}
wg.Wait()
}

func (m *Monitor) checkProvider(ctx context.Context, p provider.Provider) {
checkCtx, cancel := context.WithTimeout(ctx, m.timeout)
defer cancel()

start := time.Now()
err := p.Health(checkCtx)
latency := time.Since(start)

m.mu.Lock()
defer m.mu.Unlock()

h, exists := m.health[p.Name()]
if !exists {
h = &ProviderHealth{Name: p.Name()}
m.health[p.Name()] = h
}

h.LastCheck = time.Now()
h.Latency = latency

if err != nil {
h.ConsecutiveFails++
h.LastError = err.Error()

switch {
case h.ConsecutiveFails >= 5:
h.Status = StatusUnhealthy
case h.ConsecutiveFails >= 2:
h.Status = StatusDegraded
default:
h.Status = StatusDegraded
}
} else {
h.ConsecutiveFails = 0
h.LastError = ""
h.Status = StatusHealthy
}
}

// Get devuelve el estado de un proveedor
func (m *Monitor) Get(name string) (*ProviderHealth, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
h, exists := m.health[name]
return h, exists
}

// GetAll devuelve el estado de todos los proveedores
func (m *Monitor) GetAll() []*ProviderHealth {
m.mu.RLock()
defer m.mu.RUnlock()

result := make([]*ProviderHealth, 0, len(m.health))
for _, h := range m.health {
result = append(result, h)
}
return result
}

// IsHealthy verifica si un proveedor está saludable
func (m *Monitor) IsHealthy(name string) bool {
h, exists := m.Get(name)
if !exists {
return false
}
return h.Status == StatusHealthy
}
