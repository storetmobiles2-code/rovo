import i18next, { type i18n as I18n } from 'i18next';
import { useCallback, useMemo } from 'react';
import { initReactI18next, useTranslation } from 'react-i18next';
import {
  formatDate,
  formatDateTime,
  formatMoney,
  formatNumber,
  formatRelative,
  formatTime,
  type Money,
  type Paise,
} from '@rovo/utils';
import { lazyCatalogBackend } from './backend';
import { type Language, type Namespace } from './config';
import { detectLanguage, storeLanguage } from './detect';

export * from './config';
export { detectLanguage, readStoredLanguage, storeLanguage } from './detect';
export { pseudoLocalize, pseudoLocalizeString } from './pseudo';
export { loadCatalog } from './backend';
export { Trans, useTranslation, I18nextProvider } from 'react-i18next';

/** The shared i18next instance used by every rovo app. */
export const i18n: I18n = i18next.createInstance();

export interface InitI18nOptions {
  /** Namespaces this app uses; `common` and `errors` are always loaded. */
  namespaces: Namespace[];
  /** Force a language (tests). Defaults to stored choice / browser detection. */
  language?: Language;
  /** Allow the `en-XA` pseudo-locale (dev/staging only). */
  allowPseudo?: boolean;
}

function htmlLang(lang: string): string {
  return lang === 'te' ? 'te' : 'en';
}

function applyDocumentLanguage(lang: string): void {
  if (typeof document !== 'undefined') document.documentElement.lang = htmlLang(lang);
}

let initPromise: Promise<I18n> | null = null;

/**
 * Initialises i18next once and resolves when the current language's namespaces are loaded.
 * Apps await this before the first render so no untranslated flash appears.
 */
export function initI18n(opts: InitI18nOptions): Promise<I18n> {
  const ns = Array.from(new Set<Namespace>(['common', 'errors', ...opts.namespaces]));
  if (initPromise) {
    return initPromise.then(async (inst) => {
      await inst.loadNamespaces(ns);
      if (opts.language && inst.language !== opts.language) await inst.changeLanguage(opts.language);
      return inst;
    });
  }
  const allowPseudo = opts.allowPseudo ?? false;
  const lng = opts.language ?? detectLanguage(allowPseudo);
  initPromise = i18n
    .use(lazyCatalogBackend)
    .use(initReactI18next)
    .init({
      lng,
      fallbackLng: 'en',
      supportedLngs: allowPseudo ? ['en', 'te', 'en-XA'] : ['en', 'te'],
      load: 'currentOnly',
      ns,
      defaultNS: 'common',
      fallbackNS: 'common',
      interpolation: { escapeValue: false },
      returnNull: false,
      react: { useSuspense: false },
    })
    .then(() => {
      i18n.services.formatter?.add('money', (value: Paise | Money, l?: string) =>
        formatMoney(value, l ?? i18n.language),
      );
      applyDocumentLanguage(i18n.language);
      i18n.on('languageChanged', applyDocumentLanguage);
      return i18n;
    });
  return initPromise;
}

/** Changes the UI language, persists it and updates `<html lang>`. */
export async function setLanguage(lang: Language): Promise<void> {
  storeLanguage(lang);
  await i18n.changeLanguage(lang);
}

/** `[language, setLanguage]` for the language switcher. */
export function useLanguage(): [Language, (lang: Language) => Promise<void>] {
  const { i18n: inst } = useTranslation();
  const change = useCallback((lang: Language) => setLanguage(lang), []);
  return [(inst.resolvedLanguage ?? inst.language ?? 'en') as Language, change];
}

/** Locale-aware formatters bound to the current UI language (money is always ₹, en-IN grouping). */
export function useFormatters() {
  const { i18n: inst } = useTranslation();
  const lang = inst.language ?? 'en';
  return useMemo(
    () => ({
      money: (v: Paise | Money, opts?: { fixed?: boolean }) => formatMoney(v, lang, opts),
      number: (n: number) => formatNumber(n, lang),
      date: (v: string | number | Date) => formatDate(v, lang),
      time: (v: string | number | Date) => formatTime(v, lang),
      dateTime: (v: string | number | Date) => formatDateTime(v, lang),
      relative: (v: string | number | Date, now?: number) => formatRelative(v, now, lang),
    }),
    [lang],
  );
}
