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

RUN apk add --no-cache ca-certificates tzdata wget \
  && adduser -D -u 10001 prism

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/prism /app/prism

# Copy CA certs explicitly for HTTPS to DRM servers
RUN update-ca-certificates

RUN chown prism:prism /app/prism

USER prism

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
  CMD wget --spider -q http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/prism"]
