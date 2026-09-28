#!/usr/bin/env bash
# 竞争检测（`go test -race`）分片的**单一数据源**：分片清单、每个分片覆盖的包、generator 用例
# 切分规则都在这里；`.github/workflows/ci.yml` 的 race 矩阵由 `shards-json` 派生，不另存一份。
#
# 为什么要分片（实测，不是猜的）：
#   把全量 `go test -race ./...` 放在一个 4 vCPU runner 上时，重型包之间互相争抢 CPU，
#   端到端耗时被放大到远超各自隔离耗时：
#     tools/composereport   隔离 14.8s  → 同一 job 里 378.4s（25.5×）
#     tools/generator       隔离 238.8s → 同一 job 里 729.8s（3.1×）
#     internal/**           隔离 ≈ CI   （0.9–2.6×，没有争抢问题）
#   于是把 race 拆成互不争抢的若干 job（各自一个 runner），**每个包仍然全量跑 race**，
#   只改变「谁和谁同时跑」。分片后总 runner 分钟数反而下降 —— 被争抢烧掉的时间本来就多于
#   各分片重复支付的编译时间。
#
# 分片：
#   core          所有非 internal/capabilities、非 tools 的包（kernel/shared/contract/config/
#                 assembly/app/profiles/e2e/cmd …）—— 用「排除法」派生，新增顶层包自动归位
#   caps-a/caps-b internal/capabilities 下按能力名排序对半切 —— 不写死能力名，新增能力自动归位
#   generator-N   tools/generator 整包，但用 `-run` 把**用例**再切 N 片（见下）
#   tools         ./tools/... 去掉 tools/generator（composereport/checkcapabilities/internal/…）
#
# generator 的用例切分：两条已知最重的用例定向分到不同分片（隔离实测各占该包 ~24% / ~22%，
# 纯轮转会把它们并到一片、把最短的一片压到 1/3），其余用例按名字排序取模轮转。
# 名单从 `go test -list` 实时取，不写死 —— 新增用例自动归位，rename/删除用例也不会让分片失效。
#
# 用法：
#   scripts/race_shards.sh shards-json     # workflow 矩阵用的 JSON 数组（如 ["core",…]）
#   scripts/race_shards.sh shards          # 分片名逐行列出
#   scripts/race_shards.sh list <shard>    # 打印该分片覆盖的包
#   scripts/race_shards.sh run <shard>     # 执行该分片的 go test -race（CI 用）
#   scripts/race_shards.sh verify          # 守卫：包/用例的并集 == 全集、且互不重叠
#
# `run` 开跑前会自己跑一次守卫（fail-closed）：分片清单落后于代码树（新增包没人覆盖、
# 用例切分漏了用例）会直接失败，而不是静默少测。

set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

GENERATOR_PKG="./tools/generator"

# 模块 import 前缀从 go.mod 派生，**不写死 "jimu"**：脚本会被复制进生成项目，那里的模块名是
# `--module` 指定的，写死会让分片口径与代码树对不上。`go test` 直接吃 import path，无需转 ./ 前缀。
MODULE=$(go list -m | sed 's/ .*//')
GENERATOR_IMPORT="$MODULE/tools/generator"

# generator 分片数。改这里必须同时确认 .github/workflows/ci.yml 的 race 矩阵来自
# `shards-json`（是自动派生的，无需手改）—— 分片名会自动变成 generator-1…generator-N。
GENERATOR_SHARDS=3

# 定向分片：下标 0 是 generator-1，依此类推；空串表示该片不额外定向。
# 只影响均衡度，不影响正确性（守卫保证用例不丢、不重）。
GENERATOR_PINS=(
  ""
  "TestCheckProfilesGoldenMatchesTheGeneratedClosure"
  "TestReportSucceedsForEverySelection"
)

# 单分片超时：与分片前的 `go test -race ./... -timeout 15m` 一致。
SHARD_TEST_TIMEOUT=15m

# 当前模块的包（import path 形式）
all_packages() {
  go list ./...
}

# 有能力包的能力目录名（排序去重）。从 go list 派生而不是 ls：目录存在但没 Go 包时
# `./internal/capabilities/<dir>/...` 会匹配不到包、让 go test 直接报错。
capability_dirs() {
  go list ./internal/capabilities/... | sed "s|^$MODULE/internal/capabilities/||" | cut -d/ -f1 | sort -u
}

