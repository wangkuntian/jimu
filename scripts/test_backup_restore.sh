#!/bin/bash
# 备份/恢复往返测试：建标记数据 → 调用真实备份脚本 → 破坏数据 → 调用真实恢复脚本 → 校验还原。
#
# 两种模式：
#   MODE=docker  在数据库容器内执行脚本（脚本需已挂载，compose 默认挂载到 /opt/jimu/scripts）
#   MODE=host    用本机客户端直连数据库执行脚本（CI 的 service container 用这种）
#   MODE 留空时自动判断：容器存在可用则 docker，否则 host。
#
# 用法:
#   ./scripts/test_backup_restore.sh [container] [db_name]
#   MODE=host DB_DRIVER=postgres DB_HOST=127.0.0.1 DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=jimu_test \
#     ./scripts/test_backup_restore.sh
# 环境变量: DB_DRIVER（mysql 默认 / postgres）、DB_HOST、DB_PORT、DB_USER、DB_PASSWORD、DB_NAME

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER="${1:-${CONTAINER:-}}"
DB_DRIVER="${DB_DRIVER:-mysql}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-$([ "$DB_DRIVER" = "postgres" ] && echo 5432 || echo 3306)}"
DB_USER="${DB_USER:-$([ "$DB_DRIVER" = "postgres" ] && echo postgres || echo root)}"
DB_PASSWORD="${DB_PASSWORD:-root}"
DB_NAME="${2:-${DB_NAME:-jimu_test}}"
FORCE="${FORCE:-1}" # 往返测试属于非交互流程
export FORCE

# 自动判断执行模式
SELECTED_MODE="${MODE:-}"
if [ -z "$SELECTED_MODE" ]; then
    if [ -n "$CONTAINER" ] && docker inspect "$CONTAINER" >/dev/null 2>&1; then
        SELECTED_MODE=docker
    else
        SELECTED_MODE=host
    fi
fi

CONTAINER_DIR="/tmp/jimu-backup-rt"
WORKDIR="$(mktemp -d)"
cleanup() {
    rm -rf "$WORKDIR"
    if [ "$SELECTED_MODE" = "docker" ] && [ -n "$CONTAINER" ]; then
        docker exec "$CONTAINER" rm -rf "$CONTAINER_DIR" >/dev/null 2>&1 || true
    fi
}
trap cleanup EXIT

MARKER_TABLE="backup_restore_marker"
MARKER_VAL="roundtrip-$(date +%s)"

# 容器模式：脚本在容器内、产物也写在容器内（宿主机无需可见）
SCRIPT_PATH_IN_CONTAINER="/opt/jimu/scripts"
if [ "$SELECTED_MODE" = "docker" ]; then
    docker exec "$CONTAINER" mkdir -p "$CONTAINER_DIR"
    BACKUP_DIR="$CONTAINER_DIR"
else
    BACKUP_DIR="$WORKDIR"
fi

# 客户端参数（两种模式共用）
case "$DB_DRIVER" in
    postgres)
        CLIENT_ENV="PGPASSWORD=${DB_PASSWORD}"
        CLIENT_ARGS=(--host="$DB_HOST" --port="$DB_PORT" --username="$DB_USER" --no-password --dbname="$DB_NAME")
        ;;
    *)
        CLIENT_ENV="MYSQL_PWD=${DB_PASSWORD}"
        CLIENT_ARGS=(--host="$DB_HOST" --port="$DB_PORT" --user="$DB_USER" "$DB_NAME")
        ;;
esac

# exec_sql <sql...>：执行 SQL（docker 模式走容器内客户端，host 模式用本机客户端）
exec_sql() {
    if [ "$SELECTED_MODE" = "docker" ]; then
        case "$DB_DRIVER" in
            postgres) docker exec -e "$CLIENT_ENV" "$CONTAINER" psql "${CLIENT_ARGS[@]}" "$@" ;;
            *) docker exec -e "$CLIENT_ENV" "$CONTAINER" mariadb "${CLIENT_ARGS[@]}" "$@" ;;
        esac
    else
        case "$DB_DRIVER" in
            postgres) env "$CLIENT_ENV" psql "${CLIENT_ARGS[@]}" "$@" ;;
            *) env "$CLIENT_ENV" mariadb "${CLIENT_ARGS[@]}" "$@" ;;
        esac
    fi
}

