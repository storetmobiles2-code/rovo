#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Smoke test for the production-like stack (make smoke / make smoke-up / CI).
# Checks the edge contract of doc 22 §3.1, not business flows (the QA engineer's
# golden-flow E2E covers those).
#
#   scripts/smoke.sh [--up] [--no-build] [--down] [base-domain]
#
#   --up        bring deploy/compose/compose.prod.yaml up first (build images,
#               wait until healthy); creates .env with dev keys if missing
#   --no-build  with --up: use the existing rovo:local / rovo-web:local images
#   --down      stop the stack when the test ends (pass or fail)
#   base-domain default: $ROVO_DOMAIN or rovo.localhost
#
# Options via env:
#   ROVO_SITE_SCHEME   http (default) | https
#   SMOKE_PORT         edge port (default: $ROVO_PROD_HTTP_PORT / _HTTPS_PORT, else 80 / 443)
#   SMOKE_RESOLVE_IP   pin all five hosts to this IP with curl --resolve, bypassing
#                      DNS, /etc/hosts and proxies. Default 127.0.0.1 for
#                      *.localhost domains; unset for real domains.
#   SMOKE_CURL_OPTS    extra curl options (e.g. -k for the internal CA)
#   SMOKE_API_PATH     same-origin API path to probe (default /api/v1/public/cities)
#   SMOKE_STRICT_API=1 require 2xx JSON from SMOKE_API_PATH (default: a non-HTML
#                      404 from the API also passes with a warning, so the test
#                      is usable before the route is registered)
#   SMOKE_STRICT_TLS=1 require HSTS even when scheme=http (default: https only)
#   SMOKE_DOWN_ARGS    extra args for `compose down` with --down (e.g. -v)
#
# Exits non-zero if any check fails. Every host must answer GET /healthz from
# the API through the edge; each app host must serve its SPA index, proxy /api
# same-origin, carry the security headers and refuse a foreign CORS origin; the
# api. host must serve only webhooks.
set -euo pipefail
SCRIPT_NAME=smoke
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

do_up=0
do_build=1
do_down=0
domain=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --up) do_up=1 ;;
    --no-build) do_build=0 ;;
    --down) do_down=1 ;;
    -h | --help)
      sed -n '3,34p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    -*) die "unknown option: $1 (see --help)" ;;
    *) domain="$1" ;;
  esac
  shift
done

env_file="$ROVO_ROOT/.env"
compose=(docker compose -f "$ROVO_ROOT/deploy/compose/compose.prod.yaml")

if [[ $do_up -eq 1 ]]; then
  need docker
  [[ -f "$env_file" ]] || "$ROVO_ROOT/scripts/dev-keys.sh" "$env_file"
fi
[[ -f "$env_file" ]] && compose+=(--env-file "$env_file")

# Values from .env fill in whatever the caller did not set explicitly.
env_default() { # env_default VAR -> value from the environment, else from .env, else empty
  local v="${!1:-}"
  if [[ -z "$v" && -f "$env_file" ]]; then
    v="$(sed -n "s/^$1=//p" "$env_file" | tail -n1)"
  fi
  printf '%s' "$v"
}

domain="${domain:-$(env_default ROVO_DOMAIN)}"
domain="${domain:-rovo.localhost}"
scheme="${ROVO_SITE_SCHEME:-http}"
if [[ "$scheme" == "https" ]]; then
  port="${SMOKE_PORT:-$(env_default ROVO_PROD_HTTPS_PORT)}"
  port="${port:-443}"
  default_port=443
else
  port="${SMOKE_PORT:-$(env_default ROVO_PROD_HTTP_PORT)}"
  port="${port:-80}"
  default_port=80
fi
api_path="${SMOKE_API_PATH:-/api/v1/public/cities}"
need curl

stack_down() {
  log "stopping the production-like stack"
  # shellcheck disable=SC2086  # SMOKE_DOWN_ARGS is intentionally word-split
  "${compose[@]}" down ${SMOKE_DOWN_ARGS:-} >&2 || warn "compose down failed"
}

