#!/usr/bin/env bash
# 从 .env / 环境变量生成 Docker Secrets 文件（./secrets/*.txt）
#
# 生成内容（与 docker-compose.yml 的 secrets 段一一对应）：
#   db_root_password.txt  <- DB_ROOT_PASSWORD
#   db_password.txt       <- DB_PASSWORD
#   jwt_secret.txt        <- JWT_SECRET
#   otel_auth_password.txt<- ZO_OBSERVE_ROOT_USER_PASSWORD（jimu server OTEL 认证）
#   zo_auth_token.txt     <- ZO_OBSERVE_AUTH_TOKEN（缺省 base64(email:password)）
#
# 安全：已存在的文件默认不覆盖；MAKE_SECRETS_FORCE=1 强制重新生成。
# 变量来源：优先环境变量（make include .env + export），其次直接加载项目根 .env
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${SECRETS_DIR:-$SCRIPT_DIR/../secrets}"

# 加载项目根 .env（若存在）；已通过环境变量传入的值不会被覆盖（set -a + source 后同值覆盖无碍）
if [ -f "$SCRIPT_DIR/../.env" ]; then
  set -a
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/../.env"
  set +a
fi

# 读取变量：环境变量 > 默认值
DB_ROOT_PASSWORD_VAL="${DB_ROOT_PASSWORD:-}"
DB_PASSWORD_VAL="${DB_PASSWORD:-}"
JWT_SECRET_VAL="${JWT_SECRET:-}"
ZO_EMAIL_VAL="${ZO_OBSERVE_ROOT_USER_EMAIL:-admin@jimu.local}"
ZO_PASSWORD_VAL="${ZO_OBSERVE_ROOT_USER_PASSWORD:-}"
ZO_TOKEN_VAL="${ZO_OBSERVE_AUTH_TOKEN:-}"

missing=""
for name in "DB_ROOT_PASSWORD:$DB_ROOT_PASSWORD_VAL" \
           "DB_PASSWORD:$DB_PASSWORD_VAL" \
           "JWT_SECRET:$JWT_SECRET_VAL" \
           "ZO_OBSERVE_ROOT_USER_PASSWORD:$ZO_PASSWORD_VAL"; do
  varname="${name%%:*}"; value="${name#*:}"
  [ -n "$value" ] || missing="$missing $varname"
done
if [ -n "$missing" ]; then
  echo "❌ 缺少以下变量（请在 .env 中填写后执行 make secrets）：$missing" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"

# 按文件生成：缺失的生成，已存在的跳过（除非 MAKE_SECRETS_FORCE=1 强制重生成）
gen() { # $1=文件名  $2=内容（字符串或引用）
  if [ -n "${MAKE_SECRETS_FORCE:-}" ]; then
    printf '%s' "$2" > "$OUT_DIR/$1"
    echo "   ↻ $1（强制重生成）"
  elif [ -f "$OUT_DIR/$1" ]; then
    echo "   - $1 已存在，跳过"
  else
    printf '%s' "$2" > "$OUT_DIR/$1"
    echo "   + $1"
  fi
}

gen db_root_password.txt "$DB_ROOT_PASSWORD_VAL"
gen db_password.txt "$DB_PASSWORD_VAL"
gen jwt_secret.txt "$JWT_SECRET_VAL"
# jimu server 的 OpenObserve 认证密码（OTEL_AUTH_PASSWORD_FILE 挂载）
gen otel_auth_password.txt "$ZO_PASSWORD_VAL"
# OTel Collector 的 Basic token（base64(email:password)）；自定义 ZO_OBSERVE_AUTH_TOKEN 时优先
if [ -n "$ZO_TOKEN_VAL" ]; then
  gen zo_auth_token.txt "$ZO_TOKEN_VAL"
else
  gen zo_auth_token.txt "$(printf '%s:%s' "$ZO_EMAIL_VAL" "$ZO_PASSWORD_VAL" | base64)"
fi

echo "✅ secrets 目录已就绪：$OUT_DIR"
echo "   内容来自 .env；修改密码后删除对应文件（或 MAKE_SECRETS_FORCE=1）重跑 make secrets，再重启 compose"
