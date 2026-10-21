#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Run the Go integration tests against a throwaway PostGIS, exporting
# ROVO_TEST_DATABASE_URL (doc 20). Uses the compose `postgis` service when it is
# up, otherwise starts a dedicated throwaway container and tears it down after.
#
#   scripts/test-integration.sh [extra `go test` args...]
set -euo pipefail
SCRIPT_NAME=test-integration
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

[[ -f "$ROVO_ROOT/backend/go.mod" ]] || {
  warn "backend/ not present; nothing to test"
  exit 0
}
need docker

owned=0
cid="$(compose_pg_container dev || true)"
if [[ -n "$cid" ]]; then
  # Reuse the running dev database on a dedicated test DB.
  url="postgres://rovo:rovo@localhost:${ROVO_DEV_PG_PORT:-5432}/rovo_integration?sslmode=disable"
  docker exec -e PGUSER=rovo "$cid" psql -d rovo -v ON_ERROR_STOP=1 -tAc \
    "SELECT 1 FROM pg_database WHERE datname='rovo_integration'" | grep -q 1 \
    || docker exec -e PGUSER=rovo "$cid" createdb rovo_integration
else
  name="rovo-itest-$$"
  image="$(default_postgres_image)"
  log "starting throwaway $image"
  docker run -d --rm --name "$name" \
    -e POSTGRES_USER=rovo -e POSTGRES_PASSWORD=rovo -e POSTGRES_DB=rovo \
    -p 0:5432 "$image" >/dev/null
  owned=1
  trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
  port="$(docker port "$name" 5432/tcp | head -n1 | sed 's/.*://')"
  for _ in $(seq 1 60); do
    docker exec "$name" pg_isready -q -h 127.0.0.1 -U rovo 2>/dev/null && break
    sleep 1
  done
  url="postgres://rovo:rovo@localhost:$port/rovo?sslmode=disable"
fi

log "ROVO_TEST_DATABASE_URL -> ${url%%\?*}"
export ROVO_TEST_DATABASE_URL="$url"
args=("$@")
[[ ${#args[@]} -gt 0 ]] || args=(./...)
go -C "$ROVO_ROOT/backend" test -race -tags=integration "${args[@]}"
ok "integration tests passed"
[[ $owned -eq 1 ]] || log "left the dev database's rovo_integration DB in place (make reset to clear)"
