import { describe, expect, it } from 'vitest';
import { safeRedirect } from './redirect';

describe('safeRedirect', () => {
  it('keeps in-app paths', () => {
    expect(safeRedirect('/orders/RV-1?tab=bill')).toBe('/orders/RV-1?tab=bill');
  });

  it('rejects off-site, protocol-relative and backslash targets', () => {
    expect(safeRedirect('https://evil.example')).toBe('/');
    expect(safeRedirect('//evil.example')).toBe('/');
    expect(safeRedirect('/\\evil.example')).toBe('/');
    expect(safeRedirect('javascript:alert(1)')).toBe('/');
  });

  it('falls back for empty values and for the login page itself', () => {
    expect(safeRedirect(undefined)).toBe('/');
    expect(safeRedirect('', '/inbox')).toBe('/inbox');
    expect(safeRedirect('/login?redirect=%2Flogin', '/inbox')).toBe('/inbox');
  });
});
