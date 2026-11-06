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
#   core             所有非 internal/capabilities、非 tools 的包（kernel/shared/contract/config/
#                    assembly/app/profiles/e2e/cmd …）—— 用「排除法」派生，新增顶层包自动归位
#   caps-a/caps-b    internal/capabilities 下按能力名排序对半切 —— 不写死能力名，新增能力自动归位
#   composereport-N  tools/composereport 整包，但把它那条重型用例按**形态**再切 N 片（见下）
#   generator-N      tools/generator 整包，但用 `-run` 把**用例**再切 N 片（见下）
#   tools            ./tools/... 去掉 tools/{generator,composereport}（checkcapabilities/internal/…）
#
# composereport 的形态切分：那条重型用例在 `-race` 下每个形态都要做一次全依赖图 packages.Load +
# 全闭包行数统计（实测整包 257s，是 tools 分片唯一的压力来源）。按形态对半切后每片只度量自己那部分，
# **每个形态仍然在某个分片里被 `-race` 跑过**；跨形态关系（minimal ≤ 85% full 之类）由未设
# JIMU_METRICS_PROFILES 时的完整度量断言（非 race 的 `Capability Gates` job 全量跑该包，整包 23s，
# 见 ci.yml 的 `Run the composereport package tests`）。形态名单来自
# `tools/profileoverlay -list`（registry 唯一来源），不写死。
#
# generator 的用例切分：按**实测耗时**做 LPT 贪心装箱（见 `generator_shard_plan`）。
# 权重来自 `scripts/test_shard_weights.tsv`（`shard-weights run` 生成，入库）；没有权重的用例
# 取该表中位数作为估计，因此新增/改名用例不会让切分失效，只是配平精度暂时下降。
# 名单从 `go test -list` 实时取，不写死。
#
# 为什么不按名字取模轮转（旧做法）：用例耗时差两个数量级（最重 ~54s，中位数 0.16s），
# 按名字均分会让某一片独自扛下最重的几条 —— v0.10.0 实测 generator-4 单分片 11m41s，
# 而 generator-5 只有 2m03s，是整条流水线的唯一长杆。改为耗时装箱后 4 片的负载
# 是 121/121/121/121s（误差 <1s）。
#
# 用法：
#   scripts/test_shards.sh shards-json [all|db|nodb]
#                                                # workflow 矩阵用的 JSON 数组；db/nodb 按
#                                                # 是否真的连库拆成两个 job（nodb 不挂容器）
#   scripts/test_shards.sh shards                 # 分片名逐行列出
#   scripts/test_shards.sh list <shard>           # 打印该分片覆盖的包
#   scripts/test_shards.sh run <shard> [--cover]  # 执行该分片；--cover 同时产出 cover-<shard>.out
#   scripts/test_shards.sh merge-cover <dir> <out> # 合并分片覆盖率 profile（表头 + 正文拼接）
#   scripts/test_shards.sh verify                 # 守卫：包/用例/形态的并集 == 全集、且互不重叠
#   scripts/test_shards.sh shard-weights run      # 实测 generator 用例耗时，刷新权重表（入库）
#   scripts/test_shards.sh plan                    # 打印 generator 分片计划与各片负载（秒）
#   scripts/test_shards.sh pattern <shard>         # 打印该分片的 -run 正则 / 形态列表
#
# 本脚本同时服务两件事：竞争检测（`-race`）与覆盖率（`-covermode=atomic -coverprofile`）。
# 两者可以同跑（`go test -race -covermode=atomic`），因此 CI 只跑一遍测试套件：
# 每片产出自己的 profile，由 `merge-cover` 拼接（分片按包/用例/形态互不重叠，故拼接无重复计数；
# set 与 atomic 两种插桩模式的覆盖率实测等价，见 spec「机制探针」）。
#
# `run` 开跑前会自己跑一次守卫（fail-closed）：分片清单落后于代码树（新增包没人覆盖、
# 用例切分漏了用例、形态切片漏了形态）会直接失败，而不是静默少测。
#
# 按是否连库拆两个 job：`tools` 与 `composereport-*` 不碰数据库，ci.yml 让它们跑在
# `shards-nodb`（无 service container）。三个容器会和 `-race` 抢 4 vCPU runner 的 CPU。

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
# 分片数只影响并行度；配平由 `generator_shard_plan` 按权重完成，不需要手工重排。
# 4 片的依据见 scripts/test_shard_weights.tsv 里的 `# shards=` 行（`shard-weights run` 会
# 按当前权重表算出「再加一片还能省多少」并写回）。
GENERATOR_SHARDS=4

