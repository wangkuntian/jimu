#!/usr/bin/env bash
set -euo pipefail

# 端口用专用变量：本地 .env 会设 HTTP_HOST_PORT=8080 并经 make export 泄漏进来，
# 沿用同名变量会与本机正在运行的栈抢端口，隔离检查因而失去意义。
HTTP_HOST_PORT=${CHECK_HTTP_HOST_PORT:-18081}
MANAGEMENT_HOST_PORT=${CHECK_MANAGEMENT_HOST_PORT:-19091}
BASE_URL="http://127.0.0.1:${HTTP_HOST_PORT}/api/v1"
PROJECT_NAME="jimu-api-${RANDOM}${RANDOM}"
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
  COMPOSE_OTEL_AUTH_PASSWORD_FILE="$SECRET_DIR/otel_auth_password.txt" \
  COMPOSE_ZO_AUTH_TOKEN_FILE="$SECRET_DIR/zo_auth_token.txt" \
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
printf '%s\n' 'jimu-api-root-password' > "$SECRET_DIR/db_root_password.txt"
printf '%s\n' 'jimu-api-db-password' > "$SECRET_DIR/db_password.txt"
printf '%s\n' '01234567890123456789012345678901' > "$SECRET_DIR/jwt_secret.txt"
# compose 声明的其余 secret 也必须存在，否则干净检出（CI）里 server 的 bind mount 会直接失败
printf '%s\n' 'jimu-api-otel-password' > "$SECRET_DIR/otel_auth_password.txt"
printf '%s\n' 'jimu-api-zo-token' > "$SECRET_DIR/zo_auth_token.txt"
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

registered=0
for _ in $(seq 1 60); do
  status=$(curl -sS -o /tmp/jimu-register.json -w "%{http_code}" "${BASE_URL}/auth/register" -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"smoke_user","password":"secret123"}' || true)
  if [ "$status" = "200" ]; then
    grep -q '"code":0' /tmp/jimu-register.json
    registered=1
    break
  fi
  if [ "$status" = "409" ]; then
    grep -q '"code":2002' /tmp/jimu-register.json
    registered=1
    break
  fi
  sleep 2
done
if [ "$registered" != "1" ]; then
  cat /tmp/jimu-register.json
  exit 1
fi

LOGIN=$(curl -fsS "${BASE_URL}/auth/login" -X POST -H "Content-Type: application/json" -d '{"username":"smoke_user","password":"secret123"}')
ACCESS=$(printf "%s" "$LOGIN" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
REFRESH=$(printf "%s" "$LOGIN" | sed -n 's/.*"refresh_token":"\([^"]*\)".*/\1/p')
test -n "$ACCESS"
test -n "$REFRESH"

curl -fsS "${BASE_URL}/auth/refresh" -X POST -H "Content-Type: application/json" -d "{\"refresh_token\":\"${REFRESH}\"}" >/tmp/jimu-refresh.json

missing_status=$(curl -sS -o /tmp/jimu-logout-missing-token.json -w "%{http_code}" "${BASE_URL}/auth/logout" -X POST || true)
if [ "$missing_status" != "401" ] || ! grep -q '"code":1002' /tmp/jimu-logout-missing-token.json; then
  cat /tmp/jimu-logout-missing-token.json
  exit 1
fi

curl -fsS "${BASE_URL}/auth/logout" -X POST -H "Authorization: Bearer ${ACCESS}" >/tmp/jimu-logout.json

grep -q '"code":0' /tmp/jimu-refresh.json
grep -q '"code":0' /tmp/jimu-logout.json
