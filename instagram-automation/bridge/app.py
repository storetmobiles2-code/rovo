"""NotebookLM + Claude + Instagram bridge for the n8n Instagram workflow.

n8n has no NotebookLM node and no Instagram Reels publishing node, so this
service does three things n8n cannot:

1. turns a source document into a NotebookLM Video Overview (vertical "short"
   format by default) through the unofficial notebooklm-py client,
2. asks Claude, guided by the instagram-automation skill, for the caption,
3. publishes the Reel through the Instagram Graph API.

Instagram has no private or draft mode for API posts, so by default every Reel
waits on a local review page (http://localhost:8766) until you approve it.

Video generation takes 5-30 minutes, so work runs as background jobs that n8n
polls:

    POST /jobs                upload a document, returns {"job_id": ...}
    GET  /jobs/{id}           status, error, caption and (when posted) permalink
    POST /jobs/{id}/publish   start publishing an approved job
    GET  /instagram/check     verify the Instagram token and show the account
    GET  /health              liveness and configuration checks
    GET  /                    review page (browser, localhost only)

Job statuses: queued -> uploading -> generating -> downloading ->
awaiting_approval -> approved -> publishing -> published, or rejected / error.

API routes need the X-Bridge-Token header. The review pages are only reachable
from this computer and each approval carries a per-job secret.

Publishing modes (INSTAGRAM_LOGIN):
- "instagram" (default): Instagram API with Instagram Login. Meta fetches the
  video from a public URL, which a Cloudflare quick tunnel provides; the tunnel
  only reaches a tiny server that serves one random file name per post.
- "facebook": Instagram API with Facebook Login. Uploads the file directly
  (resumable upload), no tunnel involved; needs a Page access token.
"""

from __future__ import annotations

import asyncio
import hashlib
import html
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
import httpx
import uvicorn
from fastapi import Depends, FastAPI, File, Form, Header, HTTPException, UploadFile
from fastapi.responses import FileResponse, HTMLResponse, RedirectResponse
from notebooklm import NotebookLMClient
from notebooklm.types import VideoFormat, VideoStyle

log = logging.getLogger("bridge")
logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"), format="%(asctime)s %(levelname)s %(message)s")

BRIDGE_TOKEN = os.environ.get("BRIDGE_TOKEN", "")
DATA_DIR = Path(os.getenv("DATA_DIR", "/data"))
SKILL_PATH = Path(os.getenv("SKILL_PATH", "/skills/instagram-automation/SKILL.md"))
NOTEBOOKLM_STORAGE = os.getenv("NOTEBOOKLM_STORAGE") or None

CLAUDE_MODEL = os.getenv("CLAUDE_MODEL", "claude-opus-5-5")
CLAUDE_EFFORT = os.getenv("CLAUDE_EFFORT", "medium")
# "default" lets the API retry a declined request on a fallback model; "off" disables it.
CLAUDE_FALLBACKS = os.getenv("CLAUDE_FALLBACKS", "default")

VIDEO_FORMAT = os.getenv("VIDEO_FORMAT", "short").upper()
VIDEO_STYLE = os.getenv("VIDEO_STYLE", "auto").upper().replace("-", "_")
VIDEO_LANGUAGE = os.getenv("VIDEO_LANGUAGE", "en")
VIDEO_INSTRUCTIONS = os.getenv("VIDEO_INSTRUCTIONS", "")
DELETE_NOTEBOOK_AFTER = os.getenv("DELETE_NOTEBOOK_AFTER", "false").lower() == "true"
JOB_TIMEOUT_MINUTES = float(os.getenv("JOB_TIMEOUT_MINUTES", "60"))
KEEPALIVE_HOURS = float(os.getenv("KEEPALIVE_HOURS", "6"))