# 用例耗时权重表：`<秒>\t<用例名>`，由 `scripts/test_shards.sh shard-weights run` 实测生成。
# 入库是因为 CI 无法在不额外跑一遍测试的前提下得到权重；表过期只会让配平变差，不会让门禁失效。
GENERATOR_WEIGHTS_FILE="scripts/test_shard_weights.tsv"

# 单分片超时：与分片前的 `go test -race ./... -timeout 15m` 一致。
SHARD_TEST_TIMEOUT=15m

# tools/composereport 的重型用例按**形态**切片：它在 `-race` 下每个形态都要做一次全依赖图
# packages.Load + 全闭包行数统计。当前只保留 full/minimal 两个 profile，因此切 2 片；
# 每个形态仍被 `-race` 跑过。形态数变化时同步此值，避免出现空分片。
COMPOSEREPORT_SHARDS=2
COMPOSEREPORT_PKG="./tools/composereport"
COMPOSEREPORT_IMPORT="$MODULE/tools/composereport"
# 与 tools/composereport/main_test.go 的 profileShardEnv 同名，它是这条切片的开关
COMPOSEREPORT_ENV="JIMU_METRICS_PROFILES"

# 当前模块的包（import path 形式）
all_packages() {
  go list ./...
}

profiles_cache=""
# 形态名从 registry 派生（`tools/profileoverlay -list` 是唯一来源），不写死在这份脚本里
all_profiles() {
  if [ -z "$profiles_cache" ]; then
    profiles_cache=$(go run ./tools/profileoverlay -list)
  fi
  printf '%s\n' "$profiles_cache"
}

# 第 i 片 composereport 负责的形态（把形态列表按片数切分，保持 registry 顺序；余数分给前面的片，
# 故 COMPOSEREPORT_SHARDS 不整除形态数时每片仍非空 —— 片数多于形态数时守卫照样能抓到空片）
composereport_profiles() {
  local idx="$1" total base extra per before start end
  total=$(all_profiles | wc -l | tr -d ' ')
  base=$((total / COMPOSEREPORT_SHARDS))
  extra=$((total % COMPOSEREPORT_SHARDS))
  if [ "$idx" -le "$extra" ]; then
    per=$((base + 1))
    before=$(((idx - 1) * per))
  else
    per=$base
    before=$((extra * (base + 1) + (idx - extra - 1) * base))
  fi
  start=$((before + 1))
  end=$((before + per))
  all_profiles | sed -n "${start},${end}p"
}

