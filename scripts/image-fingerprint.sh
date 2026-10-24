#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-full}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if command -v sha256sum >/dev/null 2>&1; then
  sha_cmd=(sha256sum)
else
  sha_cmd=(shasum -a 256)
fi

{
  printf 'profile=%s\n' "$PROFILE"
  git ls-files -z --cached --others --exclude-standard -- \
    Dockerfile go.mod go.sum cmd internal tools configs conf docs/openapi |
    while IFS= read -r -d '' path; do
      [ -f "$path" ] || continue
      printf '%s\0' "$path"
      "${sha_cmd[@]}" "$path" | awk '{print $1}'
    done
} | "${sha_cmd[@]}" | awk '{print $1}'
