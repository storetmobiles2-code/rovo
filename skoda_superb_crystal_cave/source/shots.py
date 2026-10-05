"""Shot timeline + frame renderer for the Skoda Superb crystal-cave edit.
90 BPM -> 1 beat = 20 frames @30fps.  420 frames (14.0 s).
"""
import sys, os, math, time
import numpy as np, cv2, functools
import fx
from fx import *

FPS = 30
BEAT = 20
TOTAL = 420
FLOOR = 1306          # screen y of the car plane's floor-contact line in the registered plate camera
FONT_TITLE = os.path.join(HERE, "fonts", "Anton-Regular.ttf")
FONT_SUB = os.path.join(HERE, "fonts", "Rajdhani-SemiBold.ttf")
EMERALD = (0.10, 0.95, 0.50)


def card_wh(name):
    h, w = card_levels(name)[0].shape[:2]
    return w, h


# --------------------------------------------------------------------- camera model
class Cam:
    def __init__(self, zoom=1.0, pan=(0.0, 0.0), shake=(0.0, 0.0), roll=0.0):
        self.zoom, self.pan, self.shake, self.roll = zoom, pan, shake, roll


def xf(cam, depth, pos, scale):
    z = cam.zoom ** depth
    x = 540 + (pos[0] - 540) * z + (cam.pan[0] + cam.shake[0]) * depth
    y = 960 + (pos[1] - 960) * z + (cam.pan[1] + cam.shake[1]) * depth
    return (x, y), scale * z


def shake_at(f, impacts, amp=16.0, decay=7.0):
    sx = sy = 0.0
    for f0 in impacts:
        if f >= f0:
            t = (f - f0) / FPS
            e = amp * math.exp(-decay * t)
            sx += e * math.sin(t * 61 + f0); sy += e * 0.8 * math.cos(t * 53 + f0 * 1.3)
    return (sx, sy)


def idle_shake(f, amp=2.2):
    t = f / FPS
    return (amp * (math.sin(t * 1.9) + 0.5 * math.sin(t * 4.3 + 1)), amp * (math.cos(t * 1.4) + 0.5 * math.sin(t * 3.7)))


# --------------------------------------------------------------------- light on the car
def edge_light(p, color=EMERALD, sigma_in=2.6, sigma_out=12.0, gain=1.0, top_bias=0.9):
    """Rim light derived from the *placed* matte, so width stays right at every zoom."""
    a = p[..., 3]
    s_in, s_out = sigma_in * fx.RS, sigma_out * fx.RS
    ab = cv2.GaussianBlur(a, (0, 0), s_in)
    inner = np.clip(a - ab, 0, 1) * 2.0
    gy = np.gradient(cv2.GaussianBlur(a, (0, 0), s_in * 1.5), axis=0)
    gx = np.gradient(cv2.GaussianBlur(a, (0, 0), s_in * 1.5), axis=1)
    ny = -gy / (np.hypot(gx, gy) + 1e-5)
    bias = np.clip(0.55 + top_bias * (-ny), 0.25, 1.6)
    outer = cv2.GaussianBlur(a, (0, 0), s_out) * (1 - a)
    rim = (inner * bias * 0.95 + outer * 0.16)
    p[..., :3] += rim[..., None] * np.asarray(color, np.float32) * gain
    return p


def side_bounce(p, x0, y0, gain=1.0, color=(0.12, 1.0, 0.5)):
    """Emerald light bouncing off the crystals onto the paint from both flanks + faint top-down falloff."""
    a = p[..., 3]
    cols = np.where(a.max(0) > 0.5)[0]
    if len(cols) < 8: return p
    xl, xr = cols.min(), cols.max()
    wd = max(xr - xl, 1)
    xs = np.arange(p.shape[1], dtype=np.float32)
    g = np.exp(-(xs - xl) / (0.20 * wd)) + np.exp(-(xr - xs) / (0.20 * wd))
    lum = np.clip(p[..., :3].mean(2) / (a + 1e-3), 0, 1.2)
    p[..., :3] += (g[None, :] * a * (0.10 + lum) * 0.42 * gain)[..., None] * np.asarray(color, np.float32)
    rows = np.where(a.max(1) > 0.5)[0]
    if len(rows) > 8:
        yt, yb = rows.min(), rows.max()
        v = np.clip((np.arange(p.shape[0], dtype=np.float32) - yt) / max(yb - yt, 1), 0, 1)
        p[..., :3] *= (1.0 - 0.18 * v ** 1.5)[:, None, None]
    return p


def sweep(p, x0, y0, pos, width, angle, gain=0.9, color=(0.75, 1.0, 0.9)):
    h, w = p.shape[:2]
    yy, xx = np.mgrid[0:h, 0:w].astype(np.float32)
    ang = math.radians(angle)
    d = ((xx + x0) / fx.RS) * math.cos(ang) + ((yy + y0) / fx.RS) * math.sin(ang) - pos
    band = np.exp(-(d / width) ** 2)
    lum = p[..., :3].mean(2)
    p[..., :3] += (band * p[..., 3] * (0.25 + np.clip(lum / (p[..., 3] + 1e-3), 0, 1.2)))[..., None] * np.asarray(color, np.float32) * gain
    return p


def scan_reveal(p, x0, y0, scan_y, dark=0.05, edge=24.0, color=EMERALD):
    """Photo is revealed below scan line; above it the car is a dark silhouette. Bright emerald line at the front."""
    h, w = p.shape[:2]
    yy = (np.arange(h, dtype=np.float32)[:, None] + y0) / fx.RS
    m = smooth(scan_y - 6, scan_y + 6, yy)          # 1 below the line (revealed), 0 above
    m = np.clip(m, 0, 1)
    body = dark + (1 - dark) * m
    line = np.exp(-((yy - scan_y) / edge) ** 2)
    p[..., :3] *= body[..., None]
    p[..., :3] += (line * p[..., 3])[..., None] * np.asarray(color, np.float32) * 2.2
    return p


