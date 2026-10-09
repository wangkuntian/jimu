#!/usr/bin/env bash
# 依赖漏洞扫描（govulncheck）。扫描出现任意可达漏洞都会失败。
set -euo pipefail

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

cat "$out"
echo "❌ govulncheck: 发现可达漏洞（见上方输出）"
exit 1
