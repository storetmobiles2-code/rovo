# syntax=docker/dockerfile:1.7
# SPDX-License-Identifier: Apache-2.0
#
# rovo backend image: one static binary (`rovo api|worker|all|migrate|seed|version`)
# with embedded migrations, on distroless/static (non-root UID 65532).
#
# Build context: repository root.
#   docker buildx build -f deploy/docker/backend.Dockerfile -t rovo:dev .
# Test image (adds the /_test/* hooks; NEVER deploy it):
#   docker buildx build -f deploy/docker/backend.Dockerfile --build-arg BUILD_TAGS=testhooks -t rovo:test .
#
# Multi-arch: the builder runs on the build platform and cross-compiles
# (CGO_ENABLED=0), so `--platform linux/amd64,linux/arm64` needs no QEMU for Go.

# Keep in step with the `go` line of backend/go.mod (CI passes it explicitly).
ARG GO_VERSION=1.25

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
# If backend/go.mod asks for a newer toolchain than the image, fetch it.
ENV GOTOOLCHAIN=auto \
    CGO_ENABLED=0 \
    GOFLAGS=-trimpath

# Dependency layer: only go.mod/go.sum, cached across builds.
COPY backend/go.mod backend/go.sum ./backend/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go -C backend mod download

COPY backend/ ./backend/
COPY deploy/docker/healthcheck/ ./healthcheck/

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
# Empty for production images. `testhooks` only for CI/E2E test images.
ARG BUILD_TAGS=""

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -eu; \
    mkdir -p /out; \
    GOOS="$TARGETOS" GOARCH="$TARGETARCH" go -C /src/backend build \
      -tags "${BUILD_TAGS}" \
      -ldflags "-s -w -buildid= -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
      -o /out/rovo ./cmd/rovo; \
    GOOS="$TARGETOS" GOARCH="$TARGETARCH" go -C /src/healthcheck build \
      -ldflags "-s -w -buildid=" -o /out/healthcheck .

FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TAGS=""
LABEL org.opencontainers.image.title="rovo" \
      org.opencontainers.image.description="rovo food delivery platform: API, worker and migrations in one binary" \
      org.opencontainers.image.source="https://github.com/storetmobiles2-code/rovo" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      dev.rovo.build-tags="${BUILD_TAGS}"

COPY --from=build /out/rovo /usr/local/bin/rovo
COPY --from=build /out/healthcheck /usr/local/bin/healthcheck

USER 65532:65532
ENV ROVO_HTTP_ADDR=:8080 \
    ROVO_INTERNAL_ADDR=:9090
EXPOSE 8080 9090
HEALTHCHECK --interval=15s --timeout=4s --start-period=20s --retries=3 \
  CMD ["/usr/local/bin/healthcheck", "http://127.0.0.1:9090/healthz"]
ENTRYPOINT ["/usr/local/bin/rovo"]
CMD ["api"]
