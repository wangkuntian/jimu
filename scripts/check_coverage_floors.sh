#!/usr/bin/env bash
# 逐包「不低于」覆盖率下限门禁（与 scripts/check_ci_gate.sh 同风格：单一实现 + 可本地单测）。
#
# 输入：`go tool cover` 的**文本** profile（`scripts/test_shards.sh merge-cover` 的产物）。
# 解析只用 awk/grep/sort，**不需要 Go 工具链** —— CI（DB 口径）与本地（需 SQLite/无 DB 时用
# `SKIP_COVERAGE_FLOORS=1`）都能跑。
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
# 环境变量 SKIP_COVERAGE_FLOORS=1 跳过本判定（本地 `make ci` 无 DB 口径时使用；CI 不设该变量）。
set -euo pipefail

# 本地豁免开关：本机 `make ci` 无 DB 口径 ⇒ `*/infrastructure` 测试 SKIP ⇒ 读数天然低于
# 取自 CI（DB 口径）的下限。CI 的 `Coverage & Docs` job 不设该变量，判定照常执行。
if [ "${SKIP_COVERAGE_FLOORS:-}" = "1" ]; then
  echo "ℹ️ SKIP_COVERAGE_FLOORS=1：跳过逐包下限判定（本地无 DB 口径时使用；CI 不设该变量）"
  exit 0
fi

# 下限表：一处定义；`前缀|<下限%>`。下限留空 = 只打印不判定。
# 仅包含本刀涉及的包（spec §5）；数值由校准 run 填入（基线 − 0.5pp，见 Task 9）。
#
# ⚠️ 不要加入 `tools/generator` / `tools/composereport` 这类「多分片跑同一个包（不同 `-run`）」
# 的前缀：合并 profile 会含重复 block（实测 819 个：generator 721 / composereport 98），而本
# 脚本按朴素语句加权聚合 ⇒ 读数会被按 5×/4× 计权而偏高（`go tool cover -func` 会去重，故 CI
# 的 total 不受影响）。当前 8 个前缀零重复（已在合并 profile 上核过）。
FLOORS='
jimu/internal/capabilities/tenant|48.2
jimu/internal/capabilities/user|61.4
jimu/internal/capabilities/access|69.2
jimu/internal/capabilities/audit|70.7
jimu/internal/capabilities/apikey|79.6
jimu/internal/kernel/tenant|91.1
jimu/internal/kernel/auth|51.4
jimu/internal/kernel/http/middleware|91.3
'

floors="${COVERAGE_FLOORS:-$FLOORS}"

# 下限表格式校验（fail closed）：一行都不许含糊。畸形行若放行，会以两种方式让门禁
# 静默消失（fail open）——漏掉 `|` 被当成「下限未设（仅打印）」，非法下限值被 awk 的
# `+ 0` 当成 0 而报「✅ 余量」。故：格式不符即 stderr 列出非法行，exit 2。
badfloors=$(printf '%s\n' "$floors" | awk '
  {
    line = $0
    gsub(/^[ \t]+|[ \t]+$/, "", line)
    if (line == "") next
    if (line !~ /^[^|]+\|([0-9]+(\.[0-9]+)?)?$/) print line
  }
')
if [ -n "$badfloors" ]; then
  echo "❌ 下限表格式非法：每行应为「<目录前缀>|<下限%>」，下限留空或为数字（如 50.5）。非法行：" >&2
  while IFS= read -r badline; do printf '   %s\n' "$badline" >&2; done <<< "$badfloors"
  exit 2
fi

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
