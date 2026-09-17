#!/usr/bin/env bash
set -euo pipefail

# 端口用专用变量：本地 .env 会设 HTTP_HOST_PORT=8080 并经 make export 泄漏进来，
# 沿用同名变量会与本机正在运行的栈抢端口，隔离检查因而失去意义。
HTTP_HOST_PORT=${CHECK_HTTP_HOST_PORT:-18080}
MANAGEMENT_HOST_PORT=${CHECK_MANAGEMENT_HOST_PORT:-19090}
PROJECT_NAME="jimu-runtime-${RANDOM}${RANDOM}"
WORKDIR=$(mktemp -d)
ENV_FILE="$WORKDIR/.env"
SECRET_DIR="$WORKDIR/secrets"

# compose 显式覆盖所有被 docker-compose.yml 插值的变量：
# 通过 make 调用时，Makefile 的 include .env + export 会把本地 .env（含 COMPOSE_*_VOLUME）注入子进程，
# 而 shell 环境优先级高于 --env-file，会让「隔离」检查挂到本地数据卷/本地配置上。
# 这里强制指向本项目的临时卷与临时 secret，隔离性不依赖调用方的环境。
compose() {
  APP_ENV=dev \
  COMPOSE_MARIADB_VOLUME="${PROJECT_NAME}-mariadb" \
  COMPOSE_REDIS_VOLUME="${PROJECT_NAME}-redis" \
  COMPOSE_OPENOBSERVE_VOLUME="${PROJECT_NAME}-openobserve" \
  COMPOSE_DB_ROOT_PASSWORD_FILE="$SECRET_DIR/db_root_password.txt" \
  COMPOSE_DB_PASSWORD_FILE="$SECRET_DIR/db_password.txt" \
  COMPOSE_JWT_SECRET_FILE="$SECRET_DIR/jwt_secret.txt" \
  COMPOSE_DB_HOST=mariadb \
  COMPOSE_DB_PORT=3306 \
  COMPOSE_DB_USER=jimu \
  COMPOSE_DB_NAME=jimu \
  COMPOSE_REDIS_ADDR=redis:6379 \
  COMPOSE_REDIS_DB=0 \
  HTTP_HOST_PORT="$HTTP_HOST_PORT" \
  MANAGEMENT_HOST_PORT="$MANAGEMENT_HOST_PORT" \
  docker compose --project-name "$PROJECT_NAME" --env-file "$ENV_FILE" "$@"
}

cleanup() {
  compose down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

mkdir -p "$SECRET_DIR"
printf '%s\n' 'jimu-runtime-root-password' > "$SECRET_DIR/db_root_password.txt"
printf '%s\n' 'jimu-runtime-db-password' > "$SECRET_DIR/db_password.txt"
printf '%s\n' '01234567890123456789012345678901' > "$SECRET_DIR/jwt_secret.txt"
cat > "$ENV_FILE" <<ENV
APP_ENV=dev
HTTP_HOST_PORT=$HTTP_HOST_PORT
MANAGEMENT_HOST_PORT=$MANAGEMENT_HOST_PORT
COMPOSE_MARIADB_VOLUME=${PROJECT_NAME}-mariadb
COMPOSE_REDIS_VOLUME=${PROJECT_NAME}-redis
COMPOSE_DB_ROOT_PASSWORD_FILE=$SECRET_DIR/db_root_password.txt
COMPOSE_DB_PASSWORD_FILE=$SECRET_DIR/db_password.txt
COMPOSE_JWT_SECRET_FILE=$SECRET_DIR/jwt_secret.txt
ENV

compose config --quiet
compose up -d --build
compose exec -T server ./jimu migrate up

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
