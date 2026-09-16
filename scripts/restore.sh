#!/bin/bash
# 数据库恢复脚本（MySQL/MariaDB）
#
# 推荐用法：在数据库容器内执行（脚本由 compose 挂载）
#   docker compose exec -T -e FORCE=1 mariadb bash /opt/jimu/scripts/restore.sh /backups/jimu_20250101_120000.sql.gz
#   make compose-db-restore FILE=/backups/jimu_20250101_120000.sql.gz
# 主机用法（本机需有 mariadb/mysql 客户端，且数据库端口可达）：
#   ./scripts/restore.sh <backup_file>
#
# 环境变量：
#   DB_HOST/DB_PORT/DB_USER/DB_NAME  连接信息（容器内默认 127.0.0.1:3306）
#   DB_PASSWORD                      口令；为空时依次尝试 DB_PASSWORD_FILE、
#                                    MARIADB_ROOT_PASSWORD_FILE/MARIADB_PASSWORD_FILE、
#                                    /run/secrets/db_root_password、/run/secrets/db_password
#   MARIADB / MYSQL                  指定客户端命令（默认自动探测 mariadb → mysql）
#   FORCE=1                          跳过交互确认（非交互环境必须显式设置）

set -euo pipefail

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

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-jimu}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-jimu}"

# read_password 按优先级从环境变量或 secret 文件读取口令
read_password() {
    if [ -n "${DB_PASSWORD}" ]; then
        printf '%s' "$DB_PASSWORD"
        return
    fi
    # 按连接用户选择优先口令：root 用 root secret，应用用户用应用 secret（compose 两个都注入）
    local candidates
    if [ "$DB_USER" = "root" ]; then
        candidates=(
            "${DB_PASSWORD_FILE:-}"
            "${MARIADB_ROOT_PASSWORD_FILE:-}"
            "${MYSQL_ROOT_PASSWORD_FILE:-}"
            "/run/secrets/db_root_password"
            "${MARIADB_PASSWORD_FILE:-}"
            "/run/secrets/db_password"
        )
    else
        candidates=(
            "${DB_PASSWORD_FILE:-}"
            "${MARIADB_PASSWORD_FILE:-}"
            "${MYSQL_PASSWORD_FILE:-}"
            "/run/secrets/db_password"
            "${MARIADB_ROOT_PASSWORD_FILE:-}"
            "/run/secrets/db_root_password"
        )
    fi
    local candidate
    for candidate in "${candidates[@]}"; do
        if [ -n "$candidate" ] && [ -r "$candidate" ]; then
            tr -d '\r\n' < "$candidate"
            return
        fi
    done
    printf ''
}

# resolve_client 探测可用的客户端：优先 mariadb（mariadb:12 官方镜像只有它，没有 mysql）
resolve_client() {
    if [ -n "${MARIADB:-}" ]; then
        printf '%s' "$MARIADB"
        return
    fi
    if [ -n "${MYSQL:-}" ]; then
        printf '%s' "$MYSQL"
        return
    fi
    if command -v mariadb >/dev/null 2>&1; then
        printf 'mariadb'
        return
    fi
    if command -v mysql >/dev/null 2>&1; then
        printf 'mysql'
        return
    fi
    echo "❌ 未找到 mariadb 或 mysql 客户端；请在数据库容器内执行本脚本，或安装 mariadb-client/mysql-client" >&2
    exit 1
}

PASSWORD="$(read_password)"
CLIENT_BIN="$(resolve_client)"

echo "=== Jimu Database Restore ==="
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

# 口令经 MYSQL_PWD 环境变量传递，不出现在命令行参数与进程列表中
restore() {
    if [ -n "$PASSWORD" ]; then
        MYSQL_PWD="$PASSWORD" "$CLIENT_BIN" \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --user="$DB_USER" \
            --default-character-set=utf8mb4 \
            "$DB_NAME"
    else
        "$CLIENT_BIN" \
            --host="$DB_HOST" \
            --port="$DB_PORT" \
            --user="$DB_USER" \
            --default-character-set=utf8mb4 \
            "$DB_NAME"
    fi
}

echo "Restoring..."
if [[ "$BACKUP_FILE" == *.gz ]]; then
    gunzip < "$BACKUP_FILE" | restore
else
    restore < "$BACKUP_FILE"
fi

echo "✅ Restore completed successfully"
