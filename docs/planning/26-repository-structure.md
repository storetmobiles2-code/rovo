# 26 — Repository Structure

| | |
|---|---|
| **Purpose** | Defines the single-repository (monorepo) layout for rovo: the Go backend module, the OpenAPI contract, the pnpm web workspace, deployment (local Compose and cloud IaC), docs, tooling, naming conventions, code ownership, branching and versioning, task-runner targets, local development setup, licensing, and the community files to create in Phase 2. |
| **Owner** | Solution Architect |
| **Status** | Draft v1.1 — reconciled with review (31) and rulings R1–R48, 2026-10-04 (Phase 1 — planning only; nothing in this tree is created until Phase 2) |
| **Depends on** | `00-planning-baseline.md` (§4a environments), `08-system-architecture.md` (§4 module boundaries), `09-architecture-decision-records.md` (ADR-003, 005, 006, 010, 016, 025, 026) |
| **Must stay consistent with** | `17-frontend-architecture.md` §2 (web workspace packages), `20-testing-strategy.md`, `21-cicd-strategy.md`, `22-deployment-architecture.md` (§10 local Compose, §12 IaC), `27-implementation-backlog.md` |

Tags: `[ASSUMPTION]`, `[OPEN]`, `[LEGAL]`.

**Changes in v1.1 (2026-10-04)**
- Confirmed **R14** (four apps under `web/`) and **R23** (IaC under `deploy/terraform/`); alignment notes closed (§1).
- Custom CI tools `tools/tableowner` and `tools/eventschema` cut (**C9**): go-arch-lint/depguard + `table_ownership.yaml` review checklist; JSON fixtures per event (§1.1, §1.5, §5, §6).
- `platform/events` publishes with `InsertManyTx`, no fan-out worker (**R22/R42**) (§1.1).
- IaC: `closed-pilot` / `public-launch` profiles (**R32**), `/api/*` behaviour on four app hosts (**R27**), 180-day log archive (**R36**), NAT toggle (**R28**), 4 AWS accounts (**C18**) (§1.4).
- Local Docker only, no preview compose/env (**R24**); `cloudflared` quick tunnel default; WhatsApp adapters V1.1 (**C2**); no build-time prerender (**R33**); fake bot-challenge adapter and local map tiles (§1, §6, §7, §8).
- JSON field naming aligned with doc 11 (`camelCase`); seed translations as `*_i18n` (**R17**); fee slabs to 10 km (**R18**) (§2, §8).

---

## 1. Top-level layout

```
rovo/
├── backend/                 # Go module: github.com/<org>/rovo/backend (API, worker, migrate in one binary)
├── openapi/                 # OpenAPI 3.1 contract: the single source of truth for HTTP APIs
├── web/                     # pnpm workspace: 4 PWAs + shared packages
├── deploy/
│   ├── compose/             # local dev Docker Compose (the only dev/demo environment, R24)
│   ├── terraform/           # OpenTofu/Terraform IaC for staging + prod (managed cloud)
│   ├── docker/              # Dockerfiles (backend, postgres-postgis dev image), .dockerignore
│   └── observability/       # Grafana dashboards, alert rules, OTel collector configs (shared by local + cloud)
├── docs/
│   ├── planning/            # Phase 1 planning documents (this folder)
│   ├── adr/                 # one file per ADR (split from doc 09 in Phase 2), NNNN-title.md
│   ├── runbooks/            # incident, deploy, rollback, restore, key rotation, PA outage, OTP outage
│   ├── api/                 # rendered API reference (generated, not committed, or published via Pages)
│   └── contributing/        # dev setup, architecture overview, module guide, style guides
├── tools/                   # repo-internal Go tools (separate go module): seedgen (no tableowner/eventschema in V1, C9)
├── scripts/                 # thin shell scripts called by Make targets (bootstrap, tunnel, release notes)
├── .github/
│   ├── workflows/           # CI/CD (doc 21)
│   ├── ISSUE_TEMPLATE/      # bug, feature, security-redirect, docs
│   ├── PULL_REQUEST_TEMPLATE.md
│   ├── CODEOWNERS
│   └── dependabot.yml       # or renovate.json at root [OPEN: DevOps]
├── Makefile                 # single entry point for every developer/CI task (§6)
├── .editorconfig  .gitattributes  .gitignore  .tool-versions (or mise.toml)
├── README.md  LICENSE (Apache-2.0)  NOTICE  CONTRIBUTING.md  CODE_OF_CONDUCT.md  SECURITY.md  GOVERNANCE.md  CHANGELOG.md
```