def lens_flare(canvas_rgb, pos, gain, length=520, color=(0.3, 1.0, 0.85)):
    """Horizontal anamorphic streak + core at design pos."""
    if gain <= 0.001: return
    h, w = canvas_rgb.shape[:2]
    cx, cy = pos[0] * fx.RS, pos[1] * fx.RS
    yy, xx = np.mgrid[0:h, 0:w].astype(np.float32)
    streak = np.exp(-(((xx - cx) / (length * fx.RS)) ** 2 + ((yy - cy) / (3.0 * fx.RS)) ** 2)) * 1.4
    core = np.exp(-(((xx - cx) ** 2 + (yy - cy) ** 2) / (46.0 * fx.RS) ** 2)) * 1.1
    halo = np.exp(-(((xx - cx) ** 2 + (yy - cy) ** 2) / (170.0 * fx.RS) ** 2)) * 0.25
    g = (streak + core + halo)[..., None]
    canvas_rgb += g * np.asarray(color, np.float32) * gain
    canvas_rgb += (core * 1.0)[..., None] * gain * 0.8


def star(canvas_rgb, pos, gain, size=90, color=(0.75, 1.0, 0.9)):
    if gain <= 0.01: return
    h, w = canvas_rgb.shape[:2]
    cx, cy = pos[0] * fx.RS, pos[1] * fx.RS
    r = int(size * 3 * fx.RS)
    x0, x1 = max(int(cx) - r, 0), min(int(cx) + r, w); y0, y1 = max(int(cy) - r, 0), min(int(cy) + r, h)
    if x1 <= x0 or y1 <= y0: return
    yy, xx = np.mgrid[y0:y1, x0:x1].astype(np.float32)
    dx, dy = (xx - cx) / (fx.RS), (yy - cy) / (fx.RS)
    arm = np.exp(-(np.abs(dy) / 2.2)) * np.exp(-np.abs(dx) / (size * 0.5)) + np.exp(-(np.abs(dx) / 2.2)) * np.exp(-np.abs(dy) / (size * 0.5))
    d2 = (dx + dy) / 1.414; e2 = (dx - dy) / 1.414
    diag = 0.45 * (np.exp(-np.abs(e2) / 2.0) * np.exp(-np.abs(d2) / (size * 0.25)) + np.exp(-np.abs(d2) / 2.0) * np.exp(-np.abs(e2) / (size * 0.25)))
    core = np.exp(-(dx * dx + dy * dy) / (size * 0.07) ** 2) * 1.2
    canvas_rgb[y0:y1, x0:x1] += ((arm + diag + core) * gain)[..., None] * np.asarray(color, np.float32)


@functools.lru_cache(maxsize=16)
def plate_tip(name):
    a = plate(name)[..., 3]
    ys, xs = np.where(a > 0.5)
    i = ys.argmin()
    return (float(xs[i]), float(ys[i]))


def beat_pulse(f, start=0, decay=3.2, offset=2):
    v = 0.0
    for b in range(start, TOTAL + 1, BEAT):
        if f >= b + offset:
            v += math.exp(-(f - b - offset) / decay) if f - b < 40 else 0
    return v


# --------------------------------------------------------------------- the stage
def yaw_quad(base, width, height, yaw_deg, focal_mult=2.4):
    th = math.radians(yaw_deg)
    f = focal_mult * width
    pts = []
    for sx, sh in ((-0.5, 1), (0.5, 1), (0.5, 0), (-0.5, 0)):
        x = sx * width
        xr = x * math.cos(th)
        z = x * math.sin(th)
        p = f / (f + z)
        pts.append((base[0] + xr * p, base[1] - sh * height * p))
    return pts


