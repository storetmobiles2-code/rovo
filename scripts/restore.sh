#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Restore a rovo logical backup (made by scripts/backup.sh) into a database.
#
#   scripts/restore.sh <rovo-TS.dump.gz> [--stack dev|prod | --container NAME | --url URL]
#                      [--clean] [--yes] [--no-verify]
#
#   --clean      wipe the target's public schema first (required if it is not empty)
#   --yes        do not ask for confirmation
#   --no-verify  skip the sha256 manifest check (not recommended)
#
# The checksum manifest is verified before anything is touched. The restore runs
# in a single transaction (all-or-nothing). Owners/grants are not restored; the
# app roles are re-granted by migrations/IaC. After a restore of real data:
# replay the erasure ledger and run payment reconciliation (doc 23 §6, §9).
set -euo pipefail
SCRIPT_NAME=restore
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

dump=""
stack=""
clean=0
yes=0
verify=1
while [[ $# -gt 0 ]]; do
  case "$1" in
    --stack) stack="$2"; shift 2 ;;
    --container) PG_CONTAINER="$2"; shift 2 ;;
    --url) PG_URL="$2"; shift 2 ;;
    --clean) clean=1; shift ;;
    --yes | -y) yes=1; shift ;;
    --no-verify) verify=0; shift ;;
    -h | --help) sed -n '3,17p' "$0"; exit 0 ;;
    -*) die "unknown option: $1" ;;
    *) [[ -z "$dump" ]] || die "only one dump file"; dump="$1"; shift ;;
  esac
done

[[ -n "$dump" ]] || die "usage: restore.sh <rovo-TS.dump.gz> [--stack dev|prod|--container NAME|--url URL] [--clean] [--yes]"
[[ -f "$dump" ]] || die "no such file: $dump"
need docker gzip

if [[ -z "${PG_CONTAINER:-}" && -z "${PG_URL:-}" ]]; then
  PG_CONTAINER="$(compose_pg_container "${stack:-dev}")"
  [[ -n "$PG_CONTAINER" ]] || die "postgis service of the ${stack:-dev} stack is not running"
fi
export PG_CONTAINER PG_URL

dump_dir="$(cd "$(dirname "$dump")" && pwd)"
dump_name="$(basename "$dump")"
manifest="$dump_dir/${dump_name%.dump.gz}.sha256"

if [[ $verify -eq 1 ]]; then
  [[ -f "$manifest" ]] || die "manifest $manifest not found (use --no-verify to skip)"
  expected="$(awk -v f="$dump_name" '$2 == f {print $1}' "$manifest")"
  [[ -n "$expected" ]] || die "$dump_name is not listed in $manifest"
  actual="$(sha256_file "$dump")"
  [[ "$expected" == "$actual" ]] || die "checksum mismatch for $dump_name (expected $expected, got $actual)"
  ok "checksum verified ($actual)"
fi
gzip -t "$dump" || die "gzip integrity check failed"

target="$(psql_q "SELECT current_database() || ' on ' || coalesce(inet_server_addr()::text, 'local socket')")" \
  || die "cannot connect to the target database"
# User tables in any non-system schema, ignoring tables owned by extensions
# (spatial_ref_sys, tiger/topology from the postgis image's init script).
tables="$(psql_q "
  SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
  WHERE c.relkind IN ('r', 'p') AND n.nspname NOT IN ('pg_catalog', 'information_schema')
    AND n.nspname NOT LIKE 'pg\\_%'
    AND NOT EXISTS (SELECT 1 FROM pg_depend d
                    WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e')")"

if [[ "$tables" -gt 0 && $clean -eq 0 ]]; then
  die "target $target has $tables user tables; re-run with --clean to replace it"
fi
if [[ $yes -eq 0 ]]; then
  [[ -t 0 ]] || die "refusing to restore without --yes in a non-interactive shell"
  read -r -p "Restore $dump_name into $target (existing tables: $tables, clean: $clean)? Type 'restore' to continue: " answer
  [[ "$answer" == "restore" ]] || die "aborted"
fi

# Wipe (with --clean) and restore in ONE transaction: if anything fails, the
# target is left exactly as it was. The archive is converted to SQL by
# pg_restore (no connection) and applied by psql --single-transaction.
wipe_sql="DO \$\$ DECLARE s text; BEGIN
  FOR s IN SELECT nspname FROM pg_namespace
           WHERE nspname NOT IN ('pg_catalog', 'information_schema') AND nspname NOT LIKE 'pg\\_%'
  LOOP EXECUTE format('DROP SCHEMA %I CASCADE', s); END LOOP;
END \$\$;
CREATE SCHEMA public;"

log "restoring $dump_name into $target$([[ $clean -eq 1 ]] && echo ' (replacing existing objects)')"
start="$(date +%s)"
{
  if [[ $clean -eq 1 ]]; then printf '%s\n' "$wipe_sql"; fi
  gzip -dc "$dump" | pg_tool_nodb pg_restore --no-owner --no-privileges --file=-
} | pg_tool psql -X -q -v ON_ERROR_STOP=1 --single-transaction >/dev/null
psql_q "ANALYZE" >/dev/null
ok "restore finished in $(($(date +%s) - start))s"
log "reminder (real data only): replay the erasure ledger and reconcile payments for the gap window (docs/runbooks/backup-restore.md)"
