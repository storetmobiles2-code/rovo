#!/usr/bin/env python3
"""Generate optimised per-app PWA icons and header logos from the approved brand marks.

Source marks (docs/brand/, 512x512 transparent PNG):
  - rovo-mark-plate-512.png  -> customer, restaurant, admin
  - rovo-mark-wheel-512.png  -> rider

Per app (web/apps/<app>/public/), per docs/brand/README.md:
  icons/icon-192.png, icons/icon-512.png  ("any" purpose; app-specific background)
  icons/maskable-512.png                  (mark in the 80% safe zone on ink-900)
  favicon-32.png                          (32px)
  logo-header.webp + logo-header.png      (<= 20 KB header logo: mark + "rovo" wordmark)

The ~1 MB raster logo is never copied into a bundle (brand README). Idempotent.
"""
from __future__ import annotations
import sys
from pathlib import Path

try:
    from PIL import Image, ImageDraw, ImageFont, ImageOps
except ImportError:
    sys.exit("Pillow is required: pip install Pillow")

ROOT = Path(__file__).resolve().parents[2]
BRAND = ROOT / "docs" / "brand"
APPS = ROOT / "web" / "apps"

INK = (12, 24, 37, 255)          # #0C1825
INK_SOLID = (12, 24, 37)
WHITE = (255, 255, 255, 255)
FLAME_700 = (194, 65, 12)        # #C2410C accessible brand

# app -> (mark file, icon background, "monochrome white mark" flag)
APP_CONFIG = {
    "customer":   ("rovo-mark-plate-512.png", "white", False),
    "rider":      ("rovo-mark-wheel-512.png", "white", False),
    "restaurant": ("rovo-mark-plate-512.png", "ink",   False),
    "admin":      ("rovo-mark-plate-512.png", "ink",   True),
}


def load_mark(name: str) -> Image.Image:
    img = Image.open(BRAND / name).convert("RGBA")
    return img


def trim(img: Image.Image) -> Image.Image:
    """Crop to the non-transparent bounding box."""
    bbox = img.split()[3].getbbox()
    return img.crop(bbox) if bbox else img


def to_white(img: Image.Image) -> Image.Image:
    """Recolour opaque pixels to white, keeping the alpha (admin monochrome mark)."""
    alpha = img.split()[3]
    white = Image.new("RGBA", img.size, WHITE)
    white.putalpha(alpha)
    return white


def contain(mark: Image.Image, box: int, scale: float) -> Image.Image:
    """Return `mark` resized to `scale*box` and centred on a transparent `box` canvas."""
    target = max(1, int(box * scale))
    m = trim(mark)
    m = ImageOps.contain(m, (target, target), Image.LANCZOS)
    canvas = Image.new("RGBA", (box, box), (0, 0, 0, 0))
    canvas.alpha_composite(m, ((box - m.width) // 2, (box - m.height) // 2))
    return canvas


def rounded_bg(size: int, color, radius_frac: float = 0.0) -> Image.Image:
    bg = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    if color is None:
        return bg
    draw = ImageDraw.Draw(bg)
    if radius_frac > 0:
        r = int(size * radius_frac)
        draw.rounded_rectangle([0, 0, size - 1, size - 1], radius=r, fill=color)
    else:
        draw.rectangle([0, 0, size, size], fill=color)
    return bg


def make_icon(mark: Image.Image, size: int, bg_name: str, mono: bool, maskable: bool) -> Image.Image:
    m = to_white(mark) if mono else mark
    if maskable:
        # Maskable: ink background, mark within the 80% safe zone, full-bleed square.
        bg = rounded_bg(size, INK, 0.0)
        bg.alpha_composite(contain(m, size, 0.62))
        return bg
    # "any" icon: rounded tile, app background, comfortable padding.
    if bg_name == "white":
        bg = rounded_bg(size, WHITE, 0.22)
    else:
        bg = rounded_bg(size, INK, 0.22)
        if not mono:
            m = m  # plate mark already reads on ink
    bg.alpha_composite(contain(m, size, 0.72))
    return bg


def best_font(px: int) -> ImageFont.FreeTypeFont:
    for path in (
        "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        "/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
    ):
        if Path(path).exists():
            return ImageFont.truetype(path, px)
    return ImageFont.load_default()


def make_logo(mark: Image.Image, mono: bool, on_ink: bool) -> Image.Image:
    """Header logo: mark + 'rovo' wordmark, transparent background, 2x for crispness."""
    h = 96
    m = trim(to_white(mark) if mono else mark)
    m = ImageOps.contain(m, (h, h), Image.LANCZOS)
    font = best_font(64)
    text = "rovo"
    tmp = Image.new("RGBA", (1, 1))
    d = ImageDraw.Draw(tmp)
    tb = d.textbbox((0, 0), text, font=font)
    tw, th = tb[2] - tb[0], tb[3] - tb[1]
    gap = 16
    W = m.width + gap + tw + 8
    canvas = Image.new("RGBA", (W, h), (0, 0, 0, 0))
    canvas.alpha_composite(m, (0, (h - m.height) // 2))
    color = WHITE if (mono or on_ink) else INK_SOLID + (255,)
    d = ImageDraw.Draw(canvas)
    d.text((m.width + gap - tb[0], (h - th) // 2 - tb[1]), text, font=font, fill=color)
    return canvas


def save_png(img: Image.Image, path: Path) -> int:
    path.parent.mkdir(parents=True, exist_ok=True)
    img.save(path, "PNG", optimize=True)
    return path.stat().st_size


def save_webp(img: Image.Image, path: Path, quality: int = 86) -> int:
    path.parent.mkdir(parents=True, exist_ok=True)
    img.save(path, "WEBP", quality=quality, method=6)
    return path.stat().st_size


def main() -> None:
    marks = {name: load_mark(name) for name in {c[0] for c in APP_CONFIG.values()}}
    total = 0
    for app, (mark_file, bg_name, mono) in APP_CONFIG.items():
        mark = marks[mark_file]
        pub = APPS / app / "public"
        on_ink = bg_name == "ink"
        outputs = {
            pub / "icons" / "icon-192.png": make_icon(mark, 192, bg_name, mono, False),
            pub / "icons" / "icon-512.png": make_icon(mark, 512, bg_name, mono, False),
            pub / "icons" / "maskable-512.png": make_icon(mark, 512, bg_name, mono, True),
            pub / "favicon-32.png": make_icon(mark, 32, bg_name, mono, False),
        }
        for path, img in outputs.items():
            total += save_png(img, path)
        # Header logo: WebP (shipped) + PNG fallback, both <= 20 KB.
        logo = make_logo(mark, mono, on_ink)
        wp = save_webp(logo, pub / "logo-header.webp")
        pn = save_png(logo.resize((logo.width // 2, logo.height // 2), Image.LANCZOS), pub / "logo-header.png")
        # favicon.ico (16/32) for broad browser support.
        ico = make_icon(mark, 64, bg_name, mono, False)
        ico.save(pub / "favicon.ico", sizes=[(16, 16), (32, 32), (48, 48)])
        status = "OK" if max(wp, pn) <= 20 * 1024 else "OVER 20 KB!"
        print(f"{app:10s} logo webp={wp/1024:.1f}KB png={pn/1024:.1f}KB  {status}")
        if max(wp, pn) > 20 * 1024:
            sys.exit(f"{app} header logo exceeds 20 KB budget")
    print(f"Brand assets generated. Total PNG icons ~{total/1024:.0f} KB across 4 apps.")


if __name__ == "__main__":
    main()
