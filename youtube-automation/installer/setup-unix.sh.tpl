#!/usr/bin/env bash
# YouTube automation installer for macOS (and Linux).
# Google Drive -> NotebookLM Video Overview -> Claude metadata -> YouTube upload,
# orchestrated by a free self-hosted n8n.
#
# Double-click this file in Finder, or run:  bash YouTubeAutomation-Setup.command
# Re-running is safe: it updates the code, keeps your .env and n8n data.
#   --relogin   sign in to NotebookLM again (e.g. after the login expired)
#   --no-start  write the files and settings but do not start the containers
#
# Built __BUILD_DATE__ (version __VERSION__). Everything the project needs is
# embedded at the bottom of this file.

set -euo pipefail

INSTALL_DIR="${YTA_HOME:-$HOME/youtube-automation}"
NOTEBOOKLM_VERSION="__NOTEBOOKLM_VERSION__"
RELOGIN=0
START=1
for arg in "$@"; do
  case "$arg" in
    --relogin) RELOGIN=1 ;;
    --no-start) START=0 ;;
    *) echo "Unknown option: $arg"; exit 2 ;;
  esac
done

bold() { printf '\n\033[1m%s\033[0m\n' "$*"; }
info() { printf '  %s\n' "$*"; }
die()  { printf '\n\033[31mError:\033[0m %s\n' "$*" >&2; exit 1; }

# Keep the Terminal window open on exit when launched from Finder.
if [[ "${TERM_PROGRAM:-}" == "Apple_Terminal" && -z "${YTA_NO_PAUSE:-}" ]]; then
  trap 'printf "\nPress Return to close this window."; read -r _' EXIT
fi

bold "YouTube automation setup  (installs into $INSTALL_DIR)"

