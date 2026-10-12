# V6 Durable Runtime — implementation status

**Status:** in progress. Domain contracts are now backed by an initial PostgreSQL schema and a transactional/concurrency-aware Run/Task store. The runtime is not yet a complete production scheduler: end-to-end crash recovery, durable event delivery, and concurrency/tenant-isolation integration coverage remain open.

## Current increment

The Go package `internal/runtime/v6` defines:

- tenant-scoped Run and Task contracts with principal/correlation identity;
- Artifact references that keep payload bytes outside the domain record;
- versioned ExecutionEvent metadata for future lifecycle lineage;
- explicit Run and Task lifecycle states and transition predicates;
- validation for required identity fields, retry bounds, lease ownership, and timestamp ordering.

The models are deliberately storage-agnostic so persistence and worker implementations can be tested against the same invariants.

The first PostgreSQL slice adds migration `0019_v6_runtime.sql` and `PostgresStore` methods to create/read runs, perform compare-and-set Run/Task transitions, claim eligible tasks with `FOR UPDATE SKIP LOCKED`, renew only a live lease held by the current worker, and recover expired leases into a bounded retry or terminal failure. Tenant IDs scope all reads and state mutations; composite foreign keys prevent cross-tenant Run/Task/Artifact/Event relationships. Payloads remain external references.

This is not yet a complete transactional execution API: artifact/event write methods, atomic state-change plus outbox event commits, worker-loop integration, and real PostgreSQL concurrency tests remain to be implemented.

## Next implementation slices

1. Complete repository operations for artifacts and immutable lifecycle events, including atomic state-change plus outbox writes.
2. Integrate the store with a worker loop and define cancellation and stale-worker fencing semantics for external side effects.
3. Add an outbox/inbox strategy for event delivery and duplicate handling.
4. Add PostgreSQL integration tests for process interruption, concurrent claims, lease renewal/expiry, retries, duplicate delivery, and tenant isolation.
5. Add query APIs and tracing/audit lineage for execution inspection.

## Safety and correctness constraints

- Tenant and principal identity are mandatory on every Run; tenant identity is mandatory on every Task and Artifact.
- Terminal states cannot transition back to active states.
- Retry attempts are bounded by an explicit maximum.
- Lease expiry without an owner is invalid.
- External side effects must not be described as exactly-once; future adapters must use idempotency or reconciliation where available.
- A successful unit test or in-memory transition predicate is not evidence of durable recovery.

## Validation for this increment

```bash
go test ./internal/runtime/v6 ./internal/runtime/v4 ./internal/agents/v5 ./internal/storage/postgres
git diff --check
```

Durability acceptance criteria remain open until the persistence, scheduling, recovery, and integration-test slices are implemented.
