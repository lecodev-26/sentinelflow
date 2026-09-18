package sqlite

import (
"context"
"database/sql"
"time"

"github.com/lecodev-26/sentinelflow/internal/storage"
)

type userRepo struct {
db *sql.DB
}

func (r *userRepo) Create(ctx context.Context, user *storage.User) error {
if user.CreatedAt.IsZero() {
user.CreatedAt = time.Now()
}
user.UpdatedAt = time.Now()

_, err := r.db.ExecContext(ctx, `
INSERT INTO users (id, email, name, role, org_id, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
user.ID, user.Email, user.Name, user.Role, user.OrgID, user.Active, user.CreatedAt, user.UpdatedAt)
return err
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*storage.User, error) {
row := r.db.QueryRowContext(ctx, `
SELECT id, email, name, role, org_id, active, created_at, updated_at
FROM users WHERE id = ?`, id)

var u storage.User
err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active, &u.CreatedAt, &u.UpdatedAt)
if err == sql.ErrNoRows {
return nil, storage.ErrNotFound
}
if err != nil {
return nil, err
}
return &u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*storage.User, error) {
row := r.db.QueryRowContext(ctx, `
SELECT id, email, name, role, org_id, active, created_at, updated_at
FROM users WHERE email = ?`, email)

var u storage.User
err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active, &u.CreatedAt, &u.UpdatedAt)
if err == sql.ErrNoRows {
return nil, storage.ErrNotFound
}
if err != nil {
return nil, err
}
return &u, nil
}

func (r *userRepo) ListByOrg(ctx context.Context, orgID string) ([]*storage.User, error) {
rows, err := r.db.QueryContext(ctx, `
SELECT id, email, name, role, org_id, active, created_at, updated_at
FROM users WHERE org_id = ? ORDER BY created_at DESC`, orgID)
if err != nil {
return nil, err
}
defer rows.Close()

var users []*storage.User
for rows.Next() {
var u storage.User
if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active, &u.CreatedAt, &u.UpdatedAt); err != nil {
return nil, err
}
users = append(users, &u)
}
return users, rows.Err()
}

func (r *userRepo) Update(ctx context.Context, user *storage.User) error {
user.UpdatedAt = time.Now()
_, err := r.db.ExecContext(ctx, `
UPDATE users SET email=?, name=?, role=?, active=?, updated_at=?
WHERE id=?`,
user.Email, user.Name, user.Role, user.Active, user.UpdatedAt, user.ID)
return err
}

func (r *userRepo) Delete(ctx context.Context, id string) error {
_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
return err
}
