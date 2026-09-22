package v3

import (
"errors"
"sort"
"time"

providers "github.com/lecodev-26/sentinelflow/internal/providers/v3"
)

// Request describe los requisitos de la petición
type Request struct {
RequestID            string
Model                string
RequiredCapabilities []string
MinContextSize       int
MaxCost              float64
PreferredProvider    string
ExcludedProviders    []string
}

// Decision contiene la decisión de routing con explicación
type Decision struct {
Selected      Candidate `json:"selected"`
Score         Score     `json:"score"`
AllCandidates []Score   `json:"all_candidates"`
Reason        string    `json:"reason"`
Filtered      int       `json:"filtered"`
Experiment    string    `json:"experiment,omitempty"`
}

// Engine es el motor de routing inteligente
type Engine struct {
modelRegistry *ModelRegistry
providerMgr   *providers.Manager
scorer        *Scorer
experiments   *ExperimentManager
}

// NewEngine crea un nuevo engine
func NewEngine(modelRegistry *ModelRegistry, providerMgr *providers.Manager, weights Weights) *Engine {
return &Engine{
modelRegistry: modelRegistry,
providerMgr:   providerMgr,
scorer:        NewScorer(weights),
experiments:   NewExperimentManager(),
}
}

// Experiments devuelve el manager de experimentos
func (e *Engine) Experiments() *ExperimentManager {
return e.experiments
}

// Route decide el mejor provider para una petición
func (e *Engine) Route(req *Request) (*Decision, error) {
// 1. Obtener providers disponibles
available := e.providerMgr.AvailableProviders()
if len(available) == 0 {
return nil, errors.New("no providers available")
}

// 2. Aplicar experimento si existe
experimentName := ""
if exp, ok := e.experiments.Get(req.Model); ok && req.RequestID != "" {
preferred := exp.SelectProvider(req.RequestID)
if preferred != "" {
req.PreferredProvider = preferred
experimentName = exp.Name
}
}

// 3. Construir candidatos
var candidates []Candidate
for _, p := range available {
if req.PreferredProvider != "" && p.ID() != req.PreferredProvider {
continue
}
if contains(req.ExcludedProviders, p.ID()) {
continue
}

model, ok := e.modelRegistry.Get(p.ID(), req.Model)
if !ok {
continue
}

if len(req.RequiredCapabilities) > 0 && !hasAllCapabilities(model, req.RequiredCapabilities) {
continue
}

if req.MinContextSize > 0 && model.ContextSize < req.MinContextSize {
continue
}

if req.MaxCost > 0 {
cost := model.InputPer1M + model.OutputPer1M
if cost > req.MaxCost {
continue
}
}

health := "unknown"
var avgLatency time.Duration
if h, ok := e.providerMgr.HealthMonitor().Get(p.ID()); ok {
health = string(h.Status)
avgLatency = h.AvgLatency
}
cb := e.providerMgr.GetBreaker(p.ID())

candidates = append(candidates, Candidate{
ProviderID:   p.ID(),
ProviderName: p.Name(),
Model:        model,
HealthStatus: health,
AvgLatency:   avgLatency,
CircuitState: string(cb.State()),
})
}

// Si el experimento fuerza un provider que no está disponible, ignorar experimento
if len(candidates) == 0 && experimentName != "" {
req.PreferredProvider = ""
experimentName = ""
return e.Route(req)
}

if len(candidates) == 0 {
return nil, errors.New("no candidates after filtering")
}

// 4. Scoring
best, bestScore, allScores := e.scorer.SelectBest(candidates)

// 5. Ordenar por total
sort.Slice(allScores, func(i, j int) bool {
return allScores[i].Total > allScores[j].Total
})

// 6. Razón dominante
reason := computeReason(bestScore)

return &Decision{
Selected:      best,
Score:         bestScore,
AllCandidates: allScores,
Reason:        reason,
Filtered:      len(available) - len(candidates),
Experiment:    experimentName,
}, nil
}

// OrderByScore devuelve los candidatos ordenados por score (para fallback)
func (e *Engine) OrderByScore(req *Request) ([]Candidate, error) {
available := e.providerMgr.AvailableProviders()
if len(available) == 0 {
return nil, errors.New("no providers available")
}

var candidates []Candidate
for _, p := range available {
if req.PreferredProvider != "" && p.ID() != req.PreferredProvider {
continue
}
if contains(req.ExcludedProviders, p.ID()) {
continue
}
model, ok := e.modelRegistry.Get(p.ID(), req.Model)
if !ok {
continue
}
if len(req.RequiredCapabilities) > 0 && !hasAllCapabilities(model, req.RequiredCapabilities) {
continue
}

health := "unknown"
var avgLatency time.Duration
if h, ok := e.providerMgr.HealthMonitor().Get(p.ID()); ok {
health = string(h.Status)
avgLatency = h.AvgLatency
}
cb := e.providerMgr.GetBreaker(p.ID())

candidates = append(candidates, Candidate{
ProviderID:   p.ID(),
ProviderName: p.Name(),
Model:        model,
HealthStatus: health,
AvgLatency:   avgLatency,
CircuitState: string(cb.State()),
})
}

sort.Slice(candidates, func(i, j int) bool {
si := e.scorer.Score(candidates[i])
sj := e.scorer.Score(candidates[j])
return si.Total > sj.Total
})

return candidates, nil
}

// === HELPERS ===

func contains(list []string, s string) bool {
for _, item := range list {
if item == s {
return true
}
}
return false
}

func hasAllCapabilities(m *ModelInfo, caps []string) bool {
for _, c := range caps {
found := false
for _, mc := range m.Capabilities {
if mc == c {
found = true
break
}
}
if !found {
return false
}
}
return true
}

func computeReason(s Score) string {
weighted := map[string]float64{
"latency": s.LatencyScore * 0.35,
"cost":    s.CostScore * 0.25,
"health":  s.HealthScore * 0.25,
"quality": s.QualityScore * 0.15,
}
var best string
var bestScore float64
for k, v := range weighted {
if v > bestScore {
best = k
bestScore = v
}
}
return best
}
