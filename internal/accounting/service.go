package accounting

import (
	"context"
	"sync"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/logger"
)

// Service es el servicio de accounting
type Service struct {
	mu        sync.RWMutex
	pricing   *PricingEngine
	budgets   *BudgetEngine
	events    []*UsageEvent
	maxEvents int
	onUsage   func(*UsageEvent)
}

// NewService crea un nuevo servicio
func NewService(pricing *PricingEngine, budgets *BudgetEngine) *Service {
	s := &Service{
		pricing:   pricing,
		budgets:   budgets,
		events:    make([]*UsageEvent, 0),
		maxEvents: 10000,
	}

	// Suscribir a eventos del bus
	events.Subscribe(events.EventUsageRecorded, s.handleEvent)

	return s
}

// Record registra un evento de uso
func (s *Service) Record(ctx context.Context, event *UsageEvent) {
	// Calcular coste con pricing real
	if event.InputTokens > 0 || event.OutputTokens > 0 {
		total, inputCost, outputCost, err := s.pricing.Calculate(
			event.Provider, event.Model,
			event.InputTokens, event.OutputTokens, event.CachedTokens,
		)
		if err == nil {
			event.CostUSD = total
			event.InputCostUSD = inputCost
			event.OutputCostUSD = outputCost
		}
	}

	// Registrar en budget
	var alert *BudgetAlert
	if event.TenantID != "" && event.CostUSD > 0 {
		alert = s.budgets.Record(event.TenantID, event.CostUSD)
		if alert != nil {
			logger.Warnf("🚨 BUDGET ALERT: tenant=%s threshold=%d%% spent=$%.2f/%.2f forecast=$%.2f",
				alert.TenantID, alert.Threshold, alert.Spent, alert.Limit, alert.Forecast)
		}
	}

	// Guardar en historial
	s.mu.Lock()
	s.events = append(s.events, event)
	if len(s.events) > s.maxEvents {
		s.events = s.events[len(s.events)-s.maxEvents:]
	}
	s.mu.Unlock()

	// Callback
	if s.onUsage != nil {
		s.onUsage(event)
	}

	// Publicar evento al bus
	events.Publish(ctx, events.Event{
		Type:      events.EventCostRecorded,
		RequestID: event.RequestID,
		TenantID:  event.TenantID,
		UserID:    event.UserID,
		Payload: map[string]interface{}{
			"provider": event.Provider,
			"model":    event.Model,
			"tokens":   event.TotalTokens,
			"cost":     event.CostUSD,
		},
	})
}

// handleEvent procesa eventos del bus
func (s *Service) handleEvent(ctx context.Context, e events.Event) {
	// Convertir evento genérico a UsageEvent
	if e.Payload == nil {
		return
	}

	event := &UsageEvent{
		RequestID: e.RequestID,
		TenantID:  e.TenantID,
		UserID:    e.UserID,
		Timestamp: e.Timestamp,
	}

	if provider, ok := e.Payload["provider"].(string); ok {
		event.Provider = provider
	}
	if model, ok := e.Payload["model"].(string); ok {
		event.Model = model
	}
	if tokens, ok := e.Payload["tokens"].(int); ok {
		event.TotalTokens = tokens
	}

	logger.Infof("💰 Usage event: request=%s provider=%s tokens=%d",
		event.RequestID, event.Provider, event.TotalTokens)
}

// OnUsage registra un callback
func (s *Service) OnUsage(cb func(*UsageEvent)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onUsage = cb
}

// GetEvents devuelve los eventos recientes
func (s *Service) GetEvents(limit int) []*UsageEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	result := make([]*UsageEvent, limit)
	copy(result, s.events[len(s.events)-limit:])
	return result
}

// GetTotalCost devuelve el coste total para un tenant en un periodo
func (s *Service) GetTotalCost(tenantID string, since time.Duration) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-since)
	var total float64
	for _, e := range s.events {
		if e.TenantID == tenantID && e.Timestamp.After(cutoff) {
			total += e.CostUSD
		}
	}
	return total
}

// GetCostByProvider agrupa el coste por provider
func (s *Service) GetCostByProvider(tenantID string, since time.Duration) map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-since)
	result := make(map[string]float64)
	for _, e := range s.events {
		if e.TenantID == tenantID && e.Timestamp.After(cutoff) {
			result[e.Provider] += e.CostUSD
		}
	}
	return result
}

// GetCostByModel agrupa el coste por modelo
func (s *Service) GetCostByModel(tenantID string, since time.Duration) map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cutoff := time.Now().Add(-since)
	result := make(map[string]float64)
	for _, e := range s.events {
		if e.TenantID == tenantID && e.Timestamp.After(cutoff) {
			result[e.Model] += e.CostUSD
		}
	}
	return result
}

// Pricing devuelve el engine de pricing
func (s *Service) Pricing() *PricingEngine {
	return s.pricing
}

// Budgets devuelve el engine de budgets
func (s *Service) Budgets() *BudgetEngine {
	return s.budgets
}
