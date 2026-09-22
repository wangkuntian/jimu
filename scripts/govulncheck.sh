#!/usr/bin/env bash
# 依赖漏洞扫描（govulncheck），带一份**极窄**的临时豁免清单。
#
# 为什么需要豁免：GO-2026-6452（excelize 负共享字符串索引 panic，可达路径为
# internal/capabilities/dataops/excel 的 readSheet（其中调用 excelize 的 f.GetRows）；
# P2.5 驱动拆包后 Excel 导入实现从 dataops/importer 移到该驱动包）在上游尚未发布修复版本，
# govulncheck 报 "Fixed in: N/A"，会把 master 上所有 PR 卡红。代码侧已在 readSheet 加 recover
# 兜底，使该 panic 只表现为一个错误、不会打挂请求或任务 goroutine；因此这里临时豁免该 ID。
# 上游修复已进主干（rows.go 的 index < 0 防护），等 excelize 发布 v2.11.1 后删除本豁免。
# P2.5 的驱动拆包**不解除**可达性：govulncheck ./... 扫整个 module，full 形态仍 import
# dataops/excel，因此本豁免在 excelize 升级前必须保留。
#
# 规则：只豁免清单内的 ID；出现任何其他漏洞（含将来新发布的）一律失败。
set -euo pipefail

ALLOWLIST=(
  # 复核条件：excelize 发布 v2.11.1（或含 rows.go 负索引防护的正式版本）后移除
  GO-2026-6452
)

out="$(mktemp)"
trap 'rm -f "$out"' EXIT

set +e
go run golang.org/x/vuln/cmd/govulncheck@latest ./... >"$out" 2>&1
status=$?
set -e

if [ "$status" -eq 0 ]; then
  cat "$out"
  echo "✅ govulncheck: 未发现可达漏洞"
  exit 0
fi

# 可达漏洞 ID（默认输出中的 "Vulnerability #N: GO-XXXX-XXXX"）
ids="$(grep -oE 'Vulnerability #[0-9]+:[[:space:]]+GO-[0-9]{4}-[0-9]+' "$out" | awk '{print $NF}' | sort -u || true)"

unexpected=""
for id in $ids; do
  allowed=0
  for entry in "${ALLOWLIST[@]}"; do
    if [ "$id" = "$entry" ]; then
      allowed=1
      break
    fi
  done
  if [ "$allowed" -eq 0 ]; then
    unexpected="$unexpected $id"
  fi
done

# 有命中且全部在豁免清单内 → 通过；否则（含 govulncheck 自身出错、无可解析 ID）失败
if [ -n "$ids" ] && [ -z "$unexpected" ]; then
  # 默认不打印原始输出：其中的 "Your code is affected by …" 代码帧会被 CI 渲染成
  # 失败级注解（Annotations 面板），而本分支实际是通过的。需要排查时用
  # GOVULNCHECK_VERBOSE=1 make govulncheck 查看完整输出。
  if [ "${GOVULNCHECK_VERBOSE:-0}" = "1" ]; then
    cat "$out"
  else
    echo "ℹ️  已豁免的命中：$(echo "$ids" | tr '\n' ' ')"
    echo "   完整输出：GOVULNCHECK_VERBOSE=1 make govulncheck"
  fi
  echo "⚠️  govulncheck: 仅命中临时豁免清单（${ALLOWLIST[*]}），已在 internal/capabilities/dataops/excel 的 readSheet recover 兜底，视为通过"
  exit 0
fi

# 未豁免：保留完整输出，注解即证据
cat "$out"
echo "❌ govulncheck: 发现未豁免的可达漏洞（${unexpected:-扫描未成功，见上方输出}）"
exit 1
