package runtimev6

import (
	"errors"
	"testing"
	"time"
)

func TestRunValidateRequiresTenantAndPrincipal(t *testing.T) {
	now := time.Now().UTC()
	run := Run{ID: "run-1", TenantID: "tenant-1", PrincipalID: "user-1", CorrelationID: "corr-1", Status: RunPending, CreatedAt: now, UpdatedAt: now}
	if err := run.Validate(); err != nil {
		t.Fatalf("valid run rejected: %v", err)
	}
	run.TenantID = ""
	if err := run.Validate(); !errors.Is(err, ErrInvalidRun) {
		t.Fatalf("expected ErrInvalidRun, got %v", err)
	}
}

func TestRunValidateRejectsInvalidTimeOrdering(t *testing.T) {
	started := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	finished := started.Add(-time.Second)
	run := Run{ID: "r", TenantID: "t", PrincipalID: "p", CorrelationID: "c", Status: RunFailed, StartedAt: &started, FinishedAt: &finished}
	if err := run.Validate(); !errors.Is(err, ErrInvalidRun) {
		t.Fatalf("expected invalid timestamp error, got %v", err)
	}
}

func TestTaskValidateEnforcesRetryAndLeaseInvariants(t *testing.T) {
	task := Task{ID: "task-1", RunID: "run-1", TenantID: "tenant-1", CorrelationID: "corr-1", Status: TaskQueued, MaxAttempts: 3}
	if err := task.Validate(); err != nil {
		t.Fatalf("valid task rejected: %v", err)
	}
	task.Attempt = 4
	if err := task.Validate(); !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("expected attempt validation error, got %v", err)
	}
	task.Attempt = 1
	until := time.Now().UTC().Add(time.Minute)
	task.LeaseUntil = &until
	if err := task.Validate(); !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("expected lease owner validation error, got %v", err)
	}
}

func TestRunTransitionsRejectTerminalReopening(t *testing.T) {
	allowed := [][2]RunStatus{{RunPending, RunQueued}, {RunQueued, RunRunning}, {RunRunning, RunSucceeded}, {RunRunning, RunFailed}, {RunRunning, RunCancelled}}
	for _, pair := range allowed {
		if !CanTransitionRun(pair[0], pair[1]) {
			t.Errorf("expected run transition %s -> %s", pair[0], pair[1])
		}
	}
	for _, pair := range [][2]RunStatus{{RunSucceeded, RunRunning}, {RunFailed, RunQueued}, {RunCancelled, RunPending}, {RunPending, RunSucceeded}} {
		if CanTransitionRun(pair[0], pair[1]) {
			t.Errorf("unexpected run transition %s -> %s", pair[0], pair[1])
		}
	}
}

func TestTaskTransitionsModelRetryWithoutReopeningTerminalStates(t *testing.T) {
	if !CanTransitionTask(TaskRunning, TaskRetrying) || !CanTransitionTask(TaskRetrying, TaskQueued) {
		t.Fatal("expected explicit retry transition")
	}
	for _, pair := range [][2]TaskStatus{{TaskSucceeded, TaskQueued}, {TaskFailed, TaskRunning}, {TaskCancelled, TaskPending}} {
		if CanTransitionTask(pair[0], pair[1]) {
			t.Errorf("unexpected task transition %s -> %s", pair[0], pair[1])
		}
	}
}
