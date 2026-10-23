#!/usr/bin/env bash
# 本地开发：启动依赖，迁移/种子，随后用 Air 热重载服务端。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PROFILE="${PROFILE:-full}"
ENV="${ENV:-dev}"
JIMU_SECRETS_DIR="${JIMU_SECRETS_DIR:-$ROOT/secrets}"
export JIMU_SECRETS_DIR
export SECRETS_DIR="$JIMU_SECRETS_DIR"

cd "$ROOT"

./scripts/init-secrets.sh --no-legacy

docker compose --env-file /dev/null -f docker-compose.yml up -d --wait mariadb redis

export APP_ENV="$ENV"
export DB_HOST=127.0.0.1
export REDIS_ADDR=127.0.0.1:6379
export DB_PASSWORD_FILE="$JIMU_SECRETS_DIR/db_password.txt"
export JWT_SECRET_FILE="$JIMU_SECRETS_DIR/jwt_secret.txt"
export ENCRYPTION_KEY_FILE="$JIMU_SECRETS_DIR/encryption_key.txt"
export AUDIT_HASH_SECRET_FILE="$JIMU_SECRETS_DIR/audit_hash_secret.txt"
export ADMIN_PASSWORD_FILE="$JIMU_SECRETS_DIR/admin_password.txt"

JIMU_OVERLAY="$(go run ./tools/profileoverlay "$PROFILE")"
[ -n "$JIMU_OVERLAY" ] || { echo "❌ 生成 PROFILE overlay 失败" >&2; exit 1; }
export JIMU_OVERLAY

go run -overlay="$JIMU_OVERLAY" ./cmd/cli migrate up
go run -overlay="$JIMU_OVERLAY" ./cmd/cli seed

command -v air >/dev/null 2>&1 || {
  echo "air 未安装，运行: go install github.com/air-verse/air@latest" >&2
  exit 1
}

exec air
