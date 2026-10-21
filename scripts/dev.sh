#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Simple foreground process runner for `make dev`: brings up the Compose
# dependencies, runs migrate + seed, then runs the API, the worker and the four
# Vite dev servers together. Ctrl-C stops everything it started.
#
# Env: ROVO_SKIP_WEB=1 to run backend only; ROVO_SKIP_SEED=1 to skip seeding.
set -euo pipefail
SCRIPT_NAME=dev
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

cd "$ROVO_ROOT"
[[ -f .env ]] || { log ".env missing; generating dev keys"; ./scripts/dev-keys.sh .env; }
set -a
# shellcheck disable=SC1091  # .env is generated at runtime
source .env
set +a

pids=()
names=()
start() {
  local name="$1"; shift
  log "starting $name: $*"
  ( exec "$@" ) &
  pids+=($!)
  names+=("$name")
}
shutdown() {
  log "stopping ${#pids[@]} process(es)"
  local i
  for i in "${!pids[@]}"; do kill "${pids[$i]}" 2>/dev/null || true; done
  wait 2>/dev/null || true
}
trap shutdown INT TERM EXIT

make -s up
make -s migrate
[[ "${ROVO_SKIP_SEED:-0}" == "1" ]] || make -s seed || warn "seed failed (continuing)"

run_go=(go -C backend run ./cmd/rovo)
start api "${run_go[@]}" api
# Worker exposes its internal port on :9092 so Prometheus can scrape both.
( ROVO_INTERNAL_ADDR=":9092" "${run_go[@]}" worker ) &
pids+=($!); names+=(worker)
log "started worker (internal :9092)"

if [[ "${ROVO_SKIP_WEB:-0}" != "1" && -f web/package.json ]]; then
  for app in customer restaurant rider admin; do
    start "web:$app" pnpm --dir web --filter "./apps/$app" dev
  done
fi

cat >&2 <<EOF

rovo dev stack is up. URLs:
  customer   http://localhost:5173      restaurant http://localhost:5174
  rider      http://localhost:5175      admin      http://localhost:5176
  API        http://localhost:8080      Mailpit    http://localhost:8025
  MinIO console http://localhost:9001   (make obs-up adds Grafana http://localhost:3000)

Press Ctrl-C to stop.
EOF

wait -n 2>/dev/null || true
warn "a process exited; shutting the rest down"
