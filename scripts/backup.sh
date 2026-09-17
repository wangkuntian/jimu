#!/bin/bash
# 数据库备份脚本（MySQL/MariaDB/PostgreSQL）
#
# 推荐用法：在数据库容器内执行（脚本由 compose 挂载，输出目录绑定到 ./backups）
#   docker compose exec -T mariadb bash /opt/jimu/scripts/backup.sh /backups
#   make compose-db-backup
# 主机用法（本机需有 mariadb-dump/mysqldump 或 pg_dump，且数据库端口可达）：
#   ./scripts/backup.sh [output_dir]        # 默认 ./backups
#
# 环境变量：
#   DB_DRIVER                        mysql（默认）/ postgres；未设置时按可用客户端探测
#   DB_HOST/DB_PORT/DB_USER/DB_NAME  连接信息（默认端口随方言：3306 / 5432）
#   DB_PASSWORD                      口令；为空时依次尝试 DB_PASSWORD_FILE、
#                                    MARIADB_(ROOT_)PASSWORD_FILE / POSTGRES_PASSWORD_FILE、
#                                    /run/secrets/db_root_password、/run/secrets/db_password
#   MARIADB_DUMP / MYSQLDUMP / PG_DUMP  指定 dump 命令（默认自动探测）
#   RETENTION_DAYS                   备份保留天数（默认 7，0 表示不清理）
#   GZIP=0                           不压缩（默认 gzip 压缩）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/db_common.sh
source "${SCRIPT_DIR}/db_common.sh"

DB_DRIVER="$(resolve_driver)"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-$(default_db_port "$DB_DRIVER")}"
DB_USER="${DB_USER:-$(default_db_user "$DB_DRIVER")}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-jimu}"
OUTPUT_DIR="${1:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

PASSWORD="$(read_password "$DB_DRIVER")"
DUMP_BIN="$(resolve_dump "$DB_DRIVER")"

# 组装 dump 参数：两个方言都保证一致性快照语义
case "$DB_DRIVER" in
    postgres)
        # --clean --if-exists 让恢复可覆盖既有对象；--no-password 避免交互式口令提示
        DUMP_ARGS=(
            --host="$DB_HOST"
            --port="$DB_PORT"
            --username="$DB_USER"
            --no-password
            --format=plain
            --clean
            --if-exists
            "$DB_NAME"
        )
        ;;
    *)
        # --set-gtid-purged 是 MySQL 专有选项，mariadb-dump 不支持（MariaDB 12 会直接报错），
        # 因此按实际命令是否支持来决定是否传参，避免回退到 mariadb-dump 后备份失败。
        GTID_FLAG=()
        if "$DUMP_BIN" --help 2>&1 | grep -q -- '--set-gtid-purged'; then
            GTID_FLAG=(--set-gtid-purged=OFF)
        fi
        DUMP_ARGS=(
            --host="$DB_HOST"
            --port="$DB_PORT"
            --user="$DB_USER"
            --single-transaction   # InnoDB 一致性快照，不锁表
            --routines
            --triggers
            --events
            --default-character-set=utf8mb4
            "${GTID_FLAG[@]}"
            "$DB_NAME"
        )
        ;;
esac

mkdir -p "$OUTPUT_DIR"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
if [ "${GZIP:-1}" = "0" ]; then
    BACKUP_FILE="${OUTPUT_DIR}/${DB_NAME}_${TIMESTAMP}.sql"
else
    BACKUP_FILE="${OUTPUT_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"
fi

echo "=== Jimu Database Backup ==="
echo "Driver:   ${DB_DRIVER}"
echo "Host:     ${DB_HOST}:${DB_PORT}"
echo "Database: ${DB_NAME}"
echo "Tool:     ${DUMP_BIN}"
echo "Output:   ${BACKUP_FILE}"

if [ "${GZIP:-1}" = "0" ]; then
    run_with_password "$DB_DRIVER" "$DUMP_BIN" "$PASSWORD" "${DUMP_ARGS[@]}" > "$BACKUP_FILE"
else
    run_with_password "$DB_DRIVER" "$DUMP_BIN" "$PASSWORD" "${DUMP_ARGS[@]}" | gzip > "$BACKUP_FILE"
fi

if [ -s "$BACKUP_FILE" ]; then
    SIZE="$(du -h "$BACKUP_FILE" | cut -f1)"
    echo "✅ Backup completed: ${BACKUP_FILE} (${SIZE})"
else
    echo "❌ Backup failed: file is empty" >&2
    rm -f "$BACKUP_FILE"
    exit 1
fi

if [ "$RETENTION_DAYS" -gt 0 ]; then
    echo "Cleaning up backups older than ${RETENTION_DAYS} days..."
    find "$OUTPUT_DIR" -maxdepth 1 \( -name "${DB_NAME}_*.sql.gz" -o -name "${DB_NAME}_*.sql" \) -mtime +"$RETENTION_DAYS" -delete
fi

echo ""
echo "Current backups:"
ls -lh "$OUTPUT_DIR"/"${DB_NAME}"_*.sql* 2>/dev/null || echo "(none)"