INSTAGRAM_LOGIN = os.getenv("INSTAGRAM_LOGIN", "instagram").lower()
INSTAGRAM_ACCESS_TOKEN = os.getenv("INSTAGRAM_ACCESS_TOKEN", "")
INSTAGRAM_USER_ID = os.getenv("INSTAGRAM_USER_ID", "")
GRAPH_API_VERSION = os.getenv("GRAPH_API_VERSION", "v25.0")
GRAPH_HOST = "https://graph.instagram.com" if INSTAGRAM_LOGIN == "instagram" else "https://graph.facebook.com"
REQUIRE_APPROVAL = os.getenv("INSTAGRAM_REQUIRE_APPROVAL", "true").lower() != "false"
APPROVAL_TIMEOUT_HOURS = float(os.getenv("APPROVAL_TIMEOUT_HOURS", "72"))
SHARE_TO_FEED = os.getenv("INSTAGRAM_SHARE_TO_FEED", "true").lower() != "false"
THUMB_OFFSET_MS = os.getenv("INSTAGRAM_THUMB_OFFSET_MS", "")
MAX_HASHTAGS = min(int(os.getenv("INSTAGRAM_MAX_HASHTAGS", "5")), 30)
CAPTION_FOOTER = os.getenv("CAPTION_FOOTER", "🤖 Video made with AI (NotebookLM) from the source material.")
PUBLISH_TIMEOUT_MINUTES = float(os.getenv("PUBLISH_TIMEOUT_MINUTES", "15"))
TOKEN_REFRESH_DAYS = float(os.getenv("INSTAGRAM_TOKEN_REFRESH_DAYS", "7"))
PUBLIC_BASE_URL = os.getenv("PUBLIC_BASE_URL", "").rstrip("/")
TUNNEL_METRICS_URL = os.getenv("TUNNEL_METRICS_URL", "http://tunnel:20241/quicktunnel")
REVIEW_URL = os.getenv("REVIEW_URL", "http://localhost:8766")

JOBS_DIR = DATA_DIR / "jobs"
PUBLIC_DIR = DATA_DIR / "public"
TOKEN_FILE = DATA_DIR / "instagram_token.json"
TERMINAL = {"published", "rejected", "error"}
MAX_REEL_BYTES = 300 * 1024 * 1024

BRIEF_PROMPT = (
    "Write a source brief for a social media editor who has not read these sources. "
    "Include: the document's title and author or organisation if stated, who it is for, "
    "a one-paragraph summary, 5-8 key points as a bulleted list, and the single most "
    "surprising or useful fact. Plain text, no citations."
)

CAPTION_SCHEMA = {
    "type": "object",
    "properties": {
        "hook": {"type": "string"},
        "body": {"type": "string"},
        "call_to_action": {"type": "string"},
        "hashtags": {"type": "array", "items": {"type": "string"}},
        "cover_text": {"type": "string"},
        "first_comment": {"type": "string"},
    },
    "required": ["hook", "body", "call_to_action", "hashtags", "cover_text", "first_comment"],
    "additionalProperties": False,
}

# One NotebookLM job at a time: the free tier has tight generation limits and
# parallel jobs mostly trade one rate-limit error for several.
_pipeline_lock = asyncio.Lock()
_publish_lock = asyncio.Lock()
_jobs: dict[str, dict] = {}
_status: dict = {"notebooklm_keepalive": None, "token_refresh": None}


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
    PUBLIC_DIR.mkdir(parents=True, exist_ok=True)
    for path in JOBS_DIR.glob("*/job.json"):
        job = json.loads(path.read_text())
        if job["status"] in ("queued", "uploading", "generating", "downloading"):
            # The asyncio task that owned it died with the old process.
            job.update(status="error", error="Bridge restarted while this job was running; drop the file in Drive again to retry.")
            _save(job)
        elif job["status"] == "publishing":
            # It may or may not have reached Instagram; retrying could double-post.
            job.update(status="error", error="Bridge restarted while publishing. Check Instagram before posting this again.")
            _save(job)
        _jobs[job["id"]] = job


def _expire_if_stale(job: dict) -> None:
    if job["status"] == "awaiting_approval" and time.time() > job.get("approval_deadline", float("inf")):
        _update(job, status="error", error=f"Not approved within {APPROVAL_TIMEOUT_HOURS:g} hours.")


