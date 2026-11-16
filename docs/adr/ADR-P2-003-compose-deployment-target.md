# ADR-P2-003: Production-mode Docker Compose is the deployment target for this phase

| | |
|---|---|
| **Status** | Accepted (Phase 2) |
| **Date** | 2026-11-16 |
| **Owner** | Architect / Database Engineer, with DevOps/SRE |
| **Relates to** | Charter §6, baseline §4a, R24 |

## Context

The approved production target is AWS. Provisioning it needs an account, a payment method and
explicit approval at M6. R24 rules out free tiers that require a card during development. The
team still needs an environment that behaves like production to prove the golden flow end to end.

## Decision

- The deployment target for Phase 2 is a production-mode Docker Compose stack
  (`deploy/compose`): PostGIS, MinIO, Mailpit, the `rovo` api and worker, and Caddy serving the
  four web apps and routing `/api/*` same-origin. It may be exposed through a card-free tunnel
  for demos.
- The AWS infrastructure code is written and validated (`tofu validate`) but not applied.
- The backend is one binary with subcommands, so the same image runs every role:

  | Command | Role |
  |---|---|
  | `rovo migrate up` | one-shot job before a release (River schema, then goose migrations) |
  | `rovo api` | public HTTP listener plus the internal health listener |
  | `rovo worker` | River workers and periodic jobs |
  | `rovo all` | api and worker in one process, for local development |
  | `rovo seed` | demo data; refused when `ROVO_ENV=prod` |

- Configuration is environment variables only (`ROVO_*`, `platform/config`). Outside dev/test the
  process refuses to start without its signing and encryption keys.

## Consequences

- Nothing in the application may depend on a cloud-specific service. Object storage is behind
  `platform/blob.Store` (S3-compatible, MinIO locally), queues and realtime fan-out are Postgres
  (River, LISTEN/NOTIFY), rate limits are Postgres rows.
- Single-node Compose has no high availability and no managed backups. Backup and restore are
  scripted by DevOps and must be rehearsed before any real traffic.
- Moving to AWS at M6 is a deployment change (RDS, S3, a container service), not a code change.

## Revisit when

M6 approval for AWS is given.
