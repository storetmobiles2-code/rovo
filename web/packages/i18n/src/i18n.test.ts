import { describe, expect, it } from 'vitest';
// @ts-expect-error -- plain ESM script without types
import { checkCatalogs } from '../scripts/check-i18n.mjs';
import { i18n, initI18n, pseudoLocalizeString, setLanguage } from './index';

describe('catalog completeness', () => {
  it('every English key has a Telugu translation with the same variables', () => {
    const { errors } = checkCatalogs() as { errors: string[] };
    expect(errors).toEqual([]);
  });
});

describe('initI18n', () => {
  it('loads namespaces lazily, translates, pluralises and switches language', async () => {
    await initI18n({ namespaces: ['auth', 'customer'], language: 'en' });
    expect(i18n.t('actions.retry')).toBe('Try again');
    expect(i18n.t('auth:otp.resendIn', { count: 1 })).toBe('Resend code in 1 second');
    expect(i18n.t('auth:otp.resendIn', { count: 30 })).toBe('Resend code in 30 seconds');
    expect(i18n.t('errors:OTP_INVALID')).toMatch(/not correct/);
    expect(document.documentElement.lang).toBe('en');

    await setLanguage('te');
    expect(i18n.language).toBe('te');
    expect(i18n.t('actions.retry')).toBe('మళ్ళీ ప్రయత్నించండి');
    expect(i18n.t('customer:nav.cart')).toBe('కార్ట్');
    expect(document.documentElement.lang).toBe('te');
    expect(localStorage.getItem('rovo.lang')).toBe('te');
    await setLanguage('en');
  });

  it('formats money through the {{value, money}} formatter', async () => {
    await initI18n({ namespaces: [], language: 'en' });
    expect(i18n.t('x', { defaultValue: 'Total {{v, money}}', v: 12345650 })).toBe(
      'Total ₹1,23,456.50',
    );
  });
});

describe('pseudo-locale', () => {
  it('accents text, keeps interpolations and pads', () => {
    const out = pseudoLocalizeString('Resend in {{count}} seconds');
    expect(out).toContain('{{count}}');
    expect(out).toMatch(/^\[Ŕéšéñð íñ \{\{count\}\} šéçóñðš~+\]$/);
  });
});