def stage(f, cfg, cam, t):
    h, w = CH(), CW()
    C = new_canvas()
    R = new_canvas()      # reflective objects (crystals behind + car)
    Rf = new_canvas()     # near foreground crystals (own reflection line)

    # ---- background
    bg = cfg.get("bg")
    if bg:
        pos, sc = xf(cam, cfg.get("bg_depth", 0.45), (540, 960), cfg.get("bg_scale", 1.12))
        draw_plate(C, bg, sc, pos, (540, 960), blur=cfg.get("bg_blur", 0), bright=cfg.get("bg_bright", 1.0), flip=cfg.get("bg_flip", False))
    elif cfg.get("void_floor", False):
        yy = np.arange(h, dtype=np.float32)[:, None] / fx.RS
        floor_m = smooth(FLOOR - 40, FLOOR + 400, yy)
        C[..., :3] += (floor_m * 0.006)[..., None] * np.array([0.3, 1, 0.7], np.float32)
        C[..., 3] = 1

    car = cfg.get("car")
    car_base = None
    # ---- floor glow & contact shadow (under car)
    if car and car.get("name"):
        cpos, csc = xf(cam, car.get("depth", 1.0), car["base"], 1.0)
        cw = car["width"] * cam.zoom ** car.get("depth", 1.0)
        car_base = cpos
        gg = cfg.get("floor_glow", 1.0)
        if gg > 0:
            C[..., :3] += radial_glow((h, w), (cpos[0], cpos[1] + 20), (cw * 1.05, 120), (0.015, 0.26, 0.12), 2.0) * gg
        sh = radial_glow((h, w), (cpos[0], cpos[1] - 6), (cw * 0.55, 20), (1, 1, 1), 1.0)[..., 0]
        C[..., :3] *= (1 - 0.85 * sh)[..., None]

    # ---- reflective set: back crystals
    for item in cfg.get("back", []):
        nm = item["name"]
        pos, sc = xf(cam, item.get("depth", 0.85), item["pos"], item.get("scale", 1.0))
        anc = item.get("anchor", plate_anchor(nm) if item.get("use_base") else (540, 960))
        g = item.get("grow", None)
        sy = 1.0; scl = sc
        if g is not None:
            if g <= 0: continue
            gy = back_out(g) if g < 1 else 1.0
            sxm = 0.55 + 0.45 * ease_out(g)
            scl = sc * sxm; sy = max(gy, 0.002) / sxm
        draw_plate(R, nm, scl, pos, anc, blur=item.get("blur", 0), bright=item.get("bright", 1.0), flip=item.get("flip", False), sy=sy, opacity=item.get("opacity", 1.0))

    # ---- the car
    car_patch = None
    if car and car.get("name"):
        d = car.get("depth", 1.0)
        fw, fh = card_wh(car["name"])
        scale = car["width"] / fw
        pos, sc = xf(cam, d, car["base"], scale)
        afr = car.get("anchor_frac", (0.5, 1.0))
        if car.get("yaw") is not None:
            wpx = car["width"] * cam.zoom ** d
            hpx = wpx * fh / fw
            quad = yaw_quad(pos, wpx, hpx, car["yaw"])
            pt = draw_card(None, car["name"], sc, pos, afr, quad=quad, blur=car.get("blur", 0), margin=90)
        else:
            pt = draw_card(None, car["name"], sc, pos, afr, rot=car.get("rot", 0.0), blur=car.get("blur", 0), return_patch=True, margin=90)
        if pt is not None:
            p, x0, y0 = pt
            if car.get("bottom_fade"):
                yy = (np.arange(p.shape[0], dtype=np.float32)[:, None] + y0) / fx.RS
                p *= (1 - smooth(car["bottom_fade"][0], car["bottom_fade"][1], yy))[..., None]
            if car.get("scan") is not None:
                scan_reveal(p, x0, y0, car["scan"])
            if car.get("bounce", 1.0) > 0:
                side_bounce(p, x0, y0, car.get("bounce", 1.0))
            rg = car.get("rim", 1.0) * 0.62
            if rg > 0:
                edge_light(p, gain=rg, sigma_out=car.get("rim_out", 12.0))
            for sw in car.get("sweeps", []):
                sweep(p, x0, y0, sw["pos"], sw.get("width", 60), sw.get("angle", 70), sw.get("gain", 0.9))
            if car.get("tint") is not None:
                p[..., :3] *= np.asarray(car["tint"], np.float32)
            p[..., :3] *= car.get("bright", 1.0) * 1.0
            if car.get("blur", 0) < 1.0:   # crisp up the upscaled photo (unsharp on premult colour)
                bl = cv2.GaussianBlur(p[..., :3], (0, 0), 1.3 * fx.RS + 0.3)
                p[..., :3] = np.clip(p[..., :3] + (p[..., :3] - bl) * 0.55, 0, None)
            car_patch = (p, x0, y0)
            # ghost echoes (motion trail)
            for (dx, dy, op) in car.get("ghosts", []):
                g = (p.copy(), x0 + int(dx * fx.RS), y0 + int(dy * fx.RS))
                gp = g[0]
                gp[..., :3] *= np.array([0.25, 1.0, 0.65], np.float32)
                over(R, g, op, add=0.6)
            over(R, car_patch, car.get("opacity", 1.0))

    # ---- mid crystals (in front of car but still on floor)
    for item in cfg.get("mid", []):
        nm = item["name"]
        pos, sc = xf(cam, item.get("depth", 1.1), item["pos"], item.get("scale", 1.0))
        anc = item.get("anchor", plate_anchor(nm))
        g = item.get("grow", None)
        sy = 1.0; scl = sc
        if g is not None:
            if g <= 0: continue
            gy = back_out(g) if g < 1 else 1.0
            sxm = 0.55 + 0.45 * ease_out(g)
            scl = sc * sxm; sy = max(gy, 0.002) / sxm
        draw_plate(R, nm, scl, pos, anc, blur=item.get("blur", 0), bright=item.get("bright", 1.0), flip=item.get("flip", False), sy=sy)

    # ---- reflections
    yf = car_base[1] if car_base is not None else FLOOR * 1.0
    if cfg.get("reflect", 0.5) > 0:
        refl = reflect_canvas(R, yf, strength=cfg.get("reflect", 0.5), fade=cfg.get("reflect_fade", 380), ripple=cfg.get("ripple", 2.5), t=t, blur_far=cfg.get("reflect_blur", 9))
        over(C, (refl, 0, 0))
    over(C, (R, 0, 0))

    # ---- atmosphere between car and foreground
    for fl in cfg.get("fog", []):
        C[..., :3] += fog_layer(t, **fl)

    # ---- foreground (blurred, big crystals / shards) with their own floor reflection
    for item in cfg.get("fg", []):
        nm = item["name"]
        pos, sc = xf(cam, item.get("depth", 1.5), item["pos"], item.get("scale", 1.0))
        anc = item.get("anchor", plate_anchor(nm) if item.get("use_base") else (540, 960))
        tgt = Rf if item.get("reflect", False) else C
        draw_plate(tgt, nm, sc, pos, anc, blur=item.get("blur", 8), bright=item.get("bright", 1.0), flip=item.get("flip", False), rot=item.get("rot", 0.0), opacity=item.get("opacity", 1.0))
    if Rf.any():
        yfg = cfg.get("fg_floor", 1368) * 1.0
        refl2 = reflect_canvas(Rf, yfg, strength=0.4, fade=380, ripple=3.0, t=t, blur_far=18)
        over(C, (refl2, 0, 0)); over(C, (Rf, 0, 0))

    rgb = C[..., :3]
    # ---- additive light layers
    if cfg.get("motes", True):
        rgb += motes(t, n=cfg.get("motes_n", 60), seed=cfg.get("motes_seed", 3), gain=cfg.get("motes_gain", 0.9), cam_shift=(cam.pan[0] * 0.4, cam.pan[1] * 0.4))
    for sp in cfg.get("sparkles", []):
        star(rgb, sp["pos"], sp["gain"], sp.get("size", 90))
    for fl in cfg.get("flares", []):
        lens_flare(rgb, fl["pos"], fl["gain"], fl.get("length", 520), fl.get("color", (0.3, 1.0, 0.85)))
    return rgb


