# SentinelFlow V5 Production Readiness

Before production rollout, verify:

- PostgreSQL, Redis and migrations are healthy.
- Provider credentials are supplied by the configured secret store.
- OIDC/SAML/SCIM and tenant boundaries are configured and tested.
- Provider contract tests and failover tests pass.
- Memory/RAG tenant isolation is tested.
- Tool approval and AI Security policies are tested with representative traffic.
- Load, streaming and regional failover tests pass for the target deployment.
- Backups and restore procedures have been exercised.
- Metrics, traces, logs, SLOs and alerting are connected to the operating environment.
- Evaluation datasets and regression thresholds are defined for critical workloads.
- Container/SBOM/dependency/security checks pass in CI.

GA software status does not remove environment-specific operational responsibilities.
