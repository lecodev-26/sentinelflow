# ADR-0001: PostgreSQL as Source of Truth

**Status:** Accepted  
**Date:** 2026-09-18  
**Deciders:** Architecture team

## Context

V2 used SQLite + in-memory state.

Problems:
- Not horizontally scalable
- No HA
- No replication
- No point-in-time recovery

## Decision

Use **PostgreSQL (Supabase)** as the single source of truth for:
- Organizations, Projects, Users
- API Keys, RBAC
- Providers, Models, Policies
- Budgets, Audit, Usage
- Config versions

Redis remains for cache/state only.

## Consequences

### Positive
- ACID guarantees
- HA via replication
- Point-in-time recovery
- Standard SQL
- Strong ecosystem

### Negative
- Operational overhead
- Requires connection pooling
- Latency to cloud (if Supabase remote)

### Mitigation
- Use `pgx` pool for connections
- Cache hot reads in Redis
- Read replicas for analytics queries

## Alternatives Considered

- **SQLite + Litestream**: good for single node, not multi-node
- **MySQL/MariaDB**: viable, PostgreSQL preferred for JSONB
- **MongoDB**: rejected (we need SQL + joins)
