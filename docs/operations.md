# SentinelFlow - Operations

## Day-to-day Operations

### Check system health

```bash
curl http://localhost:8080/health
curl http://localhost:8081/v1/health
```

View provider status

```bash
curl http://localhost:8081/v1/providers
```

View circuit breakers

```bash
curl http://localhost:8081/v1/circuit-breakers
```

View metrics

```bash
curl http://localhost:9090/metrics
```

Incident Response

Provider down

1. Detect: Alert "Provider down"
2. Verify: curl http://localhost:8081/v1/providers
3. Impact: Check if circuit breaker opened
4. Mitigation: Traffic automatically routed to fallback
5. Fix: Investigate provider status page

High error rate

1. Detect: Alert "Error rate > 1%"
2. Verify: curl http://localhost:8081/v1/metrics/overview
3. Impact: Check logs for error patterns
4. Mitigation: Circuit breaker will isolate failing provider
5. Fix: Investigate root cause

Budget exceeded

1. Detect: Alert "Budget > 100%"
2. Verify: curl http://localhost:8081/v1/budgets
3. Impact: Tenant requests may be blocked
4. Mitigation: Increase budget or optimize usage
5. Fix: Update budget via admin API

Maintenance

Rotate API keys

```bash
# Create new key
curl -X POST http://localhost:8081/v1/users/{id}/api-keys

# Revoke old key
curl -X POST http://localhost:8081/v1/api-keys/{key}/revoke
```

Update configuration

```bash
# Edit configs/rules.yaml
# Restart service
kubectl rollout restart deployment/sentinelflow
```

Scale up/down

```bash
kubectl scale deployment sentinelflow --replicas=5
```

Backup & Recovery

Redis backup

```bash
# Enable AOF
redis-cli CONFIG SET appendonly yes

# Manual save
redis-cli BGSAVE
```

Configuration backup

```bash
# ConfigMaps are in version control
git pull origin main
```

Troubleshooting

Gateway not responding

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=sentinelflow

# Check logs
kubectl logs -l app.kubernetes.io/name=sentinelflow --tail=100
```

High latency

1. Check provider health: /v1/providers
2. Check circuit breakers: /v1/circuit-breakers
3. Check Redis connectivity
4. Check system metrics: /v1/metrics/system

Cache not working

1. Check Redis connection
2. Check cache config in rules.yaml
3. Check cache HIT/MISS headers

SLOs

Metric Target
Availability 99.9%
P95 latency < 1s
Error rate < 0.1%
Cache hit rate 30%

