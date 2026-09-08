#!/bin/bash

# Script para ejecutar benchmarks de SentinelFlow

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}📊 SentinelFlow Benchmark${NC}"
echo ""

# Variables
URL=${1:-"http://localhost:8080"}
REQUESTS=${2:-100}
CONCURRENCY=${3:-10}

echo -e "📋 Configuración:"
echo "   URL: $URL"
echo "   Requests: $REQUESTS"
echo "   Concurrency: $CONCURRENCY"
echo ""

# Comprobar si el proxy está corriendo
echo -e "${YELLOW}🔍 Comprobando proxy...${NC}"
if ! curl -s "$URL/health" > /dev/null; then
    echo -e "${RED}❌ SentinelFlow no está corriendo en $URL${NC}"
    echo "Ejecuta 'make run' antes de hacer el benchmark"
    exit 1
fi

echo -e "${GREEN}✅ Proxy OK${NC}"
echo ""

# Ejecutar benchmark
echo -e "${YELLOW}🚀 Ejecutando benchmark...${NC}"

start_time=$(date +%s%N)

# Usar curl en paralelo
for i in $(seq 1 $REQUESTS); do
    curl -s -X POST "$URL/v1/chat/completions" \
        -H "Content-Type: application/json" \
        -d '{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}' &
    
    # Limitar concurrencia
    if [ $((i % CONCURRENCY)) -eq 0 ]; then
        wait
    fi
done

wait

end_time=$(date +%s%N)

# Calcular resultados
total_time=$(( ($end_time - $start_time) / 1000000 ))
throughput=$(echo "scale=2; $REQUESTS / ($total_time / 1000)" | bc)

echo ""
echo -e "${GREEN}📊 Resultados:${NC}"
echo "   Total requests: $REQUESTS"
echo "   Concurrency: $CONCURRENCY"
echo "   Total time: ${total_time}ms"
echo "   Throughput: ${throughput} req/s"
echo ""
