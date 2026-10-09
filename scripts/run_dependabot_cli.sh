#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
[[ $# == 2 ]] || { echo "usage: run_dependabot_cli.sh <prepare|scan|apply> <result-dir>" >&2; exit 2; }
mode=$1
result_dir=$2
mkdir -p "$result_dir"

for ecosystem in go github-actions docker; do
  case "$mode" in
    prepare)
      python3 "$ROOT_DIR/scripts/dependabot_cli.py" prepare \
        "$ROOT_DIR/.github/dependabot-cli/$ecosystem.yml" "$result_dir/$ecosystem.input.json"
      ;;
    scan)
      "${DEPENDABOT_CLI_BIN:-dependabot}" update \
        -f "$result_dir/$ecosystem.input.json" --timeout 20m > "$result_dir/$ecosystem.jsonl"
      ;;
    apply)
      base_sha=$(jq -er '.job.source.commit' "$result_dir/$ecosystem.input.json")
      python3 "$ROOT_DIR/scripts/dependabot_cli.py" apply "$ecosystem" "$result_dir/$ecosystem.jsonl" "$base_sha"
      ;;
    *) echo "unsupported mode: $mode" >&2; exit 2 ;;
  esac
done
