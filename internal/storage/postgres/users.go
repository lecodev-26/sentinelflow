package postgres

import (
"context"
"database/sql"
"time"
)

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

// UserRepo gestiona usuarios
type UserRepo struct {
client *Client
}

func (c *Client) Users() *UserRepo {
return &UserRepo{client: c}
}

func (r *UserRepo) Create(ctx context.Context, u *User) error {
if u.CreatedAt.IsZero() {
u.CreatedAt = time.Now()
}
u.UpdatedAt = time.Now()

_, err := r.client.Exec(ctx, `
INSERT INTO users (id, email, name, role, org_id, active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
u.ID, u.Email, u.Name, u.Role, u.OrgID, u.Active, u.CreatedAt, u.UpdatedAt)
return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
row := r.client.QueryRow(ctx, `
SELECT id, email, name, role, org_id, active, created_at, updated_at
FROM users WHERE id = $1`, id)

var u User
err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active, &u.CreatedAt, &u.UpdatedAt)
if err == sql.ErrNoRows {
return nil, ErrNotFound
}
if err != nil {
return nil, err
}
return &u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
row := r.client.QueryRow(ctx, `
SELECT id, email, name, role, org_id, active, created_at, updated_at
FROM users WHERE email = $1`, email)

var u User
err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active, &u.CreatedAt, &u.UpdatedAt)
if err == sql.ErrNoRows {
return nil, ErrNotFound
}
if err != nil {
return nil, err
}
return &u, nil
}

func (r *UserRepo) ListByOrg(ctx context.Context, orgID string) ([]*User, error) {
rows, err := r.client.Query(ctx, `
SELECT id, email, name, role, org_id, active, created_at, updated_at
FROM users WHERE org_id = $1 ORDER BY created_at DESC`, orgID)
if err != nil {
return nil, err
}
defer rows.Close()

var users []*User
for rows.Next() {
var u User
if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active, &u.CreatedAt, &u.UpdatedAt); err != nil {
return nil, err
}
users = append(users, &u)
}
return users, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, u *User) error {
u.UpdatedAt = time.Now()
_, err := r.client.Exec(ctx, `
UPDATE users SET email=$1, name=$2, role=$3, active=$4, updated_at=$5
WHERE id=$6`,
u.Email, u.Name, u.Role, u.Active, u.UpdatedAt, u.ID)
return err
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
affected, err := r.client.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
if err != nil {
return err
}
if affected == 0 {
return ErrNotFound
}
return nil
}
