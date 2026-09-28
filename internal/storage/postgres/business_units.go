package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type BusinessUnit struct {
	ID          string                 `json:"id"`
	OrgID       string                 `json:"org_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type BusinessUnitRepo struct{ client *Client }

func (c *Client) BusinessUnits() *BusinessUnitRepo { return &BusinessUnitRepo{client: c} }

func (r *BusinessUnitRepo) Create(ctx context.Context, b *BusinessUnit) error {
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	b.UpdatedAt = time.Now()
	settings, _ := json.Marshal(b.Settings)
	if b.Settings == nil {
		settings = []byte("{}")
	}
	_, err := r.client.Exec(ctx, `INSERT INTO business_units (id,org_id,name,description,settings,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, b.ID, b.OrgID, b.Name, b.Description, settings, b.CreatedAt, b.UpdatedAt)
	return err
}
func scanBusinessUnit(row interface{ Scan(...any) error }) (*BusinessUnit, error) {
	var b BusinessUnit
	var settings []byte
	err := row.Scan(&b.ID, &b.OrgID, &b.Name, &b.Description, &settings, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(settings) > 0 {
		_ = json.Unmarshal(settings, &b.Settings)
	}
	return &b, nil
}
func (r *BusinessUnitRepo) GetByID(ctx context.Context, id string) (*BusinessUnit, error) {
	return scanBusinessUnit(r.client.QueryRow(ctx, `SELECT id,org_id,name,description,settings,created_at,updated_at FROM business_units WHERE id=$1`, id))
}
func (r *BusinessUnitRepo) ListByOrg(ctx context.Context, orgID string) ([]*BusinessUnit, error) {
	rows, err := r.client.Query(ctx, `SELECT id,org_id,name,description,settings,created_at,updated_at FROM business_units WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*BusinessUnit
	for rows.Next() {
		b, err := scanBusinessUnit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (r *BusinessUnitRepo) Delete(ctx context.Context, id string) error {
	n, err := r.client.Exec(ctx, `DELETE FROM business_units WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
