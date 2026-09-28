#!/usr/bin/env bash
set -euo pipefail
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATABASE_URL="${SENTINELFLOW_DATABASE_URL:-}"
[ -n "$DATABASE_URL" ] || { echo "SENTINELFLOW_DATABASE_URL not set" >&2; exit 1; }
command -v pg_dump >/dev/null || { echo "pg_dump is required" >&2; exit 1; }
mkdir -p "$BACKUP_DIR"
ts=$(date -u +%Y%m%dT%H%M%SZ)
file="$BACKUP_DIR/sentinelflow-$ts.sql"
pg_dump "$DATABASE_URL" --format=plain --no-owner --no-acl --clean --if-exists --file="$file"
gzip -9 "$file"
sha256sum "$file.gz" > "$file.gz.sha256"
gzip -t "$file.gz"
find "$BACKUP_DIR" -name 'sentinelflow-*.sql.gz' -mtime +"$RETENTION_DAYS" -delete
find "$BACKUP_DIR" -name 'sentinelflow-*.sql.gz.sha256' -mtime +"$RETENTION_DAYS" -delete
echo "backup=$file.gz"
echo "checksum=$file.gz.sha256"
