"""Low-level compositing toolkit (linear-light float32, premultiplied RGBA layers)."""
import cv2, numpy as np, os, math, functools

cv2.setNumThreads(2)
HERE = os.path.dirname(os.path.abspath(__file__))
DW, DH = 1080, 1920          # design space
RS = 1.0                      # render scale (0.5 -> preview)


def set_scale(s):
    global RS
    RS = s


def CW(): return int(round(DW * RS))
def CH(): return int(round(DH * RS))


# ---------------------------------------------------------------- math helpers
def clamp(x, a=0.0, b=1.0): return np.clip(x, a, b) if isinstance(x, np.ndarray) else max(a, min(b, x))
def lerp(a, b, t): return a + (b - a) * t
def ease_io(t):
    t = clamp(t); return t * t * t * (t * (t * 6 - 15) + 10)
def ease_out(t, p=3.0):
    t = clamp(t); return 1 - (1 - t) ** p
def ease_in(t, p=3.0):
    t = clamp(t); return t ** p
def smooth(a, b, x):
    t = clamp((x - a) / (b - a)); return t * t * (3 - 2 * t)
def back_out(t, s=1.70158):
    t = clamp(t) - 1; return t * t * ((s + 1) * t + s) + 1
def pulse(t, k=8.0):
    """Impact envelope: 1 at t=0 decaying."""
    return math.exp(-k * max(t, 0)) if t >= 0 else 0.0


_SRGB_LUT = (np.arange(256, dtype=np.float32) / 255.0) ** 2.2
_SRGB_LUT16 = (np.arange(65536, dtype=np.float32) / 65535.0) ** 2.2


def to_linear_u8(a):
    return _SRGB_LUT[a]


def to_srgb(x):
    return np.power(np.clip(x, 0, None), 1 / 2.2)


# ---------------------------------------------------------------- asset cache
class Cache:
    def __init__(self, cap=7):
        self.d = {}; self.order = []; self.cap = cap

    def get(self, key, fn):
        if key in self.d:
            self.order.remove(key); self.order.append(key); return self.d[key]
        v = fn(); self.d[key] = v; self.order.append(key)
        while len(self.order) > self.cap:
            k = self.order.pop(0); self.d.pop(k, None)
        return v


_cache = Cache(10)


def plate(name, kind="plates"):
    """Blender plate -> premultiplied linear float32 RGBA at design size (1080x1920)."""
    def load():
        pth = os.path.join(HERE, kind, name + ".png")
        if not os.path.exists(pth):
            print("[warn] missing plate", name, flush=True)
            return np.zeros((DH, DW, 4), np.float32)
        im = cv2.imread(pth, cv2.IMREAD_UNCHANGED)
        if im.shape[1] != DW:
            im = cv2.resize(im, (DW, DH), interpolation=cv2.INTER_AREA if im.shape[1] > DW else cv2.INTER_CUBIC)
        rgb = cv2.cvtColor(im[..., :3], cv2.COLOR_BGR2RGB)
        lin = _SRGB_LUT16[rgb] if rgb.dtype == np.uint16 else _SRGB_LUT[rgb]
        a = (im[..., 3].astype(np.float32) / (65535.0 if im.dtype == np.uint16 else 255.0)) if im.shape[2] == 4 else np.ones(im.shape[:2], np.float32)
        return np.dstack([lin * a[..., None], a]).astype(np.float32)
    return _cache.get(("plate", name), load)