**Why these top-level splits:**
- Each toolchain (Go, Node, OpenTofu) has its own root. That keeps lockfiles, linters and CI path filters simple (ADR-010).
- `openapi/` sits at the top level because both `backend/` and `web/` generate from it. It is neither side's property.
- `deploy/` holds everything needed to *run* rovo anywhere: local Compose and the managed cloud via IaC (baseline §4a).

> **Alignment notes (decided):**
> - The web workspace lives under **`web/`** (R14). Doc 17 §2 must use the `web/` prefix (31 §14 minor drift).
> - The IaC root is **`deploy/terraform/`** (R23), not `infra/`. Docs 21 and 22 must use these paths (register row 45), and doc 21's path filters must match `openapi/` and `backend/migrations/` (31 §14 minor drift).

### 1.1 `backend/` (Go module)

```
backend/
├── go.mod  go.sum                         # Go 1.27.x; toolchain directive pinned
├── cmd/
│   └── rovo/
│       └── main.go                        # subcommands: api | worker | all | migrate | seed | admin (break-glass CLI) | version
├── internal/
│   ├── app/                               # composition root: config → adapters → modules → servers (wire by hand, no DI framework)
│   ├── platform/                          # cross-cutting, imports NO module
│   │   ├── config/                        # env parsing (caarlos0/env or stdlib), validation, redaction for logs
│   │   ├── db/                            # pgx pool, tx helpers (WithTx), LISTEN conn, statement timeouts
│   │   ├── httpx/                         # middleware chain, problem+json, request id, idempotency, ETag, rate-limit headers, SSE writer
│   │   ├── auth/                          # authn middleware (JWT verify, session-state check), principal in ctx, RBAC helpers
│   │   ├── otel/                          # tracer/meter/logger setup, slog handler with trace ids
│   │   ├── events/                        # event envelope, Publish(tx, evt) → InsertManyTx one job per subscriber, static subscription table (R42; no fan-out worker, no event table)
│   │   ├── queue/                         # job interface over River (Enqueue, EnqueueTx, Schedule); River setup + middleware
│   │   ├── realtime/                      # SSE hub, topic auth, pubsub.Bus interface (postgres NOTIFY impl; redis impl)
│   │   ├── cache/  ratelimit/             # interfaces + memory/postgres/redis implementations (ADR-007)
│   │   ├── blob/                          # S3-compatible object storage interface + presign (ADR-022/025)
│   │   ├── money/  idgen/  clock/  i18n/  # paise type & rounding; UUIDv7 + order codes; injectable clock; server catalogs
│   │   └── health/                        # /healthz, /readyz checks
│   ├── adapters/                          # vendor-specific implementations ONLY here (ADR-025 depguard rule)
│   │   ├── payments/{razorpay,cashfree,fake}/
│   │   ├── notify/{webpush,msg91,gupshup,smtp,resend,fake}/   # metacloud (WhatsApp) added in V1.1 (C2)
│   │   ├── botchallenge/{turnstile,fake}/ # fake adapter for local/CI/E2E (doc 12)
│   │   └── blob/{s3,fake}/                # GCS via S3 interop uses s3; azure only if chosen
│   ├── modules/                           # one directory per bounded module (doc 08 §3.2)
│   │   ├── identity/
│   │   │   ├── identity.go                # public: Service interface, DTOs, New(...)
│   │   │   ├── events.go                  # public: exported event structs
│   │   │   ├── internal/
│   │   │   │   ├── domain/                # entities, invariants, pure logic
│   │   │   │   ├── app/                   # use cases implementing Service
│   │   │   │   ├── store/                 # sqlc output (generated) + repository wrapper
│   │   │   │   ├── httpapi/               # adapters: generated strict-server interface → app
│   │   │   │   └── jobs/                  # River workers for consumed events/timers
│   │   │   └── queries/                   # *.sql for sqlc (owned tables only)
│   │   ├── users/  geo/  catalog/  pricing/  promotions/  ordering/  payments/
│   │   ├── dispatch/  ledger/  ratings/  notifications/  support/  adminaudit/  reporting/
│   ├── api/                               # generated server code from openapi (oapi-codegen strict, std-http) — DO NOT EDIT
│   └── testutil/                          # testcontainers helpers, fixtures, golden files, fake clock
├── migrations/                            # goose SQL migrations, timestamped: 20261005120000_create_cities.sql
│   └── seeds/                             # dev/demo seed data (Mahabubnagar), never run in prod
├── table_ownership.yaml                   # table → module manifest (PR review checklist + CODEOWNERS, doc 08 §4.3, C9)
├── sqlc.yaml                              # one sqlc package per module (queries → internal/store)
├── oapi-codegen.yaml                      # codegen config (strict server, std-http, models)
├── .golangci.yml                          # incl. depguard rules (module DAG, vendor SDK ban), forbidigo (no float in money pkgs)
├── .go-arch-lint.yml                      # module dependency graph (doc 08 §3.1)
└── README.md
```

