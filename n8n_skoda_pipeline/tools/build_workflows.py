#!/usr/bin/env python3
"""Generates the three importable n8n workflows (workflows/*.json).
Single source of truth: edit here, run `python3 tools/build_workflows.py`, re-import in n8n."""
import json, uuid, os

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "workflows")
NS = uuid.UUID("5b0b3a54-2d3e-4b8e-9a53-0a7f3a1f3d11")
CRED = {"httpHeaderAuth": {"id": "gemini_key", "name": "Gemini API Key"}}


class WF:
    def __init__(self, name):
        self.name = name
        self.nodes = []
        self.conns = {}

    def add(self, name, type_, ver, pos, params=None, cred=None, **extra):
        n = {"id": str(uuid.uuid5(NS, self.name + name)), "name": name, "type": type_, "typeVersion": ver,
             "position": list(pos), "parameters": params or {}}
        if cred: n["credentials"] = cred
        n.update(extra)
        self.nodes.append(n)
        return name

    def link(self, a, b, out=0, inp=0):
        c = self.conns.setdefault(a, {"main": []})["main"]
        while len(c) <= out: c.append([])
        c[out].append({"node": b, "type": "main", "index": inp})

    def chain(self, *names):
        for a, b in zip(names, names[1:]): self.link(a, b)

    def json(self):
        return {"id": uuid.uuid5(NS, "wf" + self.name).hex[:16], "name": self.name, "nodes": self.nodes, "connections": self.conns,
                "settings": {"executionOrder": "v1", "executionTimeout": 7200, "saveManualExecutions": True},
                "pinData": {}, "active": False, "tags": []}

    def save(self, fname):
        with open(os.path.join(OUT, fname), "w") as f:
            json.dump(self.json(), f, indent=2, ensure_ascii=False)
        print("wrote", fname, len(self.nodes), "nodes")


def code(js): return {"jsCode": js.strip("\n")}


def if_(left, op_type, operation, right=None):
    cond = {"id": str(uuid.uuid4()), "leftValue": left, "operator": {"type": op_type, "operation": operation}}
    if right is not None: cond["rightValue"] = right
    if op_type == "boolean" or operation in ("exists",): cond["operator"]["singleValue"] = True
    return {"conditions": {"options": {"caseSensitive": True, "leftValue": "", "typeValidation": "loose", "version": 2},
                           "conditions": [cond], "combinator": "and"}, "options": {}}


def http(method, url, body=None, file=False, timeout=240000, retry=True):
    p = {"method": method, "url": url, "authentication": "genericCredentialType", "genericAuthType": "httpHeaderAuth", "options": {"timeout": timeout}}
    if body is not None:
        p.update({"sendBody": True, "specifyBody": "json", "jsonBody": body})
    if file:
        p["options"]["response"] = {"response": {"responseFormat": "file", "outputPropertyName": "data"}}
    extra = {"retryOnFail": True, "maxTries": 3, "waitBetweenTries": 6000} if retry else {}
    return p, extra


def note(wf, text, pos, w=520, h=300, color=4):
    wf.add("Notes " + str(len(wf.nodes)), "n8n-nodes-base.stickyNote", 1, pos, {"content": text, "width": w, "height": h, "color": color})


READ = "n8n-nodes-base.readWriteFile"
CFG_READ = lambda: {"operation": "read", "fileSelector": "={{ $('Config').first().json.data_dir }}/config/shots.json", "options": {}}
PARSE = lambda: {"operation": "fromJson", "binaryPropertyName": "data", "destinationKey": "cfg", "options": {}}
CMD = "n8n-nodes-base.executeCommand"
CODE = "n8n-nodes-base.code"

CONFIG_JS = """
// ✏️  EDIT ME — the only place where paths and the API address live.
return [{ json: {
  // Real Google API. For a free rehearsal against the bundled mock server use:
  //   http://host.docker.internal:8099/v1beta      (n8n in Docker)   or   http://127.0.0.1:8099/v1beta   (n8n on the host)
  api_base: 'https://generativelanguage.googleapis.com/v1beta',
  data_dir: '/data',          // folder that contains inputs/, config/, stills/, clips/, scripts/, out/
%EXTRA%
}}];
"""

