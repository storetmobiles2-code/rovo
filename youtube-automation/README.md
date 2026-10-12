# YouTube automation: Drive → NotebookLM → Claude → YouTube

Drop a document into a Google Drive folder and get a narrated explainer video on
your YouTube channel, with a title, description and tags written for YouTube.

```
 Google Drive folder          n8n (free, self-hosted, in Docker)
 ┌───────────────┐   poll    ┌──────────────────────────────────────────────┐
 │ report.pdf    │ ────────▶ │ Drive trigger → download → bridge job ─┐     │
 └───────────────┘  5 min    │      ▲                                 │     │
                             │      └──── wait 2 min ◀── not done ────┤     │
                             │                                  done  ▼     │
                             │   download MP4 → YouTube upload → record     │
                             └───────────────────────┬──────────────────────┘
                                                     │ HTTP (local only)
                             ┌───────────────────────▼──────────────────────┐
                             │ bridge (Python, in Docker)                    │
                             │  1. NotebookLM: new notebook + your document  │
                             │  2. NotebookLM: Video Overview (MP4)          │
                             │  3. NotebookLM: source brief                  │
                             │  4. Claude + youtube-automation skill:        │
                             │     title, description, tags, category        │
                             └──────────────────────────────────────────────┘
```

Everything runs on your own Mac or Windows PC. Nothing is hosted for you and
no n8n cloud account is needed.

## What you need

