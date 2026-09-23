package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/idgen"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

// Service agrupa la lógica de negocio de identity
type Service struct {
	orgs     *postgres.OrganizationRepo
	projects *postgres.ProjectRepo
	users    *postgres.UserRepo
	keys     *postgres.APIKeyRepo
}

// NewService crea un nuevo servicio
func NewService(client *postgres.Client) *Service {
	return &Service{
		orgs:     client.Organizations(),
		projects: client.Projects(),
		users:    client.Users(),
		keys:     client.APIKeys(),
	}
}

// === ORGANIZATIONS ===

// CreateOrganization crea una organización con validación
func (s *Service) CreateOrganization(ctx context.Context, name, description string) (*postgres.Organization, error) {
	if name = strings.TrimSpace(name); name == "" {
		return nil, errors.New("name is required")
	}
	if len(name) > 100 {
		return nil, errors.New("name too long (max 100)")
	}

	org := &postgres.Organization{
		ID:            idgen.NewID("org"),
		Name:          name,
		Description:   description,
		Residency:     "global",
		RetentionDays: 90,
		Settings:      map[string]interface{}{},
	}

	if err := s.orgs.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return org, nil
}

// GetOrganization devuelve una organización
func (s *Service) GetOrganization(ctx context.Context, id string) (*postgres.Organization, error) {
	return s.orgs.GetByID(ctx, id)
}

// ListOrganizations lista todas las organizaciones
func (s *Service) ListOrganizations(ctx context.Context) ([]*postgres.Organization, error) {
	return s.orgs.List(ctx)
}

// DeleteOrganization elimina una organización
func (s *Service) DeleteOrganization(ctx context.Context, id string) error {
	return s.orgs.Delete(ctx, id)
}

// === PROJECTS ===

// CreateProject crea un proyecto con validación
func (s *Service) CreateProject(ctx context.Context, orgID, name, description string) (*postgres.Project, error) {
	if orgID == "" {
		return nil, errors.New("org_id is required")
	}
	if name = strings.TrimSpace(name); name == "" {
		return nil, errors.New("name is required")
	}

	if _, err := s.orgs.GetByID(ctx, orgID); err != nil {
		return nil, errors.New("organization not found")
	}

	project := &postgres.Project{
		ID:          idgen.NewID("proj"),
		OrgID:       orgID,
		Name:        name,
		Description: description,
		Settings:    map[string]interface{}{},
	}

	if err := s.projects.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetProject devuelve un proyecto
func (s *Service) GetProject(ctx context.Context, id string) (*postgres.Project, error) {
	return s.projects.GetByID(ctx, id)
}

// ListProjects lista los proyectos de una org
func (s *Service) ListProjects(ctx context.Context, orgID string) ([]*postgres.Project, error) {
	return s.projects.ListByOrg(ctx, orgID)
}

// DeleteProject elimina un proyecto
func (s *Service) DeleteProject(ctx context.Context, id string) error {
	return s.projects.Delete(ctx, id)
}

// === USERS ===

// CreateUser crea un usuario con validación
func (s *Service) CreateUser(ctx context.Context, orgID, email, name, role string) (*postgres.User, error) {
	if orgID == "" {
		return nil, errors.New("org_id is required")
	}
	if email = strings.TrimSpace(strings.ToLower(email)); email == "" {
		return nil, errors.New("email is required")
	}
	if !strings.Contains(email, "@") {
		return nil, errors.New("invalid email")
	}
	if role == "" {
		role = "viewer"
	}
	if !isValidRole(role) {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	if _, err := s.orgs.GetByID(ctx, orgID); err != nil {
		return nil, errors.New("organization not found")
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, errors.New("email already exists")
	}

	user := &postgres.User{
		ID:     idgen.NewID("user"),
		Email:  email,
		Name:   name,
		Role:   role,
		OrgID:  orgID,
		Active: true,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUser devuelve un usuario
func (s *Service) GetUser(ctx context.Context, id string) (*postgres.User, error) {
	return s.users.GetByID(ctx, id)
}

// ListUsers lista los usuarios de una org
func (s *Service) ListUsers(ctx context.Context, orgID string) ([]*postgres.User, error) {
	return s.users.ListByOrg(ctx, orgID)
}

// DeleteUser elimina un usuario
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	return s.users.Delete(ctx, id)
}

// === API KEYS ===

// CreateAPIKey crea una API key para un usuario
func (s *Service) CreateAPIKey(ctx context.Context, userID, projectID, name string, scopes []string, ttl time.Duration) (string, *postgres.APIKey, error) {
	if userID == "" {
		return "", nil, errors.New("user_id is required")
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", nil, errors.New("user not found")
	}
	if !user.Active {
		return "", nil, errors.New("user is inactive")
	}

	if projectID != "" {
		if _, err := s.projects.GetByID(ctx, projectID); err != nil {
			return "", nil, errors.New("project not found")
		}
	}

	if name == "" {
		name = "default-key"
	}

	return s.keys.Create(ctx, userID, user.OrgID, projectID, name, scopes, ttl)
}

// ValidateAPIKey valida una key en claro
func (s *Service) ValidateAPIKey(ctx context.Context, rawKey string) (*postgres.APIKey, *postgres.User, error) {
	key, err := s.keys.ValidateKey(ctx, rawKey)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.users.GetByID(ctx, key.UserID)
	if err != nil {
		return nil, nil, errors.New("user not found")
	}
	if !user.Active {
		return nil, nil, errors.New("user inactive")
	}

	return key, user, nil
}

// ListAPIKeys lista las API keys de un usuario
func (s *Service) ListAPIKeys(ctx context.Context, userID string) ([]*postgres.APIKey, error) {
	return s.keys.ListByUser(ctx, userID)
}

// RevokeAPIKey revoca una API key por ID
func (s *Service) RevokeAPIKey(ctx context.Context, id string) error {
	return s.keys.Revoke(ctx, id)
}

// === HELPERS ===

func isValidRole(role string) bool {
	switch role {
	case "admin", "editor", "viewer", "member":
		return true
	}
	return false
}