# ---------------------------------------------------------------- caption


def _assemble_caption(meta: dict) -> str:
    """Build the caption and enforce Instagram's limits."""

    def no_mentions(s: str) -> str:
        # Mentions notify real accounts and Claude cannot verify handles.
        return re.sub(r"@(?=\w)", "", s).strip()

    tags = []
    for h in meta["hashtags"]:
        word = re.sub(r"[^\w]", "", h)
        if word and word.lower() not in (t.lower() for t in tags):
            tags.append(word)
    hashtags = " ".join("#" + t for t in tags[:MAX_HASHTAGS])

    parts = [no_mentions(meta["hook"]), no_mentions(meta["body"]), no_mentions(meta["call_to_action"])]
    tail = "\n\n".join(p for p in (CAPTION_FOOTER, hashtags) if p)
    caption = "\n\n".join(p for p in parts if p)
    room = 2200 - len(tail) - 2
    if len(caption) > room:
        caption = caption[: room - 1].rstrip() + "…"
    return f"{caption}\n\n{tail}" if tail else caption


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


async def _write_caption(brief: str, job: dict) -> dict:
    client = anthropic.AsyncAnthropic()
    prompt = (
        f"Job details:\n- Title hint (Drive file name): {job['title_hint']}\n"
        f"- Video: NotebookLM Video Overview, format {VIDEO_FORMAT.lower()}, language {VIDEO_LANGUAGE}\n"
        f"- Hashtags wanted: up to {MAX_HASHTAGS}\n\n"
        f"Source brief from NotebookLM:\n<brief>\n{brief}\n</brief>\n\n"
        "Write the Instagram Reel caption pieces for this video."
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
            "format": {"type": "json_schema", "schema": CAPTION_SCHEMA},
        },
        messages=[{"role": "user", "content": prompt}],
        **kwargs,
    )
    if response.stop_reason == "refusal":
        category = response.stop_details.category if response.stop_details else None
        raise RuntimeError(f"Claude declined to write the caption (category: {category})")
    if response.stop_reason == "max_tokens":
        raise RuntimeError("Claude's caption was cut off at max_tokens")
    text = next(b.text for b in response.content if b.type == "text")
    return json.loads(text)


async def _run_pipeline(job: dict) -> None:
    job_dir = JOBS_DIR / job["id"]
    video_path = job_dir / "video.mp4"
    async with _pipeline_lock:
        async with NotebookLMClient.from_storage(NOTEBOOKLM_STORAGE, allow_headless=True) as nlm:
            _update(job, status="uploading")
            nb = await nlm.notebooks.create(f"IG: {job['title_hint']}"[:100])
            _update(job, notebook_id=nb.id)
            await nlm.sources.add_file(
                nb.id, job_dir / job["source_file"], wait=True, wait_timeout=600, title=job["title_hint"]
            )

            _update(job, status="generating")
            started = await nlm.artifacts.generate_video(nb.id, **_video_options())
            if started.is_failed or started.is_rate_limited:
                raise RuntimeError(f"NotebookLM refused to start the video: {started.error or started.status}")

            # Write the caption while NotebookLM renders the video.
            brief = (await nlm.chat.ask(nb.id, BRIEF_PROMPT)).answer
            (job_dir / "brief.txt").write_text(brief)
            meta = await _write_caption(brief, job)
            _update(job, metadata=meta, caption=_assemble_caption(meta))

            final = await nlm.artifacts.wait_for_completion(
                nb.id, started.task_id, timeout=JOB_TIMEOUT_MINUTES * 60, max_interval=30
            )
            if not final.is_complete:
                raise RuntimeError(f"NotebookLM video generation ended as {final.status}: {final.error}")

            _update(job, status="downloading")
            await nlm.artifacts.download_video(nb.id, str(video_path))
            if DELETE_NOTEBOOK_AFTER:
                await nlm.notebooks.delete(nb.id)

    size = video_path.stat().st_size
    if size > MAX_REEL_BYTES:
        raise RuntimeError(f"Video is {size // 2**20} MB; Instagram Reels allow at most 300 MB.")
    if REQUIRE_APPROVAL:
        _update(
            job,
            status="awaiting_approval",
            video_bytes=size,
            approval_deadline=time.time() + APPROVAL_TIMEOUT_HOURS * 3600,
        )
        log.info("job %s ready for review at %s/review/%s?k=%s", job["id"], REVIEW_URL, job["id"], job["review_token"])
    else:
        _update(job, status="approved", video_bytes=size)


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


