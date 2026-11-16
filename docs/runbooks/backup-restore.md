# Runbook: Backup & restore

Logical backup/restore of the rovo PostgreSQL database. This is the **L4 layer**
of `docs/planning/23-backup-disaster-recovery.md` §3.1 — provider-independent
dumps, restorable on any PostgreSQL 17 + PostGIS. In the cloud, RDS PITR (L1),
cross-region replicated backups (L2) and manual snapshots (L3) sit above this;
PITR is the first choice for a recent, in-window restore (DR-1b / DR-2).

Scripts: `scripts/backup.sh`, `scripts/restore.sh`, `scripts/verify-restore.sh`
(driven by `make backup`, `make restore`, `make verify-restore`).

## Take a backup

```bash
make backup                 # dev stack, writes to ./backups
# or directly:
scripts/backup.sh --stack prod --dir ./backups
```

Each run produces a timestamped set in the backup dir:

- `rovo-<ts>.dump.gz` — `pg_dump -Fc` custom format, gzip-compressed
- `rovo-<ts>.globals.sql.gz` — roles/tablespaces (no passwords), best effort
- `rovo-<ts>.meta` — versions, migration version, key row counts
- `rovo-<ts>.sha256` — checksum manifest of the set
- `rovo_backup.prom` — textfile metrics, incl.
  `rovo_backup_last_success_timestamp_seconds` (feeds alert **A16**)

Retention: sets older than `BACKUP_RETENTION_DAYS` (default 7) are pruned, but
the newest `BACKUP_KEEP_MIN` (default 3) are always kept.

In the cloud this same script runs as a scheduled ECS task (EventBridge
03:15 IST) writing to the Object-Locked bucket in the `rovo-audit` account, with
**age** client-side encryption on top of SSE-KMS.

## Restore a backup

The restore verifies the SHA-256 manifest **before** touching anything, then
runs in a single transaction (all-or-nothing). Owners/grants are not restored —
migrations/IaC re-grant the app roles.

```bash
make restore DUMP=backups/rovo-<ts>.dump.gz            # into dev
make restore DUMP=backups/rovo-<ts>.dump.gz CLEAN=1    # wipe public schema first
# or directly:
scripts/restore.sh backups/rovo-<ts>.dump.gz --stack prod --clean --yes
```

`--clean` is required if the target schema is not empty. `--no-verify` skips the
checksum check (not recommended).

### After restoring real (non-synthetic) data

1. **Replay the erasure ledger** (`erasure_requests`) before serving traffic, so
   users who requested deletion are not resurrected (DPDP; DR §9).
2. **Run payment reconciliation** against the payment provider for the gap
   window — the PA is the source of truth after a restore (DR §6).
3. Rotate secrets if the restore was part of a compromise response (DR-5).

## Verify (restore drill D0)

```bash
make verify-restore
```

Restores the latest backup into a **throwaway** database and checks invariants
(row counts vs the `.meta`, `postgis_full_version()`, ledger-balance invariants,
latest order timestamp), then destroys it. The script includes negative tests
(tampered checksum, row-count mismatch, unbalanced ledger) that must fail as
expected. This is the weekly automated drill; failure raises **A16**.

## Alert A16 — no successful backup in 26 h

1. Check the last backup set in the backup dir / bucket and
   `rovo_backup_last_success_timestamp_seconds`.
2. Run `make backup` manually; watch for errors (disk space, DB reachability,
   `pg_dump` version mismatch).
3. If the scheduled job failed in the cloud, check the ECS task logs for the
   backup task and the bucket's PutObject permissions (the prod role may only
   `PutObject`; Object Lock blocks overwrite/delete by design).
4. Confirm the metric recovers after a successful run.
