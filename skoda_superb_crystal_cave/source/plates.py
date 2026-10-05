"""Render registered RGBA plates (cave, crystal clusters, single crystals, shards) with Blender Cycles.
usage: python plates.py <plate_name|all> [scale] [samples]
All plates share ONE camera so they register perfectly in the 2D compositor.
World: +Y forward from camera, car plane at y=0, floor z=0.
"""
import sys, time, os, math, random
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from lib import *

OUT = os.path.join(HERE, "plates")
os.makedirs(OUT, exist_ok=True)
W0, H0 = 1080, 1920
CAM = dict(lens=40, loc=(0, -6.2, 0.95), target=(0, 0, 1.75))


def base(scale=1.0, samples=64, transparent=True):
    sc = reset()
    setup_render(int(W0 * scale), int(H0 * scale), samples=samples, motion_blur=False)
    sc.render.film_transparent = transparent
    cam = add_camera(**CAM)
    return sc, cam


def cave_stage(seed=3, with_floor=True):
    rock = mat_rock()
    make_cave_shell(rock)
    make_rocks(rock, seed=seed, count=10, radius=(1.8, 3.6), zone=((-9, 9), (5, 16)))
    if with_floor:
        bpy.ops.mesh.primitive_plane_add(size=160)
        fl = bpy.context.object
        fl.data.materials.append(mat_floor())
    # light the cave: faint cold green rims + deep fill so rock silhouettes read, never flat-lit
    for loc, e, col in [((0, 13, 3.0), 9000, (0.15, 1, 0.5)), ((-6, 9, 2.0), 3200, (0.25, 1, 0.65)), ((6, 9, 2.0), 3200, (0.25, 1, 0.65)), ((0, 2.5, 2.0), 160, (0.2, 1, 0.55))]:
        bpy.ops.object.light_add(type='POINT', location=loc)
        l = bpy.context.object
        l.data.energy = e
        l.data.color = col
        l.data.shadow_soft_size = 2.5
        l.visible_glossy = False
    bpy.ops.object.light_add(type='AREA', location=(0, -1, 9))
    l = bpy.context.object
    l.data.energy = 500
    l.data.size = 10
    l.data.color = (0.5, 0.9, 0.8)
    l.visible_glossy = False


# ------------------------------------------------------------------ plate defs
def p_cave(name="cave_a", seed=3, scale=1.0, samples=64):
    sc, cam = base(scale, samples, transparent=False)
    cave_stage(seed)
    return sc, name


def p_crown(scale=1.0, samples=96):
    sc, cam = base(scale, samples)
    mat = mat_crystal(seed=5)
    make_cluster((0, 2.0, 0), 22, 2.3, 0.9, 2.8, mat, seed=11, tilt=0.55, key_height=3.9, thick=0.26)
    make_cluster((-2.2, 1.4, 0), 9, 0.9, 0.7, 2.0, mat, seed=12, tilt=0.65, thick=0.2)
    make_cluster((2.3, 1.2, 0), 9, 0.9, 0.7, 1.9, mat, seed=13, tilt=0.65, thick=0.2)
    add_softboxes(strength=14)
    return sc, "crown"


def p_side_cluster(side, scale=1.0, samples=96):
    sc, cam = base(scale, samples)
    mat = mat_crystal(seed=7 + (1 if side > 0 else 0))
    x = 2.6 * side
    make_cluster((x, -1.5, 0), 14, 0.9, 1.0, 3.6, mat, seed=31 + side, tilt=0.5, key_height=4.6, thick=0.3)
    add_softboxes(strength=14)
    return sc, "cl_right" if side > 0 else "cl_left"


def p_single(i, scale=1.0, samples=72):
    """Individual crystals in a ring on the floor at car-plane depth (for growth animation)."""
    sc, cam = base(scale, samples)
    rng = random.Random(100 + i)
    mat = mat_crystal(seed=20 + i)
    x = [-2.9, -2.2, -1.6, 1.6, 2.2, 2.9, -1.0, 1.0, -3.4, 3.4, -0.4, 0.5][i]
    y = rng.uniform(-0.6, 1.4)
    h = rng.uniform(0.9, 2.2) * (1.15 if abs(x) > 2 else 0.85)
    make_cluster((x, y, 0), 3, 0.25, h * 0.55, h, mat, seed=200 + i, tilt=0.35, key_height=h, thick=0.17)
    add_softboxes(strength=14)
    return sc, f"single_{i:02d}"


def p_shards(i, scale=1.0, samples=64):
    """Large near-camera crystal fragments for foreground whip/blur elements."""
    sc, cam = base(scale, samples)
    rng = random.Random(500 + i)
    mat = mat_crystal(seed=40 + i, glow=1.3)
    me = crystal_mesh(f"shard{i}", r=rng.uniform(0.35, 0.6), h=rng.uniform(2.0, 3.2), rng=rng)
    ob = bpy.data.objects.new(f"shard{i}", me)
    bpy.context.scene.collection.objects.link(ob)
    ob.data.materials.append(mat)
    ob.location = (rng.uniform(-0.6, 0.6), -3.2, 0.6)
    ob.rotation_euler = (rng.uniform(-0.6, 0.6), rng.uniform(-0.5, 0.5), rng.uniform(0, 6.28))
    add_softboxes(strength=14)
    return sc, f"shard_{i:02d}"


def render_still(sc, name):
    sc.render.filepath = os.path.join(OUT, name + ".png")
    t = time.time()
    bpy.ops.render.render(write_still=True)
    print(f"[plate] {name} {time.time()-t:.1f}s", flush=True)


if __name__ == "__main__":
    what = sys.argv[1]
    scale = float(sys.argv[2]) if len(sys.argv) > 2 else 1.0
    samples = int(sys.argv[3]) if len(sys.argv) > 3 else None
    kw = dict(scale=scale)
    if samples:
        kw["samples"] = samples
    jobs = []
    if what in ("cave", "all"):
        jobs += [lambda: p_cave("cave_a", 3, **kw), lambda: p_cave("cave_b", 8, **kw)]
    if what in ("crown", "all"):
        jobs += [lambda: p_crown(**kw)]
    if what in ("sides", "all"):
        jobs += [lambda: p_side_cluster(-1, **kw), lambda: p_side_cluster(1, **kw)]
    if what in ("singles", "all"):
        jobs += [(lambda i=i: p_single(i, **kw)) for i in range(12)]
    if what in ("shards", "all"):
        jobs += [(lambda i=i: p_shard(i, **kw)) for i in range(0)] + [(lambda i=i: p_shards(i, **kw)) for i in range(6)]
    for j in jobs:
        sc, name = j()
        render_still(sc, name)