| | Cost | Notes |
|---|---|---|
| [Docker Desktop](https://www.docker.com/products/docker-desktop/) | Free for personal use | The installer offers to install it. |
| n8n | Free (self-hosted community edition) | Runs in Docker. |
| Google account with NotebookLM | Free tier works | Free accounts have a daily limit on Video Overviews. |
| [Anthropic API key](https://platform.claude.com/settings/keys) | Pay per use | One short request per video. |
| Google Cloud project (OAuth client) | Free | Lets n8n read Drive and upload to YouTube. Setup below. |

## Install

Download the one file for your computer from [`dist/`](dist/):

- **Mac:** `YouTubeAutomation-Setup.command`. Double-click it. If macOS says it
  can't be opened, right-click → Open, or in Terminal run
  `bash ~/Downloads/YouTubeAutomation-Setup.command`.
  (It also works on Linux: `bash YouTubeAutomation-Setup.command`.)
- **Windows 10/11:** `YouTubeAutomation-Setup.cmd`. Double-click it. If
  SmartScreen warns, choose More info → Run anyway.

The installer:

1. checks for Docker Desktop (and offers to install it with Homebrew or winget),
2. writes the project to `~/youtube-automation` (`%USERPROFILE%\youtube-automation` on Windows),
3. creates `.env` with random secrets and asks for your Anthropic API key,
4. installs the `notebooklm` command and opens a browser so you can sign in to
   NotebookLM once (the login is saved to `~/.notebooklm`),
5. copies the `youtube-automation` skill to `~/.claude/skills/` for Claude Code,
6. starts n8n and the bridge with `docker compose`, imports the workflow, and
   opens http://localhost:5678.

Re-running the installer is safe. It updates the code and keeps your settings,
n8n account and saved credentials. On Mac, `--relogin` signs in to NotebookLM
again; on Windows, delete `%USERPROFILE%\.notebooklm\profiles\default\storage_state.json`
and re-run.

## Google credentials (one time, about 5 minutes)

n8n needs an OAuth client of your own to read Drive and upload to YouTube.

1. Go to https://console.cloud.google.com/ and create a project
   (for example "YouTube automation").
2. **APIs & Services → Library**: enable **Google Drive API** and
   **YouTube Data API v3**.
3. **APIs & Services → OAuth consent screen** (Google Auth Platform):
   user type **External**, fill in the app name and your email, and add your
   Google account under **Test users** (Audience).
4. **Credentials → Create credentials → OAuth client ID**, type
   **Web application**, and add this authorized redirect URI:
   `http://localhost:5678/rest/oauth2-credential/callback`
   Copy the Client ID and Client secret.
5. In n8n, open the workflow and:
   - **New file in Drive folder** → Credential → *Create new* → Google Drive
     OAuth2 API → paste the ID and secret → *Sign in with Google*. Then choose
     the folder to watch. Select the same credential in **Download file**.
   - **Upload to YouTube** → Credential → *Create new* → YouTube OAuth2 API →
     paste the same ID and secret → *Sign in with Google*, choosing the account
     or brand account that owns your channel.
6. Activate (publish) the workflow with the toggle at the top.

While the consent screen is in **Testing**, Google expires the sign-in after
7 days. For a personal setup, set the app to **In production** (Audience →
Publish app); you will see an "unverified app" warning when signing in, which
is expected for your own app.

### Why uploads are private

Google keeps videos uploaded through a new, unaudited API project **private**,
whatever privacy you request. That matches this project's default
(`YOUTUBE_PRIVACY=private`): you review each video in YouTube Studio and make it
public there. To publish directly, apply for a
[YouTube API audit](https://support.google.com/youtube/contact/yt_api_form) and
then set `YOUTUBE_PRIVACY=public` in `.env`. The default API quota also caps how
many videos you can upload per day; see Quotas in the Google Cloud console.

## Using it

Put a PDF, Google Doc, Google Slides deck, Word file, text or Markdown file in
the watched folder. Within 5 minutes n8n picks it up; the video usually takes
10-30 minutes to render. Follow progress in n8n's **Executions** tab, or:

```bash
cd ~/youtube-automation
docker compose logs -f bridge
```

Each job's files are kept in the bridge's data volume: the source, NotebookLM's
`brief.txt`, the metadata, and the MP4 until it has been uploaded.

## Settings

Edit `.env` in the install folder, then run `docker compose up -d` there.

| Setting | Default | Meaning |
|---|---|---|
| `YOUTUBE_PRIVACY` | `private` | `private`, `unlisted` or `public` (see above). |
| `YOUTUBE_REGION` | `US` | Region used to validate the category. |
| `YOUTUBE_NOTIFY_SUBSCRIBERS` | `false` | Notify subscribers on upload. |
| `VIDEO_FORMAT` | `explainer` | `explainer`, `brief`, `short` (vertical) or `cinematic` (needs Google AI Ultra). |
| `VIDEO_STYLE` | `auto` | `classic`, `whiteboard`, `kawaii`, `anime`, `watercolor`, `retro-print`, `heritage`, `paper-craft`. |
| `VIDEO_LANGUAGE` | `en` | Narration language code. |
| `VIDEO_INSTRUCTIONS` | empty | Extra direction for every video, e.g. "for beginners, upbeat". |
| `DELETE_NOTEBOOK_AFTER` | `false` | Delete each NotebookLM notebook once its video is downloaded. |
| `CLAUDE_MODEL` | `claude-opus-5-5` | Model that writes the metadata. |
| `CLAUDE_EFFORT` | `medium` | `low` … `max`: how hard Claude thinks. |
| `CLAUDE_FALLBACKS` | `default` | If the model declines, the API retries on a fallback model. `off` disables. |

To change *how* titles and descriptions are written, edit
`skills/youtube-automation/SKILL.md`. The bridge reads it on every job, so no
restart is needed.

## Day-to-day commands

Run these in the install folder:

| | |
|---|---|
| Stop | `docker compose down` |
| Start | `docker compose up -d` |
| Logs | `docker compose logs -f` |
| Bridge status | `curl http://localhost:8765/health` |
| Update | Run the newest installer again |
| Remove everything | `docker compose down -v`, then delete the folder and `~/.notebooklm` |

## The Claude Code skill

The same `youtube-automation` skill is installed into `~/.claude/skills/`. In
Claude Code you can ask, for example, "make a YouTube video from
./whitepaper.pdf and write the metadata", and Claude will follow the skill's
manual steps with the `notebooklm` CLI and show you the metadata before
anything is uploaded.

## Troubleshooting

- **Job error mentions authentication / cookies.** The NotebookLM login
  expired. Re-run the installer with `--relogin` (Mac) or delete
  `storage_state.json` and re-run (Windows), then `docker compose restart bridge`.
  The bridge refreshes the login every 6 hours to make this rare.
- **Job error mentions a rate limit or quota.** NotebookLM's daily Video
  Overview limit was reached. Drop the file in again tomorrow.
- **`Claude API error 401`.** Check `ANTHROPIC_API_KEY` in `.env`, then
  `docker compose up -d`.
- **YouTube node: `Request had insufficient authentication scopes` or
  `quotaExceeded`.** Reconnect the YouTube credential, or wait for the daily
  quota to reset.
- **Nothing happens when a file is added.** Make sure the workflow is active,
  the right folder is selected, and the file was *created* in that folder
  (moving an old file in may not count as new).

## Good to know

- NotebookLM has no public API for personal accounts. The bridge uses
  [notebooklm-py](https://github.com/teng-lin/notebooklm-py), an unofficial
  client that talks to the same service as the website. It works well, but a
  change on Google's side can break it until the library is updated (update
  the version in `bridge/requirements.txt` and re-run the installer).
- Secrets stay on your computer: `.env` holds your API key, and
  `~/.notebooklm/.../storage_state.json` is a sign-in for your Google account.
  Do not share them. Both ports (5678, 8765) listen on localhost only.
- Videos are AI-generated; the description includes a disclosure line. Check
  YouTube's policy on altered or synthetic content for your channel.

## For developers

```
youtube-automation/
├── docker-compose.yml        n8n + bridge
├── .env.example              every setting, documented
├── bridge/                   FastAPI service: NotebookLM + Claude
├── skills/youtube-automation/SKILL.md
├── n8n/youtube-automation.json
├── installer/                templates + build.py
└── dist/                     built single-file installers
```

After changing any of these files, rebuild the installers with
`python3 installer/build.py`.
