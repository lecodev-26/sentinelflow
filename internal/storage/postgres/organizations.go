package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Organization representa una organización persistida
type Organization struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Residency     string                 `json:"residency"`
	RetentionDays int                    `json:"retention_days"`
	Settings      map[string]interface{} `json:"settings"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// OrganizationRepo gestiona organizaciones
type OrganizationRepo struct {
	client *Client
}

// Organizations devuelve el repositorio
func (c *Client) Organizations() *OrganizationRepo {
	return &OrganizationRepo{client: c}
}

// Create crea una nueva organización
func (r *OrganizationRepo) Create(ctx context.Context, org *Organization) error {
	if org.CreatedAt.IsZero() {
		org.CreatedAt = time.Now()
	}
	org.UpdatedAt = time.Now()

	settingsJSON, _ := json.Marshal(org.Settings)
	if org.Settings == nil {
		settingsJSON = []byte("{}")
	}

	_, err := r.client.Exec(ctx, `
INSERT INTO organizations (id, name, description, residency, retention_days, settings, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		org.ID, org.Name, org.Description, org.Residency, org.RetentionDays,
		settingsJSON, org.CreatedAt, org.UpdatedAt)
	return err
}

// GetByID devuelve una organización por ID
func (r *OrganizationRepo) GetByID(ctx context.Context, id string) (*Organization, error) {
	row := r.client.QueryRow(ctx, `
SELECT id, name, description, residency, retention_days, settings, created_at, updated_at
FROM organizations WHERE id = $1`, id)

	var org Organization
	var settingsJSON []byte
	err := row.Scan(&org.ID, &org.Name, &org.Description, &org.Residency,
		&org.RetentionDays, &settingsJSON, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(settingsJSON) > 0 {
		json.Unmarshal(settingsJSON, &org.Settings)
	}
	return &org, nil
}

// List lista todas las organizaciones
func (r *OrganizationRepo) List(ctx context.Context) ([]*Organization, error) {
	rows, err := r.client.Query(ctx, `
SELECT id, name, description, residency, retention_days, settings, created_at, updated_at
FROM organizations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		var org Organization
		var settingsJSON []byte
		if err := rows.Scan(&org.ID, &org.Name, &org.Description, &org.Residency,
			&org.RetentionDays, &settingsJSON, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		if len(settingsJSON) > 0 {
			json.Unmarshal(settingsJSON, &org.Settings)
		}
		orgs = append(orgs, &org)
	}
	return orgs, rows.Err()
}

// Update actualiza una organización
func (r *OrganizationRepo) Update(ctx context.Context, org *Organization) error {
	org.UpdatedAt = time.Now()
	settingsJSON, _ := json.Marshal(org.Settings)

	_, err := r.client.Exec(ctx, `
UPDATE organizations 
SET name=$1, description=$2, residency=$3, retention_days=$4, settings=$5, updated_at=$6
WHERE id=$7`,
		org.Name, org.Description, org.Residency, org.RetentionDays,
		settingsJSON, org.UpdatedAt, org.ID)
	return err
}

// Delete elimina una organización
func (r *OrganizationRepo) Delete(ctx context.Context, id string) error {
	affected, err := r.client.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// Errores comunes
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)
