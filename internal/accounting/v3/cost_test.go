package v3

import (
	"math"
	"testing"

	routing "github.com/lecodev-26/sentinelflow/internal/routing/v3"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
)

func newRegWithModel(provider, id string, input, output float64) *routing.ModelRegistry {
	r := routing.NewModelRegistry()
	// Limpiar defaults para test controlado
	// (register no borra, pero el get busca por provider:id)
	r.Register(&routing.ModelInfo{
		ID:          id,
		Name:        id,
		Provider:    provider,
		InputPer1M:  input,
		OutputPer1M: output,
	})
	return r
}

func TestComputeCost_Basic(t *testing.T) {
	reg := newRegWithModel("openai", "gpt-test", 10.0, 30.0)
	rec := &postgres.UsageRecord{
		Provider:     "openai",
		Model:        "gpt-test",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	}
	computeCost(reg, rec)

	// 1M tokens input × $10/1M = $10
	// 1M tokens output × $30/1M = $30
	// total = $40
	if math.Abs(rec.InputCostUSD-10.0) > 0.0001 {
		t.Errorf("InputCostUSD: got %v want 10.0", rec.InputCostUSD)
	}
	if math.Abs(rec.OutputCostUSD-30.0) > 0.0001 {
		t.Errorf("OutputCostUSD: got %v want 30.0", rec.OutputCostUSD)
	}
	if math.Abs(rec.CostUSD-40.0) > 0.0001 {
		t.Errorf("CostUSD: got %v want 40.0", rec.CostUSD)
	}
}

func TestComputeCost_RealisticTokens(t *testing.T) {
	// GPT-4o: $2.5/1M input, $10/1M output
	reg := newRegWithModel("openai", "gpt-4o", 2.5, 10.0)
	rec := &postgres.UsageRecord{
		Provider:     "openai",
		Model:        "gpt-4o",
		InputTokens:  1500,
		OutputTokens: 500,
	}
	computeCost(reg, rec)

	wantInput := (1500.0 / 1_000_000.0) * 2.5  // 0.00375
	wantOutput := (500.0 / 1_000_000.0) * 10.0 // 0.005
	wantTotal := wantInput + wantOutput        // 0.00875

	if math.Abs(rec.InputCostUSD-wantInput) > 1e-9 {
		t.Errorf("InputCostUSD: got %v want %v", rec.InputCostUSD, wantInput)
	}
	if math.Abs(rec.OutputCostUSD-wantOutput) > 1e-9 {
		t.Errorf("OutputCostUSD: got %v want %v", rec.OutputCostUSD, wantOutput)
	}
	if math.Abs(rec.CostUSD-wantTotal) > 1e-9 {
		t.Errorf("CostUSD: got %v want %v", rec.CostUSD, wantTotal)
	}
}

func TestComputeCost_NilRegistry_NoOp(t *testing.T) {
	rec := &postgres.UsageRecord{
		Provider:     "openai",
		Model:        "gpt-test",
		InputTokens:  1000,
		OutputTokens: 500,
	}
	computeCost(nil, rec)
	if rec.CostUSD != 0 {
		t.Errorf("nil registry should not compute cost, got %v", rec.CostUSD)
	}
}

func TestComputeCost_NilRecord_NoPanic(t *testing.T) {
	reg := newRegWithModel("openai", "gpt-test", 10, 30)
	computeCost(reg, nil) // no debe paniquear
}

func TestComputeCost_EmptyProvider_NoOp(t *testing.T) {
	reg := newRegWithModel("openai", "gpt-test", 10, 30)
	rec := &postgres.UsageRecord{
		Provider:    "",
		Model:       "gpt-test",
		InputTokens: 1000,
	}
	computeCost(reg, rec)
	if rec.CostUSD != 0 {
		t.Errorf("empty provider should not compute, got %v", rec.CostUSD)
	}
}

func TestComputeCost_EmptyModel_NoOp(t *testing.T) {
	reg := newRegWithModel("openai", "gpt-test", 10, 30)
	rec := &postgres.UsageRecord{
		Provider:    "openai",
		Model:       "",
		InputTokens: 1000,
	}
	computeCost(reg, rec)
	if rec.CostUSD != 0 {
		t.Errorf("empty model should not compute, got %v", rec.CostUSD)
	}
}

func TestComputeCost_UnknownModel_NoOp(t *testing.T) {
	reg := newRegWithModel("openai", "gpt-test", 10, 30)
	rec := &postgres.UsageRecord{
		Provider:    "openai",
		Model:       "unknown-model",
		InputTokens: 1000,
	}
	computeCost(reg, rec)
	if rec.CostUSD != 0 {
		t.Errorf("unknown model should not compute, got %v", rec.CostUSD)
	}
}

func TestComputeCost_ZeroTokens(t *testing.T) {
	reg := newRegWithModel("openai", "gpt-test", 10, 30)
	rec := &postgres.UsageRecord{
		Provider:     "openai",
		Model:        "gpt-test",
		InputTokens:  0,
		OutputTokens: 0,
	}
	computeCost(reg, rec)
	if rec.CostUSD != 0 {
		t.Errorf("zero tokens should give zero cost, got %v", rec.CostUSD)
	}
}

func TestComputeCost_FreeModel(t *testing.T) {
	reg := newRegWithModel("ollama", "llama3", 0, 0)
	rec := &postgres.UsageRecord{
		Provider:     "ollama",
		Model:        "llama3",
		InputTokens:  10000,
		OutputTokens: 5000,
	}
	computeCost(reg, rec)
	if rec.CostUSD != 0 {
		t.Errorf("free model should give zero cost, got %v", rec.CostUSD)
	}
}

func TestComputeCost_OverwritesPreviousValues(t *testing.T) {
	reg := newRegWithModel("openai", "gpt-test", 10, 30)
	rec := &postgres.UsageRecord{
		Provider:      "openai",
		Model:         "gpt-test",
		InputTokens:   1_000_000,
		OutputTokens:  1_000_000,
		InputCostUSD:  999.0, // valores previos que deben ser sobreescritos
		OutputCostUSD: 999.0,
		CostUSD:       999.0,
	}
	computeCost(reg, rec)
	if math.Abs(rec.InputCostUSD-10.0) > 0.0001 {
		t.Errorf("InputCostUSD not overwritten: got %v", rec.InputCostUSD)
	}
	if math.Abs(rec.CostUSD-40.0) > 0.0001 {
		t.Errorf("CostUSD not overwritten: got %v", rec.CostUSD)
	}
}
