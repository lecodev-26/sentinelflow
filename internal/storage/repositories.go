package storage

import (
	"context"
	"errors"
	"time"
)

// Errores comunes
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
)

// Organization representa una organización persistida
type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Settings    string    `json:"settings"` // JSON
}

// Project representa un proyecto
type Project struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Settings    string    `json:"settings"`
}

// User representa un usuario
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	OrgID     string    `json:"org_id"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// APIKey representa una API key (con hash)
type APIKey struct {
	ID          string    `json:"id"`
	KeyHash     string    `json:"key_hash"`
	KeyPrefix   string    `json:"key_prefix"`
	Name        string    `json:"name"`
	UserID      string    `json:"user_id"`
	OrgID       string    `json:"org_id"`
	ProjectID   string    `json:"project_id"`
	Scopes      string    `json:"scopes"` // JSON array
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastUsed    time.Time `json:"last_used"`
	ExpiresAt   time.Time `json:"expires_at"`
	RotatedFrom string    `json:"rotated_from"`
}

// Budget representa un presupuesto persistido
type Budget struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	MonthlyLimit float64   `json:"monthly_limit"`
	Spent        float64   `json:"spent"`
	MonthStart   time.Time `json:"month_start"`
	ResetDate    time.Time `json:"reset_date"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Policy representa una política persistida
type Policy struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TenantID    string    `json:"tenant_id"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	Rules       string    `json:"rules"` // JSON
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditEvent representa un evento de auditoría persistido
type AuditEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	RequestID string    `json:"request_id"`
	Action    string    `json:"action"`
	Result    string    `json:"result"`
	Reason    string    `json:"reason"`
	Metadata  string    `json:"metadata"` // JSON
}

// UsageRecord representa un registro de uso persistido
type UsageRecord struct {
	ID           string    `json:"id"`
	RequestID    string    `json:"request_id"`
	TenantID     string    `json:"tenant_id"`
	ProjectID    string    `json:"project_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	LatencyMs    int64     `json:"latency_ms"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
}

// === INTERFACES ===

// OrganizationRepository gestiona organizaciones
type OrganizationRepository interface {
	Create(ctx context.Context, org *Organization) error
	GetByID(ctx context.Context, id string) (*Organization, error)
	List(ctx context.Context) ([]*Organization, error)
	Update(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id string) error
}

// ProjectRepository gestiona proyectos
type ProjectRepository interface {
	Create(ctx context.Context, proj *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	ListByOrg(ctx context.Context, orgID string) ([]*Project, error)
	Update(ctx context.Context, proj *Project) error
	Delete(ctx context.Context, id string) error
}

// UserRepository gestiona usuarios
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListByOrg(ctx context.Context, orgID string) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

// APIKeyRepository gestiona API keys
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByHash(ctx context.Context, hash string) (*APIKey, error)
	GetByID(ctx context.Context, id string) (*APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]*APIKey, error)
	UpdateLastUsed(ctx context.Context, id string, t time.Time) error
	Revoke(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

// BudgetRepository gestiona presupuestos
type BudgetRepository interface {
	Upsert(ctx context.Context, budget *Budget) error
	GetByTenant(ctx context.Context, tenantID string) (*Budget, error)
	List(ctx context.Context) ([]*Budget, error)
	UpdateSpent(ctx context.Context, tenantID string, spent float64) error
	Delete(ctx context.Context, tenantID string) error
}

// PolicyRepository gestiona políticas
type PolicyRepository interface {
	Create(ctx context.Context, policy *Policy) error
	GetByID(ctx context.Context, id string) (*Policy, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*Policy, error)
	Update(ctx context.Context, policy *Policy) error
	Delete(ctx context.Context, id string) error
}

// AuditRepository gestiona eventos de auditoría
type AuditRepository interface {
	Create(ctx context.Context, event *AuditEvent) error
	List(ctx context.Context, tenantID string, limit int) ([]*AuditEvent, error)
	ListByRequest(ctx context.Context, requestID string) ([]*AuditEvent, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// UsageRepository gestiona registros de uso
type UsageRepository interface {
	Create(ctx context.Context, record *UsageRecord) error
	ListByTenant(ctx context.Context, tenantID string, since time.Time) ([]*UsageRecord, error)
	SumCostByTenant(ctx context.Context, tenantID string, since time.Time) (float64, error)
	SumCostByProvider(ctx context.Context, tenantID string, since time.Time) (map[string]float64, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// Store agrupa todos los repositorios
type Store interface {
	Organizations() OrganizationRepository
	Projects() ProjectRepository
	Users() UserRepository
	APIKeys() APIKeyRepository
	Budgets() BudgetRepository
	Policies() PolicyRepository
	Audit() AuditRepository
	Usage() UsageRepository

	Migrate(ctx context.Context) error
	Close() error
}
