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

The PostgreSQL slice adds migrations `0019_v6_runtime.sql` and `0020_v6_runtime_lineage_integrity.sql` and `PostgresStore` methods to create/read runs, perform compare-and-set Run/Task transitions, claim eligible tasks with `FOR UPDATE SKIP LOCKED`, renew only a live lease held by the current worker, and recover expired leases into a bounded retry or terminal failure. Artifact references and immutable execution events can be persisted and queried. Event append, `CompleteTaskWithEvent`, and `RetryTaskWithEvent` now write to the existing transactional `outbox_events` table in the same PostgreSQL transaction as the lineage record. Terminal completion and retry transitions are fenced to the current live lease owner and commit the task transition, lineage event, and delivery intent atomically. The worker emits `task.retrying` lineage for failed attempts that remain eligible for retry; handlers must return an event with a stable ID on both success and failure. The existing outbox publisher provides at-least-once delivery, so consumers must deduplicate by event ID. Tenant IDs scope all reads and state mutations; composite foreign keys prevent cross-tenant Run/Task/Artifact/Event relationships, and migration `0020` additionally enforces that any artifact/event task reference belongs to the same run named by that record. Payloads remain external references.

This is not yet a complete production execution API: end-to-end worker-loop integration, production-role RLS validation, and live PostgreSQL integration execution remain open. The outbox write path is integrated, including CI-gated integration cases for atomic retry state, lineage, and outbox intent, plus end-to-end delivery of a V6 execution event through the existing publisher into an in-memory consumer. These PostgreSQL integration cases require `SENTINELFLOW_DATABASE_URL`; duplicate delivery/inbox deduplication and execution under the intended production database role still need live integration coverage.

## Next implementation slices

1. Add consumer inbox/deduplication where handlers require stronger duplicate-effect protection, and verify duplicate delivery against a live PostgreSQL/event-bus setup.
2. Run PostgreSQL integration tests against a live database, including process interruption, lease renewal/expiry, retries, duplicate delivery, same-tenant cross-run task-reference rejection, and tenant isolation under the intended database role/RLS policy.
3. Add query APIs and tracing/audit lineage for execution inspection.

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
