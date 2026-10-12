package runtimev6

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("runtime record not found")
	ErrConflict = errors.New("runtime state conflict")
)

// PostgresStore persists V6 runs and tasks. Mutations use conditional updates
// so concurrent workers cannot silently overwrite each other's lifecycle state.
type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) CreateRun(ctx context.Context, run Run) error {
	if err := run.Validate(); err != nil {
		return err
	}
	metadata, err := json.Marshal(run.Metadata)
	if err != nil {
		return fmt.Errorf("marshal run metadata: %w", err)
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO v6_runs
(id,tenant_id,principal_id,idempotency_key,correlation_id,status,payload_ref,metadata,created_at,updated_at,started_at,finished_at,error_code)
VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,COALESCE($9,NOW()),COALESCE($10,NOW()),$11,$12,$13)`,
		run.ID, run.TenantID, run.PrincipalID, run.IdempotencyKey, run.CorrelationID, string(run.Status), run.PayloadRef, metadata, nullTime(run.CreatedAt), nullTime(run.UpdatedAt), run.StartedAt, run.FinishedAt, run.ErrorCode)
	if err != nil {
		return fmt.Errorf("create run: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetRun(ctx context.Context, tenantID, id string) (Run, error) {
	var r Run
	var metadata []byte
	err := s.pool.QueryRow(ctx, `SELECT id,tenant_id,principal_id,COALESCE(idempotency_key,''),correlation_id,status,payload_ref,metadata,created_at,updated_at,started_at,finished_at,error_code FROM v6_runs WHERE tenant_id=$1 AND id=$2`, tenantID, id).Scan(
		&r.ID, &r.TenantID, &r.PrincipalID, &r.IdempotencyKey, &r.CorrelationID, &r.Status, &r.PayloadRef, &metadata, &r.CreatedAt, &r.UpdatedAt, &r.StartedAt, &r.FinishedAt, &r.ErrorCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("get run: %w", err)
	}
	if err := json.Unmarshal(metadata, &r.Metadata); err != nil {
		return Run{}, fmt.Errorf("decode run metadata: %w", err)
	}
	return r, nil
}

// TransitionRun applies a compare-and-swap lifecycle transition.
func (s *PostgresStore) TransitionRun(ctx context.Context, tenantID, id string, from, to RunStatus, errorCode string) error {
	if !CanTransitionRun(from, to) {
		return fmt.Errorf("%w: run %s -> %s", ErrInvalidTransition, from, to)
	}
	now := time.Now().UTC()
	started := any(nil)
	finished := any(nil)
	if to == RunRunning {
		started = now
	}
	if to == RunSucceeded || to == RunFailed || to == RunCancelled {
		finished = now
	}
	tag, err := s.pool.Exec(ctx, `UPDATE v6_runs SET status=$4, updated_at=$5,
started_at=COALESCE(started_at,$6), finished_at=COALESCE($7,finished_at), error_code=$8
WHERE tenant_id=$1 AND id=$2 AND status=$3`, tenantID, id, string(from), string(to), now, started, finished, errorCode)
	if err != nil {
		return fmt.Errorf("transition run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

func (s *PostgresStore) CreateTask(ctx context.Context, task Task) error {
	if err := task.Validate(); err != nil {
		return err
	}
	metadata, err := json.Marshal(task.Metadata)
	if err != nil {
		return fmt.Errorf("marshal task metadata: %w", err)
	}
	created, updated := task.CreatedAt, task.UpdatedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	if task.AvailableAt.IsZero() {
		task.AvailableAt = created
	}
	if updated.IsZero() {
		updated = created
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO v6_tasks
(id,run_id,tenant_id,correlation_id,status,attempt,max_attempts,available_at,lease_owner,lease_until,metadata,created_at,updated_at,started_at,finished_at,error_code)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`, task.ID, task.RunID, task.TenantID, task.CorrelationID, string(task.Status), task.Attempt, task.MaxAttempts, task.AvailableAt, task.LeaseOwner, task.LeaseUntil, metadata, created, updated, task.StartedAt, task.FinishedAt, task.ErrorCode)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

