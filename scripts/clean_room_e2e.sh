#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

for command in docker go curl; do
  command -v "$command" >/dev/null 2>&1 || fail "$command is not available"
done

if [ -z "${JIMU_BIN:-}" ]; then
  fail "JIMU_BIN is required"
fi
if [ ! -x "$JIMU_BIN" ]; then
  fail "JIMU_BIN is not executable: $JIMU_BIN"
fi

WORKDIR="${WORKDIR:-$(mktemp -d)}"
mkdir -p "$WORKDIR"
WORKDIR="$(cd "$WORKDIR" && pwd)"
min_dir="$WORKDIR/minimal-$$"
auth_dir="$WORKDIR/auth-$$"
min_pid=""
auth_started=0

cleanup() {
  status=$?
  trap - EXIT
  if [ -n "$min_pid" ] && kill -0 "$min_pid" 2>/dev/null; then
    kill "$min_pid" 2>/dev/null || true
    wait "$min_pid" 2>/dev/null || true
  fi
  if [ "$auth_started" = 1 ] && [ -d "$auth_dir/demo" ] && [ -n "${auth_pid:-}" ]; then
    if [ "$status" -ne 0 ]; then
      tail -n 200 "$auth_dir/server.log" >&2 || true
    fi
    kill "$auth_pid" 2>/dev/null || true
    wait "$auth_pid" 2>/dev/null || true
  fi
  exit "$status"
}
trap cleanup EXIT

wait_http() {
  local url="$1"
  local attempts="${2:-120}"
  for _ in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

mkdir -p "$min_dir"
(
  cd "$min_dir"
  "$JIMU_BIN" new demo --template=minimal
)
cd "$min_dir/demo"
"$JIMU_BIN" check .
"$JIMU_BIN" upgrade . --check
make build
HTTP_PORT=18080 MANAGEMENT_PORT=19090 ./bin/server >"$min_dir/server.log" 2>&1 &
min_pid=$!
if ! wait_http "http://127.0.0.1:19090/livez"; then
  cat "$min_dir/server.log" >&2
  fail "minimal did not become ready"
fi
curl -fsS "http://127.0.0.1:18080/api/v1/ping" >/dev/null 2>&1 || {
  cat "$min_dir/server.log" >&2
  fail "minimal endpoint failed: /api/v1/ping"
}
for path in /livez /readyz; do
  curl -fsS "http://127.0.0.1:19090${path}" >/dev/null 2>&1 || {
    cat "$min_dir/server.log" >&2
    fail "minimal endpoint failed: $path"
  }
done
kill "$min_pid" 2>/dev/null || true
wait "$min_pid" 2>/dev/null || true
min_pid=""

mkdir -p "$auth_dir"
(
  cd "$auth_dir"
  "$JIMU_BIN" new demo --template=auth
)
cd "$auth_dir/demo"
"$JIMU_BIN" check .
"$JIMU_BIN" upgrade . --check
make build
auth_started=1
DB_HOST=127.0.0.1 DB_PORT=13306 DB_USER=jimu DB_PASSWORD=jimu DB_NAME=jimu \
  REDIS_ADDR=127.0.0.1:16379 \
  APP_ENV=dev HTTP_PORT=18081 MANAGEMENT_PORT=19091 ./bin/server >"$auth_dir/server.log" 2>&1 &
auth_pid=$!
wait_http "http://127.0.0.1:19091/livez" || fail "auth did not become ready"
curl -fsS "http://127.0.0.1:18081/api/v1/ping" >/dev/null || fail "auth ping failed"
kill "$auth_pid" 2>/dev/null || true
wait "$auth_pid" 2>/dev/null || true
auth_started=0

printf 'PASS: clean-room minimal and auth presets\n'