Notes:
- **Nested `internal/`** per module makes the Go compiler forbid cross-module imports of internals (doc 08 §4.2). Other modules see only `identity.go` / `events.go`.
- **Why `migrations/` is global, not per module:** goose needs one ordered history. Ownership of each table is declared in `table_ownership.yaml`, and the migration's file name carries the module (`…_ordering_create_orders.sql`). If a module is extracted later, its migrations move with it.
- River's own schema migrations are applied by `rovo migrate` through `rivermigrate` (pinned River version). They are not copied into `migrations/`.
- Restaurant share-preview pages (`/r/{slug}`, OG tags + redirect into the SPA, R33, P1) are a small handler in `modules/catalog/internal/httpapi`.
- Generated code (`internal/api`, `internal/modules/*/internal/store`) **is committed**. Reviews show the diff, `go build` needs no codegen step, and CI verifies that it is up to date (`make generate && git diff --exit-code`).

### 1.2 `openapi/`

```
openapi/
├── openapi.yaml                 # root: info, servers, security schemes, tags, $refs to paths/components
├── paths/                       # one file per resource group: auth.yaml, orders.yaml, restaurant-orders.yaml, rider.yaml, admin-*.yaml, webhooks.yaml
├── components/
│   ├── schemas/                 # Money, Problem, Order, Quote, ... (one file per schema family)
│   ├── parameters/  responses/  headers/   # Idempotency-Key, If-Match, cursor/limit, problem responses
├── examples/                    # request/response examples (also feed MSW mocks in web/packages/testing)
├── redocly.yaml                 # lint rules + bundling config
├── STYLE.md                     # API style guide: naming, errors, pagination, the 3.0-compatible subset (ADR-003)
└── dist/openapi.bundled.yaml    # generated by `make api-bundle` (gitignored; CI publishes it as an artifact)
```

### 1.3 `web/` (pnpm workspace; structure from doc 17 §2)

```
web/
├── package.json                 # private root; scripts delegate to turbo/pnpm -r [OPEN: turbo]
├── pnpm-workspace.yaml          # apps/*, packages/*; pnpm catalogs pin shared versions
├── .npmrc  tsconfig.base.json  eslint.config.js (re-export from packages/config)
├── apps/
│   ├── customer/                # app.<domain> / apex — customer PWA (static legal pages + OG meta in index.html; no build-time prerender, R33)
│   ├── restaurant/              # restaurant.<domain> — "rovo Partner"
│   ├── rider/                   # rider.<domain> — "rovo Delivery Partner"
│   └── admin/                   # admin.<domain> — admin SPA (no service worker)
└── packages/
    ├── api-client/              # GENERATED types from ../../openapi + openapi-fetch client, query keys, SSE wrapper
    ├── domain/  utils/          # DOM-free, React-Native-safe
    ├── ui/                      # design system (Radix + Tailwind preset)
    ├── i18n/                    # en/te catalogs + formatters
    ├── pwa/                     # Workbox configs, push subscribe, wake lock, audio unlock
    ├── testing/                 # MSW handlers from OpenAPI examples, Playwright fixtures
    └── config/                  # eslint, tsconfig, tailwind preset, vite base, size-limit budgets
```

Each app has the standard Vite layout: `src/{routes,features,components,lib}`, `public/`, `index.html`, `vite.config.ts`, `playwright/` (E2E), `README.md`.

### 1.4 `deploy/`

