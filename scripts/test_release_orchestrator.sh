#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SCRIPT="$ROOT_DIR/scripts/release_orchestrator.sh"
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/jimu-release-orchestrator.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

FAKE_BIN="$TMP_DIR/bin"
mkdir -p "$FAKE_BIN"
FAKE_GH_LOG="$TMP_DIR/gh.log"

cat > "$FAKE_BIN/gh" <<'FAKE_GH'
#!/usr/bin/env bash
set -euo pipefail

log_file=${FAKE_GH_LOG:?}
printf '%s\n' "$*" >> "$log_file"

if [[ "${1:-}" != api ]]; then
  echo "unsupported fake gh command: $*" >&2
  exit 2
fi
shift

method=GET
endpoint=""
while (($#)); do
  case "$1" in
    -X|--method)
      method=$2
      shift 2
      ;;
    --input|-f|-F|--raw-field|--field|--jq)
      shift 2
      ;;
    --paginate|--silent|--include)
      shift
      ;;
    --header)
      shift 2
      ;;
    *)
      if [[ -z "$endpoint" && "$1" != -* ]]; then
        endpoint=$1
      fi
      shift
      ;;
  esac
done

scenario=${FAKE_GH_SCENARIO:-}
case "$scenario:$method:$endpoint" in
  validate:*)
    echo '{}'
    ;;
  ensure-cycle:GET:repos/wangkuntian/jimu/issues)
    if [[ ${FAKE_CYCLE_STATE:-new} == duplicate ]]; then
      echo '[{"number":99,"title":"release: v0.3.6","labels":[{"name":"release: collecting"}]}]'
    else
      echo '[]'
    fi
    ;;
  ensure-cycle:GET:repos/wangkuntian/jimu/git/ref/heads/master)
    echo '{"ref":"refs/heads/master","object":{"sha":"master-sha"}}'
    ;;
  ensure-cycle:GET:repos/wangkuntian/jimu/git/ref/tags/v0.3.5)
    exit 1
    ;;
  ensure-cycle:GET:repos/wangkuntian/jimu/git/ref/heads/release/v0.3.5|ensure-cycle:GET:repos/wangkuntian/jimu/git/ref/heads/dependabot-updates)
    if [[ ${FAKE_CYCLE_STATE:-new} == existing ]]; then
      echo '{"ref":"refs/heads/dependabot-updates","object":{"sha":"dependabot-sha"}}'
    else
      exit 1
    fi
    ;;
  ensure-cycle:GET:repos/wangkuntian/jimu/issues/77/comments)
    echo '[{"body":"<!-- jimu-release-automation:v0.3.5 -->"}]'
    ;;
  ensure-cycle:POST:repos/wangkuntian/jimu/git/refs)
    echo '{"ref":"refs/heads/release/v0.3.5","object":{"sha":"release-sha"}}'
    ;;
  ensure-cycle:POST:repos/wangkuntian/jimu/issues/77/comments)
    echo '{"id":1}'
    ;;
  collection-ready:GET:repos/wangkuntian/jimu/issues/77)
    if [[ ${FAKE_COLLECTION_STATE:-ready} == young ]]; then
      echo '{"created_at":"2026-09-28T00:00:00Z"}'
    else
      echo '{"created_at":"2026-09-20T00:00:00Z"}'
    fi
    ;;
  collection-ready:GET:repos/wangkuntian/jimu/pulls*)
    if [[ ${FAKE_COLLECTION_STATE:-ready} == open-pr ]]; then
      echo '[{"number":88,"user":{"login":"dependabot[bot]"},"state":"open"}]'
    else
      echo '[]'
    fi
    ;;
  collection-ready:GET:repos/wangkuntian/jimu/commits*)
    if [[ ${FAKE_COLLECTION_STATE:-ready} == recent ]]; then
      echo '[{"sha":"dependabot-sha","commit":{"committer":{"date":"2026-09-29T18:00:00Z"}}}]'
    else
      echo '[{"sha":"dependabot-sha","commit":{"committer":{"date":"2026-09-25T00:00:00Z"}}}]'
    fi
    ;;
  collection-ready:GET:repos/wangkuntian/jimu/compare/*)
    echo '{"ahead_by":1}'
    ;;
  create-snapshot:GET:repos/wangkuntian/jimu/git/ref/heads/dependabot-updates)
    echo '{"ref":"refs/heads/dependabot-updates","object":{"sha":"dependabot-sha"}}'
    ;;
  create-snapshot:GET:repos/wangkuntian/jimu/git/ref/heads/updates/v0.3.5)
    if [[ ${FAKE_SNAPSHOT_STATE:-new} == existing ]]; then
      echo '{"ref":"refs/heads/updates/v0.3.5","object":{"sha":"dependabot-sha"}}'
    else
      exit 1
    fi
    ;;
  create-snapshot:GET:repos/wangkuntian/jimu/pulls*)
    if [[ ${FAKE_SNAPSHOT_STATE:-new} == existing ]]; then
      echo '[{"number":123,"state":"open","head":{"ref":"updates/v0.3.5"},"base":{"ref":"release/v0.3.5"}}]'
    else
      echo '[]'
    fi
    ;;
  create-snapshot:POST:repos/wangkuntian/jimu/git/refs)
    echo '{"ref":"refs/heads/updates/v0.3.5","object":{"sha":"dependabot-sha"}}'
    ;;
  create-snapshot:POST:repos/wangkuntian/jimu/pulls)
    echo '{"number":123,"state":"open"}'
    ;;
  tag-after-merge:GET:repos/wangkuntian/jimu/pulls/42)
    echo '{"merged":true,"merged_at":"2026-09-30T01:00:00Z","head":{"ref":"release/v0.3.5"},"base":{"ref":"master"},"merge_commit_sha":"merge-sha"}'
    ;;
  tag-after-merge:GET:repos/wangkuntian/jimu/git/ref/heads/master)
    if [[ ${FAKE_TAG_STATE:-match} == mismatch ]]; then
      echo '{"ref":"refs/heads/master","object":{"sha":"other-sha"}}'
    else
      echo '{"ref":"refs/heads/master","object":{"sha":"merge-sha"}}'
    fi
    ;;
  tag-after-merge:GET:repos/wangkuntian/jimu/git/ref/tags/v0.3.5)
    exit 1
    ;;
  tag-after-merge:POST:repos/wangkuntian/jimu/git/tags)
    echo '{"sha":"annotated-tag-sha"}'
    ;;
  tag-after-merge:POST:repos/wangkuntian/jimu/git/refs)
    echo '{"ref":"refs/tags/v0.3.5","object":{"sha":"annotated-tag-sha"}}'
    ;;
  *)
    echo "unhandled fake gh request: $scenario $method $endpoint" >&2
    exit 2
    ;;
esac
FAKE_GH
chmod +x "$FAKE_BIN/gh"

run_orchestrator() {
  env \
    PATH="$FAKE_BIN:$PATH" \
    GH_REPO=wangkuntian/jimu \
    GH_TOKEN=test-token \
    FAKE_GH_LOG="$FAKE_GH_LOG" \
    RELEASE_VERSION=v0.3.5 \
    RELEASE_ISSUE_NUMBER=77 \
    NOW=2026-09-30T00:00:00Z \
    "$SCRIPT" "$@"
}

assert_contains() {
  local haystack=$1 needle=$2 message=$3
  if [[ "$haystack" != *"$needle"* ]]; then
    printf 'ASSERT FAILED: %s\noutput: %s\n' "$message" "$haystack" >&2
    exit 1
  fi
}

assert_status() {
  local expected=$1 actual=$2 message=$3
  if [[ "$expected" != "$actual" ]]; then
    printf 'ASSERT FAILED: %s (expected %s, got %s)\n' "$message" "$expected" "$actual" >&2
    exit 1
  fi
}

test_validate_version() {
  local output
  output=$(run_orchestrator validate-version)
  assert_contains "$output" 'version=v0.3.5' 'valid version is normalized'
  assert_contains "$output" 'release_branch=release/v0.3.5' 'release branch is derived'
  assert_contains "$output" 'snapshot_branch=updates/v0.3.5' 'snapshot branch is derived'

  set +e
  output=$(env PATH="$FAKE_BIN:$PATH" GH_REPO=wangkuntian/jimu GH_TOKEN=test RELEASE_VERSION=0.3.5 "$SCRIPT" validate-version 2>&1)
  local status=$?
  set -e
  assert_status 1 "$status" 'missing v prefix is rejected'
  assert_contains "$output" '::error::invalid release version' 'invalid version emits stable error'
}

test_ensure_cycle() {
  local output
  : > "$FAKE_GH_LOG"
  output=$(FAKE_GH_SCENARIO=ensure-cycle run_orchestrator ensure-cycle)
  assert_contains "$output" 'release_branch=release/v0.3.5' 'ensure-cycle reports release branch'
  assert_contains "$output" 'dependabot_branch=dependabot-updates' 'ensure-cycle reports fixed Dependabot branch'
  assert_contains "$(cat "$FAKE_GH_LOG")" 'api repos/wangkuntian/jimu/git/refs --method POST' 'ensure-cycle creates missing refs'
  assert_contains "$(cat "$FAKE_GH_LOG")" 'api repos/wangkuntian/jimu/issues/77/comments --method POST' 'ensure-cycle records managed resources'

  : > "$FAKE_GH_LOG"
  set +e
  output=$(FAKE_GH_SCENARIO=ensure-cycle FAKE_CYCLE_STATE=duplicate run_orchestrator ensure-cycle 2>&1)
  local status=$?
  set -e
  assert_status 1 "$status" 'a second active cycle is rejected'
  assert_contains "$output" '::error::another active release cycle exists' 'duplicate cycle emits stable error'

  : > "$FAKE_GH_LOG"
  output=$(FAKE_GH_SCENARIO=ensure-cycle FAKE_CYCLE_STATE=existing run_orchestrator ensure-cycle)
  if grep -q -- '--method POST' "$FAKE_GH_LOG"; then
    printf 'ASSERT FAILED: existing managed cycle was modified\n' >&2
    exit 1
  fi
}

test_collection_window() {
  local output
  output=$(FAKE_GH_SCENARIO=collection-ready FAKE_COLLECTION_STATE=ready run_orchestrator collection-ready)
  assert_contains "$output" 'ready=true' 'ready cycle passes the collection window'

  output=$(FAKE_GH_SCENARIO=collection-ready FAKE_COLLECTION_STATE=young run_orchestrator collection-ready)
  assert_contains "$output" 'ready=false' 'young cycle is held'

  output=$(FAKE_GH_SCENARIO=collection-ready FAKE_COLLECTION_STATE=open-pr run_orchestrator collection-ready)
  assert_contains "$output" 'ready=false' 'open Dependabot PR blocks collection'

  output=$(FAKE_GH_SCENARIO=collection-ready FAKE_COLLECTION_STATE=recent run_orchestrator collection-ready)
  assert_contains "$output" 'ready=false' 'recent update resets quiet window'
}

test_snapshot() {
  local output
  : > "$FAKE_GH_LOG"
  output=$(FAKE_GH_SCENARIO=create-snapshot run_orchestrator create-snapshot)
  assert_contains "$output" 'snapshot_branch=updates/v0.3.5' 'snapshot branch is created'
  assert_contains "$output" 'pr_number=123' 'snapshot PR number is returned'

  : > "$FAKE_GH_LOG"
  output=$(FAKE_GH_SCENARIO=create-snapshot FAKE_SNAPSHOT_STATE=existing run_orchestrator create-snapshot)
  assert_contains "$output" 'pr_number=123' 'existing snapshot PR is reused'
  if grep -q -- '--method POST' "$FAKE_GH_LOG"; then
    printf 'ASSERT FAILED: existing snapshot branch or PR was modified\n' >&2
    exit 1
  fi
}

test_tag_guard() {
  local output
  output=$(FAKE_GH_SCENARIO=tag-after-merge FAKE_TAG_STATE=match run_orchestrator tag-after-merge v0.3.5 42)
  assert_contains "$output" 'tag=v0.3.5' 'matching master tip allows tag creation'

  set +e
  output=$(FAKE_GH_SCENARIO=tag-after-merge FAKE_TAG_STATE=mismatch run_orchestrator tag-after-merge v0.3.5 42 2>&1)
  local status=$?
  set -e
  assert_status 1 "$status" 'stale merge commit is rejected'
  assert_contains "$output" '::error::merge commit is not current master tip' 'tag guard emits stable error'
}

test_dependabot_config() {
  local config="$ROOT_DIR/.github/dependabot.yml"
  local ecosystem
  for ecosystem in gomod github-actions docker; do
    if ! awk -v wanted="\"$ecosystem\"" '
      /^  - package-ecosystem:/ {
        in_block = index($0, wanted) > 0
        found = found || in_block
        next
      }
      in_block && /target-branch:[[:space:]]*"?dependabot-updates"?/ { target = 1 }
      END { exit !(found && target) }
    ' "$config"; then
      printf 'ASSERT FAILED: %s is missing target-branch: dependabot-updates\n' "$ecosystem" >&2
      exit 1
    fi
  done
  if grep -Eq 'security-updates|security-only|target-branch:[[:space:]]*master' "$config"; then
    printf 'ASSERT FAILED: dependabot config contains a security target override\n' >&2
    exit 1
  fi
}

test_workflow_contract() {
  local workflow="$ROOT_DIR/.github/workflows/release-dependency-automation.yml"
  [[ -f "$workflow" ]] || {
    printf 'ASSERT FAILED: release orchestration workflow is missing\n' >&2
    exit 1
  }
  local required
  for required in \
    'issues:' \
    'schedule:' \
    'pull_request:' \
    'workflow_run:' \
    'concurrency:' \
    'actions/create-github-app-token' \
    'release_orchestrator.sh validate-version' \
    'release_orchestrator.sh ensure-cycle' \
    'release_orchestrator.sh collection-ready' \
    'release_orchestrator.sh create-snapshot' \
    'release_orchestrator.sh tag-after-merge'; do
    if ! grep -Fq "$required" "$workflow"; then
      printf 'ASSERT FAILED: workflow contract is missing %s\n' "$required" >&2
      exit 1
    fi
  done
  grep -Fq 'group: release-dependency-automation' "$workflow" || {
    printf 'ASSERT FAILED: workflow does not serialize release cycles globally\n' >&2
    exit 1
  }
  grep -Fq 'cancel-in-progress: false' "$workflow" || {
    printf 'ASSERT FAILED: workflow may cancel a concurrent release cycle\n' >&2
    exit 1
  }
  for required in 'CI (Go)' 'CI (Docker)' 'CI (Commits)' 'gh pr checks' 'gh pr merge'; do
    if ! grep -Fq "$required" "$workflow"; then
      printf 'ASSERT FAILED: aggregate merge gate is missing %s\n' "$required" >&2
      exit 1
    fi
  done
  if grep -Fq -- '--auto' "$workflow"; then
    printf 'ASSERT FAILED: aggregate merge relies on GitHub auto-merge without required checks\n' >&2
    exit 1
  fi
}

test_dependabot_merge_contract() {
  local workflow="$ROOT_DIR/.github/workflows/dependabot-auto-merge.yml"
  grep -Fq 'workflow_run:' "$workflow" || {
    printf 'ASSERT FAILED: Dependabot merge must wait for a completed focused workflow\n' >&2
    exit 1
  }
  grep -Fq 'CI (Dependabot Focused)' "$workflow" || {
    printf 'ASSERT FAILED: Dependabot merge does not subscribe to focused checks\n' >&2
    exit 1
  }
  grep -Fq 'gh pr merge' "$workflow" || {
    printf 'ASSERT FAILED: Dependabot merge command is missing\n' >&2
    exit 1
  }
  if grep -Fq -- '--auto' "$workflow"; then
    printf 'ASSERT FAILED: Dependabot merge relies on GitHub auto-merge without required checks\n' >&2
    exit 1
  fi
}

test_scaffold_policy() {
  local workflow="$ROOT_DIR/.github/workflows/ci-scaffold.yml"
  local required
  for required in \
    "github.event.pull_request.user.login" \
    "dependabot[bot]" \
    "Scaffold Matrix skipped for Dependabot security update" \
    "github.event.pull_request.base.ref" \
    "startsWith(github.head_ref, 'release/')"; do
    if ! grep -Fq "$required" "$workflow"; then
      printf 'ASSERT FAILED: scaffold policy is missing %s\n' "$required" >&2
      exit 1
    fi
  done
}

case "${1:-core}" in
  core)
    test_validate_version
    test_ensure_cycle
    test_collection_window
    test_snapshot
    test_tag_guard
    ;;
  --dependabot-config)
    test_dependabot_config
    ;;
  --workflow-contract)
    test_workflow_contract
    ;;
  --dependabot-merge-contract)
    test_dependabot_merge_contract
    ;;
  --scaffold-policy)
    test_scaffold_policy
    ;;
  *)
    echo "unsupported test group: $1" >&2
    exit 2
    ;;
esac

echo 'release orchestrator tests passed'
