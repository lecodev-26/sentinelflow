package security

import (
"crypto/rand"
"encoding/hex"
"sync"
"time"
)

// APIKey representa una clave de API
type APIKey struct {
ID        string    `json:"id"`
Key       string    `json:"key"`
TenantID  string    `json:"tenant_id"`
CreatedAt time.Time `json:"created_at"`
LastUsed  time.Time `json:"last_used"`
Active    bool      `json:"active"`
}

// APIKeyManager gestiona las claves de API
type APIKeyManager struct {
mu       sync.RWMutex
keys     map[string]*APIKey // key -> APIKey
tenantID string             // tenant por defecto
}

// NewAPIKeyManager crea un nuevo manager
func NewAPIKeyManager(tenantID string) *APIKeyManager {
return &APIKeyManager{
keys:     make(map[string]*APIKey),
tenantID: tenantID,
}
}

// GenerateKey genera una nueva clave de API
func (m *APIKeyManager) GenerateKey() (string, error) {
bytes := make([]byte, 32)
if _, err := rand.Read(bytes); err != nil {
return "", err
}
return "sf_" + hex.EncodeToString(bytes), nil
}

// CreateKey crea una nueva clave de API
func (m *APIKeyManager) CreateKey(tenantID string) (*APIKey, error) {
key, err := m.GenerateKey()
if err != nil {
return nil, err
}

apiKey := &APIKey{
ID:        key[:8],
Key:       key,
TenantID:  tenantID,
CreatedAt: time.Now(),
Active:    true,
}

m.mu.Lock()
defer m.mu.Unlock()
m.keys[key] = apiKey

return apiKey, nil
}

// ValidateKey valida una clave de API
func (m *APIKeyManager) ValidateKey(key string) (*APIKey, bool) {
m.mu.RLock()
defer m.mu.RUnlock()

apiKey, exists := m.keys[key]
if !exists {
return nil, false
}

if !apiKey.Active {
return nil, false
}

// Actualizar último uso
apiKey.LastUsed = time.Now()
return apiKey, true
}

// RevokeKey revoca una clave de API
func (m *APIKeyManager) RevokeKey(key string) bool {
m.mu.Lock()
defer m.mu.Unlock()

apiKey, exists := m.keys[key]
if !exists {
return false
}

apiKey.Active = false
return true
}

// ListKeys lista todas las claves de un tenant
func (m *APIKeyManager) ListKeys(tenantID string) []*APIKey {
m.mu.RLock()
defer m.mu.RUnlock()

var result []*APIKey
for _, apiKey := range m.keys {
if apiKey.TenantID == tenantID && apiKey.Active {
result = append(result, apiKey)
}
}
return result
}
