#!/usr/bin/env bash
# 性能回归门禁：对比 baseline 与当前 benchmark，超过阈值（默认 20%）即失败。
# 也可用 --absolute 模式做绝对阈值检查（跨机器基准不稳定时用）。
# 用法：
#   scripts/bench_ci.sh                    # 用内置 baseline 对比
#   scripts/bench_ci.sh --write-baseline   # 生成 baseline 文件（master 分支跑一次）
#   scripts/bench_ci.sh --threshold 30     # 自定义阈值百分比
#   scripts/bench_ci.sh --absolute         # 绝对阈值模式（适合 CI，无历史 baseline）
set -euo pipefail

BENCH_DIRS=(
  ./internal/shared/id/...
  ./internal/modules/auth/application/...
  ./internal/platform/notification/...
)
BASELINE_FILE="${BASELINE_FILE:-.benchmarks/baseline.txt}"
TMP_DIR="$(mktemp -d)"
THRESHOLD="${THRESHOLD:-20}"
MODE="relative"
# 绝对阈值（ns/op 上限）：Snowflake 200ns、UUID 100us、Login 100ms、WebhookSend 2ms
ABSOLUTE_LIMITS='SnowflakeNextID:300|UUIDNextID:200000|BenchmarkLogin:100000000|WebhookSend:2000000'
trap 'rm -rf "$TMP_DIR"' EXIT

for arg in "$@"; do
  case "$arg" in
    --write-baseline) MODE="write" ;;
    --absolute) MODE="absolute" ;;
    --threshold) MODE="threshold" ;;
    *) ;;
  esac
done
# --threshold N 形式：取下一个参数
if [[ "$MODE" == "threshold" && -n "${2:-}" ]]; then
  THRESHOLD="$2"
  MODE="relative"
fi

case "$MODE" in
  write)
    mkdir -p "$(dirname "$BASELINE_FILE")"
    echo "==> 生成 baseline（当前代码作为基准）"
    go test -bench=. -benchmem -run='^$' -benchtime=1s "${BENCH_DIRS[@]}" | tee "$BASELINE_FILE"
    echo "Baseline 已写入: $BASELINE_FILE"
    exit 0
    ;;
  absolute)
    echo "==> 绝对阈值模式运行 benchmark..."
    OUTPUT="$TMP_DIR/current.txt"
    go test -bench=. -benchmem -run='^$' -benchtime=1s "${BENCH_DIRS[@]}" | tee "$OUTPUT"
    FAILED=0
    while IFS= read -r line; do
      name=$(echo "$line" | awk '{print $1}' | sed 's/-[0-9]*$//')
      ns=$(echo "$line" | awk '{print $3}' | sed 's/ns\/op//')
      if [[ "$ns" =~ ^[0-9.]+$ ]]; then
        limit=$(echo "$ABSOLUTE_LIMITS" | tr '|' '\n' | grep "^$name:" | cut -d: -f2 || true)
        if [[ -n "$limit" ]] && (( $(echo "$ns > $limit" | bc -l) )); then
          echo "❌ $name: ${ns}ns/op 超过绝对上限 ${limit}ns/op"
          FAILED=1
        fi
      fi
    done < <(grep -E "^Benchmark" "$OUTPUT")
    if [[ "$FAILED" -eq 1 ]]; then
      echo ""
      echo "性能门禁失败：存在超过绝对阈值的 benchmark。"
      exit 1
    fi
    echo ""
    echo "✅ 性能门禁通过（绝对阈值模式）"
    exit 0
    ;;
esac

if [[ ! -f "$BASELINE_FILE" ]]; then
    echo "!! 未找到 baseline 文件 $BASELINE_FILE"
    echo "   先在 master 分支执行: scripts/bench_ci.sh --write-baseline"
    echo "   或设置 BASELINE_FILE 指向已有 baseline"
    exit 1
fi

echo "==> 运行当前 benchmark..."
CURRENT_FILE="$TMP_DIR/current.txt"
go test -bench=. -benchmem -run='^$' -benchtime=1s "${BENCH_DIRS[@]}" | tee "$CURRENT_FILE"

echo "==> 对比 baseline 与当前..."
if ! command -v benchstat >/dev/null 2>&1; then
    echo "benchstat 未安装，安装中: go install golang.org/x/perf/cmd/benchstat@latest"
    go install golang.org/x/perf/cmd/benchstat@latest
    export PATH="$(go env GOPATH)/bin:$PATH"
fi

# benchstat 对比：/delta 列显示变化百分比，+X% 表示变慢
OUTPUT="$(benchstat "$BASELINE_FILE" "$CURRENT_FILE")"
echo "$OUTPUT"

# 解析每个 benchmark 的 delta，找出超过阈值的退化
FAILED=0
while IFS= read -r line; do
    # 提取 delta 列（第 5 列，形如 +4.35% 或 -2.1%）
    delta=$(echo "$line" | awk '{print $5}')
    if [[ "$delta" =~ ^\+[0-9.]+%$ ]]; then
        pct=${delta//%/}
        pct=${pct//+/}
        if (( $(echo "$pct > $THRESHOLD" | bc -l) )); then
            echo "❌ 性能退化: $line (超过 ${THRESHOLD}% 阈值)"
            FAILED=1
        fi
    fi
done < <(echo "$OUTPUT" | grep -E "^Benchmark")

if [[ "$FAILED" -eq 1 ]]; then
    echo ""
    echo "性能回归门禁失败：存在超过 ${THRESHOLD}% 的退化。"
    echo "若为有意优化调整，请更新 baseline: scripts/bench_ci.sh --write-baseline"
    exit 1
fi

echo ""
echo "✅ 性能回归门禁通过（阈值 ${THRESHOLD}%）"
