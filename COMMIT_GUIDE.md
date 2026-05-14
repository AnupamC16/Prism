# Commit Guide

Use this guide to split the current project into a realistic, reviewable commit history.

## Commit 1

Message: `feat: scaffold project structure and config loader`

Files:
- `go.mod`
- `go.sum`
- `internal/config/`
- `internal/logger/`
- `internal/model/errors.go`

## Commit 2

Message: `feat: add domain models for tokens, manifests, licenses`

Files:
- `internal/model/*.go`

## Commit 3

Message: `feat: implement Redis cache layer with miniredis tests`

Files:
- `internal/cache/*.go`

## Commit 4

Message: `feat: add HLS and DASH manifest generators with DRM injection`

Files:
- `internal/manifest/**`

## Commit 5

Message: `feat: implement Widevine, FairPlay, PlayReady DRM providers`

Files:
- `internal/drm/**`

## Commit 6

Message: `feat: add CloudFront URL signing and manifest URI rewriting`

Files:
- `internal/cdn/cloudfront.go`

## Commit 7

Message: `feat: implement manifest, DRM and token services with caching`

Files:
- `internal/service/**`

## Commit 8

Message: `feat: add HTTP API layer with controllers, middleware and routing`

Files:
- `internal/api/**`

## Commit 9

Message: `feat: wire dependency graph in cmd/server/main.go`

Files:
- `cmd/server/main.go`

## Commit 10

Message: `test: add controller, service and DRM unit tests with mocks`

Files:
- `mock/`
- `*_test.go` files

## Commit 11

Message: `build: add Dockerfile, docker-compose and CloudFront Terraform`

Files:
- `Dockerfile`
- `deploy/**`

## Commit 12

Message: `ci: add GitHub Actions, load tests, browser demo and README`

Files:
- `.github/`
- `scripts/`
- `player/`
- `README.md`
- `LICENSE`
- `.env.example`
- `.gitignore`
