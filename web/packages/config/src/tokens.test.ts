import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { contrastRatio, contrastRules, palette } from './tokens';

const themeCss = readFileSync(join(import.meta.dirname, '..', 'theme.css'), 'utf8').toLowerCase();

describe('design tokens', () => {
  it.each(contrastRules)('%s on %s ≥ %s:1 (%s)', (fg, bg, min) => {
    expect(contrastRatio(palette[fg], palette[bg])).toBeGreaterThanOrEqual(min);
  });

  it('brand colours match docs/brand/README.md', () => {
    expect(palette['ink-900']).toBe('#0C1825');
    expect(palette['flame-400']).toBe('#FF8603');
    expect(palette['flame-500']).toBe('#F85000');
    expect(palette['flame-600']).toBe('#E03A00');
    expect(palette['flame-700']).toBe('#C2410C');
    expect(palette['plate-50']).toBe('#F6F4EC');
  });

  it('theme.css declares every non-dark palette token with the same value', () => {
    for (const [name, hex] of Object.entries(palette)) {
      if (name.startsWith('dark-')) continue;
      expect(themeCss).toContain(`--color-${name}: ${hex.toLowerCase()};`);
    }
  });

  it('non-veg marker is not a brand flame colour', () => {
    const flames = ['flame-400', 'flame-500', 'flame-600', 'flame-700', 'flame-800'] as const;
    for (const f of flames) expect(palette.nonveg).not.toBe(palette[f]);
  });
});
