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
- CI passes formatting, tests, race detection, vetting, builds and Go vulnerability scanning for the server and public Go SDK.
- The release pipeline generates the required SBOM and provenance/signing artifacts before publication.

GA software status does not remove environment-specific operational responsibilities.
