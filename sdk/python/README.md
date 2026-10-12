# SentinelFlow Python SDK

> **Contributions welcome!** See the [contribution guide](../CONTRIBUTING.md) for the workflow and pull-request requirements.

```python
from sentinelflow import Client

client = Client("https://gateway.example.com", "sf_...")
result = client.chat("model-id", [{"role": "user", "content": "Hello"}])
```
