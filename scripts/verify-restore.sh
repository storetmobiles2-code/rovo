#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Restore-and-verify drill (R50, doc 23 §7 D0) for the local/compose stack.
#
#   scripts/verify-restore.sh [rovo-TS.dump.gz]     (default: newest in BACKUP_DIR)
#
# 1. checks the sha256 manifest, 2. restores into a THROWAWAY PostGIS container
# (no network, tmpfs data, destroyed afterwards), 3. verifies:
#   - PostGIS works (postgis_full_version)
#   - migration status: goose version == version recorded at backup time, all
#     recorded migrations applied; warns if behind the repo's newest migration;
#     optionally runs `rovo migrate status` from ROVO_IMAGE (VERIFY_WITH_ROVO=1)
#   - row counts of key tables == counts recorded in the same snapshot as the dump
#   - ledger invariants: every journal has >= 2 postings and sums to zero;
#     postings sum to zero per currency; no postings without a journal
# Exits non-zero on any failure. Prints the restore duration (RTO evidence).
set -euo pipefail
SCRIPT_NAME=verify-restore
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

dir="${BACKUP_DIR:-$ROVO_ROOT/backups}"
dump="${1:-}"
if [[ -z "$dump" ]]; then
  dump="$(find "$dir" -maxdepth 1 -name 'rovo-*.dump.gz' 2>/dev/null | sort | tail -n1 || true)"
  [[ -n "$dump" ]] || die "no backups found in $dir (run make backup first)"
fi
[[ -f "$dump" ]] || die "no such file: $dump"
meta="${dump%.dump.gz}.meta"
[[ -f "$meta" ]] || die "metadata file $meta not found"
need docker gzip awk

meta_get() { awk -F= -v k="$1" '$1 == k {sub(/^[^=]*=/, ""); print; exit}' "$meta"; }

image="$(default_postgres_image)"
name="rovo-verify-restore-$$-$RANDOM"
failures=0
check_fail() {
  fail "$*"
  failures=$((failures + 1))
}

