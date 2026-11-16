const MAP: Record<string, string> = {
  a: 'á', b: 'ƀ', c: 'ç', d: 'ð', e: 'é', f: 'ƒ', g: 'ĝ', h: 'ĥ', i: 'í', j: 'ĵ', k: 'ķ', l: 'ļ',
  m: 'ɱ', n: 'ñ', o: 'ó', p: 'þ', q: 'ǫ', r: 'ŕ', s: 'š', t: 'ţ', u: 'ú', v: 'ṽ', w: 'ŵ', x: 'ẋ',
  y: 'ý', z: 'ž', A: 'Á', B: 'Ɓ', C: 'Ç', D: 'Ð', E: 'É', F: 'Ƒ', G: 'Ĝ', H: 'Ĥ', I: 'Í', J: 'Ĵ',
  K: 'Ķ', L: 'Ļ', M: 'Ṁ', N: 'Ñ', O: 'Ó', P: 'Þ', Q: 'Ǫ', R: 'Ŕ', S: 'Š', T: 'Ţ', U: 'Ú', V: 'Ṽ',
  W: 'Ŵ', X: 'Ẋ', Y: 'Ý', Z: 'Ž',
};

/** Accents letters and pads ~40 %, leaving {{interpolations}}, $t(nesting) and <tags> intact. */
export function pseudoLocalizeString(input: string): string {
  const parts = input.split(/(\{\{[^}]+\}\}|\$t\([^)]*\)|<[^>]+>)/g);
  const body = parts
    .map((part, i) => (i % 2 === 1 ? part : part.replace(/[a-zA-Z]/g, (ch) => MAP[ch] ?? ch)))
    .join('');
  const pad = '~'.repeat(Math.max(1, Math.round(input.length * 0.4)));
  return `[${body}${pad}]`;
}

export function pseudoLocalize<T>(value: T): T {
  if (typeof value === 'string') return pseudoLocalizeString(value) as T;
  if (Array.isArray(value)) return value.map((v) => pseudoLocalize(v)) as T;
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>).map(([k, v]) => [k, pseudoLocalize(v)]),
    ) as T;
  }
  return value;
}
