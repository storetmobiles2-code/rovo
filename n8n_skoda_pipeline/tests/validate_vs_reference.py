#!/usr/bin/env python3
"""Validation gate: does a candidate edit match the REFERENCE video's craft?

  python3 tests/validate_vs_reference.py --ref reference.mp4 --ref-window 16.0:29.2 --cand out/final.mp4 [--sheet report.jpg] [--json report.json]

Measures the same things on the reference's *edit portion* and on the candidate, then applies pass/fail gates (tolerances below).
Automatic gates cover format, cut rhythm, transition energy, colour/mood and audio. What no metric can judge — real 3-D camera
motion, the car staying identical, plate legibility — is covered by the side-by-side contact sheet this script writes
(top row = reference beats, bottom row = candidate beats) and by the checklist printed at the end.
"""
import argparse, json, math, os, re, subprocess, sys, tempfile
import cv2, numpy as np


def sh(cmd):
    return subprocess.run(cmd, capture_output=True, text=True)


def window_clip(src, a, b, dst):
    sh(["ffmpeg", "-v", "error", "-y", "-ss", f"{a}", "-to", f"{b}", "-i", src, "-c:v", "libx264", "-preset", "ultrafast", "-crf", "18",
        "-c:a", "aac", "-b:a", "192k", dst])


def probe(p):
    j = json.loads(sh(["ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height,r_frame_rate,nb_frames",
                       "-show_entries", "format=duration", "-of", "json", p]).stdout)
    st = j["streams"][0]; n, d = st["r_frame_rate"].split("/")
    return dict(w=st["width"], h=st["height"], fps=float(n) / float(d), dur=float(j["format"]["duration"]))


def scene_cuts(p, thr=0.30):
    r = sh(["ffmpeg", "-nostats", "-i", p, "-vf", f"select='gt(scene,{thr})',showinfo", "-an", "-f", "null", "-"])
    return [float(m) for m in re.findall(r"pts_time:([0-9.]+)", r.stderr)]


def frames_small(p, fps=30, w=135, h=240):
    cap = cv2.VideoCapture(p); src_fps = cap.get(cv2.CAP_PROP_FPS) or 30
    out, i, step = [], 0, max(src_fps / fps, 1e-6); nxt = 0.0
    while True:
        ok, f = cap.read()
        if not ok: break
        if i >= nxt:
            out.append(cv2.resize(f, (w, h), interpolation=cv2.INTER_AREA)); nxt += step
        i += 1
    return np.array(out)


def audio_stats(p):
    with tempfile.TemporaryDirectory() as d:
        wav = os.path.join(d, "a.wav")
        sh(["ffmpeg", "-v", "error", "-y", "-i", p, "-vn", "-ac", "1", "-ar", "22050", wav])
        out = {}
        try:
            import librosa
            y, sr = librosa.load(wav, sr=22050)
            tempo = float(np.atleast_1d(librosa.beat.beat_track(y=y, sr=sr)[0])[0])
            on = librosa.onset.onset_detect(y=y, sr=sr, units="time")
            out.update(tempo=tempo, onsets_per_s=len(on) / max(len(y) / sr, 1e-6))
        except Exception as e:
            out["audio_note"] = f"librosa unavailable: {e}"
        r = sh(["ffmpeg", "-nostats", "-i", wav, "-af", "ebur128", "-f", "null", "-"])
        m = re.findall(r"I:\s+(-?[0-9.]+) LUFS", r.stderr)
        if m: out["lufs"] = float(m[-1])
        return out


def measure(p):
    pr = probe(p)
    fr = frames_small(p)
    hsv = np.array([cv2.cvtColor(f, cv2.COLOR_BGR2HSV) for f in fr])
    H = hsv[..., 0].astype(np.float32) * 2          # degrees
    Sa = hsv[..., 1].astype(np.float32) / 255
    V = hsv[..., 2].astype(np.float32) / 255
    sat_mask = (Sa > 0.25) & (V > 0.12)
    green_teal = ((H > 70) & (H < 190)) & sat_mask
    luma = np.array([cv2.cvtColor(f, cv2.COLOR_BGR2GRAY).mean() / 255 for f in fr])
    cuts = scene_cuts(p)
    # transition energy: sharpness (laplacian var) at cut frames vs typical frame — whips blur the cut frames
    lap = np.array([cv2.Laplacian(cv2.cvtColor(f, cv2.COLOR_BGR2GRAY), cv2.CV_32F).var() for f in fr])
    idx = [min(int(c * 30), len(lap) - 1) for c in cuts]
    near = [lap[max(i - 1, 0):i + 2].min() for i in idx]
    med = float(np.median(lap)) + 1e-6
    blur_ratio = float(np.median(near) / med) if near else 1.0
    iv = np.diff([0.0] + cuts + [pr["dur"]])
    res = dict(
        duration=round(pr["dur"], 2), aspect=round(pr["w"] / pr["h"], 4), size=f"{pr['w']}x{pr['h']}",
        cuts=len(cuts), cuts_per_s=round(len(cuts) / pr["dur"], 3), median_shot_s=round(float(np.median(iv)), 2),
        cut_blur_ratio=round(blur_ratio, 3),                       # <1 = cut frames are blurrier than normal frames (whip transitions)
        mean_luma=round(float(luma.mean()), 3), dark_frames=round(float((luma < 0.18).mean()), 3),
        green_teal_share=round(float(green_teal.sum() / max(sat_mask.sum(), 1)), 3),
        mean_sat=round(float(Sa.mean()), 3), contrast=round(float(luma.std()), 3),
    )
    res.update(audio_stats(p))
    res["_cut_times"] = [round(c, 2) for c in cuts]
    return res


