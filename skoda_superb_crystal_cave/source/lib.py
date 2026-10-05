"""Blender scene library for the Skoda crystal-cave edit (runs inside bpy venv)."""
import bpy, bmesh, math, random, os, json
from mathutils import Vector, Euler

HERE = os.path.dirname(os.path.abspath(__file__))
FPS = 30


# ----------------------------------------------------------------- utilities
def reset():
    bpy.ops.wm.read_factory_settings(use_empty=True)
    return bpy.context.scene


def ease(t):  # smootherstep
    t = max(0.0, min(1.0, t))
    return t * t * t * (t * (t * 6 - 15) + 10)


def keyframe_interp(obj, data_path, frame, value, index=-1, interp='BEZIER'):
    setattr(obj, data_path, value) if index == -1 else None
    obj.keyframe_insert(data_path, frame=frame, index=index)


# ----------------------------------------------------------------- materials
def mat_crystal(name="Crystal", tint=(0.03, 0.75, 0.38), glow=1.0, seed=0, emit_mix=0.18):
    """Emerald gem: dark absorbing glass body, crisp facet reflections, cloudy inner glow."""
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    nt = m.node_tree
    nt.nodes.clear()
    N = nt.nodes.new; L = nt.links.new
    out = N("ShaderNodeOutputMaterial")
    glass = N("ShaderNodeBsdfGlass")
    glass.inputs["Color"].default_value = (0.42, 1.0, 0.68, 1)
    glass.inputs["Roughness"].default_value = 0.015
    glass.inputs["IOR"].default_value = 1.58
    # cloudy interior glow
    tc = N("ShaderNodeTexCoord")
    noise = N("ShaderNodeTexNoise"); noise.noise_dimensions = '4D'
    noise.inputs["W"].default_value = seed; noise.inputs["Scale"].default_value = 2.2
    noise.inputs["Detail"].default_value = 7; noise.inputs["Roughness"].default_value = 0.62
    L(tc.outputs["Object"], noise.inputs["Vector"])
    cr = N("ShaderNodeMapRange"); cr.inputs["From Min"].default_value = 0.35; cr.inputs["From Max"].default_value = 0.75
    cr.inputs["To Min"].default_value = 0.0; cr.inputs["To Max"].default_value = glow * 2.2
    L(noise.outputs["Fac"], cr.inputs["Value"])
    emis = N("ShaderNodeEmission"); emis.inputs["Color"].default_value = (*tint, 1)
    L(cr.outputs["Result"], emis.inputs["Strength"])
    mix = N("ShaderNodeMixShader"); mix.inputs["Fac"].default_value = emit_mix
    L(glass.outputs["BSDF"], mix.inputs[1]); L(emis.outputs["Emission"], mix.inputs[2])
    lp = N("ShaderNodeLightPath"); tr = N("ShaderNodeBsdfTransparent")
    tr.inputs["Color"].default_value = (0.3, 0.9, 0.55, 1)
    mix2 = N("ShaderNodeMixShader")
    L(lp.outputs["Is Shadow Ray"], mix2.inputs["Fac"]); L(mix.outputs["Shader"], mix2.inputs[1]); L(tr.outputs["BSDF"], mix2.inputs[2])
    L(mix2.outputs["Shader"], out.inputs["Surface"])
    return m


def add_softboxes(strength=14.0, tint=(0.8, 1.0, 0.9), positions=None):
    """Large emissive strips surrounding the subject -> crisp facet highlights in glass/floor."""
    positions = positions or [((-5, -3, 3), (0.9, 0, -0.6), (0.6, 7)), ((5, -3, 3.5), (0.9, 0, 0.6), (0.6, 7)),
                              ((0, 5, 6), (-0.5, 0, 0), (9, 1.2)), ((-3, 4, 1), (1.4, 0, -0.4), (0.5, 6))]
    mat = mat_emit("softbox", tint, strength)
    objs = []
    for loc, rot, size in positions:
        bpy.ops.mesh.primitive_plane_add(size=1, location=loc, rotation=rot)
        o = bpy.context.object; o.scale = (size[0], size[1], 1); o.data.materials.append(mat)
        o.visible_camera = False
        objs.append(o)
    return objs


