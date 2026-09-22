package oidc

import (
"errors"
"sync"
)

// Provider representa un proveedor OIDC
type Provider string

const (
ProviderGoogle Provider = "google"
ProviderGitHub Provider = "github"
ProviderOkta   Provider = "okta"
ProviderAzure  Provider = "azure"
ProviderAuth0  Provider = "auth0"
)

// Config contiene la configuración de un provider OIDC
type Config struct {
Provider     Provider `json:"provider"`
ClientID     string   `json:"client_id"`
ClientSecret string   `json:"-"` // Nunca serializar
RedirectURL  string   `json:"redirect_url"`
Issuer       string   `json:"issuer"`
Scopes       []string `json:"scopes"`
}

// Endpoints contiene las URLs del provider
type Endpoints struct {
AuthURL     string `json:"auth_url"`
TokenURL    string `json:"token_url"`
UserInfoURL string `json:"user_info_url"`
JWKSURL     string `json:"jwks_url"`
}

// Manager gestiona los providers OIDC
type Manager struct {
mu       sync.RWMutex
configs  map[Provider]*Config
endpoints map[Provider]*Endpoints
}

// NewManager crea un nuevo manager OIDC
func NewManager() *Manager {
m := &Manager{
configs:   make(map[Provider]*Config),
endpoints: make(map[Provider]*Endpoints),
}
m.loadDefaultEndpoints()
return m
}

// RegisterProvider registra un provider OIDC
func (m *Manager) RegisterProvider(cfg *Config) error {
if cfg.ClientID == "" || cfg.ClientSecret == "" {
return errors.New("client_id and client_secret required")
}
if cfg.RedirectURL == "" {
return errors.New("redirect_url required")
}
if len(cfg.Scopes) == 0 {
cfg.Scopes = []string{"openid", "email", "profile"}
}

m.mu.Lock()
defer m.mu.Unlock()
m.configs[cfg.Provider] = cfg
return nil
}

// GetConfig devuelve la config de un provider
func (m *Manager) GetConfig(p Provider) (*Config, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
cfg, ok := m.configs[p]
return cfg, ok
}

// GetEndpoints devuelve los endpoints de un provider
func (m *Manager) GetEndpoints(p Provider) (*Endpoints, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
ep, ok := m.endpoints[p]
return ep, ok
}

// ListProviders lista los providers registrados
func (m *Manager) ListProviders() []Provider {
m.mu.RLock()
defer m.mu.RUnlock()
result := make([]Provider, 0, len(m.configs))
for p := range m.configs {
result = append(result, p)
}
return result
}

// loadDefaultEndpoints carga los endpoints conocidos
func (m *Manager) loadDefaultEndpoints() {
m.endpoints[ProviderGoogle] = &Endpoints{
AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
TokenURL:    "https://oauth2.googleapis.com/token",
UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
JWKSURL:     "https://www.googleapis.com/oauth2/v3/certs",
}

m.endpoints[ProviderGitHub] = &Endpoints{
AuthURL:     "https://github.com/login/oauth/authorize",
TokenURL:    "https://github.com/login/oauth/access_token",
UserInfoURL: "https://api.github.com/user",
JWKSURL:     "",
}

m.endpoints[ProviderOkta] = &Endpoints{
AuthURL:     "", // se rellena con issuer
TokenURL:    "",
UserInfoURL: "",
JWKSURL:     "",
}
}
