#!/usr/bin/env node
// Generates src/schema.d.ts from the OpenAPI contract (ADR-P2-001).
// Uses ../../../openapi/openapi.yaml when it exists; until the Architect lands the spec it
// falls back to scripts/placeholder-openapi.yaml (auth + /me only) so the workspace compiles.
import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const pkg = join(dirname(fileURLToPath(import.meta.url)), '..');
const real = join(pkg, '..', '..', '..', 'openapi', 'openapi.yaml');
const placeholder = join(pkg, 'scripts', 'placeholder-openapi.yaml');
const input = existsSync(real) ? real : placeholder;
const out = join(pkg, 'src', 'schema.d.ts');
const bin = join(pkg, 'node_modules', '.bin', 'openapi-typescript');

console.log(`gen:api: ${relative(process.cwd(), input)} → ${relative(process.cwd(), out)}`);
execFileSync(bin, [input, '-o', out, '--root-types', '--alphabetize'], { stdio: 'inherit' });
if (input === placeholder) {
  console.warn('gen:api: openapi/openapi.yaml not found — generated from the PLACEHOLDER spec.');
}
