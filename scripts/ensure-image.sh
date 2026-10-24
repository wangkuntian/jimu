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
fingerprint="$("$ROOT_DIR/scripts/image-fingerprint.sh" "$PROFILE")"

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
