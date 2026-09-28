package governancev5

import (
	"fmt"
	"sync"
	"time"
)

type AssetType string

const (
	Model   AssetType = "model"
	Agent   AssetType = "agent"
	Tool    AssetType = "tool"
	Prompt  AssetType = "prompt"
	Dataset AssetType = "dataset"
)

type Asset struct {
	ID       string
	Type     AssetType
	Owner    string
	Risk     int
	Region   string
	ReviewAt time.Time
}
type Registry struct {
	mu     sync.RWMutex
	assets map[string]Asset
}

func NewRegistry() *Registry { return &Registry{assets: map[string]Asset{}} }
func (r *Registry) Register(a Asset) error {
	if a.ID == "" || a.Owner == "" {
		return fmt.Errorf("asset id and owner required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets[a.ID] = a
	return nil
}
func (r *Registry) Get(id string) (Asset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.assets[id]
	return a, ok
}

type Rule struct {
	Models, Providers, Regions []string
	MaxRisk                    int
	RequireApproval            bool
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
func (r Rule) Allows(model, provider, region string, risk int) bool {
	if len(r.Models) > 0 && !contains(r.Models, model) {
		return false
	}
	if len(r.Providers) > 0 && !contains(r.Providers, provider) {
		return false
	}
	if len(r.Regions) > 0 && !contains(r.Regions, region) {
		return false
	}
	return r.MaxRisk == 0 || risk <= r.MaxRisk
}
