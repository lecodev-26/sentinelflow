# SentinelFlow V6 — Master Roadmap & Architecture

**Status:** architecture baseline for active V6 development. This document describes intended contracts and delivery gates; it does not claim that every capability is implemented or production-ready.

V6 evolves SentinelFlow into a durable, governed autonomous AI execution platform. Its core design goal is to make each run recoverable, policy-constrained, observable, and attributable from request acceptance through completion and artifact retention.

## Architectural principles

1. **Durability over best effort.** Persist lifecycle transitions and the information needed to recover work after a process or worker failure.
2. **Explicit identity and authority.** Every run, task, agent, tool invocation, and artifact is associated with an authenticated principal and tenant scope.
3. **Policy throughout execution.** Authorization, risk classification, budget checks, and approval requirements apply at each privileged boundary—not only at ingress.
4. **Tenant isolation by construction.** Storage queries, cache keys, event routing, credentials, artifacts, and telemetry preserve tenant boundaries.
5. **Idempotent recovery.** Retries, redelivery, and worker recovery use stable identifiers and explicit idempotency semantics; do not promise exactly-once side effects.
6. **Auditable autonomy.** Decisions and external effects have correlation, provenance, and audit records sufficient to reconstruct what happened.
7. **Bounded execution.** Time, concurrency, depth, retries, tokens/cost, and resource use have explicit limits and fail-closed behavior where limits cannot be enforced.
8. **Observable outcomes.** Execution state, latency, reliability, quality, and cost can be correlated across runs, tasks, tools, models, and providers.
9. **Compatibility by contract.** Preserve stable V5 contracts where practical; version V6 APIs and persisted/event schemas deliberately.
10. **Evidence before GA.** Experimental or partially implemented capabilities must be labelled accurately. Release readiness requires repeatable test, security, recovery, and operational evidence.

## System boundaries and target planes

- **API / ingress:** authenticates the caller, establishes tenant and request context, validates inputs, applies ingress limits, and creates or queries durable runs.
- **Control plane:** owns organizations/projects/environments, identities, policy, tool and model registrations, quotas, budgets, approval rules, and configuration history.
- **Execution plane:** schedules durable runs and tasks, leases work to workers, records lifecycle transitions, applies retry/timeout/compensation rules, and recovers interrupted execution.
- **Tool and provider boundary:** exposes versioned capability contracts; checks identity, permissions, risk, budgets and approvals before invocation; normalizes results and records external effects.
- **State and event boundary:** persists authoritative state and uses transactional outbox/inbox or equivalent idempotent delivery patterns where a database transaction must coordinate with asynchronous work.
- **Artifact and knowledge boundary:** stores outputs with tenant ownership, provenance, retention/deletion rules, and bounded access.
- **Observability and audit:** emits correlated logs, metrics, traces, cost/quality signals, security events, and immutable-enough audit history appropriate to the configured storage guarantees.

These are logical boundaries, not a requirement that each plane run as a separate service. Deployments may begin as a modular service while preserving interfaces that allow safe separation later.

## Core execution model

The initial domain vocabulary is:

- **Run:** a durable, tenant-scoped unit of requested work with an idempotency/correlation identity and an explicit lifecycle.
- **Task:** a bounded unit of work owned by a run, with defined dependencies, attempts, deadlines, and terminal outcomes.
- **Artifact:** an input, intermediate result, or output with type, provenance, access scope, and retention metadata.
- **Event:** a versioned statement about a lifecycle or business transition, carrying a stable event identity and correlation context.
- **Tool invocation:** a recorded request to exercise a registered capability under an explicit identity and policy decision.
- **Approval:** a durable decision bound to the exact action, risk/context, approver, and policy version it authorizes.

Lifecycle state changes must be validated transitions, not arbitrary status overwrites. Worker leases and retry policies must make ownership and stale-worker behavior explicit. Cancellation, timeout, retry exhaustion, and compensation are observable outcomes. External side effects cannot be assumed transactional with SentinelFlow state; use idempotency keys, deduplication, or explicit reconciliation where supported.

## Security, governance, and isolation