cleanup() { docker rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT

log "starting throwaway $image as $name"
docker run -d --rm --name "$name" --network none \
  --tmpfs /var/lib/postgresql/data:rw,size=2g \
  --tmpfs /docker-entrypoint-initdb.d:ro \
  -e POSTGRES_USER="$ROVO_PG_USER" -e POSTGRES_PASSWORD=verify -e POSTGRES_DB="$ROVO_PG_DB" \
  "$image" postgres -c fsync=off -c full_page_writes=off -c synchronous_commit=off >/dev/null

# The entrypoint's temporary init server only listens on the socket, so a TCP
# probe succeeds only once the final server is up.
for _ in $(seq 1 90); do
  if docker exec "$name" pg_isready -q -h 127.0.0.1 -U "$ROVO_PG_USER" -d "$ROVO_PG_DB" 2>/dev/null; then
    break
  fi
  sleep 1
done
docker exec "$name" pg_isready -q -h 127.0.0.1 -U "$ROVO_PG_USER" -d "$ROVO_PG_DB" || die "throwaway database did not start"

export PG_CONTAINER="$name"
unset PG_URL
start="$(date +%s)"
"$ROVO_ROOT/scripts/restore.sh" "$dump" --container "$name" --yes
restore_seconds=$(($(date +%s) - start))

# --- PostGIS -----------------------------------------------------------------
if pgv="$(psql_q 'SELECT postgis_lib_version()' 2>/dev/null)"; then
  ok "PostGIS $pgv"
else
  check_fail "PostGIS is not available in the restored database"
fi

# --- migration status ---------------------------------------------------------
want_mig="$(meta_get migration_version)"
if [[ "$want_mig" == "none" || -z "$want_mig" ]]; then
  warn "backup recorded no goose migration table (scratch or pre-migration database)"
else
  got_mig="$(psql_q "SELECT COALESCE(max(version_id), 0) FROM goose_db_version WHERE is_applied" 2>/dev/null || echo missing)"
  if [[ "$got_mig" == "$want_mig" ]]; then
    ok "migration version $got_mig matches the backup"
  else
    check_fail "migration version: restored $got_mig, backup recorded $want_mig"
  fi
  # Every migration file up to the recorded version must be applied.
  if [[ -d "$ROVO_ROOT/backend/migrations" ]]; then
    repo_versions="$(find "$ROVO_ROOT/backend/migrations" -maxdepth 1 -name '[0-9]*_*.sql' -exec basename {} \; \
      | sed -E 's/^0*([0-9]+)_.*/\1/' | sort -n)"
    applied="$(psql_q "SELECT version_id FROM goose_db_version WHERE is_applied AND version_id > 0 ORDER BY 1" 2>/dev/null || true)"
    missing=""
    for v in $repo_versions; do
      [[ "$v" -le "$want_mig" ]] || continue
      grep -qx "$v" <<<"$applied" || missing+=" $v"
    done
    if [[ -n "$missing" ]]; then
      check_fail "migrations recorded as applied are missing in the restore:$missing"
    else
      ok "all repository migrations <= $want_mig are applied"
    fi
    newest="$(tail -n1 <<<"$repo_versions")"
    if [[ -n "$newest" && "$newest" -gt "$want_mig" ]]; then
      warn "backup is at migration $want_mig; repository is at $newest ('rovo migrate up' needed before serving)"
    fi
  fi
fi
if [[ "${VERIFY_WITH_ROVO:-0}" == "1" ]]; then
  rovo_image="${ROVO_IMAGE:-rovo:local}"
  log "running '$rovo_image migrate status' against the restore"
  if docker run --rm --network "container:$name" \
    -e ROVO_DATABASE_URL="postgres://$ROVO_PG_USER:verify@127.0.0.1:5432/$ROVO_PG_DB?sslmode=disable" \
    "$rovo_image" migrate status; then
    ok "rovo migrate status"
  else
    check_fail "rovo migrate status failed"
  fi
fi

# --- row counts --------------------------------------------------------------
# Read the meta file on fd 3: psql_q (docker exec -i) would swallow stdin.
while IFS='=' read -r -u 3 key want; do
  [[ "$key" == rows.* ]] || continue
  t="${key#rows.}"
  [[ "$t" =~ ^[a-z_][a-z0-9_]*$ ]] || { check_fail "bad table name in meta: $t"; continue; }
  got="$(psql_q "SELECT count(*) FROM public.$t" 2>/dev/null || echo missing)"
  if [[ "$got" == "$want" ]]; then
    ok "rows $t = $got"
  else
    check_fail "rows $t: restored $got, backup recorded $want"
  fi
done 3<"$meta"

# --- ledger invariants (doc 10 DB-D09) -----------------------------------------
if [[ "$(psql_q "SELECT to_regclass('public.ledger_postings') IS NOT NULL AND to_regclass('public.ledger_journals') IS NOT NULL")" == "t" ]]; then
  unbalanced="$(psql_q "
    SELECT count(*) FROM (
      SELECT j.id
      FROM ledger_journals j
      LEFT JOIN ledger_postings p ON p.journal_id = j.id
      GROUP BY j.id
      HAVING COALESCE(sum(p.amount_paise), 0) <> 0 OR count(p.id) < 2
    ) x")"
  if [[ "$unbalanced" == "0" ]]; then
    ok "ledger: every journal balances with >= 2 postings"
  else
    check_fail "ledger: $unbalanced unbalanced journal(s)"
  fi
  nonzero="$(psql_q "SELECT count(*) FROM (SELECT currency FROM ledger_postings GROUP BY currency HAVING sum(amount_paise) <> 0) x")"
  if [[ "$nonzero" == "0" ]]; then
    ok "ledger: postings sum to zero per currency"
  else
    check_fail "ledger: postings do not sum to zero in $nonzero currency(ies)"
  fi
  orphans="$(psql_q "SELECT count(*) FROM ledger_postings p WHERE NOT EXISTS (SELECT 1 FROM ledger_journals j WHERE j.id = p.journal_id)")"
  if [[ "$orphans" == "0" ]]; then
    ok "ledger: no orphan postings"
  else
    check_fail "ledger: $orphans orphan posting(s)"
  fi
  if [[ "$(psql_q "SELECT to_regclass('public.ledger_account_balances') IS NOT NULL")" == "t" ]]; then
    drift="$(psql_q "
      SELECT count(*) FROM ledger_account_balances b
      LEFT JOIN (SELECT account_id, sum(amount_paise) s FROM ledger_postings GROUP BY account_id) p USING (account_id)
      WHERE b.balance_paise NOT IN (COALESCE(p.s, 0), -COALESCE(p.s, 0))")"
    if [[ "$drift" == "0" ]]; then
      ok "ledger: account balances match postings"
    else
      check_fail "ledger: $drift account balance(s) differ from the sum of their postings"
    fi
  fi
else
  warn "no ledger tables in this backup; ledger invariants skipped"
fi

last_order="n/a"
if [[ "$(psql_q "SELECT to_regclass('public.orders') IS NOT NULL")" == "t" ]]; then
  last_order="$(psql_q "SELECT COALESCE(max(created_at)::text, 'none') FROM orders")"
fi

echo
log "backup:          $(basename "$dump") (created $(meta_get created_at), PostgreSQL $(meta_get server_version))"
log "restore time:    ${restore_seconds}s"
log "latest order:    $last_order"
if [[ $failures -gt 0 ]]; then
  die "verify-restore: $failures check(s) failed"
fi
ok "verify-restore: all checks passed"