```
deploy/
├── compose/
│   ├── compose.yaml             # local: postgres(+postgis), minio(+init), mailpit, lgtm, valkey[profile cache], migrate, api, worker
│   ├── compose.override.example.yaml   # hot reload (Air), debugger ports
│   ├── env/
│   │   └── local.env.example    # all ROVO_* vars with safe dev defaults (fakes on)
│   └── initdb/                  # role creation mirroring prod (app, migrator, readonly), extensions (postgis, pg_trgm)
├── docker/
│   ├── backend.Dockerfile       # multi-stage: golang:1.27 builder → distroless/static nonroot; multi-arch
│   ├── postgres-postgis.Dockerfile  # dev/CI multi-arch PG 17 + PostGIS 3.5 image (doc 22 §10, R22)
│   └── web.Dockerfile           # optional: static build verification / preview only (prod serves from object storage)
├── terraform/                   # OpenTofu (Terraform-compatible) — ADR-026
│   ├── modules/
│   │   ├── network/             # VPC, subnets (public/private/db), NAT toggle (off at closed pilot, one NAT before Gate B, R28), VPC endpoints, flow logs (all traffic)
│   │   ├── db/                  # RDS PostgreSQL 17 + PostGIS: instance, params, subnet group, KMS, PITR, cross-region backup copy; multi_az flag per profile (R32)
│   │   ├── cache/               # managed Redis/Valkey — enabled=false by default (ADR-007)
│   │   ├── storage/             # secure bucket module: public-media, private-docs, 4 web-app buckets, backups, log-archive (180-day lifecycle, security subset ≥ 1 year, R36)
│   │   ├── cdn/                 # CloudFront for app./restaurant./rider./admin. (+ media): /api/* behaviour to the ALB on every app host, origin-verify header, SPA rewrite, WAF (admin host: rate + geo-IN rules) (R27, R37)
│   │   ├── compute/             # container cluster, api service (+LB), worker service, migrate/tools one-off tasks, autoscaling
│   │   ├── observability/       # CloudWatch log groups + S3 archive subscriptions (CERT-In 180 days, R36), alarms, notification channels, Grafana Cloud links
│   │   ├── security/            # KMS keys, secrets shells, CI OIDC roles (plan/apply/deploy), IAM boundaries
│   │   ├── registry/            # container registry + lifecycle policies
│   │   ├── dns/                 # zones, records, certificates
│   │   └── budgets/             # cost budgets + anomaly alerts (INR)
│   ├── profiles/                # tfvars sets applied to prod (R32)
│   │   ├── closed-pilot.tfvars  # RDS Single-AZ db.t4g.small, 2 small api + 1 worker, no NAT, no collector gateway
│   │   └── public-launch.tfvars # Multi-AZ (before Gate B or > 100 orders/day), sizes from the doc 20 load test, NAT on
│   ├── envs/
│   │   ├── shared/              # bootstrap: state bucket + lock, registry, org-level bits (4 AWS accounts: mgmt, prod, nonprod, audit/backup; C18)
│   │   ├── staging/             # main.tf, variables.tf, terraform.tfvars, backend.tf (closed-pilot sizes, stoppable)
│   │   └── prod/                # main.tf, variables.tf, terraform.tfvars, backend.tf
│   ├── policies/                # checkov/conftest rules (no public DB, encryption on, tags required)
│   └── README.md                # how to plan/apply, state layout, OIDC, drift detection
└── observability/
    ├── dashboards/              # Grafana JSON: RED per module, orders funnel, dispatch, payments, notifications
    ├── alerts/                  # alert rules (PromQL/LogQL) + runbook links
    └── otel-collector/          # collector config for local (lgtm); cloud exports OTLP to Grafana Cloud directly at pilot (R36)
```

**Module mapping to doc 22's AWS design** (current instantiation): `compute` ⊇ ecs-cluster, ecs-service, ecs-oneoff-task, alb. `cdn` ⊇ cloudfront-spa, cloudfront-media, waf. `db` = rds-postgres. `cache` = elasticache-valkey. `storage` = s3-bucket. `security` ⊇ kms, secrets, github-oidc. `registry` = ecr. Inside a logical module, provider-specific resources live directly in the module (single-cloud). A second cloud would add `modules/<name>/<cloud>/` variants rather than abstracting prematurely (ADR-026).

### 1.5 `tools/` and `scripts/`

```
tools/                       # separate go.mod so tool deps don't leak into backend
├── seedgen/                 # generates the Mahabubnagar demo dataset (zones, localities, restaurants, menus, riders)
└── tools.go                 # pinned versions of sqlc, goose, oapi-codegen, golangci-lint, go-arch-lint, govulncheck
scripts/
├── bootstrap.sh             # checks prerequisites (docker, go, node, pnpm, mise), installs tools
├── tunnel.sh                # exposes the local stack for demos / PA sandbox webhooks via a cloudflared quick tunnel (no account, card-free; R24) — dev only
└── release-notes.sh
```

---

## 2. Naming conventions

