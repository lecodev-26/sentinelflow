# SentinelFlow V3 - Storage

## Overview

```

┌──────────────────────┐       ┌──────────────────────┐
│    PostgreSQL        │       │      Redis           │
│    (Supabase)        │       │                      │
│                      │       │                      │
│  SOURCE OF TRUTH     │       │  CACHE / STATE       │
│                      │       │                      │
│  ─ organizations     │       │  ─ cache L1/L2       │
│  ─ projects          │       │  ─ rate limits       │
│  ─ users             │       │  ─ locks             │
│  ─ memberships       │       │  ─ dedup             │
│  ─ api_keys          │       │  ─ sessions          │
│  ─ providers         │       │  ─ health state      │
│  ─ models            │       │                      │
│  ─ policies          │       │  NEVER source        │
│  ─ routing_rules     │       │  of truth            │
│  ─ budgets           │       │                      │
│  ─ audit_logs        │       │                      │
│  ─ usage_records     │       │                      │
│  ─ webhook_configs   │       │                      │
│  ─ config_versions   │       │                      │
└──────────────────────┘       └──────────────────────┘

```

## Supabase Setup

### 1. Create project

1. Go to https://supabase.com
2. Create new project
3. Wait for provisioning
4. Get connection string

### 2. Connection string

```

postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres

```

### 3. Pooler (recommended for production)

```

postgresql://postgres.[PROJECT-REF]:[PASSWORD]@aws-0-[REGION].pooler.supabase.com:6543/postgres

```

### 4. Environment variable

```bash
export SENTINELFLOW_DATABASE_URL="postgresql://postgres:..."
```

Schema Overview

```sql
-- Identity
CREATE TABLE organizations (...);
CREATE TABLE projects (...);
CREATE TABLE users (...);
CREATE TABLE memberships (...);

-- Access
CREATE TABLE api_keys (...);
CREATE TABLE roles (...);
CREATE TABLE permissions (...);

-- Providers
CREATE TABLE providers (...);
CREATE TABLE provider_credentials (...);
CREATE TABLE models (...);

-- Policy
CREATE TABLE policies (...);
CREATE TABLE policy_versions (...);
CREATE TABLE routing_rules (...);

-- FinOps
CREATE TABLE budgets (...);
CREATE TABLE usage_records (...);

-- Audit
CREATE TABLE audit_logs (...);
CREATE TABLE config_versions (...);
CREATE TABLE webhook_configs (...);
```

Migrations

Use golang-migrate/migrate or embedded SQL migrations:

```
migrations/
├── 0001_organizations.up.sql
├── 0001_organizations.down.sql
├── 0002_projects.up.sql
├── 0002_projects.down.sql
├── ...
```

Run at startup if SENTINELFLOW_AUTO_MIGRATE=true.

Multi-region

For global deployments:

· Primary: Supabase region closest to main traffic
· Read replicas: Supabase read replicas in other regions
· Write routing: always to primary
· Read routing: to nearest replica

Redis Strategy

Data stored in Redis

Key Type TTL Purpose
cache:{tenant}:{hash} String 5m Response cache
ratelimit:{tenant}:{key} Sorted Set 1m Sliding window
lock:{resource} String 30s Distributed locks
dedup:{tenant}:{hash} String 30s Request dedup
session:{id} Hash 24h User sessions
health:{provider} Hash 1m Provider health

Redis HA

· Development: single instance
· Production: Sentinel (1 master + 2 replicas)
· Enterprise: Redis Cluster

Backup Strategy

PostgreSQL (Supabase handles this)

· Point-in-time recovery: 7-30 days (depending on plan)
· Daily snapshots: retained per plan
· Manual exports: pg_dump anytime

Additional backups

```bash
# Daily logical backup
pg_dump "$DATABASE_URL" | gzip > backup-$(date +%Y%m%d).sql.gz

# Upload to S3
aws s3 cp backup-*.sql.gz s3://sentinelflow-backups/
```

Restore

```bash
gunzip -c backup-20260918.sql.gz | psql "$DATABASE_URL"
```

Data Retention

Configurable per tenant:

```yaml
retention:
  usage_records: 90d
  audit_logs: 365d
  config_versions: forever
```

Automated cleanup worker:

· Deletes old records
· Runs daily at 03:00 UTC
· Logs actions
