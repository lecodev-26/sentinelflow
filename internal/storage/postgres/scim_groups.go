package postgres

import (
	"context"
	"database/sql"
	"time"
)

type SCIMGroup struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"orgId"`
	DisplayName string    `json:"displayName"`
	ExternalID  string    `json:"externalId,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type SCIMGroupMember struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
}
type SCIMGroupRepo struct{ client *Client }

func (c *Client) SCIMGroups() *SCIMGroupRepo { return &SCIMGroupRepo{client: c} }
func (r *SCIMGroupRepo) Create(ctx context.Context, g *SCIMGroup) error {
	_, e := r.client.Exec(ctx, `INSERT INTO scim_groups(id,org_id,display_name,external_id) VALUES($1,$2,$3,$4)`, g.ID, g.OrgID, g.DisplayName, g.ExternalID)
	return e
}
func (r *SCIMGroupRepo) Get(ctx context.Context, id string) (*SCIMGroup, error) {
	var g SCIMGroup
	e := r.client.QueryRow(ctx, `SELECT id,org_id,display_name,COALESCE(external_id,''),created_at,updated_at FROM scim_groups WHERE id=$1`, id).Scan(&g.ID, &g.OrgID, &g.DisplayName, &g.ExternalID, &g.CreatedAt, &g.UpdatedAt)
	if e == sql.ErrNoRows {
		return nil, e
	}
	if e != nil {
		return nil, e
	}
	return &g, nil
}
func (r *SCIMGroupRepo) List(ctx context.Context, org string) ([]*SCIMGroup, error) {
	rows, e := r.client.Query(ctx, `SELECT id,org_id,display_name,COALESCE(external_id,''),created_at,updated_at FROM scim_groups WHERE org_id=$1 ORDER BY display_name`, org)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []*SCIMGroup
	for rows.Next() {
		var g SCIMGroup
		if e := rows.Scan(&g.ID, &g.OrgID, &g.DisplayName, &g.ExternalID, &g.CreatedAt, &g.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, &g)
	}
	return out, rows.Err()
}
func (r *SCIMGroupRepo) Delete(ctx context.Context, id string) error {
	n, e := r.client.Exec(ctx, `DELETE FROM scim_groups WHERE id=$1`, id)
	if e != nil {
		return e
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *SCIMGroupRepo) Members(ctx context.Context, id string) ([]SCIMGroupMember, error) {
	rows, e := r.client.Query(ctx, `SELECT m.user_id,u.name FROM scim_group_members m JOIN users u ON u.id=m.user_id WHERE m.group_id=$1 ORDER BY u.name`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []SCIMGroupMember
	for rows.Next() {
		var m SCIMGroupMember
		if e := rows.Scan(&m.Value, &m.Display); e != nil {
			return nil, e
		}
		m.Type = "User"
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *SCIMGroupRepo) AddMember(ctx context.Context, id, user string) error {
	_, e := r.client.Exec(ctx, `INSERT INTO scim_group_members(group_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, user)
	return e
}
func (r *SCIMGroupRepo) RemoveMember(ctx context.Context, id, user string) error {
	_, e := r.client.Exec(ctx, `DELETE FROM scim_group_members WHERE group_id=$1 AND user_id=$2`, id, user)
	return e
}