| Thing | Convention | Example |
|---|---|---|
| Go packages | short, lower-case, no underscores. Module names singular/nouns. | `ordering`, `dispatch`, `ledger` |
| Go files | `snake_case.go`; tests `_test.go`; integration tests behind a `//go:build integration` tag | `offer_cascade.go` |
| Go interfaces | role names; `Service`/`Reader` as module facades; no `I` prefix | `payments.Provider` |
| DB tables | `snake_case`, plural nouns; FK columns `<entity>_id`; money `_paise`; timestamps `_at` (timestamptz) | `delivery_offers.expires_at` |
| DB enums | `UPPER_SNAKE` text with CHECK constraints (or PG enums, as doc 10 decides), matching API enums | `READY_FOR_PICKUP` |
| Migrations | `YYYYMMDDHHMMSS_<module>_<action>.sql` | `20261012093000_dispatch_create_delivery_offers.sql` |
| sqlc queries | `-- name: VerbNoun :one/:many/:exec` | `GetOrderForUpdate` |
| Events | Go type `PastTense`, type key `<module>.<snake_event>.v<N>` | `ordering.order_accepted.v1` |
| River job kinds | `<module>.<verb_noun>` | `dispatch.expire_offer` |
| HTTP paths | `/api/v1/<plural-resource>/{id}/<action>`; kebab-case for multiword | `/api/v1/rider/offers/{id}/accept` |
| JSON fields | `camelCase` per doc 11 API-D02 (DB columns stay `snake_case`). Enums UPPER_SNAKE. | `totalPaise`, `nameI18n` |
| Error codes | `UPPER_SNAKE`, stable | `ORDER_STATE_CONFLICT` |
| Env vars | `ROVO_<AREA>_<NAME>` | `ROVO_DB_URL`, `ROVO_PA_PROVIDER` |
| Feature flags | `kebab.dot` scope | `dispatch.broadcast-mode` |
| TS packages | `@rovo/<name>` | `@rovo/api-client` |
| React components | `PascalCase.tsx`; hooks `useThing.ts`; route files per TanStack Router conventions | `OrderStatusPill.tsx` |
| i18n keys | `namespace.section.key` | `checkout.payment.cod_unavailable` |
| Notification templates | `<audience>.<event>` | `restaurant.order_new` |
| Terraform | resources `snake_case`; modules by capability; tags `app=rovo, env, owner, cost_center` | `module "db"` |
| Docker images | `<registry>/rovo:<semver>` and `:<git-sha>` (multi-arch manifest) | `rovo:1.4.0` |
| Branches | `<type>/<short-desc>` (or `<type>/<issue>-<desc>`) | `feat/123-offer-cascade` |

---

## 3. Code ownership (CODEOWNERS)

The project starts with a small team, so ownership is by **area**, using GitHub teams (create the teams in Phase 2). It is enforced for review, and required on protected branches for the critical paths.

```
# .github/CODEOWNERS (illustrative)
*                                   @rovo/maintainers
/backend/                           @rovo/backend
/backend/internal/modules/payments/ @rovo/backend @rovo/payments-reviewers
/backend/internal/modules/ledger/   @rovo/backend @rovo/payments-reviewers
/backend/migrations/                @rovo/backend @rovo/db-reviewers
/backend/internal/platform/auth/    @rovo/backend @rovo/security
/openapi/                           @rovo/backend @rovo/frontend      # contract changes need both sides
/web/                               @rovo/frontend
/web/packages/ui/                   @rovo/frontend @rovo/design
/deploy/terraform/                  @rovo/devops
/deploy/                            @rovo/devops
/.github/workflows/                 @rovo/devops @rovo/security
/docs/adr/                          @rovo/architects
SECURITY.md LICENSE NOTICE          @rovo/maintainers
```

Two approvals are required for `payments`, `ledger`, `migrations`, `auth` and `deploy/terraform/envs/prod` (branch protection with "require review from Code Owners").

---

## 4. Branching, commits, versioning, releases

- **Trunk-based development:** `main` is always releasable. Feature branches live ≤ 2–3 days. Unfinished work is merged behind feature flags (`feature_flags` table / config). No long-lived `develop` or release branches. Hotfix: branch from the release tag, cherry-pick to `main`.
- **Pull requests:** squash-merge. The PR title follows **Conventional Commits** (`feat(dispatch): add offer cascade timeout`), enforced by a CI check on the PR title. Allowed types: `feat, fix, perf, refactor, docs, test, build, ci, chore, revert`. Scopes are module or app names. `!` or a `BREAKING CHANGE:` footer marks breaking changes.
- **Branch protection on `main`:** required checks (doc 21), Code Owners review, linear history, signed commits recommended `[OPEN: require?]`, no force-push.
- **Versioning:** **SemVer for the platform release** (`vMAJOR.MINOR.PATCH`), one version for backend and web together, because they ship together against one contract.
  - The API is versioned separately by path (`/api/v1`, ADR-023). Breaking API changes need a new path version, not just a major bump.
  - Pre-1.0 (`v0.x`) until the production launch. `v1.0.0` = Mahabubnagar go-live.