# ------------------------------------------------------------------ Docker
bold "1/6  Checking Docker"
if ! command -v docker >/dev/null 2>&1; then
  if [[ "$(uname)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
    read -r -p "  Docker Desktop is not installed. Install it with Homebrew now? [Y/n] " ans
    if [[ ! "$ans" =~ ^[Nn] ]]; then
      brew install --cask docker
      open -a Docker || true
    fi
  fi
  command -v docker >/dev/null 2>&1 || die "Install Docker Desktop from https://www.docker.com/products/docker-desktop/ , start it once, then run this installer again."
fi
if ! docker info >/dev/null 2>&1; then
  [[ "$(uname)" == "Darwin" ]] && open -a Docker 2>/dev/null || true
  info "Waiting for Docker to start..."
  for _ in $(seq 1 60); do docker info >/dev/null 2>&1 && break; sleep 3; done
  docker info >/dev/null 2>&1 || die "Docker is installed but not running. Start Docker Desktop and run this installer again."
fi
docker compose version >/dev/null 2>&1 || die "This Docker has no 'docker compose'. Update Docker Desktop."
info "Docker is running."

# ------------------------------------------------------------------ files
bold "2/6  Writing project files"
put() {
  mkdir -p "$INSTALL_DIR/$(dirname "$1")"
  printf '%s' "$2" | base64 --decode > "$INSTALL_DIR/$1"
}
__PAYLOAD__
info "Files written to $INSTALL_DIR"

# ------------------------------------------------------------------ settings
bold "3/6  Settings (.env)"
ENV_FILE="$INSTALL_DIR/.env"
[[ -f "$ENV_FILE" ]] || cp "$INSTALL_DIR/.env.example" "$ENV_FILE"
chmod 600 "$ENV_FILE"

get_env() { awk -F= -v k="$1" '$1==k {sub(/^[^=]*=/, ""); print; exit}' "$ENV_FILE"; }
set_env() {
  local tmp; tmp="$(mktemp)"
  awk -v k="$1" -v v="$2" 'BEGIN{done=0} $0 ~ "^"k"=" {print k"="v; done=1; next} {print} END{if(!done) print k"="v}' "$ENV_FILE" > "$tmp"
  cat "$tmp" > "$ENV_FILE"; rm -f "$tmp"
}
random_hex() { od -An -tx1 -N32 /dev/urandom | tr -d ' \n'; }

[[ -n "$(get_env N8N_ENCRYPTION_KEY)" ]] || set_env N8N_ENCRYPTION_KEY "$(random_hex)"
[[ -n "$(get_env BRIDGE_TOKEN)" ]] || set_env BRIDGE_TOKEN "$(random_hex)"
[[ -n "$(get_env NOTEBOOKLM_HOME)" ]] || set_env NOTEBOOKLM_HOME "$HOME/.notebooklm"
if [[ -z "$(get_env ANTHROPIC_API_KEY)" ]]; then
  info "Claude writes the YouTube title, description and tags."
  info "Create an API key at https://platform.claude.com/settings/keys"
  read -r -s -p "  Paste your Anthropic API key (input hidden): " key; echo
  [[ -n "$key" ]] || die "An Anthropic API key is required."
  set_env ANTHROPIC_API_KEY "$key"
fi
info "Settings saved in $ENV_FILE (edit it to change video style, privacy, language...)"

# ------------------------------------------------------------------ NotebookLM login
bold "4/6  NotebookLM sign-in"
NLM_HOME="$(get_env NOTEBOOKLM_HOME)"
if [[ $RELOGIN -eq 1 || ! -f "$NLM_HOME/profiles/default/storage_state.json" ]]; then
  export PATH="$HOME/.local/bin:$PATH"
  if ! command -v uv >/dev/null 2>&1; then
    info "Installing uv (Python tool installer)..."
    curl -LsSf https://astral.sh/uv/install.sh | sh
    export PATH="$HOME/.local/bin:$PATH"
  fi
  uv tool install --force "notebooklm-py[browser]==$NOTEBOOKLM_VERSION"
  info "A browser window opens next. Sign in with the Google account you use for"
  info "NotebookLM; the window closes by itself once you are signed in."
  login_args=()
  # The bundled Chromium can crash on macOS 15+; use installed Chrome when present.
  [[ -d "/Applications/Google Chrome.app" ]] && login_args=(--browser chrome)
  NOTEBOOKLM_HOME="$NLM_HOME" notebooklm login "${login_args[@]+"${login_args[@]}"}"
  NOTEBOOKLM_HOME="$NLM_HOME" notebooklm auth check --test || die "NotebookLM login check failed. Run this installer again with --relogin."
else
  info "Existing NotebookLM login found ($NLM_HOME). Use --relogin to replace it."
fi

# ------------------------------------------------------------------ Claude Code skill
bold "5/6  Installing the youtube-automation skill for Claude Code"
mkdir -p "$HOME/.claude/skills/youtube-automation"
cp "$INSTALL_DIR/skills/youtube-automation/SKILL.md" "$HOME/.claude/skills/youtube-automation/SKILL.md"
info "Copied to ~/.claude/skills/youtube-automation (the bridge uses its own copy)."

# ------------------------------------------------------------------ start
bold "6/6  Starting n8n and the bridge"
if [[ $START -eq 0 ]]; then
  info "Skipped (--no-start). Start later with: cd \"$INSTALL_DIR\" && docker compose up -d --build"
  exit 0
fi
cd "$INSTALL_DIR"
docker compose up -d --build
info "Waiting for n8n..."
for _ in $(seq 1 60); do curl -fsS http://localhost:5678/healthz >/dev/null 2>&1 && break; sleep 3; done
curl -fsS http://localhost:5678/healthz >/dev/null 2>&1 || die "n8n did not come up. See: docker compose -f \"$INSTALL_DIR/docker-compose.yml\" logs n8n"
docker compose exec -T n8n n8n import:workflow --input=/home/node/workflows/youtube-automation.json >/dev/null
info "Workflow imported."

bold "Done. Finish these steps in n8n (http://localhost:5678):"
cat <<'STEPS'
  1. Create your n8n owner account (stored only on this computer).
  2. Open the workflow "YouTube automation: Drive -> NotebookLM -> Claude -> YouTube".
  3. In "New file in Drive folder": create a Google Drive OAuth2 credential and
     pick the folder to watch. Use the same credential in "Download file".
  4. In "Upload to YouTube": create a YouTube OAuth2 credential.
     (Both need a Google Cloud OAuth client - see README.md, section
     "Google credentials", for the 5-minute setup.)
  5. Toggle the workflow to Active / Publish it.
  Drop a PDF, Doc or text file into the folder. About 10-30 minutes later the
  video is on your channel as a PRIVATE upload for you to review.
STEPS
info "Guide: $INSTALL_DIR/README.md     Logs: cd \"$INSTALL_DIR\" && docker compose logs -f"
command -v open >/dev/null 2>&1 && open http://localhost:5678 || true
