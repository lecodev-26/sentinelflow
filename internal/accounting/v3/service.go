package v3

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	routing "github.com/lecodev-26/sentinelflow/internal/routing/v3"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

// Service es el servicio de accounting
type Service struct {
	client        *postgres.Client
	usageRepo     *postgres.UsageRepo
	budgetRepo    *postgres.BudgetRepo
	modelReg      *routing.ModelRegistry
	outbox        *events.Outbox
	onBudgetAlert func(alert BudgetAlert)
}

// BudgetAlert representa una alerta de presupuesto
type BudgetAlert struct {
	TenantID  string    `json:"tenant_id"`
	Threshold int       `json:"threshold"`
	Spent     float64   `json:"spent_usd"`
	Limit     float64   `json:"limit_usd"`
	Percent   float64   `json:"percent"`
	Timestamp time.Time `json:"timestamp"`
}

// NewService crea un nuevo servicio.
// outbox puede ser nil — en ese caso no se persisten eventos (modo legacy).
func NewService(client *postgres.Client, usageRepo *postgres.UsageRepo, budgetRepo *postgres.BudgetRepo, modelReg *routing.ModelRegistry, outbox *events.Outbox) *Service {
	return &Service{
		client:     client,
		usageRepo:  usageRepo,
		budgetRepo: budgetRepo,
		modelReg:   modelReg,
		outbox:     outbox,
	}
}

// OnBudgetAlert registra un callback de alerta
func (s *Service) OnBudgetAlert(cb func(BudgetAlert)) {
	s.onBudgetAlert = cb
}

// Record calcula coste, persiste el uso y publica el evento de uso en el outbox.
//
// Garantía atómica:
//   - INSERT usage_records
//   - INSERT outbox_events
//
// ocurren en la MISMA transacción. Si algo falla, rollback: no hay uso huérfano
// ni evento sin uso.
//
// El budget se actualiza fuera de TX porque es recalculable desde usage_records.
func (s *Service) Record(ctx context.Context, rec *postgres.UsageRecord) error {
	// Calcular coste con pricing real del Model Registry
	if s.modelReg != nil && rec.Provider != "" && rec.Model != "" {
		if model, ok := s.modelReg.Get(rec.Provider, rec.Model); ok {
			rec.InputCostUSD = (float64(rec.InputTokens) / 1_000_000.0) * model.InputPer1M
			rec.OutputCostUSD = (float64(rec.OutputTokens) / 1_000_000.0) * model.OutputPer1M
			rec.CostUSD = rec.InputCostUSD + rec.OutputCostUSD
		}
	}

	// Persistir uso + evento en la misma TX (si hay outbox)
	if s.outbox != nil && s.client != nil {
		if err := s.recordWithOutbox(ctx, rec); err != nil {
			logger.Errorf("❌ Error guardando usage + event: %v", err)
			return err
		}
	} else {
		// Modo legacy: sin outbox
		if err := s.usageRepo.Create(ctx, rec); err != nil {
			logger.Errorf("❌ Error guardando usage record: %v", err)
			return err
		}
	}

	logger.Infof("💰 Usage: tenant=%s provider=%s model=%s tokens=%d cost=$%.6f",
		rec.TenantID, rec.Provider, rec.Model, rec.TotalTokens, rec.CostUSD)

	// Actualizar budget (fuera de TX — eventual consistency, recalculable)
	if rec.CostUSD > 0 {
		budget, err := s.budgetRepo.RecordSpend(ctx, rec.TenantID, rec.CostUSD)
		if err != nil {
			logger.Errorf("❌ Error actualizando budget: %v", err)
		} else {
			s.checkAlerts(ctx, budget)
		}
	}

	return nil
}

// recordWithOutbox hace INSERT usage + INSERT outbox en una sola TX.
func (s *Service) recordWithOutbox(ctx context.Context, rec *postgres.UsageRecord) error {
	return s.client.WithTx(ctx, func(tx pgx.Tx) error {
		if err := s.usageRepo.CreateTx(ctx, tx, rec); err != nil {
			return err
		}
		ev := events.NewEvent(events.EventUsageRecorded).
			WithTenant(rec.TenantID).
			WithProject(rec.ProjectID).
			WithUser(rec.UserID).
			WithRequest(rec.RequestID).
			WithPayload("usage_id", rec.ID).
			WithPayload("provider", rec.Provider).
			WithPayload("model", rec.Model).
			WithPayload("input_tokens", rec.InputTokens).
			WithPayload("output_tokens", rec.OutputTokens).
			WithPayload("cost_usd", rec.CostUSD).
			WithPayload("latency_ms", rec.LatencyMs).
			WithPayload("status", rec.Status)
		return s.outbox.EnqueueTx(ctx, tx, ev)
	})
}

// checkAlerts verifica si hay que disparar alertas
func (s *Service) checkAlerts(ctx context.Context, b *postgres.Budget) {
	if b == nil || b.MonthlyLimit <= 0 {
		return
	}

	pct := b.PercentMonth()
	var threshold int

	switch {
	case pct >= 100 && !b.Alert100Sent:
		threshold = 100
		b.Alert100Sent = true
	case pct >= 90 && !b.Alert90Sent:
		threshold = 90
		b.Alert90Sent = true
	case pct >= 80 && !b.Alert80Sent:
		threshold = 80
		b.Alert80Sent = true
	default:
		return
	}

	if err := s.budgetRepo.MarkAlert(ctx, b.TenantID, threshold); err != nil {
		logger.Errorf("❌ Error marcando alerta: %v", err)
	}

	if s.onBudgetAlert != nil {
		s.onBudgetAlert(BudgetAlert{
			TenantID:  b.TenantID,
			Threshold: threshold,
			Spent:     b.SpentMonth,
			Limit:     b.MonthlyLimit,
			Percent:   pct,
			Timestamp: time.Now(),
		})
	}
}

// Stats devuelve estadísticas agregadas de uso para un tenant en una ventana.
type Stats struct {
	TenantID   string             `json:"tenant_id"`
	Since      time.Time          `json:"since"`
	TotalCost  float64            `json:"total_cost_usd"`
	Count      int64              `json:"requests"`
	ByProvider map[string]float64 `json:"by_provider"`
	ByModel    map[string]float64 `json:"by_model"`
	ByDay      map[string]float64 `json:"by_day"`
}

func (s *Service) Stats(ctx context.Context, tenantID string, since time.Duration) (*Stats, error) {
	from := time.Now().Add(-since)
	cost, err := s.usageRepo.SumCostByTenant(ctx, tenantID, from)
	if err != nil {
		return nil, err
	}
	count, err := s.usageRepo.CountByTenant(ctx, tenantID, from)
	if err != nil {
		return nil, err
	}
	byProvider, err := s.usageRepo.SumCostByProvider(ctx, tenantID, from)
	if err != nil {
		return nil, err
	}
	byModel, err := s.usageRepo.SumCostByModel(ctx, tenantID, from)
	if err != nil {
		return nil, err
	}
	byDay, err := s.usageRepo.SumCostByDay(ctx, tenantID, from)
	if err != nil {
		return nil, err
	}
	return &Stats{
		TenantID:   tenantID,
		Since:      from,
		TotalCost:  cost,
		Count:      count,
		ByProvider: byProvider,
		ByModel:    byModel,
		ByDay:      byDay,
	}, nil
}
