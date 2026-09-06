# 🛡️ SentinelFlow - Firewall de Resiliencia para Agentes IA

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

**SentinelFlow** es un proxy inteligente que protege tus agentes IA contra fallos de infraestructura.

## ✨ Características

- ✅ **Failover automático** entre proveedores (OpenAI → Anthropic → Local)
- ✅ **Caché inteligente** para reducir costes y latencia
- ✅ **Rate Limiting** (100 req/min por IP)
- ✅ **Smart Routing** (elige el mejor proveedor según el modelo)
- ✅ **Dashboard en tiempo real** con gráficos y logs
- ✅ **Métricas Prometheus** para monitoreo
- ✅ **Logs estructurados** (JSON o texto)
- ✅ **Configuración YAML** fácil de modificar

## 🚀 Inicio rápido

```bash
# Clonar el repositorio
git clone https://github.com/lecodev-26/sentinelflow.git
cd sentinelflow

# Instalar dependencias
go mod tidy

# Ejecutar
go run cmd/proxy/main.go
```
## 📊 Dashboard
Abre tu navegador en: http://localhost:8080/dashboard

## 📈 Métricas
Métricas Prometheus disponibles en: http://localhost:9090/metrics

## 🔧 Configuración
Edita configs/rules.yaml para personalizar:
```yaml
providers:
  - name: openai
    url: "https://api.openai.com/v1"
    fallback: anthropic
    
  - name: anthropic
    url: "https://api.anthropic.com/v1"
    fallback: local-llama
    
  - name: local-llama
    url: "http://localhost:11434/api"
    fallback: ""

rules:
  - path: "/v1/chat/completions"
    method: "POST"
    cache: true
    providers: ["openai", "anthropic", "local-llama"]
```
## 🔑 Variables de entorno
```bash
export OPENAI_API_KEY="sk-tu-key"
export ANTHROPIC_API_KEY="ant-tu-key"
```
## 🏗️ Arquitectura
[Agente IA] → [SentinelFlow] → [Proveedores (OpenAI, Anthropic, Local)]
                    ↓
              [Dashboard Web]
                    ↓
              [Métricas Prometheus]
              
## 🛠️ Tecnologías
Go 1.21 - Lenguaje principal

Gorilla Mux - Router HTTP

Prometheus - Métricas

Logrus - Logs estructurados

YAML - Configuración

Chart.js - Dashboard

## 📋 Roadmap
☑ Proxy con failover
☑ Dashboard en tiempo real
☑ Métricas Prometheus
☑ Rate Limiting
☑ Smart Routing
□ Semantic Cache (próximo)
□ Autenticación JWT (próximo)

## 🤝 Contribuciones
¡Las contribuciones son bienvenidas! Abre un issue o PR.

## 📄 Licencia
MIT License - ver LICENSE para más detalles.

## ⭐ ¡Dános una estrella!
Si este proyecto te ha sido útil, ¡dale una estrella en GitHub!