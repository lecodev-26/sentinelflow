package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Project representa un proyecto
type Project struct {
	ID          string                 `json:"id"`
	OrgID       string                 `json:"org_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ProjectRepo gestiona proyectos
type ProjectRepo struct {
	client *Client
}

func (c *Client) Projects() *ProjectRepo {
	return &ProjectRepo{client: c}
}

func (r *ProjectRepo) Create(ctx context.Context, p *Project) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	p.UpdatedAt = time.Now()

	settingsJSON, _ := json.Marshal(p.Settings)
	if p.Settings == nil {
		settingsJSON = []byte("{}")
	}

	_, err := r.client.Exec(ctx, `
INSERT INTO projects (id, org_id, name, description, settings, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.OrgID, p.Name, p.Description, settingsJSON, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *ProjectRepo) GetByID(ctx context.Context, id string) (*Project, error) {
	row := r.client.QueryRow(ctx, `
SELECT id, org_id, name, description, settings, created_at, updated_at
FROM projects WHERE id = $1`, id)

	var p Project
	var settingsJSON []byte
	err := row.Scan(&p.ID, &p.OrgID, &p.Name, &p.Description, &settingsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(settingsJSON) > 0 {
		json.Unmarshal(settingsJSON, &p.Settings)
	}
	return &p, nil
}

func (r *ProjectRepo) ListByOrg(ctx context.Context, orgID string) ([]*Project, error) {
	rows, err := r.client.Query(ctx, `
SELECT id, org_id, name, description, settings, created_at, updated_at
FROM projects WHERE org_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var p Project
		var settingsJSON []byte
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Description,
			&settingsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if len(settingsJSON) > 0 {
			json.Unmarshal(settingsJSON, &p.Settings)
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepo) Update(ctx context.Context, p *Project) error {
	p.UpdatedAt = time.Now()
	settingsJSON, _ := json.Marshal(p.Settings)

	_, err := r.client.Exec(ctx, `
UPDATE projects SET name=$1, description=$2, settings=$3, updated_at=$4
WHERE id=$5`,
		p.Name, p.Description, settingsJSON, p.UpdatedAt, p.ID)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	affected, err := r.client.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
