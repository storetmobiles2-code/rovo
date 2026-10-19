/** Indian PIN codes: 6 digits, first digit 1–9. Mahabubnagar is 509xxx (Telangana: 50xxxx). */
const PIN = /^[1-9]\d{5}$/;

export function normalizePinCode(input: string): string {
  return input.replace(/\s/g, '');
}

export function isValidPinCode(input: string): boolean {
  return PIN.test(normalizePinCode(input));
}

/** Telangana PIN codes start with 50 [ASSUMPTION – doc 17 §7: UI-only hint, the server decides serviceability]. */
export function isTelanganaPinCode(input: string): boolean {
  const pin = normalizePinCode(input);
  return PIN.test(pin) && pin.startsWith('50');
}