composereport_shard_names() {
  local i
  for ((i = 1; i <= COMPOSEREPORT_SHARDS; i++)); do
    printf 'composereport-%s\n' "$i"
  done
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

# 用例权重（秒）。未在表中的用例取中位数估计：新用例既不会掉出分片，也不会因为猜成 0
# 被全塞进同一片。表不存在时全部退回中位数（= 纯轮转的等价物），门禁仍然正确。
generator_weight_of() {
  local name="$1" weights="$2"
  awk -F'\t' -v n="$name" '$2 == n { print $1; found=1; exit } END { if (!found) print "" }' \
    <<<"$weights"
}

# generator 分片计划：LPT（Longest Processing Time first）贪心——用例按耗时降序，
# 每条放进当前负载最小的分片。对「减少最大片」这个目标，LPT 有 4/3 近似保证，
# 而 35 条用例的规模下它实际上就是最优或接近最优。
#
# 输出 `generator-<i><TAB><用例名>`，按分片名排序；重复调用结果一致（同权重同排序）。
# 权重会随机器/负载漂移，但同一份表在同一 CI 配置下是稳定的，因此矩阵仍然确定性。
generator_shard_plan() {
  local weights="" median=""
  if [ -f "$GENERATOR_WEIGHTS_FILE" ]; then
    weights=$(grep -vE '^[[:space:]]*(#|$)' "$GENERATOR_WEIGHTS_FILE" || true)
    median=$(printf '%s\n' "$weights" | awk -F'\t' 'NF>=2 {print $1}' | sort -n \
      | awk '{a[NR]=$1} END { if (NR==0) print 1; else print a[int((NR+1)/2)] }')
  fi
  [ -n "$median" ] || median=1

  generator_test_list | while read -r name; do
    local w
    w=$(generator_weight_of "$name" "$weights")
    [ -n "$w" ] || w="$median"
    printf '%s\t%s\n' "$w" "$name"
  done | sort -t$'\t' -k1,1nr -k2,2 | awk -F'\t' \
    -v n="$GENERATOR_SHARDS" '
      {
        # 选当前负载最小的片；并列时选下标最小的，保证确定性。
        target = 1
        for (i = 2; i <= n; i++) if (load[i] < load[target]) target = i
        load[target] += $1
        print "generator-" target "\t" $2
      }'
}

# generator-<i> 片要跑的用例正则（`^(A|B|C)$`）。
generator_pattern() {
  local idx="$1" plan
  plan=$(generator_shard_plan_cache)
  printf '%s\n' "$plan" \
    | awk -F'\t' -v s="generator-$idx" '$1 == s { print $2 }' \
    | paste -sd'|' -
}

# 计划只算一次：generator_test_list 走 `go test -list`，每次分片都重算是纯浪费。
generator_shard_plan_cache_value=""
generator_shard_plan_cache() {
  if [ -z "$generator_shard_plan_cache_value" ]; then
    generator_shard_plan_cache_value=$(generator_shard_plan)
  fi
  printf '%s\n' "$generator_shard_plan_cache_value"
}

shard_names() {
  local names=(core caps-a caps-b)
  names+=($(composereport_shard_names))
  local i
  for ((i = 1; i <= GENERATOR_SHARDS; i++)); do
    names+=("generator-$i")
  done
  names+=(tools)
  printf '%s\n' "${names[@]}"
}

# 只覆盖「包」的分片名（generator / composereport 分片各自按用例、形态再切，包集合另外算）。
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
    composereport-*)
      printf '%s\n' "$COMPOSEREPORT_IMPORT"
      ;;
    generator-*)
      printf '%s\n' "$GENERATOR_IMPORT"
      ;;
    tools)
      all_packages | grep "^$MODULE/tools/" \
        | grep -v -x -e "$GENERATOR_IMPORT" -e "$COMPOSEREPORT_IMPORT" || true
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
  # generator 与 composereport 的**包**由各自的切片覆盖（每个分片都跑同一个包），包集合里各算一次。
  union=$(
    {
      for s in $(package_shard_names); do shard_packages "$s"; done
      printf '%s\n' "$GENERATOR_IMPORT" "$COMPOSEREPORT_IMPORT"
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

# 守卫（composereport 形态切片）：各片负责的形态并集 == `profileoverlay -list` 全集（不丢、不重），
# 且没有空片。切片是算术对半，off-by-one 会静默漏掉一个形态 —— 那正是这条守卫要抓的。
_verify_composereport_shards() {
  local rc=0 dups all picked i profiles
  all=$(all_profiles | sort)
  picked=$(
    for ((i = 1; i <= COMPOSEREPORT_SHARDS; i++)); do
      composereport_profiles "$i"
    done | sort
  )
  dups=$(printf '%s\n' "$picked" | uniq -d)
  if [ -n "$dups" ]; then
    echo "❌ 以下形态被多个 composereport 分片重复覆盖："
    printf '%s\n' "$dups"
    rc=1
  fi
  _check_sets "composereport 分片未覆盖" "composereport 分片多出" \
    "$all" "$(printf '%s\n' "$picked" | uniq)" || rc=1
  for ((i = 1; i <= COMPOSEREPORT_SHARDS; i++)); do
    profiles=$(composereport_profiles "$i")
    if [ -z "$profiles" ]; then
      echo "❌ composereport-$i 没有分到任何形态（形态数少于片数？）"
      rc=1
    fi
  done
  if [ "$rc" -eq 0 ]; then
    echo "✅ race 分片守卫（composereport 形态）：$(printf '%s\n' "$all" | wc -l | tr -d ' ') 个形态的并集 == profileoverlay -list 且无重复"
  fi
  return "$rc"
}

# 打印分片计划与各片负载：排查配平问题的入口（CI 日志里也用它对照）。
cmd_plan() {
  local plan median="n/a" planfile weightsfile
  plan=$(generator_shard_plan)
  weightsfile="$GENERATOR_WEIGHTS_FILE"
  [ -f "$weightsfile" ] || weightsfile=/dev/null
  median=$(grep -vE '^[[:space:]]*(#|$)' "$weightsfile" \
    | awk -F'\t' 'NF>=2 {print $1}' | sort -n \
    | awk '{a[NR]=$1} END { if (NR==0) print "n/a"; else print a[int((NR+1)/2)] }')
  printf 'generator 分片计划（%s 片，权重表 %s，中位数 %ss）\n' \
    "$GENERATOR_SHARDS" "$GENERATOR_WEIGHTS_FILE" "$median"

  # 把权重 join 到计划上，按分片汇总条数与耗时。
  planfile=$(mktemp)
  trap 'rm -f "$planfile"' RETURN
  printf '%s\n' "$plan" >"$planfile"
  awk -F'\t' -v n="$GENERATOR_SHARDS" -v wf="$weightsfile" -v pf="$planfile" '
    BEGIN {
      while ((getline line < wf) > 0) {
        if (line ~ /^[[:space:]]*(#|$)/) continue
        split(line, f, "\t")
        if (f[2] != "") weight[f[2]] = f[1] + 0
      }
      while ((getline line < pf) > 0) {
        split(line, g, "\t")
        count[g[1]]++
        load[g[1]] += weight[g[2]]
      }
      for (i = 1; i <= n; i++) {
        s = "generator-" i
        printf "  %-13s %3d tests  %6.1fs\n", s, count[s], load[s]
      }
    }'
}

cmd_verify() {
  _verify_packages
  _verify_composereport_shards
  _verify_generator_tests
}

# 跑一遍 generator 包的全部用例（`-race`，与 CI 同口径），把每条耗时写成权重表。
#
# 为什么入库而不是 CI 现测：现测等于把这条包跑两遍（一遍测量、一遍真跑），
# 而 CI 的耗时本身就是固定成本。表过期只会让配平变差，不会让门禁失效 ——
# 未列出的用例按中位数估计，仍然落在某个分片里。
cmd_shard_weights() {
  local output="scripts/test_shard_weights.tsv" tmp rc=0
  tmp=$(mktemp)
  trap 'rm -f "$tmp"' RETURN
  echo "▶ 测量 generator 用例耗时（-race，与 CI 同口径）…" >&2
  # `-v` 才能拿到每条用例的耗时；失败也要产出表（用已有耗时），故先落 tmp 再判退出码。
  go test -race -v -count=1 -timeout "$SHARD_TEST_TIMEOUT" "$GENERATOR_PKG" >"$tmp" 2>&1 || rc=$?

  grep -E '^--- (PASS|FAIL|SKIP): ' "$tmp" \
    | sed -E 's/^--- (PASS|FAIL|SKIP): ([^ ]+) \(([0-9.]+)s\)/\3\t\2/' \
    | sort -t$'\t' -k1,1nr >"${output}.rows" || true

  {
    echo "# generator 用例耗时权重（秒），由 \`scripts/test_shards.sh shard-weights run\` 生成。"
    echo "# 用途：生成器按 LPT 贪心装箱把用例分到 generator-1..N，使最长分片最小。"
    echo "# 未列出的用例按中位数估计；过期只会让配平变差，不会漏测。"
    echo "#"
    echo "# 刷新：bash scripts/test_shards.sh shard-weights run（4 vCPU，与 CI runner 同档）"
    echo "# shards=$GENERATOR_SHARDS"
    echo "# median=$(awk -F'\t' '{print $1}' "${output}.rows" | sort -n | awk '{a[NR]=$1} END { if (NR==0) print 1; else print a[int((NR+1)/2)] }')"
    echo "# total=$(awk -F'\t' '{s+=$1} END {printf "%.1f", s}' "${output}.rows")"
    cat "${output}.rows"
  } >"$output"
  rm -f "${output}.rows"

  echo "✅ 已写入 ${output}（$(grep -cvE '^#|^$' "$output") 条用例）" >&2
  if [ "$rc" -ne 0 ]; then
    echo "⚠️ 用例有失败（退出码 $rc），权重表仍已按实际耗时更新" >&2
    return "$rc"
  fi
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

# 需要 service container（MySQL/Postgres/Redis）的分片。
#
# 不是所有分片都连库：`tools`（纯静态工具与框架 manifest）与 `composereport`
# （只做依赖图度量）完全不碰数据库，给它们挂三个容器纯属浪费 —— 在 4 vCPU runner 上
# 容器会和 `-race` 抢 CPU，实测让同一批用例慢 4 倍（本地 154s vs CI 612s）。
# 判定依据是这些包内是否有 `testutil.SkipUnless{Mysql,Redis,Postgres}` 的调用点；
# 新增能力包时它会自动落进 caps-*，无需改这里。
shard_needs_db() {
  case "$1" in
    tools | composereport-*) return 1 ;;
    *) return 0 ;;
  esac
}

# <which> = all | db | nodb：给 workflow 的矩阵用（两个 job 各自一份矩阵）。
cmd_shards_json() {
  local which="${1:-all}" out="" s want
  for s in $(shard_names); do
    case "$which" in
      all) want=1 ;;
      db) shard_needs_db "$s" && want=1 || want=0 ;;
      nodb) shard_needs_db "$s" && want=0 || want=1 ;;
      *) echo "❌ 未知过滤: $which" >&2; exit 2 ;;
    esac
    [ "$want" -eq 1 ] && out+="\"$s\","
  done
  # 空集合要输出 `[]` 而不是 `[,]`：GitHub 的 fromJSON 对后者直接报错。
  [ -n "$out" ] || {
    printf '[]\n'
    return 0
  }
  printf '[%s]\n' "${out%,}"
}

