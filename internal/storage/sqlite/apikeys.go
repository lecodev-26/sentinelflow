package sqlite

import (
"context"
"database/sql"
"time"

"github.com/lecodev-26/sentinelflow/internal/storage"
)

type apiKeyRepo struct {
db *sql.DB
}

func (r *apiKeyRepo) Create(ctx context.Context, key *storage.APIKey) error {
if key.CreatedAt.IsZero() {
key.CreatedAt = time.Now()
}
key.UpdatedAt = time.Now()
if key.Scopes == "" {
key.Scopes = "[]"
}

_, err := r.db.ExecContext(ctx, `
INSERT INTO api_keys (id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, rotated_from)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
key.ID, key.KeyHash, key.KeyPrefix, key.Name, key.UserID, key.OrgID,
key.ProjectID, key.Scopes, key.Active, key.CreatedAt, key.UpdatedAt,
key.LastUsed, key.ExpiresAt, key.RotatedFrom)
return err
}

func (r *apiKeyRepo) GetByHash(ctx context.Context, hash string) (*storage.APIKey, error) {
row := r.db.QueryRowContext(ctx, `
SELECT id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, rotated_from
FROM api_keys WHERE key_hash = ?`, hash)

return scanAPIKey(row)
}

func (r *apiKeyRepo) GetByID(ctx context.Context, id string) (*storage.APIKey, error) {
row := r.db.QueryRowContext(ctx, `
SELECT id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, rotated_from
FROM api_keys WHERE id = ?`, id)

return scanAPIKey(row)
}

func (r *apiKeyRepo) ListByUser(ctx context.Context, userID string) ([]*storage.APIKey, error) {
rows, err := r.db.QueryContext(ctx, `
SELECT id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, rotated_from
FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`, userID)
if err != nil {
return nil, err
}
defer rows.Close()

var keys []*storage.APIKey
for rows.Next() {
var k storage.APIKey
err := rows.Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.UserID,
&k.OrgID, &k.ProjectID, &k.Scopes, &k.Active, &k.CreatedAt,
&k.UpdatedAt, &k.LastUsed, &k.ExpiresAt, &k.RotatedFrom)
if err != nil {
return nil, err
}
keys = append(keys, &k)
}
return keys, rows.Err()
}

func (r *apiKeyRepo) UpdateLastUsed(ctx context.Context, id string, t time.Time) error {
_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET last_used = ? WHERE id = ?`, t, id)
return err
}

func (r *apiKeyRepo) Revoke(ctx context.Context, id string) error {
_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET active = 0 WHERE id = ?`, id)
return err
}

func (r *apiKeyRepo) Delete(ctx context.Context, id string) error {
_, err := r.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
return err
}

func scanAPIKey(row *sql.Row) (*storage.APIKey, error) {
var k storage.APIKey
err := row.Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.UserID,
&k.OrgID, &k.ProjectID, &k.Scopes, &k.Active, &k.CreatedAt,
&k.UpdatedAt, &k.LastUsed, &k.ExpiresAt, &k.RotatedFrom)
if err == sql.ErrNoRows {
return nil, storage.ErrNotFound
}
if err != nil {
return nil, err
}
return &k, nil
}
