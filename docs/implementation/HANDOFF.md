# rovo — Session Handoff

| | |
|---|---|
| **Date** | 2026-11-16 |
| **Branch** | `ccr-3091d745-s0lm6o` · **PR** [#1](https://github.com/storetmobiles2-code/rovo/pull/1) (open → `main`) |
| **Why this exists** | The previous session built the plan and most of the Phase-2 foundation but was repeatedly interrupted by usage limits (5-hour and weekly). This file is the single, accurate starting point for the next session. |

## Read first (authoritative, in order)

1. `docs/implementation/00-implementation-charter.md` — toolchain, V1 slice, conventions, module ownership, the 3 ADR deviations.
2. `docs/implementation/01-progress.md` — per-area ✅/🟡/⬜ status.
3. `docs/planning/00-planning-baseline.md` §8/§9/§9.1 — **binding rulings R1–R59** (R59 = **no platform fee**; `platform_fee_paise` default 0, zero line hidden).
4. `docs/planning/28-milestones-dependencies.md` — milestone sequence M0→M8.
5. `docs/planning/27-implementation-backlog.md` — 197 stories.

## Honest state as of this handoff

**✅ Done & verified**
- Backend **builds** (`cd backend && GOTOOLCHAIN=auto go build ./...`).
- 12 goose migrations apply on real PostGIS 17 → **94 tables**; ledger-balance + append-only triggers fire; `fee_configs.platform_fee_paise` default = 0 (R59).
- Platform packages present & core-tested: `money, auth (Ed25519/argon2id/TOTP/fieldcrypt), clock, idgen, config, db, httpx, queue, realtime (SSE), ratelimit, obs, health, blob, events, testhooks`.
- Web: pnpm workspace + 9 shared packages + 4 app **shells** (customer/restaurant/rider/admin).
- CI templates: `.github/` CODEOWNERS, dependabot, PR + issue templates; `ci.yml`, `codeql.yml`.
- Graphify knowledge graph wired (`graphify-out/graph.json`, `.claude/` skill+hooks). Rebuild with `make graph`.
- OpenAPI `openapi/openapi.yaml` ~3,970 lines (most paths).

**🟡 / ⬜ NOT done (this is the remaining work)**
- ⬜ **`backend/cmd/rovo` is EMPTY** — there is no runnable binary yet. (A prior commit message claimed it existed; it does not. Git does not track empty dirs.)
- ⬜ **`backend/internal/modules/*` are skeletons** (≈4 `.go` files across 15 dirs) — no real domain logic, no HTTP handlers, no jobs wired.
- 🟡 OpenAPI admin/webhooks/stream paths + `redocly lint` green — verify/finish.
- 🟡 `rovo seed` (Mahabubnagar data) — not confirmed present.
- 🟡 DevOps validation (`docker compose config`, `caddy validate`, `tofu validate`, actionlint, shellcheck) + README/CONTRIBUTING/SECURITY/NOTICE — partially done.
- 🟡 Frontend: apps are shells; one auth test was failing; `pnpm -r build` green not confirmed; a stray debug file `web/packages/auth/src/zzdbg.test.tsx` may exist — delete it.

## Environment setup for a new session (the container is ephemeral)

```bash
# 1. Docker daemon (needed for Postgres/PostGIS)
(nohup dockerd >/tmp/dockerd.log 2>&1 &); sleep 8; docker info >/dev/null && echo up

# 2. Test DB
docker run -d --name rovo-pg-test -e POSTGRES_PASSWORD=postgres -p 55432:5432 postgis/postgis:17-3.5
export ROVO_TEST_DATABASE_URL=postgres://postgres:postgres@localhost:55432/postgres?sslmode=disable

# 3. Go toolchain auto-downloads 1.25
export GOTOOLCHAIN=auto

# 4. Graphify CLI (project CLAUDE.md requires querying the graph before grep)
uv tool install "graphifyy[sql,terraform]"; export PATH="$HOME/.local/bin:$PATH"
```

## Guardrails (do not relearn the hard way)

- **Never `git add` `deploy/terraform/**/.terraform/`** — 693 MB provider binaries; GitHub rejects >100 MB. They are gitignored; keep it so. Same for `*.tfstate*`.
- **Commit only real files** — empty dirs aren't tracked; don't claim a file exists until `find`/`git show --stat` proves it.
- Attribution footer on commits/PRs per the session reminder.
- Deployment target this phase is **production-mode Docker Compose** (ADR-P2-003), not AWS. IaC is authored/validated only.

## The remaining path (milestone order, from doc 28)

1. **Finish foundation → runnable**: write `backend/cmd/rovo` (`api|worker|all|migrate|seed|version`), wire the 15 modules as stub registrations so the binary starts, finish+lint OpenAPI, add `rovo seed`, bring up `make prod-up` (or compose) and curl `/healthz`.
2. **M1** identity + geo + catalog + discovery.
3. **M2** ordering + cart/quote + COD end-to-end on local.
4. **M3** dispatch + delivery → **golden-flow E2E** (decide OQ-34 batching/`trips` entity *before* writing dispatch — see `docs/research/competitor-paved-path-review.md`).
5. **M4** payments (fake PA → sandbox) + ledger + settlement/payouts.
6. **M5** admin/ops + notifications + ratings + reviews + hardening.
7. QA golden flow + security/code/UX reviews + release.

Each PR keeps `make check` green (lint + unit + integration + contract). The golden flow (Customer → Restaurant → Order → Rider → Pickup → Delivery → Payment/Settlement → Rating) is the acceptance gate.

## Open decisions for the product owner (log in `docs/planning/30-...md`)

- **OQ-34** (by M3): add a `trips` entity for batching to the dispatch schema now, or not. Recommendation: yes (cheap now, a migration later).
- **OQ-P19** (by M4): small-cart fee / packaging / free-delivery threshold, given Swiggy Toing & 99 Store pricing (RK-55). Platform fee already set to ₹0 (R59).
