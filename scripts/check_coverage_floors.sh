#!/usr/bin/env bash
# 逐包「不低于」覆盖率下限门禁（与 scripts/check_ci_gate.sh 同风格：单一实现 + 可本地单测）。
#
# 输入：`go tool cover` 的**文本** profile（`scripts/test_shards.sh merge-cover` 的产物）。
# 解析只用 awk/grep/sort，**不需要 Go 工具链** —— CI 的 `Coverage & Docs` job 与本地都能跑。
#
# 判定单元是**目录前缀**：表里的一行覆盖该前缀下的全部包（含子包），与 spec §9 的逐包读数
# 口径一致（例如 `internal/capabilities/tenant` 44.1% 是整个能力目录的合计，不是单个 Go 包）。
#
# 两段式（避免一上来就红）：
#   下限值为空（`前缀|`）→ 只打印读数、不判定、恒为 0 退出（第一段：取 CI 权威基线）。
#   下限值非空         → 现状 < 下限即非零退出，并给出「还差多少条语句」（第二段：启用判定）。
#
# 用法：check_coverage_floors.sh [coverage.out]
# 环境变量 COVERAGE_FLOORS 可覆盖内置下限表，仅用于本文件的本地单测。
set -euo pipefail

# 下限表：一处定义；`前缀|<下限%>`。下限留空 = 只打印不判定。
# 仅包含本刀涉及的包（spec §5）；数值由校准 run 填入（基线 − 0.5pp，见 Task 9）。
FLOORS='
jimu/internal/capabilities/tenant|
jimu/internal/capabilities/user|
jimu/internal/capabilities/access|
jimu/internal/capabilities/audit|
jimu/internal/capabilities/apikey|
jimu/internal/kernel/tenant|
jimu/internal/kernel/auth|
jimu/internal/kernel/http/middleware|
'

floors="${COVERAGE_FLOORS:-$FLOORS}"
profile="${1:-coverage.out}"
[ -f "$profile" ] || {
  echo "❌ 找不到覆盖率 profile: $profile" >&2
  exit 2
}
head -1 "$profile" | grep -q '^mode:' || {
  echo "❌ $profile 不是合法的 Go 覆盖率 profile（首行应为 mode: ...）" >&2
  exit 2
}

raw=$(mktemp)
floorfile=$(mktemp)
printf '%s\n' "$floors" > "$floorfile"
awk -v ffile="$floorfile" '
  BEGIN {
    while ((getline line < ffile) > 0) {
      gsub(/^[ \t]+|[ \t]+$/, "", line)
      if (line == "") continue
      split(line, kv, "|")
      p = kv[1]
      prefix[++np] = p
      floor[p] = kv[2]
    }
    close(ffile)
  }
  /^mode:/ { next }
  {
    split($1, a, ":")
    file = a[1]
    dir = file
    sub(/\/[^\/]*$/, "", dir)
    stmt = $2 + 0
    cnt = $3 + 0
    if (!(dir in seen)) { order[++nd] = dir; seen[dir] = 1 }
    stmt_sum[dir] += stmt
    if (cnt > 0) cov_sum[dir] += stmt
    for (i = 1; i <= np; i++) {
      if (index(file, prefix[i] "/") == 1) {
        tstmt[prefix[i]] += stmt
        if (cnt > 0) tcov[prefix[i]] += stmt
      }
    }
  }
  END {
    for (i = 1; i <= nd; i++) {
      d = order[i]; s = stmt_sum[d] + 0; c = cov_sum[d] + 0
      printf "DETAIL\t%s\t%d\t%d\t%.2f\n", d, s, c, (s > 0 ? c * 100 / s : 0)
    }
    for (i = 1; i <= np; i++) {
      p = prefix[i]; s = tstmt[p] + 0; c = tcov[p] + 0
      printf "ROLLUP\t%s\t%d\t%d\t%.2f\t%s\n", p, s, c, (s > 0 ? c * 100 / s : 0), floor[p]
    }
  }
' "$profile" > "$raw"

echo "📉 逐包覆盖率明细（叶子包，按现状升序）："
grep '^DETAIL' "$raw" | sort -t $'\t' -k5,5n | while IFS=$'\t' read -r _ pkg stmt cov pct; do
  printf '   %-58s %6s%%  (%s/%s 语句)\n' "$pkg" "$pct" "$cov" "$stmt"
done

echo "🚦 逐包下限判定（表内包）："
below=0
checked=0
while IFS=$'\t' read -r _ pkg stmt cov pct floor; do
  [ "$pkg" != "" ] || continue
  if [ "$stmt" -eq 0 ]; then
    if [ -z "$floor" ]; then
      printf '⚠️  %-56s 未出现在 profile 中（0 条语句，下限未设）\n' "$pkg"
    else
      printf '❌ %-56s 未出现在 profile 中（下限 %s%%）—— 包被改名/排除或未被插桩\n' "$pkg" "$floor"
      below=$((below + 1))
    fi
    continue
  fi
  if [ -z "$floor" ]; then
    printf '▶️  %-56s %6s%%  下限未设（仅打印）  (%s/%s 语句)\n' "$pkg" "$pct" "$cov" "$stmt"
    continue
  fi
  checked=$((checked + 1))
  if awk -v p="$pct" -v f="$floor" 'BEGIN { exit !(p + 0 < f + 0) }'; then
    need=$(awk -v s="$stmt" -v c="$cov" -v f="$floor" 'BEGIN { d = f * s / 100 - c; if (d < 0) d = 0; printf "%d", (d == int(d) ? d : int(d) + 1) }')
    printf '❌ %-56s %6s%%  下限 %s%%  还差 %s 条语句\n' "$pkg" "$pct" "$floor" "$need"
    below=$((below + 1))
  else
    slack=$(awk -v s="$stmt" -v c="$cov" -v f="$floor" 'BEGIN { printf "%.1f", c - f * s / 100 }')
    printf '✅ %-56s %6s%%  下限 %s%%  余量 +%s 条语句\n' "$pkg" "$pct" "$floor" "$slack"
  fi
done < <(grep '^ROLLUP' "$raw")
rm -f "$raw" "$floorfile"

if [ "$checked" -eq 0 ] && [ "$below" -eq 0 ]; then
  echo "ℹ️  下限表尚未填入数值：本次只打印读数、不判定（两段式的第一段）"
  exit 0
fi
[ "$below" -eq 0 ] || { echo "❌ 有 $below 个包低于覆盖率下限"; exit 1; }
echo "✅ 逐包覆盖率下限门禁通过（共判定 $checked 个包）"
