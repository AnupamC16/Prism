# Prism — OTT Manifest Generator & DRM Licensing Service

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![CI](https://github.com/anupam-chopra/prism/actions/workflows/ci.yml/badge.svg)](https://github.com/anupam-chopra/prism/actions/workflows/ci.yml)

Production-ready Go service for HLS and MPEG-DASH manifest generation with integrated Widevine, FairPlay and PlayReady DRM licensing. Deployed via Docker Compose and CloudFront-backed staging.

## Performance

- 600 concurrent sessions with p95 latency under 90ms
- DRM token validation under 18ms p95
- Redis caching cuts origin compute by 41% under peak ingest

## Architecture

```text
Player (dash.js/Shaka/native)
   │
   │ HTTPS via CloudFront
   ▼
┌────────────────────────┐
│  CloudFront (CDN)       │  /manifest/* cached 30s
│                         │  /license/*  no-cache
└──────────┬──────────────┘
           ▼
┌────────────────────────┐    ┌──────────┐
│  Prism (Go)             │◀──▶│  Redis   │
│  :8080                  │    └──────────┘
└──────┬─────────┬────────┘
       │         │
       ▼         ▼
Widevine    FairPlay+PlayReady
License     License Servers
Server
```

## Features

- HLS master playlist generation with codec, bandwidth and resolution filters.
- MPEG-DASH MPD generation with Widevine, FairPlay and PlayReady content protection injection.
- Local ClearKey encrypted DASH packaging with per-asset content keys for real EME playback demos.
- DRM license proxy endpoints for Widevine, FairPlay and PlayReady.
- Signed DRM tokens with JWT, Redis-backed revocation and asset-scoped validation.
- Redis manifest, token, certificate and license caching.
- CloudFront signed URL support and manifest URI rewriting.
- Request ID propagation, structured JSON logging and DRM audit logs.
- Production HTTP middleware for recovery, CORS, request logging and timeouts.
- Docker Compose local stack with Redis and optional Prometheus.
- CloudFront Terraform and AWS distribution JSON examples.
- k6 load tests for 600 concurrent sessions and DRM token validation latency.
- k6 cache benchmark demonstrating origin compute reduction from Redis caching.
- Browser demo using dash.js plus explicit EME/MSE calls for license exchange visibility.
- Unit tests with miniredis, mocks and service/controller coverage.
- GitHub Actions CI for vet, build, race tests, benchmarks, Docker smoke tests and load testing.

## Quick Start

See [QUICKSTART.md](QUICKSTART.md) for the full step-by-step guide after cloning.

```bash
git clone https://github.com/anupam-chopra/prism
cd prism
docker compose up --build
```

Open `http://localhost:8080/demo`, upload a video with **DRM Type = ClearKey**, and the full EME pipeline runs locally with no paid credentials.

Upload and process a local video via CLI:

```bash
./scripts/upload_video.sh /path/to/video.mp4 my-video
open 'http://localhost:8080/demo?asset_id=my-video'
```

Upload with ClearKey encryption:

```bash
./scripts/upload_video.sh /path/to/video.mp4 my-video clearkey
open 'http://localhost:8080/demo?asset_id=my-video'
```

## API Reference

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe — verifies Redis |
| POST | `/token` | Issue signed DRM token |
| POST | `/upload` | Upload and process a local video into HLS and DASH |
| GET | `/manifest/hls/{id}` | Generate HLS playlist |
| GET | `/manifest/dash/{id}` | Generate MPD manifest |
| GET | `/assets/{id}/hls/{file}` | Serve processed HLS segments |
| GET | `/assets/{id}/dash/{file}` | Serve processed DASH segments |
| GET | `/assets/{id}/dash_clearkey/{file}` | Serve encrypted ClearKey DASH media |
| POST | `/license/widevine` | Widevine license proxy |
| POST | `/license/fairplay` | FairPlay license proxy |
| POST | `/license/playready` | PlayReady license proxy |
| POST | `/license/clearkey` | Local ClearKey license response |

### GET /health

Request headers: none required.

Request body: none.

Response headers: `Content-Type: application/json`.

Response body:

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "version": "1.0.0",
    "uptime_sec": 42
  }
}
```

Curl:

```bash
curl http://localhost:8080/health
```

### GET /ready

Request headers: none required.

Request body: none.

Response headers: `Content-Type: application/json`.

Response body:

```json
{
  "success": true,
  "data": {
    "status": "ready",
    "cache": "ok"
  }
}
```

Curl:

```bash
curl http://localhost:8080/ready
```

### POST /token

Request headers: `Content-Type: application/json`.

Request body:

```json
{
  "asset_id": "test-asset-001",
  "viewer_id": "viewer-123",
  "ttl": 3600
}
```

Response headers: `Content-Type: application/json`.

Response body:

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "asset_id": "test-asset-001",
    "expires_at": "2025-01-01T01:00:00Z",
    "ttl_seconds": 3600
  }
}
```

