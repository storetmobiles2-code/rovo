#!/usr/bin/env python3
"""Local stand-in for the Gemini / Veo REST API (rehearsal mode: zero cost, no key needed).

Implements only what the n8n workflows call:
  GET  /v1beta/models                                   (model list, for the preflight check)
  POST /v1beta/models/<image-model>:generateContent     (returns a PNG built from the input photo)
  POST /v1beta/models/<veo-model>:predictLongRunning    (returns an operation name)
  GET  /v1beta/models/<veo-model>/operations/<id>       (done:false for the first POLLS_BEFORE_DONE polls)
  GET  /files/<id>:download                             (returns a small MP4 made by ffmpeg)
Run:  python3 tests/mock_gemini.py --port 8099
Then set  api_base = http://host.docker.internal:8099/v1beta  (or 127.0.0.1 outside Docker) in the workflows' Config node.
"""
import argparse, base64, json, os, subprocess, tempfile, threading, time, uuid, io
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

POLLS_BEFORE_DONE = int(os.environ.get("MOCK_POLLS", "2"))
FAIL_SHOT = os.environ.get("MOCK_FAIL_PROMPT_CONTAINS", "")      # set to simulate a safety-filtered clip
OPS = {}            # op name -> dict(polls, prompt, fail)
LOCK = threading.Lock()
LOG = []

IMAGE_MODELS = ["gemini-3.1-flash-image", "gemini-3-pro-image", "gemini-2.5-flash-image"]
VEO_MODELS = ["veo-3.1-generate-preview", "veo-3.1-fast-generate-preview", "veo-3.1-lite-generate-preview"]


