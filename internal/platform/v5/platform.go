package platformv5

import (
	"context"
	"fmt"
	aicore "github.com/lecodev-26/sentinelflow/internal/ai/v5/core"
	knowledge "github.com/lecodev-26/sentinelflow/internal/knowledge/v5"
	memory "github.com/lecodev-26/sentinelflow/internal/memory/v5"
	prompts "github.com/lecodev-26/sentinelflow/internal/prompts/v5"
	asec "github.com/lecodev-26/sentinelflow/internal/security/v5/ai"
	toolsv5 "github.com/lecodev-26/sentinelflow/internal/tools/v5"
	"strings"
)

type Request struct {
	TenantID, ProjectID, UserID string
	Model, Input                string
	ToolCount                   int
}
type Plan struct {
	Intent           string
	Complexity       float64
	Sensitive        bool
	Security         asec.Action
	MemoryHits       int
	KnowledgeHits    int
	PromptVersion    int
	ToolCallsAllowed bool
	Signals          []string
}
type Platform struct {
	Analyzer  *aicore.Analyzer
	Security  *asec.Detector
	Memory    *memory.Store
	Knowledge *knowledge.Index
	Prompts   *prompts.Registry
	Tools     *toolsv5.Registry
}

func New() *Platform {
	return &Platform{Analyzer: aicore.NewAnalyzer(), Security: asec.NewDetector(), Memory: memory.NewStore(), Knowledge: knowledge.NewIndex(), Prompts: prompts.NewRegistry(), Tools: toolsv5.NewRegistry()}
}
func (p *Platform) Plan(ctx context.Context, r Request) (Plan, error) {
	if r.TenantID == "" {
		return Plan{}, fmt.Errorf("tenant id required")
	}
	a := p.Analyzer.Analyze(r.Input, r.ToolCount)
	findings := p.Security.Scan(r.Input)
	scope := memory.Scope{TenantID: r.TenantID, ProjectID: r.ProjectID, UserID: r.UserID}
	mh := p.Memory.Search(ctx, scope, r.Input, 5)
	kh := p.Knowledge.Search(ctx, r.TenantID, r.ProjectID, r.Input, 5)
	active, _ := p.Prompts.Active("default")
	return Plan{Intent: string(a.Requirements.Intent), Complexity: a.Requirements.Complexity, Sensitive: a.Requirements.Sensitive, Security: asec.Highest(findings), MemoryHits: len(mh), KnowledgeHits: len(kh), PromptVersion: active.Number, ToolCallsAllowed: r.ToolCount == 0 || len(p.Tools.List()) > 0, Signals: append(a.Signals, strings.Join(a.Signals, ","))}, nil
}
