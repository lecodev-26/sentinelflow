package rbac

import (
"sync"
"time"
)

// User representa un usuario
type User struct {
ID          string    `json:"id"`
Email       string    `json:"email"`
Name        string    `json:"name"`
Role        Role      `json:"role"`
OrgID       string    `json:"org_id"`
CreatedAt   time.Time `json:"created_at"`
UpdatedAt   time.Time `json:"updated_at"`
APIKeys     []string  `json:"api_keys"`
Active      bool      `json:"active"`
}

// APIKey representa una clave de API
type APIKey struct {
ID        string    `json:"id"`
Key       string    `json:"key"`
Name      string    `json:"name"`
UserID    string    `json:"user_id"`
OrgID     string    `json:"org_id"`
ProjectID string    `json:"project_id"`
CreatedAt time.Time `json:"created_at"`
LastUsed  time.Time `json:"last_used"`
ExpiresAt time.Time `json:"expires_at"`
Active    bool      `json:"active"`
}

// UserManager gestiona usuarios y API keys
type UserManager struct {
mu      sync.RWMutex
users   map[string]*User
apiKeys map[string]*APIKey
orgMgr  *OrganizationManager
}

// NewUserManager crea un nuevo manager de usuarios
func NewUserManager(orgMgr *OrganizationManager) *UserManager {
return &UserManager{
users:   make(map[string]*User),
apiKeys: make(map[string]*APIKey),
orgMgr:  orgMgr,
}
}

// CreateUser crea un nuevo usuario
func (m *UserManager) CreateUser(email, name string, role Role, orgID string) (*User, error) {
m.mu.Lock()
defer m.mu.Unlock()

// Verificar que la organización existe
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

// Añadir a la organización
_ = m.orgMgr.AddMember(orgID, user.ID, role)

return user, nil
}

// GetUser devuelve un usuario por ID
func (m *UserManager) GetUser(id string) (*User, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
user, exists := m.users[id]
return user, exists
}

// GetUserByEmail devuelve un usuario por email
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

// CreateAPIKey crea una nueva API key para un usuario
func (m *UserManager) CreateAPIKey(userID, name, projectID string) (*APIKey, error) {
m.mu.Lock()
defer m.mu.Unlock()

user, exists := m.users[userID]
if !exists {
return nil, ErrUserNotFound
}

// Verificar que el proyecto existe
if projectID != "" {
if _, exists := m.orgMgr.GetProject(projectID); !exists {
return nil, ErrProjectNotFound
}
}

key := generateAPIKey()
apiKey := &APIKey{
ID:        generateID("apikey"),
Key:       key,
Name:      name,
UserID:    userID,
OrgID:     user.OrgID,
ProjectID: projectID,
CreatedAt: time.Now(),
LastUsed:  time.Now(),
ExpiresAt: time.Now().Add(365 * 24 * time.Hour), // 1 año
Active:    true,
}
m.apiKeys[key] = apiKey
user.APIKeys = append(user.APIKeys, apiKey.ID)
user.UpdatedAt = time.Now()

return apiKey, nil
}

// ValidateAPIKey valida una API key
func (m *UserManager) ValidateAPIKey(key string) (*APIKey, *User, bool) {
m.mu.RLock()
defer m.mu.RUnlock()

apiKey, exists := m.apiKeys[key]
if !exists {
return nil, nil, false
}

if !apiKey.Active {
return nil, nil, false
}

if time.Now().After(apiKey.ExpiresAt) {
return nil, nil, false
}

user, exists := m.users[apiKey.UserID]
if !exists {
return nil, nil, false
}

if !user.Active {
return nil, nil, false
}

apiKey.LastUsed = time.Now()
return apiKey, user, true
}

// RevokeAPIKey revoca una API key
func (m *UserManager) RevokeAPIKey(key string) bool {
m.mu.Lock()
defer m.mu.Unlock()

apiKey, exists := m.apiKeys[key]
if !exists {
return false
}

apiKey.Active = false
return true
}

func generateAPIKey() string {
return "sf_" + randomString(32)
}
