"""Synthesised score + sound design, locked to the 90 BPM / 30 fps edit grid. numpy/scipy only."""
import numpy as np, scipy.signal as sg, scipy.io.wavfile as wf, math, sys

SR = 44100
DUR = 14.0
N = int(SR * DUR)
FPS = 30
rng = np.random.RandomState(7)
L = np.zeros(N, np.float32)
R = np.zeros(N, np.float32)
RVL = np.zeros(N, np.float32)   # reverb send
RVR = np.zeros(N, np.float32)
DUCK = np.ones(N, np.float32)   # sidechain envelope for bass/pad


def t_of(f): return f / FPS


def add(buf_l, buf_r, t0, sig, pan=0.0, gain=1.0, send=0.0):
    i0 = int(t0 * SR)
    if i0 >= N: return
    if i0 < 0:
        sig = sig[-i0:]; i0 = 0
    n = min(len(sig), N - i0)
    gl = math.cos((pan + 1) * math.pi / 4) * gain
    gr = math.sin((pan + 1) * math.pi / 4) * gain
    buf_l[i0:i0 + n] += sig[:n] * gl
    buf_r[i0:i0 + n] += sig[:n] * gr
    if send > 0:
        RVL[i0:i0 + n] += sig[:n] * gl * send
        RVR[i0:i0 + n] += sig[:n] * gr * send


def env(n, a=0.002, d=0.2, curve=4.0):
    t = np.arange(n) / SR
    e = np.minimum(t / max(a, 1e-4), 1.0) * np.exp(-t / max(d, 1e-4) * curve / 4)
    return e.astype(np.float32)


def lp(x, fc, order=2):
    return sg.sosfilt(sg.butter(order, fc, 'low', fs=SR, output='sos'), x).astype(np.float32)


def hp(x, fc, order=2):
    return sg.sosfilt(sg.butter(order, fc, 'high', fs=SR, output='sos'), x).astype(np.float32)


def bp(x, f1, f2, order=2):
    return sg.sosfilt(sg.butter(order, [f1, f2], 'band', fs=SR, output='sos'), x).astype(np.float32)


def noise(n): return rng.randn(n).astype(np.float32)


# ---------------------------------------------------------------- voices
def kick(vel=1.0):
    n = int(0.55 * SR); t = np.arange(n) / SR
    f = 46 + 120 * np.exp(-t * 28)
    ph = 2 * np.pi * np.cumsum(f) / SR
    body = np.sin(ph) * np.exp(-t * 7.5)
    click = hp(noise(n), 2500) * np.exp(-t * 220) * 0.25
    return np.tanh((body * 1.5 + click) * 1.2).astype(np.float32) * vel


def clap(vel=1.0):
    n = int(0.45 * SR); t = np.arange(n) / SR
    x = np.zeros(n, np.float32)
    for d in (0, 0.011, 0.022):
        i = int(d * SR)
        seg = bp(noise(n - i), 900, 4200) * np.exp(-np.arange(n - i) / SR * 90)
        x[i:] += seg
    tail = bp(noise(n), 700, 3000) * np.exp(-t * 14) * 0.5
    return ((x + tail) * 1.3 * vel).astype(np.float32)


def hat(vel=1.0, open_=False):
    n = int((0.22 if open_ else 0.05) * SR)
    x = hp(noise(n), 7500) * np.exp(-np.arange(n) / SR * (22 if open_ else 110))
    return (x * vel).astype(np.float32)


def sub_note(freq, dur, vel=1.0, glide_from=None):
    n = int(dur * SR); t = np.arange(n) / SR
    f = freq if glide_from is None else freq + (glide_from - freq) * np.exp(-t * 14)
    ph = 2 * np.pi * np.cumsum(np.broadcast_to(f, (n,))) / SR
    x = np.sin(ph) + 0.35 * np.sin(2 * ph) + 0.12 * np.sin(3 * ph)
    e = np.minimum(t / 0.01, 1) * np.exp(-t * (1.2 / max(dur, 0.1))) * np.minimum((dur - t) / 0.05, 1)
    return (np.tanh(x * 1.3) * e * vel).astype(np.float32)


