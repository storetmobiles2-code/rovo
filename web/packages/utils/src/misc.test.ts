import { describe, expect, it } from 'vitest';
import { formatAddress } from './address';
import {
  formatDate,
  formatDateTime,
  formatDuration,
  formatRelative,
  formatTime,
  kolkataDateKey,
  secondsUntil,
} from './datetime';
import { newIdempotencyKey, uuidv7 } from './id';
import { isTelanganaPinCode, isValidPinCode } from './pincode';

describe('PIN codes', () => {
  it('validates Indian PIN codes', () => {
    expect(isValidPinCode('509001')).toBe(true);
    expect(isValidPinCode('509 001')).toBe(true);
    expect(isValidPinCode('009001')).toBe(false);
    expect(isValidPinCode('50900')).toBe(false);
    expect(isTelanganaPinCode('509001')).toBe(true);
    expect(isTelanganaPinCode('560001')).toBe(false);
  });
});

describe('formatAddress', () => {
  const addr = {
    houseNo: 'Flat 2',
    buildingStreet: 'Sai Residency',
    landmark: 'Clock Tower',
    localityName: 'New Town',
    city: 'Mahabubnagar',
    pinCode: '509001',
  };
  it('formats the full address', () => {
    expect(formatAddress(addr)).toBe(
      'Flat 2, Sai Residency, Near Clock Tower, New Town, Mahabubnagar 509001',
    );
  });
  it('formats the short address', () => {
    expect(formatAddress(addr, { short: true })).toBe('New Town · near Clock Tower');
  });
  it('skips empty parts', () => {
    expect(formatAddress({ houseNo: '12', landmark: 'near Bus Stand', pinCode: '509001' })).toBe(
      '12, near Bus Stand, 509001',
    );
  });
});

describe('Asia/Kolkata dates', () => {
  // 2026-10-04T20:35:00Z is 2026-10-05 02:05 IST — a different calendar day.
  const ts = '2026-10-04T20:35:00Z';
  it('formats in IST regardless of the device time zone', () => {
    expect(formatTime(ts)).toMatch(/^2:05\s?am$/i);
    expect(formatDate(ts)).toBe('5 Oct 2026');
    expect(formatDateTime(ts)).toMatch(/5 Oct 2026.*2:05/);
    expect(kolkataDateKey(ts)).toBe('2026-10-05');
  });
  it('formats relative times and countdowns', () => {
    const now = Date.parse('2026-10-04T20:40:00Z');
    expect(formatRelative(ts, now)).toBe('5 minutes ago');
    expect(secondsUntil(now + 44_100, now)).toBe(45);
    expect(secondsUntil(now - 1000, now)).toBe(0);
    expect(formatDuration(45)).toBe('0:45');
    expect(formatDuration(725)).toBe('12:05');
  });
  it('supports Telugu', () => {
    expect(formatDate(ts, 'te')).toContain('2026');
  });
});

describe('uuidv7', () => {
  it('produces RFC 9562 v7 ids ordered by time', () => {
    const a = uuidv7(1_700_000_000_000);
    const b = uuidv7(1_700_000_000_001);
    expect(a).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    expect(a < b).toBe(true);
    expect(newIdempotencyKey()).not.toBe(newIdempotencyKey());
  });
});
