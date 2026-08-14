#!/usr/bin/env bash
set -euo pipefail

COMPOSE=${COMPOSE:-"docker compose"}
HTTP_PORT=${HTTP_PORT:-18080}
MANAGEMENT_PORT=${MANAGEMENT_PORT:-9090}
CREATED_ENV=0
export HTTP_PORT
export MANAGEMENT_PORT

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
JIMU__AUTH__PUBLIC_REGISTRATION=false
ENV
fi

$COMPOSE config --quiet
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/server ./cmd/cli

$COMPOSE up -d --build

for endpoint in /livez /readyz; do
  ok=0
  for _ in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${MANAGEMENT_PORT}${endpoint}" >/dev/null; then
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

if curl -fsS "http://127.0.0.1:${HTTP_PORT}/debug/pprof/" >/dev/null 2>&1; then
  echo "public pprof endpoint is exposed" >&2
  exit 1
fi
