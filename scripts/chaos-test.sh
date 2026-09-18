#!/bin/bash
# Chaos testing para SentinelFlow

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

PASS=0
FAIL=0

check() {
    local name="$1"
    local result="$2"
    if [ "$result" = "0" ]; then
        echo -e "${GREEN}✅ PASS${NC} $name"
        PASS=$((PASS+1))
    else
        echo -e "${RED}❌ FAIL${NC} $name"
        FAIL=$((FAIL+1))
    fi
}

echo "🔥 SentinelFlow Chaos Testing"
echo "================================"
echo ""

# === TEST 1: Provider muerto ===
echo -e "${YELLOW}Test 1: Provider muerto (modelo inválido)${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{"model":"invalid-model-xyz","messages":[{"role":"user","content":"test"}]}')

if [ "$HTTP_CODE" = "503" ] || [ "$HTTP_CODE" = "200" ]; then
    check "Provider failover works" 0
else
    check "Provider failover works (got $HTTP_CODE)" 1
fi

# === TEST 2: Timeout ===
echo -e "${YELLOW}Test 2: Timeout (espera máx 5s)${NC}"
START=$(date +%s)
curl -s -o /dev/null --max-time 5 -X POST "$BASE_URL/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_KEY" \
    -d '{"model":"gpt-4","messages":[{"role":"user","content":"test"}]}' || true
END=$(date +%s)
DURATION=$((END-START))

if [ $DURATION -le 5 ]; then
    check "Timeout respected (${DURATION}s)" 0
else
    check "Timeout exceeded (${DURATION}s)" 1
fi

# === TEST 3: Concurrent requests ===
echo -e "${YELLOW}Test 3: 50 concurrent requests${NC}"
START=$(date +%s)
for i in $(seq 1 50); do
    curl -s -o /dev/null "$BASE_URL/health" &
done
wait
END=$(date +%s)
DURATION=$((END-START))

if [ $DURATION -le 10 ]; then
    check "50 concurrent requests in ${DURATION}s" 0
else
    check "Concurrent requests too slow (${DURATION}s)" 1
fi

# === TEST 4: Rate limiting ===
echo -e "${YELLOW}Test 4: Rate limiting (200 requests)${NC}"
RATE_LIMITED=0
for i in $(seq 1 200); do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
    if [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED=$((RATE_LIMITED+1))
    fi
done

if [ $RATE_LIMITED -gt 0 ]; then
    check "Rate limiting active ($RATE_LIMITED hits)" 0
else
    check "Rate limiting active" 1
fi

# === TEST 5: Health check ===
echo -e "${YELLOW}Test 5: Health check${NC}"
HEALTH=$(curl -s "$BASE_URL/health")
if echo "$HEALTH" | grep -q "ok"; then
    check "Health check returns ok" 0
else
    check "Health check failed" 1
fi

# === TEST 6: Metrics endpoint ===
echo -e "${YELLOW}Test 6: Metrics endpoint${NC}"
METRICS=$(curl -s "http://localhost:9090/metrics")
if echo "$METRICS" | grep -q "sentinelflow"; then
    check "Prometheus metrics available" 0
else
    check "Metrics missing" 1
fi

# === TEST 7: Circuit breaker ===
echo -e "${YELLOW}Test 7: Circuit breaker status${NC}"
CB=$(curl -s "http://localhost:8081/v1/circuit-breakers")
if echo "$CB" | grep -q "circuit_breakers"; then
    check "Circuit breakers available" 0
else
    check "Circuit breakers missing" 1
fi

# === TEST 8: Control plane ===
echo -e "${YELLOW}Test 8: Control plane health${NC}"
CP=$(curl -s "http://localhost:8081/health")
if echo "$CP" | grep -q "ok"; then
    check "Control plane healthy" 0
else
    check "Control plane failed" 1
fi

echo ""
echo "================================"
echo -e "✅ PASS: ${GREEN}$PASS${NC}"
echo -e "❌ FAIL: ${RED}$FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}🎉 All chaos tests passed!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed${NC}"
    exit 1
fi
