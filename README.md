# 🛡️ SentinelFlow - Firewall de Resiliencia para Agentes IA
[![GitHub last commit](https://img.shields.io/github/last-commit/lecodev-26/sentinelflow?style=flat-square)](https://github.com/lecodev-26/sentinelflow)
[![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/lecodev-26/sentinelflow?style=flat-square)](https://github.com/lecodev-26/sentinelflow)

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

<<<<<<< HEAD
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
=======
**SentinelFlow** es un proxy inteligente que protege tus agentes IA contra fallos de infraestructura, proporcionando:

- ✅ **Failover automático** entre proveedores (OpenAI → Anthropic → Local)
- ✅ **Caché inteligente** para reducir costes y latencia
- ✅ **Métricas en tiempo real** para monitorizar la salud
- ✅ **Logs estructurados** para depurar problemas
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859

## 🚀 Inicio rápido

```bash
# Clonar el repositorio
git clone https://github.com/lecodev-26/sentinelflow.git
cd sentinelflow

# Instalar dependencias
<<<<<<< HEAD
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
=======
make deps

# Ejecutar el proxy
make run

# Probar que funciona
curl http://localhost:8080/health
```

📋 Configuración

Edita configs/rules.yaml para definir tus proveedores y reglas:

>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
```yaml
providers:
  - name: openai
    url: "https://api.openai.com/v1"
    fallback: anthropic
<<<<<<< HEAD
    
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
=======
```

🏗️ Arquitectura

```
[Agente IA] → [SentinelFlow] → [Proveedores (OpenAI, Anthropic, ...)]
                    ↓
              [Dashboard Web]
```

📊 Roadmap

☐ Carga de configuración
☐ Proxy reverso con failover
☐ Caché en memoria
☐ Métricas y dashboard
☐ Tests y documentación
☐ Dockerización
>>>>>>> b011fca1f048d402212fded110a6fddb45a10859
