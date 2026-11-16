import { type AppLanguage, intlLocale } from './locale';

/**
 * Money is always an integer number of paise (1 ₹ = 100 paise).
 * Arithmetic is integer-only; floats are never used to add, multiply or round money.
 * The API shape is `{ amountPaise, currency: 'INR' }`.
 */
export type Paise = number;

export interface Money {
  amountPaise: Paise;
  currency: 'INR';
}

export class MoneyError extends Error {
  override name = 'MoneyError';
}

export function assertPaise(value: number): asserts value is Paise {
  if (!Number.isSafeInteger(value)) {
    throw new MoneyError(`Paise must be a safe integer, got ${String(value)}`);
  }
}

export function money(amountPaise: Paise): Money {
  assertPaise(amountPaise);
  return { amountPaise, currency: 'INR' };
}

/**
 * Parses rupees ("149", "149.5", "1,49,000.50", "₹ 99") into paise without float maths.
 * Returns `null` for invalid input or more than two decimal places.
 */
export function rupeesToPaise(input: string | number): Paise | null {
  let text: string;
  if (typeof input === 'number') {
    if (!Number.isFinite(input)) return null;
    text = String(input);
    if (/e/i.test(text)) return null;
  } else {
    text = input.replace(/[₹,\s]/g, '').replace(/^Rs\.?/i, '');
  }
  const m = /^(-)?(\d+)(?:\.(\d{0,2}))?$/.exec(text);
  if (!m) return null;
  const sign = m[1] ? -1 : 1;
  const whole = Number(m[2]);
  const frac = Number((m[3] ?? '').padEnd(2, '0'));
  const paise = sign * (whole * 100 + frac);
  return Number.isSafeInteger(paise) ? paise : null;
}

/** "14950" → "149.50"; "14900" → "149" (or "149.00" with `fixed`). Exact, string-based. */
export function paiseToRupeesString(paise: Paise, opts: { fixed?: boolean } = {}): string {
  assertPaise(paise);
  const negative = paise < 0;
  const abs = Math.abs(paise);
  const rupees = Math.trunc(abs / 100);
  const rest = abs % 100;
  const body =
    rest === 0 && !opts.fixed ? String(rupees) : `${rupees}.${String(rest).padStart(2, '0')}`;
  return negative ? `-${body}` : body;
}

export function addPaise(...values: Paise[]): Paise {
  let total = 0;
  for (const v of values) {
    assertPaise(v);
    total += v;
  }
  assertPaise(total);
  return total;
}

export function multiplyPaise(paise: Paise, quantity: number): Paise {
  assertPaise(paise);
  if (!Number.isSafeInteger(quantity)) throw new MoneyError('Quantity must be an integer');
  const result = paise * quantity;
  assertPaise(result);
  return result;
}

/**
 * Percentage of an amount in basis points (1 % = 100 bp), rounded half-up to whole paise.
 * Example: 5 % GST on ₹149 → percentOfPaise(14900, 500) = 745.
 */
export function percentOfPaise(paise: Paise, basisPoints: number): Paise {
  assertPaise(paise);
  if (!Number.isSafeInteger(basisPoints)) throw new MoneyError('Basis points must be an integer');
  const numerator = BigInt(paise) * BigInt(basisPoints);
  const denominator = 10_000n;
  const negative = numerator < 0n;
  const abs = negative ? -numerator : numerator;
  let q = abs / denominator;
  if ((abs % denominator) * 2n >= denominator) q += 1n;
  const result = Number(negative ? -q : q);
  assertPaise(result);
  return result;
}

/** Sums display totals of a list of `{ amountPaise }` objects. */
export function sumMoney(items: ReadonlyArray<{ amountPaise: Paise }>): Paise {
  return addPaise(...items.map((i) => i.amountPaise));
}

const formatterCache = new Map<string, Intl.NumberFormat>();

function currencyFormatter(locale: string, fractionDigits: 0 | 2): Intl.NumberFormat {
  const key = `${locale}|${fractionDigits}`;
  let f = formatterCache.get(key);
  if (!f) {
    f = new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: 'INR',
      minimumFractionDigits: fractionDigits,
      maximumFractionDigits: fractionDigits,
      numberingSystem: 'latn',
    });
    formatterCache.set(key, f);
  }
  return f;
}

/**
 * ₹ with Indian (lakh/crore) grouping: 12345650 → "₹1,23,456.50", 14900 → "₹149".
 * Whole-rupee amounts drop the decimals unless `fixed` is set.
 */
export function formatMoney(
  value: Paise | Money,
  lang: AppLanguage | string = 'en',
  opts: { fixed?: boolean } = {},
): string {
  const paise = typeof value === 'number' ? value : value.amountPaise;
  assertPaise(paise);
  const digits: 0 | 2 = paise % 100 === 0 && !opts.fixed ? 0 : 2;
  // Intl accepts an exact decimal string, so no float rounding can creep in.
  const decimal = paiseToRupeesString(paise, { fixed: true }) as Intl.StringNumericLiteral;
  return currencyFormatter(intlLocale(lang), digits).format(decimal);
}

/** Plain number with lakh/crore grouping (no currency). */
export function formatNumber(n: number, lang: AppLanguage | string = 'en'): string {
  return new Intl.NumberFormat(intlLocale(lang), { numberingSystem: 'latn' }).format(n);
}