Curl:

```bash
curl -X POST http://localhost:8080/token \
  -H 'Content-Type: application/json' \
  -d '{"asset_id":"test-asset-001","viewer_id":"viewer-123","ttl":3600}'
```

### POST /upload

Request body: multipart form with `file`, optional `asset_id`, and optional `drm=clearkey`.

Curl:

```bash
curl -X POST http://localhost:8080/upload \
  -F 'asset_id=my-video' \
  -F 'drm=clearkey' \
  -F 'file=@/path/to/video.mp4'
```

Response body:

```json
{
  "success": true,
  "data": {
    "asset_id": "my-video",
    "hls_manifest": "/manifest/hls/my-video",
    "dash_manifest": "/manifest/dash/my-video",
    "clearkey_dash_manifest": "/manifest/dash/my-video?drm=clearkey",
    "player_url": "/demo?asset_id=my-video"
  }
}
```

### GET /manifest/hls/{id}

Request headers: none required.

Query parameters: `codec`, `maxBandwidth`, `resolution`, `drm=clearkey`.

Request body: none.

Response headers: `Content-Type: application/vnd.apple.mpegurl`, `Cache-Control: public, max-age=30`.

Response body: HLS master playlist bytes.

Curl:

```bash
curl 'http://localhost:8080/manifest/hls/test-asset-001?codec=avc1&maxBandwidth=5000000&resolution=1080p'
```

### GET /manifest/dash/{id}

Request headers: none required.

Query parameters: `codec`, `maxBandwidth`, `resolution`.

Request body: none.

Response headers: `Content-Type: application/dash+xml`, `Cache-Control: public, max-age=30`.

Response body: MPEG-DASH MPD XML.

Curl:

```bash
curl 'http://localhost:8080/manifest/dash/test-asset-001?resolution=1080p'
```

### POST /license/clearkey

Request headers: `Content-Type: application/json`, `X-DRM-Token`, `X-Asset-ID`.

Request body: ClearKey license challenge JSON from EME/dash.js.

Response body: ClearKey license JSON with the per-asset key.

Curl:

```bash
TOKEN="$(curl -s -X POST http://localhost:8080/token \
  -H 'Content-Type: application/json' \
  -d '{"asset_id":"my-video","viewer_id":"viewer-123","ttl":3600}' | jq -r '.data.token')"

curl -X POST http://localhost:8080/license/clearkey \
  -H "X-DRM-Token: ${TOKEN}" \
  -H 'X-Asset-ID: my-video' \
  -H 'Content-Type: application/json' \
  -d '{"kids":[]}'
```

### POST /license/widevine

Request headers: `Content-Type: application/octet-stream`, `X-DRM-Token`, `X-Asset-ID`.

Request body: raw Widevine challenge bytes.

Response headers: `Content-Type: application/octet-stream`, `Cache-Control: no-store`.

Response body: raw Widevine license bytes.

Curl:

