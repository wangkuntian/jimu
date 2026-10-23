#!/usr/bin/env bash
# 初始化 Compose 所需 secret 文件。默认只补齐缺失项，不覆盖已有值。
set -euo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SECRETS_DIR="${SECRETS_DIR:-$ROOT/secrets}"
LEGACY_ENV_FILE="${LEGACY_ENV_FILE:-}"
FORCE="${FORCE:-0}"
TEMP_FILE=""

OBSERVABILITY=0
ALLOW_NEW_SECURITY_KEYS=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --observability)
      OBSERVABILITY=1
      ;;
    --allow-new-security-keys)
      ALLOW_NEW_SECURITY_KEYS=1
      ;;
    *)
      echo "未知参数：$1" >&2
      echo "用法：scripts/init-secrets.sh [--observability] [--allow-new-security-keys]" >&2
      exit 2
      ;;
  esac
  shift
done

warn() {
  printf '⚠️  %s\n' "$*" >&2
}

die() {
  printf '❌ %s\n' "$*" >&2
  exit 1
}

cleanup_temp() {
  [ -z "$TEMP_FILE" ] || rm -f "$TEMP_FILE"
}
trap cleanup_temp EXIT

command -v openssl >/dev/null 2>&1 || die "缺少 openssl，无法生成 secret"
if [ "$OBSERVABILITY" = "1" ]; then
  command -v base64 >/dev/null 2>&1 || die "缺少 base64，无法生成 OpenObserve token"
fi

raw_env_value() {
  local file="$1" key="$2" label="$3" line status
  [ -f "$file" ] || return 0
  if line="$(awk -v key="$key" '
    {
      line = $0
      sub(/\r$/, "", line)
      if (line ~ "^[[:space:]]*(export[[:space:]]+)?" key "[[:space:]]*=") {
        sub("^[[:space:]]*(export[[:space:]]+)?" key "[[:space:]]*=", "", line)
        count++
        value = line
      }
    }
    END {
      if (count > 1) exit 42
      if (count == 1) print value
    }
  ' "$file")"; then
    printf '%s' "$line"
    return 0
  else
    status=$?
  fi
  if [ "$status" = "42" ]; then
    die "${label}: ${key} 出现重复赋值；拒绝猜测优先级"
  fi
  die "${label}: 解析 ${key} 失败"
}

validate_env_tail() {
  local label="$1" tail="$2" remainder
  [ -z "$tail" ] && return 0
  case "$tail" in
    [[:space:]]*) ;;
    *) die "${label}: 不支持值后的额外内容" ;;
  esac
  remainder="${tail#"${tail%%[![:space:]]*}"}"
  [ -z "$remainder" ] && return 0
  case "$remainder" in
    \#*) return 0 ;;
    *) die "${label}: 不支持值后的额外内容" ;;
  esac
}

