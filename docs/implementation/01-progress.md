# Phase 2 — Implementation Progress

> **2026-11-16: FOUNDATION COMPLETE.** See [`HANDOFF.md`](HANDOFF.md) for how to run it. Next milestone is **M1 features** (per `docs/planning/28-milestones-dependencies.md`).

| | |
|---|---|
| **Branch** | `ccr-3091d745-s0lm6o` · **PR** [#1](https://github.com/storetmobiles2-code/rovo/pull/1) |
| **Stage** | Foundation complete; feature phase (M1) next. |

## Foundation — done & verified (2026-11-16)

- **Backend builds, vets clean, 47 tests pass** (`go test -race ./...`).
- **Binary runs**: `rovo migrate up` (12 migrations, 94 app tables), `rovo seed` (idempotent; Mahabubnagar city/zone, 4 restaurants, 45 menu items, 4 riders, 18 ledger accounts, fee config with `platform_fee_paise`=0 per R59), `rovo all` → `/healthz`+`/readyz` 200.
- **15 modules** wired through the `internal/app` composition root onto the api mux (service methods are `NotImplemented` stubs — feature phase fills them).
- **OpenAPI** 6,143 lines, 218 operations; `redocly lint --config openapi/redocly.yaml` clean.
- **DevOps**: compose/caddy/tofu/actionlint/shellcheck/promtool all validate; `scripts/smoke.sh` 47/0 on the prod-like stack; incident runbook per alert.
- **Frontend**: green across 9 packages + 4 apps (172 tests), admin app 21 routes (strict CSP), R59 zero-line hiding; customer initial JS 159.7 KB gzip; i18n te complete.
- **ADRs** P2-001/002/003; `backend-guide.md`; Graphify knowledge graph.

## M1 — started (2026-11-16, branch `feat/m1-identity-geo-catalog`)

Done and covered by integration tests (real PostGIS, full middleware chain via `internal/app/apptest`):

- **Public middleware wired** on the api listener: audience-from-Host, CSRF (cross-origin + `X-Rovo-Client`), edge-origin check, principal extraction (`App.PublicMiddleware`).
- **identity**: `POST /auth/otp/request`, `/auth/otp/verify`, `/auth/refresh`, `/auth/logout` — HMAC-stored 6-digit codes, 5 attempts, resend cooldown 30/60/120/300 s, per-phone and per-IP send budgets, refresh rotation with reuse detection (15 s race window), server-side session check.
- **users**: `GET /me`, `PATCH /me`.
- **geo**: `GET /public/cities`, `/public/serviceability`, `/public/localities`; canonical haversine + road factor in `internal/platform/geo`; fee slab and ETA lookup from `fee_configs`.
- **catalog (read side)**: `GET /public/cuisines`, `/public/restaurants` (filters, sort, signed cursor), `/public/restaurants/{id}`, `/public/restaurants/{id}/menu` (ETag), `/public/search`.

Not done yet in M1: admin email+password+TOTP login, bot challenge (Turnstile) on OTP, failed-verify day limit and lockout SMS, partner/admin catalog write endpoints (onboarding, KYC, menu editing, hours), images (cards return `image: null`), `/config/client`, `/public/offers`, waitlist and leads, `GET /stream`, cart quote, audit log, OpenAPI contract tests against these handlers, and all frontend screens.

Toolchain: Go bumped to **1.26.11** and `golang.org/x/net` to v0.62.0 (govulncheck was failing CI on Go 1.25.0 stdlib advisories); golangci-lint pinned to v2.15.0 to match.

## Known small items (not blocking; fold into feature phase)

- River workers should stop before the pgx pool closes on shutdown (benign "pool is closed" log lines during drain).
- CI/Makefile run `redocly lint` without `--config`, so they show ~6 residual warnings (accepted in `.redocly.lint-ignore.yaml`) — add the flag.
- Placeholder contacts `security@rovo.example`, `conduct@rovo.example` need real addresses before public launch.
- Docker image builds unverified *in this sandbox* (proxy CA untrusted inside build) — should build in normal CI.
- Seed: admin TOTP needs `ROVO_FIELD_ENCRYPTION_KEY` set (else sealed with an ephemeral key); demo customers/historical orders/notification templates not seeded.

## Next: feature phase (see `docs/planning/28` milestones, `27` backlog)

Implement module logic + handlers + jobs behind the OpenAPI contract, milestone by milestone: **M1** identity/geo/catalog/discovery → **M2** ordering+cart/quote+COD → **M3** dispatch+delivery (decide OQ-34 batching/`trips` first) → **M4** payments+ledger+settlement → **M5** admin/ops+notifications+reviews+hardening → QA golden flow + reviews + release. Apply auth/audience middleware to the public mux and mount `GET /stream` early in M1.
