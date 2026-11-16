import { describe, expect, it } from 'vitest';
import {
  formatIndianMobile,
  isValidIndianMobile,
  maskIndianMobile,
  normalizeIndianMobile,
  toNationalMobile,
} from './phone';

describe('normalizeIndianMobile', () => {
  it.each([
    ['9876543210', '+919876543210'],
    ['98765 43210', '+919876543210'],
    ['+91 98765-43210', '+919876543210'],
    ['919876543210', '+919876543210'],
    ['09876543210', '+919876543210'],
    ['6000000000', '+916000000000'],
    ['(+91) 7000 000 000', '+917000000000'],
  ])('%s → %s', (input, expected) => {
    expect(normalizeIndianMobile(input)).toBe(expected);
  });

  it.each(['5876543210', '1234567890', '987654321', '98765432101', '+14155550100', '+44 7700 900123', 'abc', ''])(
    'rejects %j',
    (input) => {
      expect(normalizeIndianMobile(input)).toBeNull();
      expect(isValidIndianMobile(input)).toBe(false);
    },
  );
});

describe('display helpers', () => {
  it('formats and masks', () => {
    expect(formatIndianMobile('+919876543210')).toBe('+91 98765 43210');
    expect(maskIndianMobile('+919876543210')).toBe('+91 ••••• •3210');
    expect(toNationalMobile('+919876543210')).toBe('9876543210');
  });
});
