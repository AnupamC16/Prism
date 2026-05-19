#!/usr/bin/env bash
# Smoke test: upload a video with each DRM type and verify manifest output.
# Usage: ./scripts/smoke_test.sh /path/to/video.mp4

set -euo pipefail

BASE_URL="${PRISM_URL:-http://localhost:8080}"

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 /path/to/video.mp4" >&2
  exit 1
fi

INPUT="$1"
if [ ! -f "$INPUT" ]; then
  echo "Video file not found: $INPUT" >&2
  exit 1
fi

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

upload_and_check() {
  local asset_id="$1"
  local drm="$2"
  local manifest_url="$3"
  local expected_string="$4"

  echo "==> Uploading $asset_id with DRM=$drm"
  "$SCRIPT_DIR/upload_video.sh" "$INPUT" "$asset_id" "$drm" >/dev/null

  echo "==> Fetching manifest: $manifest_url"
  local manifest
  manifest=$(curl -sf "$BASE_URL$manifest_url")

  if echo "$manifest" | grep -q "$expected_string"; then
    echo "    PASS: found '$expected_string' in manifest"
  else
    echo "    FAIL: '$expected_string' not found in manifest"
    echo "$manifest"
    exit 1
  fi
}

upload_and_check "smoke-ck" "clearkey" "/manifest/dash/smoke-ck?drm=clearkey" "ContentProtection"
upload_and_check "smoke-wv" "widevine" "/manifest/dash/smoke-wv?drm=widevine" "cenc:pssh"
upload_and_check "smoke-fp" "fairplay" "/manifest/hls/smoke-fp?drm=fairplay" "EXT-X-KEY"

echo ""
echo "All smoke tests passed."
