#!/usr/bin/env node
// Fails (exit 1) when any Telugu catalog is missing a key, has an empty value, a missing
// plural category, or different {{interpolation}} variables than English.
// English (locales/en) is the source of truth. Usage: node scripts/check-i18n.mjs
import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', 'locales');
const SOURCE = 'en';
const TARGETS = ['te'];
const PLURAL = /_(zero|one|two|few|many|other)$/;

function flatten(obj, prefix = '', out = {}) {
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) flatten(v, key, out);
    else out[key] = v;
  }
  return out;
}

const vars = (s) => new Set([...String(s).matchAll(/\{\{\s*([\w.]+)[^}]*\}\}/g)].map((m) => m[1]));
const load = (lng, ns) => flatten(JSON.parse(readFileSync(join(root, lng, `${ns}.json`), 'utf8')));

export function checkCatalogs() {
  const errors = [];
  const namespaces = readdirSync(join(root, SOURCE)).filter((f) => f.endsWith('.json')).map((f) => f.slice(0, -5));
  let keyCount = 0;
  for (const lng of TARGETS) {
    const categories = new Intl.PluralRules(lng).resolvedOptions().pluralCategories;
    for (const ns of namespaces) {
      const en = load(SOURCE, ns);
      let tr;
      try {
        tr = load(lng, ns);
      } catch {
        errors.push(`${lng}/${ns}.json is missing`);
        continue;
      }
      const enBases = new Map();
      for (const key of Object.keys(en)) {
        keyCount++;
        const base = key.replace(PLURAL, '');
        if (!enBases.has(base)) enBases.set(base, en[key]);
      }
      for (const [base, enValue] of enBases) {
        const isPlural = Object.keys(en).some((k) => k !== base && k.replace(PLURAL, '') === base);
        const required = isPlural ? categories.map((c) => `${base}_${c}`) : [base];
        for (const key of required) {
          const value = tr[key];
          if (value === undefined) errors.push(`${lng}/${ns}: missing "${key}"`);
          else if (typeof value === 'string' && value.trim() === '') errors.push(`${lng}/${ns}: empty "${key}"`);
          else {
            const want = vars(enValue);
            const got = vars(value);
            for (const v of want) if (!got.has(v)) errors.push(`${lng}/${ns}: "${key}" lacks {{${v}}}`);
            for (const v of got) if (!want.has(v)) errors.push(`${lng}/${ns}: "${key}" has unknown {{${v}}}`);
          }
        }
      }
      for (const key of Object.keys(tr)) {
        if (!enBases.has(key.replace(PLURAL, ''))) errors.push(`${lng}/${ns}: stale key "${key}" (not in en)`);
      }
    }
  }
  return { errors, namespaces, keyCount };
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  const { errors, namespaces, keyCount } = checkCatalogs();
  if (errors.length) {
    console.error(`i18n:check failed — ${errors.length} problem(s):`);
    for (const e of errors) console.error(`  • ${e}`);
    process.exit(1);
  }
  console.log(`i18n:check ok — ${namespaces.length} namespaces, ${keyCount} English keys, Telugu complete.`);
}
