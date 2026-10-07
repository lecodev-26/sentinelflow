## [5.0.1] - 2026-10-07

### Fixed
- Hardened the public release packaging and runtime version metadata.
- Corrected Docker Compose Redis configuration to use `SENTINELFLOW_REDIS_URL`.
- Release builds now inject version metadata into `internal/version`.

### Security
- Release artifacts are built and attested by the GitHub Actions release pipeline.

## [5.0.0] - 2026-09-28

### Added
- Intelligent AI core with intent, complexity and capability analysis.
- Adaptive routing and outcome learning primitives.
- Agent runtime with approvals, cancellation and tool execution.
- Tenant/project-isolated memory and knowledge/RAG foundations.
- AI Security 2.0 with risk-based ALLOW/REVIEW/BLOCK decisions.
- Predictive FinOps, semantic AI observability and evaluation primitives.
- Prompt versioning, multimodal request parts and AI governance.
- Plugin registry, global routing, self-healing, Zero Trust AI and autoscaling foundations.
- Dedicated V5 CI, regression harness and release documentation.

### Release status
- V5.0.1 is the current published GA maintenance release.
- Production deployment remains environment-specific; follow the V5 production-readiness checklist before operating at scale.

## [4.0.0] - 2026-09-28

### Added
- V4 foundation, distributed runtime, policy lifecycle, routing, FinOps, observability and enterprise contracts.
- Developer portal/playground and Go/Python SDKs.
- Docker Compose, Helm, Terraform, SBOM/security CI and production-readiness documentation.

### Changed
- V4 uses explicit PostgreSQL migrations and removes the legacy rules.yaml/SQLite configuration path.
- Repository documentation and community health files were refreshed for open-source contributors.

---

# Changelog

## [3.0.0] - 2026-09-22

### 🎉 SentinelFlow V3 - AI Gateway + Model Control Plane

Reescritura completa con arquitectura de 3 planos.

### Added

#### Foundation (V3.0)
- `internal/version`: single source of truth
- `internal/config/v3`: static/dynamic/secrets separation
- `internal/storage/postgres`: pgx pool + migraciones embebidas
- Nueva estructura `cmd/`: gateway, controlplane, worker, cli

#### Identity (V3.1)
- Repositorios persistentes: organizations, projects, users, api_keys
- API keys SHA-256 + scopes + TTL + rotation

#### Gateway Core (V3.2)
- RequestNormalizer: OpenAI/Anthropic/Gemini
- AuthMiddleware contra PostgreSQL
- IdempotencyMiddleware
- Streaming SSE real

#### Provider Platform (V3.3)
- Interface Provider con Chat/Stream/Health/Models
- Registry thread-safe
- Health monitor cada 30s
- Circuit breaker por provider
- Adapters: OpenAI, Anthropic, Ollama

#### Routing Intelligence (V3.4)
- Model Registry con 9 modelos + pricing real
- Scoring: latency 0.35 + cost 0.25 + health 0.25 + quality 0.15
- Filters + Experiments (A/B, canary)

#### Policy & Security (V3.5)
- Policy Engine versionado
- PII, Secret, Prompt Injection, SSRF scanners
- Auto-redaction

#### FinOps (V3.6)
- usage_records + budgets
- UsageRepo + BudgetRepo
- Alerts 80/90/100%

#### Enterprise (V3.7)
- OIDC (5 providers)
- SCIM 2.0
- Regional routing + GDPR

#### Observability (V3.8)
- Trace Store con rotación
- Request timeline
- X-Trace-Id header

#### HA/DR (V3.9)
- /livez /readyz probes
- Graceful shutdown
- Backup + DR tests

### Fixed
- Race condition en Cache.Get()
- Race condition en ValidateAPIKey()
- Snapshot() no copia mutex
- Migración 0005 sin expression index no-inmutable

### Security
- API keys SHA-256 hasheadas
- Idempotency-Key anti-duplicados
- SSRF protection metadata cloud
- PII auto-redaction

---

## [2.0.0] - 2026-09-16

Gateway Pipeline, Events bus, Auth mandatory, Multi-tenancy, Cost tracking, OTel, Docker+K8s+Helm+Terraform.

## [1.0.0] - 2026-08

Gateway con failover entre providers, Smart routing, Cache, Métricas, Dashboard.
