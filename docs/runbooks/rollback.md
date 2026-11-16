# Runbook: Roll back a release

Because migrations are **expand/contract** (a release only adds; destructive
changes ship a release later), the previous release still works against the
current database. So rolling back is **redeploying the previous image** — you do
**not** roll the database back. Target: ≤ 10 minutes (DR drill D4).

## Decide

Roll back when a deploy causes: a sustained 5xx spike (A12), an SLO burn (A11),
core-route latency past SLO, broken checkout/dispatch, or any P1 clearly tied to
the release. If instead the **data** is corrupt (bad migration wrote bad rows),
this is a data incident — use PITR, see DR-2 in
`docs/planning/23-backup-disaster-recovery.md` §8, not this runbook.

## Production-like stack

```bash
# Redeploy the previous good tag. migrate up on an older image is a no-op
# (its migrations are already applied), so the DB is untouched.
git checkout v<PREVIOUS>
make prod-up
make smoke
```

If the new release included a migration and you must remove its effect locally,
only use the down migration in a throwaway/dev DB — never run `migrate down` on
data you care about. Prefer PITR for real data.

## Cloud (AWS ECS) — summary

1. Redeploy the **previous task-definition revision** (pinned to the previous
   image digest):
   ```bash
   aws ecs update-service --cluster rovo --service rovo-api \
     --task-definition rovo-api:<PREV_REVISION> --force-new-deployment
   # repeat for rovo-worker
   ```
   Or re-run the deploy pipeline against the previous release tag.
2. The ECS **circuit breaker** also auto-rolls-back a failed rolling deploy on
   its own.
3. **Frontends:** re-sync the previous release's SPA bundles and invalidate
   `/index.html`, `/sw.js`, `/manifest.webmanifest`. Hashed assets are immutable,
   so old clients keep working.
4. DB down-migrations are used **only** for a failed expand step in staging,
   never on prod data.

## Verify

- `make smoke` passes; `up` for api/worker is 1.
- The firing alert that triggered the rollback clears.
- Grafana shows 5xx and latency back to baseline.

## After

- Open an incident note: what shipped, what broke, how it was caught, the fix.
- Keep the bad image/tag for forensics; do not delete it.
- Add a regression test before re-attempting the release.
