#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Post-deploy smoke test for the production-like stack (make smoke / CI e2e).
# Checks the edge contract of doc 22 §3.1, not business flows (the QA engineer's
# golden-flow smoke comes later).
#
#   scripts/smoke.sh [base-domain]        default: $ROVO_DOMAIN or rovo.localhost
#
# Options via env:
#   ROVO_SITE_SCHEME   http (default) | https
#   SMOKE_CURL_OPTS    extra curl options (e.g. -k for the internal CA, or
#                      --resolve app.rovo.localhost:80:127.0.0.1 when there is
#                      no DNS/hosts entry)
#   SMOKE_STRICT_TLS=1 require HSTS even when scheme=http (default: https only)
#
# Exits non-zero if any check fails. Each app host must serve its SPA index,
# proxy /api same-origin, carry the security headers, and refuse a foreign CORS
# origin; the api. host must serve only webhooks.
set -euo pipefail
SCRIPT_NAME=smoke
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

domain="${1:-${ROVO_DOMAIN:-rovo.localhost}}"
scheme="${ROVO_SITE_SCHEME:-http}"
read -r -a curl_opts <<<"${SMOKE_CURL_OPTS:-}"
need curl

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
    --max-time 15 "${curl_opts[@]}" "$@" "$url" 2>/dev/null || true)"
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

base="$scheme://HOST.$domain"
log "smoke test against *.$domain ($scheme)"

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

# --- same-origin API + health through each app host --------------------------
for host in app restaurant rider admin; do
  url="${base/HOST/$host}"
  req GET "$url/api/v1/public/cities"
  case "$REPLY_STATUS" in
    200 | 204) want_header "$host: /api/v1/public/cities same-origin" content-type application/json ;;
    "") checkfail "$host: /api/v1/public/cities no response (api up?)" ;;
    *) checkfail "$host: /api/v1/public/cities returned $REPLY_STATUS" ;;
  esac
done

# CORS must NOT be granted to a foreign origin (no CORS anywhere, R27).
req GET "${base/HOST/app}/api/v1/public/cities" \
  -H "Origin: https://evil.example" -H "Access-Control-Request-Method: GET"
acao="$(hdr access-control-allow-origin)"
if [[ -z "$acao" || ( "$acao" != "*" && "$acao" != "https://evil.example" ) ]]; then
  checkpass "CORS not granted to a foreign origin (ACAO='${acao:-none}')"
else
  checkfail "CORS granted to a foreign origin (ACAO='$acao')"
fi

# --- api. host: webhooks only (R55) ------------------------------------------
apihost="$scheme://api.$domain"
req GET "$apihost/"
want_status "api. host: non-webhook path refused" 404
req GET "$apihost/api/v1/public/cities"
want_status "api. host: browser API path refused" 404

echo
log "smoke: $pass passed, $failn failed"
[[ $failn -eq 0 ]] || die "smoke test failed"
ok "smoke test passed"
