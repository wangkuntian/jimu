#!/usr/bin/env bash
# 压测脚本：对本地服务端做简单 HTTP 负载测试。
# 依赖 hey（未安装时给出安装命令）。
# 用法：
#   scripts/loadtest.sh                    # 默认打 /api/v1/health，5000 请求，50 并发
#   URL=http://localhost:8080/api/v1/health scripts/loadtest.sh
#   URL=... scripts/loadtest.sh -n 10000 -c 100
set -euo pipefail

URL="${URL:-http://localhost:8080/api/v1/health}"
TARGET="${1:-}"
if [[ -n "$TARGET" ]]; then
    URL="${URL}${TARGET}"
fi

if ! command -v hey >/dev/null 2>&1; then
    echo "hey 未安装，运行: go install github.com/rakyll/hey@latest" >&2
    exit 1
fi

echo "压测目标: $URL"
hey -n "${N:-5000}" -c "${C:-50}" -m GET "$URL"
