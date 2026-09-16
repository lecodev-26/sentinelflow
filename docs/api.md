# SentinelFlow - API Reference

Base URL: `http://localhost:8081/v1`

## Health

### GET /health

Returns service health.

```json
{
  "status": "ok",
  "service": "sentinelflow-control-plane",
  "version": "1.0.0"
}
```

Organizations

GET /organizations

List all organizations.

POST /organizations

Create a new organization.

```json
{
  "name": "Acme Corp",
  "description": "Production tenant"
}
```

GET /organizations/{id}

Get organization details.

POST /organizations/{id}/projects

Create a project.

```json
{
  "name": "Production",
  "description": "Production project"
}
```

Users

POST /users

Create a user.

```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "role": "admin",
  "org_id": "org_xxx"
}
```

GET /users/{id}

Get user details.

POST /users/{id}/api-keys

Create an API key. Returns the key ONCE.

```json
{
  "name": "production-key",
  "project_id": "proj_xxx"
}
```

Response:

```json
{
  "api_key": {...},
  "key": "sf_xxxxx",
  "warning": "Save this key securely, it won't be shown again."
}
```

POST /api-keys/{key}/revoke

Revoke an API key.

Providers

GET /providers

List all providers with real health status.

```json
{
  "count": 3,
  "providers": [
    {
      "name": "openai",
      "status": {
        "status": "healthy",
        "latency_ms": 120,
        "consecutive_fails": 0
      },
      "circuit_breaker": "closed"
    }
  ]
}
```

GET /circuit-breakers

Get circuit breaker states.

GET /budgets

List all tenant budgets.

Metrics

GET /metrics/overview

```json
{
  "uptime_seconds": 3600,
  "requests": 1820000,
  "cost_usd": 483.20,
  "latency_p95_ms": 420,
  "error_rate": 0.18,
  "cache_hit_rate": 0.42,
  "trends": {
    "requests": 12.4,
    "cost": 8.1,
    "latency": -5.2,
    "error_rate": -2.1
  }
}
```

GET /metrics/providers

Provider status with real-time health.

GET /metrics/security

Security metrics:

```json
{
  "pii_blocked": 142,
  "secrets_blocked": 18,
  "injection_blocked": 39,
  "requests_blocked": 7
}
```

GET /metrics/system

System metrics:

```json
{
  "goroutines": 16,
  "heap_alloc_mb": 1.99,
  "heap_sys_mb": 7.28,
  "gc_count": 3,
  "go_version": "go1.27.1",
  "num_cpu": 8
}
```

GET /metrics/timeseries

Time series data for charts.

Errors

All errors follow this format:

```json
{
  "error": {
    "type": "invalid_request",
    "message": "model is required",
    "details": null
  }
}
```

Error types:

· authentication_error (401)
· authorization_error (403)
· policy_denied (403)
· rate_limited (429)
· quota_exceeded (402)
· provider_unavailable (503)
· provider_timeout (504)
· provider_error (502)
· invalid_request (400)
· internal_error (500)
