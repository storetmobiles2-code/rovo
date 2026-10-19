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
#   SMOKE_STRICT_TLS=1 require HSTS on https (default: only when scheme=https)
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

hdr() { awk -v k="$(tr '[:upper:]' '[:lower:]' <<<"$1")" 'BEGIN{FS=": "} {h=tolower($1)} h==k {sub(/^[^:]*: /,""); printf "%s", $0}' <<<"$REPLY_HEADERS"; }

base="$scheme://HOST.$domain"
log "smoke test against *.$domain ($scheme)"

# --- per app host ------------------------------------------------------------
declare -A app_label=([app]=customer [restaurant]=restaurant [rider]=rider [admin]=admin)
for host in app restaurant rider admin; do
  label="${app_label[$host]}"
  url="${base/HOST/$host}"

  req GET "$url/"
  if [[ "$REPLY_STATUS" == "200" && "$REPLY_BODY" == *"<"* ]]; then
    checkpass "$label: SPA index served ($REPLY_STATUS)"
  else
    checkfail "$label: SPA index not served (status '$REPLY_STATUS')"
  fi

  # SPA fallback: a deep client route returns index.html, not 404.
  req GET "$url/some/deep/client/route"
  [[ "$REPLY_STATUS" == "200" ]] \
    && checkpass "$label: SPA fallback serves index for client routes" \
    || checkfail "$label: SPA fallback missing (status '$REPLY_STATUS')"

  # Security headers present on the HTML response.
  req GET "$url/"
  [[ -n "$(hdr content-security-policy)" ]] \
    && checkpass "$label: Content-Security-Policy present" \
    || checkfail "$label: Content-Security-Policy missing"
  [[ "$(hdr x-content-type-options)" == "nosniff" ]] \
    && checkpass "$label: X-Content-Type-Options nosniff" \
    || checkfail "$label: X-Content-Type-Options missing"
  [[ -n "$(hdr referrer-policy)" ]] \
    && checkpass "$label: Referrer-Policy present" \
    || checkfail "$label: Referrer-Policy missing"
  [[ -n "$(hdr permissions-policy)" ]] \
    && checkpass "$label: Permissions-Policy present" \
    || checkfail "$label: Permissions-Policy missing"
  csp="$(hdr content-security-policy)"
  [[ "$csp" == *"frame-ancestors 'none'"* ]] \
    && checkpass "$label: CSP frame-ancestors none" \
    || checkfail "$label: CSP frame-ancestors none missing"
  if [[ "$scheme" == "https" || "${SMOKE_STRICT_TLS:-0}" == "1" ]]; then
    [[ -n "$(hdr strict-transport-security)" ]] \
      && checkpass "$label: HSTS present" \
      || checkfail "$label: HSTS missing on https"
  fi

  # The edge must strip a client-supplied origin-verify header before the API.
  # We cannot read it at the API from here, but the index must still work with
  # the header present (edge strips it; no 400).
  req GET "$url/" -H "X-Rovo-Origin-Verify: forged"
  [[ "$REPLY_STATUS" == "200" ]] \
    && checkpass "$label: forged X-Rovo-Origin-Verify tolerated (stripped at edge)" \
    || checkfail "$label: request with forged origin-verify header failed ($REPLY_STATUS)"
done

# --- same-origin API + health through each app host --------------------------
for host in app restaurant rider admin; do
  url="${base/HOST/$host}"
  # /api/v1/public/cities is reachable same-origin (PUB endpoint, doc 11).
  req GET "$url/api/v1/public/cities"
  case "$REPLY_STATUS" in
    200 | 204)
      ct="$(hdr content-type)"
      [[ "$ct" == application/json* ]] \
        && checkpass "$host: /api/v1/public/cities reachable same-origin (JSON)" \
        || checkfail "$host: /api/v1/public/cities content-type '$ct' (want JSON)"
      ;;
    "")
      checkfail "$host: /api/v1/public/cities no response (api up?)"
      ;;
    *)
      # A 5xx means the proxy reached the API but it errored; still proves routing.
      checkfail "$host: /api/v1/public/cities returned $REPLY_STATUS"
      ;;
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
[[ "$REPLY_STATUS" == "404" ]] \
  && checkpass "api. host: non-webhook path returns 404" \
  || checkfail "api. host: '/' returned '$REPLY_STATUS' (want 404)"
req GET "$apihost/api/v1/public/cities"
[[ "$REPLY_STATUS" == "404" ]] \
  && checkpass "api. host: browser API path returns 404" \
  || checkfail "api. host: /api/v1/public/cities returned '$REPLY_STATUS' (want 404)"

echo
log "smoke: $pass passed, $failn failed"
[[ $failn -eq 0 ]] || die "smoke test failed"
ok "smoke test passed"