def mat_floor(name="Floor"):
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    nt = m.node_tree
    nt.nodes.clear()
    out = nt.nodes.new("ShaderNodeOutputMaterial")
    bsdf = nt.nodes.new("ShaderNodeBsdfPrincipled")
    bsdf.inputs["Base Color"].default_value = (0.02, 0.03, 0.03, 1)
    bsdf.inputs["Metallic"].default_value = 0.9
    bsdf.inputs["Specular IOR Level"].default_value = 0.8
    tc = nt.nodes.new("ShaderNodeTexCoord")
    n1 = nt.nodes.new("ShaderNodeTexNoise"); n1.inputs["Scale"].default_value = 1.2; n1.inputs["Detail"].default_value = 8
    ramp = nt.nodes.new("ShaderNodeMapRange")
    ramp.inputs["From Min"].default_value = 0.35; ramp.inputs["From Max"].default_value = 0.7
    ramp.inputs["To Min"].default_value = 0.035; ramp.inputs["To Max"].default_value = 0.30
    nt.links.new(tc.outputs["Object"], n1.inputs["Vector"])
    nt.links.new(n1.outputs["Fac"], ramp.inputs["Value"])
    nt.links.new(ramp.outputs["Result"], bsdf.inputs["Roughness"])
    n2 = nt.nodes.new("ShaderNodeTexNoise"); n2.inputs["Scale"].default_value = 9; n2.inputs["Detail"].default_value = 10
    bump = nt.nodes.new("ShaderNodeBump"); bump.inputs["Strength"].default_value = 0.08
    nt.links.new(tc.outputs["Object"], n2.inputs["Vector"])
    nt.links.new(n2.outputs["Fac"], bump.inputs["Height"])
    nt.links.new(bump.outputs["Normal"], bsdf.inputs["Normal"])
    nt.links.new(bsdf.outputs["BSDF"], out.inputs["Surface"])
    return m


def mat_rock(name="Rock"):
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    nt = m.node_tree
    nt.nodes.clear()
    out = nt.nodes.new("ShaderNodeOutputMaterial")
    bsdf = nt.nodes.new("ShaderNodeBsdfPrincipled")
    bsdf.inputs["Base Color"].default_value = (0.014, 0.019, 0.018, 1)
    bsdf.inputs["Roughness"].default_value = 0.5
    tc = nt.nodes.new("ShaderNodeTexCoord")
    n = nt.nodes.new("ShaderNodeTexNoise"); n.inputs["Scale"].default_value = 3.5; n.inputs["Detail"].default_value = 12
    bump = nt.nodes.new("ShaderNodeBump"); bump.inputs["Strength"].default_value = 0.9
    nt.links.new(tc.outputs["Object"], n.inputs["Vector"])
    nt.links.new(n.outputs["Fac"], bump.inputs["Height"])
    nt.links.new(bump.outputs["Normal"], bsdf.inputs["Normal"])
    nt.links.new(bsdf.outputs["BSDF"], out.inputs["Surface"])
    return m