- Derive tenant and principal from authenticated context; never trust caller-supplied tenant identifiers as authorization proof.
- Apply least privilege and capability-scoped credentials. A tool or agent cannot silently expand its authority through delegation.
- Deny by default when authorization, policy evaluation, approval state, or a required budget check is unavailable.
- Treat prompts, model outputs, retrieved documents, tool results, and uploaded media as untrusted input.
- Bound and validate payloads, file/media types, redirects, destinations, execution time, and resource consumption at the relevant boundary.
- Keep secrets out of logs, traces, error messages, artifacts, and model-visible context unless explicitly permitted.
- Record policy version and decision context for consequential actions; provide revocation and audit paths.
- Enforce tenant scope consistently in primary storage, caches, queues, artifacts, search indexes, and telemetry access.

## Roadmap and dependencies

### Phase 0 — Architecture

- [Issue #20: Master Roadmap & Architecture](https://github.com/lecodev-26/sentinelflow/issues/20)
- [Issue #21: Architecture Principles & Non-Goals](https://github.com/lecodev-26/sentinelflow/issues/21)

Establish shared vocabulary, explicit non-goals, security boundaries, lifecycle semantics, and acceptance gates before expanding implementation.

### Phase 1 — Foundation

1. [Issue #22: Durable Distributed Runtime](https://github.com/lecodev-26/sentinelflow/issues/22)
2. [Issue #23: Secure Tool Platform](https://github.com/lecodev-26/sentinelflow/issues/23) — depends on the durable runtime.
3. [Issue #24: Durable Agent Runtime](https://github.com/lecodev-26/sentinelflow/issues/24) — depends on the runtime and secure tool platform.

### Phase 2 — Intelligence

4. [Issue #25: Memory & Knowledge Fabric](https://github.com/lecodev-26/sentinelflow/issues/25) — depends on durable agent execution.
5. [Issue #26: Multi-Agent Coordination & Context Engineering](https://github.com/lecodev-26/sentinelflow/issues/26) — depends on agent runtime and memory.
6. [Issue #27: Adaptive Intelligence & Evaluation](https://github.com/lecodev-26/sentinelflow/issues/27) — depends on agent runtime, coordination, and FinOps/observability foundations.

### Phase 3 — Platform

7. [Issue #28: FinOps & Observability 3.0](https://github.com/lecodev-26/sentinelflow/issues/28) — cross-cutting from the foundation onward.
8. [Issue #29: Multimodal Intelligence](https://github.com/lecodev-26/sentinelflow/issues/29) — depends on durable runtime and agent runtime.
9. [Issue #30: Developer Platform & Extensions](https://github.com/lecodev-26/sentinelflow/issues/30) — depends on stable core contracts, tool platform, and agent runtime.

### Phase 4 — Production

10. [Issue #31: Global Runtime, Self-Healing & Production GA](https://github.com/lecodev-26/sentinelflow/issues/31) — depends on validated foundation, intelligence, platform, security, governance, observability, and recovery contracts.

Security, tenant isolation, governance, auditability, and observability are cross-cutting requirements in every phase, not features deferred to the end.

## Definition of Done for V6

V6 is complete only when the supported deployment model demonstrates all of the following with documented evidence:

- Durable run/task state survives process and worker failures; recovery and retry behavior is tested.
- Idempotency, duplicate delivery, stale worker, cancellation, timeout, and retry exhaustion have deterministic, documented outcomes.
- Authorization, tenant isolation, tool permissions, approval gates, and budget limits are enforced and covered by negative tests.
- Consequential actions and artifacts are attributable to a run, identity, policy decision, and provenance chain.
- Operational telemetry supports debugging and cost/reliability analysis without leaking secrets or cross-tenant data.
- APIs, event schemas, and persistence contracts have explicit versioning and migration expectations.
- Security scanning, unit/integration/race tests where supported, load/recovery tests, and deployment/rollback procedures pass their release gates.
- Documentation distinguishes implemented, experimental, and planned capabilities; GA is not declared from roadmap completion alone.

## Non-goals

- Unbounded autonomous execution or implicit privilege escalation.
- Autonomous production mutation without explicit policy, approval where required, and rollback/reconciliation controls.
- Replacing established cryptographic primitives with custom cryptography.
- Claiming exactly-once external side effects where the external system cannot guarantee them.
- Prematurely splitting every logical plane into a separate microservice.
- Presenting unverified capabilities as production-ready.