parse_env_value() {
  local file="$1" key="$2" label="$3" raw value tail body char escaped
  local i length closed
  [ -f "$file" ] || return 0
  raw="$(raw_env_value "$file" "$key" "$label")"
  [ -n "$raw" ] || return 0

  case "${raw:0:1}" in
    "'")
      body="${raw:1}"
      case "$body" in
        *"'"*) ;;
        *) die "${label}: 单引号未闭合" ;;
      esac
      value="${body%%\'*}"
      tail="${body#*\'}"
      validate_env_tail "$label" "$tail"
      printf '%s' "$value"
      ;;
    '"')
      body="${raw:1}"
      length="${#body}"
      value=""
      i=0
      closed=0
      while [ "$i" -lt "$length" ]; do
        char="${body:$i:1}"
        if [ "$char" = '"' ]; then
          closed=1
          i=$((i + 1))
          tail="${body:$i}"
          break
        fi
        if [ "$char" = '\' ]; then
          i=$((i + 1))
          [ "$i" -lt "$length" ] || die "${label}: 双引号内的转义不完整"
          escaped="${body:$i:1}"
          case "$escaped" in
            '"'|'\'|'$'|'`') value="$value$escaped" ;;
            *) die "${label}: 不支持转义序列 \\${escaped}" ;;
          esac
        else
          case "$char" in
            '$'|'`') die "${label}: 不支持 shell 展开" ;;
          esac
          value="$value$char"
        fi
        i=$((i + 1))
      done
      [ "$closed" = "1" ] || die "${label}: 双引号未闭合"
      validate_env_tail "$label" "$tail"
      printf '%s' "$value"
      ;;
    [[:space:]])
      tail="${raw#"${raw%%[![:space:]]*}"}"
      case "$tail" in
        ''|\#*) printf '' ;;
        *) die "${label}: 不支持等号后先出现空白" ;;
      esac
      ;;
    *)
      case "$raw" in
        *[[:space:]]*)
          value="${raw%%[[:space:]]*}"
          tail="${raw:${#value}}"
          ;;
        *)
          value="$raw"
          tail=""
          ;;
      esac
      case "$value" in
        *'$'*|*'`'*|*'\'*|*';'*|*'&'*|*'|'*|*'<'*|*'>'*|*'('*|*')'*|*'{'*|*'}'*|*"'"*|*'"'*|'~'*)
          die "${label}: 不支持未加引号的 shell 语法"
          ;;
      esac
      validate_env_tail "$label" "$tail"
      printf '%s' "$value"
      ;;
  esac
}

legacy_env_value() {
  parse_env_value "$LEGACY_ENV_FILE" "$1" "旧 .env 中的 $1"
}

dotenv_value() {
  parse_env_value "$SECRETS_DIR/openobserve.env" "$1" "openobserve.env 中的 $1"
}

dotenv_quote() {
  local value="$1" escaped
  case "$value" in
    *"'"*)
      escaped="${value//\\/\\\\}"
      escaped="${escaped//\"/\\\"}"
      escaped="${escaped//\$/\\\$}"
      escaped="${escaped//\`/\\\`}"
      printf '"%s"' "$escaped"
      ;;
    *)
      printf "'%s'" "$value"
      ;;
  esac
}

write_secret() {
  local name="$1" value="$2" target tmp
  target="$SECRETS_DIR/$name"
  if [ -s "$target" ] && [ "$FORCE" != "1" ]; then
    chmod 0600 "$target"
    printf '   - %s 已存在，保留\n' "$name"
    return 0
  fi

  TEMP_FILE="$(mktemp "$SECRETS_DIR/.${name}.tmp.XXXXXX")" \
    || die "无法为 $name 创建临时文件"
  tmp="$TEMP_FILE"
  if ! printf '%s' "$value" > "$tmp"; then
    rm -f "$tmp"
    TEMP_FILE=""
    die "写入 $name 失败"
  fi
  if ! chmod 0600 "$tmp"; then
    rm -f "$tmp"
    TEMP_FILE=""
    die "设置 $name 权限失败"
  fi
  if ! mv -f "$tmp" "$target"; then
    rm -f "$tmp"
    TEMP_FILE=""
    die "原子替换 $name 失败"
  fi
  TEMP_FILE=""
  if [ "$FORCE" = "1" ]; then
    printf '   ↻ %s\n' "$name"
  else
    printf '   + %s\n' "$name"
  fi
}

import_legacy_secret() {
  local name="$1" key="$2" value
  [ -s "$SECRETS_DIR/$name" ] && return 0
  value="$(legacy_env_value "$key")"
  [ -n "$value" ] || return 0
  write_secret "$name" "$value"
}

probe_volume() {
  local volume="$1" output status
  VOLUME_PROBE_STATE="unknown"
  VOLUME_PROBE_DETAIL=""
  if ! command -v docker >/dev/null 2>&1; then
    VOLUME_PROBE_DETAIL="未找到 docker 命令"
    return 0
  fi
  if output="$(docker volume inspect "$volume" 2>&1)"; then
    VOLUME_PROBE_STATE="exists"
    return 0
  else
    status=$?
  fi
  if [ "$status" = "1" ]; then
    case "$output" in
      *"No such volume"*|*"no such volume"*)
        VOLUME_PROBE_STATE="absent"
        return 0
        ;;
    esac
  fi
  VOLUME_PROBE_DETAIL="${output:-docker volume inspect 退出码 ${status}}"
}

