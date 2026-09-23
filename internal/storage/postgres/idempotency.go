package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// IdempotencyStatus estado de una entrada
type IdempotencyStatus string

const (
	IdempotencyPending   IdempotencyStatus = "pending"
	IdempotencyCompleted IdempotencyStatus = "completed"
	IdempotencyFailed    IdempotencyStatus = "failed"
)

// IdempotencyEntry es una entrada en la tabla idempotency_keys
type IdempotencyEntry struct {
	Key         string            `json:"key"`
	TenantID    string            `json:"tenant_id"`
	RequestHash string            `json:"request_hash"`
	Status      IdempotencyStatus `json:"status"`
	Response    []byte            `json:"-"`
	StatusCode  int               `json:"status_code,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
}

// IdempotencyRepo gestiona el store de idempotencia
type IdempotencyRepo struct {
	client *Client
}

func (c *Client) Idempotency() *IdempotencyRepo {
	return &IdempotencyRepo{client: c}
}

// TryAcquire intenta crear una entrada 'pending'.
// Devuelve:
//   - (entry, true, nil)    si se adquirió el lock (primera vez)
//   - (entry, false, nil)   si ya existe (replay o conflict)
//   - (nil, false, err)     si error de DB
func (r *IdempotencyRepo) TryAcquire(ctx context.Context, key, tenantID, requestHash string, ttl time.Duration) (*IdempotencyEntry, bool, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	// INSERT ... ON CONFLICT DO NOTHING
	// Si la fila existe y NO ha expirado, la inserción falla silenciosamente
	_, err := r.client.Exec(ctx, `
INSERT INTO idempotency_keys (key, tenant_id, request_hash, status, created_at, updated_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (key) DO NOTHING`,
		key, tenantID, requestHash, string(IdempotencyPending), now, now, expiresAt)

	if err != nil {
		return nil, false, err
	}

	// Intentar leer la entrada (o bien la acabamos de crear, o ya existía)
	entry, err := r.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}

	// Determinar si fuimos nosotros los que insertamos:
	// si created_at está dentro de 1 segundo y status == pending, somos nosotros
	isNew := entry.Status == IdempotencyPending && time.Since(entry.CreatedAt) < time.Second
	return entry, isNew, nil
}

// Get devuelve una entrada por key
func (r *IdempotencyRepo) Get(ctx context.Context, key string) (*IdempotencyEntry, error) {
	row := r.client.QueryRow(ctx, `
SELECT key, tenant_id, request_hash, status, response, status_code, created_at, updated_at, expires_at
FROM idempotency_keys
WHERE key = $1`, key)

	var e IdempotencyEntry
	var status string
	err := row.Scan(&e.Key, &e.TenantID, &e.RequestHash, &status, &e.Response, &e.StatusCode, &e.CreatedAt, &e.UpdatedAt, &e.ExpiresAt)
	if err == sql.ErrNoRows || errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	e.Status = IdempotencyStatus(status)
	return &e, nil
}

// Complete marca una entrada como completada con la respuesta
func (r *IdempotencyRepo) Complete(ctx context.Context, key string, response []byte, statusCode int) error {
	_, err := r.client.Exec(ctx, `
UPDATE idempotency_keys
SET status = $1, response = $2, status_code = $3, updated_at = NOW()
WHERE key = $4`,
		string(IdempotencyCompleted), response, statusCode, key)
	return err
}

// Fail marca una entrada como fallida
func (r *IdempotencyRepo) Fail(ctx context.Context, key string) error {
	_, err := r.client.Exec(ctx, `
UPDATE idempotency_keys
SET status = $1, updated_at = NOW()
WHERE key = $2`,
		string(IdempotencyFailed), key)
	return err
}

// Delete elimina una entrada (útil para tests)
func (r *IdempotencyRepo) Delete(ctx context.Context, key string) error {
	_, err := r.client.Exec(ctx, `DELETE FROM idempotency_keys WHERE key = $1`, key)
	return err
}

// CleanupExpired elimina entradas expiradas
func (r *IdempotencyRepo) CleanupExpired(ctx context.Context) (int64, error) {
	return r.client.Exec(ctx, `DELETE FROM idempotency_keys WHERE expires_at < NOW()`)
}
