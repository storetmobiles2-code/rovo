# ŠKODA Superb — Crystal Cave edit (V01)

14 s · 1080×1920 (9:16) · 30 fps · H.264 + AAC (−14.1 LUFS) · 90 BPM, one impact per beat (20 frames).

`SKODA_SUPERB_CRYSTAL_CAVE_V01_2026-10-05.mp4` is the deliverable. It follows the structure of the reference
(black-void reveal → crystal cave → whip-pan cuts between hero and macro shots → hero finale), without the tutorial
segment, and uses only your Superb photos (TS 09 EF 6141).

## Timeline
| Frames | Beat | Shot |
|---|---|---|
| 0–40 | 0–2 | Void: scan-line reveal of the car, crystals sprout, crown grows |
| 40–80 | 2–4 | Cave impact: front hero, crown + foreground crystals, light sweep |
| 80–140 | 4–7 | Macros: number plate → rear tail-lamp → headlamp flare |
| 140–180 | 7–9 | Front 3/4 orbit hero |
| 180–220 | 9–11 | Side slide-in with echo trail |
| 220–240 | 11–12 | Mirror/handle pan |
| 240–280 | 12–14 | Rear 3/4 pull-back |
| 280–320 | 14–16 | Crystal macro → wheel macro |
| 320–380 | 16–19 | Finale side hero, crown glints, reflection |
| 380–420 | 19–21 | Grille push-in → ŠKODA / SUPERB lockup → fade |

## Open-source toolchain
Blender 5 (`bpy`, Cycles + OIDN) · Real-ESRGAN via `spandrel`/PyTorch · rembg (IS-Net) · OpenCV/NumPy/SciPy (2.5D compositor,
bloom, whip blur, reflections) · Pillow · FFmpeg · fonts Anton & Rajdhani (SIL OFL). Music and SFX are synthesised in NumPy/SciPy.

## Rebuild (`source/`)
1. `rembg` cut-outs → `upscale.py` (Real-ESRGAN x4) → `bake_cards.py side rear34 front` (relight for the cave)
2. `plates.py all` — renders the Blender cave / crystal plates (registered camera, transparent)
3. `render_all.py 1.0 frames 0 420 4` — compositor (`shots.py` = timeline, `fx.py` = toolkit)
4. `audio.py` — score + sound design, then mux with FFmpeg (`libx264 -crf 15`, `aac`).

## Honest notes
* The three car views are **2.5D photo cards** (photos of your car, matted and relit), not a 3D model, so the "orbit" is limited to ~±20°.
* Car colour is preserved (white); only lighting/grade is applied. Plate text is your real plate.
* The wrap-design collage and the Slavia photo were not used (different car/generation from the Superb in the other photos).
