package v3

import (
	"context"
	providers "github.com/lecodev-26/sentinelflow/internal/providers/v3"
	"testing"
	"time"
)

type simulationProvider struct{ id, name string }

func (p *simulationProvider) ID() string   { return p.id }
func (p *simulationProvider) Name() string { return p.name }
func (p *simulationProvider) Capabilities() []providers.Capability {
	return []providers.Capability{providers.CapChat}
}
func (p *simulationProvider) Models(context.Context) ([]providers.ModelInfo, error) { return nil, nil }
func (p *simulationProvider) Chat(context.Context, *providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, providers.ErrNotSupported
}
func (p *simulationProvider) Stream(context.Context, *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, providers.ErrNotSupported
}
func (p *simulationProvider) Health(context.Context) error { return nil }

var _ = time.Second

func TestRoutingSimulationFiltersWithoutExecution(t *testing.T) {
	reg := providers.NewRegistry()
	if err := reg.Register(&simulationProvider{id: "openai", name: "OpenAI"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(&simulationProvider{id: "ollama", name: "Ollama"}); err != nil {
		t.Fatal(err)
	}
	pm := providers.NewManager(reg)
	e := NewEngine(NewModelRegistry(), pm, DefaultWeights())
	s, err := e.Simulate(&Request{Model: "gpt-4-turbo", RequiredCapabilities: []string{"vision"}, MinContextSize: 32000})
	if err != nil {
		t.Fatal(err)
	}
	if s.Selected == nil || s.Selected.Provider != "openai" {
		t.Fatalf("expected openai selection, got %#v", s.Selected)
	}
	if s.Filtered < 1 {
		t.Fatalf("expected at least one filtered candidate")
	}
}