# ===================================================================================== 01
w = WF("01 · Skoda — Start frames (photos → cave stills)")
note(w, "## 01 · START FRAMES\n**Photos in** `/data/inputs/` (front.jpg, side.jpg, rear34.jpg)\n\n1. Reads `config/shots.json`\n2. Preflight: checks your API key + model names\n3. `ai_scene` shots → Gemini image model puts the **real car** into the crystal cave\n4. `crop_only` shots → pure ffmpeg crops of your real photo (keeps the real plate/lamp/wheel pixels)\n5. Writes `/data/stills/*.png` + a contact sheet\n\n**Cost:** ≈ $0.07 per generated still.\n**Next:** look at `stills/_contact_sheet.jpg`, copy the ones you like into `stills/approved/`, then run workflow 02.", (-200, -330), 560, 330)
w.add("Manual Trigger", "n8n-nodes-base.manualTrigger", 1, (0, 0))
w.add("Config", CODE, 2, (220, 0), code(CONFIG_JS.replace("%EXTRA%", "  auto_approve: false,         // true = copy every still into stills/approved/ automatically")))
w.add("Read shots.json", READ, 1, (440, 0), CFG_READ())
w.add("Parse shots", "n8n-nodes-base.extractFromFile", 1.1, (660, 0), PARSE())
p, e = http("GET", "={{ $('Config').first().json.api_base }}/models?pageSize=1000", retry=True)
w.add("Preflight: list models", "n8n-nodes-base.httpRequest", 4.2, (880, 0), p, CRED, **e)
w.add("Check models", CODE, 2, (1100, 0), code("""
// Stops early (cheaply) if the key is bad, and warns if a configured model id is not available to this key.
const cfg = $('Config').first().json;
const S = $('Parse shots').first().json.cfg;
const names = ($input.first().json.models || []).map(m => String(m.name || '').replace(/^models\\//, ''));
const warnings = [];
for (const [label, id] of [['image model', S.image_model], ['video model', S.veo.model]]) {
  if (!names.includes(id)) {
    const sim = names.filter(n => n.split('-')[0] === id.split('-')[0]).slice(0, 10).join(', ');
    warnings.push(`${label} "${id}" is not in your key's model list. Similar: ${sim || '(none)'} — fix it in config/shots.json`);
  }
}
return [{ json: { ...cfg, warnings, models_visible: names.length } }];
"""))
w.add("Shots → items", CODE, 2, (1320, 0), code("""
const cfg = $('Check models').first().json;
const S = $('Parse shots').first().json.cfg;
return S.shots.filter(s => s.mode !== 'text_only').map(s => ({ json: {
  ...s, data_dir: cfg.data_dir, api_base: cfg.api_base, image_model: S.image_model,
  identity_rule: S.identity_rule, car: S.meta.car, plate: S.meta.plate,
}}));
"""))
w.add("AI or crop?", "n8n-nodes-base.if", 2.2, (1540, 0), if_("={{ $json.mode }}", "string", "equals", "ai_scene"))
# ---- AI branch
w.add("Read photo", READ, 1, (1780, -140), {"operation": "read", "fileSelector": "={{ $json.data_dir }}/inputs/{{ $json.source }}", "options": {}})
w.add("Build edit request", CODE, 2, (2000, -140), code("""
const items = $input.all();
const out = [];
for (let i = 0; i < items.length; i++) {
  const meta = $('AI or crop?').itemMatching(i).json;
  const buf = await this.helpers.getBinaryDataBuffer(i, 'data');
  const mime = items[i].binary.data.mimeType || 'image/jpeg';
  const prompt = `${meta.still_prompt}\\n\\n${meta.identity_rule}\\nThe car is a ${meta.car}.`;
  out.push({ json: { ...meta, body: {
    contents: [{ role: 'user', parts: [ { text: prompt }, { inlineData: { mimeType: mime, data: buf.toString('base64') } } ] }],
    generationConfig: { responseModalities: ['IMAGE'], imageConfig: { aspectRatio: '9:16' } },
  }}});
}
return out;
"""))
p, e = http("POST", "={{ $json.api_base }}/models/{{ $json.image_model }}:generateContent", body="={{ JSON.stringify($json.body) }}", timeout=300000)
w.add("Gemini: edit image", "n8n-nodes-base.httpRequest", 4.2, (2220, -140), p, CRED, **e)
w.add("Extract image", CODE, 2, (2440, -140), code("""
const items = $input.all();
const out = [];
for (let i = 0; i < items.length; i++) {
  const meta = $('Build edit request').itemMatching(i).json;
  const res = items[i].json;
  const parts = (res.candidates?.[0]?.content?.parts) || [];
  const img = parts.map(p => p.inlineData || p.inline_data).find(Boolean);
  if (!img?.data) {
    const why = res.promptFeedback?.blockReason || res.candidates?.[0]?.finishReason || 'no image part';
    const txt = parts.map(p => p.text).filter(Boolean).join(' ').slice(0, 300);
    throw new Error(`Shot ${meta.id}: Gemini returned no image (${why}). ${txt}`);
  }
  const mime = img.mimeType || img.mime_type || 'image/png';
  const bin = await this.helpers.prepareBinaryData(Buffer.from(img.data, 'base64'), `${meta.id}.png`, mime);
  const { body, ...rest } = meta;
  out.push({ json: rest, binary: { data: bin } });
}
return out;
"""))
w.add("Write still", READ, 1, (2660, -140), {"operation": "write", "fileName": "={{ $json.data_dir }}/stills/{{ $json.id }}.png", "dataPropertyName": "data", "options": {}})
# ---- crop branch
CROP_CMD = ("=mkdir -p {{ $json.data_dir }}/stills && ffmpeg -v error -y -i {{ $json.data_dir }}/inputs/{{ $json.source }} "
            "-vf \"crop=w='trunc(ih*{{ $json.crop.h }}*9/16/2)*2':h='trunc(ih*{{ $json.crop.h }}/2)*2':"
            "x='max(0,min(iw-out_w,{{ $json.crop.cx }}*iw-out_w/2))':y='max(0,min(ih-out_h,{{ $json.crop.cy }}*ih-out_h/2))',"
            "scale=1080:1920:flags=lanczos,unsharp=5:5:0.8\" -frames:v 1 {{ $json.data_dir }}/stills/{{ $json.id }}.png "
            "&& echo cropped {{ $json.id }}")
