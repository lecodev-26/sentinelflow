# Operations

## Health

Use the gateway/control-plane health and readiness endpoints exposed by the deployed service. Prometheus metrics are exposed by the metrics endpoint configured for the deployment.

## Migrations

Always inspect migration status before deployment:

    go run ./cmd/migrator status
    go run ./cmd/migrator up

Do not edit an applied migration. Add a new migration for schema changes.

## Backups

PostgreSQL backup/restore scripts live in `scripts/backup-v37.sh` and `scripts/restore-v37.sh`. Restore requires explicit confirmation and validates the backup checksum when available.

Test restores in an isolated environment before relying on a backup for disaster recovery.

## Redis

Redis is used for distributed coordination paths. A Redis outage should be treated as an operational incident; verify the behavior of rate limiting, idempotency and event transport according to the deployed configuration.

## Security

Rotate provider credentials through the configured credential/secret management process. Never paste credentials into logs or issue reports.

## Troubleshooting

Start with:

    go test ./...
    go vet ./...
    git diff --check

Then inspect PostgreSQL/Redis connectivity, migration status, provider health, credentials, authentication configuration and worker/event processing.

For security incidents, follow `SECURITY.md`.
