# ADR-0002: Event Bus Architecture

**Status:** Accepted  
**Date:** 2026-09-18

## Context

V2 did accounting/audit/analytics inline in the request path.

Problems:
- Slows down requests
- Couples Data Plane to business logic
- Hard to add new consumers

## Decision

Introduce an **Event Bus** with asynchronous consumers.

Phase 1: In-process bus with goroutines  
Phase 2: NATS/Kafka for multi-node

## Event Schema

```go
type Event struct {
    ID        string                 `json:"id"`
    Type      EventType              `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    TenantID  string                 `json:"tenant_id"`
    ProjectID string                 `json:"project_id,omitempty"`
    RequestID string                 `json:"request_id,omitempty"`
    Payload   map[string]interface{} `json:"payload"`
}
```

Consequences

Positive

· Data Plane stays fast
· New consumers without changing Data Plane
· Natural audit trail

Negative

· Eventual consistency
· Consumers must be idempotent
· Debugging harder (async)

Mitigation

· Idempotency keys per event
· Dead-letter queue for failures
· Correlation IDs across planes
  EOF
