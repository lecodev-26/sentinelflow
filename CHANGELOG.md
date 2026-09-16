# Changelog

Todas las versiones notables de SentinelFlow se documentan en este archivo.

El formato sigue [Keep a Changelog](https://keepachangelog.com/es/1.1.0/),
y este proyecto adhiere a [Semantic Versioning](https://semver.org/lang/es/).

## [Unreleased]

## [1.0.0] - 2026-09-16

### Añadido

#### Core Gateway
- Pipeline unificado con RequestContext
- Provider Runtime con Registry
- Smart Router con estrategias (health, latency, cost)
- Circuit Breaker por provider
- Retry con backoff exponencial y jitter
- Failover automático entre providers

#### Seguridad
- API keys hasheadas con SHA-256
- RBAC (Admin, Editor, Viewer, Member)
- Organizaciones y proyectos (multi-tenancy)
- PII detection y redacción
- Secret detection (API keys, AWS, contraseñas)
- Prompt injection detection
- SSRF protection (bloquea localhost, IPs privadas, metadata)
- Audit logs
- Policy engine con reglas dinámicas

#### Resiliencia
- Health monitor en background
- Circuit breaker con estados (closed/open/half-open)
- Rate limiting (IP, tenant, user)
- Quotas por tenant (req/min, tokens/min, tokens/mes)
- Request limits (1MB body, 16KB headers)
- Graceful shutdown

#### Cost Intelligence
- Cost tracking por request con tokens reales
- Pricing registry por provider/model
- Budgets por tenant ($100/mes por defecto)
- Alertas en 80%, 90%, 100%

#### Cache
- Cache determinista con keys versionadas
- Cache semántica con embeddings
- L1 (memoria) + L2 (Redis)
- Invalidación por tenant/project

#### Observabilidad
- OpenTelemetry tracing
- Métricas Prometheus (requests, latencia, tokens, cost)
- TTFT (Time To First Token)
- Logs estructurados sin PII
- Correlation ID entre logs/traces/métricas

#### Control Plane
- API administrativa en puerto separado
- Endpoints de métricas reales
- Organizaciones, usuarios, API keys
- Providers, circuit breakers, budgets

#### Dashboard
- UI glassmorphism minimalista
- Modo oscuro/claro con persistencia
- Command palette (Ctrl+K)
- Gráficos con Chart.js
- Auto-refresh cada 5s
- Export de métricas a JSON

#### Deployment
- Dockerfile distroless nonroot
- Docker-compose con Redis + Prometheus + Grafana
- Kubernetes manifests con PDB, probes, resource limits
- Helm chart parametrizado
- Terraform para infraestructura como código

#### Testing
- Unit tests (config, rbac, ssrf, provider)
- Fuzzing tests (ChatRequest, Message)
- E2E tests (routing, chaos, security)
- CI con race detector, gosec, govulncheck, trivy

### Documentación
- README completo con arquitectura
- CHANGELOG
- API reference

---

## Tipos de cambios

- `Añadido` para nuevas características
- `Cambiado` para cambios en funcionalidad existente
- `Obsoleto` para características que serán removidas
- `Eliminado` para características removidas
- `Corregido` para bugs
- `Seguridad` para vulnerabilidades
