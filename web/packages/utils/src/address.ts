/** Address fields as returned by `GET /me/addresses` (doc 11 §2.2). */
export interface AddressLike {
  label?: string | null;
  houseNo?: string | null;
  buildingStreet?: string | null;
  areaText?: string | null;
  localityName?: string | null;
  landmark?: string | null;
  city?: string | null;
  pinCode?: string | null;
}

function clean(parts: ReadonlyArray<string | null | undefined>): string[] {
  return parts.map((p) => (p ?? '').trim()).filter((p) => p.length > 0);
}

/**
 * Full: "Flat 2, Sai Residency, Near Clock Tower, New Town, Mahabubnagar 509001"
 * Short: "New Town · near Clock Tower"
 */
export function formatAddress(addr: AddressLike, opts: { short?: boolean } = {}): string {
  const area = addr.localityName ?? addr.areaText;
  if (opts.short) {
    const landmark = addr.landmark?.trim();
    return clean([area, landmark ? `near ${landmark.replace(/^near\s+/i, '')}` : null]).join(' · ');
  }
  const landmark = addr.landmark?.trim();
  const cityPin = clean([addr.city, addr.pinCode]).join(' ');
  return clean([
    addr.houseNo,
    addr.buildingStreet,
    landmark ? (/^near\b/i.test(landmark) ? landmark : `Near ${landmark}`) : null,
    area,
    cityPin,
  ]).join(', ');
}
