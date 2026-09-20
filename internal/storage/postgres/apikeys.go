package postgres

import (
"context"
"crypto/rand"
"crypto/sha256"
"database/sql"
"encoding/hex"
"encoding/json"
"time"

"github.com/jackc/pgx/v5"
)

// APIKey representa una API key
type APIKey struct {
ID          string    `json:"id"`
KeyHash     string    `json:"-"`
KeyPrefix   string    `json:"key_prefix"`
Name        string    `json:"name"`
UserID      string    `json:"user_id"`
OrgID       string    `json:"org_id"`
ProjectID   string    `json:"project_id"`
Scopes      []string  `json:"scopes"`
Active      bool      `json:"active"`
CreatedAt   time.Time `json:"created_at"`
UpdatedAt   time.Time `json:"updated_at"`
LastUsed    time.Time `json:"last_used"`
ExpiresAt   time.Time `json:"expires_at"`
RotatedFrom string    `json:"rotated_from,omitempty"`
}

// APIKeyRepo gestiona API keys
type APIKeyRepo struct {
client *Client
}

func (c *Client) APIKeys() *APIKeyRepo {
return &APIKeyRepo{client: c}
}

// Create crea una nueva API key. Devuelve la key en claro UNA SOLA VEZ.
func (r *APIKeyRepo) Create(ctx context.Context, userID, orgID, projectID, name string, scopes []string, ttl time.Duration) (string, *APIKey, error) {
rawKey := generateRawKey()
hash := hashKey(rawKey)
prefix := rawKey[:11]

if ttl == 0 {
ttl = 365 * 24 * time.Hour
}

if len(scopes) == 0 {
scopes = []string{"chat:read", "chat:write", "models:read"}
}

scopesJSON, _ := json.Marshal(scopes)

key := &APIKey{
ID:        generateID("key"),
KeyHash:   hash,
KeyPrefix: prefix,
Name:      name,
UserID:    userID,
OrgID:     orgID,
ProjectID: projectID,
Scopes:    scopes,
Active:    true,
CreatedAt: time.Now(),
UpdatedAt: time.Now(),
LastUsed:  time.Now(),
ExpiresAt: time.Now().Add(ttl),
}

_, err := r.client.Exec(ctx, `
INSERT INTO api_keys (id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, rotated_from)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
key.ID, key.KeyHash, key.KeyPrefix, key.Name, key.UserID, key.OrgID, key.ProjectID,
scopesJSON, key.Active, key.CreatedAt, key.UpdatedAt, key.LastUsed, key.ExpiresAt, "")
if err != nil {
return "", nil, err
}

return rawKey, key, nil
}

// GetByHash devuelve una API key por hash
func (r *APIKeyRepo) GetByHash(ctx context.Context, hash string) (*APIKey, error) {
row := r.client.QueryRow(ctx, `
SELECT id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, COALESCE(rotated_from, '')
FROM api_keys WHERE key_hash = $1`, hash)

return scanAPIKey(row)
}

// GetByID devuelve una API key por ID
func (r *APIKeyRepo) GetByID(ctx context.Context, id string) (*APIKey, error) {
row := r.client.QueryRow(ctx, `
SELECT id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, COALESCE(rotated_from, '')
FROM api_keys WHERE id = $1`, id)

return scanAPIKey(row)
}

// ListByUser lista las API keys de un usuario
func (r *APIKeyRepo) ListByUser(ctx context.Context, userID string) ([]*APIKey, error) {
rows, err := r.client.Query(ctx, `
SELECT id, key_hash, key_prefix, name, user_id, org_id, project_id,
scopes, active, created_at, updated_at, last_used, expires_at, COALESCE(rotated_from, '')
FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
if err != nil {
return nil, err
}
defer rows.Close()

var keys []*APIKey
for rows.Next() {
k, err := scanAPIKeyFromRows(rows)
if err != nil {
return nil, err
}
keys = append(keys, k)
}
return keys, rows.Err()
}

// UpdateLastUsed actualiza el timestamp de último uso
func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id string, t time.Time) error {
_, err := r.client.Exec(ctx, `UPDATE api_keys SET last_used = $1 WHERE id = $2`, t, id)
return err
}

// Revoke revoca una API key
func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
affected, err := r.client.Exec(ctx, `UPDATE api_keys SET active = FALSE WHERE id = $1`, id)
if err != nil {
return err
}
if affected == 0 {
return ErrNotFound
}
return nil
}

// Delete elimina una API key
func (r *APIKeyRepo) Delete(ctx context.Context, id string) error {
affected, err := r.client.Exec(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
if err != nil {
return err
}
if affected == 0 {
return ErrNotFound
}
return nil
}

// ValidateKey valida una key en claro
func (r *APIKeyRepo) ValidateKey(ctx context.Context, rawKey string) (*APIKey, error) {
hash := hashKey(rawKey)
key, err := r.GetByHash(ctx, hash)
if err != nil {
return nil, err
}

if !key.Active {
return nil, ErrNotFound
}
if time.Now().After(key.ExpiresAt) {
return nil, ErrNotFound
}

// Update last_used (async)
go func() {
ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
_ = r.UpdateLastUsed(ctx2, key.ID, time.Now())
}()

return key, nil
}

// Helpers

func generateRawKey() string {
b := make([]byte, 32)
rand.Read(b)
return "sf_" + hex.EncodeToString(b)
}

func hashKey(key string) string {
sum := sha256.Sum256([]byte(key))
return hex.EncodeToString(sum[:])
}

func generateID(prefix string) string {
b := make([]byte, 8)
rand.Read(b)
return prefix + "_" + hex.EncodeToString(b)
}

// scanAPIKey escanea una fila de pgx.Row
func scanAPIKey(row pgx.Row) (*APIKey, error) {
var k APIKey
var scopesJSON []byte
var rotatedFrom string

err := row.Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.UserID, &k.OrgID,
&k.ProjectID, &scopesJSON, &k.Active, &k.CreatedAt, &k.UpdatedAt,
&k.LastUsed, &k.ExpiresAt, &rotatedFrom)
if err == sql.ErrNoRows || err == pgx.ErrNoRows {
return nil, ErrNotFound
}
if err != nil {
return nil, err
}

if len(scopesJSON) > 0 {
json.Unmarshal(scopesJSON, &k.Scopes)
}
k.RotatedFrom = rotatedFrom
return &k, nil
}

// scanAPIKeyFromRows escanea una fila de pgx.Rows
func scanAPIKeyFromRows(rows pgx.Rows) (*APIKey, error) {
var k APIKey
var scopesJSON []byte
var rotatedFrom string

err := rows.Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.UserID, &k.OrgID,
&k.ProjectID, &scopesJSON, &k.Active, &k.CreatedAt, &k.UpdatedAt,
&k.LastUsed, &k.ExpiresAt, &rotatedFrom)
if err != nil {
return nil, err
}

if len(scopesJSON) > 0 {
json.Unmarshal(scopesJSON, &k.Scopes)
}
k.RotatedFrom = rotatedFrom
return &k, nil
}
