# SentinelFlow - Quickstart

## Installation

### From Source

```bash
git clone https://github.com/lecodev-26/sentinelflow.git
cd sentinelflow
go mod tidy
go run cmd/proxy/main.go
```

With Docker

```bash
docker run -p 8080:8080 -p 8081:8081 -p 9090:9090 sentinelflow:latest
```

With Docker Compose (full stack)

```bash
docker-compose up -d
```

With Kubernetes

```bash
helm install sentinelflow ./deployments/helm/sentinelflow
```

Configuration

Edit configs/rules.yaml:

```yaml
providers:
  - name: openai
    url: "https://api.openai.com/v1"
    timeout: 30s
    fallback: anthropic
    headers:
      Authorization: "Bearer ${OPENAI_API_KEY}"
```

Set environment variables:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="ant-..."
```

Usage

Send a request

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

Use with OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="any-value"
)

response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[{"role": "user", "content": "Hello"}]
)
```

Access Services

Service URL
Gateway http://localhost:8080
Dashboard http://localhost:8080/dashboard
Demo http://localhost:8080/demo
Control Plane http://localhost:8081/v1
Metrics http://localhost:9090/metrics

Health Check

```bash
curl http://localhost:8080/health
```

Dashboard

Open http://localhost:8080/dashboard in your browser.

Features:

· Real-time metrics
· Provider status
· Cost tracking
· Security events
· Command palette (Ctrl+K)
· Dark/light mode
