#!/usr/bin/env bash
# Validate that commit messages contain no CJK characters.
# Enforces the all-English rule from AGENTS.md「Commit Message 规范」;
# mirrors githooks/commit-msg so local hooks and CI share one implementation.
#
# Usage:
#   scripts/check_commit_messages.sh <base> <head>   # check commits in <base>..<head>
#
# Fallbacks:
#   - empty base or zero SHA (new branch), or unresolvable base (force push):
#     only the <head> tip commit is checked
#   - failed per-commit checks accumulate; exits 1 with a summary at the end
set -euo pipefail

base="${1:-}"
head="${2:-}"
if [ -z "$head" ]; then
  echo "usage: $0 <base> <head>" >&2
  exit 2
fi

hook="$(cd "$(dirname "$0")/.." && pwd)/githooks/commit-msg"

if [ -n "$base" ] \
   && [ "$base" != "0000000000000000000000000000000000000000" ] \
   && git cat-file -e "$base" 2>/dev/null; then
  shas="$(git rev-list "$base..$head")"
else
  shas="$head"  # new branch / unavailable base: check the tip commit only
fi

if [ -z "$shas" ]; then
  echo "✅ no commits to check"
  exit 0
fi

fail=0
total=0
for sha in $shas; do
  total=$((total + 1))
  tmp="$(mktemp)"
  git log -1 --format=%B "$sha" > "$tmp"
  if ! "$hook" "$tmp"; then
    echo "  ✗ $(git log -1 --format='%h %s' "$sha")" >&2
    fail=$((fail + 1))
  fi
  rm -f "$tmp"
done

if [ "$fail" -ne 0 ]; then
  echo "❌ $fail/$total commit message(s) contain CJK characters (all-English rule, see AGENTS.md)" >&2
  exit 1
fi
echo "✅ commit message check passed ($total commit(s))"
