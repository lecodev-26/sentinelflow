# API Reference

SentinelFlow exposes two API surfaces: the **Gateway** for model traffic and the **Control Plane** for administration.

> Endpoint availability and response fields are defined by the current code. This document intentionally describes the stable surface without inventing undocumented fields.

## Gateway

Typical local address: `http://localhost:8080`.

### OpenAI-compatible model APIs

- `POST /v1/chat/completions` — chat completions.
- `POST /v1/responses` — normalized Responses-style requests.
- `POST /v1/embeddings` — embeddings.
- `GET /v1/models` — model catalog.

Gateway requests use the authenticated API key to derive tenant/project/environment context. Do not send tenant identity as an untrusted substitute for authentication.

### Operational endpoints

- `/health`, `/livez`, `/readyz` — health/readiness where enabled by the service.
- `/metrics` — Prometheus metrics.

## Control Plane

Typical local address: `http://localhost:8081`.

The control plane manages organizations, projects, users, API keys, providers, budgets, policies, approvals, environments, regions, business units, audit, analytics, security events, webhooks and enterprise identity integrations.

Representative endpoints include:

| Area | Endpoints |
|---|---|
| Organizations | `GET/POST /organizations`, `GET /organizations/{id}` |
| Business units | `GET/POST /organizations/{id}/business-units`, `GET/DELETE /business-units/{id}` |
| Projects | project endpoints under the organization hierarchy |
| API keys | API-key creation/revocation endpoints |
| Approvals | approval request create/list/approve/reject/cancel endpoints |
| Regions | `GET /v1/regions` |
| Audit | `GET /v1/audit` |
| Analytics | `GET /v1/analytics/daily` |
| Security | `GET /v1/security/events`, `GET /v1/security/anomalies` |
| Webhooks | `/v1/webhooks` CRUD and delivery history |
| SAML | `/auth/saml/metadata`, `/auth/saml/login`, `/auth/saml/acs` |
| SCIM | `/scim/v2/Groups` and group member operations |

Authentication and scopes are enforced by the service; exact route prefixes can vary between gateway and control-plane binaries. Inspect the handlers in `cmd/` and `internal/` when integrating against a pinned commit.

## Authentication

Gateway authentication uses SentinelFlow API keys. Production deployments fail closed when authentication configuration is missing or invalid.

Never put provider credentials in client requests. Provider credentials belong to SentinelFlow's provider/secret configuration.

## Idempotency

Where supported, clients can send `Idempotency-Key` to make retry behavior deterministic. Distributed idempotency state is designed for multi-instance deployments.

## Streaming

Streaming endpoints expose provider output incrementally. The gateway records stream start/chunk/completion/failure telemetry and distinguishes failures that occur before and after streaming begins.

## Errors

Clients should treat HTTP status and the response body as the source of truth. Common classes include authentication (`401`), authorization/policy (`403`), rate/quota (`429`/quota-specific), provider failures (`5xx`) and invalid requests (`400`).

## Versioning

Pin client integrations to a release/commit when strict compatibility matters. Public API changes should include tests and documentation updates.