# ---------------------------------------------------------------- Instagram


class GraphError(RuntimeError):
    pass


def _source_hash() -> str:
    return hashlib.sha256(INSTAGRAM_ACCESS_TOKEN.encode()).hexdigest()


def _token() -> str:
    """The newest Instagram token: a refreshed one if we have it, else .env's."""
    if TOKEN_FILE.exists():
        saved = json.loads(TOKEN_FILE.read_text())
        # A new token pasted into .env wins over an old refreshed one.
        if saved.get("source_hash") == _source_hash():
            return saved["access_token"]
    return INSTAGRAM_ACCESS_TOKEN


async def _graph(client: httpx.AsyncClient, method: str, path: str, **params) -> dict:
    url = f"{GRAPH_HOST}/{GRAPH_API_VERSION}/{path}"
    params["access_token"] = _token()
    if method == "GET":
        r = await client.get(url, params=params)
    else:
        r = await client.post(url, data=params)
    body = r.json() if r.headers.get("content-type", "").startswith(("application/json", "text/javascript")) else {}
    if r.status_code >= 400 or "error" in body:
        err = body.get("error", {})
        raise GraphError(f"Instagram API {r.status_code}: {err.get('message') or r.text[:300]}")
    return body


async def _account(client: httpx.AsyncClient) -> dict:
    if INSTAGRAM_LOGIN == "instagram":
        me = await _graph(client, "GET", "me", fields="user_id,username")
        return {"id": me["user_id"], "username": me.get("username")}
    if not INSTAGRAM_USER_ID:
        raise GraphError("INSTAGRAM_USER_ID must be set when INSTAGRAM_LOGIN=facebook")
    acct = await _graph(client, "GET", INSTAGRAM_USER_ID, fields="username")
    return {"id": INSTAGRAM_USER_ID, "username": acct.get("username")}


async def _public_base_url(client: httpx.AsyncClient) -> str:
    if PUBLIC_BASE_URL:
        return PUBLIC_BASE_URL
    try:
        hostname = (await client.get(TUNNEL_METRICS_URL, timeout=5)).json().get("hostname")
    except (httpx.HTTPError, ValueError) as e:
        raise RuntimeError(f"Cloudflare tunnel is not reachable ({e}); check `docker compose logs tunnel`") from e
    if not hostname:
        raise RuntimeError("Cloudflare tunnel has no public hostname yet; try again in a minute")
    return f"https://{hostname}"


async def _quota_left(client: httpx.AsyncClient, ig_id: str) -> int | None:
    """Posts left in Instagram's rolling 24-hour API publishing limit, if readable."""
    try:
        limit = await _graph(client, "GET", f"{ig_id}/content_publishing_limit", fields="quota_usage,config")
    except GraphError as e:
        log.warning("could not read publishing quota: %s", e)
        return None
    row = (limit.get("data") or [{}])[0]
    total = (row.get("config") or {}).get("quota_total")
    return None if total is None else total - row.get("quota_usage", 0)