w.add("Crop real photo (ffmpeg)", CMD, 1, (1780, 140), {"executeOnce": False, "command": CROP_CMD})
w.add("Merge branches", "n8n-nodes-base.merge", 3, (2900, 0), {"mode": "append"})
w.add("Contact sheet", CMD, 1, (3120, 0), {"executeOnce": True, "command": (
    "=cd {{ $('Config').first().json.data_dir }}/stills && ffmpeg -v error -y -pattern_type glob -i 'S*.png' "
    "-vf \"scale=270:-1,tile=8x2:padding=6:color=black\" -frames:v 1 _contact_sheet.jpg && "
    "if [ \"{{ $('Config').first().json.auto_approve }}\" = \"true\" ]; then mkdir -p approved && cp -f S*.png approved/; fi && ls -1 S*.png")})
w.add("Next steps", CODE, 2, (3340, 0), code("""
const cfg = $('Check models').first().json;
const files = String($input.first().json.stdout || '').trim().split('\\n').filter(Boolean);
return [{ json: {
  stills_created: files.length, files,
  warnings: cfg.warnings,
  next: [
    `1. Open ${cfg.data_dir}/stills/_contact_sheet.jpg and check every still (car identical? plate right? cave look?).`,
    `2. Copy the good ones into ${cfg.data_dir}/stills/approved/ (keep the file names). Re-run this workflow to redo a bad one.`,
    '3. Run workflow "02 · Animate" (it starts in DRY-RUN mode and shows the cost first).',
  ],
}}];
"""))
w.chain("Manual Trigger", "Config", "Read shots.json", "Parse shots", "Preflight: list models", "Check models", "Shots → items", "AI or crop?")
w.link("AI or crop?", "Read photo", 0); w.link("AI or crop?", "Crop real photo (ffmpeg)", 1)
w.chain("Read photo", "Build edit request", "Gemini: edit image", "Extract image", "Write still")
w.link("Write still", "Merge branches", 0, 0); w.link("Crop real photo (ffmpeg)", "Merge branches", 0, 1)
w.chain("Merge branches", "Contact sheet", "Next steps")
w.save("01_start_frames.json")

