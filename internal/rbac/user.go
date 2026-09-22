package rbac

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	OrgID     string    `json:"org_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	APIKeys   []string  `json:"api_keys"`
	Active    bool      `json:"active"`
}

type APIKey struct {
	ID          string    `json:"id"`
	KeyHash     string    `json:"-"`
	KeyPrefix   string    `json:"key_prefix"` // primeros 8 chars para identificación
	Name        string    `json:"name"`
	UserID      string    `json:"user_id"`
	OrgID       string    `json:"org_id"`
	ProjectID   string    `json:"project_id"`
	Scopes      []Scope   `json:"scopes"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	ExpiresAt   time.Time `json:"expires_at"`
	RotatedFrom string    `json:"rotated_from,omitempty"`
	Active      bool      `json:"active"`
}

type UserManager struct {
	mu      sync.RWMutex
	users   map[string]*User
	apiKeys map[string]*APIKey
	orgMgr  *OrganizationManager
}

func NewUserManager(orgMgr *OrganizationManager) *UserManager {
	return &UserManager{
		users:   make(map[string]*User),
		apiKeys: make(map[string]*APIKey),
		orgMgr:  orgMgr,
	}
}

func (m *UserManager) CreateUser(email, name string, role Role, orgID string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.orgMgr.GetOrganization(orgID); !exists {
		return nil, ErrOrganizationNotFound
	}

	user := &User{
		ID:        generateID("user"),
		Email:     email,
		Name:      name,
		Role:      role,
		OrgID:     orgID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		APIKeys:   []string{},
		Active:    true,
	}
	m.users[user.ID] = user
	_ = m.orgMgr.AddMember(orgID, user.ID, role)
	return user, nil
}

func (m *UserManager) GetUser(id string) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, exists := m.users[id]
	return user, exists
}

func (m *UserManager) GetUserByEmail(email string) (*User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if user.Email == email {
			return user, true
		}
	}
	return nil, false
}

// CreateAPIKey genera una API key con scopes específicos.
// Devuelve la key en claro UNA SOLA VEZ y almacena solo su hash.
func (m *UserManager) CreateAPIKey(userID, name, projectID string, scopes []Scope, ttl time.Duration) (string, *APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return "", nil, ErrUserNotFound
	}

	if projectID != "" {
		if _, exists := m.orgMgr.GetProject(projectID); !exists {
			return "", nil, ErrProjectNotFound
		}
	}

	// Si no hay scopes, usar los del rol
	if len(scopes) == 0 {
		scopes = ScopesForRole(user.Role).List()
	}

	// TTL por defecto: 1 año
	if ttl == 0 {
		ttl = 365 * 24 * time.Hour
	}

	rawKey := generateAPIKey()
	hash := hashKey(rawKey)
	prefix := rawKey[:11] // "sf_" + 8 chars

	apiKey := &APIKey{
		ID:        generateID("apikey"),
		KeyHash:   hash,
		KeyPrefix: prefix,
		Name:      name,
		UserID:    userID,
		OrgID:     user.OrgID,
		ProjectID: projectID,
		Scopes:    scopes,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Active:    true,
	}
	m.apiKeys[hash] = apiKey
	user.APIKeys = append(user.APIKeys, apiKey.ID)
	user.UpdatedAt = time.Now()

	return rawKey, apiKey, nil
}

// RotateAPIKey crea una nueva key y revoca la anterior
func (m *UserManager) RotateAPIKey(oldRawKey, name string, ttl time.Duration) (string, *APIKey, error) {
	m.mu.Lock()
	oldHash := hashKey(oldRawKey)
	oldKey, exists := m.apiKeys[oldHash]
	if !exists {
		m.mu.Unlock()
		return "", nil, ErrUserNotFound
	}

	userID := oldKey.UserID
	projectID := oldKey.ProjectID
	scopes := oldKey.Scopes
	oldID := oldKey.ID

	// Revocar la anterior
	oldKey.Active = false
	m.mu.Unlock()

	// Crear la nueva
	newRaw, newKey, err := m.CreateAPIKey(userID, name, projectID, scopes, ttl)
	if err != nil {
		return "", nil, err
	}

	// Marcar rotación
	m.mu.Lock()
	newKey.RotatedFrom = oldID
	m.mu.Unlock()

	return newRaw, newKey, nil
}

func (m *UserManager) ValidateAPIKey(rawKey string) (*APIKey, *User, bool) {
	hash := hashKey(rawKey)

	m.mu.RLock()
	apiKey, exists := m.apiKeys[hash]
	if !exists {
		m.mu.RUnlock()
		return nil, nil, false
	}
	user, userExists := m.users[apiKey.UserID]
	valid := apiKey.Active &&
		!time.Now().After(apiKey.ExpiresAt) &&
		userExists &&
		user.Active
	m.mu.RUnlock()

	if !valid {
		return nil, nil, false
	}

	m.mu.Lock()
	apiKey.LastUsed = time.Now()
	m.mu.Unlock()

	return apiKey, user, true
}

// ValidateAPIKeyWithScope valida la key y verifica un scope
func (m *UserManager) ValidateAPIKeyWithScope(rawKey string, requiredScope Scope) (*APIKey, *User, bool) {
	apiKey, user, valid := m.ValidateAPIKey(rawKey)
	if !valid {
		return nil, nil, false
	}

	scopeSet := NewScopeSet(apiKey.Scopes...)
	if !scopeSet.Has(requiredScope) {
		return nil, nil, false
	}

	return apiKey, user, true
}

func (m *UserManager) RevokeAPIKey(rawKey string) bool {
	hash := hashKey(rawKey)
	m.mu.Lock()
	defer m.mu.Unlock()

	apiKey, exists := m.apiKeys[hash]
	if !exists {
		return false
	}
	apiKey.Active = false
	return true
}

// ListAPIKeys devuelve las keys de un usuario (sin el hash)
func (m *UserManager) ListAPIKeys(userID string) []*APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*APIKey
	for _, key := range m.apiKeys {
		if key.UserID == userID {
			result = append(result, key)
		}
	}
	return result
}

// hashKey devuelve SHA-256 hex de la key
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// generateAPIKey usa crypto/rand
func generateAPIKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return "sf_" + hex.EncodeToString(b)
}
