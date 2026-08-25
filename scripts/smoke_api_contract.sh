#!/usr/bin/env bash
set -euo pipefail

HTTP_HOST_PORT=${HTTP_HOST_PORT:-18081}
MANAGEMENT_HOST_PORT=${MANAGEMENT_HOST_PORT:-19091}
BASE_URL="http://127.0.0.1:${HTTP_HOST_PORT}/api/v1"
PROJECT_NAME="jimu-api-${RANDOM}${RANDOM}"
WORKDIR=$(mktemp -d)
ENV_FILE="$WORKDIR/.env"
SECRET_DIR="$WORKDIR/secrets"

compose() {
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
