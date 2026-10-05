#!/usr/bin/env python3
"""Finishing engine: AI clips (Veo) + EDL  ->  beat-synced motion-designed master with sound.

  python3 finish.py --check  --edl config/edl.json --clips clips
  python3 finish.py --edl config/edl.json --clips clips --audio audio/score.wav --out out/final.mp4 [--scale 0.5] [--workers 4]

Everything is open source: FFmpeg (decode/encode), OpenCV + NumPy (compositing in linear light), Pillow (type).
Per frame it applies: speed-ramp retime (frame blending) -> zoom-punch / shake -> auto star-glints on the real highlights ->
light sweeps -> impact flash -> crystal-shard wipes -> whip blur + radial zoom + RGB split at the cuts -> bloom / anamorphic
streaks -> unified emerald grade -> vignette -> grain -> kinetic title lock-up -> fade.
"""
import argparse, json, math, multiprocessing as mp, os, shutil, subprocess, sys, tempfile, time
import numpy as np, cv2

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import fx
from fx import lerp, clamp, ease_io, ease_out, ease_in, smooth, pulse

FONT_TITLE = os.path.join(HERE, "fonts", "Anton-Regular.ttf")
FONT_SUB = os.path.join(HERE, "fonts", "Rajdhani-SemiBold.ttf")
ASSETS = os.path.join(HERE, "assets")
EMERALD = (0.10, 0.95, 0.50)

# ------------------------------------------------------------------------------------------ EDL
def load_edl(path):
    e = json.load(open(path))
    e.setdefault("fps", 30); e.setdefault("bpm", 90)
    e.setdefault("grade", {}); e.setdefault("title", {"text": "ŠKODA", "sub": "SUPERB", "delay_frames": 8})
    return e


def build_timeline(edl):
    fps, bpm = edl["fps"], edl["bpm"]
    per_beat = fps * 60.0 / bpm
    t, out = 0, []
    for s in edl["segments"]:
        n = int(round(s["beats"] * per_beat))
        d = dict(s); d["start"], d["end"], d["n"] = t, t + n, n
        d.setdefault("speed", [[0, 1.0], [1, 1.0]])
        out.append(d); t += n
    return out, t, per_beat


def speed_at(seg, u):
    k = seg["speed"]
    if u <= k[0][0]: return k[0][1]
    for (u0, s0), (u1, s1) in zip(k, k[1:]):
        if u <= u1: return lerp(s0, s1, (u - u0) / max(u1 - u0, 1e-6))
    return k[-1][1]


def src_times(seg, fps):
    n = seg["n"]; t = seg["src_in"]; out = []
    for k in range(n):
        out.append(t)
        t += max(speed_at(seg, k / max(n - 1, 1)), 0.05) / fps
    return out


def probe(path):
    r = subprocess.run(["ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height,r_frame_rate,duration",
                        "-show_entries", "format=duration", "-of", "json", path], capture_output=True, text=True)
    j = json.loads(r.stdout or "{}")
    st = (j.get("streams") or [{}])[0]
    num, den = (st.get("r_frame_rate", "24/1").split("/") + ["1"])[:2]
    dur = float(st.get("duration") or j.get("format", {}).get("duration") or 0)
    return dict(w=st.get("width"), h=st.get("height"), fps=float(num) / max(float(den), 1), dur=dur)


def check(edl, clips_dir):
    tl, total, _ = build_timeline(edl)
    need, problems, info = {}, [], {}
    for s in tl:
        tmax = src_times(s, edl["fps"])[-1] + 0.15
        need[s["clip"]] = max(need.get(s["clip"], 0), tmax)
    for clip, tmax in need.items():
        p = os.path.join(clips_dir, clip)
        if not os.path.exists(p):
            note = os.path.join(clips_dir, clip.replace(".mp4", "") + ".FAILED.txt")
            problems.append(f"MISSING {clip}" + ("  (generation FAILED — see " + os.path.basename(note) + ")" if os.path.exists(note) else "  (run workflow 02)"))
            continue
        pr = probe(p); info[clip] = pr
        if pr["dur"] + 0.05 < tmax:
            problems.append(f"SHORT {clip}: {pr['dur']:.1f}s available, EDL needs {tmax:.1f}s — lower src_in/beats or slow-mo in config/edl.json")
        if pr["w"] and pr["h"] and pr["w"] > pr["h"]:
            problems.append(f"LANDSCAPE {clip}: {pr['w']}x{pr['h']} (expected 9:16)")
    res = dict(ok=not problems, frames=total, seconds=total / edl["fps"], segments=len(tl), clips_needed=sorted(need), problems=problems)
    print(json.dumps(res, indent=1, ensure_ascii=False))
    return 0 if not problems else 2


