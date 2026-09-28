# Deployment

SentinelFlow provides reference deployment assets under `deploy/v5`.

## Docker Compose

    docker compose -f deploy/v5/docker-compose.yml up --build

The stack includes Gateway, Control Plane, Worker, PostgreSQL and Redis.

## Kubernetes / Helm

The chart is under `deploy/v5/helm/sentinelflow`.

    helm lint deploy/v5/helm/sentinelflow
    helm upgrade --install sentinelflow deploy/v5/helm/sentinelflow

Provide PostgreSQL, Redis, provider credentials and identity/secret configuration through your cluster's secret management. Do not put production secrets in `values.yaml`.

## Terraform

Reference Kubernetes resources are under `deploy/v5/terraform`.

    terraform init
    terraform plan
    terraform apply

Review the generated plan for your target cluster before applying.

## Production

Read [production-readiness.md](v5/ga/production-readiness.md). Multi-region, data residency, backups, SLOs and provider credentials require environment-specific configuration; the repository's manifests are foundations, not a claim that every cloud environment is production-ready without review.
