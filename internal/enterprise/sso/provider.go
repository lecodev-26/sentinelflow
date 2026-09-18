package sso

import (
"context"
"crypto/rand"
"encoding/base64"
"errors"
"fmt"
"sync"
"time"
)

// Provider representa un proveedor SSO
type Provider string

const (
ProviderGoogle Provider = "google"
ProviderGitHub Provider = "github"
ProviderOkta   Provider = "okta"
ProviderAzure  Provider = "azure"
ProviderAuth0  Provider = "auth0"
ProviderCustom Provider = "custom"
)

// Config contiene la configuración de un provider SSO
type Config struct {
Provider     Provider `json:"provider"`
ClientID     string   `json:"client_id"`
ClientSecret string   `json:"client_secret"`
RedirectURL  string   `json:"redirect_url"`
Issuer       string   `json:"issuer"`
Scopes       []string `json:"scopes"`
}

// Session representa una sesión SSO
type Session struct {
ID           string    `json:"id"`
UserID       string    `json:"user_id"`
Email        string    `json:"email"`
Name         string    `json:"name"`
Provider     Provider  `json:"provider"`
AccessToken  string    `json:"-"` // Nunca serializar
RefreshToken string    `json:"-"`
ExpiresAt    time.Time `json:"expires_at"`
CreatedAt    time.Time `json:"created_at"`
}

// IsExpired verifica si la sesión expiró
func (s *Session) IsExpired() bool {
return time.Now().After(s.ExpiresAt)
}

// Manager gestiona el SSO
type Manager struct {
mu       sync.RWMutex
configs  map[Provider]*Config
sessions map[string]*Session
}

// NewManager crea un nuevo manager SSO
func NewManager() *Manager {
return &Manager{
configs:  make(map[Provider]*Config),
sessions: make(map[string]*Session),
}
}

// RegisterProvider registra un provider SSO
func (m *Manager) RegisterProvider(cfg *Config) error {
if cfg.ClientID == "" || cfg.ClientSecret == "" {
return errors.New("client_id and client_secret are required")
}
if cfg.RedirectURL == "" {
return errors.New("redirect_url is required")
}
if len(cfg.Scopes) == 0 {
cfg.Scopes = []string{"openid", "email", "profile"}
}

m.mu.Lock()
defer m.mu.Unlock()
m.configs[cfg.Provider] = cfg
return nil
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

// GetAuthURL devuelve la URL de autorización
func (m *Manager) GetAuthURL(provider Provider, state string) (string, error) {
m.mu.RLock()
cfg, exists := m.configs[provider]
m.mu.RUnlock()
if !exists {
return "", fmt.Errorf("provider not registered: %s", provider)
}

// En producción, construir URL específica por provider
switch provider {
case ProviderGoogle:
return fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
cfg.ClientID, cfg.RedirectURL, joinScopes(cfg.Scopes), state), nil
case ProviderGitHub:
return fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
cfg.ClientID, cfg.RedirectURL, joinScopes(cfg.Scopes), state), nil
default:
return "", fmt.Errorf("provider not supported: %s", provider)
}
}

// CreateSession crea una nueva sesión
func (m *Manager) CreateSession(userID, email, name string, provider Provider, accessToken, refreshToken string, ttl time.Duration) *Session {
s := &Session{
ID:           generateID(),
UserID:       userID,
Email:        email,
Name:         name,
Provider:     provider,
AccessToken:  accessToken,
RefreshToken: refreshToken,
ExpiresAt:    time.Now().Add(ttl),
CreatedAt:    time.Now(),
}

m.mu.Lock()
m.sessions[s.ID] = s
m.mu.Unlock()

return s
}

// GetSession devuelve una sesión por ID
func (m *Manager) GetSession(id string) (*Session, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
s, exists := m.sessions[id]
if exists && s.IsExpired() {
return nil, false
}
return s, exists
}

// RevokeSession revoca una sesión
func (m *Manager) RevokeSession(id string) bool {
m.mu.Lock()
defer m.mu.Unlock()
if _, exists := m.sessions[id]; !exists {
return false
}
delete(m.sessions, id)
return true
}

func joinScopes(scopes []string) string {
result := ""
for i, s := range scopes {
if i > 0 {
result += " "
}
result += s
}
return result
}

func generateID() string {
b := make([]byte, 16)
rand.Read(b)
return base64.URLEncoding.EncodeToString(b)
}

var _ = context.Background
