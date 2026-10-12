"""NotebookLM + Claude bridge for the n8n YouTube workflow.

n8n has no NotebookLM node, so this small service does the two steps n8n
cannot: it turns a source document into a NotebookLM Video Overview (through the
unofficial notebooklm-py client) and asks Claude, guided by the
youtube-automation skill, for the YouTube title, description and tags.

Video generation takes 5-30 minutes, longer than an HTTP request should stay
open, so work runs as background jobs:

    POST /jobs                 upload a document, returns {"job_id": ...}
    GET  /jobs/{id}            status, error and (when done) metadata
    GET  /jobs/{id}/video      the finished MP4
    POST /jobs/{id}/published  record the YouTube video id after upload
    GET  /health               liveness and NotebookLM login status

Every route except /health needs the X-Bridge-Token header.
"""

from __future__ import annotations

import asyncio
import json
import logging
import mimetypes
import os
import re
import secrets
import shutil
import time
import uuid
from contextlib import asynccontextmanager
from pathlib import Path

import anthropic
from fastapi import Depends, FastAPI, File, Form, Header, HTTPException, UploadFile
from fastapi.responses import FileResponse
from notebooklm import NotebookLMClient
from notebooklm.types import VideoFormat, VideoStyle
from pydantic import BaseModel

log = logging.getLogger("bridge")
logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"), format="%(asctime)s %(levelname)s %(message)s")

BRIDGE_TOKEN = os.environ.get("BRIDGE_TOKEN", "")
DATA_DIR = Path(os.getenv("DATA_DIR", "/data"))
SKILL_PATH = Path(os.getenv("SKILL_PATH", "/skills/youtube-automation/SKILL.md"))
NOTEBOOKLM_STORAGE = os.getenv("NOTEBOOKLM_STORAGE") or None

CLAUDE_MODEL = os.getenv("CLAUDE_MODEL", "claude-opus-5-5")
CLAUDE_EFFORT = os.getenv("CLAUDE_EFFORT", "medium")
# "default" lets the API retry a declined request on a fallback model; "off" disables it.
CLAUDE_FALLBACKS = os.getenv("CLAUDE_FALLBACKS", "default")

VIDEO_FORMAT = os.getenv("VIDEO_FORMAT", "explainer").upper()
VIDEO_STYLE = os.getenv("VIDEO_STYLE", "auto").upper().replace("-", "_")
VIDEO_LANGUAGE = os.getenv("VIDEO_LANGUAGE", "en")
VIDEO_INSTRUCTIONS = os.getenv("VIDEO_INSTRUCTIONS", "")
DELETE_NOTEBOOK_AFTER = os.getenv("DELETE_NOTEBOOK_AFTER", "false").lower() == "true"
JOB_TIMEOUT_MINUTES = float(os.getenv("JOB_TIMEOUT_MINUTES", "60"))
KEEPALIVE_HOURS = float(os.getenv("KEEPALIVE_HOURS", "6"))

JOBS_DIR = DATA_DIR / "jobs"
TERMINAL = {"done", "error"}

BRIEF_PROMPT = (
    "Write a source brief for a YouTube editor who has not read these sources. "
    "Include: the document's title and author or organisation if stated, who it is for, "
    "a one-paragraph summary, 5-8 key points as a bulleted list, and any specific numbers, "
    "names or terms a viewer would search for. Plain text, no citations."
)

METADATA_SCHEMA = {
    "type": "object",
    "properties": {
        "title": {"type": "string"},
        "description": {"type": "string"},
        "tags": {"type": "array", "items": {"type": "string"}},
        "hashtags": {"type": "array", "items": {"type": "string"}},
        "category_id": {
            "type": "string",
            "enum": ["1", "2", "10", "15", "17", "19", "20", "22", "23", "24", "25", "26", "27", "28", "29"],
        },
        "pinned_comment": {"type": "string"},
        "thumbnail_text": {"type": "string"},
    },
    "required": ["title", "description", "tags", "hashtags", "category_id", "pinned_comment", "thumbnail_text"],
    "additionalProperties": False,
}

