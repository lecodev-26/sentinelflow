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

// CreateArtifact stores an immutable reference to externally stored artifact
// bytes. The composite foreign keys enforce tenant ownership of the run/task.
func (s *PostgresStore) CreateArtifact(ctx context.Context, artifact Artifact) error {
	if artifact.ID == "" || artifact.TenantID == "" || artifact.RunID == "" || artifact.Kind == "" || artifact.StorageRef == "" || artifact.CreatedBy == "" {
		return fmt.Errorf("artifact id, tenant, run, kind, storage reference and creator are required")
	}
	createdAt := artifact.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO v6_artifacts
(id,tenant_id,run_id,task_id,kind,media_type,storage_ref,checksum,created_at,created_by,policy_version)
VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11)`, artifact.ID, artifact.TenantID, artifact.RunID, artifact.TaskID, artifact.Kind, artifact.MediaType, artifact.StorageRef, artifact.Checksum, createdAt, artifact.CreatedBy, artifact.PolicyVersion)
	if err != nil {
		return fmt.Errorf("create artifact: %w", err)
	}
	return nil
}

// GetArtifact never resolves the external storage reference; callers must apply
// their authorization policy before fetching artifact bytes.
func (s *PostgresStore) GetArtifact(ctx context.Context, tenantID, id string) (Artifact, error) {
	var a Artifact
	err := s.pool.QueryRow(ctx, `SELECT id,tenant_id,run_id,COALESCE(task_id,''),kind,media_type,storage_ref,checksum,created_at,created_by,policy_version
FROM v6_artifacts WHERE tenant_id=$1 AND id=$2`, tenantID, id).Scan(&a.ID, &a.TenantID, &a.RunID, &a.TaskID, &a.Kind, &a.MediaType, &a.StorageRef, &a.Checksum, &a.CreatedAt, &a.CreatedBy, &a.PolicyVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return Artifact{}, ErrNotFound
	}
	if err != nil {
		return Artifact{}, fmt.Errorf("get artifact: %w", err)
	}
	return a, nil
}

// AppendExecutionEvent appends an immutable execution-lineage event. It does
// not publish externally; delivery requires an outbox transaction.
func (s *PostgresStore) AppendExecutionEvent(ctx context.Context, event ExecutionEvent) error {
	if event.ID == "" || event.SchemaVersion == 0 || event.TenantID == "" || event.RunID == "" || event.CorrelationID == "" || event.Type == "" {
		return fmt.Errorf("event id, schema version, tenant, run, correlation id and type are required")
	}
	occurredAt := event.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal event metadata: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin execution event: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO v6_execution_events
(id,schema_version,tenant_id,run_id,task_id,correlation_id,event_type,occurred_at,principal_id,policy_version,metadata)
VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9,$10,$11)`, event.ID, event.SchemaVersion, event.TenantID, event.RunID, event.TaskID, event.CorrelationID, event.Type, occurredAt, event.PrincipalID, event.PolicyVersion, metadata); err != nil {
		return fmt.Errorf("append execution event: %w", err)
	}
	if err := enqueueExecutionEventOutbox(ctx, tx, event, occurredAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit execution event: %w", err)
	}
	return nil
}

