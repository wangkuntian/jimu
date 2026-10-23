#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIR="$(mktemp -d)"
PROJECT_DIR="$WORKDIR/project"
SECRETS_DIR="$PROJECT_DIR/secrets"
DOCKER_LOG="$WORKDIR/docker.log"
INIT_LOG="$WORKDIR/init.log"
HOOK_LOG="$WORKDIR/hook.log"

cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

mkdir -p "$WORKDIR/bin" "$PROJECT_DIR/scripts" "$PROJECT_DIR/deploy/openobserve"
cp "$ROOT_DIR/scripts/observability.sh" "$PROJECT_DIR/scripts/observability.sh"

cat > "$WORKDIR/bin/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$DOCKER_LOG"
DOCKER
chmod +x "$WORKDIR/bin/docker"

cat > "$PROJECT_DIR/scripts/init-secrets.sh" <<'INIT'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$INIT_LOG"
mkdir -p "$SECRETS_DIR"
printf '%s' 'test-password' > "$SECRETS_DIR/otel_auth_password.txt"
printf '%s\n' \
  "ZO_ROOT_USER_EMAIL='ops@example.com'" \
  "ZO_ROOT_USER_PASSWORD='test-password'" > "$SECRETS_DIR/openobserve.env"
INIT
chmod +x "$PROJECT_DIR/scripts/init-secrets.sh"

cat > "$PROJECT_DIR/deploy/openobserve/init-dashboard.sh" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail
printf 'dashboard:%s:%s\n' "$ZO_EMAIL" "$ZO_PASSWORD" >> "$HOOK_LOG"
HOOK
cat > "$PROJECT_DIR/deploy/openobserve/sync-alerts.sh" <<'HOOK'
#!/usr/bin/env bash
set -euo pipefail
printf 'alerts:%s:%s\n' "$ZO_EMAIL" "$ZO_PASSWORD" >> "$HOOK_LOG"
HOOK
chmod +x "$PROJECT_DIR"/deploy/openobserve/*.sh

run_observability() {
  PATH="$WORKDIR/bin:$PATH" \
  JIMU_SECRETS_DIR="$SECRETS_DIR" \
  DOCKER_LOG="$DOCKER_LOG" \
  INIT_LOG="$INIT_LOG" \
  HOOK_LOG="$HOOK_LOG" \
  bash "$PROJECT_DIR/scripts/observability.sh" "$@"
}

: > "$DOCKER_LOG"
run_observability status
grep -Fq \
  "compose --env-file /dev/null -f $PROJECT_DIR/docker-compose.yml -f $PROJECT_DIR/compose.observability.yml ps openobserve otel-collector" \
  "$DOCKER_LOG"
[ ! -e "$SECRETS_DIR/otel_auth_password.txt" ]

: > "$DOCKER_LOG"
run_observability stop
grep -Fq \
  "compose --env-file /dev/null -f $PROJECT_DIR/docker-compose.yml -f $PROJECT_DIR/compose.observability.yml down openobserve otel-collector" \
  "$DOCKER_LOG"
[ ! -e "$SECRETS_DIR/otel_auth_password.txt" ]

run_observability start
[ "$(cat "$INIT_LOG")" = "--observability --no-legacy" ]
grep -Fq \
  "compose --env-file /dev/null -f $PROJECT_DIR/docker-compose.yml -f $PROJECT_DIR/compose.observability.yml up -d openobserve otel-collector" \
  "$DOCKER_LOG"
grep -Fq 'dashboard:ops@example.com:test-password' "$HOOK_LOG"
grep -Fq 'alerts:ops@example.com:test-password' "$HOOK_LOG"

echo "observability script contract passed"