generator_tests_cache=""
# generator 的顶层用例名（排序去重）。GENERATOR_LIST_FLAGS=-race 时用 race 编译，
# 让随后真正跑用例的 `go test -race -run` 复用同一份编译产物（分片场景下这是白捡的）。
generator_test_list() {
  if [ -z "$generator_tests_cache" ]; then
    # shellcheck disable=SC2086  # 允许为空
    generator_tests_cache=$(go test ${GENERATOR_LIST_FLAGS:-} -list '.*' "$GENERATOR_PKG" \
      | grep -E '^Test' | sort -u)
  fi
  printf '%s\n' "$generator_tests_cache"
}

# generator-<i> 片要跑的用例正则（`^(A|B|C)$`），由全量用例清单实时切分。
generator_pattern() {
  local idx="$1"
  local pinned="${GENERATOR_PINS[$((idx - 1))]:-}"
  local rest rot p
  rest=$(generator_test_list)
  # **所有**被定向的用例都要从轮转池里剔除，且每个分片用同一份池子：只剔除本分片那条会让
  # 各分片的池子长度不同，取模的相位随之错开 —— 结果既重复又漏（守卫抓过这个 bug）。
  for p in "${GENERATOR_PINS[@]}"; do
    [ -n "$p" ] || continue
    rest=$(printf '%s\n' "$rest" | grep -vxF -- "$p" || true)
  done
  rot=$(printf '%s\n' "$rest" \
    | awk -v n="$GENERATOR_SHARDS" -v i="$((idx - 1))" 'NR % n == i')
  { printf '%s\n' "$pinned"; printf '%s\n' "$rot"; } | grep -v '^$' | paste -sd'|' -
}

shard_names() {
  local names=(core caps-a caps-b)
  local i
  for ((i = 1; i <= GENERATOR_SHARDS; i++)); do
    names+=("generator-$i")
  done
  names+=(tools)
  printf '%s\n' "${names[@]}"
}

# 只覆盖「包」的分片名（generator 分片按用例切，包集合另外算）。
package_shard_names() {
  printf '%s\n' core caps-a caps-b tools
}

shard_packages() {
  case "$1" in
    core)
      all_packages | grep -v -e "^$MODULE/internal/capabilities/" -e "^$MODULE/tools/" || true
      ;;
    caps-a | caps-b)
      local total half dirs patterns
      total=$(capability_dirs | wc -l | tr -d ' ')
      half=$((total / 2))
      if [ "$1" = "caps-a" ]; then
        dirs=$(capability_dirs | head -n "$half")
      else
        dirs=$(capability_dirs | tail -n "+$((half + 1))")
      fi
      [ -n "$dirs" ] || {
        echo "❌ 分片 $1 没有匹配到能力目录（capability_dirs 为空？）" >&2
        return 1
      }
      patterns=$(printf '%s\n' "$dirs" | sed 's|^|./internal/capabilities/|; s|$|/...|')
      # 展开成**具体包路径**再输出，与 core/tools 同口径 —— 否则守卫拿模式去和 ./... 的
      # 包路径逐条比对，会把整个 capabilities 判成「没人覆盖」。
      # shellcheck disable=SC2086  # 模式列表按空格拆分传给 go list
      go list $patterns
      ;;
    generator-*)
      printf '%s\n' "$GENERATOR_IMPORT"
      ;;
    tools)
      all_packages | grep "^$MODULE/tools/" | grep -v -x -e "$GENERATOR_IMPORT" || true
      ;;
    *)
      echo "❌ 未知分片: $1" >&2
      return 2
      ;;
  esac
}

# 守卫（包）：包分片并集 == ./...（不丢、不重）。只花几次 go list，所有分片开跑前都跑。
_verify_packages() {
  local rc=0 dups all union
  all=$(all_packages | sort)
  # generator 的**包**由 generator-1..N 的用例切分覆盖（每个分片都跑同一个包），包集合里只算一次。
  union=$(
    {
      for s in $(package_shard_names); do shard_packages "$s"; done
      printf '%s\n' "$GENERATOR_IMPORT"
    } | sort
  )
  dups=$(printf '%s\n' "$union" | uniq -d)
  if [ -n "$dups" ]; then
    echo "❌ 以下包被多个分片重复覆盖："
    printf '%s\n' "$dups"
    rc=1
  fi
  _check_sets "分片未覆盖" "分片多出" "$all" "$(printf '%s\n' "$union" | uniq)" || rc=1
  if [ "$rc" -eq 0 ]; then
    echo "✅ race 分片守卫（包）：$(printf '%s\n' "$all" | wc -l | tr -d ' ') 个包的并集 == go list ./... 且无重复"
  fi
  return "$rc"
}

