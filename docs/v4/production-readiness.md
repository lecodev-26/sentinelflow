# SentinelFlow V4 production readiness

## Runtime
- [x] Separate gateway, control plane, worker, CLI and migrator commands
- [x] PostgreSQL and Redis are explicit runtime dependencies
- [x] Graceful shutdown and health/readiness endpoints
- [x] Distributed idempotency, rate limiting and event/outbox primitives exist

## Security
- [x] Production authentication is fail-safe
- [x] Tenant/environment isolation is enforced
- [x] SAML 2.0 and SCIM groups are implemented
- [x] ABAC decision engine exists
- [x] Legacy rules.yaml and SQLite backup path removed
- [x] CI includes dependency verification, vulnerability scan, SBOM and signing

## Data plane
- [x] Provider failover and circuit breakers
- [x] Multi-region placement and residency decisions
- [x] Routing simulation and explainable decisions
- [x] Tool permissions and approval flow
- [x] Cost-aware routing primitives
- [x] Streaming metrics/failure handling

## Operations
- [x] PostgreSQL backup/restore scripts require explicit restore confirmation
- [x] SLO/error-budget engine
- [x] Helm and Terraform deployment manifests
- [x] Chaos/failover tests
- [x] Explicit migrations via cmd/migrator

A deployment is not considered production-ready until sfctl doctor, migration status, provider credentials, Redis/PostgreSQL connectivity, observability and external secret configuration are verified in the target environment.
