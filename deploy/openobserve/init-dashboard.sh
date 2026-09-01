#!/usr/bin/env bash
# OpenObserve 默认 dashboard 初始化入口（兼容旧用法）
#
# 等同执行 sync-dashboard.sh：等待 OpenObserve 就绪后，以 git 中
# deploy/openobserve/dashboards/*.json 为准创建/重建 dashboard（幂等）。
# 面板配置在 dashboards/*.json 中维护，UI 手工调整可用
# ./deploy/openobserve/sync-dashboard.sh --export <名称> 同步回 git。
#
# 用法与环境变量同 sync-dashboard.sh。
set -euo pipefail
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/sync-dashboard.sh" "$@"