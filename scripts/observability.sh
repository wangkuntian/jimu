#!/usr/bin/env bash
# OpenObserve 监控栈管理脚本
#
# 管理 observability profile 下的服务（openobserve + otel-collector），
# 不影响 mariadb / redis / server 等业务容器。
#
# 用法:
#   ./scripts/observability.sh start     # 启动 openobserve + otel-collector 并初始化 dashboard
#   ./scripts/observability.sh stop      # 停止并删除监控栈容器（保留业务容器与数据卷）
#   ./scripts/observability.sh restart   # 重启监控栈
#   ./scripts/observability.sh status    # 查看监控栈容器状态
#
# 环境变量（来自 .env，缺省同 docker-compose.yml）：
#   ZO_OBSERVE_HTTP_PORT / ZO_OBSERVE_ROOT_USER_EMAIL / ZO_OBSERVE_ROOT_USER_PASSWORD
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# 加载项目根 .env（若存在）
if [ -f "$ROOT_DIR/.env" ]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
  set +a
fi

COMPOSE=(docker compose --profile observability)
SERVICES=(openobserve otel-collector)
ZO_HTTP_PORT="${ZO_OBSERVE_HTTP_PORT:-5080}"
ZO_HTTP="http://127.0.0.1:$ZO_HTTP_PORT"

start() {
  echo "==> 启动监控栈服务：${SERVICES[*]}"
  "${COMPOSE[@]}" up -d "${SERVICES[@]}"
  echo "==> 初始化 dashboard（幂等）"
  ZO_HTTP="$ZO_HTTP" \
    ZO_EMAIL="${ZO_OBSERVE_ROOT_USER_EMAIL:-admin@jimu.local}" \
    ZO_PASSWORD="${ZO_OBSERVE_ROOT_USER_PASSWORD:-}" \
    "$ROOT_DIR/deploy/openobserve/init-dashboard.sh"
  echo "✅ OpenObserve:   $ZO_HTTP （默认账号 ${ZO_OBSERVE_ROOT_USER_EMAIL:-admin@jimu.local}）"
  echo "   OTLP gRPC:    127.0.0.1:${ZO_OBSERVE_GRPC_PORT:-5081}"
  echo "   启用应用推送：OTEL_ENABLED=true make compose-up"
}

stop() {
  echo "==> 停止并删除监控栈（不影响 mariadb/redis/server）"
  "${COMPOSE[@]}" down "${SERVICES[@]}"
  echo "✅ 监控栈已停止"
}

restart() {
  stop
  start
}

status() {
  "${COMPOSE[@]}" ps "${SERVICES[@]}"
}

case "${1:-start}" in
  start)   start ;;
  stop)    stop ;;
  restart) restart ;;
  status)  status ;;
  *)
    echo "用法: $0 {start|stop|restart|status}" >&2
    exit 1
    ;;
esac