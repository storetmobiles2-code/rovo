#!/usr/bin/env bash
# One-time setup, run INSIDE the container:  docker compose exec n8n bash /data/scripts/setup_n8n.sh
# Stores GEMINI_API_KEY (from .env) as n8n's encrypted credential "Gemini API Key" and imports the 3 workflows.
set -euo pipefail
: "${GEMINI_API_KEY:?GEMINI_API_KEY is empty — put it in .env and run: docker compose up -d --force-recreate}"
TMP="$(mktemp)"; trap 'rm -f "$TMP"' EXIT
python3 - "$TMP" <<'PY'
import json, os, sys
json.dump([{"id": "gemini_key", "name": "Gemini API Key", "type": "httpHeaderAuth",
            "data": {"name": "x-goog-api-key", "value": os.environ["GEMINI_API_KEY"]}}], open(sys.argv[1], "w"))
PY
n8n import:credentials --input="$TMP"
for f in /data/workflows/0*.json; do n8n import:workflow --input="$f"; done
echo; echo "Done. Open http://localhost:5678 → Workflows → run 01, then 02, then 03."
