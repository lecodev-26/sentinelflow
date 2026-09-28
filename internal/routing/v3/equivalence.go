package v3

import "sort"

// ModelEquivalence describe una clase lógica de modelos intercambiables para
// una tarea. Los miembros se expresan como provider:model y se validan contra
// el catálogo antes de usarlos en routing.
type ModelEquivalence struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Description          string   `json:"description,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	MinContextSize       int      `json:"min_context_size,omitempty"`
	Members              []string `json:"members"`
}

type EquivalenceMatch struct {
	Alias        string     `json:"alias"`
	Provider     string     `json:"provider"`
	Model        *ModelInfo `json:"model"`
	Capabilities []string   `json:"capabilities"`
	ContextSize  int        `json:"context_size"`
}

// ModelEquivalenceRegistry mantiene las clases lógicas en memoria. La
// configuración es deliberadamente explícita; no inventa equivalencias.
type ModelEquivalenceRegistry struct {
	groups map[string]*ModelEquivalence
}

func NewModelEquivalenceRegistry() *ModelEquivalenceRegistry {
	return &ModelEquivalenceRegistry{groups: make(map[string]*ModelEquivalence)}
}

func (r *ModelEquivalenceRegistry) Register(g *ModelEquivalence) {
	if g == nil || g.ID == "" {
		return
	}
	cp := *g
	cp.Members = append([]string(nil), g.Members...)
	r.groups[g.ID] = &cp
}

func (r *ModelEquivalenceRegistry) Get(id string) (*ModelEquivalence, bool) {
	g, ok := r.groups[id]
	return g, ok
}

func (r *ModelEquivalenceRegistry) List() []*ModelEquivalence {
	out := make([]*ModelEquivalence, 0, len(r.groups))
	for _, g := range r.groups {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Resolve devuelve solamente miembros existentes y compatibles con los
// requisitos. No ejecuta ninguna petición.
func (r *ModelEquivalenceRegistry) Resolve(id string, catalog *ModelRegistry, capabilities []string, minContext int) []EquivalenceMatch {
	g, ok := r.groups[id]
	if !ok || catalog == nil {
		return nil
	}
	required := append([]string(nil), g.RequiredCapabilities...)
	required = append(required, capabilities...)
	if minContext < g.MinContextSize {
		minContext = g.MinContextSize
	}
	out := make([]EquivalenceMatch, 0, len(g.Members))
	for _, member := range g.Members {
		parts := splitModelRef(member)
		if len(parts) != 2 {
			continue
		}
		m, exists := catalog.Get(parts[0], parts[1])
		if !exists || m.Deprecated || m.ContextSize < minContext || !hasAllCapabilities(m, required) {
			continue
		}
		out = append(out, EquivalenceMatch{Alias: id, Provider: parts[0], Model: m, Capabilities: append([]string(nil), m.Capabilities...), ContextSize: m.ContextSize})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Model.InputPer1M+out[i].Model.OutputPer1M == out[j].Model.InputPer1M+out[j].Model.OutputPer1M {
			return out[i].Provider+":"+out[i].Model.ID < out[j].Provider+":"+out[j].Model.ID
		}
		return out[i].Model.InputPer1M+out[i].Model.OutputPer1M < out[j].Model.InputPer1M+out[j].Model.OutputPer1M
	})
	return out
}

func splitModelRef(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
