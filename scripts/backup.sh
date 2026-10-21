#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# Logical backup of the rovo database (doc 23 §3.1 L4, local/compose form).
#
#   scripts/backup.sh [--stack dev|prod] [--container NAME] [--url URL] [--dir DIR]
#
# Source (first match): --container / PG_CONTAINER, --url / PG_URL, --stack
# (compose postgis service; default dev), ROVO_DATABASE_URL.
#
# Produces in DIR (default ./backups):
#   rovo-<UTC ts>.dump.gz         pg_dump custom format (-Fc -Z0), gzip-compressed
#   rovo-<UTC ts>.globals.sql.gz  roles/tablespaces without passwords (best effort)
#   rovo-<UTC ts>.meta            key=value: versions, migration version, row counts
#   rovo-<UTC ts>.sha256          sha256sum-compatible manifest of the above
#   rovo_backup.prom              textfile metrics (rovo_backup_last_success_timestamp_seconds, ...)
# Retention: deletes sets older than BACKUP_RETENTION_DAYS (default 7) but always
# keeps the newest BACKUP_KEEP_MIN (default 3).
#
# Cloud: the same script runs as a scheduled ECS task writing to the Object-Locked
# bucket in rovo-audit, with age encryption on top (doc 23 §3.1).
set -euo pipefail
SCRIPT_NAME=backup
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

stack=""
dir="${BACKUP_DIR:-$ROVO_ROOT/backups}"
retention_days="${BACKUP_RETENTION_DAYS:-7}"
keep_min="${BACKUP_KEEP_MIN:-3}"
key_tables="${ROVO_BACKUP_KEY_TABLES:-cities users restaurants menu_items orders order_status_history payments refunds deliveries ledger_accounts ledger_journals ledger_postings audit_logs}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --stack) stack="$2"; shift 2 ;;
    --container) PG_CONTAINER="$2"; shift 2 ;;
    --url) PG_URL="$2"; shift 2 ;;
    --dir) dir="$2"; shift 2 ;;
    -h | --help) sed -n '3,22p' "$0"; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
done

need docker gzip awk
if [[ -z "${PG_CONTAINER:-}" && -z "${PG_URL:-}" ]]; then
  if [[ -n "$stack" || -z "${ROVO_DATABASE_URL:-}" ]]; then
    PG_CONTAINER="$(compose_pg_container "${stack:-dev}")"
    [[ -n "$PG_CONTAINER" ]] || die "postgis service of the ${stack:-dev} stack is not running (make up / make prod-up)"
  else
    PG_URL="$ROVO_DATABASE_URL"
  fi
fi
export PG_CONTAINER PG_URL

mkdir -p "$dir"
chmod 700 "$dir" 2>/dev/null || true
ts="$(date -u +%Y%m%dT%H%M%SZ)"
base="$dir/rovo-$ts"
start="$(date +%s)"

log "checking source database"
server_version="$(psql_q 'SHOW server_version')" || die "cannot connect to the source database"
db_name="$(psql_q 'SELECT current_database()')"

# Hold one REPEATABLE READ transaction open and export its snapshot: pg_dump
# uses it, and the migration version and key-table row counts are read in the
# same snapshot, so verify-restore.sh can demand exact equality even on a live DB.
fifo_dir="$(mktemp -d)"
mkfifo "$fifo_dir/in" "$fifo_dir/out"
pg_tool psql -X -q -At -v ON_ERROR_STOP=1 <"$fifo_dir/in" >"$fifo_dir/out" 2>"$fifo_dir/err" &
snap_pid=$!
exec 3>"$fifo_dir/in" 4<"$fifo_dir/out"
cleanup_snapshot() {
  { exec 3>&- 4<&-; } 2>/dev/null || true
  wait "$snap_pid" 2>/dev/null || true
  rm -rf "$fifo_dir"
}
trap cleanup_snapshot EXIT
snap_query() {
  local line
  printf '%s\n' "$1" >&3
  IFS= read -r -t 60 line <&4 || die "snapshot session did not answer: $(cat "$fifo_dir/err" 2>/dev/null)"
  printf '%s' "$line"
}
snap_query "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SELECT 1;" >/dev/null
snapshot="$(snap_query "SELECT pg_export_snapshot();")"
[[ "$snapshot" =~ ^[0-9A-F-]+$ ]] || die "could not export a snapshot (got: $snapshot)"

