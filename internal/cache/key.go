package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// KeyVersion permite invalidar todo el caché cuando cambie el esquema
const KeyVersion = "v1"

// CacheKey contiene los componentes de la clave
type CacheKey struct {
	Version     string
	TenantID    string
	ProjectID   string
	Model       string
	Provider    string
	RequestHash string
}

// NewCacheKey construye una clave a partir del request
func NewCacheKey(tenantID, projectID, model, provider string, request interface{}) *CacheKey {
	// Serializar request para hashear
	data, _ := json.Marshal(request)
	hash := sha256.Sum256(data)

	return &CacheKey{
		Version:     KeyVersion,
		TenantID:    tenantID,
		ProjectID:   projectID,
		Model:       model,
		Provider:    provider,
		RequestHash: hex.EncodeToString(hash[:16]), // 16 bytes = 32 hex chars
	}
}

// String devuelve la clave en formato seguro
func (k *CacheKey) String() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		k.Version,
		k.TenantID,
		k.ProjectID,
		k.Model,
		k.Provider,
		k.RequestHash,
	)
}

// TenantPrefix devuelve el prefijo para invalidar todo un tenant
func TenantPrefix(tenantID string) string {
	return fmt.Sprintf("%s:%s:", KeyVersion, tenantID)
}

// ProjectPrefix devuelve el prefijo para invalidar un proyecto
func ProjectPrefix(tenantID, projectID string) string {
	return fmt.Sprintf("%s:%s:%s:", KeyVersion, tenantID, projectID)
}
