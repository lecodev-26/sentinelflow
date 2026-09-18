# SentinelFlow V2 - Architecture

## Overview

SentinelFlow V2 is a production-grade AI Gateway + Control Plane.

```

┌─────────────────────────────────────────────────────────────┐
│                       CLIENTS                               │
└─────────────────────────┬───────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────┐
│              GATEWAY (port 8080)                            │
│                                                             │
│  Pipeline V2.10:                                            │
│  Context → Observability → Metrics → Limits → Bulkhead →   │
│  Dedup → Auth → Security → RateLimit → Quota → Cache →     │
│  Accounting → Engine                                        │
└─────────────────────────┬───────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────┐
│              INTELLIGENT ROUTER                             │
│                                                             │
│  Filter → Score (latency/cost/health/quality) → Select     │
│  → Fallback ordered by score                                │
└─────────────────────────┬───────────────────────────────────┘
│
┌─────────────────┼─────────────────┐
▼                 ▼                 ▼
┌────────┐        ┌────────┐        ┌────────┐
│ OpenAI │        │Anthropic│       │ Local  │
└────────┘        └────────┘        └────────┘

┌─────────────────────────────────────────────────────────────┐
│            CONTROL PLANE (port 8081, /v1)                   │
│                                                             │
│  Organizations · Projects · Users · API Keys · Policies    │
│  Providers · Models · Budgets · Traces · Analytics         │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│              PERSISTENCE                                    │
│                                                             │
│  SQLite (source of truth) + Redis (cache/state)            │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│              METRICS (port 9090)                            │
│                                                             │
│  Prometheus + Grafana + Alertmanager                       │
└─────────────────────────────────────────────────────────────┘

```

## Components

### Gateway Pipeline

Every request goes through 13 stages:

1. **Context** - RequestID, TraceID, Tenant
2. **Observability** - OpenTelemetry span
3. **Metrics** - Prometheus counters
4. **Limits** - 1MB body, 16KB headers
5. **Bulkhead** - Max 200 concurrent
6. **Dedup** - Identical concurrent requests share response
7. **Auth** - API key + scopes
8. **Security** - PII, secrets, prompt injection, SSRF
9. **RateLimit** - Local + distributed (Redis)
10. **Quota** - Per-tenant quotas
11. **Cache** - L1 (memory) → L2 (Redis) → L3 (semantic)
12. **Accounting** - Real tokens + cost + budget
13. **Engine** - Intelligent routing

### Intelligent Router

Decision pipeline:

```

Request
↓
Filter (provider, circuit, health, capabilities, cost)
↓
Score (latency × 0.35 + cost × 0.25 + health × 0.25 + quality × 0.15)
↓
Select best
↓
Log with reason
↓
Fallback ordered by score

```

### Persistence

| Store | Purpose |
|-------|---------|
| **SQLite** | Source of truth: orgs, users, keys, budgets, policies, audit, usage |
| **Redis** | Cache L1, rate limits, distributed state |

### Security Pipeline

```

Request
↓
Auth (API key + scopes)
↓
PII Scanner (email, phone, credit card, SSN)
↓
Secret Scanner (API keys, AWS, passwords)
↓
Prompt Injection Scanner (jailbreak, ignore rules, token theft)
↓
SSRF Scanner (localhost, private IPs, metadata endpoints)
↓
SecurityDecision (allow/redact/warn/block)
↓
Audit (all events)

```

## Version History

| Version | Milestone | Key Features |
|---------|-----------|--------------|
| V2.0 | Foundation | GatewayExecutor, Events, RequestContext |
| V2.1 | Provider Platform | Model Registry, ProviderDescriptor, Discovery |
| V2.2 | Intelligent Routing | Scoring, Filters, Explainability |
| V2.3 | Security | Scopes, Rotation, SecurityDecision, SSRF, Audit |
| V2.4 | Data Plane | Cache L1/L2/L3, Distributed rate limit, Dedup, Bulkhead |
| V2.5 | Accounting | UsageEvent, PricingEngine, BudgetEngine |
| V2.6 | Persistence | SQLite, Repositories, Migrations |
| V2.7 | Observability | Traces, Analytics (provider, routing, cost) |
| V2.8 | Dashboard V2 | Multi-view SPA, Trace Explorer |
| V2.9 | Enterprise | SSO, SCIM, Hierarchy, Vault, Regions |
| V2.10 | Production | HA Redis, Backups, SLOs, Chaos, Load, Security |
