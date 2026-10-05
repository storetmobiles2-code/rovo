# Flow pack — SKODA_SUPERB_CRYSTAL_CAVE

Make **9 clips** (+1 optional) in Google Flow with your Google AI Pro subscription. Settings for every clip: **Veo (Fast or Quality), vertical 9:16, 8 seconds**.
Save each result as an MP4 named exactly like the **file name** shown for the shot (e.g. `S03_plate_macro.mp4`) and send me all of them.

Menu names in Flow change from time to time — look for *Frames to Video* (start-frame + prompt) and *Text to Video*.

**Identity rule (add to every prompt that shows the car):** Keep the car EXACTLY as in the photo: same body shape, white paint, wheels, headlamps, grille, badges and the number plate reading TS 09 EF 6141. Change only the environment and lighting.

**Negative prompt (if Flow has the field):** text overlays, subtitles, watermark, logo changes, deformed car, warped wheels, extra wheels, melted body, distorted number plate, people, cartoon, low quality

## 1. `S01_reveal_orbit.mp4` — Void reveal + orbit (hook)
**Step A — make the start frame** in the Gemini app (image generation) or Flow's image tool: upload `photos/front.jpg`, then paste:

> Place this exact car on a glossy black mirror-like floor in a pitch-black void, low front three-quarter view. A thin emerald rim light traces the body edges. Small emerald crystals are just beginning to grow from the floor around the wheels. Soft sharp reflection of the car in the floor. Cinematic automotive commercial still, vertical 9:16. Keep the car EXACTLY as in the photo: same body shape, white paint, wheels, headlamps, grille, badges and the number plate reading TS 09 EF 6141. Change only the environment and lighting.

Check it: car identical, plate readable. Save it as `start_frames/S01_reveal_orbit.png` and use it as the first frame below.

**Video prompt:**

> Slow cinematic orbit around the car starting from the front three-quarter and sweeping toward the side, emerald crystals rapidly grow and ignite around it, a bright light sweep runs along the bodywork, reflections ripple on the wet black floor, tiny dust motes drift through the beams. Smooth stable camera, photoreal, the car stays rigid and unchanged.

## 2. `S02_cave_hero_pushin.mp4` — Cave impact hero push-in
**Step A — make the start frame** in the Gemini app (image generation) or Flow's image tool: upload `photos/front.jpg`, then paste:

> Place this exact car in a vast dark crystal cave, low-angle hero shot from the front, a giant crown of translucent emerald crystals rising behind it, wet black reflective floor, volumetric mist, cold green rim lights, dramatic and symmetrical. Vertical 9:16. Keep the car EXACTLY as in the photo: same body shape, white paint, wheels, headlamps, grille, badges and the number plate reading TS 09 EF 6141. Change only the environment and lighting.

Check it: car identical, plate readable. Save it as `start_frames/S02_cave_hero_pushin.png` and use it as the first frame below.

**Video prompt:**

> Fast confident dolly push-in from a low angle toward the front of the car, giant emerald crystals glint and pulse behind it, mist swirls across the floor, the headlamps shine through the haze, slight handheld energy, ends tight on the grille. The car stays unchanged.

## 3. `S03_plate_macro.mp4` — Number-plate macro
**Start frame:** `start_frames/S03_plate_macro.png` (a crop of your real photo — upload it as the first frame; real plate/lamp/wheel pixels).

**Video prompt:**

> Extreme macro push-in on the front number plate and chrome grille, shallow depth of field, out-of-focus emerald crystals slide through the foreground, cold green light glints sweep across the chrome, a faint mist drifts. The plate text TS 09 EF 6141 stays perfectly sharp and unchanged.

## 4. `S04_lamp_macro.mp4` — Headlamp macro + flare
**Start frame:** `start_frames/S04_lamp_macro.png` (a crop of your real photo — upload it as the first frame; real plate/lamp/wheel pixels).

**Video prompt:**

> Macro rack-focus on the headlamp, the LED daytime running light ignites with a bright horizontal anamorphic flare, emerald crystal reflections drift across the lens glass, very slow push-in, dark cave background with bokeh. The lamp shape stays unchanged.