def make_png_from(b64: str | None) -> str:
    """Return base64 PNG: the input photo, graded emerald so the pipeline output is visibly 'processed'."""
    try:
        import cv2, numpy as np
        raw = base64.b64decode(b64) if b64 else None
        if raw:
            im = cv2.imdecode(np.frombuffer(raw, np.uint8), cv2.IMREAD_COLOR)
        else:
            im = np.zeros((1920, 1080, 3), np.uint8)
        h, w = im.shape[:2]
        # force 9:16 by centre-crop
        tw = int(h * 9 / 16)
        if tw <= w:
            x0 = (w - tw) // 2; im = im[:, x0:x0 + tw]
        else:
            th = int(w * 16 / 9); y0 = max(0, (h - th) // 2); im = im[y0:y0 + th]
        im = cv2.resize(im, (1080, 1920), interpolation=cv2.INTER_CUBIC)
        tint = np.array([60, 140, 40], np.float32) / 255
        im = np.clip(im.astype(np.float32) * (0.55 + tint), 0, 255).astype(np.uint8)
        ok, buf = cv2.imencode(".png", im)
        return base64.b64encode(buf.tobytes()).decode()
    except Exception as e:  # pragma: no cover
        print("png fallback", e)
        return "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="


def make_mp4(seed: int) -> bytes:
    with tempfile.TemporaryDirectory() as d:
        out = os.path.join(d, "o.mp4")
        subprocess.run(["ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", f"testsrc2=size=1080x1920:rate=24:duration=8", "-vf",
                        f"hue=h={(seed*37)%360},format=yuv420p", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "30", out], check=True)
        return open(out, "rb").read()


class H(BaseHTTPRequestHandler):
    def log_message(self, fmt, *a):
        pass

    def _send(self, code, obj=None, raw=None, ctype="application/json"):
        body = raw if raw is not None else json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _auth(self):
        if not self.headers.get("x-goog-api-key"):
            self._send(403, {"error": {"code": 403, "status": "PERMISSION_DENIED", "message": "API key required (mock)"}})
            return False
        return True

    def do_GET(self):
        LOG.append(("GET", self.path)); 
        if self.path.startswith("/__log"):
            return self._send(200, LOG)
        if not self._auth(): return
        p = self.path.split("?")[0]
        if p == "/v1beta/models":
            return self._send(200, {"models": [{"name": f"models/{m}"} for m in IMAGE_MODELS + VEO_MODELS]})
        if "/operations/" in p:
            name = p[len("/v1beta/"):]
            with LOCK:
                op = OPS.get(name)
                if not op:
                    return self._send(404, {"error": {"code": 404, "message": "operation not found"}})
                op["polls"] += 1
                done = op["polls"] > op["need"]
            if not done:
                return self._send(200, {"name": name, "metadata": {"@type": "mock"}})
            if op["fail"]:
                return self._send(200, {"name": name, "done": True, "response": {"generateVideoResponse": {"raiMediaFilteredCount": 1, "raiMediaFilteredReasons": ["Mock: blocked by safety filter"]}}})
            vid = uuid.uuid4().hex[:8]
            op["file"] = vid
            host = self.headers.get("Host")
            return self._send(200, {"name": name, "done": True, "response": {"@type": "mock", "generateVideoResponse": {"generatedSamples": [{"video": {"uri": f"http://{host}/files/{vid}:download?alt=media"}}]}}})
        if p.startswith("/files/"):
            seed = abs(hash(p)) % 1000
            return self._send(200, raw=make_mp4(seed), ctype="video/mp4")
        self._send(404, {"error": {"message": "not found " + p}})

    def do_POST(self):
        n = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(n) or b"{}")
        LOG.append(("POST", self.path, {k: (v if k != "instances" and k != "contents" else "…") for k, v in body.items()}))
        if not self._auth(): return
        p = self.path.split("?")[0]
        if p.endswith(":generateContent"):
            b64 = None
            for part in body.get("contents", [{}])[0].get("parts", []):
                d = part.get("inlineData") or part.get("inline_data")
                if d: b64 = d.get("data")
            if not b64:
                return self._send(400, {"error": {"code": 400, "message": "no input image (mock)"}})
            return self._send(200, {"candidates": [{"content": {"parts": [{"text": "ok"}, {"inlineData": {"mimeType": "image/png", "data": make_png_from(b64)}}]}, "finishReason": "STOP"}]})
        if p.endswith(":predictLongRunning"):
            inst = body.get("instances", [{}])[0]
            prm = body.get("parameters", {})
            if "image" in inst:
                img = inst["image"]
                if not (img.get("inlineData") or img.get("bytesBase64Encoded")):
                    return self._send(400, {"error": {"code": 400, "message": "bad image (mock)"}})
            if prm.get("aspectRatio") not in ("9:16", "16:9"):
                return self._send(400, {"error": {"code": 400, "message": "aspectRatio (mock)"}})
            if prm.get("resolution") in ("1080p", "4k") and str(prm.get("durationSeconds")) != "8":
                return self._send(400, {"error": {"code": 400, "message": "1080p requires 8s (mock)"}})
            name = f"{p.split(':')[0].lstrip('/v1beta/')}"
            model = p.split("/models/")[1].split(":")[0]
            opname = f"models/{model}/operations/{uuid.uuid4().hex[:10]}"
            fail = bool(FAIL_SHOT) and FAIL_SHOT in inst.get("prompt", "")
            with LOCK:
                OPS[opname] = {"polls": 0, "prompt": inst.get("prompt", ""), "fail": fail,
                               "need": POLLS_BEFORE_DONE + (sum(map(ord, inst.get("prompt", ""))) % 3)}   # staggered: shots finish on different polls
            return self._send(200, {"name": opname})
        self._send(404, {"error": {"message": "not found " + p}})


if __name__ == "__main__":
    ap = argparse.ArgumentParser(); ap.add_argument("--port", type=int, default=8099); ap.add_argument("--host", default="0.0.0.0")
    a = ap.parse_args()
    print(f"mock Gemini/Veo on http://{a.host}:{a.port}/v1beta  (polls before done: {POLLS_BEFORE_DONE})", flush=True)
    ThreadingHTTPServer((a.host, a.port), H).serve_forever()
