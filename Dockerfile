# syntax=docker/dockerfile:1.7

###
# Build a static secretsync binary for the requested platform.
# Tests now run in CI (outside Docker), so this Dockerfile focuses purely
# on compiling and packaging the runtime image.
###
FROM golang:1.25-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT
ARG CGO_ENABLED=0

ARG VERSION=dev

ENV CGO_ENABLED=${CGO_ENABLED} \
    GOTOOLCHAIN=auto
WORKDIR /src

# Update CA certificates
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./

# Cache module and build downloads between runs
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    GOARM=${TARGETVARIANT#v} \
    go build -trimpath \
      -ldflags="-s -w" \
      -o /out/secretsync ./cmd/secretsync

###
# Runtime image: tiny BusyBox container that only carries the binary and certs.
###
FROM busybox:1.36.1-musl AS runtime

ARG VERSION=dev
ARG SECRETSYNC_CONFIG=/etc/secretsync/config.yaml

ENV SECRETSYNC_CONFIG=${SECRETSYNC_CONFIG} \
    SECRETSYNC_VERSION=${VERSION}

LABEL org.opencontainers.image.title="secretsync" \
      org.opencontainers.image.source="https://github.com/jbcom/secretsync" \
      org.opencontainers.image.version=${VERSION}

WORKDIR /app

RUN mkdir -p /etc/ssl/certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/secretsync /usr/local/bin/secretsync

# Copy entrypoint script for GitHub Actions
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Don't set USER here - GitHub Actions may need root access
# The action will run with the default user

ENTRYPOINT ["/entrypoint.sh"]