// ListExecutionEvents returns a run's immutable lineage in occurrence order.
func (s *PostgresStore) ListExecutionEvents(ctx context.Context, tenantID, runID string, limit int) ([]ExecutionEvent, error) {
	if tenantID == "" || runID == "" {
		return nil, fmt.Errorf("tenant and run are required")
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT id,schema_version,tenant_id,run_id,COALESCE(task_id,''),correlation_id,event_type,occurred_at,principal_id,policy_version,metadata
FROM v6_execution_events WHERE tenant_id=$1 AND run_id=$2 ORDER BY occurred_at,id LIMIT $3`, tenantID, runID, limit)
	if err != nil {
		return nil, fmt.Errorf("list execution events: %w", err)
	}
	defer rows.Close()
	events := make([]ExecutionEvent, 0)
	for rows.Next() {
		var event ExecutionEvent
		var metadata []byte
		if err := rows.Scan(&event.ID, &event.SchemaVersion, &event.TenantID, &event.RunID, &event.TaskID, &event.CorrelationID, &event.Type, &event.OccurredAt, &event.PrincipalID, &event.PolicyVersion, &metadata); err != nil {
			return nil, fmt.Errorf("scan execution event: %w", err)
		}
		if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
			return nil, fmt.Errorf("decode execution event metadata: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution events: %w", err)
	}
	return events, nil
}

// CompleteTaskWithEvent fences completion to the worker that currently owns a
// live lease and commits the terminal transition plus lineage event atomically.
// External side effects still need their own idempotency/reconciliation.
func (s *PostgresStore) CompleteTaskWithEvent(ctx context.Context, tenantID, taskID, workerID string, to TaskStatus, errorCode string, event ExecutionEvent) error {
	if to != TaskSucceeded && to != TaskFailed && to != TaskCancelled {
		return fmt.Errorf("completion status must be terminal")
	}
	if event.ID == "" || event.SchemaVersion == 0 || event.Type == "" {
		return fmt.Errorf("event id, schema version and type are required")
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal event metadata: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin task completion: %w", err)
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	tag, err := tx.Exec(ctx, `UPDATE v6_tasks SET status=$4,updated_at=$5,finished_at=$5,error_code=$6,lease_owner='',lease_until=NULL
WHERE tenant_id=$1 AND id=$2 AND status='running' AND lease_owner=$3 AND lease_until>NOW()`, tenantID, taskID, workerID, string(to), now, errorCode)
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	var runID, correlationID string
	if err := tx.QueryRow(ctx, `SELECT run_id,correlation_id FROM v6_tasks WHERE tenant_id=$1 AND id=$2`, tenantID, taskID).Scan(&runID, &correlationID); err != nil {
		return fmt.Errorf("load completed task lineage: %w", err)
	}
	if tenantID == "" || taskID == "" || workerID == "" {
		return fmt.Errorf("tenant, task and worker are required")
	}
	if event.TenantID != "" && event.TenantID != tenantID || event.TaskID != "" && event.TaskID != taskID || event.RunID != "" && event.RunID != runID || event.CorrelationID != "" && event.CorrelationID != correlationID {
		return fmt.Errorf("event tenant/run/task/correlation does not match completed task")
	}
	if event.TenantID == "" {
		event.TenantID = tenantID
	}
	if event.RunID == "" {
		event.RunID = runID
	}
	if event.TaskID == "" {
		event.TaskID = taskID
	}
	if event.CorrelationID == "" {
		event.CorrelationID = correlationID
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	if _, err := tx.Exec(ctx, `INSERT INTO v6_execution_events
(id,schema_version,tenant_id,run_id,task_id,correlation_id,event_type,occurred_at,principal_id,policy_version,metadata)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, event.ID, event.SchemaVersion, event.TenantID, event.RunID, event.TaskID, event.CorrelationID, event.Type, event.OccurredAt, event.PrincipalID, event.PolicyVersion, metadata); err != nil {
		return fmt.Errorf("append completion event: %w", err)
	}
	if err := enqueueExecutionEventOutbox(ctx, tx, event, event.OccurredAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit task completion: %w", err)
	}
	return nil
}

// enqueueExecutionEventOutbox makes the event available to the existing
// at-least-once event publisher in the same transaction as its lineage record.
// Consumers must deduplicate by event ID because delivery is intentionally at-least-once.
func enqueueExecutionEventOutbox(ctx context.Context, tx pgx.Tx, event ExecutionEvent, occurredAt time.Time) error {
	payload, err := json.Marshal(map[string]interface{}{
		"schema_version": event.SchemaVersion,
		"run_id":         event.RunID,
		"task_id":        event.TaskID,
		"correlation_id": event.CorrelationID,
		"principal_id":   event.PrincipalID,
		"policy_version": event.PolicyVersion,
		"metadata":       event.Metadata,
	})
	if err != nil {
		return fmt.Errorf("marshal execution event outbox payload: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events
(event_id,event_type,tenant_id,user_id,request_id,trace_id,payload,status,attempts,next_attempt_at,occurred_at,created_at)
VALUES ($1,$2,$3,NULLIF($4,''),$5,$5,$6,'pending',0,NOW(),$7,NOW())
ON CONFLICT (event_id) DO NOTHING`, event.ID, event.Type, event.TenantID, event.PrincipalID, event.CorrelationID, payload, occurredAt)
	if err != nil {
		return fmt.Errorf("enqueue execution event outbox: %w", err)
	}
	return nil
}
