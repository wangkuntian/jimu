#!/bin/bash
# 数据库备份脚本（MySQL/MariaDB）
#
# 推荐用法：在数据库容器内执行（脚本由 compose 挂载，输出目录绑定到 ./backups）
#   docker compose exec -T mariadb bash /opt/jimu/scripts/backup.sh /backups
#   make compose-db-backup
# 主机用法（本机需有 mariadb-dump/mysqldump 客户端，且数据库端口可达）：
#   ./scripts/backup.sh [output_dir]        # 默认 ./backups
#
# 环境变量：
#   DB_HOST/DB_PORT/DB_USER/DB_NAME  连接信息（容器内默认 127.0.0.1:3306）
#   DB_PASSWORD                      口令；为空时依次尝试 DB_PASSWORD_FILE、
#                                    MARIADB_ROOT_PASSWORD_FILE/MARIADB_PASSWORD_FILE、
#                                    /run/secrets/db_root_password、/run/secrets/db_password
#   MARIADB_DUMP / MYSQLDUMP         指定 dump 命令（默认自动探测 mariadb-dump → mysqldump）
#   RETENTION_DAYS                   备份保留天数（默认 7，0 表示不清理）
#   GZIP=0                           不压缩（默认 gzip 压缩）

set -euo pipefail

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-jimu}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-jimu}"
OUTPUT_DIR="${1:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

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

# resolve_dump 探测可用的 dump 命令：优先 mariadb-dump（mariadb:12 官方镜像只有它，没有 mysqldump）
resolve_dump() {
    if [ -n "${MARIADB_DUMP:-}" ]; then
        printf '%s' "$MARIADB_DUMP"
        return
    fi
    if [ -n "${MYSQLDUMP:-}" ]; then
        printf '%s' "$MYSQLDUMP"
        return
    fi
    if command -v mariadb-dump >/dev/null 2>&1; then
        printf 'mariadb-dump'
        return
    fi
    if command -v mysqldump >/dev/null 2>&1; then
        printf 'mysqldump'
        return
    fi
    echo "❌ 未找到 mariadb-dump 或 mysqldump；请在数据库容器内执行本脚本，或安装 mariadb-client/mysql-client" >&2
    exit 1
}

PASSWORD="$(read_password)"
DUMP_BIN="$(resolve_dump)"

# --set-gtid-purged 是 MySQL 专有选项，mariadb-dump 不支持（MariaDB 12 会直接报错），
# 因此按实际命令是否支持来决定是否传参，避免回退到 mariadb-dump 后备份失败。
GTID_FLAG=()
if "$DUMP_BIN" --help 2>&1 | grep -q -- '--set-gtid-purged'; then
    GTID_FLAG=(--set-gtid-purged=OFF)
fi

mkdir -p "$OUTPUT_DIR"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
if [ "${GZIP:-1}" = "0" ]; then
    BACKUP_FILE="${OUTPUT_DIR}/${DB_NAME}_${TIMESTAMP}.sql"
else
    BACKUP_FILE="${OUTPUT_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"
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

echo "=== Jimu Database Backup ==="
echo "Host:     ${DB_HOST}:${DB_PORT}"
echo "Database: ${DB_NAME}"
echo "Tool:     ${DUMP_BIN}"
echo "Output:   ${BACKUP_FILE}"

# 口令经 MYSQL_PWD 环境变量传递，不出现在命令行参数与进程列表中
dump() {
    if [ -n "$PASSWORD" ]; then
        MYSQL_PWD="$PASSWORD" "$DUMP_BIN" "${DUMP_ARGS[@]}"
    else
        "$DUMP_BIN" "${DUMP_ARGS[@]}"
    fi
}

if [ "${GZIP:-1}" = "0" ]; then
    dump > "$BACKUP_FILE"
else
    dump | gzip > "$BACKUP_FILE"
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
