"""Relight the daylight photos of the Skoda for the emerald cave and save RGBA cards."""
import cv2, numpy as np, os, sys

os.makedirs('cards', exist_ok=True)


def smoothstep(a, b, x):
    t = np.clip((x - a) / (b - a), 0, 1)
    return t * t * (3 - 2 * t)


def masked_blur(img2d, a, sigma):
    num = cv2.GaussianBlur(img2d * a, (0, 0), sigma)
    den = cv2.GaussianBlur(a, (0, 0), sigma) + 1e-4
    return num / den


def bake(name, flat_sigma_frac=0.07, flat_k=0.55, dim=0.82, bounce=0.5, rim_gain=1.0, rim_w=0.0009):
    im = cv2.imread(f'up/{name}_x4.png', cv2.IMREAD_UNCHANGED)
    a = im[..., 3].astype(np.float32) / 255
    ys, xs = np.where(a > 0.12)
    y0, y1, x0, x1 = ys.min(), ys.max() + 1, xs.min(), xs.max() + 1
    pad = 4
    y0, x0 = max(0, y0 - pad), max(0, x0 - pad)
    y1, x1 = min(im.shape[0], y1 + pad), min(im.shape[1], x1 + pad)
    im = im[y0:y1, x0:x1]
    a = im[..., 3].astype(np.float32) / 255
    rgb = cv2.cvtColor(im[..., :3], cv2.COLOR_BGR2RGB).astype(np.float32) / 255
    H, W = a.shape
    lin = rgb ** 2.2

    # 1) flatten large-scale daylight dapple (foliage shadows) without killing body shading
    L = lin @ np.array([0.2126, 0.7152, 0.0722], np.float32)
    sig = W * flat_sigma_frac
    ref0 = np.percentile(L[a > 0.9], 85)
    paint = a * smoothstep(0.30, 0.55, L / ref0)          # only bright paint pixels vote for illumination
    Lb = masked_blur(L, paint, sig)
    ref = np.percentile(Lb[paint > 0.8], 60)
    gain = np.clip((ref / (Lb + 1e-3)) ** flat_k, 0.7, 1.8)
    lin = lin * gain[..., None]

    # 2) cave grade: contrast + cool emerald shadow tint, keep paint white
    lin = np.clip(lin, 0, 4)
    lin = lin * dim
    lum = (lin @ np.array([0.2126, 0.7152, 0.0722], np.float32))[..., None]
    shadow = np.clip(1 - lum * 2.2, 0, 1)
    lin = lin * (1 - 0.35 * shadow) + shadow * lum * np.array([0.35, 1.0, 0.8], np.float32) * 0.55
    lin = lin * np.array([0.93, 1.0, 0.97], np.float32)

    # 3) rim light from matte edge: emerald from behind/sides, strongest on top edges
    ab = cv2.GaussianBlur(a, (0, 0), W * rim_w * 0.6)
    inside = cv2.erode((a > 0.5).astype(np.uint8), np.ones((3, 3), np.uint8))
    dist = cv2.distanceTransform(inside, cv2.DIST_L2, 5)
    rw = W * rim_w
    rim = np.exp(-dist / rw) * a
    gy, gx = np.gradient(cv2.GaussianBlur(a, (0, 0), W * rim_w))
    ny = -gy / (np.hypot(gx, gy) + 1e-6)  # outward normal y (image coords: down positive)
    top_bias = np.clip(0.45 + 0.9 * (-ny), 0.25, 1.4)  # upward-facing edges catch more
    rim = rim * top_bias
    rimc = np.array([0.20, 1.0, 0.55], np.float32)
    lin = lin + rim[..., None] * rimc * 1.15 * rim_gain

    # 4) floor bounce: emerald glow climbing the lower body from the glowing floor
    yy = np.linspace(0, 1, H, dtype=np.float32)[:, None]
    lowmask = smoothstep(0.55, 1.0, yy) ** 1.3
    lin = lin + (lowmask * (0.10 + 0.20 * (lum[..., 0] ** 0.5)))[..., None] * np.array([0.12, 0.85, 0.45], np.float32) * bounce

    # wide additive rim map for the compositor (scaled per shot so the glow stays readable at every zoom)
    rwide = np.exp(-dist / (W * 0.0030)) * a * top_bias
    rwide = cv2.resize(rwide.astype(np.float32), (W // 2, H // 2), interpolation=cv2.INTER_AREA)
    cv2.imwrite(f'cards/{name}_rim.png', (np.clip(rwide, 0, 1) * 255).astype(np.uint8))
    out = np.clip(lin, 0, 1.6) ** (1 / 2.2)
    out = np.clip(out, 0, 1)
    rgba = np.dstack([cv2.cvtColor((out * 255).astype(np.uint8), cv2.COLOR_RGB2BGR), (a * 255).astype(np.uint8)])
    cv2.imwrite(f'cards/{name}.png', rgba)
    print(name, rgba.shape)
    # quick preview over dark green
    bg = np.zeros((H, W, 3), np.float32) + np.array([0.02, 0.06, 0.04])
    prev = (out * a[..., None] + bg * (1 - a[..., None]))
    cv2.imwrite(f'tmp/card_{name}.jpg', cv2.resize((prev * 255).astype(np.uint8)[..., ::-1], None, fx=0.25, fy=0.25, interpolation=cv2.INTER_AREA))


if __name__ == '__main__':
    P = {'rear34': dict(flat_sigma_frac=0.035, flat_k=0.85), 'front': dict(flat_sigma_frac=0.05, flat_k=0.8), 'side': dict(flat_sigma_frac=0.06, flat_k=0.6)}
    for n in sys.argv[1:]:
        bake(n, **P.get(n, {}))
