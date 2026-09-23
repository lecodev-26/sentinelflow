package executor

import (
"context"

"fmt"
"time"

"github.com/lecodev-26/sentinelflow/internal/logger"
providers "github.com/lecodev-26/sentinelflow/internal/providers/v3"
routingv3 "github.com/lecodev-26/sentinelflow/internal/routing/v3"
)

// AttemptResult guarda el resultado de un intento
type AttemptResult struct {
ProviderID   string        `json:"provider_id"`
ProviderName string        `json:"provider_name"`
Latency      time.Duration `json:"latency"`
Error        string        `json:"error,omitempty"`
StatusCode   int           `json:"status_code,omitempty"`
Success      bool          `json:"success"`
}

// ExecuteError incluye detalle de todos los intentos
type ExecuteError struct {
Attempts []AttemptResult
LastErr  error
}

func (e *ExecuteError) Error() string {
if len(e.Attempts) == 0 {
return "no attempts made"
}
msg := fmt.Sprintf("all %d provider(s) failed", len(e.Attempts))
for i, a := range e.Attempts {
msg += fmt.Sprintf(" | attempt %d: %s (%v)", i+1, a.ProviderID, a.Error)
}
return msg
}

// Executor ejecuta peticiones con failover real
type Executor struct {
registry *providers.Registry
manager  *providers.Manager
}

// NewExecutor crea un nuevo executor
func NewExecutor(reg *providers.Registry, mgr *providers.Manager) *Executor {
return &Executor{
registry: reg,
manager:  mgr,
}
}

// ExecuteChat intenta ejecutar Chat en cada candidato en orden de score
func (e *Executor) ExecuteChat(
ctx context.Context,
candidates []routingv3.Candidate,
req *providers.ChatRequest,
) (*providers.ChatResponse, []AttemptResult, error) {
if len(candidates) == 0 {
return nil, nil, fmt.Errorf("no candidates provided")
}

attempts := make([]AttemptResult, 0, len(candidates))

for i, c := range candidates {
select {
case <-ctx.Done():
attempts = append(attempts, AttemptResult{
ProviderID: c.ProviderID,
Error:      "context cancelled",
})
return nil, attempts, ctx.Err()
default:
}

provider, ok := e.registry.Get(c.ProviderID)
if !ok {
logger.Warnf("⚠️ Attempt %d: provider %s not found in registry", i+1, c.ProviderID)
attempts = append(attempts, AttemptResult{
ProviderID: c.ProviderID,
Error:      "provider not found",
})
continue
}

cb := e.manager.GetBreaker(provider.ID())
if !cb.Allow() {
logger.Warnf("🚫 Attempt %d: circuit breaker OPEN for %s", i+1, provider.ID())
attempts = append(attempts, AttemptResult{
ProviderID:   provider.ID(),
ProviderName: provider.Name(),
Error:        "circuit breaker open",
})
continue
}

logger.Infof("🔄 Attempt %d/%d: trying %s", i+1, len(candidates), provider.ID())
start := time.Now()
resp, err := provider.Chat(ctx, req)
latency := time.Since(start)

if err == nil {
cb.RecordSuccess()
attempts = append(attempts, AttemptResult{
ProviderID:   provider.ID(),
ProviderName: provider.Name(),
Latency:      latency,
Success:      true,
})
logger.Infof("✅ Success with %s (%.0fms, attempt %d/%d)",
provider.ID(), float64(latency.Microseconds())/1000.0, i+1, len(candidates))
return resp, attempts, nil
}

cb.RecordFailure()
attempts = append(attempts, AttemptResult{
ProviderID:   provider.ID(),
ProviderName: provider.Name(),
Latency:      latency,
Error:        err.Error(),
})
logger.Warnf("❌ Attempt %d failed with %s: %v", i+1, provider.ID(), err)
}

return nil, attempts, &ExecuteError{
Attempts: attempts,
LastErr:  fmt.Errorf("all %d providers failed", len(attempts)),
}
}

// ExecuteStream intenta ejecutar Stream en cada candidato en orden de score.
// IMPORTANTE: una vez que un provider empieza a emitir chunks, NO se puede
// hacer failover a otro provider porque el cliente ya recibió parte de la
// respuesta. Por eso solo hacemos fallback ANTES del primer chunk.
func (e *Executor) ExecuteStream(
ctx context.Context,
candidates []routingv3.Candidate,
req *providers.ChatRequest,
) (<-chan providers.StreamChunk, []AttemptResult, error) {
if len(candidates) == 0 {
return nil, nil, fmt.Errorf("no candidates provided")
}

attempts := make([]AttemptResult, 0, len(candidates))

for i, c := range candidates {
select {
case <-ctx.Done():
return nil, attempts, ctx.Err()
default:
}

provider, ok := e.registry.Get(c.ProviderID)
if !ok {
attempts = append(attempts, AttemptResult{
ProviderID: c.ProviderID,
Error:      "provider not found",
})
continue
}

cb := e.manager.GetBreaker(provider.ID())
if !cb.Allow() {
attempts = append(attempts, AttemptResult{
ProviderID:   provider.ID(),
ProviderName: provider.Name(),
Error:        "circuit breaker open",
})
continue
}

logger.Infof("🔄 Stream attempt %d/%d: trying %s", i+1, len(candidates), provider.ID())
start := time.Now()

chunks, err := provider.Stream(ctx, req)
latency := time.Since(start)

if err != nil {
cb.RecordFailure()
attempts = append(attempts, AttemptResult{
ProviderID:   provider.ID(),
ProviderName: provider.Name(),
Latency:      latency,
Error:        err.Error(),
})
logger.Warnf("❌ Stream attempt %d failed with %s: %v", i+1, provider.ID(), err)
continue
}

cb.RecordSuccess()
attempts = append(attempts, AttemptResult{
ProviderID:   provider.ID(),
ProviderName: provider.Name(),
Latency:      latency,
Success:      true,
})
logger.Infof("✅ Stream started with %s (%.0fms, attempt %d/%d)",
provider.ID(), float64(latency.Microseconds())/1000.0, i+1, len(candidates))
return chunks, attempts, nil
}

return nil, attempts, &ExecuteError{
Attempts: attempts,
LastErr:  fmt.Errorf("all %d stream providers failed", len(attempts)),
}
}