GATES = [  # (key, kind, tolerance, description)
    ("aspect", "abs", 0.01, "9:16 vertical"),
    ("duration", "rel", 0.30, "length within ±30% of the reference edit"),
    ("cuts_per_s", "rel", 0.45, "cut density within ±45%"),
    ("median_shot_s", "rel", 0.50, "median shot length within ±50%"),
    ("cut_blur_ratio", "max", 0.85, "cuts are hidden by whip/motion blur (cut-frame sharpness ≤ 0.85× normal)"),
    ("green_teal_share", "rel", 0.35, "same emerald/teal colour world (±35%)"),
    ("dark_frames", "abs", 0.25, "same dark, high-contrast mood (±0.25 of frames)"),
    ("contrast", "rel", 0.45, "luma contrast within ±45%"),
    ("tempo", "abs", 4.0, "tempo within ±4 BPM"),
    ("onsets_per_s", "rel", 0.45, "rhythmic density within ±45%"),
    ("lufs", "abs", 4.0, "loudness within ±4 LU"),
]


def gate(ref, cand):
    rows, ok_all = [], True
    for key, kind, tol, desc in GATES:
        if key not in ref or key not in cand: continue
        r, c = ref[key], cand[key]
        if kind == "abs": ok = abs(c - r) <= tol
        elif kind == "rel": ok = abs(c - r) <= tol * abs(r)
        else: ok = c <= tol
        ok_all &= ok
        rows.append(dict(gate=key, desc=desc, ref=r, cand=c, ok=bool(ok)))
    return rows, ok_all


def sheet(ref_p, cand_p, ref_win, out, n=14):
    def grab(p, times):
        cap = cv2.VideoCapture(p); ims = []
        for t in times:
            cap.set(cv2.CAP_PROP_POS_MSEC, t * 1000); ok, f = cap.read()
            f = f if ok else np.zeros((1920, 1080, 3), np.uint8)
            ims.append(cv2.resize(f, (200, int(200 * f.shape[0] / f.shape[1]))))
        return np.hstack(ims)
    rp = probe(ref_p); cp = probe(cand_p)
    rt = np.linspace(ref_win[0] + 0.1, ref_win[1] - 0.2, n); ct = np.linspace(0.1, cp["dur"] - 0.2, n)
    a, b = grab(ref_p, rt), grab(cand_p, ct)
    h = min(a.shape[0], b.shape[0]); a, b = a[:h], b[:h]
    lab = lambda im, t: cv2.putText(im.copy(), t, (6, 24), cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 255, 255), 2)
    cv2.imwrite(out, np.vstack([lab(a, "REFERENCE"), lab(b, "CANDIDATE")]))


CHECKLIST = [
    "Opener: car on black mirror floor, crystals growing over/around it (first ~2 s)",
    "Cave reveal lands on the first big beat with an impact",
    "Real camera motion around the car (orbit / push-in / tracking) — not a sliding sticker",
    "Macro cuts: number plate, headlamp, tail-lamp, wheel/door",
    "Foreground crystals wipe across the lens during cuts",
    "Side-profile hero: crystal crown behind the car, mirror reflection on the floor",
    "Closer: macro on the car badge that turns into a faceted crystal gem (the reference's last beat)",
    "Ends on a push-in / defocus into the lock-up",
    "Car identical to the photos: white paint, wheels, grille, plate TS 09 EF 6141 legible",
]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--ref", required=True); ap.add_argument("--ref-window", default="16.0:29.2")
    ap.add_argument("--cand", required=True); ap.add_argument("--sheet"); ap.add_argument("--json")
    a = ap.parse_args()
    w0, w1 = map(float, a.ref_window.split(":"))
    with tempfile.TemporaryDirectory() as d:
        rw = os.path.join(d, "ref_edit.mp4"); window_clip(a.ref, w0, w1, rw)
        ref, cand = measure(rw), measure(a.cand)
        if a.sheet: sheet(a.ref, a.cand, (w0, w1), a.sheet)
    rows, ok = gate(ref, cand)
    print(f"{'GATE':18s} {'REF':>9s} {'CAND':>9s}  RESULT  (what it checks)")
    for r in rows:
        print(f"{r['gate']:18s} {r['ref']:>9} {r['cand']:>9}  {'PASS' if r['ok'] else 'FAIL'}    {r['desc']}")
    print("\nOVERALL AUTOMATIC GATES:", "PASS" if ok else "FAIL")
    print("\nHuman/visual checklist (use the contact sheet):")
    for c in CHECKLIST: print("  [ ]", c)
    if a.json: json.dump(dict(reference=ref, candidate=cand, gates=rows, pass_all=bool(ok)), open(a.json, "w"), indent=1)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