// TransitionTask applies a compare-and-swap lifecycle transition. Leaving the
// running state clears the lease so a stale worker cannot renew it.
func (s *PostgresStore) TransitionTask(ctx context.Context, tenantID, id string, from, to TaskStatus, errorCode string) error {
	if !CanTransitionTask(from, to) {
		return fmt.Errorf("%w: task %s -> %s", ErrInvalidTransition, from, to)
	}
	now := time.Now().UTC()
	finished := any(nil)
	if to == TaskSucceeded || to == TaskFailed || to == TaskCancelled {
		finished = now
	}
	tag, err := s.pool.Exec(ctx, `UPDATE v6_tasks SET status=$4, updated_at=$5,
finished_at=COALESCE($6,finished_at), error_code=$7,
lease_owner=CASE WHEN $4='running' THEN lease_owner ELSE '' END,
lease_until=CASE WHEN $4='running' THEN lease_until ELSE NULL END
WHERE tenant_id=$1 AND id=$2 AND status=$3`, tenantID, id, string(from), string(to), now, finished, errorCode)
	if err != nil {
		return fmt.Errorf("transition task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// ClaimTask atomically leases the oldest eligible task. SKIP LOCKED allows
// independent workers to claim concurrently without double-claiming a row.
func (s *PostgresStore) ClaimTask(ctx context.Context, tenantID, workerID string, lease time.Duration) (Task, error) {
	if tenantID == "" || workerID == "" || lease <= 0 {
		return Task{}, fmt.Errorf("tenant, worker and positive lease are required")
	}
	var t Task
	var metadata []byte
	err := s.pool.QueryRow(ctx, `WITH candidate AS (
 SELECT id FROM v6_tasks WHERE tenant_id=$1 AND status IN ('queued','retrying') AND available_at<=NOW() AND attempt<max_attempts
 ORDER BY available_at,created_at FOR UPDATE SKIP LOCKED LIMIT 1
), claimed AS (
 UPDATE v6_tasks t SET status='running', attempt=t.attempt+1, lease_owner=$2, lease_until=NOW()+$3::interval,
 started_at=COALESCE(t.started_at,NOW()), updated_at=NOW()
 FROM candidate c WHERE t.id=c.id
 RETURNING t.id,t.run_id,t.tenant_id,t.correlation_id,t.status,t.attempt,t.max_attempts,t.available_at,t.lease_owner,t.lease_until,t.metadata,t.created_at,t.updated_at,t.started_at,t.finished_at,t.error_code
) SELECT * FROM claimed`, tenantID, workerID, fmt.Sprintf("%f seconds", lease.Seconds())).Scan(
		&t.ID, &t.RunID, &t.TenantID, &t.CorrelationID, &t.Status, &t.Attempt, &t.MaxAttempts, &t.AvailableAt, &t.LeaseOwner, &t.LeaseUntil, &metadata, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.FinishedAt, &t.ErrorCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("claim task: %w", err)
	}
	if err := json.Unmarshal(metadata, &t.Metadata); err != nil {
		return Task{}, fmt.Errorf("decode task metadata: %w", err)
	}
	return t, nil
}

// RenewLease only succeeds for the current owner while the lease is still live.
func (s *PostgresStore) RenewLease(ctx context.Context, tenantID, taskID, workerID string, lease time.Duration) error {
	if lease <= 0 {
		return fmt.Errorf("positive lease is required")
	}
	tag, err := s.pool.Exec(ctx, `UPDATE v6_tasks SET lease_until=NOW()+$4::interval,updated_at=NOW()
WHERE tenant_id=$1 AND id=$2 AND status='running' AND lease_owner=$3 AND lease_until>NOW()`, tenantID, taskID, workerID, fmt.Sprintf("%f seconds", lease.Seconds()))
	if err != nil {
		return fmt.Errorf("renew task lease: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// RecoverExpiredLeases moves expired running tasks to retrying or failed, based
// on the bounded attempt count. It is safe to run concurrently on many workers.
func (s *PostgresStore) RecoverExpiredLeases(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE v6_tasks SET status=CASE WHEN attempt<max_attempts THEN 'retrying' ELSE 'failed' END,
 available_at=NOW(),lease_owner='',lease_until=NULL,updated_at=NOW(),finished_at=CASE WHEN attempt<max_attempts THEN NULL ELSE NOW() END,
 error_code=CASE WHEN attempt<max_attempts THEN error_code ELSE 'lease_expired_attempts_exhausted' END
WHERE status='running' AND lease_until IS NOT NULL AND lease_until<=NOW()`)
	if err != nil {
		return 0, fmt.Errorf("recover expired task leases: %w", err)
	}
	return tag.RowsAffected(), nil
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
