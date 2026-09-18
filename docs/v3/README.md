# SentinelFlow V3 - Documentation

## Overview

SentinelFlow V3 transforms the V2 gateway into a **production-grade AI Control Plane** with:

- **3 separated planes**: Data, Control, Event
- **PostgreSQL (Supabase)** as source of truth
- **Stateless gateway** (horizontal scaling)
- **Real SSE streaming** end-to-end
- **Tenant isolation** everywhere
- **Policy Engine** with versioning
- **Event-driven** accounting/audit/analytics
- **HA + Chaos + DR** demonstrated by tests

## Index

| Document | Purpose |
|----------|---------|
| [Architecture](./architecture.md) | System design and component diagram |
| [Planes](./planes.md) | Data / Control / Event plane separation |
| [Storage](./storage.md) | PostgreSQL + Redis strategy |
| [Security](./security.md) | Auth, RBAC, tenant isolation |
| [Policy Engine](./policy-engine.md) | Versioned policy engine |
| [Routing](./routing.md) | Constraints, scoring, fallback, experiments |
| [Observability](./observability.md) | OTel, metrics, traces, SLO |
| [Production](./production.md) | HA, DR, chaos, certification |
| [ADRs](./adr/) | Architecture Decision Records |

## Roadmap

| Phase | Deliverable |
|-------|-------------|
| V3.0 | Foundation (Postgres, config system, project structure) |
| V3.1 | Identity (orgs, projects, users, RBAC, API keys) |
| V3.2 | Gateway Core (normalizer, streaming, deadlines, idempotency) |
| V3.3 | Provider Platform (interface, adapters, discovery, health, CB) |
| V3.4 | Routing Intelligence (constraints, scoring, fallback, experiments) |
| V3.5 | Policy & Security (Policy Engine, PII, SSRF, audit) |
| V3.6 | FinOps (UsageEvent, Pricing, Budgets, Forecast) |
| V3.7 | Enterprise (OIDC, SAML, SCIM, Vault, Residency) |
| V3.8 | Observability (OTel, metrics, traces, SLO) |
| V3.9 | HA / DR (multi-replica, Postgres HA, Redis HA, DR) |
| V3.10 | Production Certification (objective tests) |

## Definition of Done

V3 is Production Ready ONLY if:

### Code
- [ ] gofmt
- [ ] go vet
- [ ] go test
- [ ] go test -race
- [ ] go test -fuzz

### Security
- [ ] gosec
- [ ] govulncheck
- [ ] trivy
- [ ] SBOM
- [ ] secrets scan

### Infra
- [ ] multi-node
- [ ] Redis HA
- [ ] Postgres HA
- [ ] backup
- [ ] restore

### Gateway
- [ ] streaming real
- [ ] fallback
- [ ] retry
- [ ] idempotency
- [ ] cache tenant-aware
- [ ] rate-limit tenant-aware

### Enterprise
- [ ] OIDC
- [ ] SAML
- [ ] SCIM
- [ ] RBAC
- [ ] audit
- [ ] residency

### Reliability (Chaos)
- [ ] provider down → PASS
- [ ] redis down → PASS
- [ ] gateway killed → PASS
- [ ] postgres failover → PASS
- [ ] network latency → PASS
- [ ] network loss → PASS
