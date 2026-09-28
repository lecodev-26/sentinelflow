# SentinelFlow V4 configuration

Runtime policy is managed through the V4 control plane and PostgreSQL. The legacy `rules.yaml` configuration is intentionally removed.

Production secrets must be supplied through the configured `SecretStore`; they are never generated implicitly by the service.
