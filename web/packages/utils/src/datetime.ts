import { type AppLanguage, CITY_TIME_ZONE, intlLocale } from './locale';

type DateInput = string | number | Date;

function toDate(v: DateInput): Date {
  return v instanceof Date ? v : new Date(v);
}

const cache = new Map<string, Intl.DateTimeFormat>();
function dtf(lang: string, opts: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
  const key = `${lang}|${JSON.stringify(opts)}`;
  let f = cache.get(key);
  if (!f) {
    f = new Intl.DateTimeFormat(intlLocale(lang), {
      timeZone: CITY_TIME_ZONE,
      numberingSystem: 'latn',
      ...opts,
    });
    cache.set(key, f);
  }
  return f;
}

/** "2:05 pm" in Asia/Kolkata. */
export function formatTime(v: DateInput, lang: AppLanguage | string = 'en'): string {
  return dtf(lang, { hour: 'numeric', minute: '2-digit', hour12: true }).format(toDate(v));
}

/** "4 Oct 2026" in Asia/Kolkata. */
export function formatDate(v: DateInput, lang: AppLanguage | string = 'en'): string {
  return dtf(lang, { day: 'numeric', month: 'short', year: 'numeric' }).format(toDate(v));
}

/** "4 Oct 2026, 2:05 pm" in Asia/Kolkata. */
export function formatDateTime(v: DateInput, lang: AppLanguage | string = 'en'): string {
  return dtf(lang, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  }).format(toDate(v));
}

/** Calendar day in Asia/Kolkata as "YYYY-MM-DD" (for grouping orders by business day). */
export function kolkataDateKey(v: DateInput): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: CITY_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(toDate(v));
  const get = (t: string) => parts.find((p) => p.type === t)?.value ?? '';
  return `${get('year')}-${get('month')}-${get('day')}`;
}

/** "5 min ago" / "in 2 hours". */
export function formatRelative(
  v: DateInput,
  now: DateInput = Date.now(),
  lang: AppLanguage | string = 'en',
): string {
  const diffSec = Math.round((toDate(v).getTime() - toDate(now).getTime()) / 1000);
  const abs = Math.abs(diffSec);
  const rtf = new Intl.RelativeTimeFormat(intlLocale(lang), { numeric: 'auto' });
  if (abs < 60) return rtf.format(diffSec, 'second');
  if (abs < 3600) return rtf.format(Math.round(diffSec / 60), 'minute');
  if (abs < 86_400) return rtf.format(Math.round(diffSec / 3600), 'hour');
  return rtf.format(Math.round(diffSec / 86_400), 'day');
}

/** Whole seconds remaining until `deadline` (never negative). */
export function secondsUntil(deadline: DateInput, now: DateInput = Date.now()): number {
  return Math.max(0, Math.ceil((toDate(deadline).getTime() - toDate(now).getTime()) / 1000));
}

/** "0:45" / "12:05" for countdowns. */
export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const m = Math.floor(s / 60);
  return `${m}:${String(s % 60).padStart(2, '0')}`;
}