async def _create_container(client: httpx.AsyncClient, ig_id: str, video: Path, caption: str, public_name: str) -> str:
    params: dict = {"media_type": "REELS", "caption": caption, "share_to_feed": str(SHARE_TO_FEED).lower()}
    if THUMB_OFFSET_MS:
        params["thumb_offset"] = THUMB_OFFSET_MS

    if INSTAGRAM_LOGIN == "facebook":
        created = await _graph(client, "POST", f"{ig_id}/media", upload_type="resumable", **params)
        upload = await client.post(
            created.get("uri") or f"https://rupload.facebook.com/ig-api-upload/{GRAPH_API_VERSION}/{created['id']}",
            headers={"Authorization": f"OAuth {_token()}", "offset": "0", "file_size": str(video.stat().st_size)},
            content=video.read_bytes(),
            timeout=httpx.Timeout(600),
        )
        if upload.status_code >= 400:
            raise GraphError(f"Instagram upload failed {upload.status_code}: {upload.text[:300]}")
        return created["id"]

    # Instagram Login: Meta downloads the video from a public URL. Serve a copy
    # under a random name through the tunnel; the caller deletes it afterwards.
    shutil.copyfile(video, PUBLIC_DIR / public_name)
    url = f"{await _public_base_url(client)}/media/{public_name}"
    return (await _graph(client, "POST", f"{ig_id}/media", video_url=url, **params))["id"]


async def _publish(job: dict) -> None:
    video = JOBS_DIR / job["id"] / "video.mp4"
    public_name = f"{secrets.token_hex(16)}.mp4"
    try:
        async with _publish_lock, httpx.AsyncClient(timeout=60) as client:
            account = await _account(client)
            ig_id = account["id"]
            left = await _quota_left(client, ig_id)
            if left is not None and left <= 0:
                raise GraphError("Instagram's limit on API posts per 24 hours is used up; approve it again later.")

            container_id = await _create_container(client, ig_id, video, job["caption"], public_name)
            _update(job, container_id=container_id, instagram_username=account["username"])

            deadline = time.time() + PUBLISH_TIMEOUT_MINUTES * 60
            while True:
                state = await _graph(client, "GET", container_id, fields="status_code,status")
                code = state.get("status_code")
                if code == "FINISHED":
                    break
                if code in ("ERROR", "EXPIRED"):
                    raise GraphError(f"Instagram could not process the video ({code}): {state.get('status')}")
                if time.time() > deadline:
                    raise GraphError(f"Instagram was still processing after {PUBLISH_TIMEOUT_MINUTES:g} minutes")
                await asyncio.sleep(30)

            media = await _graph(client, "POST", f"{ig_id}/media_publish", creation_id=container_id)
            try:
                permalink = (await _graph(client, "GET", media["id"], fields="permalink")).get("permalink")
            except GraphError:
                permalink = None
        _update(job, status="published", instagram_media_id=media["id"], permalink=permalink)
        video.unlink(missing_ok=True)
    except Exception as e:  # noqa: BLE001
        log.exception("publishing job %s failed", job["id"])
        _update(job, status="error", error=f"{type(e).__name__}: {e}")
    finally:
        (PUBLIC_DIR / public_name).unlink(missing_ok=True)


async def _refresh_instagram_token() -> None:
    """Long-lived Instagram Login tokens last 60 days; refreshing resets that."""
    saved = json.loads(TOKEN_FILE.read_text()) if TOKEN_FILE.exists() else {}
    if saved.get("source_hash") != _source_hash():
        # New token in .env: start counting from now (it must be 24h old to refresh).
        saved = {"source_hash": _source_hash(), "access_token": INSTAGRAM_ACCESS_TOKEN, "refreshed_at": time.time()}
        TOKEN_FILE.write_text(json.dumps(saved))
        return
    if time.time() - saved["refreshed_at"] < TOKEN_REFRESH_DAYS * 86400:
        return
    async with httpx.AsyncClient(timeout=30) as client:
        r = await client.get(
            "https://graph.instagram.com/refresh_access_token",
            params={"grant_type": "ig_refresh_token", "access_token": saved["access_token"]},
        )
    body = r.json()
    if "access_token" not in body:
        raise GraphError(f"token refresh failed: {body.get('error', {}).get('message', r.text[:200])}")
    saved.update(access_token=body["access_token"], refreshed_at=time.time(), expires_in=body.get("expires_in"))
    tmp = TOKEN_FILE.with_suffix(".tmp")
    tmp.write_text(json.dumps(saved))
    tmp.replace(TOKEN_FILE)
    log.info("Instagram access token refreshed")


