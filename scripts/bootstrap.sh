#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Check prerequisites and prepare a developer machine for rovo (make bootstrap).
# Does not install system packages; it reports what is missing and sets up the
# things it safely can (.env with dev keys, pnpm via corepack, Go tools).
set -euo pipefail
SCRIPT_NAME=bootstrap
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

missing=0
check_version() {
  local name="$1" have="$2" want="$3"
  if [[ -z "$have" ]]; then
    fail "$name not found (need >= $want)"
    missing=$((missing + 1))
    return
  fi
  if [[ "$(printf '%s\n%s\n' "$want" "$have" | sort -V | head -n1)" == "$want" ]]; then
    ok "$name $have (>= $want)"
  else
    warn "$name $have is older than the recommended $want"
  fi
}

ver() { "$1" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -n1; }

log "checking prerequisites"
if command -v docker >/dev/null; then
  if docker info >/dev/null 2>&1; then
    ok "docker $(ver docker), daemon reachable"
  else
    fail "docker is installed but the daemon is not reachable"
    missing=$((missing + 1))
  fi
  if docker compose version >/dev/null 2>&1; then
    ok "docker compose $(docker compose version --short 2>/dev/null)"
  else
    fail "docker compose v2 plugin not found"
    missing=$((missing + 1))
  fi
else
  fail "docker not found"
  missing=$((missing + 1))
fi

check_version Go "$(go version 2>/dev/null | grep -oE 'go[0-9.]+' | tr -d 'go' | head -n1)" 1.25
check_version Node "$(ver node)" 22.12.0
if command -v corepack >/dev/null; then
  ok "corepack present"
else
  warn "corepack not found (ships with Node >= 16.9; 'npm i -g corepack' otherwise)"
fi
if command -v make >/dev/null; then ok "make present"; else fail "make not found"; missing=$((missing + 1)); fi
if command -v openssl >/dev/null; then ok "openssl present"; else fail "openssl not found (needed by make dev-keys)"; missing=$((missing + 1)); fi

if [[ $missing -gt 0 ]]; then
  die "$missing prerequisite(s) missing; install them and re-run make bootstrap"
fi

log "preparing local configuration"
"$ROVO_ROOT/scripts/dev-keys.sh" "$ROVO_ROOT/.env"

if command -v corepack >/dev/null && [[ -f "$ROVO_ROOT/web/package.json" ]]; then
  log "enabling pnpm via corepack and installing web dependencies"
  corepack enable >/dev/null 2>&1 || warn "corepack enable needs sudo; run it manually once"
  (cd "$ROVO_ROOT/web" && pnpm install --frozen-lockfile) || warn "pnpm install failed (web/ may be incomplete)"
else
  warn "skipping pnpm install (web/ not ready or corepack missing)"
fi

ok "bootstrap complete — run 'make dev' to start the full stack"
