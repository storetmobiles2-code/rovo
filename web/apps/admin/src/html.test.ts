// @vitest-environment node
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const html = readFileSync(join(import.meta.dirname, '..', 'index.html'), 'utf8');

describe('admin index.html (strict CSP, doc 17 §15.6)', () => {
  it('has no inline scripts, inline styles or event-handler attributes', () => {
    const scripts = [...html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/g)];
    expect(scripts.length).toBeGreaterThan(0);
    for (const [, attrs, body] of scripts) {
      expect(attrs).toMatch(/\bsrc=/);
      expect(body!.trim()).toBe('');
    }
    expect(html).not.toMatch(/<style\b/);
    expect(html).not.toMatch(/\sstyle=/);
    expect(html).not.toMatch(/\son[a-z]+=/i);
  });

  it('registers no service worker or web manifest', () => {
    expect(html).not.toMatch(/manifest|serviceWorker|registerSW/i);
  });

  it('is not indexable', () => {
    expect(html).toMatch(/<meta name="robots" content="noindex"/);
  });
});
