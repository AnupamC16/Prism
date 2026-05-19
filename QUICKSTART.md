# Prism — Quick Start

Everything runs locally with Docker. No paid DRM credentials needed.

## 1. Start the stack

```bash
git clone https://github.com/anupam-chopra/prism
cd prism
docker compose up --build
```

Wait until you see the health check pass:

```
prism-prism-1  | {"level":"INFO","msg":"server listening","addr":":8080"}
```

Verify:

```bash
curl http://localhost:8080/health
```

## 2. Open the demo UI

```
http://localhost:8080/demo
```

## 3. Upload a video with ClearKey encryption

In the demo UI:

1. **DRM Type** → `ClearKey`
2. Click **Choose File** and pick any `.mp4`
3. Click **Upload**
4. Wait ~1–2 minutes for FFmpeg transcoding and Shaka Packager encryption

You will see in the event log:

```
UPLOAD_START: your-video.mp4
UPLOAD_PROCESSING: FFmpeg transcoding + Shaka packaging…
UPLOAD_DONE: asset_id=test-asset-001 manifest=/manifest/dash/test-asset-001?drm=clearkey
```

## 4. Issue a DRM token

The player issues a token automatically after upload. You can also issue one manually:

```bash
curl -s -X POST http://localhost:8080/token \
  -H 'Content-Type: application/json' \
  -d '{"asset_id":"test-asset-001","viewer_id":"viewer-123","ttl":3600}' | jq .
```

## 5. Load the manifest and play

After upload completes the player loads automatically. If not:

1. **DRM Type** → `ClearKey`
2. **Stream Type** → `DASH`
3. Click **Load Manifest**
4. Press **▶** on the video

You will see the full EME flow in the event log:

```
MANIFEST_LOADED
LICENSE_OK: License acquired — keys usable
PLAYBACK_STARTED: Video is playing!
```

## 6. Inspect the manifest 

Click **Inspect Manifest** to see the real DRM signaling Shaka Packager wrote into the MPD:

- `<ContentProtection>` — proves segments are AES-128 encrypted
- `<cenc:pssh>` — Protection System Specific Header

## 7. Manual EME demo

Click **Manual EME Demo** to step through the W3C Encrypted Media Extensions API explicitly:

```
REQUEST_MEDIA_KEY_SYSTEM_ACCESS: org.w3.clearkey
CREATE_MEDIA_KEYS
SESSION_CREATED
GENERATE_REQUEST
ENCRYPTED_MESSAGE: Got keymessage
LICENSE_FETCHED: Received license from /license/clearkey
LICENSE_UPDATED: Session updated — keys are usable
KEY_STATUSES_CHANGE: usable
```

This is the same EME flow used by Widevine and FairPlay — only the key system string and license URL differ.

## DRM combinations

| DRM | Stream | Works locally |
|-----|--------|--------------|
| None | HLS | Yes |
| None | DASH | Yes |
| ClearKey | DASH | Yes — full offline |
| Widevine | DASH | Needs Google credentials in `.env` |
| PlayReady | DASH | Needs Microsoft credentials in `.env` |
| FairPlay | HLS | Needs Apple credentials + Safari |

## Stopping the stack

```bash
docker compose down
```

Processed assets are persisted in `data/assets/` and survive restarts.
