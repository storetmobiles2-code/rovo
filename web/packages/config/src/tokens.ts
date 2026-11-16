/**
 * rovo design tokens — the single source of truth for colour values.
 * Brand colours come from docs/brand/README.md (approved 2026-10-05).
 * `theme.css` mirrors these values for Tailwind v4; `tokens.test.ts` keeps the
 * two in sync and enforces WCAG 2.2 AA contrast for every documented pairing.
 */
export const palette = {
  white: '#FFFFFF',
  'ink-900': '#0C1825', // brand ink: primary text, dark surfaces
  'ink-700': '#1E2A3A', // secondary dark surface
  'ink-500': '#4B5768', // muted text (7.3:1 on white)
  'ink-400': '#7A8494', // input borders / non-text UI (3.8:1 on white)
  'ink-200': '#D5DAE1', // dividers (decorative)
  'ink-100': '#EEF0F3', // subtle fills, skeletons
  'flame-50': '#FFF4EB',
  'flame-100': '#FFE4CC',
  'flame-400': '#FF8603', // decorative only (2.4:1 on white)
  'flame-500': '#F85000', // large text / non-text UI only on white (3.4:1)
  'flame-600': '#E03A00', // gradient end
  'flame-700': '#C2410C', // accessible brand: text/icons on white, primary button fill
  'flame-800': '#9A3412', // primary hover/pressed
  'plate-50': '#F6F4EC', // warm off-white background
  // Status colours — deliberately not brand orange (brand README).
  'success-700': '#15803D',
  'success-50': '#ECFDF3',
  'warning-700': '#A15C07',
  'warning-50': '#FFFAEB',
  'danger-700': '#B42318',
  'danger-50': '#FEF3F2',
  'info-700': '#1D4ED8',
  'info-50': '#EFF4FF',
  // Food markers (FSSAI convention; never flame-*).
  veg: '#0F7B34',
  nonveg: '#7A3E1D',
  egg: '#A15C07',
  // Dark theme (rider app follows the system theme).
  'dark-text': '#F6F4EC',
  'dark-muted': '#B8C0CC',
  'dark-border': '#8A94A3',
  'dark-primary': '#FF8A4C',
  'dark-success': '#4ADE80',
  'dark-danger': '#F87171',
  'dark-focus': '#FFB061',
} as const;

export type PaletteToken = keyof typeof palette;

/** Pairs that must meet a minimum contrast ratio. [foreground, background, minimum, use]. */
export const contrastRules: ReadonlyArray<readonly [PaletteToken, PaletteToken, number, string]> = [
  ['ink-900', 'white', 4.5, 'body text'],
  ['ink-900', 'plate-50', 4.5, 'body text on warm background'],
  ['ink-500', 'white', 4.5, 'muted text'],
  ['ink-500', 'plate-50', 4.5, 'muted text on warm background'],
  ['white', 'flame-700', 4.5, 'primary button label'],
  ['flame-700', 'white', 4.5, 'brand links'],
  ['white', 'flame-800', 4.5, 'primary button hover label'],
  ['ink-900', 'flame-500', 4.5, 'dark label on accent'],
  ['ink-900', 'flame-400', 4.5, 'chip on bright orange'],
  ['flame-700', 'flame-50', 4.5, 'brand text on tint'],
  ['ink-400', 'white', 3, 'input borders (SC 1.4.11)'],
  ['flame-500', 'white', 3, 'large text / non-text accent'],
  ['success-700', 'white', 4.5, 'success text'],
  ['success-700', 'success-50', 4.5, 'success badge'],
  ['warning-700', 'white', 4.5, 'warning text'],
  ['warning-700', 'warning-50', 4.5, 'warning badge'],
  ['danger-700', 'white', 4.5, 'error text'],
  ['danger-700', 'danger-50', 4.5, 'error badge'],
  ['info-700', 'white', 4.5, 'info text'],
  ['info-700', 'info-50', 4.5, 'info badge'],
  ['veg', 'white', 3, 'veg marker (non-text)'],
  ['nonveg', 'white', 3, 'non-veg marker (non-text)'],
  ['egg', 'white', 3, 'egg marker (non-text)'],
  ['white', 'ink-900', 4.5, 'text on dark header'],
  ['dark-text', 'ink-900', 4.5, 'dark theme text'],
  ['dark-muted', 'ink-900', 4.5, 'dark theme muted text'],
  ['dark-muted', 'ink-700', 4.5, 'dark theme muted text on surface'],
  ['dark-border', 'ink-700', 3, 'dark theme borders'],
  ['dark-primary', 'ink-900', 4.5, 'dark theme primary'],
  ['ink-900', 'dark-primary', 4.5, 'dark theme primary button label'],
  ['dark-success', 'ink-700', 4.5, 'dark theme success'],
  ['dark-danger', 'ink-700', 4.5, 'dark theme danger'],
  ['dark-focus', 'ink-900', 3, 'dark theme focus ring'],
  ['ink-900', 'white', 3, 'light theme focus ring'],
];

function channel(c: number): number {
  const s = c / 255;
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
}

export function relativeLuminance(hex: string): number {
  const h = hex.replace('#', '');
  const r = parseInt(h.slice(0, 2), 16);
  const g = parseInt(h.slice(2, 4), 16);
  const b = parseInt(h.slice(4, 6), 16);
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

export function contrastRatio(a: string, b: string): number {
  const la = relativeLuminance(a);
  const lb = relativeLuminance(b);
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}

/** PWA / browser chrome colours per app (brand README "PWA icons"). */
export const appThemeColor = '#0C1825';