# One NotebookLM job at a time: the free tier has tight generation limits and
# parallel jobs mostly trade one rate-limit error for several.
_pipeline_lock = asyncio.Lock()
_jobs: dict[str, dict] = {}
_last_keepalive: dict = {"at": None, "ok": None, "error": None}


# ---------------------------------------------------------------- job storage


def _job_path(job_id: str) -> Path:
    return JOBS_DIR / job_id / "job.json"


def _save(job: dict) -> None:
    job["updated_at"] = time.time()
    path = _job_path(job["id"])
    tmp = path.with_suffix(".tmp")
    tmp.write_text(json.dumps(job, indent=2))
    tmp.replace(path)


def _update(job: dict, **fields) -> None:
    job.update(fields)
    _save(job)
    if "status" in fields:
        log.info("job %s -> %s", job["id"], fields["status"])


def _load_jobs() -> None:
    JOBS_DIR.mkdir(parents=True, exist_ok=True)
    for path in JOBS_DIR.glob("*/job.json"):
        job = json.loads(path.read_text())
        if job["status"] not in TERMINAL:
            # The asyncio task that owned it died with the old process.
            job["status"] = "error"
            job["error"] = "Bridge restarted while this job was running; drop the file in Drive again to retry."
            _save(job)
        _jobs[job["id"]] = job


# ---------------------------------------------------------------- pipeline


def _video_options() -> dict:
    fmt = VideoFormat[VIDEO_FORMAT]
    opts: dict = {"video_format": fmt, "language": VIDEO_LANGUAGE}
    # Cinematic and short formats have a fixed look and reject a style.
    if fmt not in (VideoFormat.CINEMATIC, VideoFormat.SHORT):
        opts["video_style"] = VideoStyle["AUTO_SELECT" if VIDEO_STYLE == "AUTO" else VIDEO_STYLE]
    if VIDEO_INSTRUCTIONS:
        opts["instructions"] = VIDEO_INSTRUCTIONS
    return opts


def _clean_metadata(meta: dict) -> dict:
    """Enforce the limits YouTube's API rejects uploads over."""

    def strip_brackets(s: str) -> str:
        return s.replace("<", "").replace(">", "").strip()

    meta["title"] = strip_brackets(meta["title"])[:100] or "Untitled video"
    meta["description"] = strip_brackets(meta["description"]).encode("utf-8")[:4900].decode("utf-8", "ignore")
    tags, total = [], 0
    for tag in meta["tags"]:
        tag = " ".join(strip_brackets(tag).replace(",", " ").lstrip("#").split())
        if not tag or tag.lower() in (t.lower() for t in tags):
            continue
        cost = len(tag) + (2 if " " in tag else 0) + 1  # YouTube counts quotes around multi-word tags
        if total + cost > 480:
            break
        tags.append(tag)
        total += cost
    meta["tags"] = tags
    # n8n's YouTube node splits on "," without trimming, so join without spaces.
    meta["tags_csv"] = ",".join(tags)
    meta["hashtags"] = ["#" + re.sub(r"[^\w]", "", h) for h in meta["hashtags"] if re.sub(r"[^\w]", "", h)][:3]
    return meta


