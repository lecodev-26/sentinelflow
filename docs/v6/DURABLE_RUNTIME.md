# V6 Durable Runtime — implementation status

**Status:** in progress. The first increment establishes storage-agnostic domain contracts and lifecycle transition rules. It does not yet provide durable persistence, distributed scheduling, worker leases, or crash recovery.

## Current increment

The Go package `internal/runtime/v6` defines:

- tenant-scoped Run and Task contracts with principal/correlation identity;
- Artifact references that keep payload bytes outside the domain record;
- versioned ExecutionEvent metadata for future lifecycle lineage;
- explicit Run and Task lifecycle states and transition predicates;
- validation for required identity fields, retry bounds, lease ownership, and timestamp ordering.

The models are deliberately storage-agnostic so persistence and worker implementations can be tested against the same invariants.

## Next implementation slices

1. Define PostgreSQL schema and transactional repository interfaces for runs, tasks, artifacts, idempotency records, and lifecycle events.
2. Implement compare-and-set state transitions and atomic task claiming/lease renewal.
3. Add an outbox/inbox strategy for event delivery and duplicate handling.
4. Implement bounded retries, lease expiry recovery, cancellation, and stale-worker protection.
5. Add integration tests for process interruption, concurrent claims, duplicate delivery, and tenant isolation.
6. Add query APIs and tracing/audit lineage for execution inspection.

## Safety and correctness constraints

- Tenant and principal identity are mandatory on every Run; tenant identity is mandatory on every Task and Artifact.
- Terminal states cannot transition back to active states.
- Retry attempts are bounded by an explicit maximum.
- Lease expiry without an owner is invalid.
- External side effects must not be described as exactly-once; future adapters must use idempotency or reconciliation where available.
- A successful unit test or in-memory transition predicate is not evidence of durable recovery.

## Validation for this increment

```bash
go test ./internal/runtime/v6 ./internal/runtime/v4 ./internal/agents/v5
git diff --check
```

Durability acceptance criteria remain open until the persistence, scheduling, recovery, and integration-test slices are implemented.
