package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/storage"
	_ "modernc.org/sqlite"
)

// Store es la implementación SQLite del storage.Store
type Store struct {
	db *sql.DB

	orgs     *organizationRepo
	projects *projectRepo
	users    *userRepo
	apiKeys  *apiKeyRepo
	budgets  *budgetRepo
	policies *policyRepo
	audit    *auditRepo
	usage    *usageRepo
}

// NewStore abre una base de datos SQLite
func NewStore(path string) (*Store, error) {
	if path == "" {
		path = "sentinelflow.db"
	}

	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	// Limitar conexiones (SQLite no soporta bien muchas)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	s := &Store{db: db}
	s.orgs = &organizationRepo{db: db}
	s.projects = &projectRepo{db: db}
	s.users = &userRepo{db: db}
	s.apiKeys = &apiKeyRepo{db: db}
	s.budgets = &budgetRepo{db: db}
	s.policies = &policyRepo{db: db}
	s.audit = &auditRepo{db: db}
	s.usage = &usageRepo{db: db}

	return s, nil
}

// Migrate aplica el schema
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, Schema); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Registrar migración
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		1, time.Now(),
	)
	return err
}

// Close cierra la base de datos
func (s *Store) Close() error {
	return s.db.Close()
}

// DB devuelve la conexión raw
func (s *Store) DB() *sql.DB {
	return s.db
}

// Repositorios
func (s *Store) Organizations() storage.OrganizationRepository { return s.orgs }
func (s *Store) Projects() storage.ProjectRepository           { return s.projects }
func (s *Store) Users() storage.UserRepository                 { return s.users }
func (s *Store) APIKeys() storage.APIKeyRepository             { return s.apiKeys }
func (s *Store) Budgets() storage.BudgetRepository             { return s.budgets }
func (s *Store) Policies() storage.PolicyRepository            { return s.policies }
func (s *Store) Audit() storage.AuditRepository                { return s.audit }
func (s *Store) Usage() storage.UsageRepository                { return s.usage }
