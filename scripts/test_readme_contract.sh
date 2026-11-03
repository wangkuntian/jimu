#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
README="$ROOT_DIR/README.md"
failed=0

require_text() {
  local file="$1"
  local text="$2"
  if ! grep -Fq "$text" "$file"; then
    echo "FAIL: $file is missing: $text" >&2
    failed=1
  fi
}

for doc in CAPABILITIES CONFIGURATION OPERATIONS UPGRADING COMPARISON PRODUCTION; do
  file="$ROOT_DIR/docs/$doc.md"
  if [ ! -s "$file" ]; then
    echo "FAIL: missing or empty docs/$doc.md" >&2
    failed=1
  fi
done

require_text "$README" "Go 后端项目脚手架/工程基线"
require_text "$README" "basic-api"
require_text "$README" "auth-api"
require_text "$README" "docs/CAPABILITIES.md"
require_text "$README" "docs/CONFIGURATION.md"
require_text "$README" "docs/OPERATIONS.md"
require_text "$README" "docs/UPGRADING.md"
require_text "$README" "docs/COMPARISON.md"
require_text "$README" "docs/PRODUCTION.md"
require_text "$ROOT_DIR/docs/COMPARISON.md" "Gin"
require_text "$ROOT_DIR/docs/COMPARISON.md" "Echo"
require_text "$ROOT_DIR/docs/COMPARISON.md" "go-zero"
require_text "$ROOT_DIR/docs/COMPARISON.md" "Kratos"
require_text "$ROOT_DIR/docs/PRODUCTION.md" "P1-02"
require_text "$ROOT_DIR/docs/PRODUCTION.md" "未完成"

if [ "$failed" -ne 0 ]; then
  echo "FAIL: README contract is incomplete" >&2
  exit 1
fi
echo "PASS: README contract"
