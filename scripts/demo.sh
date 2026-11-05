#!/usr/bin/env bash
# 全栈 demo：用独立 Compose 项目启动 full server，再用 Vite 启动 jimu-web。
#
# 默认端口刻意避开常规开发栈：demo backend 18080/19090，MySQL 23306，Redis 26379，
# 前端 5173。所有值都可用 JIMU_DEMO_* 环境变量覆盖。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WEB_DIR="${JIMU_WEB_DIR:-$ROOT/../jimu-web}"
SECRETS_DIR="${JIMU_SECRETS_DIR:-$ROOT/secrets}"

DEMO_PROJECT="${JIMU_DEMO_PROJECT:-jimu-demo}"
HTTP_PORT="${JIMU_DEMO_HTTP_PORT:-18080}"
MGMT_PORT="${JIMU_DEMO_MANAGEMENT_PORT:-19090}"
DB_PORT="${JIMU_DEMO_DB_PORT:-23306}"
REDIS_PORT="${JIMU_DEMO_REDIS_PORT:-26379}"
WEB_PORT="${JIMU_DEMO_WEB_PORT:-5173}"
READY_TIMEOUT="${JIMU_DEMO_READY_TIMEOUT:-180}"

for cmd in docker curl bun openssl; do
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "❌ 缺少 ${cmd}；请先安装 Docker Compose v2、curl、Bun 和 openssl" >&2
    exit 1
  }
done
docker compose version >/dev/null 2>&1 || {
  echo "❌ 需要 Docker Compose v2" >&2
  exit 1
}

[ -d "$WEB_DIR" ] && [ -f "$WEB_DIR/package.json" ] || {
  echo "❌ 找不到 jimu-web：${WEB_DIR}（可用 JIMU_WEB_DIR 覆盖）" >&2
  exit 1
}

if ! grep -q 'JIMU_API_PROXY_TARGET' "$WEB_DIR/vite.config.ts"; then
  echo "❌ jimu-web 缺少 JIMU_API_PROXY_TARGET 支持；请更新到包含 B2 demo 的提交" >&2
  exit 1
fi

check_port() {
  local port="$1" name="$2" var="$3"
  if (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
    echo "❌ ${name}端口 ${port} 已被占用；用 ${var}=<端口> 覆盖" >&2
    exit 1
  fi
}
check_port "$WEB_PORT" "UI" "JIMU_DEMO_WEB_PORT"
check_port "$HTTP_PORT" "HTTP API" "JIMU_DEMO_HTTP_PORT"
check_port "$MGMT_PORT" "管理端" "JIMU_DEMO_MANAGEMENT_PORT"
check_port "$DB_PORT" "MariaDB" "JIMU_DEMO_DB_PORT"
check_port "$REDIS_PORT" "Redis" "JIMU_DEMO_REDIS_PORT"

export COMPOSE_PROJECT_NAME="$DEMO_PROJECT"
export COMPOSE_MARIADB_VOLUME="${JIMU_DEMO_MARIADB_VOLUME:-$DEMO_PROJECT-mariadb-data}"
export COMPOSE_REDIS_VOLUME="${JIMU_DEMO_REDIS_VOLUME:-$DEMO_PROJECT-redis-data}"
export JIMU_SECRETS_DIR="$SECRETS_DIR"
export JIMU_DB_HOST_PORT="$DB_PORT"
export JIMU_REDIS_HOST_PORT="$REDIS_PORT"
export HTTP_HOST_PORT="$HTTP_PORT"
export MANAGEMENT_HOST_PORT="$MGMT_PORT"
export PROFILE=full

web_pid=""
started=0
cleanup() {
  local status=$?
  trap - EXIT INT TERM HUP
  trap '' HUP
  if [ -n "$web_pid" ] && kill -0 "$web_pid" 2>/dev/null; then
    kill "$web_pid" 2>/dev/null || true
    wait "$web_pid" 2>/dev/null || true
  fi
  if [ "$started" = "1" ]; then
    echo "==> 停止 demo Compose 栈"
    ( cd "$ROOT" && make down ) || true
  fi
  exit "$status"
}
trap cleanup EXIT INT TERM HUP

echo "==> 启动 demo 后端（project=${DEMO_PROJECT}，HTTP=${HTTP_PORT}，management=${MGMT_PORT}）"
started=1
( cd "$ROOT" && make up )

echo "==> 等待后端 /readyz"
deadline=$((SECONDS + READY_TIMEOUT))
until curl -fsS "http://127.0.0.1:${MGMT_PORT}/readyz" >/dev/null 2>&1; do
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "❌ 后端在 ${READY_TIMEOUT}s 内未就绪；查看日志：make logs" >&2
    exit 1
  fi
  sleep 2
done

if [ "${JIMU_DEMO_E2E:-0}" = "1" ]; then
  echo "==> 运行真实后端 E2E（Playwright 启动 Vite）"
  ( cd "$WEB_DIR" && bun install --frozen-lockfile )
  (
    cd "$WEB_DIR"
    E2E_REAL_BACKEND=1 \
      E2E_ADMIN_PASSWORD="$(cat "$SECRETS_DIR/admin_password.txt")" \
      JIMU_API_PROXY_TARGET="http://127.0.0.1:${HTTP_PORT}" \
      bun run test:e2e e2e/real-backend.spec.ts
  )
  exit 0
fi

echo "==> 启动 jimu-web（Vite，端口 ${WEB_PORT}）"
( cd "$WEB_DIR" && bun install --frozen-lockfile )
(
  cd "$WEB_DIR"
  JIMU_API_PROXY_TARGET="http://127.0.0.1:${HTTP_PORT}" \
    bun run dev -- --host 127.0.0.1 --port "$WEB_PORT"
) &
web_pid=$!

cat <<EOF

────────────────────────────────────────
  Jimu full-stack demo
────────────────────────────────────────
  Demo UI:       http://127.0.0.1:${WEB_PORT}
  Backend API:   http://127.0.0.1:${HTTP_PORT}/api/v1
  Admin user:    admin
  Admin password: ${SECRETS_DIR}/admin_password.txt

  演示路径：admin 登录 → 租户/套餐 → 用户/角色 → 账号安全
  Ctrl-C 会同时停止前端和 demo Compose 栈。
────────────────────────────────────────

EOF

wait "$web_pid"
