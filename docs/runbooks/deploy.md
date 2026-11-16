# Runbook: Deploy a release

How a release reaches the running stack. Based on `docs/planning/22-deployment-architecture.md`
§8 and `docs/planning/21-cicd-strategy.md`. Migrations are **expand/contract**,
so a new release is always compatible with the previous one and rollback does
not require a DB rollback.

## Preconditions

- `make check` green on the commit; CI green on `main`.
- The image is built and, for cloud, signed (cosign keyless; verification is
  **mandatory** before a production deploy).
- A pre-release database snapshot exists (see [backup-restore.md](backup-restore.md));
  cloud tags it `pre-vX.Y.Z`.
- You have the release tag / image digest to deploy.

## Production-like stack (ADR-P2-003)

```bash
# From the repo root, with .env present (make dev-keys if not).
git fetch --tags && git checkout vX.Y.Z

# Build + start; migrate runs first and must complete before api/worker start.
make prod-up                 # TLS=off (default) | internal | acme

# Verify.
make smoke                   # edge contract: health, security headers, CORS, webhooks host
make prod-logs               # watch for clean startup; migrate exits 0
```

`compose.prod.yaml` enforces order: `migrate` runs `migrate up` and must exit
successfully before `api` and `worker` start (`depends_on: service_completed_successfully`),
and `api` waits for `minio-init` too. The edge (`caddy`) waits for `api` to be
healthy.

To seed demo data (non-production only):

```bash
make prod-seed
```

## Cloud (AWS ECS/Fargate) — summary

1. CI builds the multi-arch image, pushes to ECR + the GHCR mirror as
   `sha-<git sha>`, signs it, records the **digest**.
2. **Staging (on merge to `main`):** run the `migrate` task (expand-only) →
   `aws ecs update-service` for `api` then `worker`, task definition pinned to
   the digest → wait for steady state → smoke + synthetic order test.
3. **Production (on approved release tag):** the **same digest** through the same
   steps, behind the `production` GitHub Environment with required reviewers.
   Rolling update with the ECS circuit breaker (auto-rollback on failure). SSE
   clients reconnect transparently.
4. **Frontends:** `aws s3 sync` hashed assets (immutable) first, then
   `index.html` + `sw.js` (no-cache) last, then a CloudFront invalidation of
   `/index.html`, `/sw.js`, `/manifest.webmanifest` only.

## Post-deploy checks

- `make smoke` passes (or the cloud synthetic order test).
- Grafana: 5xx ratio flat, p95 latency within SLO, `up` for `rovo-api` and
  `rovo-worker` is 1, no new firing alerts. See [incident.md](incident.md).
- `migrate` ran and reported the expected migration version.

## If it goes wrong

Go to [rollback.md](rollback.md). For data corruption from a bad migration, use
PITR (see [backup-restore.md](backup-restore.md) and DR-2 in
`docs/planning/23-backup-disaster-recovery.md` §8).
