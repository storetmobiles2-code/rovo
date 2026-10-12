# Instagram automation: Drive → NotebookLM → Claude → Instagram Reels

Drop a document into a Google Drive folder and get a narrated vertical video,
with a caption and hashtags written for Instagram, ready to post as a Reel.
Nothing goes public until you approve it.

```
 Google Drive folder         n8n (free, self-hosted, in Docker)
 ┌──────────────┐   poll    ┌───────────────────────────────────────────────┐
 │ report.pdf   │ ────────▶ │ Drive trigger → download → bridge job ─┐      │
 └──────────────┘  5 min    │     ▲                                  │      │
                            │     └── wait 2 min (10 min in review) ◀┤      │
                            │                  approved → publish ───┤      │
                            │        published / rejected / failed → end    │
                            └────────────────────────┬──────────────────────┘
                                                     │ HTTP (local only)
 You ── http://localhost:8766 ──▶ ┌──────────────────▼────────────────────┐
        watch, edit caption,      │ bridge (Python, in Docker)            │
        Approve & post            │  1. NotebookLM: notebook + document   │
                                  │  2. NotebookLM: vertical video (MP4)  │
                                  │  3. Claude + instagram-automation     │
                                  │     skill: caption, hashtags          │
                                  │  4. Instagram Graph API: post Reel    │
                                  └──────────────────┬────────────────────┘
                                   one random video URL while posting
                                  ┌──────────────────▼────────────────────┐
                                  │ Cloudflare quick tunnel (free)        │ ◀── Instagram fetches the MP4
                                  └───────────────────────────────────────┘
```

Everything runs on your own Mac or Windows PC. It uses different ports from
the [YouTube automation](../youtube-automation/), so both can run side by side
and share the same NotebookLM sign-in.

## What you need

