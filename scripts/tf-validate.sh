#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# OpenTofu fmt-check + validate for every module and env under deploy/terraform.
# VALIDATE ONLY — never `apply`. Runs tofu in a container (no local install
# needed) with the backend disabled, so no cloud credentials or state are
# touched.
#
#   scripts/tf-validate.sh
set -euo pipefail
SCRIPT_NAME=tf-validate
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

tf_dir="$ROVO_ROOT/deploy/terraform"
[[ -d "$tf_dir" ]] || die "no deploy/terraform directory"
img="${TOFU_IMG:-ghcr.io/opentofu/opentofu:1.13.1}"
need docker

# tofu downloads providers during `init`; route it through the agent proxy and
# trust the proxy CA. --network host reaches the loopback proxy; both are
# harmless when there is no proxy.
docker_common=(--rm --network host -v "$tf_dir":/w)
proxy_env=()
[[ -n "${HTTPS_PROXY:-}" ]] && proxy_env+=(-e "HTTPS_PROXY=$HTTPS_PROXY")
[[ -n "${HTTP_PROXY:-}" ]] && proxy_env+=(-e "HTTP_PROXY=$HTTP_PROXY")
[[ -n "${NO_PROXY:-}" ]] && proxy_env+=(-e "NO_PROXY=$NO_PROXY")
ca_mount=()
if [[ -f /root/.ccr/ca-bundle.crt ]]; then
  ca_mount=(-v /root/.ccr/ca-bundle.crt:/ca.crt:ro)
  proxy_env+=(-e SSL_CERT_FILE=/ca.crt)
fi

tofu() { docker run "${docker_common[@]}" "${proxy_env[@]}" "${ca_mount[@]}" -w /w --entrypoint /usr/local/bin/tofu "$img" "$@"; }

log "tofu fmt -check -recursive"
if ! tofu fmt -check -recursive; then
  die "terraform files need formatting (run: docker run --rm -v \"$tf_dir\":/w -w /w $img fmt -recursive)"
fi
ok "formatting is canonical"

# Validate each env (they wire the modules together). -backend=false means no
# remote state is configured or contacted.
rc=0
while IFS= read -r env_dir; do
  rel="${env_dir#"$tf_dir"/}"
  log "init+validate $rel"
  if ! docker run "${docker_common[@]}" "${proxy_env[@]}" "${ca_mount[@]}" -w "/w/$rel" --entrypoint /bin/sh "$img" -c \
    'tofu init -backend=false -input=false -no-color >/dev/null && tofu validate -no-color'; then
    fail "$rel failed validation"
    rc=1
  else
    ok "$rel valid"
  fi
done < <(find "$tf_dir/envs" -mindepth 1 -maxdepth 1 -type d | sort)

[[ $rc -eq 0 ]] || die "tofu validate failed"
ok "all OpenTofu envs validate"
