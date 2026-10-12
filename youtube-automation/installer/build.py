#!/usr/bin/env python3
"""Build the single-file installers in dist/.

Embeds every project file (compose file, bridge, skill, n8n workflow, guide)
into the two installer templates as base64, so each installer is one file a
person can download and double-click:

    dist/YouTubeAutomation-Setup.command   macOS (also runs on Linux with bash)
    dist/YouTubeAutomation-Setup.cmd       Windows 10/11

Run after changing any project file:  python3 installer/build.py
"""

from __future__ import annotations

import base64
import datetime
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DIST = ROOT / "dist"
VERSION = "1.0.0"

PAYLOAD = [
    "README.md",
    "docker-compose.yml",
    ".env.example",
    "bridge/Dockerfile",
    "bridge/requirements.txt",
    "bridge/app.py",
    "skills/youtube-automation/SKILL.md",
    "n8n/youtube-automation.json",
]


def notebooklm_version() -> str:
    reqs = (ROOT / "bridge/requirements.txt").read_text()
    return re.search(r"^notebooklm-py\[[^\]]*\]==([\w.]+)$", reqs, re.M).group(1)


def b64(rel: str) -> str:
    data = (ROOT / rel).read_bytes().replace(b"\r\n", b"\n")
    return base64.b64encode(data).decode()


def render(template: str, payload: str) -> str:
    return (
        template.replace("__PAYLOAD__", payload)
        .replace("__VERSION__", VERSION)
        .replace("__BUILD_DATE__", datetime.date.today().isoformat())
        .replace("__NOTEBOOKLM_VERSION__", notebooklm_version())
    )


def main() -> None:
    DIST.mkdir(exist_ok=True)

    unix_payload = "\n".join(f"put '{rel}' '{b64(rel)}'" for rel in PAYLOAD)
    unix = render((ROOT / "installer/setup-unix.sh.tpl").read_text(), unix_payload)
    mac = DIST / "YouTubeAutomation-Setup.command"
    mac.write_text(unix, newline="\n")
    mac.chmod(0o755)

    win_payload = "\n".join(f"Put '{rel}' '{b64(rel)}'" for rel in PAYLOAD)
    win = render((ROOT / "installer/setup-windows.ps1.tpl").read_text(), win_payload)
    # cmd.exe needs CRLF line endings to parse labels and blocks reliably.
    (DIST / "YouTubeAutomation-Setup.cmd").write_text(win, newline="\r\n")

    for f in sorted(DIST.iterdir()):
        print(f"{f.relative_to(ROOT)}  {f.stat().st_size // 1024} KB")


if __name__ == "__main__":
    main()