# query_value <sql>：取单个标量值
query_value() {
    if [ "$SELECTED_MODE" = "docker" ]; then
        case "$DB_DRIVER" in
            postgres) docker exec -e "$CLIENT_ENV" "$CONTAINER" psql "${CLIENT_ARGS[@]}" -t -A -c "$1" ;;
            *) docker exec -e "$CLIENT_ENV" "$CONTAINER" mariadb "${CLIENT_ARGS[@]}" -N -e "$1" ;;
        esac
    else
        case "$DB_DRIVER" in
            postgres) env "$CLIENT_ENV" psql "${CLIENT_ARGS[@]}" -t -A -c "$1" ;;
            *) env "$CLIENT_ENV" mariadb "${CLIENT_ARGS[@]}" -N -e "$1" ;;
        esac
    fi
}

# sql <statement>：执行一条语句（忽略输出）
sql() {
    if [ "$DB_DRIVER" = "postgres" ]; then
        exec_sql -c "$1" >/dev/null
    else
        exec_sql -e "$1" >/dev/null
    fi
}

# run_script <script> <args...>：调用真实备份/恢复脚本
run_script() {
    local script="$1"
    shift
    if [ "$SELECTED_MODE" = "docker" ]; then
        local envs=(-e "DB_DRIVER=$DB_DRIVER" -e "DB_HOST=$DB_HOST" -e "DB_PORT=$DB_PORT" -e "DB_USER=$DB_USER" -e "DB_PASSWORD=$DB_PASSWORD" -e "DB_NAME=$DB_NAME" -e "FORCE=$FORCE")
        docker exec "${envs[@]}" "$CONTAINER" bash "${SCRIPT_PATH_IN_CONTAINER}/${script}" "$@"
    else
        DB_DRIVER="$DB_DRIVER" DB_HOST="$DB_HOST" DB_PORT="$DB_PORT" DB_USER="$DB_USER" DB_PASSWORD="$DB_PASSWORD" DB_NAME="$DB_NAME" \
            bash "${SCRIPT_DIR}/${script}" "$@"
    fi
}

echo "=== 备份/恢复往返测试 ==="
echo "mode:     ${SELECTED_MODE}"
[ "$SELECTED_MODE" = "docker" ] && echo "container:${CONTAINER}"
echo "driver:   ${DB_DRIVER}"
echo "database: ${DB_NAME}"

# 1. 建标记表 + 写入一行
sql "DROP TABLE IF EXISTS ${MARKER_TABLE}"
sql "CREATE TABLE ${MARKER_TABLE} (id INT PRIMARY KEY, val VARCHAR(255))"
sql "INSERT INTO ${MARKER_TABLE} (id, val) VALUES (1, '${MARKER_VAL}')"
echo "✅ 写入标记数据: ${MARKER_VAL}"

# 2. 调用真实备份脚本
run_script backup.sh "$BACKUP_DIR" >/dev/null
if [ "$SELECTED_MODE" = "docker" ]; then
    DUMP_FILE="$(docker exec "$CONTAINER" sh -c "ls -1 ${BACKUP_DIR}/${DB_NAME}_*.sql* | head -1")"
    docker exec "$CONTAINER" sh -c "[ -s '${DUMP_FILE}' ]" || { echo "❌ dump 为空"; exit 1; }
    DUMP_SIZE="$(docker exec "$CONTAINER" du -h "$DUMP_FILE" | cut -f1)"
else
    DUMP_FILE="$(ls -1 "${BACKUP_DIR}/${DB_NAME}"_*.sql* | head -1)"
    [ -s "$DUMP_FILE" ] || { echo "❌ dump 为空"; exit 1; }
    DUMP_SIZE="$(du -h "$DUMP_FILE" | cut -f1)"
fi
echo "✅ 备份完成: ${DUMP_FILE} (${DUMP_SIZE})"

# 3. 破坏数据
sql "TRUNCATE TABLE ${MARKER_TABLE}"
COUNT="$(query_value "SELECT count(*) FROM ${MARKER_TABLE}" | tr -d '[:space:]')"
[ "$COUNT" = "0" ] || { echo "❌ truncate 后应无数据，实际 $COUNT"; exit 1; }
echo "✅ 数据已清空"

# 4. 调用真实恢复脚本
run_script restore.sh "$DUMP_FILE" >/dev/null

# 5. 验证数据还原
RESTORED="$(query_value "SELECT val FROM ${MARKER_TABLE} WHERE id = 1" | tr -d '[:space:]')"
if [ "$RESTORED" = "$MARKER_VAL" ]; then
    echo "✅ 恢复成功，数据还原: ${RESTORED}"
else
    echo "❌ 恢复失败，期望 '${MARKER_VAL}'，实际 '${RESTORED}'" >&2
    exit 1
fi

# 清理
sql "DROP TABLE ${MARKER_TABLE}"
echo "✅ 往返测试通过"