- **Release flow:**
  1. Merges to `main` auto-deploy to **staging** (doc 21).
  2. Release-please (or git-cliff) maintains a release PR and generates `CHANGELOG.md` from conventional commits `[OPEN: DevOps picks]`.
  3. Merging it tags `vX.Y.Z`, builds or promotes the **same image digest** and requests approval for **prod**.
  4. Database migrations follow expand/contract, so the previous release remains compatible for rollback.
- **Artifacts:** multi-arch container image (cloud registry + GHCR for the community), web app bundles (versioned object-storage prefixes), bundled OpenAPI spec, SBOM, and release notes.

---

## 5. Generated code policy

| Generated | Source | Command | Committed? | CI check |
|---|---|---|---|---|
| Go server interfaces/types | `openapi/` | `make gen-api` (oapi-codegen) | yes (`backend/internal/api`) | drift diff |
| sqlc store code | `backend/internal/modules/*/queries`, `migrations` | `make gen-sqlc` | yes | drift diff + `sqlc vet` |
| TS API types | `openapi/` | `make gen-ts` (openapi-typescript) | yes (`web/packages/api-client/src/schema.d.ts`) | drift diff |
| MSW handlers from examples | `openapi/examples` | `make gen-mocks` | yes | drift diff |
| ~~Event JSON Schemas~~ | — | — | — | Replaced by hand-written **JSON fixtures per event** checked by unit tests (C9, doc 20 §6.7) |
| Bundled OpenAPI | `openapi/` | `make api-bundle` | no (artifact) | lint (Redocly) + `oasdiff breaking` vs last release |

All generators are pinned in `tools/tools.go` or `web/package.json`, so builds are reproducible.

---

## 6. Task runner: `Makefile` targets

**Make** is the single entry point (ubiquitous, no extra install; doc 22 already uses `make up`). Targets delegate to `scripts/` or tool CLIs. Taskfile was considered: it is nicer to write, but it is one more prerequisite for contributors. `make help` lists everything.

| Target | Does |
|---|---|
| `make bootstrap` | Check prerequisites and install pinned tools (`go install` from `tools/`, `corepack enable`, `pnpm install`) |
| `make up` / `make down` / `make reset` | Start/stop the local Compose stack / wipe volumes |
| `make up-cache` | Start with the Valkey profile (exercise the Redis adapters) |
| `make migrate` / `make migrate-new name=<module>_<desc>` / `make migrate-down` | goose (+ River) migrations against the local DB / new timestamped file / one step down (local only) |
| `make seed` | Load the Mahabubnagar demo dataset (§8) |
| `make run-api` / `make run-worker` / `make run-all` | Run the Go binary locally with hot reload (Air) |
| `make web-dev app=customer` | Vite dev server for one app (proxy `/api` → `:8080`) |
| `make dev` | `up` + `migrate` + `seed` + api + worker + all four Vite apps (process manager) |
| `make generate` | All generators (§5) |
| `make lint` | golangci-lint (+ depguard), go-arch-lint, Redocly lint, ESLint, Prettier check, `tofu fmt -check`, tflint, hadolint, markdownlint |
| `make test` | Go unit tests + Vitest |
| `make test-integration` | Go integration tests with testcontainers (Postgres+PostGIS, MinIO, Valkey) |
| `make test-e2e` | Playwright against the local stack with fake providers (golden flow) |
| `make test-contract` | Provider adapter contract tests (recorded fixtures) |
| `make load` | k6 scenarios (doc 20) against local or a target URL |
| `make build` / `make image` | Build the binary and web bundles / build multi-arch images locally (buildx) |
| `make api-bundle` / `make api-docs` | Bundle the spec / render a local API reference |
| `make vuln` | govulncheck, `pnpm audit`, trivy fs |
| `make tf-plan env=staging` / `make tf-validate` | OpenTofu plan / validate + checkov (apply runs only from CI) |
| `make tunnel` | Expose the local stack via a cloudflared quick tunnel (demos, PA sandbox webhooks) |
| `make otp phone=+91…` | Print the last fake OTP for a phone (local only) |
| `make help` | List targets |

---

## 7. Configuration conventions

