package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/storage"
)

type projectRepo struct {
	db *sql.DB
}

func (r *projectRepo) Create(ctx context.Context, p *storage.Project) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	p.UpdatedAt = time.Now()
	if p.Settings == "" {
		p.Settings = "{}"
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO projects (id, org_id, name, description, settings, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.OrgID, p.Name, p.Description, p.Settings, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *projectRepo) GetByID(ctx context.Context, id string) (*storage.Project, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, org_id, name, description, settings, created_at, updated_at
FROM projects WHERE id = ?`, id)

	var p storage.Project
	err := row.Scan(&p.ID, &p.OrgID, &p.Name, &p.Description, &p.Settings, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *projectRepo) ListByOrg(ctx context.Context, orgID string) ([]*storage.Project, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, org_id, name, description, settings, created_at, updated_at
FROM projects WHERE org_id = ? ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*storage.Project
	for rows.Next() {
		var p storage.Project
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Description, &p.Settings, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

func (r *projectRepo) Update(ctx context.Context, p *storage.Project) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
UPDATE projects SET name=?, description=?, settings=?, updated_at=? WHERE id=?`,
		p.Name, p.Description, p.Settings, p.UpdatedAt, p.ID)
	return err
}

func (r *projectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	return err
}
