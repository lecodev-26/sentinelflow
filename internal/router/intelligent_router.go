package router

import (
"context"
"fmt"
"time"

"github.com/lecodev-26/sentinelflow/internal/logger"
"github.com/lecodev-26/sentinelflow/internal/provider"
"github.com/lecodev-26/sentinelflow/internal/provider/health"
"github.com/lecodev-26/sentinelflow/internal/provider/model"
"github.com/lecodev-26/sentinelflow/internal/provider/registry"
"github.com/lecodev-26/sentinelflow/internal/resilience"
"github.com/lecodev-26/sentinelflow/internal/router/filters"
"github.com/lecodev-26/sentinelflow/internal/router/scoring"
"github.com/lecodev-26/sentinelflow/internal/router/strategies"
)

// IntelligentRouter es el router de V2.2
// Filtra candidatos → Aplica scoring → Selecciona el mejor
type IntelligentRouter struct {
registry        *registry.Registry
modelRegistry   *model.Registry
healthMonitor   *health.Monitor
metrics         *strategies.MetricsStore
scorer          *scoring.Scorer
filter          *filters.Filter
circuitBreakers map[string]*resilience.CircuitBreaker
cbConfig        CBConfig
}

// NewIntelligentRouter crea el router inteligente
func NewIntelligentRouter(
reg *registry.Registry,
modelReg *model.Registry,
healthMon *health.Monitor,
weights scoring.Weights,
cbCfg CBConfig,
) *IntelligentRouter {
metrics := strategies.NewMetricsStore()

r := &IntelligentRouter{
registry:        reg,
modelRegistry:   modelReg,
healthMonitor:   healthMon,
metrics:         metrics,
scorer:          scoring.NewScorer(weights),
filter:          filters.NewFilter(modelReg),
circuitBreakers: make(map[string]*resilience.CircuitBreaker),
cbConfig:        cbCfg,
}

// Crear circuit breakers
for _, p := range reg.GetAll() {
r.circuitBreakers[p.Name()] = resilience.NewCircuitBreaker(
cbCfg.MaxFailures,
cbCfg.CooldownPeriod,
)
}

return r
}

// Route ejecuta una petición con routing inteligente
func (r *IntelligentRouter) Route(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
// 1. Preparar estado de health y circuit breakers
healthStatus := r.getHealthStatus()
circuitStatus := r.getCircuitStatus()

// 2. Construir requisitos
filterReq := filters.Request{
Model: req.Model,
}

// 3. Filtrar candidatos
providers := r.registry.GetAll()
candidates := r.filter.Apply(providers, healthStatus, circuitStatus, filterReq)

if len(candidates) == 0 {
return nil, fmt.Errorf("no candidates available after filtering")
}

// 4. Enriquecer candidatos con métricas
for i := range candidates {
candidates[i].Metrics = r.metrics.GetOrCreate(candidates[i].ProviderName)
}

// 5. Scoring
best, bestScore, allScores := r.scorer.SelectBest(candidates)

// 6. Log explicativo
logger.Infof("🎯 Routing decision:")
for _, s := range allScores {
marker := "  "
if s.Provider == best.ProviderName {
marker = "→ "
}
logger.Infof("   %s%-12s total=%.3f (lat=%.2f cost=%.2f health=%.2f quality=%.2f)",
marker, s.Provider, s.Total, s.LatencyScore, s.CostScore, s.HealthScore, s.QualityScore)
}
logger.Infof("   Reason: %s", bestScore.Reason)

// 7. Ejecutar con el mejor candidato (con fallback en cadena)
return r.executeWithFallback(ctx, req, candidates, best.ProviderName)
}

// executeWithFallback ejecuta la petición con el mejor candidato,
// y si falla, prueba los siguientes en orden de score.
func (r *IntelligentRouter) executeWithFallback(
ctx context.Context,
req *provider.ChatRequest,
candidates []scoring.Candidate,
selectedName string,
) (*provider.ChatResponse, error) {
// Ordenar candidatos por score (ya tenemos los scores, reutilizamos)
ordered := r.orderCandidatesByScore(candidates)

var lastErr error
for _, c := range ordered {
cb := r.getBreaker(c.ProviderName)

if !cb.Allow() {
logger.Warnf("🚫 Circuit open para %s", c.ProviderName)
continue
}

p, ok := r.registry.Get(c.ProviderName)
if !ok {
continue
}

startTime := time.Now()
logger.Infof("🔄 Intentando: %s (model=%s)", c.ProviderName, req.Model)

resp, err := p.Chat(ctx, req)
latency := time.Since(startTime)

if err == nil {
cb.RecordSuccess()
r.metrics.RecordRequest(c.ProviderName, latency, true, resp.Usage.TotalTokens, 0)
logger.Infof("✅ Éxito con %s (%.2fms)", c.ProviderName, float64(latency.Microseconds())/1000.0)
return resp, nil
}

cb.RecordFailure()
r.metrics.RecordRequest(c.ProviderName, latency, false, 0, 0)
logger.Warnf("❌ Falló %s: %v", c.ProviderName, err)
lastErr = err
}

return nil, fmt.Errorf("all candidates failed: %v", lastErr)
}

// orderCandidatesByScore ordena candidatos por score descendente
func (r *IntelligentRouter) orderCandidatesByScore(candidates []scoring.Candidate) []scoring.Candidate {
type scored struct {
c     scoring.Candidate
score float64
}

scoredList := make([]scored, 0, len(candidates))
for _, c := range candidates {
s := r.scorer.Score(c)
scoredList = append(scoredList, scored{c: c, score: s.Total})
}

// Ordenar por score descendente
for i := 0; i < len(scoredList); i++ {
for j := i + 1; j < len(scoredList); j++ {
if scoredList[j].score > scoredList[i].score {
scoredList[i], scoredList[j] = scoredList[j], scoredList[i]
}
}
}

result := make([]scoring.Candidate, len(scoredList))
for i, s := range scoredList {
result[i] = s.c
}
return result
}

// getHealthStatus devuelve el mapa de estado de salud
func (r *IntelligentRouter) getHealthStatus() map[string]string {
status := make(map[string]string)
for _, h := range r.healthMonitor.GetAll() {
status[h.Name] = string(h.Status)
}
return status
}

// getCircuitStatus devuelve el estado de los circuit breakers
func (r *IntelligentRouter) getCircuitStatus() map[string]string {
status := make(map[string]string)
for name, cb := range r.circuitBreakers {
status[name] = cb.StateString()
}
return status
}

// getBreaker devuelve el circuit breaker de un provider
func (r *IntelligentRouter) getBreaker(name string) *resilience.CircuitBreaker {
if cb, exists := r.circuitBreakers[name]; exists {
return cb
}
cb := resilience.NewCircuitBreaker(r.cbConfig.MaxFailures, r.cbConfig.CooldownPeriod)
r.circuitBreakers[name] = cb
return cb
}

// GetCircuitBreakerStatus devuelve el estado de los CB
func (r *IntelligentRouter) GetCircuitBreakerStatus() map[string]string {
return r.getCircuitStatus()
}

// GetMetrics devuelve las métricas
func (r *IntelligentRouter) GetMetrics() map[string]*strategies.ProviderMetrics {
return r.metrics.GetMetrics()
}

// Strategy devuelve el nombre de la estrategia
func (r *IntelligentRouter) Strategy() string {
return "intelligent-weighted-scoring"
}
