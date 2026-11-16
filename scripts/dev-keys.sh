#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Create .env from .env.example (if missing) and fill every EMPTY secret with a
# fresh random dev-only value. Existing non-empty values are kept unless
# FORCE=1. Never use these keys outside a developer machine / CI.
#
#   scripts/dev-keys.sh [env-file]      (default: .env)
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
env_file="${1:-$root/.env}"
example="$root/.env.example"
force="${FORCE:-0}"

command -v openssl >/dev/null || { echo "dev-keys: openssl is required" >&2; exit 1; }

if [[ ! -f "$env_file" ]]; then
  cp "$example" "$env_file"
  echo "dev-keys: created $env_file from .env.example"
fi
chmod 600 "$env_file"

rand_b64() { openssl rand -base64 32 | tr -d '\n'; }
rand_pw() { openssl rand -base64 24 | tr -d '\n/+=' | cut -c1-24; }

vapid_pub=""
vapid_priv=""
if command -v node >/dev/null; then
  # P-256 key pair in the base64url "raw" form Web Push libraries expect.
  read -r vapid_pub vapid_priv < <(node -e '
    const c = require("crypto");
    const { publicKey, privateKey } = c.generateKeyPairSync("ec", { namedCurve: "prime256v1" });
    const pub = publicKey.export({ format: "jwk" }), priv = privateKey.export({ format: "jwk" });
    const raw = Buffer.concat([Buffer.from([4]), Buffer.from(pub.x, "base64url"), Buffer.from(pub.y, "base64url")]);
    console.log(raw.toString("base64url") + " " + priv.d);
  ')
else
  echo "dev-keys: node not found; leaving VAPID keys empty (fake push needs none)" >&2
fi

# Value generator per secret key (bash 3.2 compatible: no associative arrays).
gen_for() {
  case "$1" in
    ROVO_JWT_SIGNING_KEY | ROVO_QUOTE_SIGNING_KEY | ROVO_FIELD_ENCRYPTION_KEY | \
      ROVO_TRUSTED_ORIGIN_SECRET | ROVO_FAKEPA_WEBHOOK_SECRET | ROVO_TESTHOOKS_SECRET)
      rand_b64 ;;
    ROVO_ADMIN_BOOTSTRAP_PASSWORD) printf 'dev-%s' "$(rand_pw)" ;;
    ROVO_PUSH_VAPID_PUBLIC_KEY) printf '%s' "$vapid_pub" ;;
    ROVO_PUSH_VAPID_PRIVATE_KEY) printf '%s' "$vapid_priv" ;;
    *) return 1 ;;
  esac
}

tmp="$(mktemp "${env_file}.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
changed=""
new=""
while IFS= read -r line || [[ -n "$line" ]]; do
  key="${line%%=*}"
  if [[ "$line" == *=* && "$line" != \#* ]] && new="$(gen_for "$key")"; then
    value="${line#*=}"
    if [[ -n "$new" && ( -z "$value" || "$force" == "1" ) ]]; then
      line="${key}=${new}"
      changed="$changed $key"
    fi
  fi
  printf '%s\n' "$line"
done <"$env_file" >"$tmp"
cat "$tmp" >"$env_file"

if [[ -n "$changed" ]]; then
  echo "dev-keys: generated${changed}"
else
  echo "dev-keys: all secrets already set (FORCE=1 to rotate)"
fi
