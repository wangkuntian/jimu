#!/usr/bin/env bash
# 5 个形态（profile）入口的构建门禁 + 可选启动/健康检查。
#
# 默认（不设 JIMU_PROFILES_SMOKE=1）：只构建 —— 这是硬要求，任一形态构建失败即非零退出；
# 每个形态仍打印一行 SKIP，说明启动与健康检查需要 DB+Redis，不静默跳过。
#
# JIMU_PROFILES_SMOKE=1：构建后逐个以 APP_ENV=dev 启动，轮询管理端 readiness
# （GET /readyz 检查 DB+Redis 可达，即内核的 /health 语义）直到就绪，然后关停；
# 任一形态启动失败或超时即非零退出。为避开本机已运行的服务，默认用隔离端口
# 18080/19090（HTTP_PORT/MANAGEMENT_PORT），可用 JIMU_PROFILES_HTTP_PORT、
# JIMU_PROFILES_MGMT_PORT、JIMU_PROFILES_READY_TIMEOUT_SEC 覆盖。
#
# DB/Redis 连接沿用应用配置与环境变量（DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/REDIS_ADDR…）。
# 注意：环境里设了 ADMIN_PASSWORD 时，启动会先执行结构性种子（见 internal/profiles
# 的 StructuralSeed），需要先跑过 `jimu migrate up`；未设置时跳过种子。
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

PROFILES=(full minimal saas enterprise machine)
HTTP_PORT="${JIMU_PROFILES_HTTP_PORT:-18080}"
MGMT_PORT="${JIMU_PROFILES_MGMT_PORT:-19090}"
READY_TIMEOUT_SEC="${JIMU_PROFILES_READY_TIMEOUT_SEC:-30}"

BIN_DIR="$(mktemp -d "${TMPDIR:-/tmp}/jimu-profiles-XXXXXX")"
LOG_DIR="$BIN_DIR/logs"
mkdir -p "$LOG_DIR"
trap 'rm -rf "$BIN_DIR"' EXIT

echo "==> 构建 5 个形态入口"
for p in "${PROFILES[@]}"; do
  go build -o "$BIN_DIR/jimu-$p" "./profiles/$p"
  echo "    ok  profiles/$p"
done

if [[ "${JIMU_PROFILES_SMOKE:-}" != "1" ]]; then
  for p in "${PROFILES[@]}"; do
    echo "SKIP profiles/$p 启动与健康检查：需要 DB+Redis（设置 JIMU_PROFILES_SMOKE=1 启用；APP_ENV=dev，端口 ${HTTP_PORT}/${MGMT_PORT}）"
  done
  echo "✅ 5 个形态入口构建通过（启动与健康检查已跳过：未设置 JIMU_PROFILES_SMOKE=1）"
  exit 0
fi

echo "==> 启动与健康检查（APP_ENV=dev，HTTP ${HTTP_PORT} / management ${MGMT_PORT}）"
for p in "${PROFILES[@]}"; do
  log="$LOG_DIR/$p.log"
  APP_ENV=dev HTTP_PORT="$HTTP_PORT" MANAGEMENT_PORT="$MGMT_PORT" MANAGEMENT_HOST=127.0.0.1 \
    "$BIN_DIR/jimu-$p" >"$log" 2>&1 &
  pid=$!

  ready=0
  for _ in $(seq 1 "$READY_TIMEOUT_SEC"); do
    if ! kill -0 "$pid" 2>/dev/null; then
      break # 进程已退出：直接看日志
    fi
    if curl -fsS "http://127.0.0.1:${MGMT_PORT}/readyz" >/dev/null 2>&1; then
      ready=1
      break
    fi
    sleep 1
  done

  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true

  if [[ "$ready" != "1" ]]; then
    echo "❌ profiles/$p 未在 ${READY_TIMEOUT_SEC}s 内就绪，日志：" >&2
    sed 's/^/    /' "$log" >&2
    exit 1
  fi
  echo "    ok  profiles/$p 就绪（GET /readyz）"
done

echo "✅ 5 个形态入口构建 + 启动 + 健康检查通过"
