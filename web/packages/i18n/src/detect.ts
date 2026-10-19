import { LANGUAGE_STORAGE_KEY, type Language } from './config';

function storage(): Storage | null {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage;
  } catch {
    return null; // private mode / disabled storage
  }
}

export function isLanguage(v: unknown, allowPseudo: boolean): v is Language {
  return v === 'en' || v === 'te' || (allowPseudo && v === 'en-XA');
}

export function readStoredLanguage(allowPseudo = false): Language | null {
  try {
    const v = storage()?.getItem(LANGUAGE_STORAGE_KEY);
    return isLanguage(v, allowPseudo) ? v : null;
  } catch {
    return null;
  }
}

export function storeLanguage(lang: Language): void {
  try {
    storage()?.setItem(LANGUAGE_STORAGE_KEY, lang);
  } catch {
    /* ignore quota / privacy errors */
  }
}

/** Stored choice → `?lang=` (dev) → browser languages containing Telugu → English. */
export function detectLanguage(allowPseudo = false): Language {
  if (allowPseudo && typeof location !== 'undefined') {
    const q = new URLSearchParams(location.search).get('lang');
    if (isLanguage(q, true)) return q;
  }
  const stored = readStoredLanguage(allowPseudo);
  if (stored) return stored;
  const langs = typeof navigator === 'undefined' ? [] : (navigator.languages ?? [navigator.language]);
  return langs.some((l) => l?.toLowerCase().startsWith('te')) ? 'te' : 'en';
}
