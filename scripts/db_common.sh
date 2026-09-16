#!/bin/bash
# 备份/恢复脚本的公共逻辑：方言探测、客户端探测、口令读取。
# 由 scripts/backup.sh 与 scripts/restore.sh source，不单独执行。

# resolve_driver 返回数据库方言：DB_DRIVER 显式指定，否则按可用客户端探测
resolve_driver() {
    if [ -n "${DB_DRIVER:-}" ]; then
        printf '%s' "$DB_DRIVER"
        return
    fi
    if command -v mariadb-dump >/dev/null 2>&1 || command -v mysqldump >/dev/null 2>&1; then
        printf 'mysql'
        return
    fi
    if command -v pg_dump >/dev/null 2>&1; then
        printf 'postgres'
        return
    fi
    printf 'mysql' # 交给下游给出「未找到客户端」的明确提示
}

# default_db_port 返回方言默认端口
default_db_port() {
    case "$1" in
        postgres) printf '5432' ;;
        *) printf '3306' ;;
    esac
}

# default_db_user 返回方言默认用户（MariaDB/MySQL 用应用账号，PostgreSQL 用超级用户）
default_db_user() {
    case "$1" in
        postgres) printf 'postgres' ;;
        *) printf 'jimu' ;;
    esac
}

# resolve_dump 探测 dump 命令：mysql 优先 mariadb-dump（mariadb:12 官方镜像只有它），postgres 用 pg_dump
resolve_dump() {
    case "$1" in
        postgres)
            if [ -n "${PG_DUMP:-}" ]; then printf '%s' "$PG_DUMP"; return; fi
            if command -v pg_dump >/dev/null 2>&1; then printf 'pg_dump'; return; fi
            echo "❌ 未找到 pg_dump；请在数据库容器内执行本脚本，或安装 postgresql-client" >&2
            exit 1
            ;;
        *)
            if [ -n "${MARIADB_DUMP:-}" ]; then printf '%s' "$MARIADB_DUMP"; return; fi
            if [ -n "${MYSQLDUMP:-}" ]; then printf '%s' "$MYSQLDUMP"; return; fi
            if command -v mariadb-dump >/dev/null 2>&1; then printf 'mariadb-dump'; return; fi
            if command -v mysqldump >/dev/null 2>&1; then printf 'mysqldump'; return; fi
            echo "❌ 未找到 mariadb-dump 或 mysqldump；请在数据库容器内执行本脚本，或安装 mariadb-client/mysql-client" >&2
            exit 1
            ;;
    esac
}

# resolve_client 探测恢复用客户端：mysql 优先 mariadb，postgres 用 psql
resolve_client() {
    case "$1" in
        postgres)
            if [ -n "${PSQL:-}" ]; then printf '%s' "$PSQL"; return; fi
            if command -v psql >/dev/null 2>&1; then printf 'psql'; return; fi
            echo "❌ 未找到 psql；请在数据库容器内执行本脚本，或安装 postgresql-client" >&2
            exit 1
            ;;
        *)
            if [ -n "${MARIADB:-}" ]; then printf '%s' "$MARIADB"; return; fi
            if [ -n "${MYSQL:-}" ]; then printf '%s' "$MYSQL"; return; fi
            if command -v mariadb >/dev/null 2>&1; then printf 'mariadb'; return; fi
            if command -v mysql >/dev/null 2>&1; then printf 'mysql'; return; fi
            echo "❌ 未找到 mariadb 或 mysql 客户端；请在数据库容器内执行本脚本，或安装 mariadb-client/mysql-client" >&2
            exit 1
            ;;
    esac
}

# password_env_name 返回客户端读取口令的环境变量名
password_env_name() {
    case "$1" in
        postgres) printf 'PGPASSWORD' ;;
        *) printf 'MYSQL_PWD' ;;
    esac
}

# read_password 按优先级从环境变量或 secret 文件读取口令：
#   DB_PASSWORD > DB_PASSWORD_FILE > 方言语义 secret（按 DB_USER 是否为超级用户择优先项）
read_password() {
    if [ -n "${DB_PASSWORD:-}" ]; then
        printf '%s' "$DB_PASSWORD"
        return
    fi

    local candidates=()
    case "$1" in
        postgres)
            if [ "$DB_USER" = "postgres" ]; then
                candidates=("${DB_PASSWORD_FILE:-}" "${POSTGRES_PASSWORD_FILE:-}" "/run/secrets/db_root_password" "/run/secrets/db_password")
            else
                candidates=("${DB_PASSWORD_FILE:-}" "${POSTGRES_PASSWORD_FILE:-}" "/run/secrets/db_password" "/run/secrets/db_root_password")
            fi
            ;;
        *)
            if [ "$DB_USER" = "root" ]; then
                candidates=("${DB_PASSWORD_FILE:-}" "${MARIADB_ROOT_PASSWORD_FILE:-}" "${MYSQL_ROOT_PASSWORD_FILE:-}" "/run/secrets/db_root_password" "${MARIADB_PASSWORD_FILE:-}" "/run/secrets/db_password")
            else
                candidates=("${DB_PASSWORD_FILE:-}" "${MARIADB_PASSWORD_FILE:-}" "${MYSQL_PASSWORD_FILE:-}" "/run/secrets/db_password" "${MARIADB_ROOT_PASSWORD_FILE:-}" "/run/secrets/db_root_password")
            fi
            ;;
    esac

    local candidate
    for candidate in "${candidates[@]}"; do
        if [ -n "$candidate" ] && [ -r "$candidate" ]; then
            tr -d '\r\n' < "$candidate"
            return
        fi
    done
    printf ''
}

# run_with_password <driver> <client> <password> <args...>：口令经环境变量传递，不进入命令行
run_with_password() {
    local driver="$1" client="$2" password="$3"
    shift 3
    if [ -n "$password" ]; then
        env "$(password_env_name "$driver")=$password" "$client" "$@"
    else
        "$client" "$@"
    fi
}