def mat_card(img_path, name, exposure=1.0, color_mult=(1, 1, 1)):
    """Shadeless photo card with alpha. Photo carries its own baked grade."""
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    nt = m.node_tree
    nt.nodes.clear()
    out = nt.nodes.new("ShaderNodeOutputMaterial")
    tex = nt.nodes.new("ShaderNodeTexImage")
    tex.image = bpy.data.images.load(img_path)
    tex.image.colorspace_settings.name = 'sRGB'
    tex.interpolation = 'Cubic'
    tex.extension = 'CLIP'
    emis = nt.nodes.new("ShaderNodeEmission")
    emis.inputs["Strength"].default_value = exposure
    mulc = nt.nodes.new("ShaderNodeMix"); mulc.data_type = 'RGBA'; mulc.blend_type = 'MULTIPLY'
    mulc.inputs["Factor"].default_value = 1.0
    mulc.inputs[7].default_value = (*color_mult, 1)
    nt.links.new(tex.outputs["Color"], mulc.inputs[6])
    nt.links.new(mulc.outputs[2], emis.inputs["Color"])
    tr = nt.nodes.new("ShaderNodeBsdfTransparent")
    mix = nt.nodes.new("ShaderNodeMixShader")
    nt.links.new(tex.outputs["Alpha"], mix.inputs["Fac"])
    nt.links.new(tr.outputs["BSDF"], mix.inputs[1])
    nt.links.new(emis.outputs["Emission"], mix.inputs[2])
    nt.links.new(mix.outputs["Shader"], out.inputs["Surface"])
    m.blend_method = 'BLEND' if hasattr(m, 'blend_method') else None
    return m, tex


def mat_emit(name, color, strength):
    m = bpy.data.materials.new(name)
    m.use_nodes = True
    nt = m.node_tree
    nt.nodes.clear()
    out = nt.nodes.new("ShaderNodeOutputMaterial")
    e = nt.nodes.new("ShaderNodeEmission")
    e.inputs["Color"].default_value = (*color, 1)
    e.inputs["Strength"].default_value = strength
    nt.links.new(e.outputs["Emission"], out.inputs["Surface"])
    return m


# ----------------------------------------------------------------- geometry
def crystal_mesh(name, r=0.12, h=1.0, rng=None, sides=6, taper=0.92, tip_h=0.22):
    """Faceted hexagonal prism with a pointed cap — flat shaded."""
    rng = rng or random.Random()
    bm = bmesh.new()
    ang = [2 * math.pi * i / sides + rng.uniform(-0.12, 0.12) for i in range(sides)]
    rad = [r * rng.uniform(0.82, 1.12) for _ in range(sides)]
    z_sh = h * (1 - tip_h)
    bot = [bm.verts.new((math.cos(a) * rr * 0.9, math.sin(a) * rr * 0.9, 0)) for a, rr in zip(ang, rad)]
    mid = [bm.verts.new((math.cos(a) * rr, math.sin(a) * rr, z_sh * 0.55)) for a, rr in zip(ang, rad)]
    sh = [bm.verts.new((math.cos(a) * rr * taper, math.sin(a) * rr * taper, z_sh)) for a, rr in zip(ang, rad)]
    tip = bm.verts.new((rng.uniform(-r, r) * 0.25, rng.uniform(-r, r) * 0.25, h))
    for i in range(sides):
        j = (i + 1) % sides
        bm.faces.new((bot[i], bot[j], mid[j], mid[i]))
        bm.faces.new((mid[i], mid[j], sh[j], sh[i]))
        bm.faces.new((sh[i], sh[j], tip))
    bm.faces.new(bot[::-1])
    bm.normal_update()
    me = bpy.data.meshes.new(name)
    bm.to_mesh(me)
    bm.free()
    for p in me.polygons:
        p.use_smooth = False
    return me