# ===================================================================================== 02
w = WF("02 · Skoda — Animate approved stills (Veo)")
note(w, "## 02 · ANIMATE\nTakes every PNG in `/data/stills/approved/` and turns it into an 8 s clip with **Veo** (image-to-video, 9:16, 1080p).\n\n⚠️ **Starts in DRY-RUN** — it only prints the cost. Set `dry_run: false` in *Config* to spend money.\n\nAll clips are generated **in parallel**; the loop below polls every 20 s until each one is done, downloads it to `/data/clips/<shot>.mp4` and writes `<shot>.FAILED.txt` if a clip is blocked/failed (the rest continue).\n\nVeo keeps results only **2 days** — download happens automatically.\n\nTip: set `only: ['S01_reveal_orbit']` to test with one clip first (≈ $1).", (-200, -360), 600, 330)
w.add("Manual Trigger", "n8n-nodes-base.manualTrigger", 1, (0, 0))
w.add("Config", CODE, 2, (220, 0), code(CONFIG_JS.replace("%EXTRA%", "  dry_run: true,               // ← set false to actually generate (costs money)\n  only: [],                   // e.g. ['S01_reveal_orbit'] to run just some shots")))
w.add("Read shots.json", READ, 1, (440, 0), CFG_READ())
w.add("Parse shots", "n8n-nodes-base.extractFromFile", 1.1, (660, 0), PARSE())
w.add("Read approved stills", READ, 1, (880, 0), {"operation": "read", "fileSelector": "={{ $('Config').first().json.data_dir }}/stills/approved/*.png", "options": {}})
w.add("Plan shots + cost", CODE, 2, (1100, 0), code("""
const cfg = $('Config').first().json;
const S = $('Parse shots').first().json.cfg;
const V = S.veo;
const files = $input.all();
const byId = Object.fromEntries(S.shots.map(s => [s.id, s]));
const want = id => !(cfg.only && cfg.only.length) || cfg.only.includes(id);
const plan = [];
for (let i = 0; i < files.length; i++) {
  const id = String(files[i].binary?.data?.fileName || '').replace(/\\.[^.]+$/, '');
  const shot = byId[id];
  if (!shot || !want(id)) continue;
  const buf = await this.helpers.getBinaryDataBuffer(i, 'data');
  plan.push({ shot, image: { mimeType: files[i].binary.data.mimeType || 'image/png', data: buf.toString('base64') } });
}
for (const s of S.shots.filter(s => s.mode === 'text_only' && want(s.id))) plan.push({ shot: s });
if (!plan.length) throw new Error(`Nothing to animate: no approved stills matched a shot id in ${cfg.data_dir}/stills/approved/`);
const cost = plan.length * V.duration_seconds * V.price_per_second_usd;
if (cfg.dry_run) {
  return [{ json: { is_dry_run: true, shots: plan.map(p => p.shot.id), clips: plan.length, seconds_each: V.duration_seconds,
    model: V.model, resolution: V.resolution, est_cost_usd: Math.round(cost * 100) / 100,
    message: 'DRY RUN — nothing was sent to Google. Set dry_run to false in the Config node to generate.' } }];
}
return plan.map(p => {
  const inst = { prompt: p.shot.video_prompt };
  if (p.image) inst.image = { inlineData: p.image };
  const parameters = { aspectRatio: V.aspect_ratio, resolution: V.resolution, durationSeconds: String(V.duration_seconds) };
  if (V.send_negative_prompt && S.negative_prompt) parameters.negativePrompt = S.negative_prompt;
  return { json: { id: p.shot.id, title: p.shot.title, api_base: cfg.api_base, data_dir: cfg.data_dir, veo_model: V.model,
    poll_s: V.poll_every_seconds, timeout_min: V.timeout_minutes, est_cost_usd: V.duration_seconds * V.price_per_second_usd,
    request: { instances: [inst], parameters } } };
});
"""))
w.add("Dry run?", "n8n-nodes-base.if", 2.2, (1320, 0), if_("={{ $json.is_dry_run === true }}", "boolean", "true"))
w.add("DRY RUN — nothing sent (read the cost here)", "n8n-nodes-base.noOp", 1, (1560, -120))
p, e = http("POST", "={{ $json.api_base }}/models/{{ $json.veo_model }}:predictLongRunning", body="={{ JSON.stringify($json.request) }}", timeout=300000)
w.add("Veo: start", "n8n-nodes-base.httpRequest", 4.2, (1560, 120), p, CRED, **e)
w.add("Init poll", CODE, 2, (1780, 120), code("""
const items = $input.all();
return items.map((it, i) => {
  const m = $('Dry run?').itemMatching(i).json;
  if (!it.json.name) throw new Error(`Shot ${m.id}: Veo did not return an operation name: ${JSON.stringify(it.json).slice(0, 300)}`);
  return { json: { id: m.id, title: m.title, op_name: it.json.name, started: Date.now(), api_base: m.api_base, data_dir: m.data_dir, poll_s: m.poll_s, timeout_min: m.timeout_min } };
});
"""))
w.add("Wait", "n8n-nodes-base.wait", 1.1, (2000, 120), {"resume": "timeInterval", "amount": "={{ $json.poll_s }}", "unit": "seconds"})
p, e = http("GET", "={{ $json.api_base }}/{{ $json.op_name }}", timeout=120000)
w.add("Poll operation", "n8n-nodes-base.httpRequest", 4.2, (2220, 120), p, CRED, **e)
w.add("Evaluate", CODE, 2, (2440, 120), code("""
const init = $('Init poll').all();
return $input.all().map(it => {
  const r = it.json;
  const m = (init.find(x => x.json.op_name === r.name) || {}).json || {};
  const elapsed = (Date.now() - (m.started || Date.now())) / 60000;
  let status = 'pending', error = null, video_uri = null;
  if (r.error) { status = 'failed'; error = JSON.stringify(r.error); }
  else if (r.done) {
    const g = r.response?.generateVideoResponse;
    video_uri = g?.generatedSamples?.[0]?.video?.uri || r.response?.generatedVideos?.[0]?.video?.uri || null;
    if (video_uri) status = 'ok';
    else { status = 'failed'; error = 'No video returned (usually blocked by a safety filter). ' + JSON.stringify(g?.raiMediaFilteredReasons || r.response || {}).slice(0, 500); }
  } else if (elapsed > (m.timeout_min || 15)) { status = 'failed'; error = `Timed out after ${m.timeout_min} min`; }
  return { json: { ...m, status, error, video_uri, elapsed_min: Math.round(elapsed * 10) / 10 } };
});
"""))
w.add("Finished?", "n8n-nodes-base.if", 2.2, (2660, 120), if_("={{ $json.status }}", "string", "notEquals", "pending"))
w.add("Succeeded?", "n8n-nodes-base.if", 2.2, (2880, 40), if_("={{ $json.status }}", "string", "equals", "ok"))
p, e = http("GET", "={{ $json.video_uri }}", file=True, timeout=600000)
w.add("Download clip", "n8n-nodes-base.httpRequest", 4.2, (3100, -40), p, CRED, **e)
w.add("Name clip", CODE, 2, (3320, -40), code("""
const items = $input.all();
return items.map((it, i) => ({ json: { ...$('Succeeded?').itemMatching(i).json }, binary: it.binary }));
"""))
w.add("Write clip", READ, 1, (3540, -40), {"operation": "write", "fileName": "={{ $json.data_dir }}/clips/{{ $json.id }}.mp4", "dataPropertyName": "data", "options": {}})
w.add("Failure note", CODE, 2, (3100, 200), code("""
const out = [];
for (const it of $input.all()) {
  const j = it.json;
  const txt = `Shot ${j.id} FAILED after ${j.elapsed_min} min\\n${j.error}\\n\\nFix the prompt / still in shots.json and re-run workflow 02 with only: ['${j.id}'].\\n`;
  out.push({ json: j, binary: { data: await this.helpers.prepareBinaryData(Buffer.from(txt), `${j.id}.FAILED.txt`, 'text/plain') } });
}
return out;
"""))
w.add("Write failure note", READ, 1, (3320, 200), {"operation": "write", "fileName": "={{ $json.data_dir }}/clips/{{ $json.id }}.FAILED.txt", "dataPropertyName": "data", "options": {}})
w.chain("Manual Trigger", "Config", "Read shots.json", "Parse shots", "Read approved stills", "Plan shots + cost", "Dry run?")
w.link("Dry run?", "DRY RUN — nothing sent (read the cost here)", 0); w.link("Dry run?", "Veo: start", 1)
w.chain("Veo: start", "Init poll", "Wait", "Poll operation", "Evaluate", "Finished?")
w.link("Finished?", "Wait", 1)                       # still pending -> wait again
w.link("Finished?", "Succeeded?", 0)
w.link("Succeeded?", "Download clip", 0); w.link("Succeeded?", "Failure note", 1)
w.chain("Download clip", "Name clip", "Write clip")
w.chain("Failure note", "Write failure note")
w.save("02_animate_clips.json")