migration_version="none"
if [[ "$(snap_query "SELECT to_regclass('public.goose_db_version') IS NOT NULL;")" == "t" ]]; then
  migration_version="$(snap_query 'SELECT COALESCE(max(version_id), 0) FROM goose_db_version WHERE is_applied;')"
fi
row_counts=""
for t in $key_tables; do
  [[ "$t" =~ ^[a-z_][a-z0-9_]*$ ]] || die "invalid table name in ROVO_BACKUP_KEY_TABLES: $t"
  if [[ "$(snap_query "SELECT to_regclass('public.$t') IS NOT NULL;")" == "t" ]]; then
    row_counts+="rows.$t=$(snap_query "SELECT count(*) FROM public.$t;")"$'\n'
  fi
done

log "dumping $db_name (PostgreSQL $server_version) to ${base}.dump.gz"
pg_tool pg_dump --snapshot="$snapshot" --format=custom --compress=0 --no-owner --no-privileges \
  | gzip -6 >"${base}.dump.gz.partial"
mv "${base}.dump.gz.partial" "${base}.dump.gz"
snap_query "COMMIT; SELECT 1;" >/dev/null
cleanup_snapshot
trap - EXIT

# Sanity: the archive must list a TOC (doc 23 §3.1 integrity).
toc_entries="$(gzip -dc "${base}.dump.gz" | pg_tool_nodb pg_restore --list | grep -cv '^;' || true)"
[[ "$toc_entries" -gt 0 ]] || die "dump has an empty table of contents"

if pg_tool pg_dumpall --globals-only --no-role-passwords 2>/dev/null | gzip -6 >"${base}.globals.sql.gz.partial"; then
  mv "${base}.globals.sql.gz.partial" "${base}.globals.sql.gz"
else
  rm -f "${base}.globals.sql.gz.partial"
  warn "globals dump skipped (needs superuser; RDS master secret in the cloud)"
fi

pg_dump_version="$(pg_tool_nodb pg_dump --version | awk '{print $3}')"
{
  echo "format=rovo-backup/1"
  echo "created_at=$ts"
  echo "database=$db_name"
  echo "server_version=$server_version"
  echo "pg_dump_version=$pg_dump_version"
  echo "migration_version=$migration_version"
  echo "toc_entries=$toc_entries"
  printf '%s' "$row_counts"
} >"${base}.meta"

(
  cd "$dir"
  for f in "rovo-$ts.dump.gz" "rovo-$ts.globals.sql.gz" "rovo-$ts.meta"; do
    [[ -f "$f" ]] && printf '%s  %s\n' "$(sha256_file "$f")" "$f"
  done
) >"${base}.sha256"

size="$(file_size "${base}.dump.gz")"
duration=$(($(date +%s) - start))

# Retention: newest first; keep >= keep_min sets, drop sets older than the cutoff.
cutoff="$(date -u -d "-${retention_days} days" +%Y%m%dT%H%M%SZ 2>/dev/null || date -u -v-"${retention_days}"d +%Y%m%dT%H%M%SZ)"
i=0
for f in $(find "$dir" -maxdepth 1 -name 'rovo-*.dump.gz' | sort -r); do
  i=$((i + 1))
  set_ts="$(basename "$f" .dump.gz)"
  set_ts="${set_ts#rovo-}"
  if [[ $i -gt $keep_min && "$set_ts" < "$cutoff" ]]; then
    log "retention: removing set $set_ts"
    rm -f "$dir/rovo-$set_ts".{dump.gz,globals.sql.gz,meta,sha256}
  fi
done

cat >"$dir/rovo_backup.prom" <<EOF
# HELP rovo_backup_last_success_timestamp_seconds Unix time of the last successful backup.
# TYPE rovo_backup_last_success_timestamp_seconds gauge
rovo_backup_last_success_timestamp_seconds $(date +%s)
# HELP rovo_backup_size_bytes Size of the last backup archive.
# TYPE rovo_backup_size_bytes gauge
rovo_backup_size_bytes $size
# HELP rovo_backup_duration_seconds Duration of the last backup.
# TYPE rovo_backup_duration_seconds gauge
rovo_backup_duration_seconds $duration
EOF

ok "backup rovo-$ts: $size bytes, $toc_entries TOC entries, migration $migration_version, ${duration}s"
echo "${base}.dump.gz"