# ------------------------------------------------------------------------------------------ source frames
def extract(edl, tl, clips_dir, tmp):
    """Pull exactly the needed source range of every segment (conformed to the output size) as JPEGs with ffmpeg."""
    W, H = fx.CW(), fx.CH()
    meta = {}
    for s in tl:
        times = src_times(s, edl["fps"])
        t0 = max(times[0] - 0.05, 0); t1 = times[-1] + 0.25
        clip = os.path.join(clips_dir, s["clip"]); pr = probe(clip); nf = pr["fps"] or 24
        d = os.path.join(tmp, s["id"]); os.makedirs(d, exist_ok=True)
        subprocess.run(["ffmpeg", "-v", "error", "-y", "-ss", f"{t0:.3f}", "-t", f"{t1 - t0:.3f}", "-i", clip,
                        "-vf", f"scale={W}:{H}:force_original_aspect_ratio=increase:flags=lanczos,crop={W}:{H},fps={nf}",
                        "-q:v", "2", os.path.join(d, "%04d.jpg")], check=True)
        n = len([f for f in os.listdir(d) if f.endswith(".jpg")])
        meta[s["id"]] = dict(dir=d, t0=t0, fps=nf, count=n, times=times)
    return meta


# ------------------------------------------------------------------------------------------ fx helpers
_shard = {}
def shard(name):
    if name not in _shard:
        im = cv2.imread(os.path.join(ASSETS, name + ".png"), cv2.IMREAD_UNCHANGED).astype(np.float32) / 255
        rgb = fx.to_linear_u8(cv2.cvtColor((im[..., :3] * 255).astype(np.uint8), cv2.COLOR_BGR2RGB))
        a = im[..., 3:4]
        _shard[name] = np.dstack([rgb * a, a[..., 0]]).astype(np.float32)
    return _shard[name]


def star(rgb, pos, gain, size=90, color=(0.75, 1.0, 0.9)):
    if gain <= 0.01: return
    h, w = rgb.shape[:2]
    cx, cy = pos[0], pos[1]
    r = int(size * 3 * fx.RS)
    x0, x1 = max(int(cx) - r, 0), min(int(cx) + r, w); y0, y1 = max(int(cy) - r, 0), min(int(cy) + r, h)
    if x1 <= x0 or y1 <= y0: return
    yy, xx = np.mgrid[y0:y1, x0:x1].astype(np.float32)
    dx, dy = (xx - cx) / fx.RS, (yy - cy) / fx.RS
    arm = np.exp(-np.abs(dy) / 2.2) * np.exp(-np.abs(dx) / (size * 0.5)) + np.exp(-np.abs(dx) / 2.2) * np.exp(-np.abs(dy) / (size * 0.5))
    d2, e2 = (dx + dy) / 1.414, (dx - dy) / 1.414
    diag = 0.45 * (np.exp(-np.abs(e2) / 2.0) * np.exp(-np.abs(d2) / (size * 0.25)) + np.exp(-np.abs(d2) / 2.0) * np.exp(-np.abs(e2) / (size * 0.25)))
    core = np.exp(-(dx * dx + dy * dy) / (size * 0.07) ** 2) * 1.2
    rgb[y0:y1, x0:x1] += ((arm + diag + core) * gain)[..., None] * np.asarray(color, np.float32)


