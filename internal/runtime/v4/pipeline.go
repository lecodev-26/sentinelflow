package runtimev4

import (
	"context"
	"fmt"
	"time"
)

type Decision struct {
	RequestID, TenantID, ProjectID, Provider, Model string
	PolicyVersion                                   string
	Allowed                                         bool
	Reasons                                         []string
}
type Context struct {
	Context  context.Context
	Decision Decision
	Metadata map[string]string
}
type Stage interface {
	Name() string
	Handle(*Context) error
}
type StageFunc struct {
	StageName string
	Fn        func(*Context) error
}

func (s StageFunc) Name() string            { return s.StageName }
func (s StageFunc) Handle(c *Context) error { return s.Fn(c) }

type Result struct {
	Decision Decision
	Latency  map[string]time.Duration
}
type Pipeline struct{ stages []Stage }

func New(stages ...Stage) *Pipeline { return &Pipeline{stages: stages} }
func (p *Pipeline) Run(c *Context) (Result, error) {
	if c == nil {
		return Result{}, fmt.Errorf("nil runtime context")
	}
	if c.Metadata == nil {
		c.Metadata = map[string]string{}
	}
	lat := map[string]time.Duration{}
	for _, s := range p.stages {
		start := time.Now()
		if err := s.Handle(c); err != nil {
			return Result{Decision: c.Decision, Latency: lat}, fmt.Errorf("%s: %w", s.Name(), err)
		}
		lat[s.Name()] = time.Since(start)
	}
	return Result{Decision: c.Decision, Latency: lat}, nil
}
