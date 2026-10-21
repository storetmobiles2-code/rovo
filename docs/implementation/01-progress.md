# Phase 2 — Implementation Progress

| | |
|---|---|
| **Updated** | 2026-10-21 |
| **Branch** | `ccr-3091d745-s0lm6o` |
| **Stage** | Foundation (pre-feature). Build-green; no feature endpoints wired yet. |
| **Charter** | `docs/implementation/00-implementation-charter.md` |

## Snapshot

Three foundation agents (Architect/DB, DevOps, Frontend) are building the scaffold in parallel. They have been interrupted repeatedly by the shared usage limit and resumed each time; all work is committed. **Nothing is lost.** The next reset is 00:20 UTC.

**Legend:** ✅ done & validated · 🟡 in progress / partial · ⬜ not started

## Backend (`backend/`)

| Item | Status | Notes |
|---|---|---|
| Go module, `go 1.25`, builds | ✅ | `go build ./...` green at last check |
| Migrations (goose, embedded) | ✅ | **12 migrations, 94 tables**, apply cleanly on PostGIS 17 |
| Ledger balance + append-only triggers | ✅ | `trg_ledger_postings_balanced`, `*_immutable`, bill-line balance triggers verified |
| Platform packages | ✅ | `app apperr auth blob clock config db devoutbox events health httpx idgen migrate money obs queue ratelimit realtime testhooks testutil` all present |
| — money (paise, half-up rounding) | ✅ | unit-tested |
| — auth (Ed25519 JWT, argon2id, TOTP, field crypto, refresh rotation) | ✅ | unit-tested |
| — clock / idgen (UUIDv7, order codes) | ✅ | unit-tested |
| — httpx, realtime (SSE), queue (River), ratelimit, obs (OTel/Prom), testhooks | 🟡 | on disk; integration wiring pending |
| `cmd/rovo` (api\|worker\|all\|migrate\|seed\|version) | ⬜ | **not yet** — no runnable binary |
| Module services (`internal/modules/*`) | ⬜ | **not yet** — the 15 domain modules are empty |
| Seed data (Mahabubnagar) | ⬜ | pending |

## API contract (`openapi/`)

| Item | Status | Notes |
|---|---|---|
| `openapi.yaml` | 🟡 | ~3,970 lines; public/auth/me/cart/orders/payments/partner/rider paths in; **admin + webhooks + stream remaining**; redocly lint not yet green |

## Frontend (`web/`)

| Item | Status | Notes |
|---|---|---|
| pnpm workspace + 9 shared packages | ✅ | `api-client auth config i18n pwa shell testing ui utils` — packages pass typecheck/lint/test |
| Brand icons (plate/wheel marks → PWA 192/512/maskable) | ✅ | from `docs/brand/` |
| App shells | 🟡 | **customer, restaurant, rider done**; **admin app in progress** (no SW, English, email+TOTP login) |
| `gen:api` wired to openapi.yaml | 🟡 | regenerate once spec lints |

## DevOps (`deploy/`, `.github/`)

| Item | Status | Notes |
|---|---|---|
| Local compose (postgis/minio/mailpit) | 🟡 | on disk; `docker compose config` validation pending |
| Prod-like compose + Caddy (4 hosts, same-origin /api) | 🟡 | Caddyfile formatting fixed; validate pending |
| Dockerfiles (backend distroless, web caddy) | 🟡 | on disk |
| Terraform/OpenTofu (modules + envs, validate-only) | 🟡 | modules: network/database/compute/edge/iam/secrets/storage/observability; **provider binaries gitignored** (`.terraform/`) |
| CI workflows | 🟡 | `ci.yml`, `codeql.yml` present; **dependabot, CODEOWNERS, PR/issue templates remaining** |
| Backup/restore/verify scripts | ✅ | `scripts/` with negative tests (tampered checksum, row-count, unbalanced ledger) |
| Docs (README/CONTRIBUTING/SECURITY/NOTICE + trademark) | ⬜ | pending |

## Validated so far (real, not claimed)

- Backend compiles (`go build ./...`).
- All 12 migrations apply on a real `postgis/postgis:17-3.5` container → 94 tables.
- Ledger-balance and append-only triggers exist and fire.
- No secrets or >100 MB files in git history (a 693 MB stray Terraform provider binary was caught and purged; `.terraform/` now ignored).

## Remaining before the foundation is "done"

1. `cmd/rovo` entrypoint (so the API/worker actually run).
2. Finish + lint `openapi.yaml` (admin, webhooks, stream).
3. Backend module skeletons (interfaces, so feature engineers can start).
4. Finish admin app; `pnpm -r build` green across all 4 apps.
5. DevOps: compose/caddy/tofu validate green; remaining CI files; docs.
6. One end-to-end **smoke**: `make prod-up` → health on all 4 hosts.

## Then (feature phase, per charter §3/§5)

Backend A/B/C engineers implement the 15 modules against the contract; frontend app engineers build the screens; QA writes the golden-flow Playwright run; security/code/UX reviews; release. The golden flow (Customer → Restaurant → Order → Rider → Pickup → Delivery → Payment/Settlement → Rating) is the acceptance gate.

## Known risks / notes

- **Usage limit** is the main throttle: agents stop mid-task and resume cleanly after each reset. Progress is durable (committed), so this costs wall-clock time, not work.
- **Deployment** for this phase is the production-mode Docker stack (ADR-P2-003), not AWS — AWS needs accounts/approval at M6.
- Container is ephemeral: Docker daemon + test DB are re-created on each resume.
