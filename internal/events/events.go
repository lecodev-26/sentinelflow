package events

import (
"context"
"sync"
"time"
)

// Type representa el tipo de evento
type Type string

const (
// Request lifecycle
EventRequestReceived         Type = "request.received"
EventAuthenticationSucceeded Type = "auth.succeeded"
EventAuthenticationFailed    Type = "auth.failed"
EventAuthorizationSucceeded  Type = "authz.succeeded"
EventAuthorizationFailed     Type = "authz.failed"

// Security
EventSecurityPassed Type = "security.passed"
EventSecurityBlocked Type = "security.blocked"
EventPIIDetected    Type = "security.pii_detected"
EventSecretDetected Type = "security.secret_detected"
EventInjectionDetected Type = "security.injection_detected"

// Policy
EventPolicyEvaluated Type = "policy.evaluated"
EventPolicyDenied    Type = "policy.denied"

// Rate limit / Quota
EventRateLimited  Type = "rate_limit.exceeded"
EventQuotaExceeded Type = "quota.exceeded"

// Cache
EventCacheHit   Type = "cache.hit"
EventCacheMiss  Type = "cache.miss"
EventCacheSet   Type = "cache.set"

// Routing
EventRouteSelected Type = "route.selected"
EventRouteFailed   Type = "route.failed"

// Provider
EventProviderStarted  Type = "provider.started"
EventProviderFailed   Type = "provider.failed"
EventProviderFallback Type = "provider.fallback"
EventProviderCompleted Type = "provider.completed"

// Streaming
EventStreamStarted   Type = "stream.started"
EventStreamFirstToken Type = "stream.first_token"
EventStreamCompleted Type = "stream.completed"
EventStreamError     Type = "stream.error"

// Accounting
EventUsageRecorded Type = "usage.recorded"
EventCostRecorded  Type = "cost.recorded"

// Audit
EventAuditRecorded Type = "audit.recorded"
)

// Event es un evento interno del sistema
type Event struct {
Type      Type                   `json:"type"`
Timestamp time.Time              `json:"timestamp"`
RequestID string                 `json:"request_id,omitempty"`
TraceID   string                 `json:"trace_id,omitempty"`
TenantID  string                 `json:"tenant_id,omitempty"`
UserID    string                 `json:"user_id,omitempty"`
Payload   map[string]interface{} `json:"payload,omitempty"`
}

// Handler procesa eventos
type Handler func(ctx context.Context, event Event)

// Bus es el bus de eventos in-process
type Bus struct {
mu       sync.RWMutex
handlers map[Type][]Handler
all      []Handler
}

// NewBus crea un nuevo bus de eventos
func NewBus() *Bus {
return &Bus{
handlers: make(map[Type][]Handler),
all:      []Handler{},
}
}

// Subscribe registra un handler para un tipo de evento
func (b *Bus) Subscribe(eventType Type, handler Handler) {
b.mu.Lock()
defer b.mu.Unlock()
b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// SubscribeAll registra un handler para todos los eventos
func (b *Bus) SubscribeAll(handler Handler) {
b.mu.Lock()
defer b.mu.Unlock()
b.all = append(b.all, handler)
}

// Publish publica un evento al bus
func (b *Bus) Publish(ctx context.Context, event Event) {
if event.Timestamp.IsZero() {
event.Timestamp = time.Now()
}

b.mu.RLock()
handlers := make([]Handler, 0)
handlers = append(handlers, b.all...)
handlers = append(handlers, b.handlers[event.Type]...)
b.mu.RUnlock()

for _, h := range handlers {
go h(ctx, event)
}
}

// Global bus
var globalBus = NewBus()

// Global devuelve el bus global
func Global() *Bus {
return globalBus
}

// Publish publica en el bus global
func Publish(ctx context.Context, event Event) {
globalBus.Publish(ctx, event)
}

// Subscribe suscribe al bus global
func Subscribe(eventType Type, handler Handler) {
globalBus.Subscribe(eventType, handler)
}

// SubscribeAll suscribe al bus global
func SubscribeAll(handler Handler) {
globalBus.SubscribeAll(handler)
}
