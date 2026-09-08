package security

import (
"sync"
"time"
)

// Quota define los límites de uso
type Quota struct {
RequestsPerMinute int `json:"requests_per_minute"`
TokensPerMinute   int `json:"tokens_per_minute"`
}

// QuotaManager gestiona las cuotas en tiempo real
type QuotaManager struct {
mu        sync.RWMutex
usage     map[string]*Usage // tenantID -> Usage
defaultQuota *Quota
}

// Usage representa el uso actual de un tenant
type Usage struct {
mu               sync.Mutex
Requests         int       `json:"requests"`
Tokens           int       `json:"tokens"`
WindowStart      time.Time `json:"window_start"`
LimitRequests    int       `json:"limit_requests"`
LimitTokens      int       `json:"limit_tokens"`
}

// NewQuotaManager crea un nuevo manager de cuotas
func NewQuotaManager(defaultQuota *Quota) *QuotaManager {
return &QuotaManager{
usage:        make(map[string]*Usage),
defaultQuota: defaultQuota,
}
}

// getUsage obtiene o crea el uso de un tenant
func (m *QuotaManager) getUsage(tenantID string, quota *Quota) *Usage {
m.mu.Lock()
defer m.mu.Unlock()

usage, exists := m.usage[tenantID]
if !exists {
usage = &Usage{
WindowStart:   time.Now(),
LimitRequests: quota.RequestsPerMinute,
LimitTokens:   quota.TokensPerMinute,
}
m.usage[tenantID] = usage
}

// Resetear si pasó la ventana
if time.Since(usage.WindowStart) > time.Minute {
usage.mu.Lock()
usage.Requests = 0
usage.Tokens = 0
usage.WindowStart = time.Now()
usage.LimitRequests = quota.RequestsPerMinute
usage.LimitTokens = quota.TokensPerMinute
usage.mu.Unlock()
}

return usage
}

// CheckAndRecord verifica y registra una petición
func (m *QuotaManager) CheckAndRecord(tenantID string, quota *Quota, tokens int) (bool, error) {
usage := m.getUsage(tenantID, quota)

usage.mu.Lock()
defer usage.mu.Unlock()

// Verificar límite de requests
if usage.Requests >= usage.LimitRequests {
return false, nil
}

// Verificar límite de tokens
if usage.Tokens+tokens > usage.LimitTokens {
return false, nil
}

// Registrar uso
usage.Requests++
usage.Tokens += tokens

return true, nil
}

// GetUsage devuelve el uso actual de un tenant
func (m *QuotaManager) GetUsage(tenantID string) *Usage {
m.mu.RLock()
defer m.mu.RUnlock()

return m.usage[tenantID]
}
