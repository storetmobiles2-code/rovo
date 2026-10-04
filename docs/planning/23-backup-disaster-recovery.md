# 23 — Backup & Disaster Recovery

| | |
|---|---|
| **Purpose** | Set RPO/RTO targets and define backups (database, object storage, configuration, secrets), immutability, restore drills and DR runbooks for rovo on AWS (`ap-south-1` primary, `ap-south-2` DR). Also covers payment reconciliation after a restore and DPDP considerations. |
| **Owner** | DevOps Architect |
| **Status** | Draft v1 (aligned with baseline §4a, 2026-10-04) |
| **Depends on** | `22-deployment-architecture.md` (accounts, RDS, S3, KMS); `14-payment-architecture.md` (PA as source of truth, ledger); `10-database-schema.md` (tables, retention); `12-auth-rbac.md`/`19-security-threat-model.md` (ransomware, insider); `24-observability-strategy.md` (backup alerts); `25` (verified prices `[Cn]`) |
| **Feeds** | `29-production-readiness-checklist.md`, `30-risks-assumptions-decisions.md` |

---

## 1. Targets

| Scenario | RPO (max data loss) | RTO (time to restore service) | Mechanism |
|---|---|---|---|
| Single task/instance failure | 0 | < 2 min | ECS replaces tasks; 2 api + 2 worker across 2 AZs |
| **AZ failure** (DB primary AZ) | **0** (synchronous standby) | **1–3 min** (Multi-AZ failover [ASSUMPTION: AWS typical]) | RDS Multi-AZ DB instance |
| **Logical corruption / accidental delete / bad migration** | **≤ 5 min** before the incident (PITR granularity [ASSUMPTION: RDS latest restorable time ≈ 5 min]) | **≤ 2 h** | RDS PITR to a new instance + selective repair or cut-over |
| **Region outage (Mumbai)** | **≤ 30 min** [ASSUMPTION: cross-Region log replication lag; to be measured in drills] | **≤ 4 h** (pilot) | Cross-Region automated backups to Hyderabad [C6] + IaC rebuild |
| **AWS account compromise / suspension** | **≤ 24 h** (nightly logical dump) | **≤ 8 h** | Object-Locked dumps in the separate `rovo-audit` account; rebuild in a clean account, or the GCP alternative (`25` §16) |
| Observability vendor outage | n/a (telemetry loss tolerated) | n/a | CloudWatch last-resort alarms (`24`) |

These targets are **pilot targets**. They are reviewed when orders exceed 5,000/day, and possibly tightened with an RDS cross-Region **read replica** in Hyderabad (RPO seconds, RTO < 1 h) at extra cost.

---

## 2. What must be protected

| Asset | Criticality | Primary copy | Backup copies |
|---|---|---|---|
| PostgreSQL (orders, ledger, users, zones, audit log) | **Critical** | RDS `ap-south-1` Multi-AZ | PITR 14 d; cross-Region automated backups 7 d (`ap-south-2`); manual snapshots; nightly logical dumps (Object Lock, separate account, both India regions) |
| KYC documents (S3 `kyc-private`) | **Critical + sensitive** [LEGAL] | S3 `ap-south-1`, versioned, SSE-KMS | S3 Cross-Region Replication → `ap-south-2` bucket in `rovo-audit` (versioned, Object Lock governance) |
| Public media (menu/restaurant images) | Medium (re-uploadable) | S3 `media-public`, versioned | CRR → `ap-south-2` (cheap; avoids re-onboarding 50 restaurants) |
| SPA builds | Low | S3 web buckets | Rebuildable from git tag |
| Container images | Medium | ECR `rovo-shared` (`ap-south-1`) | **ECR cross-Region replication → `ap-south-2`**; GHCR public mirror |
| IaC + app code | Critical | GitHub | Weekly mirror to a second git host [OPEN: Codeberg/GitLab]; every developer clone |
| OpenTofu state | High | S3 (versioned) in `rovo-shared` | CRR → `ap-south-2` |
| Secrets (PA keys, webhook secrets, JWT keys, DLT/SMS keys) | Critical | Secrets Manager (`ap-south-1`) | **Secrets Manager multi-Region replica → `ap-south-2`** (replica billed as a secret [ASSUMPTION]); break-glass offline copy of *provider-issued* secrets in a shared password-manager vault owned by 2 founders |
| KMS keys | Critical | KMS | **Multi-Region keys** for `backups` and `kyc` so the DR region can decrypt replicas [ASSUMPTION: MRK cost = 1 key each region] |
| Dashboards/alerts | Low | Grafana Cloud | As code in git (`deploy/observability/`) |