- **12-factor:** all runtime config comes from env vars (`ROVO_*`) or mounted files. There are no config files baked into images. `internal/platform/config` validates at startup, fails fast, and logs a **redacted** config summary.
- **Provider selection is config:** `ROVO_PA_PROVIDER=fake|razorpay|cashfree`, `ROVO_OTP_PRIMARY=sms` (`whatsapp` from V1.1, C2), `ROVO_SMS_PROVIDER=fake|msg91|gupshup`, `ROVO_SMS_SECONDARY_PROVIDER=…`, `ROVO_BOTCHALLENGE=fake|turnstile`, `ROVO_EMAIL_PROVIDER=smtp|resend`, `ROVO_BLOB_ENDPOINT` (MinIO locally), `ROVO_CACHE_BACKEND=memory|redis`, `ROVO_PUBSUB_BACKEND=postgres|redis`.
- **Secrets:** never in the repo. Local defaults live in `deploy/compose/env/local.env.example` (fake values only). Cloud secrets come from the secrets manager via the platform (doc 22). `gitleaks` runs in pre-commit and CI.
- **Business config** (fees, zones, dispatch parameters, feature flags) lives in the **database**, managed via admin, and is seeded for local. It is not in env vars.

---

## 8. Local development setup

**Prerequisites:** Docker (or Podman) with Compose v2, Go 1.27.x, Node 22 LTS+ with corepack (pnpm 12.x) `[ASSUMPTION: Node LTS line at Phase 2 start]`, Make. Optional: mise/asdf (`.tool-versions` pins versions).

**First run (target ≤ 15 minutes on a fresh laptop):**
```
git clone … && cd rovo
make bootstrap
make dev            # compose up → migrate → seed → api+worker → 4 Vite apps
# customer  http://localhost:5173   restaurant http://localhost:5174
# rider     http://localhost:5175   admin      http://localhost:5176
# API       http://localhost:8080   Mailpit    http://localhost:8025
# Grafana   http://localhost:3000   MinIO      http://localhost:9001   River UI http://localhost:8080/admin/river (dev only)
```

**Local services** (Compose, doc 22 §10):
- Postgres 17 + PostGIS 3.5 (multi-arch image, same major as prod, R22; roles mirroring prod).
- MinIO (S3 API) with buckets mirroring prod names.
- **Mailpit** (SMTP catcher).
- **grafana/otel-lgtm** (OTLP → Grafana/Loki/Tempo/Prometheus).
- Valkey (profile `cache`).
- **Fake providers** inside the binary:
  - `fakepay`: a checkout page with Success / Fail / Delay webhook / Late capture buttons, plus webhook replay.
  - `fakeotp`: OTP printed to logs and shown on `make otp`. It also accepts the code `000000` in `env=local` only.
  - `fakepush`/`fakesms`: log sinks with an admin "message log viewer".
  - `BotChallenge=fake`: accepts any token, so login works offline.
- Map tiles: a small PMTiles extract or blank style served from MinIO (`tileStyleUrl` configurable).
- The full golden flow runs **offline** (baseline §4a).

**PA sandbox (optional, mainly staging):** set `ROVO_PA_PROVIDER=razorpay` with test keys in `deploy/compose/env/local.env` (gitignored), run `make tunnel`, and register the tunnel URL as the sandbox webhook. Nobody needs it to develop: the fakepay contract suite covers every doc 14 §20 scenario. Optional SaaS sign-ups (PA sandbox, Grafana Cloud) are never required for the golden flow.

**Seeded Mahabubnagar data** (`backend/migrations/seeds`, generated by `tools/seedgen`, all fictional `[ASSUMPTION: coordinates approximate, to be validated by Ops]`):
- City `Mahabubnagar` (`Asia/Kolkata`, road factor 1.3), 3–4 zone polygons around the town centre, ~25 localities with Telugu names, and PIN codes 509001/509002 `[verify]`.
- 12 demo restaurants (veg and non-veg; biryani, tiffins, Chinese, bakery). Fictional names, menus with `name_i18n` (en + te, R17), packaging rules and hours.
- 8 demo riders with positions spread across zones, 20 demo customers with addresses and landmarks.
- Admin users for each role (TOTP secret printed in the seed output).
- A second **test city** (`Testpur`) for multi-city tests (ADR-020), CI only.
- Fee configs per the doc 16 defaults (R48), slabs to 10 km road distance `[lo,hi)` (R18); sample coupons (platform- and restaurant-funded; `SHARED` funding cut, C16). Seed values come from doc 10 §15.

**Editor setup:** a `.vscode/` recommendations file (Go, ESLint, Prettier, Tailwind, YAML with the OpenAPI schema) is optional and committed as `extensions.json` only.