def impact(size=1.0):
    n = int(3.2 * SR); t = np.arange(n) / SR
    f = 34 + 70 * np.exp(-t * 6)
    ph = 2 * np.pi * np.cumsum(f) / SR
    boom = np.sin(ph) * np.exp(-t * (1.1 / size))
    thump = lp(noise(n), 160) * np.exp(-t * 4.0) * 1.2
    crack = hp(noise(n), 900) * np.exp(-t * 18) * 0.7
    shimmer = bp(noise(n), 3000, 9000) * np.exp(-t * 3.0) * 0.12
    return (np.tanh((boom * 1.4 + thump + crack) * 1.1) * size + shimmer).astype(np.float32)


def whoosh(dur=0.34, f0=300, f1=7000, rev=False):
    n = int(dur * SR)
    x = noise(n)
    out = np.zeros(n, np.float32)
    steps = 12
    seg = n // steps
    for i in range(steps):
        a = i / (steps - 1)
        fc = f0 * (f1 / f0) ** a
        s = bp(x[i * seg:(i + 1) * seg + 200], fc * 0.6, min(fc * 1.7, SR * 0.45), 2)
        out[i * seg:i * seg + len(s)][:len(out[i * seg:])] += s[:len(out[i * seg:i * seg + len(s)])]
    t = np.linspace(0, 1, n)
    e = np.sin(np.pi * t ** 1.5) ** 1.3 if not rev else t ** 2.2
    return (out * e * 2.2).astype(np.float32)


def riser(dur, f0=180, f1=4200, noise_amt=0.6):
    n = int(dur * SR); t = np.arange(n) / SR; k = t / dur
    f = f0 * (f1 / f0) ** (k ** 1.6)
    ph = 2 * np.pi * np.cumsum(f) / SR
    tone = (np.sin(ph) + 0.5 * np.sin(2 * ph + 1)) * 0.25
    ns = hp(noise(n), 2000) * (k ** 2.5) * noise_amt
    e = k ** 1.8
    return ((tone * 0.6 + ns) * e).astype(np.float32)


def chime(freq, vel=1.0, dur=1.6):
    n = int(dur * SR); t = np.arange(n) / SR
    x = np.zeros(n, np.float32)
    for r, a, d in ((1, 1.0, 1.0), (2.76, 0.55, 0.7), (5.4, 0.3, 0.45), (8.93, 0.18, 0.3)):
        x += np.sin(2 * np.pi * freq * r * t + rng.rand() * 6) * a * np.exp(-t * (3.2 / d))
    x *= np.minimum(t / 0.002, 1)
    return (x * vel * 0.35).astype(np.float32)


def pluck(freq, dur=0.32, vel=1.0, bright=2600):
    n = int(dur * SR); t = np.arange(n) / SR
    x = sg.sawtooth(2 * np.pi * freq * t) * 0.6 + sg.sawtooth(2 * np.pi * freq * 1.006 * t) * 0.5
    fc = bright * np.exp(-t * 10) + 400
    # time-varying lowpass approximated by blending 3 static filters
    a, b, c = lp(x, 500), lp(x, 1400), lp(x, 4200)
    w = np.exp(-t * 9)
    y = a * (1 - w) + (b * np.exp(-t * 5) * 0.5 + c * w * 0.6)
    return (y * np.exp(-t * 7) * vel * 0.5 * np.minimum(t / 0.003, 1)).astype(np.float32)


def pad(freqs, dur, vel=1.0, cutoff=900):
    n = int(dur * SR); t = np.arange(n) / SR
    x = np.zeros(n, np.float32)
    for fq in freqs:
        for det in (-0.004, 0.0, 0.005):
            x += sg.sawtooth(2 * np.pi * fq * (1 + det) * t + rng.rand() * 6).astype(np.float32) * 0.12
    x = lp(x, cutoff, 3)
    e = np.minimum(t / 0.6, 1) * np.minimum((dur - t) / 0.8, 1)
    return (x * e * vel).astype(np.float32)


# ---------------------------------------------------------------- reverb IR
def make_ir(sec=2.4):
    n = int(sec * SR); t = np.arange(n) / SR
    irs = []
    for _ in range(2):
        x = rng.randn(n) * np.exp(-t * 2.6)
        x = lp(x.astype(np.float32), 5200)
        x = hp(x, 160)
        x[:int(0.012 * SR)] *= np.linspace(0, 1, int(0.012 * SR))
        x = x / np.sqrt((x ** 2).sum())
        irs.append(x.astype(np.float32))
    return irs


