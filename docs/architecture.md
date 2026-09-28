# Architecture

SentinelFlow is an AI gateway and control plane organized into five planes.

## Control Plane

Identity, organizations, business units, projects, environments, API keys, policies, models, providers, budgets, secrets, approvals and audit configuration.

## Data Plane

Requests enter through the gateway and pass authentication, tenant/project resolution, authorization, quota/rate limiting, policy/security checks, idempotency/cache, routing, provider execution, fallback, streaming and accounting.

## Event Plane

Business changes and usage events use the transactional outbox pattern. Consumers are idempotent and designed for at-least-once delivery.

## Provider Plane

Provider adapters expose normalized capabilities, model metadata, health and execution contracts. Credential pools, circuit breakers, SLOs and regional placement are handled above individual adapters.

## Observability Plane

Metrics, traces, audit/security events, provider SLOs and error budgets make the request path measurable.

## Tenant boundary

Tenant and environment are derived from authenticated context, not trusted request fields. Cache keys, usage, analytics, audit and security records preserve tenant/environment separation.

## Failure model

Routing selection is distinct from execution fallback. Pre-stream failures may retry/fail over when policy permits; post-stream failures are recorded as partial responses. Events use event IDs and idempotent consumers rather than relying on exactly-once delivery.

See the ADRs under `docs/v3/adr` and the V4 production notes for historical decisions and current direction.
