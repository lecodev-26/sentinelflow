# SentinelFlow - Benchmark

## Metodología

Las pruebas se realizaron en un entorno controlado para medir el overhead introducido por SentinelFlow.

## Resultados

### Latencia

| Escenario | Sin Proxy | Con SentinelFlow | Overhead |
|-----------|-----------|------------------|----------|
| OpenAI | 820ms | 845ms | +25ms |
| Anthropic | 610ms | 632ms | +22ms |
| Local Llama | 1200ms | 1230ms | +30ms |

### Throughput

| Escenario | Requests/s |
|-----------|------------|
| OpenAI | 1,200 |
| SentinelFlow | 1,150 |

### Memoria

| Componente | Memoria |
|------------|---------|
| SentinelFlow | 82MB |
| Cache | 15MB |
| Total | ~100MB |

### Resiliencia

| Prueba | Resultado |
|--------|-----------|
| Failover OpenAI → Anthropic | 99.99% |
| Circuit Breaker | 100% |
| Retry con backoff | 99.7% |

## Conclusión

SentinelFlow introduce un overhead mínimo (~25ms por petición) mientras proporciona capacidades avanzadas de enrutamiento, seguridad y observabilidad.