| | Cost | Notes |
|---|---|---|
| [Docker Desktop](https://www.docker.com/products/docker-desktop/) | Free for personal use | The installer offers to install it. |
| n8n | Free (self-hosted community edition) | Runs in Docker. |
| Google account with NotebookLM | Free tier works | Free accounts have a daily limit on Video Overviews. |
| [Anthropic API key](https://platform.claude.com/settings/keys) | Pay per use | One short request per Reel. |
| Instagram **Business or Creator** account | Free | Personal accounts cannot post through the API. Switch in the Instagram app: Settings → Account type and tools. |
| Meta developer app | Free | Gives you the Instagram access token. Setup below. |
| Google Cloud project (OAuth client) | Free | Lets n8n read your Drive folder. Setup below. |
| Cloudflare quick tunnel | Free, no account | Started automatically; see "How the video reaches Instagram". |

## Install

Download the one file for your computer from [`dist/`](dist/):

- **Mac:** `InstagramAutomation-Setup.command`. Double-click it. If macOS says
  it can't be opened, right-click → Open, or run
  `bash ~/Downloads/InstagramAutomation-Setup.command` in Terminal.
  (It also works on Linux.)
- **Windows 10/11:** `InstagramAutomation-Setup.cmd`. Double-click it. If
  SmartScreen warns, choose More info → Run anyway.

The installer checks for Docker Desktop and writes the project to
`~/instagram-automation`. It creates `.env` with random secrets and asks for
your Anthropic API key and Instagram access token. If you are not already
signed in to NotebookLM, it opens a browser so you can sign in once. It then
copies the `instagram-automation` skill to `~/.claude/skills/`, starts
everything with `docker compose`, imports the workflow, and checks your
Instagram token.

You can press Enter at the Instagram token prompt and add it to `.env` later.
Re-running the installer is safe: it keeps your settings, n8n account and
saved credentials.

## Instagram access token (one time, about 10 minutes)

This uses the **Instagram API with Instagram Login**, which needs no Facebook
Page. Meta's own walkthrough is
[here](https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/);
the short version:

1. Make sure your Instagram account is a Business or Creator account.
2. Go to https://developers.facebook.com/apps, **Create app**, and choose the
   use case for managing messaging and content on Instagram.
3. In the app, open **Instagram → API setup with Instagram business login**.
4. Under **Generate access tokens**, add your Instagram account and sign in.
   If asked, accept the tester invitation in Instagram (Settings → Website
   permissions / Apps and websites → Tester invites).
5. Click **Generate token**, approve the permissions
   (`instagram_business_basic`, `instagram_business_content_publish`), and copy
   the token.
6. Paste it when the installer asks, or put it in `.env` as
   `INSTAGRAM_ACCESS_TOKEN=...` and run `docker compose up -d`.

The token lasts 60 days; the bridge refreshes it every 7 days, so it does not
run out while the bridge is running. If the bridge is stopped for more than
about 50 days, generate a new token. Your app can stay in development mode
because it only posts to your own account.

Check it any time:

```bash
curl -H "X-Bridge-Token: <BRIDGE_TOKEN from .env>" http://localhost:8766/instagram/check
```

## Google credentials (one time, about 5 minutes)

n8n needs an OAuth client of your own to read the Drive folder. If you already
made one for the YouTube automation, reuse it and just add the second redirect
URI from step 4.

1. Go to https://console.cloud.google.com/ and create (or pick) a project.
2. **APIs & Services → Library**: enable the **Google Drive API**.
3. **OAuth consent screen** (Google Auth Platform): user type **External**,
   fill in the app name and your email, and add your Google account as a
   **Test user**.
4. **Credentials → Create credentials → OAuth client ID**, type
   **Web application**, with the authorized redirect URI
   `http://localhost:5680/rest/oauth2-credential/callback`.
5. In n8n (http://localhost:5680), open the workflow. In **New file in Drive
   folder** choose Credential → *Create new* → Google Drive OAuth2 API, paste
   the client ID and secret, *Sign in with Google*, and pick the folder to
   watch. Select the same credential in **Download file**.
6. Activate (publish) the workflow with the toggle at the top.

While the consent screen is in **Testing**, Google expires the sign-in after
7 days. For a personal setup, set it to **In production**; the "unverified app"
warning you then see when signing in is expected for your own app.

## Using it

1. Put a PDF, Google Doc, Slides deck, Word, text or Markdown file in the
   watched folder.
2. About 10-30 minutes later the Reel shows up at **http://localhost:8766**
   with status *Review & approve*.
3. Watch it, edit the caption if you like, and click **Approve & post**, or
   **Reject**.
4. Within about 10 minutes n8n publishes it, and the review page links to
   the live Reel.

Reels you don't approve within 72 hours expire. To skip the review step and
post automatically, set `INSTAGRAM_REQUIRE_APPROVAL=false`. Only do this once
you trust the output: Instagram has no private or draft mode for API posts.

Progress is also visible in n8n's **Executions** tab, or with
`docker compose logs -f bridge` in the install folder.

## How the video reaches Instagram

With Instagram Login, Instagram does not accept a file upload. It downloads
the video from a public `https://` address. So the stack runs a free
[Cloudflare quick tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/do-more-with-tunnels/trycloudflare/),
which gives the bridge a temporary public address. The tunnel only reaches a
tiny server inside the bridge that serves files named by a 128-bit random
token. A file exists there only while Instagram is fetching that one Reel, and
is deleted right after. Nothing else (n8n, the review page, the API) is
reachable from the internet.

Quick tunnels are meant for light use and come without uptime guarantees.
Alternatives:

- **Your own public URL:** if you already expose the bridge's port 8001 through
  your own domain or tunnel, set `PUBLIC_BASE_URL=https://your.domain` and run
  `docker compose up -d --scale tunnel=0`.
- **Facebook Login (direct upload, no tunnel):** link the Instagram account to
  a Facebook Page, get a Page access token with `instagram_basic`,
  `instagram_content_publish` and `pages_read_engagement`, then set
  `INSTAGRAM_LOGIN=facebook`, `INSTAGRAM_ACCESS_TOKEN=<page token>` and
  `INSTAGRAM_USER_ID=<Instagram business account id>`. The bridge then uploads
  the file straight to Meta's resumable upload endpoint.

## Settings

Edit `.env` in the install folder, then run `docker compose up -d` there.

| Setting | Default | Meaning |
|---|---|---|
| `INSTAGRAM_REQUIRE_APPROVAL` | `true` | Wait for your approval on the review page. |
| `APPROVAL_TIMEOUT_HOURS` | `72` | Unapproved Reels expire after this. |
| `INSTAGRAM_SHARE_TO_FEED` | `true` | Also show the Reel in your main feed and profile grid. |
| `INSTAGRAM_THUMB_OFFSET_MS` | empty | Cover frame in milliseconds (empty = first frame). |
| `INSTAGRAM_MAX_HASHTAGS` | `5` | Hashtags kept from Claude's suggestions (Instagram allows 30). |
| `CAPTION_FOOTER` | AI disclosure line | Appended to every caption; empty to drop. |
| `INSTAGRAM_LOGIN` | `instagram` | `instagram` or `facebook` (see above). |
| `VIDEO_FORMAT` | `short` | `short` is vertical 9:16, best for Reels. `explainer`/`brief` are landscape. |
| `VIDEO_STYLE` | `auto` | Look of `explainer`/`brief` videos: `classic`, `whiteboard`, `kawaii`, `anime`, `watercolor`, `retro-print`, `heritage`, `paper-craft`. |
| `VIDEO_LANGUAGE` | `en` | Narration language code. |
| `VIDEO_INSTRUCTIONS` | empty | Extra direction for every video. |
| `CLAUDE_MODEL` / `CLAUDE_EFFORT` | `claude-opus-5-5` / `medium` | Model and thinking effort for captions. |

To change how captions are written, edit
`skills/instagram-automation/SKILL.md`. The bridge reads it on every job.

## Day-to-day commands

Run these in the install folder:

| | |
|---|---|
| Review page | http://localhost:8766 |
| Stop / start | `docker compose down` / `docker compose up -d` |
| Logs | `docker compose logs -f` |
| Bridge status | `curl http://localhost:8766/health` |
| Update | Run the newest installer again |
| Remove everything | `docker compose down -v`, then delete the folder |

## Troubleshooting

- **`Instagram API 400: ... video_url ...` or the container ends in `ERROR`.**
  Instagram could not fetch or process the video. Check
  `docker compose logs tunnel` for a `trycloudflare.com` address (restart it
  with `docker compose restart tunnel` if there is none), then drop the file
  into the Drive folder again. Reels must be 3 seconds to 15 minutes long and
  under 300 MB.
- **`Instagram API 190` / `Invalid OAuth access token`.** The token expired or
  was revoked. Generate a new one, put it in `.env`, and run `docker compose up -d`.
- **`limit on API posts per 24 hours is used up`.** Meta limits accounts to
  100 API-published posts per rolling 24 hours.
- **Job error mentions NotebookLM authentication or cookies.** Run the
  installer again with `--relogin` (Mac), or delete `storage_state.json` and
  re-run (Windows), then `docker compose restart bridge`.
- **Nothing happens when a file is added.** Make sure the workflow is active,
  the right folder is selected, and the file was created in that folder.

## Good to know

- NotebookLM has no public API for personal accounts. The bridge uses
  [notebooklm-py](https://github.com/teng-lin/notebooklm-py), an unofficial
  client; a change on Google's side can break it until the library is updated.
- Secrets stay on your computer: `.env` holds your API key and Instagram token,
  and `~/.notebooklm/.../storage_state.json` signs in to your Google account.
  The n8n (5680) and review (8766) ports listen on localhost only.
- Captions include a line saying the video was made with AI. Instagram may
  also add its own "AI info" label; follow Meta's rules on AI-generated content.

## For developers

```
instagram-automation/
├── docker-compose.yml        n8n + bridge + Cloudflare tunnel
├── .env.example              every setting, documented
├── bridge/                   FastAPI: NotebookLM, Claude, review UI, Instagram publishing
├── skills/instagram-automation/SKILL.md
├── n8n/instagram-automation.json
├── installer/                templates + build.py
└── dist/                     built single-file installers
```

After changing any of these files, rebuild the installers with
`python3 installer/build.py`.