# ---------------------------------------------------------------- arrangement
BEAT = 20  # frames
Tb = lambda b: (b * BEAT) / FPS

# fixed events (frames)
CUTS = [80, 100, 120, 140, 180, 220, 240, 280, 300, 320, 380]

# --- opening: riser + sub swell into impact at f40
add(L, R, 0.0, riser(t_of(40), 140, 3800, 0.55), 0, 0.5, 0.35)
sw = sub_note(43.65, 1.4, 0.0)
n = int(1.33 * SR); tt = np.arange(n) / SR
swell = (np.sin(2 * np.pi * 43.65 * tt) * (tt / 1.33) ** 2 * 0.7).astype(np.float32)
add(L, R, 0.0, swell, 0, 1.0)
# crystals sprout: rising chimes in F minor pentatonic, one per growth start
notes = [1396.9, 1244.5, 1046.5, 1568.0, 1865.0, 1244.5, 1568.0, 2093.0, 1396.9, 2489.0, 1865.0, 2093.0]
for i, nf in enumerate(notes):
    f_start = 6 + i * 1.9 + 8
    add(L, R, t_of(f_start), chime(nf, 0.7 + 0.03 * i), pan=((i % 5) - 2) * 0.35, gain=0.6, send=0.5)
# tick of the scan line
add(L, R, t_of(3), hp(noise(int(0.9 * SR)), 5000) * np.linspace(0, 0.18, int(0.9 * SR)) ** 1.5, 0, 0.6, 0.3)

# --- impact at cave reveal
add(L, R, t_of(40), impact(1.15), 0, 0.95, 0.35)
add(L, R, t_of(40) - 0.002, kick(1.2), 0, 0.9)

# --- groove from beat 2 to 19
bass_roots = [43.65, 43.65, 51.91, 51.91, 38.89, 38.89, 34.65 * 1.5, 43.65]
for b in range(2, 21):
    t0 = Tb(b)
    if b == 2: pass
    else:
        add(L, R, t0, kick(1.0 if b % 2 == 0 else 0.85), 0, 0.85)
    for i in range(int(0)):
        pass
# duck envelope from kicks
for b in range(2, 21):
    i0 = int(Tb(b) * SR); n = int(0.28 * SR)
    DUCK[i0:i0 + n] *= (1 - 0.75 * np.exp(-np.arange(min(n, N - i0)) / SR * 14))[:max(0, min(n, N - i0))]
# claps on 2 & 4 of each bar (beats 3,5,7,... )
for b in range(3, 20, 2):
    add(L, R, Tb(b), clap(0.9), 0.0, 0.55, 0.45)
# hats: 8ths with swing-ish velocity pattern, 16th rolls before certain cuts
for b in range(3, 20):
    for sub in (0, 0.5):
        v = 0.55 if sub == 0 else 0.35
        add(L, R, Tb(b + sub), hat(v), pan=0.25 if sub else -0.25, gain=0.35)
    if b in (6, 8, 13, 15, 17):
        for k in range(1, 4):
            add(L, R, Tb(b + 0.5 + k * 0.125), hat(0.3 + 0.1 * k), 0.1 * k, 0.3)
add(L, R, Tb(13) + 0.07, hat(0.9, True), 0.3, 0.35, 0.3)
add(L, R, Tb(17) + 0.07, hat(0.9, True), -0.3, 0.35, 0.3)

# bass: sub notes held 2 beats each, ducked by kick
bass = np.zeros(N, np.float32)
prog = [43.65, 43.65, 51.91, 51.91, 38.89, 38.89, 46.25, 46.25, 43.65, 43.65]
for k, b in enumerate(range(2, 20, 2)):
    fq = prog[k % len(prog)]
    s = sub_note(fq, Tb(2), 0.85, glide_from=fq * 1.5)
    i0 = int(Tb(b) * SR); n = min(len(s), N - i0)
    bass[i0:i0 + n] += s[:n]
L += bass * DUCK * 0.9; R += bass * DUCK * 0.9

# dark pad (F minor9 colour) + evolving shimmer
padL = np.zeros(N, np.float32)
chords = [[87.31, 130.81, 155.56], [69.30, 103.83, 138.59], [77.78, 116.54, 155.56], [87.31, 130.81, 207.65]]
for k, b in enumerate(range(2, 20, 4)):
    s = pad(chords[k % 4], Tb(4) + 0.4, 0.55, cutoff=700 + 260 * k)
    i0 = int(Tb(b) * SR); n = min(len(s), N - i0)
    padL[i0:i0 + n] += s[:n]
