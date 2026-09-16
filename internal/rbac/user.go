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
ID        string    `json:"id"`
KeyHash   string    `json:"-"`
Name      string    `json:"name"`
UserID    string    `json:"user_id"`
OrgID     string    `json:"org_id"`
ProjectID string    `json:"project_id"`
CreatedAt time.Time `json:"created_at"`
LastUsed  time.Time `json:"last_used"`
ExpiresAt time.Time `json:"expires_at"`
Active    bool      `json:"active"`
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

func (m *UserManager) CreateAPIKey(userID, name, projectID string) (string, *APIKey, error) {
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

rawKey := generateAPIKey()
hash := hashKey(rawKey)

apiKey := &APIKey{
ID:        generateID("apikey"),
KeyHash:   hash,
Name:      name,
UserID:    userID,
OrgID:     user.OrgID,
ProjectID: projectID,
CreatedAt: time.Now(),
LastUsed:  time.Now(),
ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
Active:    true,
}
m.apiKeys[hash] = apiKey
user.APIKeys = append(user.APIKeys, apiKey.ID)
user.UpdatedAt = time.Now()

return rawKey, apiKey, nil
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

func hashKey(key string) string {
sum := sha256.Sum256([]byte(key))
return hex.EncodeToString(sum[:])
}

func generateAPIKey() string {
b := make([]byte, 32)
if _, err := rand.Read(b); err != nil {
panic("crypto/rand failed: " + err.Error())
}
return "sf_" + hex.EncodeToString(b)
}
