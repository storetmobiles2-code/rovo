# syntax=docker/dockerfile:1.7
# SPDX-License-Identifier: Apache-2.0
#
# rovo edge image for the production-like Compose stack (ADR-P2-003):
# builds the four SPAs (customer, restaurant, rider, admin) and serves them
# with Caddy, which also routes /api/* same-origin to the `api` service.
#
# Build context: repository root.
#   docker buildx build -f deploy/docker/web.Dockerfile -t rovo-web:dev .
#
# On AWS the same `dist/` folders go to S3 + CloudFront instead (doc 22 §8);
# this image is the local/demo equivalent of that edge.

ARG NODE_VERSION=22
ARG CADDY_VERSION=2.11

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-bookworm-slim AS build
ENV PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH \
    CI=true \
    COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable
WORKDIR /src

# The generated TS client lives in web/packages/api-client and is committed,
# but the spec is copied too so a codegen step can run if the package wants it.
COPY openapi/ ./openapi/
COPY web/ ./web/

WORKDIR /src/web
RUN --mount=type=cache,id=pnpm-store,target=/pnpm/store \
    set -eu; \
    pnpm config set store-dir /pnpm/store; \
    if [ -f pnpm-lock.yaml ]; then pnpm install --frozen-lockfile; else pnpm install; fi

ARG VITE_API_BASE=/api/v1
ARG VERSION=dev
ENV VITE_API_BASE=${VITE_API_BASE} \
    VITE_APP_VERSION=${VERSION}
RUN set -eu; \
    for app in customer restaurant rider admin; do \
      echo "==> building ${app}"; \
      pnpm --filter "./apps/${app}" build; \
      test -f "apps/${app}/dist/index.html"; \
    done

FROM caddy:${CADDY_VERSION}
ARG VERSION=dev
ARG COMMIT=unknown
LABEL org.opencontainers.image.title="rovo-web" \
      org.opencontainers.image.description="rovo SPAs (customer, restaurant, rider, admin) behind Caddy" \
      org.opencontainers.image.source="https://github.com/storetmobiles2-code/rovo" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}"

COPY deploy/caddy/ /etc/caddy/
COPY --from=build /src/web/apps/customer/dist/   /srv/customer/
COPY --from=build /src/web/apps/restaurant/dist/ /srv/restaurant/
COPY --from=build /src/web/apps/rider/dist/      /srv/rider/
COPY --from=build /src/web/apps/admin/dist/      /srv/admin/

RUN caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile \
      || (echo "Caddyfile failed validation" >&2; exit 1)

EXPOSE 80 443
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD ["wget", "-q", "-O", "/dev/null", "http://127.0.0.1:8081/healthz"]
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
