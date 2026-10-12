# rovo

## YouTube automation

[`youtube-automation/`](youtube-automation/) turns documents dropped into a Google Drive
folder into narrated NotebookLM Video Overviews, has Claude write the YouTube title,
description and tags, and uploads them to your channel, all orchestrated by a free
self-hosted n8n running in Docker on your own Mac or Windows PC.

One-file installers:

- macOS: [`youtube-automation/dist/YouTubeAutomation-Setup.command`](youtube-automation/dist/YouTubeAutomation-Setup.command)
- Windows: [`youtube-automation/dist/YouTubeAutomation-Setup.cmd`](youtube-automation/dist/YouTubeAutomation-Setup.cmd)

Setup guide: [`youtube-automation/README.md`](youtube-automation/README.md)

## Instagram automation

[`instagram-automation/`](instagram-automation/) is the same idea for Instagram Reels: a
document dropped into Google Drive becomes a vertical NotebookLM video with a
Claude-written caption, which waits on a local review page until you approve
it and is then posted as a Reel. It runs on separate ports, so it can sit
alongside the YouTube stack.

One-file installers:

- macOS: [`instagram-automation/dist/InstagramAutomation-Setup.command`](instagram-automation/dist/InstagramAutomation-Setup.command)
- Windows: [`instagram-automation/dist/InstagramAutomation-Setup.cmd`](instagram-automation/dist/InstagramAutomation-Setup.cmd)

Setup guide: [`instagram-automation/README.md`](instagram-automation/README.md)
