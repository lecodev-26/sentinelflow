# SentinelFlow V2 - Operations

## Deployment

### Docker Compose (recommended)

```bash
cd deployments/docker
cp .env.example .env
# Edit .env with your keys
docker-compose -f docker-compose.prod.yml up -d
```

Services:

· SentinelFlow: http://localhost:8080
· Control Plane: http://localhost:8081
· Prometheus: http://localhost:9091
· Grafana: http://localhost:3000
· Alertmanager: http://localhost:9093

Kubernetes

```bash
helm install sentinelflow ./deployments/helm/sentinelflow \
  --namespace sentinelflow \
  --create-namespace
```

Monitoring

SLOs

Metric Target
Availability 99.9%
P95 Latency < 1s
Error rate < 0.1%
Cache hit rate 30%

Alerts

Configured in deployments/prometheus/alerts.yml:

· High error rate (>0.1% for 5m)
· High latency (P95 >1s for 5m)
· Provider down (2m)
· Circuit breaker open (5m)
· Budget >80%
· Redis down (1m)
· High rate limiting
· High memory usage
· Goroutine leak

Backup

Manual

```bash
./scripts/backup.sh
```

Automatic (in docker-compose)

The backup service runs hourly and:

1. Backs up SQLite with .backup (safe)
2. Compresses with gzip
3. Uploads to S3 if configured
4. Deletes backups older than 7 days

Restore

```bash
gunzip -c backups/sentinelflow-20260918.db.gz | sqlite3 sentinelflow.db
```

Chaos Testing

Run the chaos test suite:

```bash
API_KEY=sf_xxx ./scripts/chaos-test.sh
```

Tests:

· Provider failover
· Timeout respected
· 50 concurrent requests
· Rate limiting active
· Health check
· Prometheus metrics
· Circuit breaker status
· Control plane health

Load Testing

```bash
k6 run tests/load/k6-test.js
```

Stages:

· Ramp to 10 users (30s)
· Stay at 50 users (1m)
· Peak 100 users (1m)
· Ramp down (30s)

Thresholds:

· P95 < 2000ms
· Error rate < 1%

Security Testing

```bash
API_KEY=sf_xxx ./scripts/security-test.sh
```

Tests:

· Auth required
· Invalid auth rejected
· Body size limit
· Prompt injection blocked
· Path traversal blocked
· Method restrictions
· SSRF attempts blocked

Incident Response

Provider down

1. Alert: "Provider X unhealthy"
2. Check: curl http://localhost:8081/v1/providers
3. Traffic automatically routed to fallback
4. Circuit breaker opens after 5 failures
5. Check provider status page

High error rate

1. Alert: "Error rate > 0.1%"
2. Check: curl http://localhost:8081/v1/metrics/overview
3. Check logs: docker logs sentinelflow -f
4. Check traces: curl http://localhost:8081/v1/traces

Redis down

1. Alert: "Redis down"
2. SentinelFlow continues with L1 cache only
3. Rate limiting falls back to local
4. Redis sentinel will promote replica

Budget exceeded

1. Alert: "Budget tenant=X at 90%"
2. Check: curl http://localhost:8081/v1/metrics/costs
3. Increase budget or optimize usage

Scaling

Vertical

Update docker-compose resources:

```yaml
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 4G
```

Horizontal

```bash
docker-compose -f docker-compose.prod.yml up -d --scale sentinelflow=3
```

For Kubernetes:

```bash
kubectl scale deployment sentinelflow --replicas=5
```

Key Rotation

API Keys

```bash
# Create new key
curl -X POST http://localhost:8081/v1/users/{id}/api-keys \
  -H "Content-Type: application/json" \
  -d '{"name":"new-key"}'

# Revoke old key
curl -X POST http://localhost:8081/v1/api-keys/{key}/revoke
```

Vault Key

```go
// Programmatic
vault.RotateKey("new-master-key")
```

Provider Credentials

```bash
# Update in vault
# Vault supports AES-256-GCM with key rotation
```

