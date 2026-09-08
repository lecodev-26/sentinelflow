# SentinelFlow - Documentación

## Índice

1. [Introducción](#introducción)
2. [Instalación](#instalación)
3. [Configuración](#configuración)
4. [Proveedores](#proveedores)
5. [Seguridad](#seguridad)
6. [Observabilidad](#observabilidad)
7. [Benchmark](#benchmark)

## Introducción

SentinelFlow es un AI Gateway que actúa como capa de control para aplicaciones multi-LLM.

### Características principales

- ✅ **Failover automático** entre proveedores
- ✅ **Smart Routing** por coste, latencia y salud
- ✅ **Seguridad** con PII detection y prompt injection
- ✅ **Observabilidad** con OpenTelemetry y métricas
- ✅ **Multi-tenancy** con RBAC y API keys
- ✅ **Caché** semántica y distribuida

## Instalación

### Con Docker

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

Configuración

El archivo configs/rules.yaml permite configurar:

```yaml
providers:
  - name: openai
    url: "https://api.openai.com/v1"
    fallback: anthropic

rules:
  - path: "/v1/chat/completions"
    method: "POST"
    cache: true
    providers: ["openai", "anthropic", "local-llama"]
```

Proveedores

Proveedor Soporte Estado
OpenAI ✅ Estable
Anthropic ✅ Estable
Local Llama ✅ Experimental

Seguridad

SentinelFlow incluye:

· PII detection
· Secret detection
· Prompt injection detection
· Policy engine
· API keys + JWT

Observabilidad

Métricas disponibles:

· TTFT (Time To First Token)
· Coste por petición
· Tokens por modelo
· Tasa de fallos
· Latencia

Benchmark

Ver benchmark.md para resultados detallados.