```bash
TOKEN="$(curl -s -X POST http://localhost:8080/token \
  -H 'Content-Type: application/json' \
  -d '{"asset_id":"test-asset-001","viewer_id":"viewer-123","ttl":3600}' | jq -r '.data.token')"

curl -X POST http://localhost:8080/license/widevine \
  -H "X-DRM-Token: ${TOKEN}" \
  -H 'X-Asset-ID: test-asset-001' \
  -H 'Content-Type: application/octet-stream' \
  --data-binary @challenge.bin
```

### POST /license/fairplay

Request headers: `Content-Type: application/octet-stream`, `X-DRM-Token`, `X-Asset-ID`, `X-FairPlay-SPC`.

Request body: raw FairPlay SPC/challenge bytes.

Response headers: `Content-Type: application/octet-stream`, `Cache-Control: no-store`.

Response body: raw FairPlay CKC bytes.

Curl:

```bash
curl -X POST http://localhost:8080/license/fairplay \
  -H "X-DRM-Token: ${TOKEN}" \
  -H 'X-Asset-ID: test-asset-001' \
  -H "X-FairPlay-SPC: $(base64 < spc.bin)" \
  -H 'Content-Type: application/octet-stream' \
  --data-binary @spc.bin
```

### POST /license/playready

Request headers: `Content-Type: text/xml; charset=utf-8`, `X-DRM-Token`, `X-Asset-ID`.

Request body: PlayReady challenge XML.

Response headers: `Content-Type: text/xml; charset=utf-8`, `Cache-Control: no-store`.

Response body: PlayReady license XML.

Curl:

```bash
curl -X POST http://localhost:8080/license/playready \
  -H "X-DRM-Token: ${TOKEN}" \
  -H 'X-Asset-ID: test-asset-001' \
  -H 'Content-Type: text/xml; charset=utf-8' \
  --data-binary @playready-challenge.xml
```

## Configuration

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | HTTP listen address. | `:8080` |
| `LOG_LEVEL` | Structured logger level: `debug`, `info`, `warn`, `error`. | `info` |
| `REDIS_URL` | Redis connection URL. | Required |
| `JWT_SECRET` | HS256 JWT signing secret, minimum 32 characters. | Required |
| `TOKEN_TTL_SECONDS` | Default DRM token TTL in seconds. | `3600` |
| `WIDEVINE_URL` | Upstream Widevine license server URL. | Required |
| `WIDEVINE_API_KEY` | API key sent to the Widevine upstream. | Required |
| `FAIRPLAY_URL` | Upstream FairPlay CKC endpoint. | Required |
| `FAIRPLAY_CERT_URL` | Upstream FairPlay application certificate URL. | Required |
| `FAIRPLAY_SECRET` | Shared secret sent to the FairPlay upstream. | Required |
| `PLAYREADY_URL` | Upstream PlayReady license server URL. | Required |
| `CLOUDFRONT_DOMAIN` | CloudFront distribution domain for signed media URIs. Set all CloudFront variables to enable URL signing. | Optional |
| `CLOUDFRONT_KEY_PAIR_ID` | CloudFront public key ID used in signed URLs. Set all CloudFront variables to enable URL signing. | Optional |
| `CLOUDFRONT_PRIVATE_KEY` | PEM-encoded RSA private key for CloudFront URL signing. Set all CloudFront variables to enable URL signing. | Optional |
| `MANIFEST_CACHE_TTL` | Manifest cache TTL in seconds. | `30` |
| `LICENSE_CACHE_TTL` | DRM license response cache TTL in seconds. | `300` |
| `CERT_CACHE_TTL` | FairPlay certificate cache TTL in seconds. | `3600` |
| `MEDIA_ROOT` | Directory for uploaded and processed local media assets. | `data/assets` |
| `FFMPEG_PATH` | FFmpeg executable used for local video processing. | `ffmpeg` |
| `SHAKA_PACKAGER_PATH` | Shaka Packager executable used for ClearKey DASH packaging. | `packager` |
| `PLAYER_DEMO_PATH` | HTML demo file served at `/demo`. | `player/demo/index.html` |
| `VERSION` | Application version reported by health responses. | `1.0.0` |

## Project Structure