generate_random_password() {
  openssl rand -hex 24
}

generate_encryption_key() {
  openssl rand -base64 32 | tr -d '\n' | cut -c1-32
}

generate_openobserve_password() {
  printf 'Ao1!%s' "$(openssl rand -hex 16)"
}

mkdir -p "$SECRETS_DIR"
chmod 0700 "$SECRETS_DIR"

validate_encryption_key() {
  local file="$SECRETS_DIR/encryption_key.txt" size
  [ -e "$file" ] || return 0
  [ -f "$file" ] || die "encryption_key.txt 不是普通文件"
  size="$(wc -c < "$file" | tr -d '[:space:]')"
  [ "$size" = "32" ] || die "encryption_key.txt 必须恰好 32 字节，当前为 ${size} 字节"
  LC_ALL=C grep -Eq '^[ -~]{32}$' "$file" \
    || die "encryption_key.txt 必须是 32 个可打印 ASCII 字节且无换行"
}

validate_encryption_key

if [ -n "$LEGACY_ENV_FILE" ] && [ -f "$LEGACY_ENV_FILE" ]; then
  warn "检测到旧配置文件 ${LEGACY_ENV_FILE}；该文件已停用，仅迁移缺失的 secret，不删除原文件"
  for key in DB_ROOT_PASSWORD DB_PASSWORD JWT_SECRET ADMIN_PASSWORD ZO_OBSERVE_ROOT_USER_PASSWORD; do
    raw_env_value "$LEGACY_ENV_FILE" "$key" "旧 .env 中的 $key" >/dev/null
  done
  import_legacy_secret "db_root_password.txt" "DB_ROOT_PASSWORD"
  import_legacy_secret "db_password.txt" "DB_PASSWORD"
  import_legacy_secret "jwt_secret.txt" "JWT_SECRET"
  if [ -z "${ADMIN_PASSWORD:-}" ]; then
    import_legacy_secret "admin_password.txt" "ADMIN_PASSWORD"
  fi
fi

MARIADB_VOLUME="${COMPOSE_MARIADB_VOLUME:-jimu-mariadb-data}"
OPENOBSERVE_VOLUME="${COMPOSE_OPENOBSERVE_VOLUME:-jimu-openobserve-data}"
MARIADB_VOLUME_EXISTS=0
OPENOBSERVE_VOLUME_EXISTS=0
probe_volume "$MARIADB_VOLUME"
case "$VOLUME_PROBE_STATE" in
  exists) MARIADB_VOLUME_EXISTS=1 ;;
  absent) ;;
  *)
    die "无法确认 Docker 数据卷 ${MARIADB_VOLUME} 是否存在：${VOLUME_PROBE_DETAIL}；为防止覆盖已有数据，拒绝生成 secret"
    ;;
esac
if [ "$OBSERVABILITY" = "1" ]; then
  probe_volume "$OPENOBSERVE_VOLUME"
  case "$VOLUME_PROBE_STATE" in
    exists) OPENOBSERVE_VOLUME_EXISTS=1 ;;
    absent) ;;
    *)
      die "无法确认 Docker 数据卷 ${OPENOBSERVE_VOLUME} 是否存在：${VOLUME_PROBE_DETAIL}；为防止覆盖已有管理员，拒绝生成 secret"
      ;;
  esac
fi

