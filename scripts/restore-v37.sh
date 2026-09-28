#!/usr/bin/env bash
set -euo pipefail
BACKUP_FILE="${1:-}"
DATABASE_URL="${SENTINELFLOW_DATABASE_URL:-}"
[ -n "$BACKUP_FILE" ] || { echo "usage: restore-v37.sh <backup.sql.gz>" >&2; exit 2; }
[ -n "$DATABASE_URL" ] || { echo "SENTINELFLOW_DATABASE_URL not set" >&2; exit 1; }
[ -f "$BACKUP_FILE" ] || { echo "backup not found: $BACKUP_FILE" >&2; exit 1; }
command -v psql >/dev/null || { echo "psql is required" >&2; exit 1; }
if [ -f "$BACKUP_FILE.sha256" ]; then sha256sum -c "$BACKUP_FILE.sha256"; fi
gzip -t "$BACKUP_FILE"
echo "RESTORE WILL REPLACE DATABASE OBJECTS. Set SF_CONFIRM_RESTORE=YES to continue."
[ "${SF_CONFIRM_RESTORE:-}" = "YES" ] || exit 3
gzip -dc "$BACKUP_FILE" | psql "$DATABASE_URL" --set ON_ERROR_STOP=1
echo "restore completed"
