package audit

import (
	"context"
	"time"

	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/logger"
)

// AuditEvent representa un evento de auditoría
type AuditEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	TenantID  string                 `json:"tenant_id"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Actor     string                 `json:"actor,omitempty"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource,omitempty"`
	Result    string                 `json:"result"` // success, failure, denied
	Reason    string                 `json:"reason,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AuditWriter escribe eventos de auditoría
type AuditWriter interface {
	Write(ctx context.Context, event AuditEvent) error
}

// LoggerWriter escribe a logs (fallback)
type LoggerWriter struct{}

func (w *LoggerWriter) Write(ctx context.Context, event AuditEvent) error {
	logger.WithFields(map[string]interface{}{
		"audit_id":   event.ID,
		"tenant_id":  event.TenantID,
		"user_id":    event.UserID,
		"request_id": event.RequestID,
		"action":     event.Action,
		"resource":   event.Resource,
		"result":     event.Result,
		"reason":     event.Reason,
	}).Info("audit")
	return nil
}

// SubscribeAll suscribe el audit writer al bus de eventos
func SubscribeAll(writer AuditWriter) {
	events.SubscribeAll(func(ctx context.Context, e events.Event) {
		auditEvent := convertToAudit(e)
		_ = writer.Write(ctx, auditEvent)
	})
}

// convertToAudit convierte un evento interno a evento de auditoría
func convertToAudit(e events.Event) AuditEvent {
	result := "success"
	if e.Type == events.EventAuthenticationFailed ||
		e.Type == events.EventAuthorizationFailed ||
		e.Type == events.EventPolicyDenied ||
		e.Type == events.EventSecurityBlocked ||
		e.Type == events.EventRateLimited ||
		e.Type == events.EventQuotaExceeded {
		result = "denied"
	}

	action := string(e.Type)
	if idx := len(action) - 1; idx >= 0 && action[idx] == 'd' {
		action = action[:idx] + "ed"
	}

	return AuditEvent{
		ID:        e.RequestID + "-" + time.Now().Format("150405.000"),
		Timestamp: e.Timestamp,
		Type:      string(e.Type),
		TenantID:  e.TenantID,
		UserID:    e.UserID,
		RequestID: e.RequestID,
		Action:    action,
		Result:    result,
		Metadata:  e.Payload,
	}
}