## 5. `S05_side_tracking.mp4` — Side tracking with crystal wipes
**Step A — make the start frame** in the Gemini app (image generation) or Flow's image tool: upload `photos/side.jpg`, then paste:

> Place this exact car in a dark emerald crystal cave, perfect side profile, wet black mirror floor with a crisp reflection, big out-of-focus emerald crystals in the foreground at both edges, volumetric mist, cold rim light. Vertical 9:16. Keep the car EXACTLY as in the photo: same body shape, white paint, wheels, headlamps, grille, badges and the number plate reading TS 09 EF 6141. Change only the environment and lighting.

Check it: car identical, plate readable. Save it as `start_frames/S05_side_tracking.png` and use it as the first frame below.

**Video prompt:**

> Smooth lateral tracking shot parallel to the car, giant out-of-focus emerald crystals sweep through the foreground like wipes, reflections slide across the wet floor, mist drifts, then the foreground clears to reveal the full side profile. Steady speed. The car stays unchanged.

## 6. `S06_rear_reveal.mp4` — Rear quarter reveal
**Step A — make the start frame** in the Gemini app (image generation) or Flow's image tool: upload `photos/rear34.jpg`, then paste:

> Place this exact car in a dark emerald crystal cave, rear three-quarter view, the tail lamp glowing red, emerald highlights on the white paint, wet black floor, crystals framing the car, volumetric mist. Vertical 9:16. Keep the car EXACTLY as in the photo: same body shape, white paint, wheels, headlamps, grille, badges and the number plate reading TS 09 EF 6141. Change only the environment and lighting.

Check it: car identical, plate readable. Save it as `start_frames/S06_rear_reveal.png` and use it as the first frame below.

**Video prompt:**

> Slow dolly around the rear quarter starting from behind a crystal cluster, the red tail lamp glows, emerald light slides across the white paint, mist curls, camera glides left to right and slightly upward. The car stays unchanged.

## 7. `S07_wheel_macro.mp4` — Alloy wheel macro
**Start frame:** `start_frames/S07_wheel_macro.png` (a crop of your real photo — upload it as the first frame; real plate/lamp/wheel pixels).

**Video prompt:**

> Macro push-in on the alloy wheel, bright light glints sweep across the spokes, wet floor reflecting an emerald glow, blurred crystals in the background. The wheel stays round and unchanged.

## 8. `S08_finale_hero.mp4` — Finale crown hero (pull-back)
**Step A — make the start frame** in the Gemini app (image generation) or Flow's image tool: upload `photos/side.jpg`, then paste:

> Place this exact car in a vast dark crystal cave, side profile hero shot, a giant crown of translucent emerald crystals rising behind it, a perfect mirror reflection on the black floor, floating glowing spores, volumetric mist. Vertical 9:16. Keep the car EXACTLY as in the photo: same body shape, white paint, wheels, headlamps, grille, badges and the number plate reading TS 09 EF 6141. Change only the environment and lighting.

Check it: car identical, plate readable. Save it as `start_frames/S08_finale_hero.png` and use it as the first frame below.

**Video prompt:**

> Slow majestic crane-style pull-back and rise from the side profile revealing a giant crown of emerald crystals behind the car, the crystals pulse with inner light, perfect mirror reflection on the floor, glowing spores float upward. Calm, premium, stable. The car stays unchanged.

## 9. `S09_badge_gem.mp4` — Badge macro → crystal gem (reference closer)
**Start frame:** `start_frames/S09_badge_gem.png` (a crop of your real photo — upload it as the first frame; real plate/lamp/wheel pixels).

**Video prompt:**

> Extreme macro on the round Škoda badge on the bonnet, slow push-in; the chrome badge ripples and crystallises into a perfect faceted emerald gem that catches the light, light rays glint across its facets, cave darkness behind. The badge position and size stay the same.

## 10. `T01_crystal_whip.mp4` — Crystal whip (transition element)
**No start frame** (text-to-video).

**Video prompt:**

> Camera whips very fast past enormous translucent emerald crystals that fill the frame, heavy directional motion blur, dark background, bright facet glints, no car, no people, no text.

## Then
Send me the MP4s. I run the beat-synced edit, check it against your reference with the validation gate, and iterate.