async def _maintenance_loop() -> None:
    while True:
        if INSTAGRAM_LOGIN == "instagram" and INSTAGRAM_ACCESS_TOKEN:
            try:
                await _refresh_instagram_token()
                _status["token_refresh"] = {"at": time.time(), "ok": True}
            except Exception as e:  # noqa: BLE001
                log.warning("Instagram token refresh failed: %s", e)
                _status["token_refresh"] = {"at": time.time(), "ok": False, "error": str(e)}
        if not _pipeline_lock.locked():
            # Touch NotebookLM so the saved Google cookies keep rotating.
            try:
                async with NotebookLMClient.from_storage(NOTEBOOKLM_STORAGE, allow_headless=True) as nlm:
                    await nlm.notebooks.list()
                _status["notebooklm_keepalive"] = {"at": time.time(), "ok": True}
            except Exception as e:  # noqa: BLE001
                log.warning("NotebookLM keepalive failed: %s", e)
                _status["notebooklm_keepalive"] = {"at": time.time(), "ok": False, "error": str(e)}
        await asyncio.sleep(KEEPALIVE_HOURS * 3600)


# ---------------------------------------------------------------- public media server

# The only thing the Cloudflare tunnel can reach: files named by a 128-bit
# random token, present only while Instagram is fetching them.
media_app = FastAPI(docs_url=None, redoc_url=None, openapi_url=None)


@media_app.get("/media/{name}")
async def public_media(name: str) -> FileResponse:
    path = PUBLIC_DIR / name
    if not re.fullmatch(r"[0-9a-f]{32}\.mp4", name) or not path.exists():
        raise HTTPException(status_code=404)
    return FileResponse(path, media_type="video/mp4")


# ---------------------------------------------------------------- HTTP API


@asynccontextmanager
async def lifespan(_: FastAPI):
    if not BRIDGE_TOKEN:
        raise RuntimeError("BRIDGE_TOKEN is not set")
    if INSTAGRAM_LOGIN not in ("instagram", "facebook"):
        raise RuntimeError("INSTAGRAM_LOGIN must be 'instagram' or 'facebook'")
    try:
        _video_options()
    except KeyError as e:
        raise RuntimeError(f"Unknown VIDEO_FORMAT/VIDEO_STYLE in .env: {e}") from None
    _load_jobs()
    media_server = uvicorn.Server(
        uvicorn.Config(media_app, host="0.0.0.0", port=8001, log_config=None, access_log=False, lifespan="off")
    )
    maintenance = asyncio.create_task(_maintenance_loop())
    media = asyncio.create_task(media_server.serve())
    yield
    maintenance.cancel()
    media_server.should_exit = True
    await asyncio.wait({media}, timeout=5)


app = FastAPI(title="Instagram automation bridge", lifespan=lifespan)


def require_token(x_bridge_token: str = Header(default="")) -> None:
    if not secrets.compare_digest(x_bridge_token, BRIDGE_TOKEN):
        raise HTTPException(status_code=401, detail="bad or missing X-Bridge-Token")


def _get_job(job_id: str) -> dict:
    job = _jobs.get(job_id)
    if job is None:
        raise HTTPException(status_code=404, detail="no such job")
    _expire_if_stale(job)
    return job


def _public_job(job: dict) -> dict:
    return {k: v for k, v in job.items() if k != "review_token"}


