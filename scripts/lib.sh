# SPDX-License-Identifier: Apache-2.0
# Shared helpers for scripts/*.sh (sourced, not executed).
# shellcheck shell=bash

ROVO_ROOT="${ROVO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ROVO_POSTGRES_IMAGE="${ROVO_POSTGRES_IMAGE:-postgis/postgis:17-3.5}"
ROVO_PG_USER="${ROVO_PG_USER:-rovo}"
ROVO_PG_DB="${ROVO_PG_DB:-rovo}"

if [[ -t 2 ]]; then
  _c_red=$'\033[31m' _c_grn=$'\033[32m' _c_yel=$'\033[33m' _c_off=$'\033[0m'
else
  _c_red="" _c_grn="" _c_yel="" _c_off=""
fi

log() { printf '%s [%s] %s\n' "$(date -u +%H:%M:%S)" "${SCRIPT_NAME:-rovo}" "$*" >&2; }
ok() { printf '%s%s%s\n' "$_c_grn" "ok   $*" "$_c_off" >&2; }
warn() { printf '%s%s%s\n' "$_c_yel" "warn $*" "$_c_off" >&2; }
fail() { printf '%s%s%s\n' "$_c_red" "FAIL $*" "$_c_off" >&2; }
die() {
  fail "$*"
  exit 1
}

need() {
  local c
  for c in "$@"; do
    command -v "$c" >/dev/null 2>&1 || die "required command not found: $c"
  done
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

file_size() { wc -c <"$1" | tr -d ' '; }

# Default to the right PostGIS image for this machine (official image is amd64-only).
default_postgres_image() {
  case "$(uname -m)" in
    arm64 | aarch64) echo "${ROVO_POSTGRES_IMAGE_ARM64:-imresamu/postgis:17-3.5}" ;;
    *) echo "$ROVO_POSTGRES_IMAGE" ;;
  esac
}

# Resolve the container id of the compose `postgis` service.
#   compose_pg_container dev|prod
compose_pg_container() {
  local stack="$1" file
  case "$stack" in
    dev) file="$ROVO_ROOT/deploy/compose/compose.yaml" ;;
    prod) file="$ROVO_ROOT/deploy/compose/compose.prod.yaml" ;;
    *) die "unknown stack: $stack (dev|prod)" ;;
  esac
  docker compose -f "$file" ps -q postgis 2>/dev/null | head -n1
}

# pg_tool <tool> [args...]
# Runs a PostgreSQL client tool (psql, pg_dump, pg_restore, pg_dumpall,
# pg_isready) against the configured database, adding --dbname. stdin is passed
# through. Source selection (first match wins):
#   PG_CONTAINER=<id|name>  docker exec into a running postgres container
#   PG_URL=<postgres://...> local client if it is >= v17, else a throwaway
#                           client container on the host network
pg_tool() {
  local tool="$1"
  shift
  if [[ -n "${PG_CONTAINER:-}" ]]; then
    _pg_run "$tool" --dbname="dbname=$ROVO_PG_DB" "$@"
  elif [[ -n "${PG_URL:-}" ]]; then
    _pg_run "$tool" --dbname="$PG_URL" "$@"
  else
    die "no database source: set PG_CONTAINER or PG_URL"
  fi
}

# pg_tool_nodb <tool> [args...]: same runner without --dbname (e.g. pg_restore --list).
pg_tool_nodb() { _pg_run "$@"; }

_pg_run() {
  local tool="$1"
  shift
  if [[ -n "${PG_CONTAINER:-}" ]]; then
    docker exec -i -e PGUSER="$ROVO_PG_USER" "$PG_CONTAINER" "$tool" "$@"
    return
  fi
  local ver=""
  if command -v "$tool" >/dev/null 2>&1; then
    ver="$("$tool" --version | sed -E 's/[^0-9]*([0-9]+).*/\1/')"
  fi
  if [[ -n "$ver" && "$ver" -ge 17 ]]; then
    "$tool" "$@"
  else
    docker run --rm -i --network host "$(default_postgres_image)" "$tool" "$@"
  fi
}

# psql_q <sql>  -> unaligned, tuples-only output; fails on error.
psql_q() {
  pg_tool psql -X -v ON_ERROR_STOP=1 -At -c "$1"
}
