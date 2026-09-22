# 🛡️ SentinelFlow V3

**AI Gateway & Model Control Plane**

[![Go Version](https://img.shields.io/badge/Go-1.27-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/badge/version-3.0.0-blue?style=for-the-badge)](https://github.com/lecodev-26/sentinelflow)

## 📖 ¿Qué es SentinelFlow?

SentinelFlow es un **AI Gateway + Model Control Plane** de nivel empresarial.

### Data Plane
- ✅ **Routing con scoring** (latencia 0.35 + coste 0.25 + health 0.25 + quality 0.15)
- ✅ **Circuit breaker** por provider con auto-recovery
- ✅ **Streaming SSE real** end-to-end
- ✅ **Idempotency** con `Idempotency-Key`
- ✅ **Multi-provider**: OpenAI, Anthropic, Ollama

### Control Plane
- ✅ **PostgreSQL (Supabase)** como source of truth
- ✅ **Multi-tenant** con API keys + scopes
- ✅ **Policy Engine** versionado
- ✅ **OIDC + SCIM 2.0**
- ✅ **Regional routing** + GDPR

### Seguridad
- ✅ **PII detection**: email, phone, credit card, SSN, IPv4
- ✅ **Secret detection**: OpenAI, Anthropic, AWS, GitHub, keys
- ✅ **Prompt injection**: jailbreak, override, reveal
- ✅ **SSRF protection**: localhost, private IPs, cloud metadata
- ✅ **Auto-redaction** de PII

### FinOps
- ✅ **Usage tracking** con tokens reales
- ✅ **Budgets** por tenant con forecasts
- ✅ **Alertas** 80/90/100%

### Observability
- ✅ **Trace Store** con spans por etapa
- ✅ **Request timeline** (`/v1/traces/{id}`)
- ✅ **Structured logs** con request_id + trace_id

### HA/DR
- ✅ **Liveness + Readiness** (`/livez`, `/readyz`)
- ✅ **Graceful shutdown** con drain
- ✅ **Backup automático** + DR tests

## 🚀 Quick Start

```bash
git clone https://github.com/lecodev-26/sentinelflow.git
cd sentinelflow
go mod download

cat > .env << 'EOF'
SENTINELFLOW_DATABASE_URL=postgresql://...
SENTINELFLOW_ENV=development
SENTINELFLOW_VAULT_KEY=your-vault-key
EOF

# Gateway (puerto 8080)
go run cmd/gateway/main.go

# Control Plane (puerto 8081)
go run cmd/controlplane/main.go
```

📡 Endpoints

Gateway (8080)

Método Endpoint Descripción
GET /livez Liveness probe
GET /readyz Readiness probe
GET /health Health completo
GET /v1/providers Estado de providers
GET /v1/models Catálogo con pricing
GET /v1/traces Últimos traces
GET /v1/traces/{id} Trace completo
GET /v1/usage/stats Estadísticas de uso
POST /v1/chat/completions Chat con routing inteligente

Control Plane (8081)

Método Endpoint Descripción
GET/POST /v1/organizations Organizaciones
GET/POST /v1/projects Proyectos
GET/POST /v1/users Usuarios
POST /v1/users/{id}/api-keys API keys
GET /auth/oidc/providers OIDC providers
GET /auth/oidc/{provider}/login OIDC login
GET/POST /scim/v2/Users SCIM 2.0

🏗️ Arquitectura

```
              Clients
                 │
                 ▼
        ┌────────────────┐
        │    GATEWAY     │
        │  (stateless)   │
        │                │
        │ auth → trace   │
        │ policy → sec   │
        │ routing → acc  │
        └────────┬───────┘
                 │
    ┌────────────┼────────────┐
    ▼            ▼            ▼
 OpenAI      Anthropic     Ollama
    │            │            │
    └────────────┼────────────┘
                 │
                 ▼
        ┌────────────────┐
        │   PostgreSQL   │
        │   (Supabase)   │
        │ Source of Truth│
        └────────────────┘
```

📊 Milestones

Versión Highlights
v3.0.0 Control Plane, Routing, FinOps, Enterprise, HA/DR
v2.10.0 Production ready
v1.0.0 Gateway con failover

📄 License

MIT — ver LICENSE.

⭐ ¿Te gusta?

¡Dale una estrella en GitHub!