# --------------------------------------------------------------------- SHOTS
# each shot fn(f,u) -> (cfg, cam, post)   (u = 0..1 within the shot, f global frame)
S = {}


def single_layout(grow_t0, i_range=range(12), k=0.97, dur=16, stagger=1.3):
    """Ring of individual crystals; returns (back_items, mid_items)."""
    back, mid = [], []
    order = [3, 2, 8, 9, 4, 1, 5, 0, 6, 7, 10, 11]
    for rank, i in enumerate(order):
        nm = f"single_{i:02d}"
        ax, ay = plate_anchor(nm)
        px = 540 + (ax - 540) * k
        item = dict(name=nm, pos=(px, ay + 0), anchor=(ax, ay), scale=1.22, depth=1.12 if abs(ax - 540) > 260 else 0.95)
        item["_rank"] = rank
        (mid if abs(ax - 540) > 330 else back).append(item)
    return back, mid


def ring_static():
    """Outer floor crystals, fully grown, for hero shots."""
    _, mid = single_layout(0)
    for it in mid:
        it.pop("_rank", None); it["grow"] = 1.0
    return mid


def post_default(**k):
    d = dict(exposure=1.0, bloom=0.55, chroma=0.0016, flash=0.0, fade=0.0, whip=None, grain=0.016, contrast=1.1, vig=0.5)
    d.update(k); return d


# --- S1 : the void --------------------------------------------------------------------------
def shot_void(f, u):
    t = f / FPS
    g0 = 6
    back, mid = single_layout(0)
    for it in back + mid:
        r = it.pop("_rank")
        it["grow"] = clamp((f - (g0 + r * 1.9)) / 15.0)
    z = lerp(1.0, 1.14, ease_io(u))
    cam = Cam(zoom=z, shake=idle_shake(f, 1.5))
    scan = lerp(FLOOR + 10, FLOOR - 640, ease_io(clamp((f - 3) / 24)))
    car = dict(name="front", width=720, base=(540, FLOOR), depth=1.0, scan=scan if f < 30 else None, rim=lerp(1.6, 0.9, ease_io(clamp((f - 24) / 12))))
    if f >= 30: car["scan"] = FLOOR - 700
    back.insert(0, dict(name="crown", pos=(540, 1290), anchor=(540, 1290), scale=1.12, depth=0.9, grow=clamp((f - 16) / 20.0), bright=1.0))
    cfg = dict(void_floor=True, car=car, back=back, mid=mid, reflect=0.5, fog=[], motes=True, motes_n=30, floor_glow=lerp(0.3, 1.6, ease_io(clamp((f - 4) / 30))))
    return cfg, cam, post_default(bloom=0.7, fade=1 - smooth(0, 5, f))


def crown_sparkles(cam, f, scale, depth, flip=False, pos=(540, 1290), anchor=(540, 1290), extra=()):
    tip = plate_tip("crown")
    tx = (DW - tip[0]) if flip else tip[0]
    anc = (DW - anchor[0], anchor[1]) if flip else anchor
    p, sc = xf(cam, depth, pos, scale)
    sx = p[0] + (tx - anc[0]) * sc; sy = p[1] + (tip[1] - anc[1]) * sc
    g = 0.25 + 0.9 * beat_pulse(f)
    out = [dict(pos=(sx, sy + 6), gain=g, size=130)]
    for dx, dy, k in extra:
        out.append(dict(pos=(sx + dx * sc, sy + dy * sc), gain=g * k, size=70))
    return out


# --- S2 : cave hero (front) -------------------------------------------------------------------
def shot_hero_front(f, u):
    t = f / FPS
    lf = f - 40
    z = lerp(1.34, 1.0, ease_out(lf / 22.0, 3)) * lerp(1.0, 1.06, ease_io(u))
    cam = Cam(zoom=z, shake=shake_at(f, [40], 20, 8))
    lfade = ease_out(clamp(lf / 14.0))
    car = dict(name="front", width=800, base=(540, FLOOR), depth=1.0, rim=0.9,
               sweeps=[dict(pos=lerp(-100, 1250, ease_io(clamp((f - 54) / 18.0))), width=70, angle=62, gain=1.0)])
    cfg = dict(bg="cave_a", bg_bright=2.4, car=car,
               back=[dict(name="crown", pos=(540, 1290), anchor=(540, 1290), scale=1.42, depth=0.8, bright=1.0)],
               mid=ring_static(), sparkles=crown_sparkles(cam, f, 1.42, 0.8, extra=((-140, 120, 0.5), (130, 160, 0.45))),
               fg=[dict(name="cl_left", pos=(540 - lerp(520, 0, ease_out(lf / 16)), 1368), anchor=(540, 1368), scale=1.0, depth=1.4, blur=6, reflect=True),
                   dict(name="cl_right", pos=(540 + lerp(520, 0, ease_out(lf / 16)), 1368), anchor=(540, 1368), scale=1.0, depth=1.4, blur=6, reflect=True)],
               fog=[dict(seed=1, speed=(18, 2), density=0.55), dict(seed=2, speed=(-10, 4), density=0.3, scale=1.4)],
               reflect=0.5, motes_n=70, floor_glow=1.2)
    return cfg, cam, post_default(flash=0.0, exposure=lerp(1.35, 1.0, ease_out(lf / 8.0)), chroma=0.0016 + 0.012 * pulse(lf / FPS, 10))


