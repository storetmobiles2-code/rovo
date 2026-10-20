#!/usr/bin/env node
// Checks each app's initial JS budget (gzip) after `vite build` (doc 17 §12).
// Initial JS = the entry chunk index-*.js + the eagerly-loaded i18n namespace chunks that
// block first paint (common/errors + the app's own namespaces). Lazy route chunks don't count.
import { gzipSync } from 'node:zlib';
import { readdirSync, readFileSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const web = join(dirname(fileURLToPath(import.meta.url)), '..');

// [app, warnKB, failKB]. Customer is the tightest (doc 17 §12).
const BUDGETS = [
  ['customer', 170, 200],
  ['restaurant', 200, 240],
  ['rider', 200, 240],
  ['admin', 300, 350],
];

function gz(path) {
  return gzipSync(readFileSync(path)).length;
}

let failed = false;
const rows = [];
for (const [app, warnKB, failKB] of BUDGETS) {
  const assets = join(web, 'apps', app, 'dist', 'assets');
  if (!existsSync(assets)) {
    rows.push(`  ${app.padEnd(11)} (not built — run \`pnpm --filter @rovo/${app} build\`)`);
    continue;
  }
  const files = readdirSync(assets).filter((f) => f.endsWith('.js'));
  const entry = files.find((f) => f.startsWith('index-'));
  const initial = files.filter((f) => f.startsWith('index-') || /^(common|errors|auth|customer|restaurant|rider|admin)-/.test(f));
  const entryGz = entry ? gz(join(assets, entry)) : 0;
  const initialGz = initial.reduce((sum, f) => sum + gz(join(assets, f)), 0);
  const kb = initialGz / 1024;
  const status = kb > failKB ? 'FAIL' : kb > warnKB ? 'warn' : 'ok';
  if (status === 'FAIL') failed = true;
  rows.push(
    `  ${app.padEnd(11)} initial ${kb.toFixed(1).padStart(6)} KB gzip  (entry ${(entryGz / 1024).toFixed(1)} KB)  budget ${warnKB}/${failKB} KB  [${status}]`,
  );
}

console.log('Bundle size (initial JS, gzip):');
for (const r of rows) console.log(r);
if (failed) {
  console.error('\nBundle size check FAILED: an app exceeds its hard budget.');
  process.exit(1);
}
console.log('\nAll built apps are within budget.');
