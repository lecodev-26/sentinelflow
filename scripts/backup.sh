#!/bin/bash
# Backup automático de SentinelFlow

set -e

# Config
DB_PATH="${DB_PATH:-sentinelflow.db}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
S3_BUCKET="${S3_BUCKET:-}"

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="$BACKUP_DIR/sentinelflow-$TIMESTAMP.db"

echo -e "${YELLOW}🗄️  Iniciando backup...${NC}"

# Crear directorio
mkdir -p "$BACKUP_DIR"

# Verificar DB existe
if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}❌ DB no encontrada: $DB_PATH${NC}"
    exit 1
fi

# Backup con SQLite
if command -v sqlite3 &> /dev/null; then
    sqlite3 "$DB_PATH" ".backup $BACKUP_FILE"
    echo -e "${GREEN}✅ Backup local: $BACKUP_FILE${NC}"
else
    # Fallback: copia simple
    cp "$DB_PATH" "$BACKUP_FILE"
    echo -e "${YELLOW}⚠️  sqlite3 no disponible, copia simple${NC}"
fi

# Comprimir
gzip "$BACKUP_FILE"
BACKUP_FILE="$BACKUP_FILE.gz"

# Upload a S3 si está configurado
if [ -n "$S3_BUCKET" ] && command -v aws &> /dev/null; then
    aws s3 cp "$BACKUP_FILE" "s3://$S3_BUCKET/sentinelflow-backups/"
    echo -e "${GREEN}✅ Backup subido a S3${NC}"
fi

# Limpiar backups antiguos
find "$BACKUP_DIR" -name "sentinelflow-*.db.gz" -mtime +$RETENTION_DAYS -delete
echo -e "${GREEN}✅ Backups antiguos eliminados (>$RETENTION_DAYS días)${NC}"

# Restaurar info
echo ""
echo "📋 Para restaurar:"
echo "   gunzip -c $BACKUP_FILE | sqlite3 $DB_PATH"
