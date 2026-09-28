package agentsv5

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Status string

const (
	StatusPending         Status = "pending"
	StatusRunning         Status = "running"
	StatusWaitingApproval Status = "waiting_approval"
	StatusCompleted       Status = "completed"
	StatusFailed          Status = "failed"
	StatusCancelled       Status = "cancelled"
)

type Step struct {
	ID               string
	Kind             string
	Tool             string
	Input            map[string]any
	RequiresApproval bool
}
type Result struct {
	StepID   string
	Output   any
	Err      error
	Duration time.Duration
}
type Run struct {
	ID, AgentID           string
	Status                Status
	Steps                 []Step
	Results               []Result
	StartedAt, FinishedAt time.Time
}
type Executor interface {
	Execute(context.Context, Step) (any, error)
}
type Approval interface {
	Request(context.Context, Run, Step) (bool, error)
}
type Runtime struct {
	mu       sync.RWMutex
	runs     map[string]*Run
	exec     Executor
	approval Approval
}

func NewRuntime(exec Executor, approval Approval) *Runtime {
	return &Runtime{runs: map[string]*Run{}, exec: exec, approval: approval}
}
func (r *Runtime) Start(ctx context.Context, run *Run) (*Run, error) {
	if run == nil || run.ID == "" || run.AgentID == "" {
		return nil, errors.New("run and agent id required")
	}
	r.mu.Lock()
	if _, ok := r.runs[run.ID]; ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("run %s already exists", run.ID)
	}
	run.Status = StatusRunning
	run.StartedAt = time.Now().UTC()
	r.runs[run.ID] = run
	r.mu.Unlock()
	for _, step := range run.Steps {
		select {
		case <-ctx.Done():
			r.finish(run, StatusCancelled)
			return run, ctx.Err()
		default:
		}
		if step.RequiresApproval && r.approval != nil {
			ok, err := r.approval.Request(ctx, *run, step)
			if err != nil {
				r.finish(run, StatusFailed)
				return run, err
			}
			if !ok {
				r.finish(run, StatusCancelled)
				return run, fmt.Errorf("approval denied for step %s", step.ID)
			}
		}
		start := time.Now()
		out, err := r.exec.Execute(ctx, step)
		run.Results = append(run.Results, Result{StepID: step.ID, Output: out, Err: err, Duration: time.Since(start)})
		if err != nil {
			r.finish(run, StatusFailed)
			return run, err
		}
	}
	r.finish(run, StatusCompleted)
	return run, nil
}
func (r *Runtime) finish(run *Run, s Status) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run.Status = s
	run.FinishedAt = time.Now().UTC()
}
func (r *Runtime) Cancel(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[id]
	if !ok || run.Status != StatusRunning {
		return false
	}
	run.Status = StatusCancelled
	run.FinishedAt = time.Now().UTC()
	return true
}
func (r *Runtime) Get(id string) (*Run, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.runs[id]
	return x, ok
}