# ===================================================================================== 03
w = WF("03 · Skoda — Edit, motion design & sound (finish)")
note(w, "## 03 · FINISH\nRuns the **open-source finishing engine** (`scripts/finish.py` + `audio.py`):\n\n- beat-synced cut from `config/edl.json` (90 BPM, 1 beat = 20 frames)\n- speed ramps, whip-blur / zoom-punch transitions, impact flashes, crystal-shard wipes\n- auto star-glints locked to the brightest real highlights, light sweeps\n- unified emerald grade, bloom, anamorphic streaks, grain\n- kinetic title lock-up (ŠKODA / SUPERB)\n- synthesised score + whooshes + impacts cut to the same beat grid\n\nSet `preview: true` for a fast half-res draft.", (-200, -330), 560, 330)
w.add("Manual Trigger", "n8n-nodes-base.manualTrigger", 1, (0, 0))
w.add("Config", CODE, 2, (220, 0), code(CONFIG_JS.replace("%EXTRA%", "  preview: true,              // true = quick 540×960 draft, false = final 1080×1920\n  output_name: 'SKODA_SUPERB_CRYSTAL_CAVE_V02',\n  workers: 4,")))
w.add("Check clips", CMD, 1, (440, 0), {"executeOnce": True, "command":
    "=python3 {{ $json.data_dir }}/scripts/finish.py --check --edl {{ $json.data_dir }}/config/edl.json --clips {{ $json.data_dir }}/clips"})
