# Quickstart

## Local development

Requirements: Go 1.27+, PostgreSQL and Redis for distributed paths.

    git clone https://github.com/lecodev-26/sentinelflow.git
    cd sentinelflow
    cp .env.example .env
    go test ./...
    go build ./cmd/gateway ./cmd/controlplane ./cmd/worker ./cmd/cli ./cmd/migrator

Configure your own PostgreSQL/Redis/provider credentials in the environment. Never commit `.env`.

## V5 Compose

    docker compose -f deploy/v5/docker-compose.yml up --build

For production deployment, follow [deployment.md](deployment.md) and the [readiness checklist](v5/ga/production-readiness.md).

## Migrations

    go run ./cmd/migrator status
    go run ./cmd/migrator up

Migrations are explicit; application startup should not silently modify production schemas.
