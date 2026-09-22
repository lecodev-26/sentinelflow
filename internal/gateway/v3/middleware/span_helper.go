package middleware

import (
"context"

observabilityv3 "github.com/lecodev-26/sentinelflow/internal/observability/v3"
)

// RecordSpan registra un span en el trace del contexto
func RecordSpan(ctx context.Context, name string, attrs map[string]interface{}) {
trace, ok := ctx.Value(CtxTrace).(*observabilityv3.Trace)
if !ok {
return
}

builder := observabilityv3.StartSpan(name)
for k, v := range attrs {
builder.WithAttr(k, v)
}
span := builder.End()

// Añadir al trace (usando store del contexto)
if store, ok := ctx.Value("store").(*observabilityv3.Store); ok {
store.AddSpan(trace.TraceID, span)
}
}

// GetTrace devuelve el trace del contexto
func GetTrace(ctx context.Context) (*observabilityv3.Trace, bool) {
t, ok := ctx.Value(CtxTrace).(*observabilityv3.Trace)
return t, ok
}
