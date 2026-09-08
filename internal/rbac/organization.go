package rbac

import (
"sync"
"time"
)

// Organization representa una organización
type Organization struct {
ID          string    `json:"id"`
Name        string    `json:"name"`
Description string    `json:"description"`
CreatedAt   time.Time `json:"created_at"`
UpdatedAt   time.Time `json:"updated_at"`
Projects    []string  `json:"projects"`
Members     []string  `json:"members"`
Settings    map[string]interface{} `json:"settings"`
}

// Project representa un proyecto dentro de una organización
type Project struct {
ID          string    `json:"id"`
Name        string    `json:"name"`
Description string    `json:"description"`
OrgID       string    `json:"org_id"`
CreatedAt   time.Time `json:"created_at"`
UpdatedAt   time.Time `json:"updated_at"`
APIKeys     []string  `json:"api_keys"`
Settings    map[string]interface{} `json:"settings"`
}

// OrganizationManager gestiona organizaciones y proyectos
type OrganizationManager struct {
mu          sync.RWMutex
orgs        map[string]*Organization
projects    map[string]*Project
}

// NewOrganizationManager crea un nuevo manager
func NewOrganizationManager() *OrganizationManager {
return &OrganizationManager{
orgs:     make(map[string]*Organization),
projects: make(map[string]*Project),
}
}

// CreateOrganization crea una nueva organización
func (m *OrganizationManager) CreateOrganization(name, description string) *Organization {
m.mu.Lock()
defer m.mu.Unlock()

org := &Organization{
ID:          generateID("org"),
Name:        name,
Description: description,
CreatedAt:   time.Now(),
UpdatedAt:   time.Now(),
Projects:    []string{},
Members:     []string{},
Settings:    make(map[string]interface{}),
}
m.orgs[org.ID] = org
return org
}

// GetOrganization devuelve una organización por ID
func (m *OrganizationManager) GetOrganization(id string) (*Organization, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
org, exists := m.orgs[id]
return org, exists
}

// CreateProject crea un nuevo proyecto
func (m *OrganizationManager) CreateProject(orgID, name, description string) (*Project, error) {
m.mu.Lock()
defer m.mu.Unlock()

org, exists := m.orgs[orgID]
if !exists {
return nil, ErrOrganizationNotFound
}

project := &Project{
ID:          generateID("proj"),
Name:        name,
Description: description,
OrgID:       orgID,
CreatedAt:   time.Now(),
UpdatedAt:   time.Now(),
APIKeys:     []string{},
Settings:    make(map[string]interface{}),
}
m.projects[project.ID] = project
org.Projects = append(org.Projects, project.ID)
org.UpdatedAt = time.Now()

return project, nil
}

// GetProject devuelve un proyecto por ID
func (m *OrganizationManager) GetProject(id string) (*Project, bool) {
m.mu.RLock()
defer m.mu.RUnlock()
proj, exists := m.projects[id]
return proj, exists
}

// ListProjects devuelve todos los proyectos de una organización
func (m *OrganizationManager) ListProjects(orgID string) []*Project {
m.mu.RLock()
defer m.mu.RUnlock()

org, exists := m.orgs[orgID]
if !exists {
return nil
}

projects := make([]*Project, 0, len(org.Projects))
for _, id := range org.Projects {
if p, ok := m.projects[id]; ok {
projects = append(projects, p)
}
}
return projects
}

// AddMember añade un miembro a una organización
func (m *OrganizationManager) AddMember(orgID, userID string, role Role) error {
m.mu.Lock()
defer m.mu.Unlock()

org, exists := m.orgs[orgID]
if !exists {
return ErrOrganizationNotFound
}

// Verificar si ya es miembro
for _, member := range org.Members {
if member == userID {
return ErrUserAlreadyMember
}
}

org.Members = append(org.Members, userID)
org.UpdatedAt = time.Now()
return nil
}

// RemoveMember elimina un miembro de una organización
func (m *OrganizationManager) RemoveMember(orgID, userID string) error {
m.mu.Lock()
defer m.mu.Unlock()

org, exists := m.orgs[orgID]
if !exists {
return ErrOrganizationNotFound
}

for i, member := range org.Members {
if member == userID {
org.Members = append(org.Members[:i], org.Members[i+1:]...)
org.UpdatedAt = time.Now()
return nil
}
}
return ErrUserNotFound
}

func generateID(prefix string) string {
return prefix + "_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func randomString(n int) string {
const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
b := make([]byte, n)
for i := range b {
b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
}
return string(b)
}