---

## 3. PostgreSQL backup design

### 3.1 Layers

| Layer | What | Retention | Purpose |
|---|---|---|---|
| L1 | **RDS automated backups** (daily snapshot + transaction logs) → **PITR** | **14 days** | Any-second restore within window |
| L2 | **Cross-Region automated backup replication** Mumbai → Hyderabad (supported for Multi-AZ DB instances [C6]) | **7 days** | Region loss; PITR in DR region |
| L3 | **Manual snapshots**: before each release with migrations (`pre-vX.Y.Z`) and monthly (`monthly-YYYY-MM`), copied to `ap-south-2` | pre-release 30 d; monthly **6 months** | Long-horizon restore, audits |
| L4 | **Logical dump**: nightly `pg_dump -Fc --no-owner` (plus `pg_dumpall --globals-only` without passwords) from a scheduled ECS task (EventBridge Scheduler 03:15 IST) → S3 `rovo-audit/backup-dumps-aps1` (**Object Lock, compliance mode**) with **S3 CRR → `ap-south-2`** | **7 daily / 4 weekly / 6 monthly** (lifecycle by prefix) | Provider-independent (restorable on GCP/any PG 17 + PostGIS), protects against account-level loss and snapshot-level issues; enables single-table restore |

- **Encryption:** RDS/snapshots use CMK `rds` (an MRK, or re-encrypted on copy with the DR-region key). Dumps are encrypted client-side with **age** (recipient key; private key offline with 2 custodians) **and** server-side SSE-KMS (`backups` MRK). Two independent keys mean S3/KMS compromise alone cannot read dumps.
- **Integrity:** each dump has a SHA-256 manifest. The dump job runs `pg_restore --list` as a sanity check and records size, row counts of key tables and the duration as metrics (`24`: alert "backup failed / not run in 26 h / size −30% vs 7-day median").
- **Separate account:** the dump bucket lives in `rovo-audit`. The prod backup task role can **only `PutObject`** (no delete/overwrite; Object Lock blocks it anyway). Nobody in `rovo-prod` can shorten retention.

### 3.2 Cost (excl. GST)

| Item | Estimate |
|---|---|
| L1 within free allowance (backup storage up to 100% of provisioned storage is free [ASSUMPTION: RDS policy, not fetched]) | ≈ $0 |
| L2 cross-Region backups ≈ 15 GB × $0.095 [C2] + transfer | ≈ $2 |
| L3 monthly snapshots (incremental) ≈ 10 GB × $0.095 | ≈ $1 |
| L4 dumps ≈ (7 + 4 + 6) × 0.5 GB × 2 regions × $0.025 [C13] | < $1 |
| **Total** | **≈ $4–5/month (≈ ₹450)** at pilot |

---

## 4. Object storage protection

- **Versioning** on all data buckets (`22` §6.3). Lifecycle: noncurrent versions expire after 30 d (KYC), 90 d (media).
- **S3 CRR** for `kyc-private` and `media-public` → `ap-south-2` buckets in `rovo-audit`, with replica-modification sync off and **delete-marker replication OFF**, so deletes in prod do not propagate.
- **KYC deletion workflow:** KYC retention/erasure (DPDP [LEGAL]) is executed by an app job that deletes current + noncurrent versions in prod. The DR replica gets a matching **erasure manifest** that a scheduled job in `rovo-audit` applies, so erasure is honoured everywhere within 30 days [LEGAL: confirm acceptable window].
- **Bucket policies** deny `s3:DeleteObjectVersion` and `s3:PutLifecycleConfiguration` to every principal except the `BackupAdmin` permission set (MFA required).

---

## 5. Ransomware / malicious-insider considerations

| Threat | Control |
|---|---|
| Attacker with prod admin deletes RDS + snapshots | RDS deletion protection; SCP denies `rds:DeleteDBSnapshot` outside `BackupAdmin`; **L4 dumps in a separate account with Object Lock compliance mode** cannot be deleted before expiry by anyone, root included |
| Attacker encrypts/overwrites S3 objects | Versioning + CRR to the audit account (no delete-marker replication) + Object Lock on backup buckets |
| Attacker deletes KMS keys | KMS deletion has a mandatory waiting period (set **30 days**); alarm on `ScheduleKeyDeletion` (`24`) |
| Compromised CI | OIDC roles cannot touch backups or KMS key policies; infra apply needs 2 approvals (`21` §7.3) |
| Account suspended by provider (billing/abuse) | Billing via card + backup payment method; Budgets alerts; `rovo-audit` owned by the same org but with an independent root email + MFA; worst case: rebuild from L4 dumps (portable) on GCP |

