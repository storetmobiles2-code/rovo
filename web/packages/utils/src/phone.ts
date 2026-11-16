/**
 * Indian mobile numbers (V1 is India-only). E.164 form: "+91" + 10 digits starting 6–9.
 * Customer and partner apps use this light validator (doc 17 §2.2), not libphonenumber.
 */
export type E164 = `+91${string}`;

const MOBILE = /^[6-9]\d{9}$/;

/** Strips formatting and the +91 / 91 / 0 prefixes; returns the 10-digit national number or null. */
export function toNationalMobile(input: string): string | null {
  let digits = input.replace(/[\s\-().]/g, '');
  if (digits.startsWith('+91')) digits = digits.slice(3);
  else if (digits.startsWith('+')) return null; // other country codes are not supported
  else if (digits.length === 12 && digits.startsWith('91')) digits = digits.slice(2);
  else if (digits.length === 11 && digits.startsWith('0')) digits = digits.slice(1);
  return MOBILE.test(digits) ? digits : null;
}

export function normalizeIndianMobile(input: string): E164 | null {
  const national = toNationalMobile(input);
  return national ? `+91${national}` : null;
}

export function isValidIndianMobile(input: string): boolean {
  return toNationalMobile(input) !== null;
}

/** "+919876543210" → "+91 98765 43210". */
export function formatIndianMobile(e164: string): string {
  const national = toNationalMobile(e164);
  if (!national) return e164;
  return `+91 ${national.slice(0, 5)} ${national.slice(5)}`;
}

/** "+919876543210" → "+91 ••••• •3210" (for display when the server did not mask). */
export function maskIndianMobile(e164: string): string {
  const national = toNationalMobile(e164);
  if (!national) return '•••';
  return `+91 ••••• •${national.slice(6)}`;
}