padd = padL * (0.4 + 0.6 * DUCK)
add(L, R, 0, padd, -0.2, 0.55, 0.35)

# melodic pluck motif (every beat from beat 5), delayed echo in send
motif = [698.46, 830.61, 1046.5, 830.61, 1244.5, 1046.5, 830.61, 622.25]
step = 0
for b in range(5, 19):
    for sub in (0, 0.5):
        if (int(b * 2 + sub * 2) % 3) == 2 and b not in (8, 9): continue
        fq = motif[step % len(motif)] * (0.5 if b < 8 else 1.0)
        step += 1
        add(L, R, Tb(b + sub) + 0.01, pluck(fq, 0.3, 0.6), pan=-0.3 + 0.6 * (step % 2), gain=0.30, send=0.55)

# whooshes into every cut
for i, c in enumerate(CUTS):
    t_c = t_of(c)
    wsh = whoosh(0.36, 350, 7500)
    add(L, R, t_c - 0.36 + 0.02, wsh, pan=-0.6 if i % 2 == 0 else 0.6, gain=0.40, send=0.25)
    add(L, R, t_c, whoosh(0.28, 6000, 500), pan=0.5 if i % 2 == 0 else -0.5, gain=0.22, send=0.2)

# punctuating hits on key cuts
for c, size, g in ((100, 0.5, 0.5), (120, 0.5, 0.5), (140, 0.8, 0.7), (180, 0.9, 0.75), (220, 0.5, 0.5), (240, 0.8, 0.7), (280, 0.7, 0.65), (300, 0.5, 0.5), (320, 1.2, 0.95)):
    add(L, R, t_of(c), impact(size), 0, g, 0.3)
# lamp power-up sweep before headlamp flare (f128..f140)
add(L, R, t_of(124), riser(t_of(16), 400, 5200, 0.2), 0, 0.5, 0.4)
# glass sparkle on the finale impact
for i in range(7):
    add(L, R, t_of(320) + 0.04 * i, chime(2093 * (1.2 ** (i % 4)) * 0.8, 0.7), pan=(i % 3 - 1) * 0.5, gain=0.55, send=0.5)

# finale -> title: reverse swell, sub drop, long tail
add(L, R, t_of(380) - 0.9, riser(0.9, 300, 6500, 0.9), 0, 0.6, 0.45)
add(L, R, t_of(380), impact(1.3), 0, 0.9, 0.5)
add(L, R, t_of(380), chime(1046.5, 1.0, 2.6), -0.2, 0.7, 0.7)
add(L, R, t_of(380) + 0.12, chime(1567.98, 0.8, 2.6), 0.2, 0.6, 0.7)
add(L, R, t_of(380) + 0.30, chime(2093.0, 0.7, 2.6), 0.0, 0.5, 0.7)
add(L, R, t_of(384), pad([87.31, 130.81, 207.65, 261.63], 1.9, 0.6, 1400), 0, 0.5, 0.7)

# ---------------------------------------------------------------- reverb + master
irl, irr = make_ir()
revL = sg.fftconvolve(RVL, irl)[:N].astype(np.float32)
revR = sg.fftconvolve(RVR, irr)[:N].astype(np.float32)
L += revL * 0.9; R += revR * 0.9

# gentle bus glue + soft limiter
def bus(x):
    x = np.tanh(x * 0.95) / np.tanh(0.95)
    return x
L, R = bus(L * 0.42), bus(R * 0.42)
peak = max(np.abs(L).max(), np.abs(R).max())
g = 0.89 / peak
L *= g; R *= g
# fade-out of the tail & tiny fade-in
fo = int(0.9 * SR)
L[-fo:] *= np.linspace(1, 0, fo) ** 1.5; R[-fo:] *= np.linspace(1, 0, fo) ** 1.5
L[:int(0.005 * SR)] *= np.linspace(0, 1, int(0.005 * SR)); R[:int(0.005 * SR)] *= np.linspace(0, 1, int(0.005 * SR))
out = np.stack([L, R], 1)
wf.write('audio.wav', SR, (out * 32767).astype(np.int16))
print('audio ok', out.shape, 'peak', np.abs(out).max(), 'rms', float(np.sqrt((out ** 2).mean())))