if [[ $do_up -eq 1 ]]; then
  up_args=(up -d --wait)
  if [[ $do_build -eq 1 ]]; then up_args+=(--build); else up_args+=(--no-build); fi
  tls=off
  [[ "$scheme" == "https" ]] && tls="${ROVO_CADDY_TLS:-internal}"
  log "starting the production-like stack (${up_args[*]}; TLS=$tls)"
  [[ $do_down -eq 1 ]] && trap stack_down EXIT
  if ! ROVO_DOMAIN="$domain" ROVO_SITE_SCHEME="$scheme" ROVO_CADDY_TLS="$tls" \
    "${compose[@]}" "${up_args[@]}" >&2; then
    "${compose[@]}" ps >&2 || true
    "${compose[@]}" logs --no-color --tail=40 migrate api worker caddy >&2 || true
    die "stack did not become healthy"
  fi
  ok "stack is up and healthy"
elif [[ $do_down -eq 1 ]]; then
  trap stack_down EXIT
fi

read -r -a curl_opts <<<"${SMOKE_CURL_OPTS:-}"
resolve_ip="${SMOKE_RESOLVE_IP-}"
if [[ -z "${SMOKE_RESOLVE_IP+set}" && ( "$domain" == "localhost" || "$domain" == *.localhost ) ]]; then
  resolve_ip=127.0.0.1
fi
if [[ -n "$resolve_ip" ]]; then
  for h in app restaurant rider admin api; do
    curl_opts+=(--resolve "$h.$domain:$port:$resolve_ip")
  done
  curl_opts+=(--noproxy "*")
fi
hostport=""
[[ "$port" != "$default_port" ]] && hostport=":$port"

pass=0
failn=0
checkpass() {
  ok "$*"
  pass=$((pass + 1))
}
checkfail() {
  fail "$*"
  failn=$((failn + 1))
}

# req METHOD URL [extra curl args...] -> sets REPLY_STATUS, REPLY_HEADERS, REPLY_BODY
req() {
  local method="$1" url="$2"
  shift 2
  local tmp
  tmp="$(mktemp)"
  REPLY_HEADERS="$(curl -sS -o "$tmp" -D - -X "$method" \
    --max-time 15 ${curl_opts[@]+"${curl_opts[@]}"} "$@" "$url" 2>/dev/null | tr -d '\r' || true)"
  REPLY_STATUS="$(awk 'toupper($1) ~ /^HTTP/ {s=$2} END {print s}' <<<"$REPLY_HEADERS")"
  REPLY_BODY="$(cat "$tmp")"
  rm -f "$tmp"
}

hdr() { awk -v k="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" 'BEGIN{FS=": "} {h=tolower($1)} h==k {sub(/^[^:]*: /,""); printf "%s", $0}' <<<"$REPLY_HEADERS"; }

want_status() { # want_status "<label>" "<expected status>"
  if [[ "$REPLY_STATUS" == "$2" ]]; then
    checkpass "$1 ($REPLY_STATUS)"
  else
    checkfail "$1 (status '${REPLY_STATUS:-none}', want $2)"
  fi
}

want_header() { # want_header "<label>" "<header>" ["<substring that must be present>"]
  local val
  val="$(hdr "$2")"
  if [[ -z "$val" ]]; then
    checkfail "$1: $2 missing"
  elif [[ -n "${3:-}" && "$val" != *"$3"* ]]; then
    checkfail "$1: $2='$val' (want to contain '$3')"
  else
    checkpass "$1: $2 present"
  fi
}

base="$scheme://HOST.$domain$hostport"
log "smoke test against $scheme://*.$domain$hostport${resolve_ip:+ (pinned to $resolve_ip)}"

# --- /healthz through the edge on every host ---------------------------------
# Caddy proxies GET /healthz to the API's internal listener, so a 200 here
# proves edge -> API reachability per host (not just that Caddy is up).
for host in app restaurant rider admin api; do
  req GET "${base/HOST/$host}/healthz"
  ctype="$(hdr content-type)"
  if [[ "$REPLY_STATUS" == "200" && "$ctype" == application/json* ]]; then
    checkpass "$host: /healthz answered by the API ($REPLY_STATUS)"
  else
    checkfail "$host: /healthz status '${REPLY_STATUS:-none}' content-type '${ctype:-none}' (want 200 application/json)"
  fi
done