S["void"] = (0, 40, shot_void)
S["hero_front"] = (40, 80, shot_hero_front)


# --- S3 : plate macro ---------------------------------------------------------------------------
def shot_plate(f, u):
    lf = f - 80
    t = f / FPS
    z = lerp(1.0, 1.16, ease_io(u))
    cam = Cam(zoom=1.0, shake=idle_shake(f, 2.0) if lf > 4 else (0, 0))
    sc = lerp(1.02, 1.22, ease_io(u))
    fw, fh = card_wh("front")
    blur = lerp(10, 0, ease_out(lf / 7.0))
    car = dict(name="front", width=fw * sc, base=(540, 1010), anchor_frac=(0.372, 0.805), depth=1.0, rim=0.8, blur=blur,
               sweeps=[dict(pos=lerp(-100, 1200, ease_io(clamp((lf - 4) / 14.0))), width=55, angle=75, gain=1.2)])
    cfg = dict(bg="cave_b", bg_bright=2.2, bg_blur=14, bg_scale=1.5, car=car, back=[], mid=[],
               fg=[dict(name="shard_00", pos=(lerp(1500, -700, ease_in(clamp(lf / 9.0), 1.6)), 1050), scale=2.2, depth=1.0, blur=26, rot=0.3)],
               fog=[dict(seed=2, speed=(30, 6), density=0.35)], reflect=0.0, floor_glow=0, motes_n=40)
    return cfg, cam, post_default(bloom=0.5)


S["plate"] = (80, 100, shot_plate)


# --- S4 : tail-lamp macro (rear 3/4) -----------------------------------------------------------
def shot_tail(f, u):
    lf = f - 100
    fw, fh = card_wh("rear34")
    sc = lerp(0.62, 0.74, ease_io(u))
    afr = (0.095, 0.865)
    cam = Cam(shake=idle_shake(f, 2.5))
    car = dict(name="rear34", width=fw * sc, base=(lerp(300, 620, ease_io(u)), lerp(1180, 1040, ease_io(u))), anchor_frac=afr, rot=lerp(-0.12, -0.05, u), rim=0.7,
               blur=lerp(8, 0, ease_out(lf / 6.0)),
               sweeps=[dict(pos=lerp(1200, -100, ease_io(clamp((lf - 3) / 15.0))), width=70, angle=110, gain=0.9)])
    cfg = dict(bg="cave_a", bg_bright=1.8, bg_blur=18, bg_scale=1.6, bg_flip=True, car=car, back=[], mid=[],
               fg=[dict(name="shard_01", pos=(lerp(-600, 1700, ease_in(clamp((lf - 8) / 12.0), 1.4)), 900), scale=2.0, depth=1.0, blur=30, rot=-0.4)],
               fog=[dict(seed=1, speed=(-26, 3), density=0.4)], reflect=0, floor_glow=0, motes_n=40, motes_seed=9)
    return cfg, cam, post_default(bloom=0.6)


S["tail"] = (100, 120, shot_tail)


# --- S5 : headlamp macro + flare ---------------------------------------------------------------
def shot_lamp(f, u):
    lf = f - 120
    fw, fh = card_wh("front")
    sc = lerp(1.35, 1.62, ease_io(u))
    cam = Cam(shake=idle_shake(f, 2.0))
    car = dict(name="front", width=fw * sc, base=(lerp(600, 560, u), lerp(940, 900, u)), anchor_frac=(0.795, 0.585), rot=lerp(0.06, -0.02, u), rim=0.7,
               blur=lerp(9, 0, ease_out(lf / 6.0)))
    flare = ease_io(clamp((lf - 8) / 10.0)) * (1 - ease_in(clamp((lf - 18) / 3)))
    cfg = dict(bg="cave_b", bg_bright=2.0, bg_blur=16, bg_scale=1.5, car=car, back=[], mid=[],
               fg=[], fog=[dict(seed=2, speed=(22, 4), density=0.35)], reflect=0, floor_glow=0, motes_n=50, motes_seed=5,
               flares=[dict(pos=(car["base"][0] - 6, car["base"][1] + 4), gain=flare * 1.1, length=620)])
    return cfg, cam, post_default(bloom=0.7)


S["lamp"] = (120, 140, shot_lamp)


# --- S6 : front 3/4 orbit hero ------------------------------------------------------------------
def shot_orbit(f, u):
    lf = f - 140
    t = f / FPS
    yaw = lerp(-24, 10, ease_io(u))
    cam = Cam(zoom=lerp(1.16, 1.0, ease_out(clamp(lf / 14.0))) * lerp(1.0, 1.05, u), pan=(lerp(60, -40, ease_io(u)), 0), shake=shake_at(f, [140], 12, 9))
    car = dict(name="front", width=830, base=(540, FLOOR), yaw=yaw, depth=1.0, rim=0.95,
               sweeps=[dict(pos=lerp(1300, -100, ease_io(clamp((f - 150) / 22.0))), width=65, angle=115, gain=1.0)])
    cfg = dict(bg="cave_b", bg_bright=2.3, bg_flip=True, car=car,
               back=[dict(name="crown", pos=(540, 1290), anchor=(540, 1290), scale=1.3, depth=0.75, flip=True)],
               mid=ring_static(), sparkles=crown_sparkles(cam, f, 1.3, 0.75, flip=True), fg=[dict(name="cl_left", pos=(540 + 30, 1368), anchor=(540, 1368), scale=1.05, depth=1.7, blur=7, reflect=True, flip=True),
                           dict(name="cl_right", pos=(540 - 20, 1368), anchor=(540, 1368), scale=1.0, depth=1.6, blur=8, reflect=True, flip=True)],
               fog=[dict(seed=1, speed=(20, 2), density=0.5), dict(seed=2, speed=(-12, 4), density=0.28, scale=1.3)], reflect=0.5, motes_n=70, motes_seed=11, floor_glow=1.2)
    return cfg, cam, post_default(chroma=0.0016 + 0.009 * pulse((f - 140) / FPS, 10))