# 守卫（generator 用例）：N 个用例分片并集 == `go test -list` 全集（不丢、不重）。
# 只有 generator 分片需要它（其它分片不碰这个包），且调用方要先 export GENERATOR_LIST_FLAGS=-race，
# 让它与随后的 `go test -race -run` 共用同一份编译产物 —— 否则等于白编译一遍。
_verify_generator_tests() {
  local rc=0 dups tests picked
  tests=$(generator_test_list | sort)
  picked=$(
    for ((i = 1; i <= GENERATOR_SHARDS; i++)); do
      printf '%s\n' "$tests" | grep -E "^($(generator_pattern "$i"))$" || true
    done | sort
  )
  dups=$(printf '%s\n' "$picked" | uniq -d)
  if [ -n "$dups" ]; then
    echo "❌ 以下用例被多个 generator 分片重复覆盖："
    printf '%s\n' "$dups"
    rc=1
  fi
  _check_sets "generator 分片未覆盖" "generator 分片多出" "$tests" "$(printf '%s\n' "$picked" | uniq)" || rc=1
  if [ "$rc" -eq 0 ]; then
    echo "✅ race 分片守卫（generator 用例）：$(printf '%s\n' "$tests" | wc -l | tr -d ' ') 条用例的并集 == go test -list 全集且无重复"
  fi
  return "$rc"
}

cmd_verify() {
  _verify_packages
  _verify_generator_tests
}

# _check_sets <未覆盖说明> <多出说明> <期望全集> <实际并集>：用 comm 双向比对（都须已排序）。
_check_sets() {
  local want_list="$1" extra_msg="$2" want="$3" got="$4" missing extra
  missing=$(comm -23 <(printf '%s\n' "$want") <(printf '%s\n' "$got"))
  extra=$(comm -13 <(printf '%s\n' "$want") <(printf '%s\n' "$got"))
  if [ -n "$missing" ]; then
    echo "❌ ${want_list}："
    printf '%s\n' "$missing"
    return 1
  fi
  if [ -n "$extra" ]; then
    echo "❌ ${extra_msg}："
    printf '%s\n' "$extra"
    return 1
  fi
  return 0
}

cmd_shards() {
  shard_names
}

cmd_shards_json() {
  local out="" s
  for s in $(shard_names); do out+="\"$s\","; done
  printf '[%s]\n' "${out%,}"
}

cmd_list() {
  local shard="${1:-}"
  [ -n "$shard" ] || {
    echo "用法: race_shards.sh list <shard>" >&2
    exit 2
  }
  shard_packages "$shard"
}

cmd_run() {
  local shard="${1:-}"
  [ -n "$shard" ] || {
    echo "用法: race_shards.sh run <shard>" >&2
    exit 2
  }
  # fail-closed：先证明分片清单没有落后于代码树。包集每次必查；用例切分只在 generator 分片查，
  # 且复用下面 go test 的 race 编译产物（不额外付编译费）。
  _verify_packages
  case "$shard" in
    generator-*) export GENERATOR_LIST_FLAGS=-race && _verify_generator_tests ;;
  esac

  case "$shard" in
    generator-*)
      local idx="${shard#generator-}" pattern
      pattern=$(generator_pattern "$idx")
      [ -n "$pattern" ] || {
        echo "❌ 分片 $shard 的用例正则为空" >&2
        exit 1
      }
      echo "▶ race 分片 ${shard}：$GENERATOR_PKG 的 $(printf '%s' "$pattern" | tr '|' '\n' | wc -l | tr -d ' ') 条用例"
      go test -race -run "^(${pattern})$" "$GENERATOR_IMPORT" -timeout "$SHARD_TEST_TIMEOUT"
      ;;
    *)
      local pkgs
      pkgs=$(shard_packages "$shard")
      [ -n "$pkgs" ] || {
        echo "❌ 分片 $shard 没有匹配到任何包" >&2
        exit 1
      }
      echo "▶ race 分片 ${shard}：$(printf '%s\n' "$pkgs" | wc -l | tr -d ' ') 个包"
      # shellcheck disable=SC2086  # 包列表按空格拆分传给 go test
      go test -race $pkgs -timeout "$SHARD_TEST_TIMEOUT"
      ;;
  esac
}

usage() {
  # 从本文件的注释头里现取「用法」段 —— 不写死行号，改动上面的说明不会让 --help 失真
  sed -n '/^# 用法：/,/^#$/p' "$0" | sed 's/^# \{0,1\}//'
}

case "${1:-}" in
  shards) cmd_shards ;;
  shards-json) cmd_shards_json ;;
  list) shift; cmd_list "$@" ;;
  run) shift; cmd_run "$@" ;;
  verify) cmd_verify ;;
  "" | -h | --help) usage ;;
  *)
    echo "❌ 未知子命令: $1" >&2
    usage >&2
    exit 2
    ;;
esac
