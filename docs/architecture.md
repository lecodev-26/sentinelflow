# SentinelFlow - Architecture

## Overview

SentinelFlow is an AI Gateway that sits between your application and LLM providers.

```

┌────────────────────────────────────────────────────────────────┐
│                        CLIENTS                                 │
│              Applications / Agents / SDKs                      │
└──────────────────────────┬─────────────────────────────────────┘
│
▼
┌────────────────────────────────────────────────────────────────┐
│                    GATEWAY (port 8080)                         │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Pipeline                              │  │
│  │                                                          │  │
│  │  Context → Observability → Metrics → Limits → Auth →    │  │
│  │  Security → RateLimit → Quota → Cache → Cost → Engine   │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────┬─────────────────────────────────────┘
│
▼
┌────────────────────────────────────────────────────────────────┐
│                   ENGINE (rules.Engine)                        │
│                                                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Router     │  │   Registry   │  │   Health     │         │
│  │ (strategies) │  │  (providers) │  │  (monitor)   │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Circuit Breaker per provider                │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────┬─────────────────────────────────────┘
│
┌──────────────────┼──────────────────┐
▼                  ▼                  ▼
┌─────────┐        ┌─────────┐        ┌─────────┐
│ OpenAI  │        │Anthropic│        │  Local  │
└─────────┘        └─────────┘        └─────────┘

┌────────────────────────────────────────────────────────────────┐
│              CONTROL PLANE (port 8081, /v1/)                   │
│                                                                │
│  Organizations / Users / API Keys / Providers / Metrics       │
└────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────┐
│                 METRICS (port 9090, /metrics)                  │
│                                                                │
│                    Prometheus endpoint                         │
└────────────────────────────────────────────────────────────────┘

```

## Components

### Gateway Pipeline

Every request goes through:

1. **Context**: Creates RequestContext (request_id, trace_id, tenant, user)
2. **Observability**: Starts OpenTelemetry span
3. **Metrics**: Records Prometheus metrics
4. **Limits**: Enforces body/header/method limits
5. **Auth**: Validates API key, extracts tenant/user/role
6. **Security**: PII, secrets, prompt injection detection
7. **RateLimit**: Per-IP rate limiting
8. **Quota**: Per-tenant quotas
9. **Cache**: Checks cache before routing
10. **Cost**: Records real token usage and cost
11. **Engine**: Routes request to best provider

### Engine

The Engine is the core router:

- **Registry**: Manages all providers
- **Router**: Selects best provider using strategies
- **Health Monitor**: Background checks provider health
- **Circuit Breaker**: Prevents cascading failures

### Providers

Each provider implements:

```go
type Provider interface {
    Name() string
    Chat(ctx, *ChatRequest) (*ChatResponse, error)
    Stream(ctx, *ChatRequest) (<-chan Event, error)
    Health(ctx) error
    Models(ctx) ([]string, error)
}
```

Control Plane

REST API for administration:

· /v1/organizations - Manage tenants
· /v1/users - Manage users
· /v1/api-keys - Manage API keys
· /v1/providers - View provider status
· /v1/metrics/* - View metrics

Data Flow

Request Flow

```
1. Client → Gateway
2. Pipeline processes request
3. Engine selects provider
4. Provider executes request
5. Response → Cost recording
6. Response → Client
```

Failover Flow

```
1. Engine tries provider A
2. If A fails → circuit breaker records failure
3. If A fails → try provider B
4. If B fails → try provider C
5. If all fail → 503
```

Cache Flow

```
1. Check L1 (memory) → HIT? Return
2. Check L2 (Redis) → HIT? Return + populate L1
3. MISS → execute request
4. Store response in L1 + L2
```

Security

Authentication

· API keys hashed with SHA-256
· Never stored in plaintext
· Validated on every request

Authorization

· RBAC: Admin, Editor, Viewer, Member
· Tenant isolation: strict separation
· Policy engine: dynamic rules

Protection

· SSRF: blocks localhost, private IPs, metadata
· PII: detects and redacts
· Secrets: detects and blocks
· Prompt injection: detects and blocks
