package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Session representa una sesión OIDC
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Provider     Provider  `json:"provider"`
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// IsExpired verifica si la sesión expiró
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Flow gestiona el flujo OAuth2/OIDC
type Flow struct {
	manager  *Manager
	mu       sync.RWMutex
	states   map[string]stateEntry
	sessions map[string]*Session
}

type stateEntry struct {
	CreatedAt time.Time
	Redirect  string
	Nonce     string
}

// NewFlow crea un nuevo flow
func NewFlow(mgr *Manager) *Flow {
	f := &Flow{
		manager:  mgr,
		states:   make(map[string]stateEntry),
		sessions: make(map[string]*Session),
	}
	go f.cleanupStates()
	return f
}

// BuildAuthURL construye la URL de autorización
func (f *Flow) BuildAuthURL(provider Provider, redirectAfterLogin string) (string, error) {
	cfg, ok := f.manager.GetConfig(provider)
	if !ok {
		return "", errors.New("provider not registered")
	}
	ep, ok := f.manager.GetEndpoints(provider)
	if !ok {
		return "", errors.New("provider endpoints not found")
	}

	// Generar state + nonce
	state := generateState()
	nonce := generateState()

	f.mu.Lock()
	f.states[state] = stateEntry{
		CreatedAt: time.Now(),
		Redirect:  redirectAfterLogin,
		Nonce:     nonce,
	}
	f.mu.Unlock()

	// Construir URL
	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", cfg.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(cfg.Scopes, " "))
	params.Set("state", state)
	params.Set("nonce", nonce)

	return ep.AuthURL + "?" + params.Encode(), nil
}

// ValidateState valida el state OAuth y devuelve el redirect + nonce
func (f *Flow) ValidateState(state string) (redirect string, nonce string, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entry, exists := f.states[state]
	if !exists {
		return "", "", false
	}
	if time.Since(entry.CreatedAt) > 10*time.Minute {
		delete(f.states, state)
		return "", "", false
	}
	delete(f.states, state)
	return entry.Redirect, entry.Nonce, true
}

// CreateSession crea una nueva sesión
func (f *Flow) CreateSession(userID, email, name string, provider Provider, accessToken, refreshToken string, ttl time.Duration) *Session {
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

	f.mu.Lock()
	f.sessions[s.ID] = s
	f.mu.Unlock()

	return s
}

// GetSession devuelve una sesión por ID
func (f *Flow) GetSession(id string) (*Session, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	s, ok := f.sessions[id]
	if !ok || s.IsExpired() {
		return nil, false
	}
	return s, true
}

// RevokeSession revoca una sesión
func (f *Flow) RevokeSession(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sessions[id]; !ok {
		return false
	}
	delete(f.sessions, id)
	return true
}

// cleanupStates limpia los states antiguos
func (f *Flow) cleanupStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		f.mu.Lock()
		for k, v := range f.states {
			if time.Since(v.CreatedAt) > 15*time.Minute {
				delete(f.states, k)
			}
		}
		f.mu.Unlock()
	}
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
