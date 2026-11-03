#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKFLOW="$ROOT_DIR/.github/workflows/release.yml"
failed=0

check() {
  local pattern="$1"
  if ! grep -Fq "$pattern" "$WORKFLOW"; then
    echo "FAIL: release workflow is missing: $pattern" >&2
    failed=1
  fi
}

check 'build-cli:'
check './cmd/jimu'
check 'jimu.BundleVersion'
check 'jimu.BundleCommit'
check 'template-bundle.json'
check 'sha256sum'
check 'jimu-${GITHUB_REF_NAME}-checksums.txt'
check 'goos: linux'
check 'goarch: amd64'
check 'goos: darwin'
check 'goarch: arm64'

if [ "$failed" -ne 0 ]; then
  echo "FAIL: release workflow contract is incomplete" >&2
  exit 1
fi
echo "PASS: release workflow contract"
