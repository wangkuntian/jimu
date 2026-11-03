#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

for command in docker go curl; do
  command -v "$command" >/dev/null 2>&1 || fail "$command is not available"
done
docker compose version >/dev/null 2>&1 || fail "docker compose is not available"

if [ -z "${JIMU_BIN:-}" ]; then
  fail "JIMU_BIN is required"
fi
if [ ! -x "$JIMU_BIN" ]; then
  fail "JIMU_BIN is not executable: $JIMU_BIN"
fi

WORKDIR="${WORKDIR:-$(mktemp -d)}"
mkdir -p "$WORKDIR"
WORKDIR="$(cd "$WORKDIR" && pwd)"
basic_dir="$WORKDIR/basic-$$"
auth_dir="$WORKDIR/auth-$$"
basic_pid=""
auth_started=0

cleanup() {
  status=$?
  trap - EXIT
  if [ -n "$basic_pid" ] && kill -0 "$basic_pid" 2>/dev/null; then
    kill "$basic_pid" 2>/dev/null || true
    wait "$basic_pid" 2>/dev/null || true
  fi
  if [ "$auth_started" = 1 ] && [ -d "$auth_dir/demo" ]; then
    if [ "$status" -ne 0 ]; then
      (cd "$auth_dir/demo" && docker compose logs --no-color --tail=200 || true) >&2
    fi
    (cd "$auth_dir/demo" && make clean >/dev/null 2>&1 || true)
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

mkdir -p "$basic_dir"
(
  cd "$basic_dir"
  "$JIMU_BIN" new demo --template=basic-api
)
cd "$basic_dir/demo"
"$JIMU_BIN" check .
"$JIMU_BIN" upgrade . --check
make build
HTTP_PORT=18080 MANAGEMENT_PORT=19090 ./bin/server >"$basic_dir/server.log" 2>&1 &
basic_pid=$!
if ! wait_http "http://127.0.0.1:19090/livez"; then
  cat "$basic_dir/server.log" >&2
  fail "basic-api did not become ready"
fi
curl -fsS "http://127.0.0.1:18080/api/v1/ping" >/dev/null 2>&1 || {
  cat "$basic_dir/server.log" >&2
  fail "basic-api endpoint failed: /api/v1/ping"
}
for path in /livez /readyz; do
  curl -fsS "http://127.0.0.1:19090${path}" >/dev/null 2>&1 || {
    cat "$basic_dir/server.log" >&2
    fail "basic-api endpoint failed: $path"
  }
done
kill "$basic_pid" 2>/dev/null || true
wait "$basic_pid" 2>/dev/null || true
basic_pid=""

mkdir -p "$auth_dir"
(
  cd "$auth_dir"
  "$JIMU_BIN" new demo --template=auth-api
)
cd "$auth_dir/demo"
"$JIMU_BIN" check .
"$JIMU_BIN" upgrade . --check
auth_started=1
AUTH_API_PORT=18081 AUTH_API_MANAGEMENT_PORT=19091 make up
AUTH_API_PORT=18081 AUTH_API_MANAGEMENT_PORT=19091 make verify
make clean
auth_started=0

printf 'PASS: clean-room basic-api and auth-api\n'
