#!/usr/bin/env bash
set -euo pipefail

HTTP_HOST_PORT=${HTTP_HOST_PORT:-18080}
MANAGEMENT_HOST_PORT=${MANAGEMENT_HOST_PORT:-19090}
PROJECT_NAME="jimu-runtime-${RANDOM}${RANDOM}"
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