PROTECT_SECURITY_SECRETS=0
if [ "$MARIADB_VOLUME_EXISTS" = "1" ]; then
  for file in db_root_password.txt db_password.txt; do
    [ -s "$SECRETS_DIR/$file" ] \
      || die "检测到已有 MariaDB 数据卷 ${MARIADB_VOLUME}，但缺少 ${file}；请恢复原数据库凭据，不能自动生成新密码"
  done

  missing_security=""
  for file in jwt_secret.txt encryption_key.txt audit_hash_secret.txt admin_password.txt; do
    if [ ! -s "$SECRETS_DIR/$file" ]; then
      missing_security="$missing_security $file"
    fi
  done
  if [ -n "$missing_security" ]; then
    if [ "$ALLOW_NEW_SECURITY_KEYS" != "1" ]; then
      die "检测到已有 MariaDB 数据卷，但缺少安全 secret:${missing_security}；请恢复原值，或确认可生成新密钥后传 --allow-new-security-keys"
    fi
    warn "警告：将生成新的安全 secret:${missing_security}；已有 Token、加密字段或审计链可能因此无法继续使用"
  else
    PROTECT_SECURITY_SECRETS=1
  fi
fi

OBS_EMAIL="${ZO_OBSERVE_ROOT_USER_EMAIL:-}"
OBS_FILE_EMAIL=""
OBS_PASSWORD=""
OBS_FILE_PASSWORD=""
LEGACY_OBS_PASSWORD=""
if [ "$OBSERVABILITY" = "1" ]; then
  if [ -e "$SECRETS_DIR/otel_auth_password.txt" ]; then
    [ -s "$SECRETS_DIR/otel_auth_password.txt" ] \
      || die "otel_auth_password.txt 为空，无法作为 OpenObserve 密码源"
    OBS_PASSWORD="$(cat "$SECRETS_DIR/otel_auth_password.txt")"
  fi
  if [ -e "$SECRETS_DIR/openobserve.env" ]; then
    [ -s "$SECRETS_DIR/openobserve.env" ] \
      || die "openobserve.env 为空；请删除后重试，或从 otel_auth_password.txt 恢复"
    OBS_FILE_EMAIL="$(dotenv_value "ZO_ROOT_USER_EMAIL")"
    OBS_FILE_PASSWORD="$(dotenv_value "ZO_ROOT_USER_PASSWORD")"
    [ -n "$OBS_FILE_EMAIL" ] || die "openobserve.env 缺少 ZO_ROOT_USER_EMAIL"
    [ -n "$OBS_FILE_PASSWORD" ] || die "openobserve.env 缺少 ZO_ROOT_USER_PASSWORD"
    if [ -n "$OBS_EMAIL" ] && [ "$OBS_EMAIL" != "$OBS_FILE_EMAIL" ]; then
      die "openobserve.env 中的邮箱与当前配置不一致；请先人工确认或删除派生文件"
    fi
    OBS_EMAIL="$OBS_FILE_EMAIL"
    if [ -n "$OBS_PASSWORD" ] && [ "$OBS_FILE_PASSWORD" != "$OBS_PASSWORD" ]; then
      die "openobserve.env 中的密码与 otel_auth_password.txt 不一致；请先人工确认或删除派生文件"
    fi
  fi
  if [ -n "$LEGACY_ENV_FILE" ] && [ -f "$LEGACY_ENV_FILE" ]; then
    LEGACY_OBS_PASSWORD="$(legacy_env_value "ZO_OBSERVE_ROOT_USER_PASSWORD")"
  fi
  if [ -z "$OBS_PASSWORD" ]; then
    OBS_PASSWORD="$OBS_FILE_PASSWORD"
  fi
  if [ -z "$OBS_PASSWORD" ]; then
    OBS_PASSWORD="$LEGACY_OBS_PASSWORD"
  fi
  if [ "$OPENOBSERVE_VOLUME_EXISTS" = "1" ] && [ -z "$OBS_PASSWORD" ]; then
    die "检测到已有 OpenObserve 数据卷 ${OPENOBSERVE_VOLUME}，但找不到原管理员密码；请从 otel_auth_password.txt、openobserve.env 或旧 .env 恢复，不能生成新密码覆盖已有管理员"
  fi
  if [ -z "$OBS_PASSWORD" ]; then
    OBS_PASSWORD="$(generate_openobserve_password)" || die "生成 OpenObserve 密码失败"
    [ -n "$OBS_PASSWORD" ] || die "生成 OpenObserve 密码失败"
  fi
  OBS_EMAIL="${OBS_EMAIL:-admin@jimu.local}"
  case "$OBS_EMAIL$OBS_PASSWORD" in
    *$'\n'*|*$'\r'*) die "OpenObserve 邮箱或密码不能包含换行" ;;
  esac