S["orbit"] = (140, 180, shot_orbit)


# --- S7 : side slide w/ echoes -------------------------------------------------------------------
def shot_slide(f, u):
    lf = f - 180
    t = f / FPS
    fw, fh = card_wh("side")
    x = lerp(1650, 540, back_out(clamp(lf / 17.0), 1.1))
    vel = (x - lerp(1650, 540, back_out(clamp((lf - 1) / 17.0), 1.1)))
    ghosts = []
    if lf < 22:
        for i, op in enumerate((0.40, 0.24, 0.12)):
            ghosts.append((-(i + 1) * clamp(abs(vel) * 5.0, 0, 330), 0, op))
    cam = Cam(zoom=lerp(1.0, 1.06, ease_io(clamp((lf - 16) / 24.0))), shake=shake_at(f, [196], 9, 9))
    car = dict(name="side", width=960, base=(x, FLOOR), depth=1.0, rim=1.0, ghosts=ghosts,
               sweeps=[dict(pos=lerp(-100, 1250, ease_io(clamp((lf - 22) / 16.0))), width=70, angle=68, gain=1.0)])
    cfg = dict(bg="cave_a", bg_bright=2.3, car=car,
               back=[dict(name="crown", pos=(540 + (x - 540) * 0.25, 1290), anchor=(540, 1290), scale=1.1, depth=0.8)],
               sparkles=crown_sparkles(cam, f, 1.1, 0.8, pos=(540 + (x - 540) * 0.25, 1290)),
               fg=[dict(name="cl_left", pos=(540 - 40 - (x - 540) * 0.6, 1368), anchor=(540, 1368), scale=1.0, depth=1.5, blur=8, reflect=True),
                   dict(name="shard_02", pos=(lerp(-500, 1700, ease_in(clamp(lf / 10.0), 1.3)), 1000), scale=2.0, depth=1.0, blur=34, rot=0.5, opacity=1.0)],
               fog=[dict(seed=1, speed=(18, 2), density=0.5)], reflect=0.5, motes_n=60, motes_seed=2, floor_glow=1.2)
    return cfg, cam, post_default(chroma=0.0016 + 0.008 * pulse((f - 196) / FPS, 10))


S["slide"] = (180, 220, shot_slide)


# --- S8 : mirror/handle pan --------------------------------------------------------------------------
def shot_handle(f, u):
    lf = f - 220
    fw, fh = card_wh("side")
    sc = lerp(0.52, 0.60, ease_io(u))
    afr = (lerp(0.285, 0.485, ease_io(u)), lerp(0.28, 0.435, ease_io(u)))
    cam = Cam(shake=idle_shake(f, 2.2))
    car = dict(name="side", width=fw * sc, base=(540, 980), anchor_frac=afr, rot=lerp(0.05, -0.03, u), rim=0.7, blur=lerp(8, 0, ease_out(lf / 6.0)),
               sweeps=[dict(pos=lerp(-100, 1200, ease_io(clamp((lf - 3) / 14.0))), width=60, angle=70, gain=1.1)])
    cfg = dict(bg="cave_b", bg_bright=2.0, bg_blur=14, bg_scale=1.5, car=car, back=[], mid=[],
               fg=[dict(name="shard_03", pos=(lerp(1600, -600, ease_in(clamp((lf - 6) / 11.0), 1.3)), 1000), scale=2.0, depth=1.0, blur=30, rot=0.2)],
               fog=[dict(seed=2, speed=(26, 4), density=0.3)], reflect=0, floor_glow=0, motes_n=40, motes_seed=14)
    return cfg, cam, post_default(bloom=0.6)


S["handle"] = (220, 240, shot_handle)


# --- S9 : rear 3/4 pull-back ------------------------------------------------------------------------
def shot_rear(f, u):
    lf = f - 240
    fw, fh = card_wh("rear34")
    z = lerp(1.2, 1.0, ease_out(clamp(lf / 30.0), 2.2))
    cam = Cam(zoom=z, pan=(lerp(-70, 50, ease_io(u)), 0), shake=shake_at(f, [240], 11, 9))
    car = dict(name="rear34", width=1220, base=(540, 1215), anchor_frac=(0.5, 1.0), rot=lerp(-0.09, -0.035, ease_io(u)), depth=1.0, rim=1.0,
               bottom_fade=(1090, 1250),
               sweeps=[dict(pos=lerp(1300, -100, ease_io(clamp((lf - 8) / 24.0))), width=70, angle=105, gain=1.0)])
    cfg = dict(bg="cave_a", bg_bright=2.4, bg_flip=True, car=car,
               back=[dict(name="crown", pos=(540, 1290), anchor=(540, 1290), scale=1.55, depth=0.7, flip=False)],
               sparkles=crown_sparkles(cam, f, 1.55, 0.7),
               fg=[dict(name="cl_right", pos=(540 - 10, 1368), anchor=(540, 1368), scale=1.05, depth=1.7, blur=7, reflect=True),
                   dict(name="cl_left", pos=(540 + 10, 1368), anchor=(540, 1368), scale=1.02, depth=1.6, blur=9, reflect=True)],
               mid=ring_static(),
               fog=[dict(seed=1, speed=(-16, 2), density=1.1, horizon=1150, height_fall=300), dict(seed=2, speed=(10, 3), density=0.5, scale=1.3, horizon=1100)], reflect=0.0, motes_n=70, motes_seed=6, floor_glow=0.5)
    return cfg, cam, post_default(chroma=0.0016 + 0.009 * pulse((f - 240) / FPS, 10))


