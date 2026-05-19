#!/usr/bin/env sh
set -eu

API_BASE="${PRISM_URL:-http://localhost:8080}"

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 /path/to/video.mp4 [asset-id] [drm]" >&2
  echo "       drm: clearkey | widevine | fairplay" >&2
  echo "Example:                  $0 ~/Movies/sample.mp4 sample-video" >&2
  echo "Example with ClearKey:    $0 ~/Movies/sample.mp4 sample-video clearkey" >&2
  echo "Example with Widevine:    $0 ~/Movies/sample.mp4 sample-video widevine" >&2
  echo "Example with FairPlay:    $0 ~/Movies/sample.mp4 sample-video fairplay" >&2
  exit 1
fi

VIDEO_PATH="$1"
ASSET_ID="${2:-}"
DRM="${3:-}"

case "$DRM" in
  ""|clearkey|widevine|fairplay) ;;
  *)
    echo "Invalid drm value: $DRM (must be clearkey, widevine, or fairplay)" >&2
    exit 1
    ;;
esac

if [ ! -f "$VIDEO_PATH" ]; then
  echo "Video file not found: $VIDEO_PATH" >&2
  exit 1
fi

set -- -fsS -X POST "$API_BASE/upload" -F "file=@$VIDEO_PATH"
if [ -n "$ASSET_ID" ]; then
  set -- "$@" -F "asset_id=$ASSET_ID"
fi
if [ -n "$DRM" ]; then
  set -- "$@" -F "drm=$DRM"
fi

curl "$@"

echo
echo "Open $API_BASE/demo"