def find_highlights(lin, n=2):
    """Brightest specular points of the *real* frame (headlamps, chrome, crystal tips) for glints."""
    lum = lin.max(2)
    small = cv2.resize(lum, (lum.shape[1] // 8, lum.shape[0] // 8), interpolation=cv2.INTER_AREA)
    thr = max(0.65, float(np.percentile(small, 99.7)))
    peaks = (small >= cv2.dilate(small, np.ones((9, 9), np.uint8))) & (small >= thr)
    ys, xs = np.where(peaks)
    order = np.argsort(-small[ys, xs])[:n]
    return [(float(xs[i] * 8 + 4), float(ys[i] * 8 + 4), float(small[ys[i], xs[i]])) for i in order]


def beat_pulse(g, per_beat, decay=3.2, offset=2):
    ph = g % per_beat
    return math.exp(-max(ph - offset, 0) / decay) if ph >= offset else math.exp(-(ph + per_beat - offset) / decay) * 0.0


def shake_vec(k, amp, decay=7.0, fps=30):
    if amp <= 0: return (0.0, 0.0)
    t = k / fps; e = amp * math.exp(-decay * t)
    return (e * math.sin(t * 61 + 1.3), e * 0.8 * math.cos(t * 53))


# ------------------------------------------------------------------------------------------ frame renderer
G = {}   # shared (fork) state: edl, timeline, meta, args

def seg_of(g):
    for s in G["tl"]:
        if s["start"] <= g < s["end"]: return s
    return G["tl"][-1]


def load_src(seg, k):
    m = G["meta"][seg["id"]]
    pos = (m["times"][k] - m["t0"]) * m["fps"]
    i0 = int(math.floor(pos)); a = pos - i0
    def rd(i):
        i = min(max(i, 0), m["count"] - 1)
        return cv2.imread(os.path.join(m["dir"], f"{i + 1:04d}.jpg"))
    f0 = rd(i0)
    if a > 0.02 and i0 + 1 < m["count"]:
        f0 = cv2.addWeighted(f0, 1 - a, rd(i0 + 1), a, 0)
    return f0


def whip_for(g):
    for s in G["tl"][:-1]:
        o = s.get("out")
        if not o: continue
        c = s["end"]
        if c - 3 <= g < c:
            k = (g - (c - 3) + 1) / 3.0
            return dict(angle=o.get("dir", 0), length=lerp(20, o.get("len", 130), k), zoom=o.get("zoom", 0.04) * k)
        if c <= g < c + 3:
            k = 1 - (g - c) / 3.0
            return dict(angle=o.get("dir", 0), length=lerp(0, o.get("len", 130), k), zoom=o.get("zoom", 0.04) * k * 0.6)
    return None


def shard_wipe(rgb, g):
    for s in G["tl"][:-1]:
        w = s.get("wipe")
        if not w: continue
        c, n = s["end"], int(w.get("frames", 7))
        a0 = c - n // 2
        if a0 <= g < a0 + n:
            u = (g - a0) / max(n - 1, 1)
            H, Wd = rgb.shape[:2]
            sh = shard(w["plate"])
            sc = 1.9 * fx.RS * (1 + 0.15 * u)
            dirn = 1 if w.get("dir", "lr") == "lr" else -1
            x = lerp(-0.35, 1.35, u) if dirn == 1 else lerp(1.35, -0.35, u)
            M = np.array([[sc, 0, x * Wd - sc * 540], [0, sc, 0.55 * H - sc * 1100]], np.float32)
            p = cv2.warpAffine(sh, M, (Wd, H), flags=cv2.INTER_LINEAR, borderMode=cv2.BORDER_CONSTANT)
            p = fx.blur_rgba(p, 22 * fx.RS)
            rgb[...] = p[..., :3] * 1.1 + rgb * (1 - p[..., 3:4])
    return rgb


def title_overlay(rgb, lf, t):
    delay = G["edl"]["title"].get("delay_frames", 8)
    if lf < delay: return rgb
    W, H = fx.CW(), fx.CH()
    a_in = ease_out(clamp((lf - delay) / 14.0))
    tr = lerp(60, 14, ease_out(clamp((lf - delay) / 20.0), 3))
    m1, _ = fx.text_layer(t["text"], FONT_TITLE, 206, tracking=tr, pos=(540, 900))
    sub = " ".join(t["sub"]) if False else t["sub"]
    m2, _ = fx.text_layer(sub, FONT_SUB, 82, tracking=lerp(60, 34, ease_out(clamp((lf - delay - 6) / 20.0))), pos=(540, 1050))
    a1 = cv2.resize(m1, (W, H), interpolation=cv2.INTER_AREA); a2 = cv2.resize(m2, (W, H), interpolation=cv2.INTER_AREA)
    xx = np.arange(W, dtype=np.float32)[None, :] / fx.RS
    glint = np.exp(-((xx - lerp(-100, 1200, ease_io(clamp((lf - delay - 8) / 20.0)))) / 80.0) ** 2)
    col = a1[..., None] * np.array([0.93, 1.0, 0.97], np.float32) * a_in * (0.85 + 1.2 * glint[..., None] * np.array([0.1, 0.9, 0.5], np.float32))
    col2 = a2[..., None] * np.array([0.30, 1.0, 0.60], np.float32) * ease_out(clamp((lf - delay - 6) / 12.0)) * 0.9
    line = np.zeros((H, W, 1), np.float32)
    lw = int(lerp(0, 360, ease_out(clamp((lf - delay - 4) / 18.0))) * fx.RS)
    if lw > 1:
        y = int(992 * fx.RS)
        cv2.line(line, (int(540 * fx.RS) - lw // 2, y), (int(540 * fx.RS) + lw // 2, y), 1.0, max(1, int(2 * fx.RS)), cv2.LINE_AA)
    col3 = line * np.array([0.3, 1.0, 0.6], np.float32) * 0.8
    glow = cv2.GaussianBlur(col + col2 + col3, (0, 0), 14 * fx.RS) * 0.9
    return rgb + col + col2 + col3 + glow


def render_frame(g):
    edl, per_beat, total = G["edl"], G["per_beat"], G["total"]
    gr = edl.get("grade", {})
    seg = seg_of(g); k = g - seg["start"]; n = seg["n"]; u = k / max(n - 1, 1)
    W, H = fx.CW(), fx.CH()
    rs = fx.RS

    # 1) source frame -> linear light
    src = load_src(seg, k)
    lin = fx.to_linear_u8(cv2.cvtColor(src, cv2.COLOR_BGR2RGB))

    # 2) camera: slow push + zoom-punch + decaying shake (all about the centre)
    z0, z1 = seg.get("zoom", [1.0, 1.0])
    z = lerp(z0, z1, ease_io(u)) * (1 + seg.get("punch", 0.0) * pulse(k / edl["fps"], 7.0))
    sx, sy = shake_vec(k, seg.get("shake", 0.0))
    if z != 1.0 or sx or sy:
        M = np.array([[z, 0, W / 2 - z * W / 2 + sx * rs], [0, z, H / 2 - z * H / 2 + sy * rs]], np.float32)
        lin = cv2.warpAffine(lin, M, (W, H), flags=cv2.INTER_LINEAR, borderMode=cv2.BORDER_REFLECT)

    rgb = lin.copy()

    # 3) auto star-glints on the real highlights, pulsing on the beat
    if seg.get("glints"):
        bp = beat_pulse(g, per_beat)
        for (x, y, v) in find_highlights(lin, 2):
            star(rgb, (x, y), (0.12 + 0.9 * bp) * min(1.0, (v - 0.5) * 2.0), size=110)

    # 4) light sweep (diagonal band, only lights up what is already bright)
    sw = seg.get("sweep")
    if sw:
        c0 = sw["at"]; width = sw.get("width", 70)
        pos = lerp(-150, 1250, ease_io(clamp((u - c0 + 0.25) / 0.5)))
        yy, xx = np.mgrid[0:H, 0:W].astype(np.float32)
        a = math.radians(sw.get("angle", 65))
        band = np.exp(-((((xx / rs) * math.cos(a) + (yy / rs) * math.sin(a)) - pos) / width) ** 2)
        lum = np.clip(lin.max(2), 0, 1.2)
        rgb += (band * (0.2 + lum) * sw.get("gain", 0.5))[..., None] * np.array([0.75, 1.0, 0.9], np.float32)

    # 5) impact flash (multiplicative so blacks stay black)
    fl = 0.0
    if k == 0 or True:
        fl += seg.get("flash", 0.0) * math.exp(-k / 1.7)
        for (uf, amp) in seg.get("flash_at", []):
            kf = uf * (n - 1)
            if k >= kf: fl += amp * math.exp(-(k - kf) / 1.7)
    if fl > 0.004:
        rgb = rgb * (1.0 + fl * 2.6) + fl * np.array([0.12, 0.45, 0.30], np.float32) * 0.9

    # 6) crystal-shard wipe over the cut
    rgb = shard_wipe(rgb, g)

    # 7) whip / zoom blur / RGB split around every cut
    wp = whip_for(g)
    if wp:
        rgb = fx.motion_blur(rgb, wp["length"], wp["angle"])
        rgb = fx.radial_zoom_blur(rgb, wp["zoom"], steps=6)

    # 8) bloom + streaks + chroma + vignette
    rgb = fx.bloom(rgb, thresh=0.8, gain=gr.get("bloom", 0.55) * 0.7)
    rgb = fx.anamorphic(rgb, thresh=0.95, gain=0.22, length=300)
    rgb = fx.chroma(rgb, 0.0016 + (0.010 * (wp["length"] / 130) if wp else 0) + 0.012 * pulse(k / edl["fps"], 10.0) * (1 if seg.get("impact") else 0))
    rgb = rgb * fx.vignette(H, W, gr.get("vignette", 0.5), 2.2)
    dim = seg.get("dim")
    if dim:
        rgb = rgb * lerp(dim[0], dim[1], smooth(0, 12, k))

    # 9) title lock-up on the final segment
    if seg.get("title"):
        rgb = title_overlay(rgb, k, edl["title"])

    out = fx.grade(rgb * gr.get("exposure", 1.0), contrast=gr.get("contrast", 1.1), teal=gr.get("teal", 0.1), sat=gr.get("sat", 1.05))
    out = fx.grain(out, gr.get("grain", 0.016), seed=g)
    fo = seg.get("fade_out", 0)
    if fo and k >= n - fo:
        out = out * (1 - (k - (n - fo) + 1) / fo)
    return out


def worker(args):
    a, b, outdir = args
    fx.set_scale(G["scale"])
    for g in range(a, b):
        im = render_frame(g)
        cv2.imwrite(os.path.join(outdir, f"f_{g:05d}.png"), cv2.cvtColor((im * 255 + 0.5).astype(np.uint8), cv2.COLOR_RGB2BGR))
    return b - a


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--edl", required=True); ap.add_argument("--clips", required=True)
    ap.add_argument("--audio"); ap.add_argument("--out"); ap.add_argument("--scale", type=float, default=1.0)
    ap.add_argument("--workers", type=int, default=4); ap.add_argument("--check", action="store_true")
    ap.add_argument("--frames", help="a:b render only this frame range (debug); skips encode")
    a = ap.parse_args()
    edl = load_edl(a.edl)
    if a.check: sys.exit(check(edl, a.clips))
    rc = check(edl, a.clips)
    if rc: sys.exit(rc)
    fx.set_scale(a.scale)
    tl, total, per_beat = build_timeline(edl)
    tmp = tempfile.mkdtemp(prefix="finish_")
    try:
        t = time.time()
        meta = extract(edl, tl, a.clips, os.path.join(tmp, "src"))
        print(f"extracted source ranges in {time.time() - t:.0f}s", flush=True)
        G.update(edl=edl, tl=tl, meta=meta, per_beat=per_beat, total=total, scale=a.scale)
        fr = os.path.join(tmp, "frames"); os.makedirs(fr)
        f0, f1 = (0, total) if not a.frames else map(int, a.frames.split(":"))
        chunk = 6
        jobs = [(s, min(s + chunk, f1), fr) for s in range(f0, f1, chunk)]
        done = 0; t = time.time()
        with mp.Pool(a.workers) as pool:
            for nfr in pool.imap_unordered(worker, jobs):
                done += nfr
                if done % 24 < chunk: print(f"  rendered {done}/{f1 - f0} frames ({time.time() - t:.0f}s)", flush=True)
        if a.frames:
            dst = os.path.dirname(os.path.abspath(a.out)) if a.out else "."
            os.makedirs(dst, exist_ok=True)
            for f in sorted(os.listdir(fr)): shutil.copy(os.path.join(fr, f), dst)
            print("debug frames copied to", dst); return
        os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
        cmd = ["ffmpeg", "-v", "error", "-y", "-framerate", str(edl["fps"]), "-i", os.path.join(fr, "f_%05d.png")]
        if a.audio and os.path.exists(a.audio): cmd += ["-i", a.audio]
        cmd += ["-vf", "scale=out_color_matrix=bt709:out_range=tv,format=yuv420p", "-c:v", "libx264", "-preset", "slow", "-crf", "17",
                "-profile:v", "high", "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709"]
        if a.audio and os.path.exists(a.audio): cmd += ["-c:a", "aac", "-b:a", "256k", "-ar", "44100", "-shortest"]
        cmd += ["-movflags", "+faststart", a.out]
        subprocess.run(cmd, check=True)
        print("wrote", a.out, f"({total} frames, {total / edl['fps']:.1f}s)")
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


if __name__ == "__main__":
    main()
