# rovo brand assets

| | |
|---|---|
| **Status** | Approved by the product owner on 2026-10-05. This replaces the Phase 1 placeholder decision ("neutral icon set"). |
| **Primary logo** | [`rovo-logo-primary.png`](rovo-logo-primary.png) (1774×887). Wordmark "rovo" whose first "o" is a plate with steam, spoon and fork, and whose last "o" is a wheel with speed lines. Tagline "FOOD DELIVERY". |
| **Previous logo** | [`rovo-logo-previous.png`](rovo-logo-previous.png). Superseded; kept for reference only. Do not use in new work. |
| **App-icon marks** (derived, 512×512, transparent) | [`rovo-mark-plate-512.png`](rovo-mark-plate-512.png): plate "o", the **default app icon** (customer app). [`rovo-mark-wheel-512.png`](rovo-mark-wheel-512.png): wheel "o", for the **rider** app. Restaurant and admin use the plate mark on an ink background (see below). |

## Colour tokens

These were sampled from the primary logo. The hex values are the source of truth for `web/packages/config` theme tokens.

| Token | Hex | Use |
|---|---|---|
| `ink-900` (brand ink) | `#0C1825` | Wordmark, primary text, dark surfaces, app-icon background |
| `ink-700` | `#1E2A3A` | Secondary dark surfaces |
| `flame-400` | `#FF8603` | Gradient start, highlights, illustrations |
| `flame-500` | `#F85000` | Gradient middle, brand accent, active states, rating stars |
| `flame-600` | `#E03A00` | Gradient end (red-orange speed lines) |
| `flame-700` (accessible) | `#C2410C` | **Text or icons on white** that need brand colour; white text on primary buttons |
| `plate-50` | `#F6F4EC` | Warm off-white background (plate centre) |
| Brand gradient | `linear-gradient(90deg, #FF8603, #F85000 55%, #E03A00)` | Logo swooshes, hero accents, progress bars. Never under body text. |

## Accessibility rules (WCAG 2.2 AA)

Measured contrast ratios:

| Combination | Ratio | Rule |
|---|---|---|
| `ink-900` on white | 17.9 : 1 | Default text colour |
| White on `ink-900` | 17.9 : 1 | Dark headers, rider app |
| `flame-700` on white / white on `flame-700` | 5.18 : 1 | Primary button fill (white label); brand-coloured links |
| `ink-900` on `flame-400` | 7.39 : 1 | Badges and chips on bright orange |
| `ink-900` on `flame-500` | 5.23 : 1 | Alternative primary button (dark label) |
| `flame-500` on white | 3.42 : 1 | **Large text (≥ 24 px / 18.66 px bold) and non-text UI only** |
| `flame-400` on white | 2.42 : 1 | **Decorative only.** Never for text or essential icons. |

Status colours (veg green, non-veg brown-red, error, success) stay as defined in doc 17. They must not be confused with the brand orange. In particular, the non-veg marker must not use `flame-*`.

## Usage

- Keep clear space around the wordmark equal to the height of the "v". Minimum wordmark width: 96 px on screen.
- Do not recolour, stretch, rotate or add effects. Do not place the full-colour logo on busy photos. On dark backgrounds, use a white wordmark variant (to be produced as SVG, see below).
- **PWA icons:** generate 192/512 px "any" icons and a maskable icon (mark centred inside the 80% safe zone, `ink-900` background) per app:

  | App | Icon mark | Icon background | `theme_color` |
  |---|---|---|---|
  | customer | plate | white | `#0C1825` |
  | rider | wheel | white | `#0C1825` |
  | restaurant | plate | `ink-900` | `#0C1825` |
  | admin | plate, monochrome white | `ink-900` | `#0C1825` |

  The admin app has no service worker, but it still gets a favicon.
- **Favicon:** 32/48 px from the plate mark. At favicon size the steam may need simplifying; that comes with the vector version.

## Outstanding brand work

1. **Vector source.** The approved logo is a raster (PNG with soft shading). Phase 2 needs SVG versions of the wordmark, the white/monochrome variants and the two "o" marks, so the apps stay crisp at every size and stay small for low-end phones. The full raster logo is about 1 MB and must not ship in app bundles. Until vectors exist, apps use optimised WebP/PNG exports at their display size (header ≤ 20 KB).
2. **Trademark check** for "rovo" in India (Class 39 / 43) before public launch [LEGAL].
3. **Telugu wordmark (రోవో),** if marketing wants one. Optional.

## Licensing note

The repository code is Apache-2.0, and that licence does **not** grant rights to the rovo name or logo. Forks must remove or replace the brand assets. A trademark notice belongs in `NOTICE` and `CONTRIBUTING.md`; DevOps owns that file in Phase 2.
