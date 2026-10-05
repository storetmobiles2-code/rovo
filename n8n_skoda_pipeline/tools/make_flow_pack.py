#!/usr/bin/env python3
"""Build a 'Flow pack': everything you need to make the shots in Google Flow / the Gemini app with a Google AI Pro subscription
(no API key, no API billing). Reads config/shots.json + your photos; writes start frames (real-photo crops via ffmpeg), the
original photos for the AI-scene shots, and one markdown checklist with the exact prompt for every shot.

  python3 tools/make_flow_pack.py --inputs data/inputs --shots config/shots.json --out flow_pack
"""
import argparse, json, os, shutil, subprocess


def main():
    ap = argparse.ArgumentParser(); ap.add_argument("--inputs", required=True); ap.add_argument("--shots", required=True); ap.add_argument("--out", required=True)
    a = ap.parse_args(); cfg = json.load(open(a.shots)); V = cfg["veo"]
    os.makedirs(os.path.join(a.out, "start_frames"), exist_ok=True); os.makedirs(os.path.join(a.out, "photos"), exist_ok=True)
    for f in ("front.jpg", "side.jpg", "rear34.jpg"):
        if os.path.exists(os.path.join(a.inputs, f)): shutil.copy(os.path.join(a.inputs, f), os.path.join(a.out, "photos", f))
    md = [f"# Flow pack — {cfg['meta']['project']}", "",
          "Make **9 clips** (+1 optional) in Google Flow with your Google AI Pro subscription. Settings for every clip: **Veo (Fast or Quality), vertical 9:16, 8 seconds**.",
          "Save each result as an MP4 named exactly like the **file name** shown for the shot (e.g. `S03_plate_macro.mp4`) and send me all of them.", "",
          "Menu names in Flow change from time to time — look for *Frames to Video* (start-frame + prompt) and *Text to Video*.", "",
          f"**Identity rule (add to every prompt that shows the car):** {cfg['identity_rule']}", "", f"**Negative prompt (if Flow has the field):** {cfg['negative_prompt']}", ""]
    for i, s in enumerate(cfg["shots"], 1):
        md += [f"## {i}. `{s['id']}.mp4` — {s['title']}"]
        if s["mode"] == "crop_only":
            out = os.path.join(a.out, "start_frames", s["id"] + ".png"); c = s["crop"]
            vf = (f"crop=w='trunc(ih*{c['h']}*9/16/2)*2':h='trunc(ih*{c['h']}/2)*2':x='max(0,min(iw-out_w,{c['cx']}*iw-out_w/2))':"
                  f"y='max(0,min(ih-out_h,{c['cy']}*ih-out_h/2))',scale=1080:1920:flags=lanczos,unsharp=5:5:0.8")
            subprocess.run(["ffmpeg", "-v", "error", "-y", "-i", os.path.join(a.inputs, s["source"]), "-vf", vf, "-frames:v", "1", out], check=True)
            md += [f"**Start frame:** `start_frames/{s['id']}.png` (a crop of your real photo — upload it as the first frame; real plate/lamp/wheel pixels).", ""]
        elif s["mode"] == "ai_scene":
            md += [f"**Step A — make the start frame** in the Gemini app (image generation) or Flow's image tool: upload `photos/{s['source']}`, then paste:", "",
                   f"> {s['still_prompt']} {cfg['identity_rule']}", "",
                   f"Check it: car identical, plate readable. Save it as `start_frames/{s['id']}.png` and use it as the first frame below.", ""]
        else:
            md += ["**No start frame** (text-to-video).", ""]
        md += ["**Video prompt:**", "", f"> {s['video_prompt']}", ""]
    md += ["## Then", "Send me the MP4s. I run the beat-synced edit, check it against your reference with the validation gate, and iterate."]
    open(os.path.join(a.out, "FLOW_PACK.md"), "w").write("\n".join(md))
    print("wrote", a.out, "—", len(cfg["shots"]), "shots")


if __name__ == "__main__":
    main()
