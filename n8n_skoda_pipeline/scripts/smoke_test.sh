#!/usr/bin/env bash
# FREE sanity check (no generation, no spend): is the key valid, and do the configured model ids exist for it?
#   GEMINI_API_KEY=... bash scripts/smoke_test.sh [path/to/shots.json]
set -euo pipefail
: "${GEMINI_API_KEY:?set GEMINI_API_KEY}"
CFG="${1:-$(dirname "$0")/../config/shots.json}"
BASE="${API_BASE:-https://generativelanguage.googleapis.com/v1beta}"
python3 - "$CFG" "$BASE" <<'PY'
import json, os, sys, urllib.request
cfg = json.load(open(sys.argv[1])); base = sys.argv[2]
req = urllib.request.Request(f"{base}/models?pageSize=1000", headers={"x-goog-api-key": os.environ["GEMINI_API_KEY"]})
try:
    names = [m["name"].removeprefix("models/") for m in json.load(urllib.request.urlopen(req, timeout=30)).get("models", [])]
except Exception as e:
    sys.exit(f"API call failed: {e}\n→ wrong key, billing not enabled, or network blocked.")
print(f"key OK — {len(names)} models visible")
bad = 0
for label, mid in (("image model", cfg["image_model"]), ("video model", cfg["veo"]["model"])):
    ok = mid in names
    print(f"  {'✓' if ok else '✗'} {label}: {mid}")
    if not ok:
        bad = 1
        print("     similar:", ", ".join(n for n in names if n.split("-")[0] == mid.split("-")[0])[:300])
v = cfg["veo"]; n = sum(1 for s in cfg["shots"])
print(f"estimated Veo cost for all {n} shots: ${n * v['duration_seconds'] * v['price_per_second_usd']:.2f} "
      f"({v['model']}, {v['resolution']}, {v['duration_seconds']}s each @ ${v['price_per_second_usd']}/s)")
sys.exit(bad)
PY
