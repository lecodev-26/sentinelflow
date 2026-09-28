package executor

import (
	"context"
	"errors"
	providers "github.com/lecodev-26/sentinelflow/internal/providers/v3"
	routing "github.com/lecodev-26/sentinelflow/internal/routing/v3"
	"testing"
	"time"
)

type chaosProvider struct {
	id   string
	fail bool
}

func (p *chaosProvider) ID() string   { return p.id }
func (p *chaosProvider) Name() string { return p.id }
func (p *chaosProvider) Capabilities() []providers.Capability {
	return []providers.Capability{providers.CapChat}
}
func (p *chaosProvider) Models(context.Context) ([]providers.ModelInfo, error) { return nil, nil }
func (p *chaosProvider) Chat(context.Context, *providers.ChatRequest) (*providers.ChatResponse, error) {
	if p.fail {
		return nil, errors.New("chaos provider failure")
	}
	return &providers.ChatResponse{ID: "ok", Provider: p.id, Model: "test", Timestamp: time.Now()}, nil
}
func (p *chaosProvider) Stream(context.Context, *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	if p.fail {
		return nil, errors.New("chaos stream failure")
	}
	ch := make(chan providers.StreamChunk, 1)
	ch <- providers.StreamChunk{Delta: "ok"}
	close(ch)
	return ch, nil
}
func (p *chaosProvider) Health(context.Context) error {
	if p.fail {
		return errors.New("unhealthy")
	}
	return nil
}

func TestChaosFailoverToSecondProvider(t *testing.T) {
	reg := providers.NewRegistry()
	bad := &chaosProvider{id: "bad", fail: true}
	good := &chaosProvider{id: "good"}
	if err := reg.Register(bad); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(good); err != nil {
		t.Fatal(err)
	}
	mgr := providers.NewManager(reg)
	defer mgr.Stop()
	ex := NewExecutor(reg, mgr)
	cs := []routing.Candidate{{ProviderID: "bad"}, {ProviderID: "good"}}
	resp, attempts, err := ex.ExecuteChat(context.Background(), cs, &providers.ChatRequest{Model: "test"})
	if err != nil || resp == nil {
		t.Fatalf("failover failed: %v", err)
	}
	if len(attempts) != 2 || attempts[0].Success || !attempts[1].Success {
		t.Fatalf("unexpected attempts: %#v", attempts)
	}
}

func TestChaosAllProvidersFail(t *testing.T) {
	reg := providers.NewRegistry()
	_ = reg.Register(&chaosProvider{id: "bad1", fail: true})
	_ = reg.Register(&chaosProvider{id: "bad2", fail: true})
	mgr := providers.NewManager(reg)
	defer mgr.Stop()
	ex := NewExecutor(reg, mgr)
	_, attempts, err := ex.ExecuteChat(context.Background(), []routing.Candidate{{ProviderID: "bad1"}, {ProviderID: "bad2"}}, &providers.ChatRequest{Model: "test"})
	if err == nil || len(attempts) != 2 {
		t.Fatalf("expected complete failure, attempts=%d err=%v", len(attempts), err)
	}
}