---

## 6. Payment provider as source of truth after restore

Any restore that loses data (PITR point < incident time, region failover with replication lag, dump restore) creates a **reconciliation window** `[restore_point − 15 min, cutover_time]`.

```mermaid
sequenceDiagram
  participant Ops
  participant App as rovo (restored)
  participant PA as Payment aggregator API
  Ops->>App: enable maintenance mode (orders closed, SSE banner)
  Ops->>App: run `rovo recon payments --from T0 --to T1`
  App->>PA: list payments / refunds / settlements in window
  PA-->>App: authoritative records
  App->>App: upsert payments idempotently (provider_payment_id unique), re-derive order status, post missing ledger entries (idempotency keys)
  App->>App: list orders in PENDING_PAYMENT / PLACED with captured payments → flag for ops
  Ops->>App: review recon report; contact affected customers/restaurants/riders
  Ops->>App: replay PA webhooks (PA dashboard "resend") for the window
  Ops->>App: disable maintenance mode
```

- **COD and rider cash:** cash collections recorded in the window may be lost from the DB. The rider PWA keeps an **outbox of submitted COD confirmations** (IndexedDB) and re-submits on reconnect/sync (`18`). Ops also runs a **cash-in-hand attestation** with riders for the window.
- **Restaurant payouts:** payouts are manual in V1 (`14`). The payout register (bank/UPI references) is re-entered from bank statements for the window.
- **Comms:** template messages for customers whose orders are affected (refund/cancel), plus a restaurant partner notice.
- Requirement for `14`/`10`: every payment/ledger write is **idempotent on the provider's IDs**. That makes reconciliation a replay, not a merge.

---

## 7. Restore drills

| Drill | Frequency | Procedure (scripted in `deploy/dr/`) | Success criteria |
|---|---|---|---|
| **D1 PITR clone** | **Monthly** (automated, `21` §9) | Restore prod PITR (T−1 h) to a new instance in the **staging account** from a shared snapshot copy → run verification SQL (row counts vs prod metrics, `SELECT postgis_full_version()`, ledger balance invariants, latest order timestamp) → measure duration → **destroy** | Restore ≤ 60 min; invariants pass; report filed |
| **D2 Logical dump restore** | Monthly (alternating with D1) | Fetch latest L4 dump (age decrypt with drill key custodian) → restore into a disposable RDS (or local PG 17 + PostGIS container) → same verification | Restore ≤ 90 min |
| **D3 Region failover game day** | **Quarterly**, in **staging** (staging also has cross-Region backup enabled for the drill) | Execute DR-3 end-to-end in `ap-south-2`, including DNS cut-over of `*.staging` hostnames | RTO ≤ 4 h measured; runbook gaps fixed |
| **D4 App rollback** | Monthly in staging | `ecs-rollback.sh` | ≤ 10 min |
| **D5 Payment reconciliation** | Quarterly in staging with PA sandbox | Delete last 30 min of payments in a staging clone → run §6 → compare | 100% of sandbox payments reconciled |
| **D6 KYC object restore** | Quarterly | Restore a deleted object version + read from DR replica | Object retrievable; access logged |

Drill output (date, operator, timings, issues) goes into `docs/ops/drills/YYYY-MM.md` (Phase 2) and as a Grafana annotation. If a drill misses its target, the gap is fixed before the next release.

**Privacy rule for drills:** prod data restored for drills stays in the staging account **only for the drill's duration** (auto-destroy ≤ 6 h). Access is limited to the drill operator. It is never used for testing features [LEGAL: DPDP purpose limitation].

---

## 8. DR runbooks (summaries; full step-by-step scripts in Phase 2)

### DR-1 AZ failure
Automatic: RDS Multi-AZ failover, ECS reschedules tasks in the healthy AZ, ALB stops routing to the unhealthy AZ.
Manual: confirm in dashboards; if ECS capacity is short in the remaining AZ, temporarily raise desired count. No data action needed.