w.add("Score (audio.py)", CMD, 1, (660, 0), {"executeOnce": True, "command":
    "=mkdir -p {{ $('Config').first().json.data_dir }}/audio && python3 {{ $('Config').first().json.data_dir }}/scripts/audio.py --edl {{ $('Config').first().json.data_dir }}/config/edl.json --out {{ $('Config').first().json.data_dir }}/audio/score.wav"})
w.add("Render (finish.py)", CMD, 1, (880, 0), {"executeOnce": True, "command":
    "=mkdir -p {{ $('Config').first().json.data_dir }}/out && python3 {{ $('Config').first().json.data_dir }}/scripts/finish.py --edl {{ $('Config').first().json.data_dir }}/config/edl.json "
    "--clips {{ $('Config').first().json.data_dir }}/clips --audio {{ $('Config').first().json.data_dir }}/audio/score.wav "
    "--out {{ $('Config').first().json.data_dir }}/out/{{ $('Config').first().json.output_name }}{{ $('Config').first().json.preview ? '_PREVIEW' : '' }}.mp4 "
    "--scale {{ $('Config').first().json.preview ? 0.5 : 1.0 }} --workers {{ $('Config').first().json.workers }}"})
w.add("Probe output", CMD, 1, (1100, 0), {"executeOnce": True, "command":
    "=ffprobe -v error -show_entries stream=codec_name,width,height,r_frame_rate,duration -of json {{ $('Config').first().json.data_dir }}/out/{{ $('Config').first().json.output_name }}{{ $('Config').first().json.preview ? '_PREVIEW' : '' }}.mp4"})
w.add("Report", CODE, 2, (1320, 0), code("""
const cfg = $('Config').first().json;
let probe = {}; try { probe = JSON.parse($input.first().json.stdout); } catch (e) {}
return [{ json: { done: true, file: `${cfg.data_dir}/out/${cfg.output_name}${cfg.preview ? '_PREVIEW' : ''}.mp4`, streams: probe.streams,
  hint: cfg.preview ? 'This is a half-res draft. Set preview:false in Config for the final 1080×1920 render.' : 'Final render complete.' } }];
"""))
w.chain("Manual Trigger", "Config", "Check clips", "Score (audio.py)", "Render (finish.py)", "Probe output", "Report")
w.save("03_edit_and_finish.json")
