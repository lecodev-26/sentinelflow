package scoring

import (
	"time"

	"github.com/lecodev-26/sentinelflow/internal/provider/model"
	"github.com/lecodev-26/sentinelflow/internal/router/strategies"
)

// Weights define los pesos de cada factor
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

// Score contiene el desglose de la puntuación
type Score struct {
	Provider     string  `json:"provider"`
	ModelID      string  `json:"model_id,omitempty"`
	Total        float64 `json:"total"`
	LatencyScore float64 `json:"latency_score"`
	CostScore    float64 `json:"cost_score"`
	HealthScore  float64 `json:"health_score"`
	QualityScore float64 `json:"quality_score"`
	Reason       string  `json:"reason,omitempty"`
}

// Candidate es un candidato a ser seleccionado
type Candidate struct {
	ProviderName string
	Model        *model.Model
	Metrics      *strategies.ProviderMetrics
	HealthStatus string
	CircuitState string
}

// Scorer calcula scores para candidatos
type Scorer struct {
	weights Weights
}

// NewScorer crea un nuevo scorer
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
		Provider: c.ProviderName,
	}

	if c.Model != nil {
		result.ModelID = c.Model.ID
	}

	// 1. Latency score (menor latencia = mayor score)
	result.LatencyScore = normalizeLatency(c.Metrics)

	// 2. Cost score (menor coste = mayor score)
	result.CostScore = normalizeCost(c.Model)

	// 3. Health score (0-1)
	result.HealthScore = normalizeHealth(c.Metrics, c.HealthStatus, c.CircuitState)

	// 4. Quality score (basado en capabilities y reputación del modelo)
	result.QualityScore = normalizeQuality(c.Model)

	// Total ponderado
	result.Total = result.LatencyScore*s.weights.Latency +
		result.CostScore*s.weights.Cost +
		result.HealthScore*s.weights.Health +
		result.QualityScore*s.weights.Quality

	return result
}

// normalizeLatency convierte latencia a score 0-1
// Latencia de 0ms → 1.0, 5000ms → 0.0
func normalizeLatency(m *strategies.ProviderMetrics) float64 {
	if m == nil {
		return 0.5
	}
	latencyMs := float64(m.AvgLatency.Milliseconds())
	if latencyMs <= 0 {
		return 1.0
	}
	maxLatency := 5000.0
	score := 1.0 - (latencyMs / maxLatency)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// normalizeCost convierte precio a score 0-1
// Precio $0 → 1.0, $50/1M → 0.0
func normalizeCost(m *model.Model) float64 {
	if m == nil {
		return 0.5
	}
	costPer1M := m.Pricing.InputPer1M + m.Pricing.OutputPer1M
	if costPer1M <= 0 {
		return 1.0
	}
	maxCost := 100.0
	score := 1.0 - (costPer1M / maxCost)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// normalizeHealth combina success rate y circuit state
func normalizeHealth(m *strategies.ProviderMetrics, healthStatus, circuitState string) float64 {
	// Circuit breaker abierto = 0
	if circuitState == "open" {
		return 0
	}

	// Success rate del provider (0-100)
	successRate := 0.5
	if m != nil && m.SuccessRate > 0 {
		successRate = m.SuccessRate / 100.0
	}

	// Penalización por estado de salud
	penalty := 1.0
	switch healthStatus {
	case "healthy":
		penalty = 1.0
	case "degraded":
		penalty = 0.7
	case "unhealthy":
		penalty = 0.3
	case "disabled":
		penalty = 0.0
	case "unknown":
		penalty = 0.5
	}

	return successRate * penalty
}

// normalizeQuality puntúa la calidad del modelo
func normalizeQuality(m *model.Model) float64 {
	if m == nil {
		return 0.5
	}

	score := 0.5

	// Más capacidades = mejor
	score += float64(len(m.Capabilities)) * 0.05

	// Modelos con contexto largo son mejores
	if m.Limits.MaxContextTokens >= 128000 {
		score += 0.2
	} else if m.Limits.MaxContextTokens >= 32000 {
		score += 0.1
	}

	// Modelos deprecated bajan
	if m.Deprecated {
		score -= 0.3
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

	// Generar razón
	bestScore.Reason = formatReason(bestScore)

	return best, bestScore, scores
}

func formatReason(s Score) string {
	// Identificar el factor dominante
	latencyWeighted := s.LatencyScore * 0.35
	costWeighted := s.CostScore * 0.25
	healthWeighted := s.HealthScore * 0.25
	qualityWeighted := s.QualityScore * 0.15

	dominant := "balanced"
	maxWeighted := latencyWeighted
	if costWeighted > maxWeighted {
		dominant = "cost"
		maxWeighted = costWeighted
	}
	if healthWeighted > maxWeighted {
		dominant = "health"
		maxWeighted = healthWeighted
	}
	if qualityWeighted > maxWeighted {
		dominant = "quality"
	}

	return dominant
}

// Strategy devuelve el nombre de la estrategia
func (s *Scorer) Strategy() string {
	return "weighted-scoring"
}

// LatencyTarget se usa para filtrar candidatos lentos
func LatencyTarget(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
