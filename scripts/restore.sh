#!/bin/bash
# 数据库恢复脚本
# 用法: ./scripts/restore.sh <backup_file>
# 环境变量: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME

set -euo pipefail

# 检查参数
if [ $# -lt 1 ]; then
    echo "Usage: $0 <backup_file>"
    echo "Example: $0 ./backups/jimu_20240101_120000.sql.gz"
    exit 1
fi

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ Backup file not found: $BACKUP_FILE"
    exit 1
fi

# 配置
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-jimu}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-jimu}"

echo "=== Jimu Database Restore ==="
echo "Host: ${DB_HOST}:${DB_PORT}"
echo "Database: ${DB_NAME}"
echo "Source: ${BACKUP_FILE}"
echo ""

# 确认
read -p "⚠️  This will OVERWRITE the database. Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

# 判断文件类型并恢复
echo "Restoring..."
if [[ "$BACKUP_FILE" == *.gz ]]; then
    if [ -n "$DB_PASSWORD" ]; then
        gunzip < "$BACKUP_FILE" | mysql \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --user="$DB_USER" \
            --password="$DB_PASSWORD" \
            "$DB_NAME"
    else
        gunzip < "$BACKUP_FILE" | mysql \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --user="$DB_USER" \
            "$DB_NAME"
    fi
else
    if [ -n "$DB_PASSWORD" ]; then
        mysql \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --user="$DB_USER" \
            --password="$DB_PASSWORD" \
            "$DB_NAME" < "$BACKUP_FILE"
    else
        mysql \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --user="$DB_USER" \
            "$DB_NAME" < "$BACKUP_FILE"
    fi
fi

echo "✅ Restore completed successfully"
