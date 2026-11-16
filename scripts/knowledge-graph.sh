#!/usr/bin/env bash
# Regenerate the raw inputs for docs/implementation/knowledge-graph.md
# from the repo's own sources (migrations + OpenAPI). Read-only; prints to stdout.
set -euo pipefail
cd "$(dirname "$0")/.."
echo "## Tables by migration"
for f in backend/migrations/0*.sql; do
  n=$(basename "$f" .sql)
  t=$(grep -oiE 'create table[[:space:]]+(if not exists[[:space:]]+)?[a-z_]+' "$f" \
      | sed -E 's/.*[[:space:]]//' | paste -sd, -)
  [ -n "$t" ] && echo "- $n: $t"
done
echo
echo "## Total tables"
grep -rhoiE 'create table[[:space:]]+(if not exists[[:space:]]+)?[a-z_]+' backend/migrations/*.sql | wc -l
echo
echo "## OpenAPI tag groups"
grep -E '^  - name:' openapi/openapi.yaml | sed 's/^  - name: /- /'
