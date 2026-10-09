#!/usr/bin/env bash
set -euo pipefail

error() {
  printf '::error::%s\n' "$*" >&2
  exit 1
}

usage() {
  cat >&2 <<'EOF'
usage: release_orchestrator.sh <validate-version|ensure-cycle|collection-ready|create-snapshot|tag-after-merge>
EOF
  exit 2
}

command -v gh >/dev/null 2>&1 || error "gh CLI is required"
command -v jq >/dev/null 2>&1 || error "jq is required"

repo=${GH_REPO:-}
[[ -n "$repo" ]] || error "GH_REPO is required"

api() {
  local endpoint=$1 arg has_method=false
  shift
  for arg in "$@"; do
    if [[ "$arg" == --method || "$arg" == -X ]]; then
      has_method=true
      break
    fi
  done
  if [[ "$has_method" == true ]]; then
    gh api "$endpoint" "$@"
  else
    gh api "$endpoint" --method GET "$@"
  fi
}

write_output() {
  local key=$1 value=$2
  printf '%s=%s\n' "$key" "$value"
  if [[ -n ${GITHUB_OUTPUT:-} ]]; then
    printf '%s=%s\n' "$key" "$value" >> "$GITHUB_OUTPUT"
  fi
}

version_value() {
  local value=${1:-${RELEASE_VERSION:-}}
  [[ -n "$value" ]] || error "release version is required"
  [[ "$value" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || error "invalid release version: $value"
  printf '%s\n' "$value"
}

version_context() {
  local value
  value=$(version_value "${1:-}")
  VERSION=$value
  RELEASE_BRANCH="release/$value"
  AGGREGATE_BRANCH=dependabot-updates
}

ref_json() {
  local kind=$1 name=$2
  api "repos/$repo/git/ref/$kind/$name"
}

ref_sha() {
  local kind=$1 name=$2 json
  if ! json=$(ref_json "$kind" "$name" 2>/dev/null); then
    return 1
  fi
  jq -er '.object.sha' <<<"$json"
}

ref_exists() {
  ref_sha "$1" "$2" >/dev/null
}

create_ref() {
  local kind=$1 name=$2 sha=$3
  api "repos/$repo/git/refs" \
    --method POST \
    --field "ref=refs/$kind/$name" \
    --field "sha=$sha" >/dev/null
}

issue_comments() {
  local issue=$1
  api "repos/$repo/issues/$issue/comments" --paginate
}

record_cycle_comment() {
  local issue=$1 version=$2 release_branch=$3 dependabot_branch=$4 release_sha=$5
  local body
  body=$(cat <<EOF
<!-- jimu-release-automation:$version -->
Release cycle \`$version\` is managed by the automation.

- release branch: \`$release_branch\`
- Dependabot branch: \`$dependabot_branch\`
- release base: \`$release_sha\`
EOF
)
  api "repos/$repo/issues/$issue/comments" --method POST --field "body=$body" >/dev/null
}

reset_terminal_dependabot_cycle() {
  local current_issue=$1 issue_json previous_issue="" previous_version="" previous_base="" number title labels comments marker
  issue_json=$(api "repos/$repo/issues" --paginate --field state=all --field per_page=100)
  while IFS=$'\t' read -r number title labels; do
    [[ -n "$number" ]] || continue
    [[ ",$labels," == *"release: published"* || ",$labels," == *"release: blocked"* ]] || continue
    comments=$(issue_comments "$number" 2>/dev/null || true)
    marker=$(grep -Eo '<!-- jimu-release-automation:v[0-9]+\.[0-9]+\.[0-9]+ -->' <<<"$comments" | head -n1 || true)
    previous_base=$(grep -Eo 'release base: `[0-9a-f]{40}`' <<<"$comments" | head -n1 | sed -E 's/.*`([0-9a-f]{40})`.*/\1/' || true)
    if [[ "$marker" =~ jimu-release-automation:(v[0-9]+\.[0-9]+\.[0-9]+) ]]; then
      [[ "$title" == "release: ${BASH_REMATCH[1]}" ]] || continue
      previous_issue=$number
      previous_version=${BASH_REMATCH[1]}
      [[ -n "$previous_base" ]] && break
    fi
  done < <(jq -r '.[] | select(.pull_request == null) | [.number, .title, ([.labels[]?.name] | join(","))] | @tsv' <<<"$issue_json")

  [[ -n "$previous_issue" && "$previous_issue" != "$current_issue" && -n "$previous_version" && -n "$previous_base" ]] || \
    error "dependabot-updates exists without a terminal managed release cycle"

  local dependabot_sha comparison merge_base
  dependabot_sha=$(ref_sha heads dependabot-updates) || error "fixed Dependabot branch ref is unavailable"
  comparison=$(api "repos/$repo/compare/$previous_base...$dependabot_sha")
  merge_base=$(jq -r '.merge_base_commit.sha // empty' <<<"$comparison")
  [[ "$merge_base" == "$previous_base" ]] || \
    error "dependabot-updates does not descend from the terminal cycle baseline"

  local pulls number author body
  pulls=$(api "repos/$repo/pulls" --paginate --field state=open --field base=dependabot-updates --field per_page=100)
  while IFS=$'\t' read -r number author body; do
    [[ -n "$number" ]] || continue
    if [[ "$author" != "dependabot[bot]" ]]; then
      [[ -n ${RELEASE_APP_SLUG:-} && "$author" == "${RELEASE_APP_SLUG}[bot]" && \
         "$body" == *"<!-- jimu-release-automation:$previous_version dependency:"* ]] || \
        error "unmanaged open PR targets dependabot-updates: #$number"
    fi
    api "repos/$repo/pulls/$number" --method PATCH --field state=closed >/dev/null || \
      error "failed to close stale Dependabot PR #$number"
  done < <(jq -r '.[] | [.number, .user.login, (.body // "")] | @tsv' <<<"$pulls")

  api "repos/$repo/git/refs/heads/dependabot-updates" --method DELETE >/dev/null || \
    error "failed to delete terminal cycle dependabot-updates branch"
}

ensure_cycle() {
  version_context "${1:-}"
  local issue=${RELEASE_ISSUE_NUMBER:-}
  [[ "$issue" =~ ^[0-9]+$ ]] || error "RELEASE_ISSUE_NUMBER is required"

  if ref_exists tags "$VERSION"; then
    error "release tag already exists: $VERSION"
  fi

  local active_issues line number title labels
  active_issues=$(api "repos/$repo/issues" --paginate --field state=open --field per_page=100)
  while IFS=$'\t' read -r number title labels; do
    [[ -n "$number" ]] || continue
    [[ "$number" == "$issue" ]] && continue
    [[ "$title" =~ ^release:\ v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || continue
    if [[ ",$labels," != *,release:\ published,* && ",$labels," != *,release:\ blocked,* ]]; then
      error "another active release cycle exists: issue #$number ($title)"
    fi
  done < <(jq -r '.[] | select(.pull_request == null) | [.number, .title, ([.labels[]?.name] | join(","))] | @tsv' <<<"$active_issues")

  local master_sha release_sha dependabot_sha current_comments current_base release_ref_sha fixed_ref_sha comparison merge_base
  current_comments=$(issue_comments "$issue" 2>/dev/null || true)
  if grep -Fq "<!-- jimu-release-automation:$VERSION -->" <<<"$current_comments"; then
    managed_comment=true
    current_base=$(grep -Eo 'release base: `[0-9a-f]{40}`' <<<"$current_comments" | tail -n1 | sed -E 's/.*`([0-9a-f]{40})`.*/\1/' || true)
    [[ "$current_base" =~ ^[0-9a-f]{40}$ ]] || error "release Issue marker is missing a valid baseline"
    release_sha=$current_base
  else
    master_sha=$(ref_sha heads master) || error "master branch ref is unavailable"
    release_sha=$master_sha
    if release_ref_sha=$(ref_sha heads "$RELEASE_BRANCH"); then
      error "release branch already exists and is not managed"
    fi
    if ref_exists heads dependabot-updates; then
      reset_terminal_dependabot_cycle "$issue"
    fi
    record_cycle_comment "$issue" "$VERSION" "$RELEASE_BRANCH" dependabot-updates "$release_sha"
  fi

  if release_ref_sha=$(ref_sha heads "$RELEASE_BRANCH"); then
    comparison=$(api "repos/$repo/compare/$release_sha...$release_ref_sha")
    merge_base=$(jq -r '.merge_base_commit.sha // empty' <<<"$comparison")
    [[ "$merge_base" == "$release_sha" ]] || error "release branch does not descend from the managed baseline"
  else
    create_ref heads "$RELEASE_BRANCH" "$release_sha" || error "failed to create $RELEASE_BRANCH"
    release_ref_sha=$release_sha
  fi

  if fixed_ref_sha=$(ref_sha heads dependabot-updates); then
    comparison=$(api "repos/$repo/compare/$release_sha...$fixed_ref_sha")
    merge_base=$(jq -r '.merge_base_commit.sha // empty' <<<"$comparison")
    [[ "$merge_base" == "$release_sha" ]] || error "dependabot-updates does not descend from the active release baseline"
    dependabot_sha=$fixed_ref_sha
  else
    create_ref heads dependabot-updates "$release_sha" || error "failed to create dependabot-updates"
    dependabot_sha=$release_sha
  fi
  write_output version "$VERSION"
  write_output release_branch "$RELEASE_BRANCH"
  write_output dependabot_branch dependabot-updates
  write_output release_sha "$release_sha"
  write_output dependabot_sha "$dependabot_sha"
}

collection_ready() {
  version_context "${1:-}"
  local issue=${RELEASE_ISSUE_NUMBER:-}
  [[ "$issue" =~ ^[0-9]+$ ]] || error "RELEASE_ISSUE_NUMBER is required"

  local pulls dependabot_count
  pulls=$(api "repos/$repo/pulls" --paginate --field state=open --field base=dependabot-updates --field per_page=100)
  # Every pending update must settle before the aggregate is merged, including CLI App PRs.
  dependabot_count=$(jq 'length' <<<"$pulls")
  if (( dependabot_count > 0 )); then
    write_output ready false
    write_output reason "open Dependabot PRs remain on dependabot-updates"
    return 0
  fi

  local compare changed_files
  compare=$(api "repos/$repo/compare/$RELEASE_BRANCH...dependabot-updates")
  changed_files=$(jq -er '.files | length' <<<"$compare") || error "dependency comparison has no file list"
  if (( changed_files < 1 )); then
    write_output ready false
    write_output reason "no dependency update was merged in this cycle"
    return 0
  fi

  write_output ready true
  write_output reason "dependency changes are ready for CI validation"
  write_output version "$VERSION"
}

create_snapshot() {
  version_context "${1:-}"
  local issue=${RELEASE_ISSUE_NUMBER:-}
  [[ "$issue" =~ ^[0-9]+$ ]] || error "RELEASE_ISSUE_NUMBER is required"

  local compare changed_files pr_list matching_pr pr_number body aggregate_marker
  fixed_sha=$(ref_sha heads dependabot-updates) || error "dependabot-updates ref is unavailable"
  compare=$(api "repos/$repo/compare/$RELEASE_BRANCH...dependabot-updates")
  changed_files=$(jq -er '.files | length' <<<"$compare") || error "dependency comparison has no file list"
  if (( changed_files < 1 )); then
    write_output version "$VERSION"
    write_output aggregate_branch dependabot-updates
    write_output created false
    write_output reason "dependabot-updates has no dependency commit to aggregate"
    return 0
  fi

  aggregate_marker="<!-- jimu-release-automation:$VERSION aggregate -->"
  pr_list=$(api "repos/$repo/pulls" --paginate --field state=open --field head="${repo%%/*}:$AGGREGATE_BRANCH" --field base="$RELEASE_BRANCH" --field per_page=100)
  matching_pr=$(jq -c '[.[] | select(.head.ref == $head and .base.ref == $base)] | .[0] // empty' \
    --arg head "$AGGREGATE_BRANCH" --arg base "$RELEASE_BRANCH" <<<"$pr_list")
  if [[ -n "$matching_pr" ]]; then
    local pr_body
    pr_body=$(jq -r '.body // ""' <<<"$matching_pr")
    [[ "$pr_body" == *"$aggregate_marker"* ]] || error "aggregate PR exists without a matching managed marker"
    pr_number=$(jq -er '.number' <<<"$matching_pr")
  else
    body=$(cat <<EOF
${aggregate_marker}
Aggregated Dependabot updates for \`$VERSION\`.
EOF
)
    pr_number=$(api "repos/$repo/pulls" --method POST \
      --field "title=chore(release): aggregate Dependabot updates for $VERSION" \
      --field "head=$AGGREGATE_BRANCH" \
      --field "base=$RELEASE_BRANCH" \
      --field "body=$body" | jq -er '.number') || error "failed to create aggregate PR"
  fi

  write_output version "$VERSION"
  write_output aggregate_branch dependabot-updates
  write_output created true
  write_output pr_number "$pr_number"
}

tag_after_merge() {
  version_context "${1:-}"
  local pr_number=${2:-${RELEASE_PR_NUMBER:-}}
  [[ "$pr_number" =~ ^[0-9]+$ ]] || error "release PR number is required"
  local issue=${RELEASE_ISSUE_NUMBER:-}
  [[ "$issue" =~ ^[0-9]+$ ]] || error "RELEASE_ISSUE_NUMBER is required"

  local pr_json merged merged_at head_ref base_ref merge_sha pr_body
  pr_json=$(api "repos/$repo/pulls/$pr_number")
  merged=$(jq -r '.merged // false' <<<"$pr_json")
  merged_at=$(jq -r '.merged_at // empty' <<<"$pr_json")
  head_ref=$(jq -r '.head.ref // empty' <<<"$pr_json")
  base_ref=$(jq -r '.base.ref // empty' <<<"$pr_json")
  merge_sha=$(jq -r '.merge_commit_sha // empty' <<<"$pr_json")
  pr_body=$(jq -r '.body // ""' <<<"$pr_json")
  [[ "$merged" == true && -n "$merged_at" ]] || error "release PR is not merged"
  [[ "$head_ref" == "$RELEASE_BRANCH" && "$base_ref" == master ]] || error "release PR must be release/* -> master"
  [[ "$(jq -r '.user.login // empty' <<<"$pr_json")" == "${RELEASE_APP_SLUG:-}[bot]" ]] || \
    error "release PR was not created by the configured App"
  [[ "$pr_body" == *"<!-- jimu-release-automation:$VERSION final -->"* ]] || \
    error "release PR is not managed by this automation"
  local issue_json issue_title eligible comments
  issue_json=$(api "repos/$repo/issues/$issue")
  issue_title=$(jq -r '.title // empty' <<<"$issue_json")
  [[ "$issue_title" == "release: $VERSION" ]] || error "release Issue does not match tag version"
  eligible=$(jq -r '[.labels[]?.name] | any(. == "release: candidate" or . == "release: blocked")' <<<"$issue_json")
  [[ "$eligible" == true ]] || error "release issue is not in candidate or retryable blocked state"
  comments=$(issue_comments "$issue")
  grep -Fq "<!-- jimu-release-automation:$VERSION -->" <<<"$comments" || \
    error "release Issue is missing the automation marker"

  local master_sha
  master_sha=$(ref_sha heads master) || error "master branch ref is unavailable"
  [[ "$merge_sha" == "$master_sha" ]] || error "merge commit is not current master tip"

  local existing_tag_json existing_tag_sha existing_tag_type existing_commit tag_json tag_sha
  if existing_tag_json=$(ref_json tags "$VERSION" 2>/dev/null); then
    existing_tag_sha=$(jq -er '.object.sha' <<<"$existing_tag_json") || error "existing release tag has no object SHA"
    existing_tag_type=$(jq -r '.object.type // empty' <<<"$existing_tag_json")
    if [[ "$existing_tag_type" == tag ]]; then
      existing_tag_json=$(api "repos/$repo/git/tags/$existing_tag_sha")
      existing_commit=$(jq -er '.object.sha' <<<"$existing_tag_json") || error "existing release tag object is invalid"
    else
      existing_commit=$existing_tag_sha
    fi
    [[ "$existing_commit" == "$merge_sha" ]] || error "release tag already points to a different commit"
    write_output version "$VERSION"
    write_output tag "$VERSION"
    write_output tag_sha "$existing_tag_sha"
    write_output merge_sha "$merge_sha"
    return 0
  fi

  tag_json=$(api "repos/$repo/git/tags" --method POST \
    --field "tag=$VERSION" \
    --field "message=Release $VERSION" \
    --field "object=$merge_sha" \
    --field "type=commit")
  tag_sha=$(jq -er '.sha' <<<"$tag_json") || error "failed to create annotated tag object"
  api "repos/$repo/git/refs" --method POST \
    --field "ref=refs/tags/$VERSION" \
    --field "sha=$tag_sha" >/dev/null || error "failed to create release tag ref"

  write_output version "$VERSION"
  write_output tag "$VERSION"
  write_output tag_sha "$tag_sha"
  write_output merge_sha "$merge_sha"
}

command=${1:-}
shift || true
case "$command" in
  validate-version)
    version_context "${1:-}"
    write_output version "$VERSION"
    write_output release_branch "$RELEASE_BRANCH"
    write_output aggregate_branch dependabot-updates
    ;;
  ensure-cycle)
    ensure_cycle "${1:-}"
    ;;
  collection-ready)
    collection_ready "${1:-}"
    ;;
  create-snapshot)
    create_snapshot "${1:-}"
    ;;
  tag-after-merge)
    tag_after_merge "${1:-}" "${1:+${2:-}}"
    ;;
  *)
    usage
    ;;
esac
