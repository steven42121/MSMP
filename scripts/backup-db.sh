#!/bin/bash
# MSMP 数据库备份脚本（PostgreSQL）
# 用法: ./scripts/backup-db.sh [输出目录]
# 建议配合 crontab: 0 2 * * * /path/to/backup-db.sh /var/backups/msmp

set -euo pipefail

# ── 配置（可通过环境变量覆盖） ──
DB_HOST="${MSMP_DB_HOST:-127.0.0.1}"
DB_PORT="${MSMP_DB_PORT:-5432}"
DB_NAME="${MSMP_DB_NAME:-msmp}"
DB_USER="${MSMP_DB_USER:-msmp}"
BACKUP_DIR="${1:-/var/backups/msmp}"
RETAIN_DAYS="${MSMP_BACKUP_RETAIN_DAYS:-30}"

# ── 创建备份目录 ──
mkdir -p "$BACKUP_DIR"

# ── 执行备份 ──
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/msmp_${TIMESTAMP}.sql.gz"

echo "[$(date)] Starting backup: $DB_NAME@$DB_HOST:$DB_PORT"
PGPASSWORD="${MSMP_DB_PASSWORD:-}" pg_dump \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --no-owner \
    --no-privileges \
    -F p \
    | gzip > "$BACKUP_FILE"

SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo "[$(date)] Backup complete: $BACKUP_FILE ($SIZE)"

# ── 清理过期备份 ──
DELETED=$(find "$BACKUP_DIR" -name "msmp_*.sql.gz" -mtime +"$RETAIN_DAYS" -delete -print | wc -l)
if [ "$DELETED" -gt 0 ]; then
    echo "[$(date)] Cleaned $DELETED expired backup(s) (older than ${RETAIN_DAYS}d)"
fi

echo "[$(date)] Backup finished successfully"
