#!/usr/bin/env bash
set -euo pipefail

COMPOSE=${COMPOSE:-"docker compose"}
HTTP_PORT=${HTTP_PORT:-18081}
BASE_URL="http://127.0.0.1:${HTTP_PORT}/api/v1"
CREATED_ENV=0
export HTTP_PORT

cleanup() {
  $COMPOSE down
  if [ "$CREATED_ENV" = "1" ]; then
    rm -f .env
  fi
}
trap cleanup EXIT

if [ ! -f .env ]; then
  CREATED_ENV=1
  cat > .env <<'ENV'
JIMU_ENV=prod
DB_ROOT_PASSWORD=jimu-root-change-me
DB_USER=jimu
DB_PASSWORD=jimu-db-change-me
DB_DATABASE=jimu
JIMU__DB__HOST=mariadb
JIMU__DB__PORT=3306
JIMU__DB__USER=jimu
JIMU__DB__PASSWORD=jimu-db-change-me
JIMU__REDIS__ADDR=redis:6379
JIMU__AUTH__JWT_SECRET=01234567890123456789012345678901
JIMU__AUTH__PUBLIC_REGISTRATION=true
JIMU__AUTH__LOGIN_RATE_LIMIT=100
JIMU__AUTH__REGISTER_RATE_LIMIT=100
ENV
fi

$COMPOSE up -d --build
$COMPOSE exec -T server ./jimu migrate up

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