S["rear"] = (240, 280, shot_rear)


# --- S10 : crystal macro -------------------------------------------------------------------------------
def shot_cmacro(f, u):
    lf = f - 280
    cam = Cam(shake=idle_shake(f, 2.5))
    sc = lerp(2.6, 3.4, ease_io(u))
    cfg = dict(bg="cave_b", bg_bright=1.8, bg_blur=22, bg_scale=1.5,
               back=[dict(name="crown", pos=(lerp(430, 640, ease_io(u)), 1080), anchor=(610, 735), scale=sc, depth=1.0, blur=lerp(10, 0, ease_out(lf / 6.0)))],
               fg=[dict(name="shard_04", pos=(lerp(-500, 1800, ease_in(clamp((lf - 5) / 12.0), 1.3)), 1000), scale=2.2, depth=1.0, blur=34, rot=0.3)],
               fog=[dict(seed=2, speed=(20, 5), density=0.35)], reflect=0, floor_glow=0, motes_n=60, motes_seed=21, motes_gain=1.2)
    return cfg, cam, post_default(bloom=0.8)


S["cmacro"] = (280, 300, shot_cmacro)


# --- S11 : wheel macro ------------------------------------------------------------------------------------
def shot_wheel(f, u):
    lf = f - 300
    fw, fh = card_wh("side")
    sc = lerp(1.0, 1.2, ease_io(u))
    cam = Cam(shake=idle_shake(f, 2.2))
    car = dict(name="side", width=fw * sc, base=(540, 1000), anchor_frac=(lerp(0.15, 0.145, u), 0.775), depth=1.0, rim=0.6, blur=lerp(9, 0, ease_out(lf / 6.0)),
               sweeps=[dict(pos=lerp(1200, -100, ease_io(clamp((lf - 3) / 14.0))), width=60, angle=100, gain=1.3)])
    cfg = dict(bg="cave_a", bg_bright=1.8, bg_blur=18, bg_scale=1.6, car=car, back=[], fg=[], mid=[],
               fog=[dict(seed=1, speed=(-30, 3), density=0.4)], reflect=0, floor_glow=0, motes_n=45, motes_seed=17)
    return cfg, cam, post_default(bloom=0.65)


S["wheel"] = (300, 320, shot_wheel)


# --- S12 : finale hero (side) -----------------------------------------------------------------------------------
def shot_finale(f, u):
    lf = f - 320
    cam = Cam(zoom=lerp(1.22, 1.0, ease_out(clamp(lf / 18.0), 3)) * lerp(1.0, 1.07, ease_io(u)), shake=shake_at(f, [320], 18, 8))
    car = dict(name="side", width=990, base=(540, FLOOR), depth=1.0, rim=1.0,
               sweeps=[dict(pos=lerp(-100, 1250, ease_io(clamp((lf - 14) / 22.0))), width=75, angle=66, gain=0.55),
                       dict(pos=lerp(-100, 1250, ease_io(clamp((lf - 38) / 18.0))), width=45, angle=66, gain=0.4)])
    cfg = dict(bg="cave_a", bg_bright=2.4, car=car,
               back=[dict(name="crown", pos=(540, 1290), anchor=(540, 1290), scale=1.12, depth=0.8)],
               sparkles=crown_sparkles(cam, f, 1.12, 0.8, extra=((-150, 100, 0.5), (140, 150, 0.45))),
               fg=[dict(name="cl_left", pos=(540 - lerp(500, 0, ease_out(lf / 18.0)), 1368), anchor=(540, 1368), scale=1.0, depth=1.45, blur=6, reflect=True),
                   dict(name="cl_right", pos=(540 + lerp(500, 0, ease_out(lf / 18.0)), 1368), anchor=(540, 1368), scale=1.0, depth=1.45, blur=6, reflect=True)],
               fog=[dict(seed=1, speed=(16, 2), density=0.6), dict(seed=2, speed=(-10, 4), density=0.3, scale=1.4)], reflect=0.5, motes_n=80, motes_seed=4, floor_glow=1.3)
    return cfg, cam, post_default(flash=0.0, chroma=0.0016 + 0.012 * pulse((f - 320) / FPS, 10), bloom=0.65)


S["finale"] = (320, 380, shot_finale)


# --- S13 : push-in on the badge + title -----------------------------------------------------------------------------
def shot_title(f, u):
    lf = f - 380
    fw, fh = card_wh("front")
    sc = lerp(1.9, 2.5, ease_io(u))
    cam = Cam(shake=idle_shake(f, 1.5))
    dim = lerp(0.9, 0.07, smooth(0, 12, lf))
    car = dict(name="front", width=fw * sc, base=(540, 900), anchor_frac=(0.372, 0.56), depth=1.0, rim=0.5, blur=lerp(2, 26, ease_in(u, 1.2)), bright=dim)
    cfg = dict(bg="cave_b", bg_bright=1.4, bg_blur=20, bg_scale=1.5, car=car, back=[], fg=[], mid=[],
               fog=[dict(seed=2, speed=(14, 3), density=0.35)], reflect=0, floor_glow=0, motes_n=50, motes_seed=8)
    fade = smooth(30, 40, lf)
    return cfg, cam, post_default(bloom=0.6, fade=fade)


S["title"] = (380, 420, shot_title)


