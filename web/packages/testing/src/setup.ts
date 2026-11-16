/**
 * Shared Vitest setup (jsdom): jest-dom matchers, Testing Library cleanup, browser API
 * polyfills Radix needs, and English i18n with every namespace loaded.
 * Use via `test.setupFiles: ['@rovo/testing/setup']`.
 */
import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
import { i18n, initI18n, NAMESPACES } from '@rovo/i18n';

if (typeof window !== 'undefined') {
  if (!window.matchMedia) {
    window.matchMedia = (query: string) =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
        dispatchEvent: () => false,
      }) as MediaQueryList;
  }
  if (!('ResizeObserver' in window)) {
    class ResizeObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    (window as unknown as { ResizeObserver: unknown }).ResizeObserver = ResizeObserverStub;
  }
  const proto = window.HTMLElement.prototype as unknown as Record<string, unknown>;
  proto.scrollIntoView ??= () => {};
  proto.hasPointerCapture ??= () => false;
  proto.releasePointerCapture ??= () => {};
}

await initI18n({ namespaces: [...NAMESPACES], language: 'en' });

afterEach(async () => {
  cleanup();
  try {
    localStorage.clear();
  } catch {
    /* ignore */
  }
  if (i18n.language !== 'en') await i18n.changeLanguage('en');
});
