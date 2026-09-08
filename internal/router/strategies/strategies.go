package strategies

import (
"fmt"
"sort"
"time"

"github.com/lecodev-26/sentinelflow/internal/provider"
)

// RoutingStrategy define cómo elegir un proveedor
type RoutingStrategy interface {
Name() string
Select(ctx *SelectionContext) (string, error)
}

// SelectionContext contiene la información para seleccionar
type SelectionContext struct {
Request       *provider.ChatRequest
Providers     []provider.Provider
Metrics       *MetricsStore
LatencyTarget time.Duration
CostTarget    float64
}

// CostStrategy selecciona el proveedor más barato
type CostStrategy struct{}

func (s *CostStrategy) Name() string { return "cost" }

func (s *CostStrategy) Select(ctx *SelectionContext) (string, error) {
if len(ctx.Providers) == 0 {
return "", fmt.Errorf("no providers available")
}

var bestProvider string
var bestCost float64 = -1

for _, p := range ctx.Providers {
metrics := ctx.Metrics.GetOrCreate(p.Name())
cost := metrics.CostPerToken

if bestCost == -1 || cost < bestCost {
bestCost = cost
bestProvider = p.Name()
}
}

if bestProvider == "" {
return ctx.Providers[0].Name(), nil
}
return bestProvider, nil
}

// LatencyStrategy selecciona el proveedor con menor latencia
type LatencyStrategy struct{}

func (s *LatencyStrategy) Name() string { return "latency" }

func (s *LatencyStrategy) Select(ctx *SelectionContext) (string, error) {
if len(ctx.Providers) == 0 {
return "", fmt.Errorf("no providers available")
}

var bestProvider string
var bestLatency time.Duration = -1

for _, p := range ctx.Providers {
metrics := ctx.Metrics.GetOrCreate(p.Name())
latency := metrics.AvgLatency

if bestLatency == -1 || latency < bestLatency {
bestLatency = latency
bestProvider = p.Name()
}
}

if bestProvider == "" {
return ctx.Providers[0].Name(), nil
}
return bestProvider, nil
}

// HealthStrategy selecciona el proveedor más saludable
type HealthStrategy struct{}

func (s *HealthStrategy) Name() string { return "health" }

func (s *HealthStrategy) Select(ctx *SelectionContext) (string, error) {
if len(ctx.Providers) == 0 {
return "", fmt.Errorf("no providers available")
}

var bestProvider string
var bestHealth float64 = -1

for _, p := range ctx.Providers {
metrics := ctx.Metrics.GetOrCreate(p.Name())
health := metrics.SuccessRate

if bestHealth == -1 || health > bestHealth {
bestHealth = health
bestProvider = p.Name()
}
}

if bestProvider == "" {
return ctx.Providers[0].Name(), nil
}
return bestProvider, nil
}

// WeightedStrategy selecciona basado en pesos
type WeightedStrategy struct {
weights map[string]float64
}

func NewWeightedStrategy(weights map[string]float64) *WeightedStrategy {
return &WeightedStrategy{weights: weights}
}

func (s *WeightedStrategy) Name() string { return "weighted" }

func (s *WeightedStrategy) Select(ctx *SelectionContext) (string, error) {
if len(ctx.Providers) == 0 {
return "", fmt.Errorf("no providers available")
}

// Calcular scores combinados
type scored struct {
name  string
score float64
}

var scores []scored
for _, p := range ctx.Providers {
metrics := ctx.Metrics.GetOrCreate(p.Name())
weight := s.weights[p.Name()]
if weight == 0 {
weight = 1.0
}

// Score = (success_rate * weight) - (latency_penalty)
latencyPenalty := float64(metrics.AvgLatency) / float64(time.Second) * 0.1
score := (metrics.SuccessRate/100)*weight - latencyPenalty

scores = append(scores, scored{name: p.Name(), score: score})
}

// Ordenar por score descendente
sort.Slice(scores, func(i, j int) bool {
return scores[i].score > scores[j].score
})

if len(scores) == 0 {
return ctx.Providers[0].Name(), nil
}
return scores[0].name, nil
}

// CompositeStrategy combina múltiples estrategias
type CompositeStrategy struct {
strategies []RoutingStrategy
}

func NewCompositeStrategy(strategies ...RoutingStrategy) *CompositeStrategy {
return &CompositeStrategy{strategies: strategies}
}

func (s *CompositeStrategy) Name() string { return "composite" }

func (s *CompositeStrategy) Select(ctx *SelectionContext) (string, error) {
if len(ctx.Providers) == 0 {
return "", fmt.Errorf("no providers available")
}

// Votación entre estrategias
votes := make(map[string]int)
for _, strategy := range s.strategies {
selected, err := strategy.Select(ctx)
if err == nil {
votes[selected]++
}
}

// Encontrar el más votado
var best string
var maxVotes int
for provider, count := range votes {
if count > maxVotes {
maxVotes = count
best = provider
}
}

if best == "" {
return ctx.Providers[0].Name(), nil
}
return best, nil
}
