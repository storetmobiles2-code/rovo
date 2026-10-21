import { describe, expect, it } from 'vitest';
import {
  addPaise,
  formatMoney,
  formatNumber,
  MoneyError,
  multiplyPaise,
  paiseToRupeesString,
  percentOfPaise,
  rupeesToPaise,
  sumMoney,
} from './money';

describe('rupeesToPaise', () => {
  it.each([
    ['149', 14900],
    ['149.5', 14950],
    ['149.50', 14950],
    ['0.05', 5],
    ['1,23,456.50', 12345650],
    ['₹ 99', 9900],
    ['Rs. 10', 1000],
    [149.5, 14950],
    [0.1, 10],
    ['-20', -2000],
  ] as const)('%s → %s', (input, expected) => {
    expect(rupeesToPaise(input)).toBe(expected);
  });

  it.each(['', 'abc', '1.234', '1.2.3', '1e5'])('rejects %j', (input) => {
    expect(rupeesToPaise(input)).toBeNull();
  });

  it('avoids float drift that 0.1 + 0.2 would cause', () => {
    expect(addPaise(rupeesToPaise('0.1')!, rupeesToPaise('0.2')!)).toBe(30);
  });
});

describe('paiseToRupeesString', () => {
  it('drops zero decimals unless fixed', () => {
    expect(paiseToRupeesString(14900)).toBe('149');
    expect(paiseToRupeesString(14900, { fixed: true })).toBe('149.00');
    expect(paiseToRupeesString(14905)).toBe('149.05');
    expect(paiseToRupeesString(-5)).toBe('-0.05');
  });
});

describe('integer arithmetic', () => {
  it('adds, multiplies and sums', () => {
    expect(addPaise(14900, 2500, 745)).toBe(18145);
    expect(multiplyPaise(14900, 3)).toBe(44700);
    expect(sumMoney([{ amountPaise: 100 }, { amountPaise: 250 }])).toBe(350);
  });

  it('rounds percentages half-up to whole paise', () => {
    expect(percentOfPaise(14900, 500)).toBe(745); // 5 % GST
    expect(percentOfPaise(999, 500)).toBe(50); // 49.95 → 50
    expect(percentOfPaise(989, 500)).toBe(49); // 49.45 → 49
    expect(percentOfPaise(-999, 500)).toBe(-50);
  });

  it('refuses fractional paise', () => {
    expect(() => addPaise(1.5)).toThrow(MoneyError);
    expect(() => multiplyPaise(100, 1.5)).toThrow(MoneyError);
  });
});

describe('formatMoney', () => {
  it('uses ₹ and Indian grouping', () => {
    expect(formatMoney(12345650)).toBe('₹1,23,456.50');
    expect(formatMoney(14900)).toBe('₹149');
    expect(formatMoney({ amountPaise: 5, currency: 'INR' })).toBe('₹0.05');
    expect(formatMoney(14900, 'en', { fixed: true })).toBe('₹149.00');
  });

  it('formats Telugu with Latin digits and lakh grouping', () => {
    const s = formatMoney(12345650, 'te');
    expect(s).toContain('1,23,456.50');
    expect(s).toContain('₹');
  });

  it('formats plain numbers with lakh grouping', () => {
    expect(formatNumber(1234567)).toBe('12,34,567');
  });
});