@app.get("/health")
async def health() -> dict:
    storage = Path(NOTEBOOKLM_STORAGE) if NOTEBOOKLM_STORAGE else Path.home() / ".notebooklm/profiles/default/storage_state.json"
    return {
        "ok": True,
        "notebooklm_login_found": storage.exists(),
        "skill_found": SKILL_PATH.exists(),
        "anthropic_key_set": bool(os.getenv("ANTHROPIC_API_KEY")),
        "instagram_token_set": bool(INSTAGRAM_ACCESS_TOKEN),
        "instagram_login": INSTAGRAM_LOGIN,
        "require_approval": REQUIRE_APPROVAL,
        "maintenance": _status,
        "busy": _pipeline_lock.locked(),
    }


@app.get("/instagram/check", dependencies=[Depends(require_token)])
async def instagram_check() -> dict:
    async with httpx.AsyncClient(timeout=30) as client:
        try:
            account = await _account(client)
        except GraphError as e:
            raise HTTPException(status_code=502, detail=str(e)) from e
        public = None
        if INSTAGRAM_LOGIN == "instagram":
            try:
                public = await _public_base_url(client)
            except RuntimeError as e:
                public = f"not ready: {e}"
    return {"ok": True, "account": account, "public_base_url": public}


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
        "caption": None,
        "review_token": secrets.token_urlsafe(24),
        "notebook_id": None,
        "instagram_media_id": None,
        "permalink": None,
    }
    _jobs[job_id] = job
    _save(job)
    asyncio.create_task(_run_job(job))
    return {"job_id": job_id, "status": "queued", "review_url": f"{REVIEW_URL}/"}


@app.get("/jobs/{job_id}", dependencies=[Depends(require_token)])
async def get_job(job_id: str) -> dict:
    return _public_job(_get_job(job_id))


@app.post("/jobs/{job_id}/publish", dependencies=[Depends(require_token)])
async def publish(job_id: str) -> dict:
    job = _get_job(job_id)
    if job["status"] in ("publishing", "published"):
        return _public_job(job)  # already under way; never post twice
    if job["status"] != "approved":
        raise HTTPException(status_code=409, detail=f"job is {job['status']}, not approved")
    _update(job, status="publishing")
    asyncio.create_task(_publish(job))
    return _public_job(job)


# ---------------------------------------------------------------- review pages

PAGE_STYLE = """
:root { --bg:#fafafa; --fg:#1a1a1a; --muted:#666; --card:#fff; --line:#ddd; --accent:#c13584; --ok:#1a7f37; --bad:#b42318; }
@media (prefers-color-scheme: dark) { :root { --bg:#121212; --fg:#eee; --muted:#aaa; --card:#1e1e1e; --line:#333; --accent:#e1306c; --ok:#3fb950; --bad:#f85149; } }
* { box-sizing: border-box; }
body { margin:0; padding:24px 16px; background:var(--bg); color:var(--fg); font:15px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif; }
main { max-width:880px; margin:0 auto; }
h1 { font-size:22px; margin:0 0 4px; }
.muted { color:var(--muted); }
table { width:100%; border-collapse:collapse; background:var(--card); border:1px solid var(--line); border-radius:8px; overflow:hidden; }
td, th { padding:10px 12px; border-bottom:1px solid var(--line); text-align:left; vertical-align:top; }
.review { display:flex; gap:24px; flex-wrap:wrap; align-items:flex-start; }
video { width:300px; max-width:100%; border-radius:10px; background:#000; }
.panel { flex:1; min-width:280px; }
textarea { width:100%; min-height:300px; padding:10px; font:inherit; color:var(--fg); background:var(--card); border:1px solid var(--line); border-radius:8px; }
button { font:inherit; padding:10px 18px; border-radius:8px; border:0; cursor:pointer; color:#fff; margin-right:8px; }
.approve { background:var(--ok); } .reject { background:var(--bad); }
a { color:var(--accent); }
"""


def _page(title: str, body: str) -> HTMLResponse:
    return HTMLResponse(
        f"<!doctype html><html><head><meta charset='utf-8'><meta name='viewport' content='width=device-width,initial-scale=1'>"
        f"<title>{html.escape(title)}</title><style>{PAGE_STYLE}</style></head><body><main>{body}</main></body></html>",
        headers={"Cache-Control": "no-store", "X-Frame-Options": "DENY"},
    )


