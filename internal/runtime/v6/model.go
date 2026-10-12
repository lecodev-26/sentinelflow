// Package runtimev6 defines the durable execution domain contracts for SentinelFlow V6.
//
// These types are deliberately storage-agnostic. Persistence, worker leases, and
// scheduling must implement these contracts without weakening their invariants.
package runtimev6

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunQueued    RunStatus = "queued"
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskQueued    TaskStatus = "queued"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
	TaskRetrying  TaskStatus = "retrying"
)

var (
	ErrInvalidTransition = errors.New("invalid lifecycle transition")
	ErrInvalidRun        = errors.New("invalid run")
	ErrInvalidTask       = errors.New("invalid task")
)

// Run is the tenant-scoped durable unit of requested work. Payload is an
// application-owned opaque reference; sensitive payloads should not be copied
// into logs or event metadata.
type Run struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	PrincipalID    string            `json:"principal_id"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	CorrelationID  string            `json:"correlation_id"`
	Status         RunStatus         `json:"status"`
	PayloadRef     string            `json:"payload_ref,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	StartedAt      *time.Time        `json:"started_at,omitempty"`
	FinishedAt     *time.Time        `json:"finished_at,omitempty"`
	ErrorCode      string            `json:"error_code,omitempty"`
}

// Task is a bounded, retryable unit of work owned by a Run.
type Task struct {
	ID            string            `json:"id"`
	RunID         string            `json:"run_id"`
	TenantID      string            `json:"tenant_id"`
	CorrelationID string            `json:"correlation_id"`
	Status        TaskStatus        `json:"status"`
	Attempt       uint32            `json:"attempt"`
	MaxAttempts   uint32            `json:"max_attempts"`
	AvailableAt   time.Time         `json:"available_at"`
	LeaseOwner    string            `json:"lease_owner,omitempty"`
	LeaseUntil    *time.Time        `json:"lease_until,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	FinishedAt    *time.Time        `json:"finished_at,omitempty"`
	ErrorCode     string            `json:"error_code,omitempty"`
}

// Artifact records a tenant-scoped input, intermediate value, or output.
// Data is referenced rather than embedded so persistence and access policy can
// be enforced by the artifact store.
type Artifact struct {
	ID            string    `json:"id"`
	RunID         string    `json:"run_id"`
	TaskID        string    `json:"task_id,omitempty"`
	TenantID      string    `json:"tenant_id"`
	Kind          string    `json:"kind"`
	MediaType     string    `json:"media_type,omitempty"`
	StorageRef    string    `json:"storage_ref"`
	Checksum      string    `json:"checksum,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedBy     string    `json:"created_by"`
	PolicyVersion string    `json:"policy_version,omitempty"`
}

// ExecutionEvent is an immutable, versioned lifecycle fact. Payload should
// contain only non-secret metadata and must be serialized by the event codec.
type ExecutionEvent struct {
	ID            string            `json:"id"`
	SchemaVersion uint32            `json:"schema_version"`
	TenantID      string            `json:"tenant_id"`
	RunID         string            `json:"run_id"`
	TaskID        string            `json:"task_id,omitempty"`
	CorrelationID string            `json:"correlation_id"`
	Type          string            `json:"type"`
	OccurredAt    time.Time         `json:"occurred_at"`
	PrincipalID   string            `json:"principal_id,omitempty"`
	PolicyVersion string            `json:"policy_version,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func (r Run) Validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.TenantID) == "" ||
		strings.TrimSpace(r.PrincipalID) == "" || strings.TrimSpace(r.CorrelationID) == "" {
		return fmt.Errorf("%w: id, tenant, principal and correlation id are required", ErrInvalidRun)
	}
	if !validRunStatus(r.Status) {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidRun, r.Status)
	}
	if !r.CreatedAt.IsZero() && !r.UpdatedAt.IsZero() && r.UpdatedAt.Before(r.CreatedAt) {
		return fmt.Errorf("%w: updated_at precedes created_at", ErrInvalidRun)
	}
	if r.FinishedAt != nil && r.StartedAt == nil {
		return fmt.Errorf("%w: finished run must have started_at", ErrInvalidRun)
	}
	if r.StartedAt != nil && r.FinishedAt != nil && r.FinishedAt.Before(*r.StartedAt) {
		return fmt.Errorf("%w: finished_at precedes started_at", ErrInvalidRun)
	}
	return nil
}

func (t Task) Validate() error {
	if strings.TrimSpace(t.ID) == "" || strings.TrimSpace(t.RunID) == "" ||
		strings.TrimSpace(t.TenantID) == "" || strings.TrimSpace(t.CorrelationID) == "" {
		return fmt.Errorf("%w: id, run, tenant and correlation id are required", ErrInvalidTask)
	}
	if !validTaskStatus(t.Status) {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidTask, t.Status)
	}
	if t.MaxAttempts == 0 {
		return fmt.Errorf("%w: max_attempts must be positive", ErrInvalidTask)
	}
	if t.Attempt > t.MaxAttempts {
		return fmt.Errorf("%w: attempt exceeds max_attempts", ErrInvalidTask)
	}
	if t.LeaseUntil != nil && strings.TrimSpace(t.LeaseOwner) == "" {
		return fmt.Errorf("%w: lease expiry requires a lease owner", ErrInvalidTask)
	}
	return nil
}

func CanTransitionRun(from, to RunStatus) bool {
	switch from {
	case RunPending:
		return to == RunQueued || to == RunCancelled || to == RunFailed
	case RunQueued:
		return to == RunRunning || to == RunCancelled || to == RunFailed
	case RunRunning:
		return to == RunSucceeded || to == RunFailed || to == RunCancelled
	default:
		return false
	}
}

func CanTransitionTask(from, to TaskStatus) bool {
	switch from {
	case TaskPending:
		return to == TaskQueued || to == TaskCancelled || to == TaskFailed
	case TaskQueued:
		return to == TaskRunning || to == TaskCancelled || to == TaskFailed
	case TaskRunning:
		return to == TaskSucceeded || to == TaskFailed || to == TaskCancelled || to == TaskRetrying
	case TaskRetrying:
		return to == TaskQueued || to == TaskFailed || to == TaskCancelled
	default:
		return false
	}
}

func validRunStatus(s RunStatus) bool {
	switch s {
	case RunPending, RunQueued, RunRunning, RunSucceeded, RunFailed, RunCancelled:
		return true
	default:
		return false
	}
}

func validTaskStatus(s TaskStatus) bool {
	switch s {
	case TaskPending, TaskQueued, TaskRunning, TaskSucceeded, TaskFailed, TaskCancelled, TaskRetrying:
		return true
	default:
		return false
	}
}
