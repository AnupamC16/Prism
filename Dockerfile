# Stage 1 — Builder
FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum first — separate layer for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy entire source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-w -s -X main.version=$(cat VERSION 2>/dev/null || echo dev)" \
  -o /build/prism \
  ./cmd/server

# Stage 2 — Runtime
FROM alpine:3.19

ARG SHAKA_PACKAGER_VERSION=3.4.2

RUN apk add --no-cache ca-certificates tzdata wget ffmpeg \
  && adduser -D -u 10001 prism \
  && mkdir -p /data/assets \
  && chown -R prism:prism /data

RUN set -eux; \
  arch="$(uname -m)"; \
  case "$arch" in \
    x86_64) shaka_arch="x64" ;; \
    aarch64) shaka_arch="arm64" ;; \
    *) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
  esac; \
  wget -O /usr/local/bin/packager "https://github.com/shaka-project/shaka-packager/releases/download/v${SHAKA_PACKAGER_VERSION}/packager-linux-${shaka_arch}"; \
  chmod +x /usr/local/bin/packager; \
  /usr/local/bin/packager --version

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/prism /app/prism
COPY player/demo /app/player/demo

# Copy CA certs explicitly for HTTPS to DRM servers
RUN update-ca-certificates

RUN chown prism:prism /app/prism

USER prism

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
  CMD wget --spider -q http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/prism"]
