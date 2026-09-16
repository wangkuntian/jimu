#!/bin/bash
# 数据库恢复脚本（MySQL/MariaDB/PostgreSQL）
#
# 推荐用法：在数据库容器内执行（脚本由 compose 挂载）
#   docker compose exec -T -e FORCE=1 mariadb bash /opt/jimu/scripts/restore.sh /backups/jimu_20250101_120000.sql.gz
#   make compose-db-restore FILE=/backups/jimu_20250101_120000.sql.gz
# 主机用法（本机需有 mariadb/mysql 或 psql，且数据库端口可达）：
#   ./scripts/restore.sh <backup_file>
#
# 环境变量：
#   DB_DRIVER                        mysql（默认）/ postgres；未设置时按可用客户端探测
#   DB_HOST/DB_PORT/DB_USER/DB_NAME  连接信息（默认端口随方言：3306 / 5432）
#   DB_PASSWORD                      口令；为空时依次尝试 DB_PASSWORD_FILE、
#                                    MARIADB_(ROOT_)PASSWORD_FILE / POSTGRES_PASSWORD_FILE、
#                                    /run/secrets/db_root_password、/run/secrets/db_password
#   MARIADB / MYSQL / PSQL           指定客户端命令（默认自动探测）
#   FORCE=1                          跳过交互确认（非交互环境必须显式设置）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/db_common.sh
source "${SCRIPT_DIR}/db_common.sh"

if [ $# -lt 1 ]; then
    echo "Usage: $0 <backup_file>"
    echo "Example: $0 ./backups/jimu_20240101_120000.sql.gz"
    exit 1
fi

BACKUP_FILE="$1"
if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ Backup file not found: $BACKUP_FILE" >&2
    exit 1
fi
if [ ! -s "$BACKUP_FILE" ]; then
    echo "❌ Backup file is empty: $BACKUP_FILE" >&2
    exit 1
fi

DB_DRIVER="$(resolve_driver)"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-$(default_db_port "$DB_DRIVER")}"
DB_USER="${DB_USER:-$(default_db_user "$DB_DRIVER")}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-jimu}"

PASSWORD="$(read_password "$DB_DRIVER")"
CLIENT_BIN="$(resolve_client "$DB_DRIVER")"

echo "=== Jimu Database Restore ==="
echo "Driver:   ${DB_DRIVER}"
echo "Host:     ${DB_HOST}:${DB_PORT}"
echo "Database: ${DB_NAME}"
echo "Source:   ${BACKUP_FILE}"
echo "Tool:     ${CLIENT_BIN}"
echo ""

# 确认（FORCE=1 跳过交互，供 make/CI 使用）
if [ "${FORCE:-0}" != "1" ]; then
    if [ ! -t 0 ]; then
        # 非交互环境（如 docker compose exec -T）无法交互确认，必须显式声明风险
        echo "❌ 非交互环境：恢复会覆盖现有数据，请显式设置 FORCE=1 后重试" >&2
        exit 1
    fi
    read -r -p "⚠️  This will OVERWRITE the database. Continue? (y/N) " -n 1
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Aborted."
        exit 0
    fi
fi

# 客户端参数：口令经环境变量传递，不进入命令行
case "$DB_DRIVER" in
    postgres)
        CLIENT_ARGS=(
            --host="$DB_HOST"
            --port="$DB_PORT"
            --username="$DB_USER"
            --no-password
            --set=ON_ERROR_STOP=on   # 遇错立即失败，避免半截恢复被当成成功
            --dbname="$DB_NAME"
        )
        ;;
    *)
        CLIENT_ARGS=(
            --host="$DB_HOST"
            --port="$DB_PORT"
            --user="$DB_USER"
            --default-character-set=utf8mb4
            "$DB_NAME"
        )
        ;;
esac

echo "Restoring..."
if [[ "$BACKUP_FILE" == *.gz ]]; then
    gunzip < "$BACKUP_FILE" | run_with_password "$DB_DRIVER" "$CLIENT_BIN" "$PASSWORD" "${CLIENT_ARGS[@]}"
else
    run_with_password "$DB_DRIVER" "$CLIENT_BIN" "$PASSWORD" "${CLIENT_ARGS[@]}" < "$BACKUP_FILE"
fi

echo "✅ Restore completed successfully"