def card_levels(name):
    """uint8 BGRA mip chain for a baked car card."""
    def load():
        im = cv2.imread(os.path.join(HERE, "cards", name + ".png"), cv2.IMREAD_UNCHANGED)
        lv = [im]
        for _ in range(3):
            h, w = lv[-1].shape[:2]
            lv.append(cv2.resize(lv[-1], (w // 2, h // 2), interpolation=cv2.INTER_AREA))
        return lv
    return _cache.get(("card", name), load)


def card_rim(name):
    def load():
        return cv2.imread(os.path.join(HERE, "cards", name + "_rim.png"), cv2.IMREAD_UNCHANGED).astype(np.float32) / 255.0
    return _cache.get(("rim", name), load)


# ---------------------------------------------------------------- canvas ops
def new_canvas():
    return np.zeros((CH(), CW(), 4), np.float32)


def affine(scale, pos, anchor, rot=0.0, sy=1.0):
    """Matrix mapping source px -> canvas px (design space*RS). pos in design px. sy: extra vertical stretch."""
    c, s = math.cos(rot) * scale * RS, math.sin(rot) * scale * RS
    ax, ay = anchor
    px, py = pos[0] * RS, pos[1] * RS
    A = np.array([[c, -s * sy], [s, c * sy]], np.float32)
    t = np.array([px, py], np.float32) - A @ np.array([ax, ay], np.float32)
    return np.array([[A[0, 0], A[0, 1], t[0]], [A[1, 0], A[1, 1], t[1]]], np.float32)


def _bbox_of(M, w, h, margin=2):
    pts = np.array([[0, 0, 1], [w, 0, 1], [w, h, 1], [0, h, 1]], np.float32) @ M.T
    x0, y0 = np.floor(pts.min(0)).astype(int) - margin
    x1, y1 = np.ceil(pts.max(0)).astype(int) + margin
    return max(x0, 0), max(y0, 0), min(x1, CW()), min(y1, CH())


def warp_f(src, M, margin=2):
    """Warp float32 premult RGBA with affine M into bbox patch. returns (patch, x0, y0) or None."""
    h, w = src.shape[:2]
    x0, y0, x1, y1 = _bbox_of(M, w, h, margin)
    if x1 <= x0 or y1 <= y0:
        return None
    M2 = M.copy(); M2[0, 2] -= x0; M2[1, 2] -= y0
    sc = math.sqrt(abs(np.linalg.det(M[:, :2])))
    interp = cv2.INTER_CUBIC if sc > 1.3 else cv2.INTER_LINEAR
    patch = cv2.warpAffine(src, M2, (x1 - x0, y1 - y0), flags=interp, borderMode=cv2.BORDER_CONSTANT, borderValue=0)
    return patch, x0, y0


def _clip_patch(canvas, p, x0, y0):
    H_, W_ = canvas.shape[:2]
    h, w = p.shape[:2]
    cx0, cy0 = max(x0, 0), max(y0, 0)
    cx1, cy1 = min(x0 + w, W_), min(y0 + h, H_)
    if cx1 <= cx0 or cy1 <= cy0: return None
    return p[cy0 - y0:cy1 - y0, cx0 - x0:cx1 - x0], cx0, cy0, cx1, cy1


def over(canvas, patch_xy, opacity=1.0, tint=None, add=0.0):
    if patch_xy is None or canvas is None: return canvas
    p, x0, y0 = patch_xy
    c = _clip_patch(canvas, p, x0, y0)
    if c is None: return canvas
    pp, cx0, cy0, cx1, cy1 = c
    reg = canvas[cy0:cy1, cx0:cx1]
    pp = pp * opacity
    if tint is not None:
        pp = pp * np.array([*tint, 1.0], np.float32)
    occl = pp[..., 3:4] * (1 - add)
    reg[...] = pp + reg * (1 - occl)
    return canvas


def add_rgb(canvas, patch_xy, gain=1.0, tint=(1, 1, 1)):
    if patch_xy is None: return canvas
    p, x0, y0 = patch_xy
    c = _clip_patch(canvas, p, x0, y0)
    if c is None: return canvas
    pp, cx0, cy0, cx1, cy1 = c
    canvas[cy0:cy1, cx0:cx1, :3] += pp[..., :3] * gain * np.asarray(tint, np.float32)
    return canvas


def plate_flipped(name):
    return _cache.get(("plateflip", name), lambda: np.ascontiguousarray(plate(name)[:, ::-1]))


@functools.lru_cache(maxsize=64)
def plate_anchor(name):
    """Base-centre of a plate's alpha (bottom-most opaque row, centroid of bottom rows)."""
    a = plate(name)[..., 3]
    ys, xs = np.where(a > 0.2)
    if len(ys) == 0: return (DW / 2, DH / 2)
    yb = ys.max(); sel = ys > yb - 18
    return (float(xs[sel].mean()), float(yb))


def draw_plate(canvas, name, scale, pos, anchor, rot=0.0, opacity=1.0, blur=0.0, tint=None, bright=1.0, sy=1.0, flip=False, add=0.0):
    src = plate_flipped(name) if flip else plate(name)
    if flip: anchor = (DW - anchor[0], anchor[1])
    M = affine(scale, pos, anchor, rot, sy)
    pt = warp_f(src, M, margin=int(3 * blur * RS) + 2)
    if pt is None: return None
    p, x0, y0 = pt
    if blur > 0.3:
        p = blur_rgba(p, blur * RS)
    if bright != 1.0:
        p = p.copy(); p[..., :3] *= bright
    over(canvas, (p, x0, y0), opacity, tint, add)
    return (p, x0, y0)


def draw_card(canvas, name, scale, pos, anchor_frac=(0.5, 1.0), rot=0.0, opacity=1.0, blur=0.0, tint=None, crop=None,
              return_patch=False, bright=1.0, quad=None, margin=0):
    """Draw baked car card. anchor_frac in card-fraction (default bottom-centre). scale: design px per FULL-RES card px.
    quad: optional 4 dst corner points (design px, TL,TR,BR,BL) -> perspective warp instead of affine."""
    lv = card_levels(name)
    full_h, full_w = lv[0].shape[:2]
    s_eff = scale * RS
    L = 0
    while L < 3 and s_eff * (2 ** (L + 1)) <= 1.15:
        L += 1
    img = lv[L]
    k = 2 ** L
    ih, iw = img.shape[:2]
    if quad is not None:
        dst = np.array(quad, np.float32) * RS
        src_pts = np.array([[0, 0], [iw, 0], [iw, ih], [0, ih]], np.float32)
        Hm = cv2.getPerspectiveTransform(src_pts, dst)
        mg = 2 + int(margin * RS)
        x0, y0 = np.floor(dst.min(0)).astype(int) - mg; x1, y1 = np.ceil(dst.max(0)).astype(int) + mg
        x0, y0, x1, y1 = max(x0, 0), max(y0, 0), min(x1, CW()), min(y1, CH())
        if x1 <= x0 or y1 <= y0: return None
        T = np.array([[1, 0, -x0], [0, 1, -y0], [0, 0, 1]], np.float32)
        a = img[..., 3:4].astype(np.float32) / 255
        pm = np.dstack([to_linear_u8(img[..., 2::-1].copy()) * a, a[..., 0]]).astype(np.float32) if False else None
        rgb = to_linear_u8(cv2.cvtColor(img[..., :3], cv2.COLOR_BGR2RGB))
        src = np.dstack([rgb * a, a[..., 0]]).astype(np.float32)
        patch = cv2.warpPerspective(src, T @ Hm, (x1 - x0, y1 - y0), flags=cv2.INTER_LINEAR)
        if blur > 0.3: patch = blur_rgba(patch, blur * RS)
        if bright != 1.0: patch[..., :3] *= bright
        pt = (patch, x0, y0)
        over(canvas, pt, opacity, tint)
        return pt
    anchor = (anchor_frac[0] * iw, anchor_frac[1] * ih)
    M = affine(scale * k, pos, anchor, rot)       # level px -> canvas
    # crop source to the region that lands on canvas (keeps float conversion small for macro shots)
    Minv = cv2.invertAffineTransform(M)
    corners = np.array([[0, 0, 1], [CW(), 0, 1], [CW(), CH(), 1], [0, CH(), 1]], np.float32) @ Minv.T
    sx0 = int(max(0, math.floor(corners[:, 0].min()) - 4)); sx1 = int(min(iw, math.ceil(corners[:, 0].max()) + 4))
    sy0 = int(max(0, math.floor(corners[:, 1].min()) - 4)); sy1 = int(min(ih, math.ceil(corners[:, 1].max()) + 4))
    if crop is not None:
        pass
    if sx1 <= sx0 or sy1 <= sy0: return None
    sub = img[sy0:sy1, sx0:sx1]
    a = sub[..., 3:4].astype(np.float32) / 255
    rgb = to_linear_u8(cv2.cvtColor(sub[..., :3], cv2.COLOR_BGR2RGB))
    src = np.dstack([rgb * a, a[..., 0]]).astype(np.float32)
    M2 = M.copy(); M2[0, 2] += M[0, 0] * sx0 + M[0, 1] * sy0; M2[1, 2] += M[1, 0] * sx0 + M[1, 1] * sy0
    pt = warp_f(src, M2, margin=int(3 * blur * RS) + 2 + int(margin * RS))
    if pt is None: return None
    p, x0, y0 = pt
    if blur > 0.3: p = blur_rgba(p, blur * RS)
    if bright != 1.0: p = p.copy(); p[..., :3] *= bright
    pt = (p, x0, y0)
    over(canvas, pt, opacity, tint)
    return pt


def rim_patch(name, M_level0, anchor_level0):
    """Warp the wide additive rim map using a level-0 style transform."""
    rim = card_rim(name)  # half-res of level0
    M = M_level0.copy() * 1.0
    # rim map is 1/2 of level 0 -> scale source coords by 2
    M[:, 0] *= 2; M[:, 1] *= 2
    src = np.dstack([rim, rim, rim, rim]).astype(np.float32)
    return warp_f(src, M)


# ---------------------------------------------------------------- blur family
def blur_rgba(p, sigma):
    if sigma < 0.4: return p
    if sigma > 14:   # blur at reduced res for speed
        f = 4 if sigma > 40 else 2
        h, w = p.shape[:2]
        sm = cv2.resize(p, (max(1, w // f), max(1, h // f)), interpolation=cv2.INTER_AREA)
        sm = cv2.GaussianBlur(sm, (0, 0), sigma / f, borderType=cv2.BORDER_CONSTANT)
        return cv2.resize(sm, (w, h), interpolation=cv2.INTER_LINEAR)
    return cv2.GaussianBlur(p, (0, 0), sigma, borderType=cv2.BORDER_CONSTANT)


def motion_blur(img, length, angle_deg=0.0, passes=1):
    """Directional blur on whole canvas (any channels), edge-replicated."""
    L = int(max(1, round(length * RS)))
    if L < 2: return img
    f = 2 if L > 60 else 1
    src = img if f == 1 else cv2.resize(img, (img.shape[1] // f, img.shape[0] // f), interpolation=cv2.INTER_AREA)
    Ls = max(1, L // f)
    k = np.zeros((Ls, Ls), np.float32)
    c = (Ls - 1) / 2
    ang = math.radians(angle_deg)
    cv2.line(k, (int(round(c - math.cos(ang) * c)), int(round(c - math.sin(ang) * c))),
             (int(round(c + math.cos(ang) * c)), int(round(c + math.sin(ang) * c))), 1.0, 1, cv2.LINE_AA)
    k /= k.sum() + 1e-8
    out = cv2.filter2D(src, -1, k, borderType=cv2.BORDER_REPLICATE)
    if f != 1: out = cv2.resize(out, (img.shape[1], img.shape[0]), interpolation=cv2.INTER_LINEAR)
    return out


def radial_zoom_blur(img, amount=0.06, center=(0.5, 0.5), steps=8):
    """Zoom blur: average of progressively scaled copies about centre."""
    h, w = img.shape[:2]
    cx, cy = center[0] * w, center[1] * h
    acc = np.zeros_like(img)
    for i in range(steps):
        s = 1 + amount * (i / (steps - 1) - 0.5)
        M = np.array([[s, 0, cx - s * cx], [0, s, cy - s * cy]], np.float32)
        acc += cv2.warpAffine(img, M, (w, h), flags=cv2.INTER_LINEAR, borderMode=cv2.BORDER_REPLICATE)
    return acc / steps


# ---------------------------------------------------------------- reflection
def reflect_canvas(layers_canvas, yf, strength=0.55, fade=420, ripple=0.0, t=0.0, blur_far=14, tint=(0.55, 0.95, 0.8)):
    """Mirror a premult RGBA canvas about design-space line y=yf, with distance fade, blur and ripple."""
    h, w = layers_canvas.shape[:2]
    yfp = yf * RS
    rows = np.arange(h, dtype=np.float32)
    src_y = 2 * yfp - rows
    valid = (src_y >= 0) & (src_y < h) & (rows > yfp)
    map_y = np.tile(src_y[:, None], (1, w)).astype(np.float32)
    map_x = np.tile(np.arange(w, dtype=np.float32)[None, :], (h, 1))
    if ripple > 0:
        d = (rows - yfp) / max(fade * RS, 1)
        wob = np.sin(rows * 0.045 / RS * 0.9 + t * 3.1)[:, None] * ripple * RS * (0.3 + 0.7 * np.clip(d, 0, 1))[:, None]
        wob2 = np.sin(np.arange(w, dtype=np.float32) * 0.011 / RS + t * 1.7)[None, :] * ripple * 0.5 * RS
        map_x = map_x + wob + wob2
        map_y = map_y + (np.sin(np.arange(w, dtype=np.float32) * 0.02 / RS + t * 2.3)[None, :] * ripple * 0.35 * RS)
    refl = cv2.remap(layers_canvas, map_x, map_y, cv2.INTER_LINEAR, borderMode=cv2.BORDER_CONSTANT)
    dist = np.clip((rows - yfp) / (fade * RS), 0, 1)
    fall = (strength * (1 - dist) ** 1.6) * valid
    # progressive blur: mix sharp with blurred by distance
    bl = blur_rgba(refl, blur_far * RS)
    mixv = np.clip(dist * 1.6, 0, 1)[:, None, None]
    refl = refl * (1 - mixv) + bl * mixv
    refl = refl * fall[:, None, None]
    refl[..., :3] *= np.asarray(tint, np.float32)
    return refl


# ---------------------------------------------------------------- light, fog, particles
def radial_glow(shape_hw, center, radii, color, power=2.0):
    h, w = shape_hw
    yy, xx = np.mgrid[0:h, 0:w].astype(np.float32)
    d = np.sqrt(((xx - center[0] * RS) / (radii[0] * RS)) ** 2 + ((yy - center[1] * RS) / (radii[1] * RS)) ** 2)
    g = np.clip(1 - d, 0, 1) ** power
    return g[..., None] * np.asarray(color, np.float32)


def vignette(h, w, strength=0.55, power=2.2):
    yy, xx = np.mgrid[0:h, 0:w].astype(np.float32)
    d = np.sqrt(((xx - w / 2) / (w / 2)) ** 2 + ((yy - h * 0.52) / (h * 0.6)) ** 2)
    return (1 - strength * np.clip(d, 0, 1.4) ** power)[..., None]


@functools.lru_cache(maxsize=4)
def fbm_tex(seed, size=(640, 360), octaves=7):
    rng = np.random.RandomState(seed)
    h, w = size
    acc = np.zeros((h, w), np.float32); amp = 1.0; tot = 0
    for o in range(octaves):
        cells = 3 * 2 ** o                       # 3,6,12,24,48,96,192 cells across
        n = rng.rand(max(2, int(cells * h / w)), cells).astype(np.float32)
        n = cv2.resize(n, (w, h), interpolation=cv2.INTER_CUBIC)
        acc += n * amp; tot += amp; amp *= 0.62
    acc /= tot
    acc = (acc - acc.min()) / (acc.max() - acc.min())
    return acc


def fog_layer(t, seed=1, speed=(14, 3), density=0.5, horizon=830, floor_y=1306, color=(0.10, 0.55, 0.38), scale=1.0, height_fall=420):
    h, w = CH(), CW()
    tex = fbm_tex(seed)
    th, tw = tex.shape
    # scroll through the tiled texture
    ox = (t * speed[0]) / 1080 * tw * 0.5; oy = (t * speed[1]) / 1920 * th * 0.5
    M = np.array([[tw / w * 0.8 / scale, 0, ox], [0, th / h * 0.8 / scale, oy]], np.float32)
    big = cv2.warpAffine(tex, M, (w, h), flags=cv2.INTER_LINEAR | cv2.WARP_INVERSE_MAP, borderMode=cv2.BORDER_WRAP)
    yy = np.arange(h, dtype=np.float32)[:, None] / RS
    # more fog hugging the floor / horizon band
    band = np.exp(-((yy - (horizon + 150)) / height_fall) ** 2) * 0.9 + 0.12
    m = np.clip((big - 0.42) * 1.7, 0, 1) ** 1.4 * band * density * 0.45
    return m[..., None] * np.asarray(color, np.float32)


def motes(t, n=70, seed=3, drift=(6, -14), depth_spread=1.0, color=(0.55, 1.0, 0.8), size=(1.2, 4.5), gain=1.0, cam_shift=(0, 0), focus_blur=1.8):
    """Floating dust / spore bokeh. Returns additive RGB canvas."""
    h, w = CH(), CW()
    rng = np.random.RandomState(seed)
    layer = np.zeros((h, w, 3), np.float32)
    big = np.zeros((h, w, 3), np.float32)
    for i in range(n):
        z = rng.rand() ** 1.6 * depth_spread
        x0, y0 = rng.rand() * DW, rng.rand() * DH
        sp = 0.4 + 1.6 * z
        x = (x0 + drift[0] * t * sp + cam_shift[0] * (0.4 + z) + 25 * math.sin(t * (0.6 + rng.rand()) + i)) % DW
        y = (y0 + drift[1] * t * sp + cam_shift[1] * (0.4 + z)) % DH
        r = (size[0] + (size[1] - size[0]) * z) * 0.8
        tw = 0.55 + 0.45 * math.sin(t * (2 + 3 * rng.rand()) + i * 1.7)
        v = gain * tw * (0.35 + 0.65 * z)
        tgt = big if z > 0.72 else layer
        cv2.circle(tgt, (int(x * RS), int(y * RS)), max(1, int(r * RS * (2.2 if z > 0.72 else 1))), tuple(float(c * v) for c in color), -1, cv2.LINE_AA)
    layer = cv2.GaussianBlur(layer, (0, 0), focus_blur * RS)
    big = cv2.GaussianBlur(big, (0, 0), 6 * RS)
    return layer + big * 0.5


def band_mask(h, w, pos, width, angle_deg, soft=1.0):
    """Diagonal light-sweep band (design px). pos moves along the normal."""
    yy, xx = np.mgrid[0:h, 0:w].astype(np.float32)
    a = math.radians(angle_deg)
    d = ((xx / RS) * math.cos(a) + (yy / RS) * math.sin(a)) - pos
    return np.exp(-(d / width) ** 2 * soft)


# ---------------------------------------------------------------- post
def bloom(img, thresh=0.75, gain=0.55, radii=(6, 18, 52), tint=(0.85, 1.0, 0.95)):
    lum = img[..., :3].max(2, keepdims=True)
    hi = np.clip(img[..., :3] - thresh, 0, None) * (lum > thresh)
    out = np.zeros_like(hi)
    h, w = hi.shape[:2]
    small = cv2.resize(hi, (w // 4, h // 4), interpolation=cv2.INTER_AREA)
    for r in radii:
        out += cv2.resize(cv2.GaussianBlur(small, (0, 0), r * RS / 4), (w, h), interpolation=cv2.INTER_LINEAR) / len(radii)
    return img[..., :3] + out * gain * np.asarray(tint, np.float32) * 3.0


def anamorphic(img, thresh=0.9, gain=0.35, length=260, tint=(0.35, 1.0, 0.8)):
    hi = np.clip(img[..., :3].max(2) - thresh, 0, None)[..., None] * img[..., :3]
    h, w = hi.shape[:2]
    small = cv2.resize(hi, (w // 4, h // 4), interpolation=cv2.INTER_AREA)
    s = cv2.GaussianBlur(small, (int(length * RS / 4) | 1, 1), 0)
    s = cv2.resize(s, (w, h), interpolation=cv2.INTER_LINEAR)
    return img[..., :3] + s * gain * np.asarray(tint, np.float32) * 4


def chroma(img, amount=0.004, center=(0.5, 0.5)):
    if amount <= 0: return img
    h, w = img.shape[:2]
    cx, cy = center[0] * w, center[1] * h
    out = img.copy()
    for ch, s in ((0, 1 + amount), (2, 1 - amount)):
        M = np.array([[s, 0, cx - s * cx], [0, s, cy - s * cy]], np.float32)
        out[..., ch] = cv2.warpAffine(img[..., ch], M, (w, h), flags=cv2.INTER_LINEAR, borderMode=cv2.BORDER_REPLICATE)
    return out


def grade(img, exposure=1.0, contrast=1.12, teal=0.10, sat=1.05, lift=0.004):
    """Linear in -> sRGB out. Cool emerald shadows, protected whites."""
    x = np.clip(img, 0, None) * exposure + lift
    lum = (x @ np.array([0.2126, 0.7152, 0.0722], np.float32))[..., None]
    sh = np.clip(1 - lum * 3.0, 0, 1)
    x = x + sh * np.array([-0.002, 0.008, 0.006], np.float32) * (teal * 10)
    # filmic-ish shoulder (keeps highlights colour)
    x = x / (1 + 0.18 * x)
    x = x * 1.18
    s = to_srgb(np.clip(x, 0, 1))
    s = (s - 0.5) * contrast + 0.5
    g = (s @ np.array([0.299, 0.587, 0.114], np.float32))[..., None]
    s = g + (s - g) * sat
    return np.clip(s, 0, 1)


def grain(img, amount=0.018, seed=0):
    rng = np.random.RandomState(seed)
    n = rng.randn(img.shape[0] // 2 + 1, img.shape[1] // 2 + 1).astype(np.float32)
    n = cv2.resize(n, (img.shape[1], img.shape[0]), interpolation=cv2.INTER_LINEAR)
    lum = img.mean(2, keepdims=True)
    return np.clip(img + n[..., None] * amount * (0.4 + (1 - np.abs(lum - 0.4))), 0, 1)


def text_layer(txt, font, size, tracking=0, color=(1, 1, 1), anchor="mm", pos=None, canvas_wh=None, blur=0):
    from PIL import Image, ImageDraw, ImageFont
    W, H = canvas_wh or (DW, DH)
    im = Image.new("L", (W, H), 0)
    d = ImageDraw.Draw(im)
    f = ImageFont.truetype(font, size)
    # manual tracking
    widths = [d.textlength(ch, font=f) for ch in txt]
    total = sum(widths) + tracking * (len(txt) - 1)
    x = (pos[0] - total / 2) if anchor == "mm" else pos[0]
    for ch, wd in zip(txt, widths):
        d.text((x, pos[1]), ch, font=f, fill=255, anchor="lm")
        x += wd + tracking
    a = np.asarray(im, np.float32) / 255.0
    return a, total
