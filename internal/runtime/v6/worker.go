package runtimev6

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TaskHandler performs one task attempt. Handlers must honor ctx cancellation
// and make external side effects idempotent; lease fencing cannot undo an
// already-issued request to another system.
type TaskHandler func(context.Context, Task) (ExecutionEvent, error)

// Worker runs bounded task attempts against the durable PostgreSQL store.
type Worker struct {
	Store        *PostgresStore
	TenantID     string
	WorkerID     string
	Lease        time.Duration
	PollInterval time.Duration
	Handler      TaskHandler
}

// Run polls until cancellation. ErrNotFound means the queue is currently empty.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.validate(); err != nil {
		return err
	}
	interval := w.PollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil && !errors.Is(err, ErrNotFound) {
			// A lost lease is an expected concurrency outcome, not a reason to
			// stop the worker process. The next poll can claim another task.
			if !errors.Is(err, ErrConflict) && ctx.Err() == nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// RunOnce claims and executes at most one task. It renews the lease while the
// handler runs, cancels the handler on lease loss, and commits completion only
// if the store still recognizes this worker as the live lease owner.
func (w *Worker) RunOnce(ctx context.Context) error {
	if err := w.validate(); err != nil {
		return err
	}
	task, err := w.Store.ClaimTask(ctx, w.TenantID, w.WorkerID, w.Lease)
	if err != nil {
		return err
	}
	attemptCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	leaseErr := make(chan error, 1)
	renewEvery := w.Lease / 3
	if renewEvery <= 0 {
		renewEvery = time.Millisecond
	}
	renewDone := make(chan struct{})
	go func() {
		defer close(renewDone)
		ticker := time.NewTicker(renewEvery)
		defer ticker.Stop()
		for {
			select {
			case <-attemptCtx.Done():
				return
			case <-ticker.C:
				if err := w.Store.RenewLease(attemptCtx, w.TenantID, task.ID, w.WorkerID, w.Lease); err != nil {
					select {
					case leaseErr <- err:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()

	event, handlerErr := w.Handler(attemptCtx, task)
	cancel()
	<-renewDone
	select {
	case err := <-leaseErr:
		return fmt.Errorf("task lease lost: %w", err)
	default:
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	status := TaskSucceeded
	errorCode := ""
	if handlerErr != nil {
		status = TaskFailed
		errorCode = "handler_failed"
		if task.Attempt < task.MaxAttempts {
			status = TaskRetrying
		}
	}
	if status == TaskRetrying {
		// Retry eligibility is durable and bounded; backoff avoids hot-looping
		// a failing dependency. A subsequent claim increments the attempt.
		backoff := time.Duration(task.Attempt) * time.Second
		if backoff > time.Minute {
			backoff = time.Minute
		}
		if err := w.Store.RetryTask(ctx, w.TenantID, task.ID, w.WorkerID, backoff, errorCode); err != nil {
			return err
		}
		return nil
	}
	if status == TaskFailed {
		event.Type = "task.failed"
	} else {
		event.Type = "task.succeeded"
	}
	if event.ID == "" {
		return fmt.Errorf("handler must return an execution event with a stable ID")
	}
	if err := w.Store.CompleteTaskWithEvent(ctx, w.TenantID, task.ID, w.WorkerID, status, errorCode, event); err != nil {
		return err
	}
	return handlerErr
}

func (w *Worker) validate() error {
	if w == nil || w.Store == nil || w.TenantID == "" || w.WorkerID == "" || w.Lease <= 0 || w.Handler == nil {
		return fmt.Errorf("worker requires store, tenant, worker id, positive lease and handler")
	}
	return nil
}