fi

create_generated_secret() {
  local name="$1" value
  shift
  if [ -s "$SECRETS_DIR/$name" ] && [ "$FORCE" != "1" ]; then
    chmod 0600 "$SECRETS_DIR/$name"
    printf '   - %s 已存在，保留\n' "$name"
    return 0
  fi
  value="$("$@")" || die "生成 ${name} 失败"
  [ -n "$value" ] || die "生成 ${name} 为空"
  write_secret "$name" "$value"
}

create_admin_password() {
  if [ -n "${ADMIN_PASSWORD:-}" ]; then
    printf '%s' "$ADMIN_PASSWORD"
    return 0
  fi
  if [ -t 0 ]; then
    local value
    read -r -s -p "管理员初始密码（留空自动生成）: " value
    printf '\n' >&2
    if [ -n "$value" ]; then
      printf '%s' "$value"
      return 0
    fi
  fi
  generate_random_password
}

if [ "$MARIADB_VOLUME_EXISTS" = "1" ]; then
  for file in db_root_password.txt db_password.txt; do
    chmod 0600 "$SECRETS_DIR/$file"
    printf '   - %s 已有 MariaDB 数据卷，保留\n' "$file"
  done
else
  create_generated_secret "db_root_password.txt" generate_random_password
  create_generated_secret "db_password.txt" generate_random_password
fi
if [ "$PROTECT_SECURITY_SECRETS" = "1" ]; then
  for file in jwt_secret.txt encryption_key.txt audit_hash_secret.txt admin_password.txt; do
    chmod 0600 "$SECRETS_DIR/$file"
    printf '   - %s 已有 MariaDB 数据卷且未授权轮换，保留\n' "$file"
  done
else
  create_generated_secret "jwt_secret.txt" openssl rand -hex 32
  create_generated_secret "encryption_key.txt" generate_encryption_key
  create_generated_secret "audit_hash_secret.txt" openssl rand -hex 32
  create_generated_secret "admin_password.txt" create_admin_password
fi

if [ "$OBSERVABILITY" = "1" ]; then
  write_secret "otel_auth_password.txt" "$OBS_PASSWORD"

  if [ -f "$SECRETS_DIR/openobserve.env" ] && [ "$FORCE" != "1" ]; then
    chmod 0600 "$SECRETS_DIR/openobserve.env"
    printf '   - openobserve.env 已存在，保留\n'
  else
    write_secret "openobserve.env" "ZO_ROOT_USER_EMAIL=$(dotenv_quote "$OBS_EMAIL")
ZO_ROOT_USER_PASSWORD=$(dotenv_quote "$OBS_PASSWORD")"
  fi

  expected_token="$(printf '%s:%s' "$OBS_EMAIL" "$OBS_PASSWORD" | base64 | tr -d '\n')" \
    || die "生成 OpenObserve token 失败"
  [ -n "$expected_token" ] || die "生成 OpenObserve token 为空"
  if [ -s "$SECRETS_DIR/zo_auth_token.txt" ] && [ "$FORCE" != "1" ]; then
    chmod 0600 "$SECRETS_DIR/zo_auth_token.txt"
    [ "$(cat "$SECRETS_DIR/zo_auth_token.txt")" = "$expected_token" ] \
      || die "zo_auth_token.txt 与当前 OpenObserve 凭据不一致；请从 otel_auth_password.txt 重新派生或删除该文件"
    printf '   - zo_auth_token.txt 已存在，保留\n'
  else
    write_secret "zo_auth_token.txt" "$expected_token"
  fi
fi

echo "✅ secret 目录已就绪：$SECRETS_DIR"
if [ "$OBSERVABILITY" = "1" ]; then
  echo "   已包含 OpenObserve 管理员凭据派生文件"
fi
