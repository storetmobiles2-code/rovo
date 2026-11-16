export const SUPPORTED_LANGUAGES = ['en', 'te'] as const;
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];
/** `en-XA` is a pseudo-locale (accented, ~40 % longer) for catching hard-coded strings in dev. */
export type Language = SupportedLanguage | 'en-XA';

export const NAMESPACES = [
  'common',
  'auth',
  'errors',
  'customer',
  'restaurant',
  'rider',
  'admin',
] as const;
export type Namespace = (typeof NAMESPACES)[number];

/** localStorage key for the user's language choice (doc 17 §10). */
export const LANGUAGE_STORAGE_KEY = 'rovo.lang';