# --------------------------------------------------------------------- global FX tables
CUTS = [80, 100, 120, 140, 180, 220, 240, 280, 300, 320, 380]
FLASHES = {40: 0.55, 140: 0.07, 180: 0.05, 240: 0.07, 320: 0.30, 280: 0.07}
WHIP_DIR = {80: 0, 100: 15, 120: 175, 140: 0, 180: 0, 220: 0, 240: 160, 280: 0, 300: 190, 320: 90, 380: 0}


def whip_params(f):
    """3 frames before and after each cut: directional blur + zoom to hide the cut."""
    for c in CUTS:
        if c - 3 <= f < c:
            k = (f - (c - 3) + 1) / 3.0
            return dict(angle=WHIP_DIR[c], length=lerp(20, 130, k), zoom=0.04 * k)
        if c <= f < c + 3:
            k = 1 - (f - c) / 3.0
            return dict(angle=WHIP_DIR[c], length=lerp(0, 130, k), zoom=0.04 * k * 0.6)
    return None


def flash_at(f):
    v = 0.0
    for f0, amp in FLASHES.items():
        if f >= f0:
            v += amp * math.exp(-(f - f0) / 1.7)
    return v


def title_overlay(rgb, f):
    lf = f - 380
    if lf < 8: return rgb
    a_in = ease_out(clamp((lf - 8) / 14.0))
    # reveal: letters track in from wide
    tr = lerp(60, 14, ease_out(clamp((lf - 8) / 20.0), 3))
    mask, tw = text_layer("ŠKODA", FONT_TITLE, 206, tracking=tr, pos=(540, 900))
    mask2, tw2 = text_layer("SUPERB", FONT_SUB, 82, tracking=lerp(60, 34, ease_out(clamp((lf - 14) / 20.0))), pos=(540, 1050))
    a1 = cv2.resize(mask, (CW(), CH()), interpolation=cv2.INTER_AREA)
    a2 = cv2.resize(mask2, (CW(), CH()), interpolation=cv2.INTER_AREA)
    # emerald glint sweeping through the wordmark
    xx = (np.arange(CW(), dtype=np.float32)[None, :]) / fx.RS
    glint = np.exp(-((xx - lerp(-100, 1200, ease_io(clamp((lf - 16) / 20.0)))) / 80.0) ** 2)
    txt = np.array([0.93, 1.0, 0.97], np.float32)
    col = a1[..., None] * txt * a_in * (0.85 + 0.6 * glint[..., None] * np.array([0.1, 0.9, 0.5], np.float32) * 2.0)
    col2 = a2[..., None] * np.array([0.30, 1.0, 0.60], np.float32) * ease_out(clamp((lf - 14) / 12.0)) * 0.9
    # hairline between
    line = np.zeros((CH(), CW(), 1), np.float32)
    lw = int(lerp(0, 360, ease_out(clamp((lf - 12) / 18.0))) * fx.RS)
    if lw > 1:
        y = int(992 * fx.RS)
        cv2.line(line, (int(540 * fx.RS) - lw // 2, y), (int(540 * fx.RS) + lw // 2, y), 1.0, max(1, int(2 * fx.RS)), cv2.LINE_AA)
    col3 = line * np.array([0.3, 1.0, 0.6], np.float32) * 0.8
    glow = cv2.GaussianBlur((col + col2 + col3), (0, 0), 14 * fx.RS) * 0.9
    return rgb + col + col2 + col3 + glow


# --------------------------------------------------------------------- frame renderer
def find_shot(f):
    for name, (a, b, fn) in S.items():
        if a <= f < b:
            return name, a, b, fn
    raise ValueError(f)


def render_frame(f):
    name, a, b, fn = find_shot(f)
    u = (f - a) / max(1, (b - a - 1))
    cfg, cam, post = fn(f, u)
    t = f / FPS
    rgb = stage(f, cfg, cam, t)

    # ---- post (linear)
    h, w = rgb.shape[:2]
    rgb = bloom(rgb, thresh=0.8, gain=post["bloom"] * 0.7)
    rgb = anamorphic(rgb, thresh=0.95, gain=0.22, length=300)
    wp = whip_params(f)
    if wp:
        rgb = motion_blur(rgb, wp["length"], wp["angle"])
        rgb = radial_zoom_blur(rgb, wp["zoom"], steps=6)
    if post["chroma"] > 0 or wp:
        rgb = chroma(rgb, post["chroma"] + (0.010 * (wp["length"] / 130) if wp else 0))
    rgb = rgb * vignette(h, w, post["vig"], 2.2) * post["exposure"]
    fl = flash_at(f) + post["flash"]
    if fl > 0.005:
        rgb = rgb * (1.0 + fl * 2.6) + fl * np.array([0.12, 0.45, 0.30], np.float32) * 0.9
    if name == "title":
        rgb = title_overlay(rgb, f)
    out = grade(rgb, contrast=post["contrast"])
    out = grain(out, post["grain"], seed=f)
    if post["fade"] > 0:
        out = out * (1 - post["fade"])
    return out


def save_frame(f, outdir):
    im = render_frame(f)
    cv2.imwrite(os.path.join(outdir, f"f_{f:04d}.png"), cv2.cvtColor((im * 255 + 0.5).astype(np.uint8), cv2.COLOR_RGB2BGR))


if __name__ == "__main__":
    # usage: python shots.py <scale> <outdir> <f0> <f1> [step]
    scale = float(sys.argv[1]); outdir = sys.argv[2]
    f0, f1 = int(sys.argv[3]), int(sys.argv[4]); step = int(sys.argv[5]) if len(sys.argv) > 5 else 1
    fx.set_scale(scale)
    os.makedirs(outdir, exist_ok=True)
    for f in range(f0, f1, step):
        t = time.time(); save_frame(f, outdir); print(f, f"{time.time()-t:.2f}s", flush=True)
