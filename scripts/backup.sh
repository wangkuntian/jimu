#!/bin/bash
# 数据库备份脚本
# 用法: ./scripts/backup.sh [output_dir]
# 环境变量: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

set -euo pipefail

# 配置（可通过环境变量覆盖）
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-jimu}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-jimu}"
OUTPUT_DIR="${1:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 生成文件名
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${OUTPUT_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"

echo "=== Jimu Database Backup ==="
echo "Host: ${DB_HOST}:${DB_PORT}"
echo "Database: ${DB_NAME}"
echo "Output: ${BACKUP_FILE}"

# 执行备份
if [ -n "$DB_PASSWORD" ]; then
    mysqldump \
        --host="$DB_HOST" \
        --port="$DB_PORT" \
        --user="$DB_USER" \
        --password="$DB_PASSWORD" \
        --single-transaction \
        --routines \
        --triggers \
        --events \
        --set-gtid-purged=OFF \
        "$DB_NAME" | gzip > "$BACKUP_FILE"
else
    mysqldump \
        --host="$DB_HOST" \
        --port="$DB_PORT" \
        --user="$DB_USER" \
        --single-transaction \
        --routines \
        --triggers \
        --events \
        --set-gtid-purged=OFF \
        "$DB_NAME" | gzip > "$BACKUP_FILE"
fi

# 验证备份
if [ -s "$BACKUP_FILE" ]; then
    SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo "✅ Backup completed: ${BACKUP_FILE} (${SIZE})"
else
    echo "❌ Backup failed: file is empty"
    rm -f "$BACKUP_FILE"
    exit 1
fi

# 清理旧备份
echo "Cleaning up backups older than ${RETENTION_DAYS} days..."
find "$OUTPUT_DIR" -name "${DB_NAME}_*.sql.gz" -mtime +"$RETENTION_DAYS" -delete

# 列出当前备份
echo ""
echo "Current backups:"
ls -lh "$OUTPUT_DIR"/*.sql.gz 2>/dev/null || echo "(none)"
