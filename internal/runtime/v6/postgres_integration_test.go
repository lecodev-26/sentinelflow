package runtimev6

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lecodev-26/sentinelflow/internal/events"
)

func integrationStore(t *testing.T) (*pgxpool.Pool, *PostgresStore, context.Context) {
	t.Helper()
	dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SENTINELFLOW_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("SENTINELFLOW_DATABASE_URL is set but PostgreSQL is unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool, NewPostgresStore(pool), ctx
}

func integrationRun(tenant, id string) Run {
	return Run{ID: id, TenantID: tenant, PrincipalID: "integration-test", CorrelationID: "corr-" + id, Status: RunQueued, Metadata: map[string]string{"suite": "runtime-v6"}}
}

func integrationTask(tenant, runID, id string) Task {
	return Task{ID: id, RunID: runID, TenantID: tenant, CorrelationID: "corr-" + runID, Status: TaskQueued, MaxAttempts: 3, Metadata: map[string]string{"suite": "runtime-v6"}}
}

func cleanupRun(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenant, id string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := pool.Exec(cleanupCtx, `DELETE FROM v6_runs WHERE tenant_id=$1 AND id=$2`, tenant, id)
		if err != nil {
			t.Errorf("cleanup integration run %s: %v", id, err)
		}
	})
}

