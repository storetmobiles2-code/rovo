#!/usr/bin/env python3
"""Make test copies of the workflows: point Config at the mock API / a scratch data dir (only the Config node is touched)."""
import json, sys
src, dst, data_dir, dry = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4] == "dry"
d = json.load(open(src))
for n in d["nodes"]:
    if n["name"] == "Config":
        js = n["parameters"]["jsCode"]
        js = js.replace("'https://generativelanguage.googleapis.com/v1beta'", "'http://127.0.0.1:8099/v1beta'")
        js = js.replace("data_dir: '/data'", f"data_dir: '{data_dir}'")
        js = js.replace("dry_run: true", f"dry_run: {'true' if dry else 'false'}")
        js = js.replace("auto_approve: false", "auto_approve: true")
        n["parameters"]["jsCode"] = js
if len(sys.argv) > 5: d["id"], d["name"] = sys.argv[5], sys.argv[6]
json.dump(d, open(dst, "w"))