async def _write_metadata(brief: str, job: dict) -> dict:
    client = anthropic.AsyncAnthropic()
    prompt = (
        f"Job details:\n- Title hint (Drive file name): {job['title_hint']}\n"
        f"- Video: NotebookLM Video Overview, format {VIDEO_FORMAT.lower()}, language {VIDEO_LANGUAGE}\n\n"
        f"Source brief from NotebookLM:\n<brief>\n{brief}\n</brief>\n\n"
        "Write the YouTube metadata for this video."
    )
    kwargs: dict = {}
    if CLAUDE_FALLBACKS != "off":
        kwargs = {"betas": ["server-side-fallback-2026-07-01"], "fallbacks": CLAUDE_FALLBACKS}
    response = await client.beta.messages.create(
        model=CLAUDE_MODEL,
        max_tokens=16000,
        system=SKILL_PATH.read_text(),
        output_config={
            "effort": CLAUDE_EFFORT,
            "format": {"type": "json_schema", "schema": METADATA_SCHEMA},
        },
        messages=[{"role": "user", "content": prompt}],
        **kwargs,
    )
    if response.stop_reason == "refusal":
        category = response.stop_details.category if response.stop_details else None
        raise RuntimeError(f"Claude declined to write metadata (category: {category})")
    if response.stop_reason == "max_tokens":
        raise RuntimeError("Claude's metadata answer was cut off at max_tokens")
    text = next(b.text for b in response.content if b.type == "text")
    return _clean_metadata(json.loads(text))


async def _run_pipeline(job: dict) -> None:
    job_dir = JOBS_DIR / job["id"]
    video_path = job_dir / "video.mp4"
    async with _pipeline_lock:
        async with NotebookLMClient.from_storage(NOTEBOOKLM_STORAGE, allow_headless=True) as nlm:
            _update(job, status="uploading")
            nb = await nlm.notebooks.create(f"YT: {job['title_hint']}"[:100])
            _update(job, notebook_id=nb.id)
            await nlm.sources.add_file(
                nb.id, job_dir / job["source_file"], wait=True, wait_timeout=600, title=job["title_hint"]
            )

            _update(job, status="generating")
            started = await nlm.artifacts.generate_video(nb.id, **_video_options())
            if started.is_failed or started.is_rate_limited:
                raise RuntimeError(f"NotebookLM refused to start the video: {started.error or started.status}")

            # Write the metadata while NotebookLM renders the video.
            brief = (await nlm.chat.ask(nb.id, BRIEF_PROMPT)).answer
            (job_dir / "brief.txt").write_text(brief)
            _update(job, metadata=await _write_metadata(brief, job))

            final = await nlm.artifacts.wait_for_completion(
                nb.id, started.task_id, timeout=JOB_TIMEOUT_MINUTES * 60, max_interval=30
            )
            if not final.is_complete:
                raise RuntimeError(f"NotebookLM video generation ended as {final.status}: {final.error}")

            _update(job, status="downloading")
            await nlm.artifacts.download_video(nb.id, str(video_path))
            if DELETE_NOTEBOOK_AFTER:
                await nlm.notebooks.delete(nb.id)

    _update(job, status="done", video_bytes=video_path.stat().st_size)


async def _run_job(job: dict) -> None:
    try:
        await _run_pipeline(job)
    except anthropic.APIStatusError as e:
        _update(job, status="error", error=f"Claude API error {e.status_code}: {e.message}")
    except anthropic.APIConnectionError as e:
        _update(job, status="error", error=f"Could not reach the Claude API: {e}")
    except Exception as e:  # noqa: BLE001 - every failure must land in the job record for n8n
        log.exception("job %s failed", job["id"])
        _update(job, status="error", error=f"{type(e).__name__}: {e}")


async def _keepalive_loop() -> None:
    """Touch NotebookLM regularly so the saved Google cookies keep rotating."""
    while True:
        await asyncio.sleep(KEEPALIVE_HOURS * 3600)
        if _pipeline_lock.locked():
            continue  # a running job is already keeping the session warm
        try:
            async with NotebookLMClient.from_storage(NOTEBOOKLM_STORAGE, allow_headless=True) as nlm:
                await nlm.notebooks.list()
            _last_keepalive.update(at=time.time(), ok=True, error=None)
        except Exception as e:  # noqa: BLE001
            log.warning("NotebookLM keepalive failed: %s", e)
            _last_keepalive.update(at=time.time(), ok=False, error=str(e))


