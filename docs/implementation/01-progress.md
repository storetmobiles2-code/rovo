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

## Known small items (not blocking; fold into feature phase)

- River workers should stop before the pgx pool closes on shutdown (benign "pool is closed" log lines during drain).
- CI/Makefile run `redocly lint` without `--config`, so they show ~6 residual warnings (accepted in `.redocly.lint-ignore.yaml`) — add the flag.
- Placeholder contacts `security@rovo.example`, `conduct@rovo.example` need real addresses before public launch.
- Docker image builds unverified *in this sandbox* (proxy CA untrusted inside build) — should build in normal CI.
- Seed: admin TOTP needs `ROVO_FIELD_ENCRYPTION_KEY` set (else sealed with an ephemeral key); demo customers/historical orders/notification templates not seeded.

## Next: feature phase (see `docs/planning/28` milestones, `27` backlog)

Implement module logic + handlers + jobs behind the OpenAPI contract, milestone by milestone: **M1** identity/geo/catalog/discovery → **M2** ordering+cart/quote+COD → **M3** dispatch+delivery (decide OQ-34 batching/`trips` first) → **M4** payments+ledger+settlement → **M5** admin/ops+notifications+reviews+hardening → QA golden flow + reviews + release. Apply auth/audience middleware to the public mux and mount `GET /stream` early in M1.
