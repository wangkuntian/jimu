#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROFILE="${PROFILE:-full}"
VERSION="${VERSION:-dev}"
IMAGE_REPO="${IMAGE_REPO:-jimu}"
IMAGE_STAMP_DIR="${IMAGE_STAMP_DIR:-$ROOT_DIR/.overlay/images}"
IMAGE_TAG="$IMAGE_REPO:$VERSION"
LATEST_TAG="$IMAGE_REPO:latest"
STAMP="$IMAGE_STAMP_DIR/$VERSION-$PROFILE.sha256"

if command -v sha256sum >/dev/null 2>&1; then
  sha_cmd=(sha256sum)
else
  sha_cmd=(shasum -a 256)
fi

fingerprint="$(
  cd "$ROOT_DIR"
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
)"

if [ "${FORCE:-0}" != "1" ] &&
  docker image inspect "$IMAGE_TAG" >/dev/null 2>&1 &&
  [ -f "$STAMP" ] &&
  [ "$(cat "$STAMP")" = "$fingerprint" ]; then
  echo "image $IMAGE_TAG is up to date"
  exit 0
fi

docker build --build-arg PROFILE="$PROFILE" -t "$IMAGE_TAG" "$ROOT_DIR"
docker tag "$IMAGE_TAG" "$LATEST_TAG"
mkdir -p "$IMAGE_STAMP_DIR"
printf '%s\n' "$fingerprint" >"$STAMP"