# ---------------------------------------------------------------- HTTP API


@asynccontextmanager
async def lifespan(_: FastAPI):
    if not BRIDGE_TOKEN:
        raise RuntimeError("BRIDGE_TOKEN is not set")
    try:
        _video_options()
    except KeyError as e:
        raise RuntimeError(f"Unknown VIDEO_FORMAT/VIDEO_STYLE in .env: {e}") from None
    _load_jobs()
    task = asyncio.create_task(_keepalive_loop())
    yield
    task.cancel()


app = FastAPI(title="YouTube automation bridge", lifespan=lifespan)


def require_token(x_bridge_token: str = Header(default="")) -> None:
    if not secrets.compare_digest(x_bridge_token, BRIDGE_TOKEN):
        raise HTTPException(status_code=401, detail="bad or missing X-Bridge-Token")


def _get_job(job_id: str) -> dict:
    job = _jobs.get(job_id)
    if job is None:
        raise HTTPException(status_code=404, detail="no such job")
    return job


@app.get("/health")
async def health() -> dict:
    storage = Path(NOTEBOOKLM_STORAGE) if NOTEBOOKLM_STORAGE else Path.home() / ".notebooklm/profiles/default/storage_state.json"
    return {
        "ok": True,
        "notebooklm_login_found": storage.exists(),
        "skill_found": SKILL_PATH.exists(),
        "anthropic_key_set": bool(os.getenv("ANTHROPIC_API_KEY")),
        "last_keepalive": _last_keepalive,
        "busy": _pipeline_lock.locked(),
    }


@app.post("/jobs", dependencies=[Depends(require_token)])
async def create_job(
    file: UploadFile = File(...),
    title_hint: str = Form(""),
    mime_type: str = Form(""),
) -> dict:
    job_id = uuid.uuid4().hex[:12]
    job_dir = JOBS_DIR / job_id
    job_dir.mkdir(parents=True)

    name = file.filename or "source"
    suffix = Path(name).suffix.lower()
    if not re.fullmatch(r"\.[a-z0-9]{1,8}", suffix or ""):
        suffix = mimetypes.guess_extension(mime_type or file.content_type or "") or ".txt"
    source_file = f"source{suffix}"
    with (job_dir / source_file).open("wb") as out:
        shutil.copyfileobj(file.file, out)

    job = {
        "id": job_id,
        "status": "queued",
        "title_hint": (title_hint or Path(name).stem).strip()[:200],
        "source_file": source_file,
        "created_at": time.time(),
        "error": None,
        "metadata": None,
        "notebook_id": None,
        "youtube_video_id": None,
    }
    _jobs[job_id] = job
    _save(job)
    asyncio.create_task(_run_job(job))
    return {"job_id": job_id, "status": "queued"}


@app.get("/jobs/{job_id}", dependencies=[Depends(require_token)])
async def get_job(job_id: str) -> dict:
    return _get_job(job_id)


@app.get("/jobs/{job_id}/video", dependencies=[Depends(require_token)])
async def get_video(job_id: str) -> FileResponse:
    job = _get_job(job_id)
    path = JOBS_DIR / job_id / "video.mp4"
    if job["status"] != "done" or not path.exists():
        raise HTTPException(status_code=409, detail=f"video not ready (status {job['status']})")
    return FileResponse(path, media_type="video/mp4", filename=f"{job_id}.mp4")


class Published(BaseModel):
    youtube_video_id: str


@app.post("/jobs/{job_id}/published", dependencies=[Depends(require_token)])
async def mark_published(job_id: str, body: Published) -> dict:
    job = _get_job(job_id)
    _update(job, youtube_video_id=body.youtube_video_id)
    # The upload is the record now; drop the large local copy.
    (JOBS_DIR / job_id / "video.mp4").unlink(missing_ok=True)
    return job
