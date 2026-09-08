package security

import (
"sync"
)

// Tenant representa un cliente/tenant
type Tenant struct {
ID           string                 `json:"id"`
Name         string                 `json:"name"`
Config       map[string]interface{} `json:"config"`
AllowedModels []string              `json:"allowed_models"`
Quotas       *Quota                 `json:"quotas"`
}

// TenantManager gestiona los tenants
type TenantManager struct {
mu       sync.RWMutex
tenants  map[string]*Tenant
defaultQuota *Quota
}

// NewTenantManager crea un nuevo manager de tenants
func NewTenantManager() *TenantManager {
return &TenantManager{
tenants:  make(map[string]*Tenant),
defaultQuota: &Quota{
RequestsPerMinute: 100,
TokensPerMinute:   10000,
},
}
}

// CreateTenant crea un nuevo tenant
func (m *TenantManager) CreateTenant(id, name string) *Tenant {
m.mu.Lock()
defer m.mu.Unlock()

tenant := &Tenant{
ID:           id,
Name:         name,
Config:       make(map[string]interface{}),
AllowedModels: []string{"gpt-3.5-turbo", "gpt-4", "claude-3"},
Quotas:       m.defaultQuota,
}

m.tenants[id] = tenant
return tenant
}

// GetTenant devuelve un tenant por ID
func (m *TenantManager) GetTenant(id string) (*Tenant, bool) {
m.mu.RLock()
defer m.mu.RUnlock()

tenant, exists := m.tenants[id]
return tenant, exists
}

// UpdateQuota actualiza las cuotas de un tenant
func (m *TenantManager) UpdateQuota(tenantID string, quota *Quota) bool {
m.mu.Lock()
defer m.mu.Unlock()

tenant, exists := m.tenants[tenantID]
if !exists {
return false
}

tenant.Quotas = quota
return true
}

// AllowModel verifica si un modelo está permitido para un tenant
func (m *TenantManager) AllowModel(tenantID, model string) bool {
m.mu.RLock()
defer m.mu.RUnlock()

tenant, exists := m.tenants[tenantID]
if !exists {
return false
}

for _, allowed := range tenant.AllowedModels {
if allowed == model {
return true
}
}
return false
}