def make_cluster(center, n, spread, hmin, hmax, mat, seed, tilt=0.5, coll=None, thick=0.14, key_height=None):
    rng = random.Random(seed)
    objs = []
    for i in range(n):
        h = rng.uniform(hmin, hmax)
        if i == 0 and key_height:
            h = key_height
        r = thick * (0.6 + h / (hmax * 1.4)) * rng.uniform(0.8, 1.2)
        me = crystal_mesh(f"cr{seed}_{i}", r=r, h=h, rng=rng)
        ob = bpy.data.objects.new(f"crystal_{seed}_{i}", me)
        (coll or bpy.context.scene.collection).objects.link(ob)
        ob.data.materials.append(mat)
        dx = rng.uniform(-spread, spread) * (0.25 if i == 0 else 1)
        dy = rng.uniform(-spread * 0.6, spread * 0.6) * (0.25 if i == 0 else 1)
        ob.location = (center[0] + dx, center[1] + dy, center[2] - 0.05)
        out_ang = math.atan2(dy, dx) if (dx or dy) else 0
        t = tilt * (0.15 if i == 0 else rng.uniform(0.4, 1.0)) * min(1.0, math.hypot(dx, dy) / max(spread, 1e-3) + 0.15)
        # tilt away from cluster centre
        ob.rotation_euler = (math.sin(out_ang) * -t * -1, math.cos(out_ang) * t, rng.uniform(0, 6.28))
        # correct orientation: tilt axis perpendicular to outward direction
        ob.rotation_euler = Euler((t * math.sin(out_ang), -t * math.cos(out_ang), rng.uniform(0, 6.28)), 'XYZ')
        objs.append(ob)
    return objs


def make_rocks(mat, seed=3, count=14, radius=(2.5, 5.0), zone=((-14, 14), (6, 20))):
    rng = random.Random(seed)
    coll = []
    for i in range(count):
        bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=5, radius=1)
        ob = bpy.context.object
        ob.scale = (rng.uniform(*radius), rng.uniform(*radius) * 0.9, rng.uniform(*radius) * 1.3)
        ob.location = (rng.uniform(*zone[0]), rng.uniform(*zone[1]), rng.uniform(0.5, 4.0))
        disp = ob.modifiers.new("d", 'DISPLACE')
        tex = bpy.data.textures.new(f"rk{i}", 'CLOUDS'); tex.noise_scale = rng.uniform(0.6, 1.2); tex.noise_depth = 4
        disp.texture = tex; disp.strength = 1.6
        ob.data.materials.append(mat)
        coll.append(ob)
    return coll


def make_cave_shell(mat, length=60, radius=11):
    """Big noisy tunnel around the stage so the background reads as a rocky cave, not a void."""
    bpy.ops.mesh.primitive_cylinder_add(vertices=96, radius=radius, depth=length, rotation=(math.pi / 2, 0, 0), location=(0, length / 2 - 6, radius * 0.45), end_fill_type='NOTHING')
    ob = bpy.context.object
    bpy.ops.object.shade_smooth()
    sub = ob.modifiers.new("s", 'SUBSURF'); sub.levels = 2; sub.render_levels = 2
    disp = ob.modifiers.new("d", 'DISPLACE')
    tex = bpy.data.textures.new("cave_n", 'CLOUDS'); tex.noise_scale = 1.4; tex.noise_depth = 5
    disp.texture = tex; disp.strength = 2.8
    ob.data.materials.append(mat)
    return ob


def make_motes(count, area, mat, seed=7, size=0.012):
    rng = random.Random(seed)
    objs = []
    bpy.ops.mesh.primitive_ico_sphere_add(subdivisions=1, radius=size)
    base = bpy.context.object
    base.data.materials.append(mat)
    for i in range(count):
        ob = base.copy(); ob.data = base.data
        ob.scale = (rng.uniform(0.4, 1.8),) * 3
        ob.location = (rng.uniform(*area[0]), rng.uniform(*area[1]), rng.uniform(*area[2]))
        bpy.context.scene.collection.objects.link(ob)
        objs.append(ob)
    base.hide_render = True
    return objs


