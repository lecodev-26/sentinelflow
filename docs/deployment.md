# SentinelFlow - Deployment

## Docker

### Build

```bash
docker build -t sentinelflow:1.0.0 .
```

Run

```bash
docker run -d \
  --name sentinelflow \
  -p 8080:8080 \
  -p 8081:8081 \
  -p 9090:9090 \
  -e OPENAI_API_KEY=sk-... \
  -e ANTHROPIC_API_KEY=ant-... \
  sentinelflow:1.0.0
```

Docker Compose

Full stack with Redis, Prometheus, Grafana:

```bash
docker-compose up -d
```

Services:

· SentinelFlow: http://localhost:8080
· Control Plane: http://localhost:8081
· Prometheus: http://localhost:9091
· Grafana: http://localhost:3000 (admin/admin)

Kubernetes

Prerequisites

· Kubernetes 1.24+
· kubectl
· Helm 3+

With Helm

```bash
helm install sentinelflow ./deployments/helm/sentinelflow \
  --namespace sentinelflow \
  --create-namespace \
  --set secrets.openai.apiKey=sk-... \
  --set secrets.anthropic.apiKey=ant-...
```

With Terraform

```bash
cd deployments/terraform
terraform init
terraform plan
terraform apply
```

Production Checklist

☐ API keys stored in secrets (not env vars)
☐ TLS enabled on all endpoints
☐ Redis in HA mode
☐ Prometheus scraping enabled
☐ Alertmanager configured
☐ Backup strategy for Redis
☐ Log aggregation setup
☐ Resource limits configured
☐ PDB configured (minAvailable: 2)
☐ HPA configured (min: 2, max: 10)
☐ Network policies applied
☐ Pod security standards enforced

Monitoring

Prometheus

Metrics available at /metrics:

· sentinelflow_requests_total
· sentinelflow_request_duration_seconds
· sentinelflow_provider_failures_total
· sentinelflow_fallbacks_total
· sentinelflow_cache_hits_total
· sentinelflow_ttft_seconds
· sentinelflow_tokens_total
· sentinelflow_cost_usd_total

Grafana

Import dashboards from deployments/grafana/dashboards/.

Alerts

Recommended alerts:

· Provider down > 5 minutes
· Error rate > 1%
· P95 latency > 5s
· Budget > 80%
· Circuit breaker open > 5 minutes