### DR-2 Logical corruption / accidental delete / bad migration
1. Declare an incident (`24` on-call). Enable **maintenance mode** if writes would compound the damage.
2. Identify T_bad from audit log / CloudTrail / app logs.
3. `aws rds restore-db-instance-to-point-in-time --restore-time <T_bad − 1 min>` → `rovo-prod-pitr-<ts>` (same subnet group/SG, Single-AZ).
4. Decide:
   - **(a) Selective repair:** copy the affected rows/tables from the PITR instance into prod (`postgres_fdw` or dump/restore of specific tables) inside a transaction, with a reviewed script.
   - **(b) Full cut-over:** point the `DATABASE_URL` secret at the restored instance (after enabling Multi-AZ on it), restart services, then run §6 reconciliation for the gap.
5. Keep the old instance for 7 days (forensics), then delete with a final snapshot.

### DR-3 Region outage (Mumbai → Hyderabad)
Pre-provisioned in `ap-south-2` (cheap, idle): VPC + subnets + SGs (IaC, $0), ECR replica, Secrets replicas, KMS MRKs, S3 CRR buckets, ACM cert for ALB.
1. Decide failover (Mumbai down > 30 min with no AWS ETA, or AWS Health confirms a regional event). Decision by: on-call + 1 founder.
2. `aws rds restore-db-instance-to-point-in-time --source-db-instance-automated-backups-arn <replicated ARN> --use-latest-restorable-time` in `ap-south-2` → enable Multi-AZ later.
3. `tofu apply -var region=ap-south-2 -var dr_mode=true` in `infra/envs/prod-dr` (ECS cluster, services from the ECR replica digest, ALB, WAF).
4. Point app config at DR S3 replica buckets (KYC, media) via `ROVO_S3_*` vars, or flip CloudFront `cdn.` origin to the DR media bucket.
5. Route 53: update `api.` alias to the DR ALB (TTL 60 s pre-set). SPAs unchanged (CloudFront is global; S3 web origins rebuilt in DR if Mumbai S3 is down → CloudFront origin group failover [OPEN: configure origin groups at setup]).
6. Run §6 reconciliation for the replication-lag window. Announce to partners.
7. **Fail-back** later: logical dump/restore or RDS cross-Region backup in the other direction (Hyderabad → Mumbai supported [C6]), in a planned window.

### DR-4 AWS account compromise or suspension
1. Contain: Identity Center session revocation, root credential rotation, SCP deny-all on `rovo-prod` (from `rovo-mgmt`).
2. If `rovo-prod` is unusable: create a new prod account (`rovo-prod-2`) from Organizations → `tofu apply` → restore the **L4 dump** from `rovo-audit` (Object Lock guarantees integrity) → restore KYC from the CRR replica → DNS cut-over.
3. If the whole AWS organisation is unavailable: stand up on the **GCP alternative** (Cloud Run + Cloud SQL, `25` §16) from the same images (GHCR mirror) and the L4 dump. A copy of the latest weekly dump is also exported to a non-AWS store for this case [OPEN: GCS bucket in asia-south1 or R2; cost ≈ $0; residency [LEGAL]].

### DR-5 Ransomware / mass deletion
Treat as DR-4 + DR-2: assume prod credentials are compromised, restore from immutable L4 / CRR versions, rotate all secrets (PA keys via the PA dashboard, JWT keys → force re-login), and notify per DPDP breach rules [LEGAL].

### DR-6 Third-party outages (non-AWS)
PA down: COD-only mode flag (`14`). SMS/OTP provider down: fallback provider or WhatsApp OTP [OPEN, `15`]. Grafana Cloud down: CloudWatch last-resort alarms (`24`).

---

## 9. Data-protection (DPDP) notes [LEGAL]

- Backups contain personal data. Retention of backups (max 6 months for monthly dumps) must align with the retention schedule in `10`/`12` and the privacy notice.
- **Erasure requests vs immutable backups:** the platform keeps an **erasure ledger** (user IDs + timestamp, no PII). Any restore **replays the erasure ledger** before the restored DB serves traffic. Object-Locked dumps expire naturally within ≤ 6 months. Confirm with counsel that this is acceptable.
- All backup copies stay in **India regions** (`ap-south-1`, `ap-south-2`). The optional non-AWS emergency copy (DR-4 step 3) must also be in India, or be approved by counsel.
- Access to backups is logged (CloudTrail data events on dump buckets) and reviewed quarterly.

---

## 10. Dev/preview & local

- **Preview (Oracle free):** synthetic data only, so backups are optional (nightly `pg_dump` to R2 for convenience; rebuild from seed otherwise).
- **Local:** `make db-dump` / `make db-restore` helpers. Developers **never** pull prod dumps.