def _review_job(job_id: str, k: str) -> dict:
    job = _get_job(job_id)
    if not secrets.compare_digest(k, job.get("review_token", "")):
        raise HTTPException(status_code=403, detail="bad review link")
    return job


@app.get("/", response_class=HTMLResponse)
async def review_index() -> HTMLResponse:
    rows = []
    for job in sorted(_jobs.values(), key=lambda j: j["created_at"], reverse=True)[:50]:
        _expire_if_stale(job)
        status = job["status"]
        if status == "awaiting_approval":
            status = f"<a href='/review/{job['id']}?k={job['review_token']}'><b>Review &amp; approve</b></a>"
        elif status == "published" and job.get("permalink"):
            status = f"published · <a href='{html.escape(job['permalink'])}' target='_blank' rel='noopener'>view</a>"
        elif status == "error":
            status = f"error <span class='muted'>· {html.escape(job.get('error') or '')}</span>"
        when = time.strftime("%Y-%m-%d %H:%M", time.localtime(job["created_at"]))
        rows.append(f"<tr><td>{html.escape(job['title_hint'])}</td><td>{when}</td><td>{status}</td></tr>")
    table = (
        "<table><tr><th>Source document</th><th>Started</th><th>Status</th></tr>" + "".join(rows) + "</table>"
        if rows
        else "<p class='muted'>No jobs yet. Drop a document into your watched Google Drive folder.</p>"
    )
    mode = "Reels wait here for your approval before they go public." if REQUIRE_APPROVAL else "Auto-publish is on: approved automatically."
    return _page("Instagram review", f"<h1>Instagram Reels</h1><p class='muted'>{mode}</p>{table}")


@app.get("/review/{job_id}", response_class=HTMLResponse)
async def review_page(job_id: str, k: str = "") -> HTMLResponse:
    job = _review_job(job_id, k)
    if job["status"] != "awaiting_approval":
        return RedirectResponse("/", status_code=303)
    meta = job.get("metadata") or {}
    esc_k = html.escape(k)
    body = f"""
      <p><a href="/">← All jobs</a></p>
      <h1>{html.escape(job['title_hint'])}</h1>
      <p class="muted">Approving posts this Reel publicly on Instagram within a few minutes.</p>
      <div class="review">
        <video controls playsinline src="/review/{job_id}/video?k={esc_k}"></video>
        <form class="panel" method="post" action="/review/{job_id}">
          <input type="hidden" name="k" value="{esc_k}">
          <label for="caption"><b>Caption</b> <span class="muted">(max 2,200 characters)</span></label>
          <textarea id="caption" name="caption" maxlength="2200">{html.escape(job.get('caption') or '')}</textarea>
          <p class="muted">Cover text idea: {html.escape(meta.get('cover_text', ''))}<br>
             First comment idea: {html.escape(meta.get('first_comment', ''))}</p>
          <button class="approve" name="action" value="approve">Approve &amp; post</button>
          <button class="reject" name="action" value="reject">Reject</button>
        </form>
      </div>"""
    return _page("Review Reel", body)


@app.get("/review/{job_id}/video")
async def review_video(job_id: str, k: str = "") -> FileResponse:
    _review_job(job_id, k)
    path = JOBS_DIR / job_id / "video.mp4"
    if not path.exists():
        raise HTTPException(status_code=404)
    return FileResponse(path, media_type="video/mp4")


@app.post("/review/{job_id}")
async def review_decision(job_id: str, k: str = Form(""), action: str = Form(...), caption: str = Form("")) -> RedirectResponse:
    job = _review_job(job_id, k)
    if job["status"] == "awaiting_approval":
        if action == "approve":
            _update(job, status="approved", caption=caption.strip()[:2200] or job["caption"], approved_at=time.time())
        elif action == "reject":
            _update(job, status="rejected")
            (JOBS_DIR / job_id / "video.mp4").unlink(missing_ok=True)
    return RedirectResponse("/", status_code=303)
