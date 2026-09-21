#!/bin/bash
# MSMP 数据库恢复脚本（PostgreSQL）
# 用法: ./scripts/restore-db.sh <备份文件.sql.gz>
# 警告: 此操作会覆盖现有数据！

set -euo pipefail

BACKUP_FILE="$1"
if [ -z "$BACKUP_FILE" ] || [ ! -f "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file.sql.gz>"
    echo "Example: $0 /var/backups/msmp/msmp_20260921_020000.sql.gz"
    exit 1
fi

DB_HOST="${MSMP_DB_HOST:-127.0.0.1}"
DB_PORT="${MSMP_DB_PORT:-5432}"
DB_NAME="${MSMP_DB_NAME:-msmp}"
DB_USER="${MSMP_DB_USER:-msmp}"

echo "WARNING: This will overwrite the database '$DB_NAME' with backup from: $BACKUP_FILE"
read -p "Continue? (y/N): " CONFIRM
if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
    echo "Aborted."
    exit 0
fi

echo "[$(date)] Stopping MSMP server (if running)..."
# 如果通过 systemd 运行，取消注释下行：
# systemctl stop msmp-server || true

echo "[$(date)] Dropping and recreating database..."
PGPASSWORD="${MSMP_DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;"
PGPASSWORD="${MSMP_DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

echo "[$(date)] Restoring from backup..."
gunzip -c "$BACKUP_FILE" | PGPASSWORD="${MSMP_DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -q

echo "[$(date)] Restore complete."
echo "Remember to restart the MSMP server."
