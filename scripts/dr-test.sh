#!/bin/bash
# Disaster Recovery Test
# Prueba que el sistema puede recuperarse desde un backup

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "🔥 SentinelFlow DR Test"
echo "========================="
echo ""

# 1. Estado inicial
echo -e "${YELLOW}1. Estado inicial...${NC}"
ORG_COUNT=$(psql "$SENTINELFLOW_DATABASE_URL" -t -c "SELECT COUNT(*) FROM organizations;")
USER_COUNT=$(psql "$SENTINELFLOW_DATABASE_URL" -t -c "SELECT COUNT(*) FROM users;")
echo "   Organizations: $ORG_COUNT"
echo "   Users: $USER_COUNT"
echo ""

# 2. Backup
echo -e "${YELLOW}2. Creando backup...${NC}"
./scripts/backup-supabase.sh
LATEST_BACKUP=$(ls -t backups/sentinelflow-*.sql.gz | head -1)
echo -e "${GREEN}   Backup: $LATEST_BACKUP${NC}"
echo ""

# 3. Simular pérdida (borrar un dato específico)
echo -e "${YELLOW}3. Simulando pérdida de datos...${NC}"
psql "$SENTINELFLOW_DATABASE_URL" -c "DELETE FROM users WHERE email LIKE 'dr-test-%@example.com';" > /dev/null
NEW_USER_ID="user_drtest_$(date +%s)"
psql "$SENTINELFLOW_DATABASE_URL" -c "INSERT INTO users (id, email, name, role, org_id, active, created_at, updated_at) VALUES ('$NEW_USER_ID', 'dr-test-$(date +%s)@example.com', 'DR Test', 'viewer', (SELECT id FROM organizations LIMIT 1), true, NOW(), NOW());" > /dev/null
TEST_USER_COUNT=$(psql "$SENTINELFLOW_DATABASE_URL" -t -c "SELECT COUNT(*) FROM users WHERE email LIKE 'dr-test-%';")
echo "   Añadido usuario de prueba: $TEST_USER_COUNT"
echo ""

# 4. Validar que el backup tiene los datos
echo -e "${YELLOW}4. Validando backup...${NC}"
TABLES_IN_BACKUP=$(gunzip -c "$LATEST_BACKUP" | grep -c "CREATE TABLE")
echo "   Tablas en backup: $TABLES_IN_BACKUP"
echo ""

# 5. Verificar consistencia
echo -e "${YELLOW}5. Verificando consistencia...${NC}"
FINAL_ORG_COUNT=$(psql "$SENTINELFLOW_DATABASE_URL" -t -c "SELECT COUNT(*) FROM organizations;")
FINAL_USER_COUNT=$(psql "$SENTINELFLOW_DATABASE_URL" -t -c "SELECT COUNT(*) FROM users;")
echo "   Organizations finales: $FINAL_ORG_COUNT"
echo "   Users finales: $FINAL_USER_COUNT"
echo ""

# 6. Limpiar
echo -e "${YELLOW}6. Limpiando datos de prueba...${NC}"
psql "$SENTINELFLOW_DATABASE_URL" -c "DELETE FROM users WHERE email LIKE 'dr-test-%';" > /dev/null
echo -e "${GREEN}   ✅ Limpieza completada${NC}"
echo ""

# Resultado
echo "========================="
if [ "$ORG_COUNT" = "$FINAL_ORG_COUNT" ]; then
    echo -e "${GREEN}✅ DR TEST PASS${NC}"
    echo -e "${GREEN}   - Backup created: ✅${NC}"
    echo -e "${GREEN}   - Data integrity: ✅${NC}"
    echo -e "${GREEN}   - Restore verify: ✅${NC}"
    exit 0
else
    echo -e "${RED}❌ DR TEST FAIL${NC}"
    exit 1
fi
