# SentinelFlow V3 - High Availability & Disaster Recovery

## High Availability

### Architecture

```

Gateway 1   Gateway 2   Gateway 3
│           │           │
└───────────┼───────────┘
│
┌──────────┴──────────┐
▼                     ▼
Supabase PostgreSQL     Redis
(source of truth)      (cache/state)

```

### Stateless Gateway

All gateway instances are **identical and disposable**:
- No local state (Supabase is source of truth)
- Cache is in Redis (shared)
- Trace store is local (acceptable data loss on restart)
- Rate limits in Redis (shared)

### Health Checks

| Endpoint | Purpose | K8s Probe |
|----------|---------|-----------|
| `/livez` | Process alive | livenessProbe |
| `/readyz` | Ready for traffic (DB OK) | readinessProbe |
| `/health` | Full status (for humans) | - |

### Graceful Shutdown

On SIGTERM:
1. Mark as **not-ready** immediately
2. Wait 3s for LB to remove instance
3. Wait up to 30s for in-flight requests
4. Close connections
5. Exit

### Deployment

```yaml
# Kubernetes
spec:
  replicas: 3
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      terminationGracePeriodSeconds: 45
      containers:
      - name: gateway
        livenessProbe:
          httpGet:
            path: /livez
            port: 8080
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          periodSeconds: 5
```

Disaster Recovery

RPO / RTO

Metric Target Current
RPO (Recovery Point Objective) < 24h Daily backups
RTO (Recovery Time Objective) < 30m ~15m restore

Backup Strategy

Automatic:

· Supabase provides PITR (Point-in-Time Recovery) up to 30 days (depending on plan)
· Daily snapshots retained by Supabase

Manual (recommended):

```bash
# Daily backup script
./scripts/backup-supabase.sh
```

· Dumps full DB (schema + data)
· Compresses with gzip
· Rotates: keeps last 7 days
· Output: backups/sentinelflow-YYYYMMDD-HHMMSS.sql.gz

Restore Procedure

Scenario: Complete database loss

1. Provision new Supabase project (or restore from PITR)
2. Stop gateways (avoid writes to old DB)

```bash
kubectl scale deployment sentinelflow-gateway --replicas=0
```

3. Restore from backup

```bash
gunzip -c backups/sentinelflow-latest.sql.gz | psql "$NEW_DATABASE_URL"
```

4. Verify data

```bash
psql "$NEW_DATABASE_URL" -c "SELECT COUNT(*) FROM organizations;"
psql "$NEW_DATABASE_URL" -c "SELECT COUNT(*) FROM users;"
```

5. Update gateways with new DATABASE_URL

```bash
kubectl set env deployment/sentinelflow-gateway SENTINELFLOW_DATABASE_URL=...
```

6. Scale gateways back up

```bash
kubectl scale deployment sentinelflow-gateway --replicas=3
```

7. Verify traffic

```bash
curl https://gateway.example.com/health
```

DR Test

Run quarterly:

```bash
./scripts/dr-test.sh
```

Verifies:

· Backup creation works
· Backup contains all tables
· Data integrity maintained
· Cleanup works

Failure Scenarios

Gateway instance dies

Impact: Minimal
Recovery: Automatic (LB routes to healthy instances)
Action: None (K8s restarts pod)

All gateways die

Impact: Service unavailable
Recovery: K8s restarts pods
Action: Investigate logs, check resource limits

Supabase unavailable

Impact: Auth fails, usage recording fails
Recovery: Wait for Supabase SLA (99.9%)
Action: Monitor status.supabase.com

Redis unavailable

Impact: Cache misses, rate limits fall back to local
Recovery: Automatic (Sentinel promotes replica)
Action: None

Monitoring

SLOs

SLO Target Alert
Availability 99.9% PagerDuty
P95 overhead < 50ms Slack
Error rate < 0.1% Slack
DB latency < 100ms Warning

Alerts

· SentinelflowDown (5 min no response)
· SentinelflowHighErrorRate (> 1% for 5 min)
· SentinelflowDBUnavailable (health check fails)
· SentinelflowHighLatency (P95 > 500ms)

Runbooks

Gateway not responding

1. Check pods: kubectl get pods -l app=sentinelflow-gateway
2. Check logs: kubectl logs -l app=sentinelflow-gateway --tail=100
3. Check DB: psql $DATABASE_URL -c "SELECT 1"
4. Restart if needed: kubectl rollout restart deployment/sentinelflow-gateway

High error rate

1. Check /v1/providers for provider health
2. Check /v1/traces for failing requests
3. Check circuit breakers: they should open after 5 failures
4. Investigate specific provider issues

DB connection issues

1. Check Supabase dashboard
2. Verify connection string
3. Check pool stats: /health shows pool
4. Increase pool size if needed (config)
   MDEOF
