# SentinelFlow V5 configuration

> **Contributions welcome!** See the [contribution guide](../CONTRIBUTING.md) for the workflow and pull-request requirements.

Runtime policy is managed through the V5 control plane and PostgreSQL. The legacy `rules.yaml` configuration is intentionally removed.

Production secrets must be supplied through the configured `SecretStore`; they are never generated implicitly by the service.
