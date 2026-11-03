#!/bin/sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

cd "$repo_root"
go build -o "$work/jimu" ./cmd/jimu
cp -R tools/generator/testdata/upgrade/v0.5/basic-api "$work/project"

"$work/jimu" check "$work/project"
"$work/jimu" upgrade "$work/project" --check
"$work/jimu" upgrade "$work/project"
"$work/jimu" check "$work/project"

grep -q 'func SecurityHeaders()' "$work/project/internal/platform/http/middleware.go"
grep -q 'platformhttp.SecurityHeaders()' "$work/project/internal/app/router.go"
grep -q 'platformhttp.CORS(' "$work/project/internal/app/router.go"

echo "PASS: v0.5 basic-api upgraded to the current CLI with security headers and CORS"
