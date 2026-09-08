<div align="center">

# 🛡️ SentinelFlow

**AI Gateway & Control Plane para aplicaciones multi-LLM**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=for-the-badge)](http://makeapullrequest.com)
[![Stars](https://img.shields.io/github/stars/lecodev-26/sentinelflow?style=for-the-badge&color=gold)](https://github.com/lecodev-26/sentinelflow/stargazers)

</div>

---

## 📖 ¿Qué es SentinelFlow?

**SentinelFlow** es un **AI Gateway** que actúa como capa de control para aplicaciones que consumen LLMs. Pones SentinelFlow entre tu aplicación y los proveedores de IA (OpenAI, Anthropic, Llama, etc.) y él decide a qué proveedor enviar cada petición, hace failover si uno falla, aplica rate limiting, cachea respuestas y expone métricas en tiempo real.

### 🎯 ¿Por qué lo necesitas?

| Problema | Solución |
|----------|----------|
| 🔴 **OpenAI falló** | ✅ SentinelFlow cambia automáticamente a Anthropic |
| 🔴 **Anthropic está lento** | ✅ SentinelFlow usa Local Llama |
| 🔴 **Muchas peticiones** | ✅ SentinelFlow cachea respuestas y ahorra dinero |
| 🔴 **Costes descontrolados** | ✅ SentinelFlow trackea costes y aplica budgets |
| 🔴 **Fugas de datos** | ✅ SentinelFlow detecta PII y secretos |

---

## ✨ Características

| Característica | Descripción |
|----------------|-------------|
| ⚡ **Failover Automático** | Si un proveedor falla, cambia al siguiente sin intervención |
| 🎯 **Smart Routing** | Elige el mejor proveedor por coste, latencia y salud |
| 📊 **Cost Intelligence** | Trackeo de costes, budgets y alertas |
| 🛡️ **Seguridad** | PII detection, secret detection, prompt injection |
| 📝 **Audit Logs** | Registro completo de todas las peticiones |
| 💾 **Caché Semántica** | Cachea respuestas por similitud (embeddings) |
| 🔐 **Multi-tenancy** | Organizaciones, proyectos y RBAC |
| 📈 **Observabilidad** | OpenTelemetry, métricas, TTFT, tracing |
| 🚀 **Escalabilidad** | Kubernetes, Helm, HPA |

---

## 🚀 Inicio rápido

### Con Go

```bash
# Clonar
git clone https://github.com/lecodev-26/sentinelflow.git
cd sentinelflow

# Instalar dependencias
go mod tidy

# Ejecutar
make run
```

Con Docker

```bash
docker run -p 8080:8080 sentinelflow:latest
```

Con Kubernetes

```bash
helm install sentinelflow ./deployments/helm/sentinelflow
```

Con Terraform

```bash
cd deployments/terraform
terraform apply
```

---

🌐 Accede a los servicios

Servicio URL
Proxy http://localhost:8080
Dashboard http://localhost:8080/dashboard
Demo http://localhost:8080/demo
Métricas http://localhost:9090/metrics
Health Check http://localhost:8080/health

---

🔧 Configuración

Edita configs/rules.yaml para personalizar:

```yaml
providers:
  - name: openai
    url: "https://api.openai.com/v1"
    fallback: anthropic
    headers:
      Authorization: "Bearer ${OPENAI_API_KEY}"

  - name: anthropic
    url: "https://api.anthropic.com/v1"
    fallback: local-llama
    headers:
      x-api-key: "${ANTHROPIC_API_KEY}"

  - name: local-llama
    url: "http://localhost:11434/api"
    fallback: ""

rules:
  - path: "/v1/chat/completions"
    method: "POST"
    cache: true
    providers: ["openai", "anthropic", "local-llama"]
```

🔑 Variables de entorno

```bash
export OPENAI_API_KEY="sk-tu-key-aqui"
export ANTHROPIC_API_KEY="ant-tu-key-aqui"
```

---

🏗️ Arquitectura

```marckdown
┌─────────────┐     ┌─────────────────────────────────────────────────────┐
│   Clientes  │────▶│                  SentinelFlow                       │
│   Agents    │     │                                                     │
│   Apps      │     │  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│   SDKs      │     │  │Security │  │ Router  │  │Policies │            │
└─────────────┘     │  └────┬────┘  └────┬────┘  └────┬────┘            │
                    │       │            │            │                   │
                    │       └────────────┼────────────┘                   │
                    │                    ▼                                │
                    │           ┌────────────────┐                       │
                    │           │ Cost Intelligence│                      │
                    │           └────────────────┘                       │
                    │                    │                                │
                    │                    ▼                                │
                    │           ┌────────────────┐                       │
                    │           │ Observability  │                       │
                    │           └────────────────┘                       │
                    └─────────────────────────────────────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
             ┌───────────┐      ┌───────────┐      ┌───────────┐
             │  OpenAI   │      │ Anthropic │      │ Local     │
             └───────────┘      └───────────┘      └───────────┘
```

---

🛠️ Tecnologías

Tecnología Uso
Go 1.21 Lenguaje principal
Gorilla Mux Router HTTP
Prometheus Métricas y monitoreo
OpenTelemetry Tracing distribuido
Redis Caché distribuida
Kubernetes Orquestación
Helm Despliegue
Terraform Infraestructura como código

---

📋 Roadmap

Estado Funcionalidad
✅ Proxy con failover automático
✅ Dashboard en tiempo real
✅ Métricas Prometheus
✅ Rate Limiting
✅ Smart Routing (coste, latencia, salud)
✅ Caché en memoria y distribuida (Redis)
✅ Logs estructurados
✅ OpenTelemetry tracing
✅ PII / Secret Detection
✅ Prompt Injection Detection
✅ Audit Logs
✅ Policy Engine
✅ Multi-tenancy + RBAC
✅ Kubernetes + Helm + Terraform
✅ Cost Tracking + Budgets
✅ Semantic Cache
✅ Demo interactiva

---

🤝 Contribuciones

¡Las contribuciones son bienvenidas!

1. Fork el repositorio
2. Crea una rama: git checkout -b feature/nueva-funcionalidad
3. Haz commit: git commit -m "Añadir nueva funcionalidad"
4. Push: git push origin feature/nueva-funcionalidad
5. Abre un Pull Request

---

📄 Licencia

MIT License - ver LICENSE para más detalles.

---

<div align="center">

⭐ ¡Si te ha sido útil, dale una estrella! ⭐

https://img.shields.io/badge/GitHub-181717?style=for-the-badge&logo=github&logoColor=white
https://img.shields.io/badge/Twitter-1DA1F2?style=for-the-badge&logo=twitter&logoColor=white

</div>