# 打印某个 generator 分片的 `-run` 正则（供测量 workflow 与调试用）。
cmd_pattern() {
  local shard="${1:-}"
  case "$shard" in
    generator-*) generator_pattern "${shard#generator-}" ;;
    composereport-*) composereport_profiles "${shard#composereport-}" | paste -sd, - ;;
    *)
      echo "用法: test_shards.sh pattern <generator-N|composereport-N>" >&2
      exit 2
      ;;
  esac
}

cmd_list() {
  local shard="${1:-}"
  [ -n "$shard" ] || {
    echo "用法: test_shards.sh list <shard>" >&2
    exit 2
  }
  shard_packages "$shard"
}

cmd_run() {
  local shard="${1:-}"
  local cover=0
  shift || true
  case "${1:-}" in
    --cover) cover=1 ;;
    "") ;;
    *) echo "❌ 未知参数: $1" >&2; exit 2 ;;
  esac
  [ -n "$shard" ] || {
    echo "用法: test_shards.sh run <shard> [--cover]" >&2
    exit 2
  }
  # 覆盖率参数用「数组 + 变量」两份表示：数组给 go test 用，
  # 字符串给 GENERATOR_LIST_FLAGS 用（预编译必须带同样的参数才能复用编译产物）。
  local cover_args=() cover_env=""
  if [ "$cover" -eq 1 ]; then
    cover_args=(-covermode=atomic "-coverprofile=cover-${shard}.out")
    cover_env="-covermode=atomic -coverprofile=cover-${shard}.out"
  fi
  # fail-closed：先证明分片清单没有落后于代码树。包集每次必查；用例/形态切片只在对应分片上查，
  # 且复用下面 go test 的 race 编译产物（不额外付编译费）。
  _verify_packages
  case "$shard" in
    generator-*)
      export GENERATOR_LIST_FLAGS="-race $cover_env"
      _verify_generator_tests
      ;;
    composereport-*) _verify_composereport_shards ;;
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
      go test -race -run "^(${pattern})$" "$GENERATOR_IMPORT" -timeout "$SHARD_TEST_TIMEOUT" \
        ${cover_args[@]+"${cover_args[@]}"}
      ;;
    composereport-*)
      local idx="${shard#composereport-}" profiles
      profiles=$(composereport_profiles "$idx" | paste -sd, -)
      [ -n "$profiles" ] || {
        echo "❌ 分片 $shard 没有分到任何形态" >&2
        exit 1
      }
      echo "▶ race 分片 ${shard}：$COMPOSEREPORT_PKG 的形态 $profiles"
      # 注意：这里的 `-run '^TestProfileCompiledSurface$'` 只跑重型的那一条用例。该包**其余用例**
      # 由 ci.yml 的 `Capability Gates` job 以非 race 全量 `go test ./tools/composereport/...` 覆盖
      # （tools 分片显式排除本包）。改这里的 `-run` 前先确认那一侧仍在跑，否则会静默丢用例。
      env "$COMPOSEREPORT_ENV=$profiles" go test -race -run '^TestProfileCompiledSurface$' \
        "$COMPOSEREPORT_IMPORT" -timeout "$SHARD_TEST_TIMEOUT" ${cover_args[@]+"${cover_args[@]}"}
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
      go test -race $pkgs -timeout "$SHARD_TEST_TIMEOUT" ${cover_args[@]+"${cover_args[@]}"}
      ;;
  esac
}

