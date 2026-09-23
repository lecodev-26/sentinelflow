// Package events provee un bus de eventos desacoplado con múltiples
// backends (InMemory para desarrollo, Redis para producción).
//
// Uso típico:
//
// bus := events.NewInMemoryBus()
// bus.Subscribe("usage.recorded", myHandler)
// bus.Publish(ctx, events.Event{Type: "usage.recorded", ...})
package events

import (
	"context"
	"encoding/json"
	"time"
)

// Event es el formato canónico de un evento interno.
// Todos los consumidores reciben este formato, independientemente del bus.
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	ProjectID string                 `json:"project_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// NewEvent crea un evento con ID único y timestamp actual
func NewEvent(eventType string) Event {
	return Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Payload:   make(map[string]interface{}),
	}
}

// WithTenant establece el tenant
func (e Event) WithTenant(tenantID string) Event {
	e.TenantID = tenantID
	return e
}

// WithUser establece el usuario
func (e Event) WithUser(userID string) Event {
	e.UserID = userID
	return e
}

// WithRequest establece el request ID
func (e Event) WithRequest(requestID string) Event {
	e.RequestID = requestID
	return e
}

// WithPayload añade un campo al payload
func (e Event) WithPayload(key string, value interface{}) Event {
	if e.Payload == nil {
		e.Payload = make(map[string]interface{})
	}
	e.Payload[key] = value
	return e
}

// Marshal serializa el evento a JSON
func (e Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEvent deserializa un evento desde JSON
func UnmarshalEvent(data []byte) (*Event, error) {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// Handler procesa un evento. Debe ser idempotente.
type Handler func(ctx context.Context, event Event)

// Consumer agrupa un handler con un nombre legible para logs.
type Consumer struct {
	Name    string
	Handler Handler
}

// NewConsumer crea un nuevo consumidor
func NewConsumer(name string, handler Handler) Consumer {
	return Consumer{Name: name, Handler: handler}
}

// EventBus es la interfaz común a todos los backends.
type EventBus interface {
	// Publish emite un evento al bus.
	// Debe retornar cuando el evento se ha encolado (no cuando se ha procesado).
	Publish(ctx context.Context, event Event) error

	// Subscribe registra un consumidor para un tipo de evento.
	// El handler se invoca en su propia goroutine.
	Subscribe(eventType string, consumer Consumer) error

	// Unsubscribe elimina todos los handlers de un tipo de evento.
	Unsubscribe(eventType string)

	// Start arranca el bus (para backends que necesitan conexión).
	Start(ctx context.Context) error

	// Stop cierra el bus limpiamente.
	Stop() error
}

// === Event types (constantes para evitar typos) ===

const (
	// Request lifecycle
	EventRequestReceived  = "request.received"
	EventRequestCompleted = "request.completed"
	EventRequestFailed    = "request.failed"

	// Provider
	EventProviderCalled    = "provider.called"
	EventProviderFailed    = "provider.failed"
	EventProviderFallback  = "provider.fallback"
	EventProviderCompleted = "provider.completed"

	// Accounting / FinOps
	EventUsageRecorded  = "usage.recorded"
	EventCostRecorded   = "cost.recorded"
	EventBudgetWarning  = "budget.warning"
	EventBudgetExceeded = "budget.exceeded"

	// Security
	EventSecurityBlocked   = "security.blocked"
	EventPIIDetected       = "security.pii_detected"
	EventSecretDetected    = "security.secret_detected"
	EventInjectionDetected = "security.injection_detected"

	// Audit
	EventAPIKeyCreated = "audit.api_key.created"
	EventAPIKeyRevoked = "audit.api_key.revoked"
	EventPolicyChanged = "audit.policy.changed"
	EventUserCreated   = "audit.user.created"
	EventUserDeleted   = "audit.user.deleted"

	// Health
	EventProviderHealthChanged = "provider.health_changed"
)

// WithProject establece el proyecto (multi-tenancy V4)
func (e Event) WithProject(projectID string) Event {
	e.ProjectID = projectID
	return e
}

// WithTrace establece el trace ID (correlación OTel V4.5)
func (e Event) WithTrace(traceID string) Event {
	e.TraceID = traceID
	return e
}