func TestPostgresStore_RunTenantScopeAndTransitions(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenantA, tenantB, id := "it-a-"+suffix, "it-b-"+suffix, "it-run-"+suffix
	cleanupRun(t, ctx, pool, tenantA, id)
	cleanupRun(t, ctx, pool, tenantB, id)
	if err := store.CreateRun(ctx, integrationRun(tenantA, id)); err != nil {
		t.Fatalf("create run: %v", err)
	}
	got, err := store.GetRun(ctx, tenantA, id)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.TenantID != tenantA || got.Status != RunQueued || got.Metadata["suite"] != "runtime-v6" {
		t.Fatalf("unexpected run: %#v", got)
	}
	if _, err := store.GetRun(ctx, tenantB, id); err != ErrNotFound {
		t.Fatalf("cross-tenant read = %v, want ErrNotFound", err)
	}
	if err := store.TransitionRun(ctx, tenantA, id, RunQueued, RunRunning, ""); err != nil {
		t.Fatalf("transition to running: %v", err)
	}
	if err := store.TransitionRun(ctx, tenantA, id, RunQueued, RunRunning, ""); err != ErrConflict {
		t.Fatalf("stale transition = %v, want ErrConflict", err)
	}
	if err := store.TransitionRun(ctx, tenantA, id, RunRunning, RunSucceeded, ""); err != nil {
		t.Fatalf("transition to succeeded: %v", err)
	}
	got, err = store.GetRun(ctx, tenantA, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.FinishedAt == nil || got.StartedAt == nil || got.Status != RunSucceeded {
		t.Fatalf("terminal timestamps/status invalid: %#v", got)
	}
}

func TestPostgresStore_RunIdempotencyIsTenantScoped(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenantA, tenantB := "it-idem-a-"+suffix, "it-idem-b-"+suffix
	key := "request-" + suffix
	runA := integrationRun(tenantA, "it-run-a-"+suffix)
	runA.IdempotencyKey = key
	runB := integrationRun(tenantB, "it-run-b-"+suffix)
	runB.IdempotencyKey = key
	cleanupRun(t, ctx, pool, tenantA, runA.ID)
	cleanupRun(t, ctx, pool, tenantB, runB.ID)

	if err := store.CreateRun(ctx, runA); err != nil {
		t.Fatalf("create first idempotent run: %v", err)
	}
	duplicate := integrationRun(tenantA, "it-run-duplicate-"+suffix)
	duplicate.IdempotencyKey = key
	if err := store.CreateRun(ctx, duplicate); err == nil {
		t.Fatal("create run with duplicate tenant idempotency key succeeded; want conflict")
	}
	if err := store.CreateRun(ctx, runB); err != nil {
		t.Fatalf("same idempotency key in another tenant should be allowed: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM v6_runs WHERE tenant_id=$1 AND idempotency_key=$2`, tenantA, key).Scan(&count); err != nil {
		t.Fatalf("count idempotent runs: %v", err)
	}
	if count != 1 {
		t.Fatalf("tenant A has %d runs for one idempotency key, want 1", count)
	}
}

func TestPostgresStore_RetryEventAndOutboxAreAtomic(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenant, runID, taskID := "it-retry-"+suffix, "it-run-"+suffix, "it-task-"+suffix
	eventID := "it-event-" + suffix
	cleanupRun(t, ctx, pool, tenant, runID)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := pool.Exec(cleanupCtx, `DELETE FROM outbox_events WHERE event_id=$1`, eventID)
		if err != nil {
			t.Errorf("cleanup integration outbox event: %v", err)
		}
	})
	if err := store.CreateRun(ctx, integrationRun(tenant, runID)); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := store.CreateTask(ctx, integrationTask(tenant, runID, taskID)); err != nil {
		t.Fatalf("create task: %v", err)
	}
	claimed, err := store.ClaimTask(ctx, tenant, "retry-worker", time.Minute)
	if err != nil {
		t.Fatalf("claim task: %v", err)
	}
	event := ExecutionEvent{ID: eventID, SchemaVersion: 1, TenantID: tenant, RunID: runID, TaskID: taskID, CorrelationID: claimed.CorrelationID, Type: "task.retrying", Metadata: map[string]string{"attempt": "1"}}
	if err := store.RetryTaskWithEvent(ctx, tenant, taskID, "retry-worker", time.Second, "handler_failed", event); err != nil {
		t.Fatalf("retry task with event: %v", err)
	}
	var status, owner string
	var leaseUntil *time.Time
	if err := pool.QueryRow(ctx, `SELECT status,lease_owner,lease_until FROM v6_tasks WHERE tenant_id=$1 AND id=$2`, tenant, taskID).Scan(&status, &owner, &leaseUntil); err != nil {
		t.Fatal(err)
	}
	if status != string(TaskRetrying) || owner != "" || leaseUntil != nil {
		t.Fatalf("unexpected retry state status=%q owner=%q until=%v", status, owner, leaseUntil)
	}
	var lineageType, outboxType, outboxStatus string
	if err := pool.QueryRow(ctx, `SELECT event_type FROM v6_execution_events WHERE id=$1 AND tenant_id=$2`, eventID, tenant).Scan(&lineageType); err != nil {
		t.Fatalf("read retry lineage: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT event_type,status FROM outbox_events WHERE event_id=$1 AND tenant_id=$2`, eventID, tenant).Scan(&outboxType, &outboxStatus); err != nil {
		t.Fatalf("read retry outbox: %v", err)
	}
	if lineageType != "task.retrying" || outboxType != lineageType || outboxStatus != "pending" {
		t.Fatalf("unexpected retry events lineage=%q outbox=%q status=%q", lineageType, outboxType, outboxStatus)
	}
}

func TestPostgresStore_RejectsMismatchedRunTaskLineage(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenant := "it-lineage-" + suffix
	runA, runB := "it-run-a-"+suffix, "it-run-b-"+suffix
	taskID := "it-task-" + suffix
	cleanupRun(t, ctx, pool, tenant, runA)
	cleanupRun(t, ctx, pool, tenant, runB)
	if err := store.CreateRun(ctx, integrationRun(tenant, runA)); err != nil {
		t.Fatalf("create run A: %v", err)
	}
	if err := store.CreateRun(ctx, integrationRun(tenant, runB)); err != nil {
		t.Fatalf("create run B: %v", err)
	}
	if err := store.CreateTask(ctx, integrationTask(tenant, runA, taskID)); err != nil {
		t.Fatalf("create task: %v", err)
	}

	artifact := Artifact{ID: "it-artifact-" + suffix, TenantID: tenant, RunID: runB, TaskID: taskID, Kind: "result", StorageRef: "test://artifact", CreatedBy: "integration-test"}
	if err := store.CreateArtifact(ctx, artifact); err == nil {
		t.Fatal("CreateArtifact accepted a task from a different run")
	}

	event := ExecutionEvent{ID: "it-event-lineage-" + suffix, SchemaVersion: 1, TenantID: tenant, RunID: runB, TaskID: taskID, CorrelationID: "corr-" + runB, Type: "task.observed"}
	if err := store.AppendExecutionEvent(ctx, event); err == nil {
		t.Fatal("AppendExecutionEvent accepted a task from a different run")
	}
}

func TestPostgresStore_ConcurrentClaimsAndLeaseRecovery(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenant, runID := "it-lease-"+suffix, "it-run-"+suffix
	cleanupRun(t, ctx, pool, tenant, runID)
	if err := store.CreateRun(ctx, integrationRun(tenant, runID)); err != nil {
		t.Fatalf("create run: %v", err)
	}
	for i := 0; i < 8; i++ {
		if err := store.CreateTask(ctx, integrationTask(tenant, runID, fmt.Sprintf("it-task-%s-%02d", suffix, i))); err != nil {
			t.Fatalf("create task %d: %v", i, err)
		}
	}
	const workers = 8
	claimed := make(chan string, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			task, err := store.ClaimTask(ctx, tenant, fmt.Sprintf("worker-%d", i), time.Minute)
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			claimed <- task.ID
		}(i)
	}
	wg.Wait()
	close(claimed)
	seen := map[string]bool{}
	for id := range claimed {
		if seen[id] {
			t.Errorf("task %s claimed more than once", id)
		}
		seen[id] = true
	}
	if len(seen) != workers {
		t.Fatalf("claimed %d unique tasks, want %d", len(seen), workers)
	}

	var taskID string
	if err := pool.QueryRow(ctx, `SELECT id FROM v6_tasks WHERE tenant_id=$1 AND run_id=$2 ORDER BY id LIMIT 1`, tenant, runID).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	if err := store.RenewLease(ctx, tenant, taskID, "not-the-owner", time.Minute); err != ErrConflict {
		t.Fatalf("wrong-owner lease renewal = %v, want ErrConflict", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE v6_tasks SET lease_until=NOW()-INTERVAL '1 second' WHERE tenant_id=$1 AND id=$2`, tenant, taskID); err != nil {
		t.Fatal(err)
	}
	n, err := store.RecoverExpiredLeases(ctx)
	if err != nil {
		t.Fatalf("recover leases: %v", err)
	}
	if n < 1 {
		t.Fatalf("recovered %d expired leases, want at least one", n)
	}
	var status, owner string
	var leaseUntil *time.Time
	if err := pool.QueryRow(ctx, `SELECT status,lease_owner,lease_until FROM v6_tasks WHERE tenant_id=$1 AND id=$2`, tenant, taskID).Scan(&status, &owner, &leaseUntil); err != nil {
		t.Fatal(err)
	}
	if status != string(TaskRetrying) || owner != "" || leaseUntil != nil {
		t.Fatalf("recovered lease state status=%q owner=%q until=%v", status, owner, leaseUntil)
	}
}

func TestPostgresStore_ExpiredLeaseAtAttemptLimitFailsTask(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenant, runID, taskID := "it-exhausted-"+suffix, "it-run-"+suffix, "it-task-"+suffix
	cleanupRun(t, ctx, pool, tenant, runID)
	if err := store.CreateRun(ctx, integrationRun(tenant, runID)); err != nil {
		t.Fatalf("create run: %v", err)
	}
	task := integrationTask(tenant, runID, taskID)
	task.MaxAttempts = 1
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, err := store.ClaimTask(ctx, tenant, "worker-exhausted", time.Minute); err != nil {
		t.Fatalf("claim task: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE v6_tasks SET lease_until=NOW()-INTERVAL '1 second' WHERE tenant_id=$1 AND id=$2`, tenant, taskID); err != nil {
		t.Fatalf("expire task lease: %v", err)
	}
	if n, err := store.RecoverExpiredLeases(ctx); err != nil || n != 1 {
		t.Fatalf("recover expired lease: recovered=%d err=%v", n, err)
	}

	var status, owner, errorCode string
	var leaseUntil, finishedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT status,lease_owner,lease_until,error_code,finished_at FROM v6_tasks WHERE tenant_id=$1 AND id=$2`, tenant, taskID).Scan(&status, &owner, &leaseUntil, &errorCode, &finishedAt); err != nil {
		t.Fatalf("read recovered task: %v", err)
	}
	if status != string(TaskFailed) || owner != "" || leaseUntil != nil || errorCode != "lease_expired_attempts_exhausted" || finishedAt == nil {
		t.Fatalf("unexpected exhausted task state status=%q owner=%q lease=%v error=%q finished=%v", status, owner, leaseUntil, errorCode, finishedAt)
	}
}

func TestPostgresStore_ExecutionEventIsPublishedFromOutbox(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenant, runID := "it-publish-"+suffix, "it-run-"+suffix
	eventID := "it-event-publish-" + suffix
	cleanupRun(t, ctx, pool, tenant, runID)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM outbox_events WHERE event_id=$1`, eventID); err != nil {
			t.Errorf("cleanup integration outbox event: %v", err)
		}
	})
	if err := store.CreateRun(ctx, integrationRun(tenant, runID)); err != nil {
		t.Fatalf("create run: %v", err)
	}

	bus := events.NewInMemoryBus()
	if err := bus.Start(ctx); err != nil {
		t.Fatalf("start event bus: %v", err)
	}
	defer func() { _ = bus.Stop() }()
	received := make(chan events.Event, 1)
	const eventType = "task.completed"
	if err := bus.Subscribe(eventType, events.NewConsumer("runtime-v6-integration", func(_ context.Context, event events.Event) {
		select {
		case received <- event:
		default:
		}
	})); err != nil {
		t.Fatalf("subscribe to event bus: %v", err)
	}

	executionEvent := ExecutionEvent{
		ID: eventID, SchemaVersion: 1, TenantID: tenant, RunID: runID,
		CorrelationID: "corr-" + runID, Type: eventType,
		Metadata: map[string]string{"source": "runtime-v6-integration"},
	}
	if err := store.AppendExecutionEvent(ctx, executionEvent); err != nil {
		t.Fatalf("append execution event: %v", err)
	}

	cfg := events.DefaultOutboxConfig()
	cfg.PollInterval = 25 * time.Millisecond
	cfg.BatchSize = 10
	cfg.StuckThreshold = time.Minute
	publisher := events.NewOutbox(pool, bus, cfg)
	publisherCtx, cancelPublisher := context.WithCancel(ctx)
	defer cancelPublisher()
	go func() { _ = publisher.Run(publisherCtx) }()

	select {
	case got := <-received:
		if got.ID != eventID || got.Type != eventType || got.TenantID != tenant {
			t.Fatalf("published event identity mismatch: %#v", got)
		}
		if got.Payload["run_id"] != runID || got.Payload["correlation_id"] != executionEvent.CorrelationID {
			t.Fatalf("published event payload mismatch: %#v", got.Payload)
		}
		metadata, ok := got.Payload["metadata"].(map[string]interface{})
		if !ok || metadata["source"] != "runtime-v6-integration" {
			t.Fatalf("published event metadata mismatch: %#v", got.Payload["metadata"])
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for outbox delivery: %v", ctx.Err())
	case <-time.After(8 * time.Second):
		t.Fatal("timed out waiting for outbox delivery")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		var status string
		err := pool.QueryRow(ctx, `SELECT status FROM outbox_events WHERE event_id=$1`, eventID).Scan(&status)
		if err != nil {
			t.Fatalf("read outbox status: %v", err)
		}
		if status == "published" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("outbox status = %q, want published", status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestPostgresStore_ExpiredWorkerCannotCompleteReclaimedTask(t *testing.T) {
	pool, store, ctx := integrationStore(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tenant, runID, taskID := "it-fence-"+suffix, "it-run-"+suffix, "it-task-"+suffix
	staleEventID, staleRetryEventID, currentEventID := "it-event-stale-"+suffix, "it-event-stale-retry-"+suffix, "it-event-current-"+suffix
	cleanupRun(t, ctx, pool, tenant, runID)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, eventID := range []string{staleEventID, staleRetryEventID, currentEventID} {
			if _, err := pool.Exec(cleanupCtx, `DELETE FROM outbox_events WHERE event_id=$1`, eventID); err != nil {
				t.Errorf("cleanup outbox event %s: %v", eventID, err)
			}
		}
	})
	if err := store.CreateRun(ctx, integrationRun(tenant, runID)); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := store.CreateTask(ctx, integrationTask(tenant, runID, taskID)); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, err := store.ClaimTask(ctx, tenant, "worker-stale", time.Minute); err != nil {
		t.Fatalf("claim task with first worker: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE v6_tasks SET lease_until=NOW()-INTERVAL '1 second' WHERE tenant_id=$1 AND id=$2`, tenant, taskID); err != nil {
		t.Fatalf("expire first lease: %v", err)
	}
	if n, err := store.RecoverExpiredLeases(ctx); err != nil || n != 1 {
		t.Fatalf("recover expired lease: recovered=%d err=%v", n, err)
	}
	if _, err := store.ClaimTask(ctx, tenant, "worker-current", time.Minute); err != nil {
		t.Fatalf("claim task with replacement worker: %v", err)
	}

	if err := store.RenewLease(ctx, tenant, taskID, "worker-stale", time.Minute); err != ErrConflict {
		t.Fatalf("stale worker lease renewal = %v, want ErrConflict", err)
	}
	staleRetryEvent := ExecutionEvent{ID: staleRetryEventID, SchemaVersion: 1, TenantID: tenant, RunID: runID, TaskID: taskID, CorrelationID: "corr-" + runID, Type: "task.retrying"}
	if err := store.RetryTaskWithEvent(ctx, tenant, taskID, "worker-stale", time.Second, "stale_retry", staleRetryEvent); err != ErrConflict {
		t.Fatalf("stale worker retry = %v, want ErrConflict", err)
	}
	var staleRetryCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM v6_execution_events WHERE id=$1`, staleRetryEventID).Scan(&staleRetryCount); err != nil {
		t.Fatalf("check stale retry lineage: %v", err)
	}
	if staleRetryCount != 0 {
		t.Fatalf("stale worker wrote %d retry lineage events, want 0", staleRetryCount)
	}
	if err := store.RetryTask(ctx, tenant, taskID, "worker-stale", time.Second, "stale_retry"); err != ErrConflict {
		t.Fatalf("stale worker retry without event = %v, want ErrConflict", err)
	}
	var ownerAfterStaleRetry string
	var statusAfterStaleRetry string
	if err := pool.QueryRow(ctx, `SELECT status,lease_owner FROM v6_tasks WHERE tenant_id=$1 AND id=$2`, tenant, taskID).Scan(&statusAfterStaleRetry, &ownerAfterStaleRetry); err != nil {
		t.Fatalf("read task after stale retry: %v", err)
	}
	if statusAfterStaleRetry != string(TaskRunning) || ownerAfterStaleRetry != "worker-current" {
		t.Fatalf("stale retry changed current task state: status=%q owner=%q", statusAfterStaleRetry, ownerAfterStaleRetry)
	}

	staleEvent := ExecutionEvent{ID: staleEventID, SchemaVersion: 1, TenantID: tenant, RunID: runID, TaskID: taskID, CorrelationID: "corr-" + runID, Type: "task.completed"}
	if err := store.CompleteTaskWithEvent(ctx, tenant, taskID, "worker-stale", TaskSucceeded, "", staleEvent); err != ErrConflict {
		t.Fatalf("stale worker completion = %v, want ErrConflict", err)
	}
	var staleCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM v6_execution_events WHERE id=$1`, staleEventID).Scan(&staleCount); err != nil {
		t.Fatalf("check stale lineage: %v", err)
	}
	if staleCount != 0 {
		t.Fatalf("stale worker wrote %d lineage events, want 0", staleCount)
	}

	currentEvent := ExecutionEvent{ID: currentEventID, SchemaVersion: 1, TenantID: tenant, RunID: runID, TaskID: taskID, CorrelationID: "corr-" + runID, Type: "task.completed"}
	if err := store.CompleteTaskWithEvent(ctx, tenant, taskID, "worker-current", TaskSucceeded, "", currentEvent); err != nil {
		t.Fatalf("current worker completion: %v", err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM v6_tasks WHERE tenant_id=$1 AND id=$2`, tenant, taskID).Scan(&status); err != nil {
		t.Fatalf("read final task status: %v", err)
	}
	if status != string(TaskSucceeded) {
		t.Fatalf("final task status = %q, want %q", status, TaskSucceeded)
	}
}
