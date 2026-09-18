# SentinelFlow V3 - Planes

## 1. Data Plane

**What:** Per-request processing.  
**Where:** `internal/gateway/`  
**Binary:** `cmd/gateway/`

### Pipeline

```

Request
↓
Request ID (UUID)
↓
Authentication (API key)
↓
Tenant resolution (org + project + principal)
↓
Policy evaluation (compiled policy)
↓
Rate limit (tenant-aware)
↓
Deduplication (tenant + hash)
↓
Cache lookup (L1 → L2 → L3)
↓
Router (constraints + scoring)
↓
Provider execution (retry + CB + fallback)
↓
Response validation
↓
Accounting (publish event)
↓
Telemetry (metrics + traces)
↓
Client

```

### Requirements

- **Stateless**: no local state
- **Fast**: P95 overhead < 50ms
- **Isolated**: every step is tenant-aware
- **Streaming**: SSE end-to-end

---

## 2. Control Plane

**What:** Configuration + Administration.  
**Where:** `internal/controlplane/`  
**Binary:** `cmd/controlplane/`

### Responsibilities

| Domain | Entities |
|--------|----------|
| Identity | Organizations, Projects, Users |
| Access | API Keys, RBAC, Sessions |
| Providers | Providers, Models, Credentials |
| Routing | Routing Rules, Experiments |
| Policy | Policies (versioned) |
| Budgets | Organization, Project, Model |
| Config | Static, Dynamic, Secrets |
| Audit | All admin events |

### API Surface

```

/v1/admin/organizations
/v1/admin/projects
/v1/admin/users
/v1/admin/api-keys
/v1/admin/providers
/v1/admin/models
/v1/admin/policies
/v1/admin/budgets
/v1/admin/routing
/v1/admin/audit
/v1/admin/config

```

### Requirements

- **Persistent**: PostgreSQL source of truth
- **Authoritative**: publishes config changes to Data Plane
- **Versioned**: config snapshots + rollback
- **Audited**: every change logged

---

## 3. Event Plane

**What:** Asynchronous event processing.  
**Where:** `internal/events/`  
**Binary:** `cmd/worker/`

### Event Types

```go
const (
    // Request lifecycle
    RequestStarted
    RequestCompleted
    RequestFailed

    // Provider
    ProviderCalled
    ProviderFailed
    ProviderRecovered
    ProviderHealthChanged

    // Accounting
    UsageRecorded
    CostRecorded
    BudgetWarning
    BudgetExceeded

    // Security
    SecurityBlocked
    PIIDetected
    SecretDetected
    PromptInjectionDetected

    // Audit
    APIKeyCreated
    APIKeyRotated
    APIKeyRevoked
    PolicyChanged
    ConfigChanged
    UserCreated
    UserDeleted
)
```

Consumers

Consumer Subscribes To Action
Accounting UsageRecorded Update budget, publish CostRecorded
Analytics RequestCompleted Update provider/routing stats
Audit All admin events Persist to audit_logs
Billing CostRecorded External invoicing
Alerts BudgetExceeded, ProviderFailed Slack/PagerDuty
Webhooks Configurable HTTP POST with HMAC

Requirements

· At-least-once delivery
· Idempotent consumers
· Ordered per tenant
· Retry with backoff
· Dead-letter queue

Evolution

Phase 1 (V3.0-V3.6): In-process bus
Phase 2 (V3.7+): NATS / Kafka for large deployments
