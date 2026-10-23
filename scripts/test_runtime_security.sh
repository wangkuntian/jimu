#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# 端口用专用变量：调用方环境可能设置 HTTP_HOST_PORT=8080 并泄漏进来，
# 沿用同名变量会与本机正在运行的栈抢端口，隔离检查因而失去意义。
HTTP_HOST_PORT=${CHECK_HTTP_HOST_PORT:-18080}
MANAGEMENT_HOST_PORT=${CHECK_MANAGEMENT_HOST_PORT:-19090}
PROJECT_NAME="jimu-runtime-${RANDOM}${RANDOM}"
JIMU_IMAGE="${PROJECT_NAME}:latest"
WORKDIR=$(mktemp -d)
SECRET_DIR="$WORKDIR/secrets"

# 显式覆盖所有运行时输入，避免调用方环境把隔离检查接到本地数据卷或镜像上。
compose() {
  APP_ENV=dev \
  PROFILE=full \
  JIMU_SECRETS_DIR="$SECRET_DIR" \
  JIMU_IMAGE="$JIMU_IMAGE" \
  COMPOSE_MARIADB_VOLUME="${PROJECT_NAME}-mariadb" \
  COMPOSE_REDIS_VOLUME="${PROJECT_NAME}-redis" \
  JIMU_DB_HOST_PORT=0 \
  JIMU_REDIS_HOST_PORT=0 \
  HTTP_HOST_PORT="$HTTP_HOST_PORT" \
  MANAGEMENT_HOST_PORT="$MANAGEMENT_HOST_PORT" \
  docker compose --project-name "$PROJECT_NAME" --env-file /dev/null \
    -f "$ROOT_DIR/docker-compose.yml" "$@"
}

cleanup() {
  compose down --volumes --remove-orphans >/dev/null 2>&1 || true
  docker image rm "$JIMU_IMAGE" >/dev/null 2>&1 || true
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

mkdir -p "$SECRET_DIR"
chmod 0700 "$SECRET_DIR"
printf '%s' 'jimu-runtime-root-password' > "$SECRET_DIR/db_root_password.txt"
printf '%s' 'jimu-runtime-db-password' > "$SECRET_DIR/db_password.txt"
printf '%s' 'jimu-runtime-jwt-secret-012345678901234567890123456789' > "$SECRET_DIR/jwt_secret.txt"
printf '%s' '01234567890123456789012345678901' > "$SECRET_DIR/encryption_key.txt"
printf '%s' 'jimu-runtime-audit-secret-012345678901234567890123456789' > "$SECRET_DIR/audit_hash_secret.txt"
printf '%s' 'jimu-runtime-admin-password' > "$SECRET_DIR/admin_password.txt"

compose config --quiet
compose up -d --build

for endpoint in /livez /readyz; do
  ok=0
  for _ in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${MANAGEMENT_HOST_PORT}${endpoint}" >/dev/null; then
      ok=1
      break
    fi
    sleep 2
  done
  if [ "$ok" != "1" ]; then
    echo "management endpoint failed: ${endpoint}" >&2
    exit 1
  fi
done

if curl -fsS "http://127.0.0.1:${HTTP_HOST_PORT}/debug/pprof/" >/dev/null 2>&1; then
  echo "public pprof endpoint is exposed" >&2
  exit 1
fi
