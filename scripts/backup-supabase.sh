#!/bin/bash
# Backup de Supabase (PostgreSQL)

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Config
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="$BACKUP_DIR/sentinelflow-$TIMESTAMP.sql"

echo -e "${YELLOW}🗄️  Backup Supabase...${NC}"

# Verificar DATABASE_URL
if [ -z "$SENTINELFLOW_DATABASE_URL" ]; then
    echo -e "${RED}❌ SENTINELFLOW_DATABASE_URL not set${NC}"
    exit 1
fi

# Crear directorio
mkdir -p "$BACKUP_DIR"

# Ejecutar pg_dump
echo -e "${YELLOW}📦 Ejecutando pg_dump...${NC}"
pg_dump "$SENTINELFLOW_DATABASE_URL" \
    --no-owner \
    --no-acl \
    --clean \
    --if-exists \
    -f "$BACKUP_FILE"

# Comprimir
echo -e "${YELLOW}🗜️  Comprimiendo...${NC}"
gzip "$BACKUP_FILE"
BACKUP_FILE="$BACKUP_FILE.gz"

# Verificar
SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo -e "${GREEN}✅ Backup: $BACKUP_FILE ($SIZE)${NC}"

# Limpiar antiguos
echo -e "${YELLOW}🧹 Limpiando backups >$RETENTION_DAYS días...${NC}"
find "$BACKUP_DIR" -name "sentinelflow-*.sql.gz" -mtime +$RETENTION_DAYS -delete

# Contar
COUNT=$(ls -1 "$BACKUP_DIR"/sentinelflow-*.sql.gz 2>/dev/null | wc -l)
echo -e "${GREEN}📊 Backups disponibles: $COUNT${NC}"

echo ""
echo "📋 Para restaurar:"
echo "   gunzip -c $BACKUP_FILE | psql \"\$SENTINELFLOW_DATABASE_URL\""
