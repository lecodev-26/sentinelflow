# 🛡️ SentinelFlow - Firewall de Resiliencia para Agentes IA

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

**SentinelFlow** es un proxy inteligente que protege tus agentes IA contra fallos de infraestructura, proporcionando:

- ✅ **Failover automático** entre proveedores (OpenAI → Anthropic → Local)
- ✅ **Caché inteligente** para reducir costes y latencia
- ✅ **Métricas en tiempo real** para monitorizar la salud
- ✅ **Logs estructurados** para depurar problemas

## 🚀 Inicio rápido

```bash
# Clonar el repositorio
git clone https://github.com/tu-usuario/sentinelflow.git
cd sentinelflow

# Instalar dependencias
make deps

# Ejecutar el proxy
make run

# Probar que funciona
curl http://localhost:8080/health
```

📋 Configuración

Edita configs/rules.yaml para definir tus proveedores y reglas:

```yaml
providers:
  - name: openai
    url: "https://api.openai.com/v1"
    fallback: anthropic
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
