/** UI languages. `en-XA` is the pseudo-locale used in dev to catch hard-coded strings. */
export type AppLanguage = 'en' | 'te' | 'en-XA';

/** Every rovo date/time is shown in the city time zone, never the device zone. */
export const CITY_TIME_ZONE = 'Asia/Kolkata';

/** Maps a UI language to the Intl locale used for numbers, money and dates. */
export function intlLocale(lang: AppLanguage | string): 'en-IN' | 'te-IN' {
  return lang === 'te' || lang.startsWith('te-') ? 'te-IN' : 'en-IN';
}
