package v3

import (
"time"
)

// Weights define los pesos de cada factor en el scoring
type Weights struct {
Latency float64 `json:"latency"`
Cost    float64 `json:"cost"`
Health  float64 `json:"health"`
Quality float64 `json:"quality"`
}

// DefaultWeights devuelve pesos equilibrados
func DefaultWeights() Weights {
return Weights{
Latency: 0.35,
Cost:    0.25,
Health:  0.25,
Quality: 0.15,
}
}

// Normalize asegura que la suma sea 1.0
func (w *Weights) Normalize() {
sum := w.Latency + w.Cost + w.Health + w.Quality
if sum == 0 {
*w = DefaultWeights()
return
}
w.Latency /= sum
w.Cost /= sum
w.Health /= sum
w.Quality /= sum
}

// Candidate representa un candidato a ser seleccionado
type Candidate struct {
ProviderID   string
ProviderName string
Model        *ModelInfo
HealthStatus string        // "healthy", "degraded", "unhealthy"
AvgLatency   time.Duration
CircuitState string        // "closed", "open", "half-open"
}

// Score contiene el desglose de la puntuación
type Score struct {
ProviderID   string  `json:"provider_id"`
ModelID      string  `json:"model_id"`
Total        float64 `json:"total"`
LatencyScore float64 `json:"latency_score"`
CostScore    float64 `json:"cost_score"`
HealthScore  float64 `json:"health_score"`
QualityScore float64 `json:"quality_score"`
}

// Scorer calcula scores para candidatos
type Scorer struct {
weights Weights
}

// NewScorer crea un scorer con pesos personalizados
func NewScorer(weights Weights) *Scorer {
weights.Normalize()
return &Scorer{weights: weights}
}

// DefaultScorer crea un scorer con pesos por defecto
func DefaultScorer() *Scorer {
return NewScorer(DefaultWeights())
}

// Score calcula el score de un candidato
func (s *Scorer) Score(c Candidate) Score {
result := Score{
ProviderID: c.ProviderID,
}
if c.Model != nil {
result.ModelID = c.Model.ID
}

// 1. Latency score (menor latencia = mayor score)
result.LatencyScore = normalizeLatency(c.AvgLatency)

// 2. Cost score (menor coste = mayor score)
result.CostScore = normalizeCost(c.Model)

// 3. Health score
result.HealthScore = normalizeHealth(c.HealthStatus, c.CircuitState)

// 4. Quality score (basado en capabilities y reputación)
result.QualityScore = normalizeQuality(c.Model)

// Total ponderado
result.Total = result.LatencyScore*s.weights.Latency +
result.CostScore*s.weights.Cost +
result.HealthScore*s.weights.Health +
result.QualityScore*s.weights.Quality

return result
}

// normalizeLatency: 0ms → 1.0, 5000ms → 0.0
func normalizeLatency(latency time.Duration) float64 {
ms := float64(latency.Milliseconds())
if ms <= 0 {
return 0.7 // neutro si no hay datos
}
maxLatency := 5000.0
score := 1.0 - (ms / maxLatency)
if score < 0 {
return 0
}
if score > 1 {
return 1
}
return score
}

// normalizeCost: $0 → 1.0, $100/1M → 0.0
func normalizeCost(m *ModelInfo) float64 {
if m == nil {
return 0.5
}
cost := m.InputPer1M + m.OutputPer1M
if cost <= 0 {
return 1.0
}
maxCost := 100.0
score := 1.0 - (cost / maxCost)
if score < 0 {
return 0
}
if score > 1 {
return 1
}
return score
}

// normalizeHealth combina status + circuit breaker
func normalizeHealth(status, circuit string) float64 {
if circuit == "open" {
return 0.0
}

penalty := 1.0
switch status {
case "healthy":
penalty = 1.0
case "degraded":
penalty = 0.6
case "unhealthy":
penalty = 0.2
case "disabled":
penalty = 0.0
default:
penalty = 0.5
}

if circuit == "half-open" {
penalty *= 0.5
}

return penalty
}

// normalizeQuality: capabilities + context size + deprecation
func normalizeQuality(m *ModelInfo) float64 {
if m == nil {
return 0.5
}

score := 0.4

// Más capabilities = mejor
score += float64(len(m.Capabilities)) * 0.05

// Contexto largo = mejor
if m.ContextSize >= 128000 {
score += 0.2
} else if m.ContextSize >= 32000 {
score += 0.1
}

// Deprecated = penalización
if m.Deprecated {
score -= 0.4
}

if score > 1 {
return 1
}
if score < 0 {
return 0
}
return score
}

// SelectBest elige el candidato con mayor score
func (s *Scorer) SelectBest(candidates []Candidate) (Candidate, Score, []Score) {
if len(candidates) == 0 {
return Candidate{}, Score{}, nil
}

var scores []Score
var best Candidate
var bestScore Score

for i, c := range candidates {
sc := s.Score(c)
scores = append(scores, sc)
if i == 0 || sc.Total > bestScore.Total {
best = c
bestScore = sc
}
}

return best, bestScore, scores
}