# ----------------------------------------------------------------- cards
def make_card(img_path, name, height_m=None, width_m=None, exposure=1.0, color_mult=(1, 1, 1)):
    import PIL.Image as I
    w, h = I.open(img_path).size
    if width_m:
        W, H = width_m, width_m * h / w
    else:
        W, H = height_m * w / h, height_m
    bpy.ops.mesh.primitive_plane_add(size=1)
    ob = bpy.context.object
    ob.name = name
    ob.scale = (W, 1, H)
    ob.rotation_euler = (math.pi / 2, 0, 0)  # stand up, facing -Y
    bpy.ops.object.transform_apply(scale=True, rotation=False)
    # origin at bottom centre
    for v in ob.data.vertices:
        v.co.y += 0.5 * 0  # no-op (kept for clarity)
    m, tex = mat_card(img_path, name + "_m", exposure, color_mult)
    ob.data.materials.append(m)
    # shift so bottom edge sits on z=0: plane rotated; local Y is height
    for v in ob.data.vertices:
        v.co.y += H / 2
    ob.visible_shadow = False
    return ob, W, H


# ----------------------------------------------------------------- render setup
def setup_render(w=1080, h=1920, samples=48, engine_threads=4, denoise=True, motion_blur=True, shutter=0.5):
    sc = bpy.context.scene
    sc.render.engine = 'CYCLES'
    sc.cycles.device = 'CPU'
    sc.cycles.samples = samples
    sc.cycles.use_adaptive_sampling = True
    sc.cycles.adaptive_threshold = 0.03
    sc.cycles.use_denoising = denoise
    sc.cycles.denoiser = 'OPENIMAGEDENOISE'
    sc.cycles.max_bounces = 8
    sc.cycles.transmission_bounces = 8
    sc.cycles.transparent_max_bounces = 8
    sc.cycles.glossy_bounces = 4
    sc.cycles.diffuse_bounces = 2
    sc.cycles.volume_bounces = 0
    sc.cycles.caustics_reflective = False
    sc.cycles.caustics_refractive = False
    sc.cycles.sample_clamp_indirect = 8
    sc.cycles.sample_clamp_direct = 0
    sc.render.threads_mode = 'FIXED'
    sc.render.threads = engine_threads
    sc.render.resolution_x = w
    sc.render.resolution_y = h
    sc.render.resolution_percentage = 100
    sc.render.fps = FPS
    sc.render.use_motion_blur = motion_blur
    sc.render.motion_blur_shutter = shutter
    sc.view_settings.view_transform = 'Standard'
    sc.render.image_settings.file_format = 'PNG'
    sc.render.image_settings.color_depth = '16'
    sc.render.image_settings.compression = 15
    sc.render.film_transparent = False
    sc.world = bpy.data.worlds.new("W")
    sc.world.use_nodes = True
    bg = sc.world.node_tree.nodes["Background"]
    bg.inputs["Color"].default_value = (0.0, 0.004, 0.003, 1)
    bg.inputs["Strength"].default_value = 1.0
    return sc


def add_camera(lens=40, sensor=36, loc=(0, -8, 1.2), target=(0, 0, 1.0), fstop=None, focus=None):
    cam_d = bpy.data.cameras.new("cam")
    cam_d.lens = lens
    cam_d.sensor_width = sensor
    cam_d.sensor_fit = 'HORIZONTAL'
    cam = bpy.data.objects.new("cam", cam_d)
    bpy.context.scene.collection.objects.link(cam)
    bpy.context.scene.camera = cam
    cam.location = loc
    look_at(cam, target)
    if fstop:
        cam_d.dof.use_dof = True
        cam_d.dof.aperture_fstop = fstop
        cam_d.dof.focus_distance = focus or (Vector(loc) - Vector(target)).length
    return cam


def look_at(obj, target, roll=0.0):
    d = Vector(target) - Vector(obj.location)
    rot = d.to_track_quat('-Z', 'Y').to_euler()
    rot.rotate_axis('Z', 0)
    obj.rotation_euler = rot
    if roll:
        obj.rotation_euler.rotate_axis('Z', roll)


def render_range(outdir, name, f0, f1):
    sc = bpy.context.scene
    os.makedirs(outdir, exist_ok=True)
    sc.frame_start, sc.frame_end = f0, f1
    sc.render.filepath = os.path.join(outdir, name + "_####.png")
    bpy.ops.render.render(animation=True)
