#!/usr/bin/env bash
# OpenObserve 监控栈管理脚本
#
# 管理 observability 叠加文件中的服务（openobserve + otel-collector），
# 不影响 mariadb / redis / server 等业务容器。
#
# 用法:
#   ./scripts/observability.sh start     # 启动服务并初始化 dashboard；配置 webhook 时同步默认告警
#   ./scripts/observability.sh stop      # 停止并删除监控栈容器（保留业务容器与数据卷）
#   ./scripts/observability.sh restart   # 重启监控栈
#   ./scripts/observability.sh status    # 查看监控栈容器状态
#
# 环境变量：
#   JIMU_SECRETS_DIR / ZO_OBSERVE_HTTP_PORT
#   ZO_ALERT_WEBHOOK_URL（可选：默认告警的 webhook 通知地址；未设置则跳过告警同步）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
SECRETS_DIR="${JIMU_SECRETS_DIR:-$ROOT_DIR/secrets}"

COMPOSE=(
  docker compose --env-file /dev/null
  -f "$ROOT_DIR/docker-compose.yml"
  -f "$ROOT_DIR/compose.observability.yml"
)
SERVICES=(openobserve otel-collector)
ZO_HTTP_PORT="${ZO_OBSERVE_HTTP_PORT:-5080}"
ZO_HTTP="http://127.0.0.1:$ZO_HTTP_PORT"

start() {
  local env_file="$SECRETS_DIR/openobserve.env"
  local zo_email zo_password

  if [ ! -s "$env_file" ]; then
    echo "==> 初始化 observability secret：$SECRETS_DIR"
    SECRETS_DIR="$SECRETS_DIR" \
      "$ROOT_DIR/scripts/init-secrets.sh" --observability --no-legacy
  fi
  [ -s "$env_file" ] \
    || { echo "❌ 缺少可用的 $env_file；请通过 make up-observability 重新初始化观测 Secret" >&2; exit 1; }
  # openobserve.env 由初始化脚本以 0600 生成，值为 shell/dotenv 兼容引用。
  set -a
  # shellcheck disable=SC1090
  . "$env_file"
  set +a
  zo_email="${ZO_ROOT_USER_EMAIL:-}"
  zo_password="${ZO_ROOT_USER_PASSWORD:-}"
  [ -n "$zo_email" ] && [ -n "$zo_password" ] \
    || { echo "❌ $env_file 缺少 ZO_ROOT_USER_EMAIL 或 ZO_ROOT_USER_PASSWORD" >&2; exit 1; }

  echo "==> 启动监控栈服务：${SERVICES[*]}"
  "${COMPOSE[@]}" up -d "${SERVICES[@]}"
  echo "==> 初始化 dashboard（幂等）"
  ZO_HTTP="$ZO_HTTP" \
    ZO_EMAIL="$zo_email" \
    ZO_PASSWORD="$zo_password" \
    "$ROOT_DIR/deploy/openobserve/init-dashboard.sh"
  echo "==> 同步默认告警（幂等，需 ZO_ALERT_WEBHOOK_URL）"
  ZO_HTTP="$ZO_HTTP" \
    ZO_EMAIL="$zo_email" \
    ZO_PASSWORD="$zo_password" \
    "$ROOT_DIR/deploy/openobserve/sync-alerts.sh"
  echo "✅ OpenObserve:   $ZO_HTTP （默认账号 ${zo_email}）"
  echo "   OTLP gRPC:    127.0.0.1:${ZO_OBSERVE_GRPC_PORT:-5081}"
  echo "   启用应用推送：make up-observability"
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