# 合并分片覆盖率 profile：Go 的文本 profile 是「一行 mode 头 + 若干 file:line 区间计数」，
# 分片之间按包/用例/形态互不重叠（由 verify 守卫保证），所以「保留一份表头 + 正文顺序拼接」
# 即可得到与单轮全量连跑完全一致的结果（实测 total 96.1% == 96.1%，见 spec「机制探针」）。
cmd_merge_cover() {
  local dir="${1:-}" out="${2:-}"
  [ -n "$dir" ] && [ -n "$out" ] || {
    echo "用法: test_shards.sh merge-cover <dir> <out>" >&2
    exit 2
  }
  local files=()
  while IFS= read -r f; do files+=("$f"); done < <(find "$dir" -maxdepth 1 -name '*.out' | sort)
  [ "${#files[@]}" -gt 0 ] || {
    echo "❌ $dir 下没有找到任何 *.out（分片是否都没产出 profile？）" >&2
    exit 1
  }
  local header
  header=$(head -1 "${files[0]}")
  case "$header" in
    mode:*) ;;
    *) echo "❌ ${files[0]} 不是合法的 Go 覆盖率 profile（首行应为 mode: ...）" >&2; exit 1 ;;
  esac
  {
    printf '%s\n' "$header"
    local f
    for f in "${files[@]}"; do tail -n +2 "$f"; done
  } > "$out"
  echo "✅ 合并 ${#files[@]} 个分片 profile → ${out}（$(wc -l < "$out" | tr -d ' ') 行）"
}

usage() {
  # 从本文件的注释头里现取「用法」段 —— 不写死行号，改动上面的说明不会让 --help 失真
  sed -n '/^# 用法：/,/^#$/p' "$0" | sed 's/^# \{0,1\}//'
}

case "${1:-}" in
  shards) cmd_shards ;;
  shards-json) shift; cmd_shards_json "${1:-all}" ;;
  list) shift; cmd_list "$@" ;;
  pattern) shift; cmd_pattern "$@" ;;
  run) shift; cmd_run "$@" ;;
  merge-cover) shift; cmd_merge_cover "$@" ;;
  shard-weights) shift; cmd_shard_weights "$@" ;;
  plan) cmd_plan ;;
  verify) cmd_verify ;;
  "" | -h | --help) usage ;;
  *)
    echo "❌ 未知子命令: $1" >&2
    usage >&2
    exit 2
    ;;
esac
