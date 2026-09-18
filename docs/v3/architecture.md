# SentinelFlow V3 - Architecture

## Core Principle

V3 separates the system into **3 planes**:

```

┌─────────────────────────────────────────────────────────────┐
│                      CLIENTS                                │
│         Applications · Agents · SDKs                        │
└─────────────────────────┬───────────────────────────────────┘
│ HTTPS
▼
┌─────────────────────────────────────────────────────────────┐
│                     DATA PLANE                              │
│                   (stateless, fast)                         │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │                  Gateway Pipeline                     │ │
│  │                                                       │ │
│  │  RequestID → Auth → Tenant → Policy → RateLimit →   │ │
│  │  Dedup → Cache → Router → Provider → Stream →       │ │
│  │  Validation → Accounting → Telemetry                 │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────┬───────────────────────────────────┘
│
┌─────────────────┼─────────────────┐
▼                 ▼                 ▼
┌────────┐        ┌────────┐        ┌────────┐
│ OpenAI │        │Anthropic│       │ Local  │
└────────┘        └────────┘        └────────┘

┌─────────────────────────────────────────────────────────────┐
│                    CONTROL PLANE                            │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  REST API + UI                                        │ │
│  │  Organizations · Projects · Users · API Keys         │ │
│  │  Providers · Models · Policies · Budgets             │ │
│  │  Routing · Tenants · Audit · Config · Secrets        │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────┬───────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────┐
│                     EVENT PLANE                             │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  Event Bus (in-process → NATS/Kafka)                 │ │
│  │                                                       │ │
│  │  RequestCompleted · ProviderCalled · UsageRecorded  │ │
│  │  BudgetExceeded · APIKeyRotated · PolicyChanged     │ │
│  │  ProviderHealthChanged · AuditEvent                 │ │
│  └───────────────────────────────────────────────────────┘ │
│                          │                                  │
│         ┌────────────────┼────────────────┐                 │
│         ▼                ▼                ▼                 │
│    ┌────────┐      ┌────────┐       ┌────────┐            │
│    │Account │      │Analytics│      │Billing │            │
│    └────────┘      └────────┘       └────────┘            │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    PERSISTENCE                              │
│                                                             │
│  ┌──────────────────┐     ┌──────────────────┐             │
│  │   PostgreSQL     │     │     Redis        │             │
│  │   (Supabase)     │     │                  │             │
│  │                  │     │                  │             │
│  │ SOURCE OF TRUTH  │     │  CACHE / STATE   │             │
│  │                  │     │                  │             │
│  │ organizations    │     │  cache L1/L2     │             │
│  │ projects         │     │  rate limits     │             │
│  │ users            │     │  locks           │             │
│  │ api_keys         │     │  dedup           │             │
│  │ providers        │     │  sessions        │             │
│  │ models           │     │  health state    │             │
│  │ policies         │     │                  │             │
│  │ budgets          │     │  NEVER source    │             │
│  │ audit_logs       │     │  of truth        │             │
│  │ config_versions  │     │                  │             │
│  └──────────────────┘     └──────────────────┘             │
└─────────────────────────────────────────────────────────────┘

```

## Design Principles

### 1. Data Plane is Stateless

Every gateway instance is identical and disposable.

```

Load Balancer
│
├──► Gateway 1 ──┐
├──► Gateway 2 ──┼──► Redis + Postgres
└──► Gateway 3 ──┘

```

Kill any gateway → traffic continues.

### 2. Control Plane is Authoritative

All configuration lives in PostgreSQL. No in-memory state.

### 3. Event Plane is Asynchronous

Accounting, audit, analytics are **consumers**, not inline operations.

### 4. Tenant Isolation Everywhere

Every operation is scoped by `tenant_id` + `project_id`.

Includes: cache, rate limit, dedup, usage, budget, audit, routing, credentials.

### 5. Fail Safely

- No credentials → fail at startup
- Invalid config → reject, keep previous
- Redis down → fallback to local
- Postgres down → read-only mode + alert

## Component Map

```

cmd/
├── gateway/          ← Data Plane binary
├── controlplane/     ← Control Plane binary
├── worker/           ← Event consumers binary
└── cli/              ← sfctl

internal/
├── gateway/          ← Data Plane logic
├── controlplane/     ← Control Plane logic
├── identity/         ← Users, orgs, projects, keys, RBAC
├── providers/        ← Provider adapters + discovery
├── routing/          ← Routing engine
├── policy/           ← Policy engine
├── cache/            ← L1/L2/semantic
├── security/         ← PII, secrets, SSRF, prompt
├── accounting/       ← Usage, pricing, budgets
├── events/           ← Event bus
├── storage/          ← Postgres + Redis
├── observability/    ← OTel, metrics, traces
├── enterprise/       ← SSO, SCIM, Vault, Regions
└── version/          ← Single source of version

```

## Migration from V2

V3 is **incremental**:

| Component | V2 | V3 |
|-----------|-----|-----|
| Storage | SQLite | PostgreSQL (Supabase) |
| Gateway | Monolith | Stateless |
| Config | YAML | Config versions |
| Events | In-process | Async consumers |
| Secrets | Internal vault | External + internal |
| Streaming | Partial | Real SSE |
| Policy | Middleware | Policy Engine |