# --- per app host ------------------------------------------------------------
for spec in "app:customer" "restaurant:restaurant" "rider:rider" "admin:admin"; do
  host="${spec%%:*}"
  label="${spec##*:}"
  url="${base/HOST/$host}"

  req GET "$url/"
  if [[ "$REPLY_STATUS" == "200" && "$REPLY_BODY" == *"<"* ]]; then
    checkpass "$label: SPA index served ($REPLY_STATUS)"
  else
    checkfail "$label: SPA index not served (status '${REPLY_STATUS:-none}')"
  fi

  # SPA fallback: a deep client route returns index.html, not 404.
  req GET "$url/some/deep/client/route"
  want_status "$label: SPA fallback serves index for client routes" 200

  # Security headers on the HTML response.
  req GET "$url/"
  want_header "$label" content-security-policy "frame-ancestors 'none'"
  want_header "$label" x-content-type-options nosniff
  want_header "$label" referrer-policy
  want_header "$label" permissions-policy
  if [[ "$scheme" == "https" || "${SMOKE_STRICT_TLS:-0}" == "1" ]]; then
    want_header "$label" strict-transport-security
  fi

  # The edge must strip a client-supplied origin-verify header; the index still works.
  req GET "$url/" -H "X-Rovo-Origin-Verify: forged"
  want_status "$label: forged X-Rovo-Origin-Verify tolerated (stripped at edge)" 200
done

# --- same-origin API through each app host -----------------------------------
# The request must be answered by the API (JSON), never by the SPA fallback
# (text/html) and never by a failing proxy (5xx / no response).
for host in app restaurant rider admin; do
  url="${base/HOST/$host}"
  req GET "$url$api_path"
  ctype="$(hdr content-type)"
  case "$REPLY_STATUS" in
    200 | 204)
      want_header "$host: $api_path same-origin" content-type application/json
      ;;
    "")
      checkfail "$host: $api_path no response (edge up?)"
      ;;
    404)
      if [[ "$ctype" == text/html* ]]; then
        checkfail "$host: $api_path fell through to the SPA (not proxied to the API)"
      elif [[ "${SMOKE_STRICT_API:-0}" == "1" ]]; then
        checkfail "$host: $api_path returned 404 (SMOKE_STRICT_API=1)"
      else
        warn "$host: $api_path reached the API but the route is not registered yet (404)"
        checkpass "$host: /api proxied same-origin to the API (404 from API, not the SPA)"
      fi
      ;;
    5*)
      checkfail "$host: $api_path returned $REPLY_STATUS (API down or proxy error)"
      ;;
    *)
      if [[ "$ctype" == text/html* ]]; then
        checkfail "$host: $api_path returned $REPLY_STATUS text/html (not the API)"
      else
        checkpass "$host: /api proxied same-origin to the API ($REPLY_STATUS)"
      fi
      ;;
  esac
done

# CORS must NOT be granted to a foreign origin (no CORS anywhere, R27): neither
# on a simple request nor on a preflight, on any app host.
for host in app restaurant rider admin; do
  url="${base/HOST/$host}"
  for method in GET OPTIONS; do
    req "$method" "$url$api_path" \
      -H "Origin: https://evil.example" -H "Access-Control-Request-Method: POST" \
      -H "Access-Control-Request-Headers: content-type"
    acao="$(hdr access-control-allow-origin)"
    acac="$(hdr access-control-allow-credentials)"
    if [[ -z "$REPLY_STATUS" ]]; then
      checkfail "$host: CORS probe ($method) got no response"
    elif [[ "$acao" == "*" || "$acao" == "https://evil.example" || "$acao" == "null" || -n "$acac" ]]; then
      checkfail "$host: CORS granted to a foreign origin on $method (ACAO='$acao' ACAC='$acac')"
    else
      checkpass "$host: CORS not granted to a foreign origin on $method (ACAO='${acao:-none}')"
    fi
  done
done

# --- api. host: webhooks only (R55) ------------------------------------------
apihost="$scheme://api.$domain$hostport"
req GET "$apihost/"
want_status "api. host: non-webhook path refused" 404
req GET "$apihost$api_path"
want_status "api. host: browser API path refused" 404

echo
log "smoke: $pass passed, $failn failed"
[[ $failn -eq 0 ]] || die "smoke test failed"
ok "smoke test passed"
