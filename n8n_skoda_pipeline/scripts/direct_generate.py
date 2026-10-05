#!/usr/bin/env python3
"""Run the same pipeline as n8n workflows 01 + 02 directly against Google's API (no n8n needed).

The API key is read ONLY from the environment (GEMINI_API_KEY) — it is never written to disk.

  GEMINI_API_KEY=... python3 direct_generate.py stills  --data DATA [--only S01_reveal_orbit ...]
  GEMINI_API_KEY=... python3 direct_generate.py animate --data DATA [--only ...] [--yes]     # without --yes: dry-run (prints cost)

DATA contains inputs/ (front.jpg, side.jpg, rear34.jpg), config/shots.json, stills/ (+ stills/approved/), clips/.
`stills` writes stills/<id>.png and copies them to stills/approved/ with --approve.
"""
import argparse, base64, json, mimetypes, os, shutil, subprocess, sys, time, urllib.error, urllib.request

BASE = os.environ.get("API_BASE", "https://generativelanguage.googleapis.com/v1beta")


def key():
    k = os.environ.get("GEMINI_API_KEY")
    if not k: sys.exit("GEMINI_API_KEY is not set in the environment")
    return k


def call(method, url, body=None, timeout=300, tries=6, raw=False):
    data = json.dumps(body).encode() if body is not None else None
    for i in range(tries):
        req = urllib.request.Request(url, data=data, method=method, headers={"x-goog-api-key": key(), "Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=timeout) as r:
                payload = r.read()
                return payload if raw else json.loads(payload)
        except urllib.error.HTTPError as e:
            msg = e.read().decode(errors="replace")[:600]
            if e.code in (429, 500, 502, 503, 504) and i < tries - 1:
                wait = min(8 * 2 ** i, 90); print(f"   HTTP {e.code} (busy) – retry in {wait}s", flush=True); time.sleep(wait); continue
            sys.exit(f"HTTP {e.code} from {url.split('?')[0]}\n{msg}")
        except (urllib.error.URLError, TimeoutError) as e:
            if i < tries - 1: time.sleep(8); continue
            sys.exit(f"network error: {e}")


def load(data):
    return json.load(open(os.path.join(data, "config", "shots.json")))


def b64(path):
    return base64.b64encode(open(path, "rb").read()).decode(), mimetypes.guess_type(path)[0] or "image/jpeg"


def crop_still(data, s):
    c = s["crop"]; out = os.path.join(data, "stills", s["id"] + ".png")
    vf = (f"crop=w='trunc(ih*{c['h']}*9/16/2)*2':h='trunc(ih*{c['h']}/2)*2':x='max(0,min(iw-out_w,{c['cx']}*iw-out_w/2))':"
          f"y='max(0,min(ih-out_h,{c['cy']}*ih-out_h/2))',scale=1080:1920:flags=lanczos,unsharp=5:5:0.8")
    subprocess.run(["ffmpeg", "-v", "error", "-y", "-i", os.path.join(data, "inputs", s["source"]), "-vf", vf, "-frames:v", "1", out], check=True)
    return out


def ai_still(data, cfg, s):
    d64, mime = b64(os.path.join(data, "inputs", s["source"]))
    prompt = f"{s['still_prompt']}\n\n{cfg['identity_rule']}\nThe car is a {cfg['meta']['car']}."
    body = {"contents": [{"role": "user", "parts": [{"text": prompt}, {"inlineData": {"mimeType": mime, "data": d64}}]}],
            "generationConfig": {"responseModalities": ["IMAGE"], "imageConfig": {"aspectRatio": "9:16"}}}
    res = call("POST", f"{BASE}/models/{cfg['image_model']}:generateContent", body)
    parts = (res.get("candidates") or [{}])[0].get("content", {}).get("parts", [])
    img = next((p.get("inlineData") or p.get("inline_data") for p in parts if p.get("inlineData") or p.get("inline_data")), None)
    if not img:
        why = res.get("promptFeedback", {}).get("blockReason") or (res.get("candidates") or [{}])[0].get("finishReason")
        txt = " ".join(p.get("text", "") for p in parts)[:300]
        sys.exit(f"{s['id']}: no image returned ({why}) {txt}")
    out = os.path.join(data, "stills", s["id"] + ".png")
    open(out, "wb").write(base64.b64decode(img["data"]))
    return out


def cmd_stills(a):
    cfg = load(a.data); os.makedirs(os.path.join(a.data, "stills", "approved"), exist_ok=True)
    for s in cfg["shots"]:
        if s["mode"] == "text_only" or (a.only and s["id"] not in a.only): continue
        t = time.time()
        out = crop_still(a.data, s) if s["mode"] == "crop_only" else ai_still(a.data, cfg, s)
        print(f"{s['id']}: {s['mode']} -> {out} ({time.time() - t:.0f}s)", flush=True)
        if a.approve: shutil.copy(out, os.path.join(a.data, "stills", "approved", os.path.basename(out)))


def cmd_animate(a):
    cfg = load(a.data); V = cfg["veo"]
    todo = [s for s in cfg["shots"] if (not a.only or s["id"] in a.only)
            and (s["mode"] == "text_only" or os.path.exists(os.path.join(a.data, "stills", "approved", s["id"] + ".png")))]
    cost = len(todo) * V["duration_seconds"] * V["price_per_second_usd"]
    print(f"{len(todo)} clips x {V['duration_seconds']}s  {V['model']} {V['resolution']}  est. ${cost:.2f}  -> {[s['id'] for s in todo]}")
    if not a.yes: print("DRY RUN (add --yes to generate)"); return
    os.makedirs(os.path.join(a.data, "clips"), exist_ok=True)
    ops = {}
    for s in todo:
        inst = {"prompt": s["video_prompt"]}
        if s["mode"] != "text_only":
            d64, mime = b64(os.path.join(a.data, "stills", "approved", s["id"] + ".png"))
            inst["image"] = {"inlineData": {"mimeType": mime, "data": d64}}
        params = {"aspectRatio": V["aspect_ratio"], "resolution": V["resolution"], "durationSeconds": int(V["duration_seconds"])}
        if V.get("send_negative_prompt") and cfg.get("negative_prompt"): params["negativePrompt"] = cfg["negative_prompt"]
        r = call("POST", f"{BASE}/models/{V['model']}:predictLongRunning", {"instances": [inst], "parameters": params})
        ops[s["id"]] = r["name"]; print(f"started {s['id']}: {r['name']}", flush=True)
    t0 = time.time(); pending = dict(ops)
    while pending and time.time() - t0 < V["timeout_minutes"] * 60:
        time.sleep(V["poll_every_seconds"])
        for sid, name in list(pending.items()):
            r = call("GET", f"{BASE}/{name}", timeout=120)
            if not r.get("done") and not r.get("error"): continue
            del pending[sid]
            g = (r.get("response") or {}).get("generateVideoResponse", {})
            uri = ((g.get("generatedSamples") or [{}])[0].get("video") or {}).get("uri")
            if r.get("error") or not uri:
                note = os.path.join(a.data, "clips", sid + ".FAILED.txt")
                open(note, "w").write(json.dumps(r.get("error") or g or r)[:1500]); print(f"FAILED {sid}: see {note}", flush=True); continue
            out = os.path.join(a.data, "clips", sid + ".mp4")
            open(out, "wb").write(call("GET", uri, timeout=600, raw=True)); print(f"saved {out} ({os.path.getsize(out) // 1024} KB, {time.time() - t0:.0f}s)", flush=True)
    if pending: print("TIMED OUT:", list(pending))


if __name__ == "__main__":
    ap = argparse.ArgumentParser(); sub = ap.add_subparsers(dest="cmd", required=True)
    for n in ("stills", "animate"):
        p = sub.add_parser(n); p.add_argument("--data", required=True); p.add_argument("--only", nargs="*")
        if n == "stills": p.add_argument("--approve", action="store_true")
        else: p.add_argument("--yes", action="store_true")
    a = ap.parse_args()
    {"stills": cmd_stills, "animate": cmd_animate}[a.cmd](a)
