# SentinelFlow V6 — Architecture Principles & Non-Goals

**Status:** normative design baseline for V6 development. These rules constrain implementation and review; this document is not a claim that every listed control is already implemented.

## Purpose

This document turns the principles in the [V6 master architecture](ARCHITECTURE.md) into reviewable engineering rules. New V6 components, APIs, persisted data, events, tool integrations, and autonomous workflows must follow these rules or document an explicit, reviewed exception before implementation.

## Principles

### P1. Durable execution over best effort
- Persist accepted work and lifecycle transitions before acknowledging durable acceptance.
- Define recovery behavior for process restarts, worker loss, redelivery, and partial failure.
- Keep authoritative state distinct from caches and derived projections.

### P2. Explicit identity and authority
- Every request and consequential action has an authenticated principal, tenant scope, and correlation identity.
- Delegation passes only explicitly granted capabilities; agents and tools cannot self-elevate.
- Credentials are scoped, short-lived where supported, and never treated as user-controlled input.

### P3. Policy enforcement at every privileged boundary
- Evaluate authorization and applicable policy before each privileged tool/provider operation and other consequential side effect.
- Include risk, budget, approval, and environment constraints where applicable.
- Fail closed when a required policy decision or approval cannot be established.

### P4. Tenant isolation by design
- Enforce tenant scope in database access, cache keys, queues, events, object/artifact storage, retrieval indexes, logs, traces, and administrative APIs.
- Never treat a tenant ID supplied by an untrusted caller as proof of membership or authorization.
- Include negative tests for cross-tenant reads, writes, references, and metadata leakage.

### P5. Idempotent and recoverable operations
- Give runs, tasks, events, and external operations stable identifiers where appropriate.
- Specify retry, deduplication, lease expiry, stale-worker, cancellation, and reconciliation semantics.
- Do not promise exactly-once external effects unless the external system provides a verifiable guarantee.

### P6. Auditability and provenance
- Make consequential actions attributable to the principal, run/task, capability, policy version/decision, and result.
- Preserve enough ordered, correlated evidence to investigate failures and reconstruct the execution path.
- Redact secrets and sensitive payloads from ordinary logs, traces, and error responses.

### P7. Bounded execution
- Define and enforce relevant limits for time, retries, concurrency, payload size, recursion/delegation depth, and cost/resource budgets.
- Cancellation and limit exhaustion must lead to observable, deterministic outcomes.
- Do not rely solely on model instructions to enforce security or resource boundaries.

### P8. Observable outcomes
- Correlate logs, metrics, traces, audit events, latency, reliability, quality, and cost across a run.
- Keep telemetry tenant-scoped and apply retention/access controls.
- Distinguish missing telemetry from a successful or zero-cost outcome.

### P9. Contract and schema discipline
- Preserve stable V5 behavior where practical; do not silently change public contracts.
- Version public APIs and persisted/event schemas deliberately, and document compatibility and migration expectations.
- Prefer explicit interfaces and modular boundaries over premature service decomposition.

### P10. Evidence before release claims
- Label capabilities as planned, experimental, implemented, or generally available accurately.
- Release claims require reproducible test, security, recovery, operational, and documentation evidence.
- Roadmap completion or a successful build alone is not proof of production readiness.

## Non-goals

V6 will not:
- enable unbounded autonomous execution without policy, budgets, and stopping conditions;
- permit agents, prompts, or tools to bypass authorization or silently gain privileges;
- allow autonomous production mutation without explicit governance and required approvals;
- replace established cryptographic primitives with home-grown cryptography;
- claim exactly-once delivery or side effects where the underlying systems cannot guarantee them;
- split every logical component into a separate microservice before operational need is demonstrated;
- treat model output, retrieved content, tool results, or user-provided files as trusted instructions;
- expose secrets in model context, artifacts, logs, traces, or error messages by default;
- describe a planned or partially verified capability as production-ready.

## Required review questions

Before accepting a V6 change, reviewers should be able to answer:

1. What identity and tenant scope authorizes this operation?
2. Which policy, permission, budget, or approval checks run immediately before consequential effects?
3. What happens after a crash, timeout, duplicate delivery, cancellation, or stale worker?
4. How are retries made safe, and which external effects require deduplication or reconciliation?
5. What evidence links the action and its artifacts to the initiating run, principal, and policy decision?
6. Are limits enforced in code at the relevant boundary, rather than only described in prompts or docs?
7. Can any request, cache, event, artifact, or telemetry path cross tenant boundaries?
8. Are public and persisted contracts versioned, and are migration/rollback implications understood?
9. What tests demonstrate both allowed behavior and denied/failed behavior?
10. Do documentation and release claims match the implementation and evidence?

If a question is not applicable, the change description should say why. Exceptions must identify the affected boundary, risk, compensating control, owner, and expiry/revisit condition; an undocumented exception is not acceptable.

## Relationship to the roadmap

This document addresses [Issue #21 — Architecture Principles & Non-Goals](https://github.com/lecodev-26/sentinelflow/issues/21). It complements, rather than replaces, the [master roadmap and architecture](ARCHITECTURE.md). Implementation issues must apply these principles and provide evidence in their own acceptance criteria.

## Acceptance criteria

- [x] Principles and non-goals are written as explicit engineering constraints.
- [x] Security, authorization, tenant isolation, durability, idempotency, audit, observability, and bounded execution are covered.
- [x] A repeatable review checklist is provided for future V6 changes.
- [x] The distinction between intended architecture and implemented capability is explicit.