---

## 9. Licensing

- The repository is **Apache License 2.0** (already present in `LICENSE`).
- Phase 2 adds a **`NOTICE`** file (Apache-2.0 §4d) listing the project copyright and third-party notices where required.
- **SPDX headers** on source files: `// SPDX-License-Identifier: Apache-2.0`. Checked by a lint step (e.g. `addlicense -check`).
- **Inbound = outbound:** contributions are accepted under Apache-2.0 with a **DCO** sign-off (`Signed-off-by`), enforced by a DCO GitHub App or check. No CLA in V1 `[OPEN: maintainers]`.
- **Dependency licence policy:** allowed are Apache-2.0, MIT, BSD-2/3, ISC, MPL-2.0 (file-level, tooling such as OpenTofu), and Unicode/OFL for fonts. Denied are AGPL, SSPL, BSL/BUSL and other non-OSI licences in runtime dependencies. Enforced via `go-licenses` and `license-checker` in CI. Images: Noto Sans Telugu (OFL). Map data: **ODbL attribution** for OSM-derived tiles (ADR-015).
- **Brand and assets:** rovo name and logo usage guidelines in `TRADEMARKS.md` `[OPEN]`. No third-party brand assets (baseline §1: nothing copied from Zomato/Swiggy/DoorDash).
- **Data:** seed data is synthetic. No real restaurant names or menus without written consent `[LEGAL]`.

---

## 10. Community and governance files to create in Phase 2

| File | Content |
|---|---|
| `README.md` | What rovo is, screenshots, architecture diagram (from doc 08), quick start (`make dev`), status, links |
| `CONTRIBUTING.md` | Dev setup, project layout (this doc condensed), module boundary rules, coding standards (Go, TS, SQL, OpenAPI style), tests required, conventional commits, DCO, PR checklist, how to add a migration/module/notification template/provider adapter |
| `CODE_OF_CONDUCT.md` | **Contributor Covenant 2.1**, with enforcement contact |
| `SECURITY.md` | Supported versions. Private disclosure via **GitHub Private Vulnerability Reporting** + security email. Response SLAs (ack 72 h). Scope (in-scope: this repo; out of scope: the operator's production infra unless authorised). Safe harbour. **Never test against production payment flows.** |
| `GOVERNANCE.md` | Maintainers, decision-making (ADRs), how to become a maintainer |
| `CHANGELOG.md` | Generated by release tooling |
| `NOTICE` | Apache-2.0 notice |
| `.github/PULL_REQUEST_TEMPLATE.md` | Summary, linked issue, type, screenshots, checklist (tests, migration expand/contract, API changes reviewed, i18n en+te, a11y, docs/ADR updated, no secrets) |
| `.github/ISSUE_TEMPLATE/*` | Bug (env, steps, expected/actual), feature (problem, proposal), docs. Security reports redirected to SECURITY.md. |
| `docs/contributing/architecture.md` | Contributor-oriented summary of docs 08/09 |
| `docs/runbooks/*` | Deploy, rollback, restore DB (PITR), rotate secrets/JWT keys, PA outage, OTP outage, stuck queue, restaurant alert failure, payout batch errors |
| `SUPPORT.md` | Where to ask questions (Discussions) |

---

## 11. CI path filters (summary for doc 21)

| Path changed | Jobs |
|---|---|
| `backend/**`, `tools/**` | Go lint/test/integration, arch checks, sqlc vet, govulncheck, image build |
| `openapi/**` | Spec lint + oasdiff, **both** Go and TS codegen drift checks, backend + web builds |
| `web/**` | ESLint, typecheck, Vitest, size-limit budgets, Playwright (affected apps), Lighthouse CI on customer |
| `deploy/terraform/**` | `tofu fmt/validate`, tflint, checkov, plan per env (PR comment) |
| `deploy/compose/**`, `deploy/docker/**` | compose config validation, hadolint, local stack smoke test |
| `docs/**` | markdownlint, link check, Mermaid render check |

---

## 12. Open items

- ~~Prefix doc 17's workspace tree with `web/`~~ and ~~rename doc 22's `infra/`~~: decided by R14/R23; the owning docs apply them.
- `[OPEN → DevOps]` Confirm the logical module grouping and the `profiles/` mechanism in §1.4. Pick Renovate vs Dependabot and release-please vs git-cliff.
- `[OPEN → Maintainers]` DCO vs CLA. Require signed commits?
- `[OPEN → Ops]` Validate the seeded zone polygons and locality names for Mahabubnagar.
