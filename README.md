<div align="center">

# 🛡️ SentinelFlow

**Firewall de Resiliencia para Agentes IA**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=for-the-badge)](http://makeapullrequest.com)
[![Stars](https://img.shields.io/github/stars/lecodev-26/sentinelflow?style=for-the-badge&color=gold)](https://github.com/lecodev-26/sentinelflow/stargazers)

</div>

---

## 📖 ¿Qué es SentinelFlow?

**SentinelFlow** es un proxy inteligente que protege tus agentes IA contra fallos de infraestructura. Actúa como un **escudo** entre tu aplicación y los proveedores de IA, garantizando disponibilidad y reduciendo costes.

### 🎯 ¿Por qué lo necesitas?

- 🔴 **OpenAI falló** → SentinelFlow cambia automáticamente a Anthropic
- 🔴 **Anthropic está lento** → SentinelFlow usa Local Llama
- 🔴 **Muchas peticiones** → SentinelFlow cachea respuestas y ahorra dinero

---

## ✨ Características

| Característica | Descripción |
|----------------|-------------|
| ⚡ **Failover Automático** | Si un proveedor falla, cambia al siguiente sin intervención |
| 🎯 **Smart Routing** | Cada modelo se enruta al mejor proveedor (GPT → OpenAI, Claude → Anthropic) |
| 🚦 **Rate Limiting** | Protege contra abusos: 100 peticiones/min por IP |
| 💾 **Caché Inteligente** | Respuestas guardadas en caché para reducir costes y latencia |
| 📊 **Dashboard Real-time** | Monitoriza todo desde un panel visual con gráficos |
| 📈 **Métricas Prometheus** | Exporta métricas para integrar con tus sistemas de monitoreo |

---

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

## 🌐 Accede a:
Servicio	URL

```text
Proxy	http://localhost:8080
Dashboard	http://localhost:8080/dashboard
Métricas	http://localhost:9090/metrics
Health Check	http://localhost:8080/health
```

## 🔧 Configuración
Edita configs/rules.yaml:

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

rules:
  - path: "/v1/chat/completions"
    method: "POST"
    cache: true
    providers: ["openai", "anthropic", "local-llama"]
```
## 🔑 Variables de entorno
```bash
export OPENAI_API_KEY="sk-tu-key-aqui"
export ANTHROPIC_API_KEY="ant-tu-key-aqui"
```
## 🏗️ Arquitectura
```text
┌─────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Agente IA │────▶│   SentinelFlow  │────▶│  OpenAI API     │
│   (Cliente) │     │   (Proxy)       │     │  Anthropic API  │
└─────────────┘     │                 │     │  Local Llama    │
                    └────────┬────────┘     └─────────────────┘
                             │
                    ┌────────▼────────┐
                    │   Dashboard     │
                    │   Prometheus    │
                    └─────────────────┘
```
## 🛠️ Tecnologías
- Tecnología	Uso
- Go 1.21	Lenguaje principal
- Gorilla Mux	Router HTTP
- Prometheus	Métricas y monitoreo
- Logrus	Logs estructurados
- Chart.js	Gráficos en el dashboard
- YAML	Configuración

## 📋 Roadmap
```text
Estado	Funcionalidad
✅	Proxy con failover automático
✅	Dashboard en tiempo real
✅	Métricas Prometheus
✅	Rate Limiting (100 req/min)
✅	Smart Routing por modelo
✅	Caché en memoria
✅	Logs estructurados
🔜	Semantic Cache (embeddings)
🔜	Autenticación JWT
🔜	Health Checks activos
```

## 🤝 Contribuciones
¡Las contribuciones son bienvenidas!

Fork el repositorio

Crea una rama: git checkout -b feature/nueva-funcionalidad

Haz commit: git commit -m "Añadir nueva funcionalidad"

Push: git push origin feature/nueva-funcionalidad

Abre un Pull Request

## 📄 Licencia
MIT License - ver LICENSE para más detalles.

<div align="center">
⭐ ¡Si te ha sido útil, dale una estrella! ⭐

https://img.shields.io/badge/Twitter-1DA1F2?style=for-the-badge&logo=twitter&logoColor=white
https://img.shields.io/badge/LinkedIn-0A66C2?style=for-the-badge&logo=linkedin&logoColor=white

</div> 
