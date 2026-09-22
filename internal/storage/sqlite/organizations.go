package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/storage"
)

type organizationRepo struct {
	db *sql.DB
}

func (r *organizationRepo) Create(ctx context.Context, org *storage.Organization) error {
	if org.CreatedAt.IsZero() {
		org.CreatedAt = time.Now()
	}
	org.UpdatedAt = time.Now()
	if org.Settings == "" {
		org.Settings = "{}"
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO organizations (id, name, description, settings, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		org.ID, org.Name, org.Description, org.Settings, org.CreatedAt, org.UpdatedAt)
	return err
}

func (r *organizationRepo) GetByID(ctx context.Context, id string) (*storage.Organization, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, description, settings, created_at, updated_at
FROM organizations WHERE id = ?`, id)

	var org storage.Organization
	err := row.Scan(&org.ID, &org.Name, &org.Description, &org.Settings, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepo) List(ctx context.Context) ([]*storage.Organization, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, description, settings, created_at, updated_at
FROM organizations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*storage.Organization
	for rows.Next() {
		var org storage.Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Description, &org.Settings, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, &org)
	}
	return orgs, rows.Err()
}

func (r *organizationRepo) Update(ctx context.Context, org *storage.Organization) error {
	org.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
UPDATE organizations SET name=?, description=?, settings=?, updated_at=?
WHERE id=?`,
		org.Name, org.Description, org.Settings, org.UpdatedAt, org.ID)
	return err
}

func (r *organizationRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM organizations WHERE id = ?`, id)
	return err
}