```text
cmd/server/              HTTP server entrypoint and dependency wiring
deploy/                  Docker Compose, CloudFront Terraform and Kubernetes manifests
internal/api/            Router, middleware, controllers, requests and responses
internal/cache/          Redis cache interface, keys and implementation
internal/cdn/            CloudFront signed URL support
internal/config/         Environment configuration loader
internal/drm/            DRM router and Widevine/FairPlay/PlayReady providers
internal/logger/         Structured logging and request ID helpers
internal/manifest/       HLS and DASH manifest generators, filters and DRM injection
internal/model/          Domain models and typed errors
internal/service/        Manifest, DRM and token service logic
mock/                    Test mocks for services, providers and cache
player/demo/             Browser EME/MSE demo
scripts/                 k6 load tests and cache benchmark
```

## Testing

```bash
go test ./... -v
go test -bench=BenchmarkTokenValidate -benchtime=10s ./internal/service/test/
```

## Load Testing

```bash
docker compose up -d
k6 run scripts/load_test.js
open scripts/load_test_report.html
```

## Cache Benchmark

```bash
k6 run scripts/cache_benchmark.js
```

The benchmark compares cold-cache manifest generation against warm-cache Redis hits and prints the p95 reduction plus the equivalent origin compute saved percentage.

## Browser Demo

Open [player/demo/index.html](player/demo/index.html) in a browser to exercise dash.js playback configuration and inspect explicit EME/MSE license exchange code paths.

## Testing DRM locally

End-to-end packaging with real encryption (ClearKey, Widevine, FairPlay) runs against the local Docker stack. Widevine and FairPlay license acquisition still requires real upstream DRM credentials in `.env`; ClearKey works fully offline.

### Start the stack

```bash
docker compose up --build
```

### Upload and package with each DRM type

```bash
./scripts/upload_video.sh /path/to/video.mp4 my-video-ck clearkey
./scripts/upload_video.sh /path/to/video.mp4 my-video-wv widevine
./scripts/upload_video.sh /path/to/video.mp4 my-video-fp fairplay
```

### Verify packaging output

```bash
# ClearKey — DASH MPD with CENC ContentProtection
curl "http://localhost:8080/manifest/dash/my-video-ck?drm=clearkey"

# Widevine/PlayReady — DASH MPD with real <cenc:pssh> from Shaka Packager
curl "http://localhost:8080/manifest/dash/my-video-wv?drm=widevine"

# FairPlay — HLS master with EXT-X-KEY URI=skd://fairplay/<real-kid>
curl "http://localhost:8080/manifest/hls/my-video-fp?drm=fairplay"
```

### Open the demo player

Visit `http://localhost:8080/demo`, select the asset ID, matching stream type, and DRM type, then click **Load Manifest**.

- **ClearKey** plays fully in the browser with no upstream DRM server.
- **Widevine** / **PlayReady** / **FairPlay** package correctly but license acquisition requires the corresponding upstream URLs and credentials in `.env`.

### Smoke test

```bash
./scripts/smoke_test.sh /path/to/video.mp4
```

The script uploads the file once per DRM type and asserts that each generated manifest contains the expected DRM signaling (`cenc:pssh`, `EXT-X-KEY`, ClearKey).

## Deployment

- Docker: build and run the local stack with [docker-compose.yml](docker-compose.yml).
- CloudFront: provision CDN policies, signed URL key groups and origin behavior with [deploy/cloudfront.tf](deploy/cloudfront.tf).
- AWS CLI: use [deploy/cloudfront_config.json](deploy/cloudfront_config.json) with `aws cloudfront create-distribution`.
- Kubernetes: base deployment and service manifests live under [deploy/k8s](deploy/k8s).

## Contributing

1. Create a topic branch from `main`.
2. Run `go test ./...`, `go vet ./...` and any relevant k6 scripts.
3. Keep commits focused and use conventional commit messages.
4. Open a pull request with the behavior change, test evidence and any operational notes.

## License

MIT 2025 Anupam Chopra. See [LICENSE](LICENSE).
