package provider

import (
	"time"

	"github.com/lecodev-26/sentinelflow/internal/provider/model"
)

// CredentialSource indica de dónde vienen las credenciales
type CredentialSource string

const (
	CredentialSourceEnv   CredentialSource = "env"
	CredentialSourceVault CredentialSource = "vault"
	CredentialSourceFile  CredentialSource = "file"
	CredentialSourceK8s   CredentialSource = "k8s_secret"
	CredentialSourceNone  CredentialSource = "none"
)

// Credential describe cómo obtener las credenciales de un provider
type Credential struct {
	Source CredentialSource `json:"source"`
	Key    string           `json:"key,omitempty"`    // env var name o secret key
	Path   string           `json:"path,omitempty"`   // path en vault/secret manager
	Header string           `json:"header,omitempty"` // header HTTP (Authorization, x-api-key)
	Prefix string           `json:"prefix,omitempty"` // "Bearer " para OpenAI
}

// Endpoint describe un endpoint HTTP de un provider
type Endpoint struct {
	Chat       string `json:"chat"`
	Stream     string `json:"stream,omitempty"`
	Models     string `json:"models,omitempty"`
	Embeddings string `json:"embeddings,omitempty"`
}

// RateLimits contiene los límites del provider
type RateLimits struct {
	RequestsPerMinute int `json:"requests_per_minute,omitempty"`
	TokensPerMinute   int `json:"tokens_per_minute,omitempty"`
	ConcurrentReqs    int `json:"concurrent_requests,omitempty"`
}

// Descriptor describe completamente un provider
type Descriptor struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Version     string `json:"version,omitempty"`
	Region      string `json:"region,omitempty"`

	BaseURL    string     `json:"base_url"`
	Endpoints  Endpoint   `json:"endpoints"`
	Credential Credential `json:"credential"`

	Timeout    time.Duration `json:"timeout"`
	RateLimits RateLimits    `json:"rate_limits"`

	Models       []*model.Model     `json:"models,omitempty"`
	Capabilities []model.Capability `json:"capabilities,omitempty"`

	Enabled  bool `json:"enabled"`
	Priority int  `json:"priority,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HasCapability verifica si el provider tiene una capacidad
func (d *Descriptor) HasCapability(c model.Capability) bool {
	for _, cap := range d.Capabilities {
		if cap == c {
			return true
		}
	}
	return false
}

// SupportsModel verifica si el provider soporta un modelo
func (d *Descriptor) SupportsModel(modelID string) bool {
	for _, m := range d.Models {
		if m.ID == modelID {
			return true
		}
	}
	return false
}
