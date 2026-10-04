# 33 — Phase 2 Implementation Prompt

| | |
|---|---|
| **Purpose** | A self-contained prompt for a new Claude Code session (or a team of sessions) that implements rovo V1 from the Phase 1 plan without repeating the planning. |
| **Owner** | Lead Architect |
| **Status** | Final v1 — 2026-10-04 |
| **Dependencies** | Every document in `docs/planning/` (00–32) |

Copy everything between the two horizontal rules below into the new session.

---

## PROMPT — rovo Phase 2: implement V1

You are the **lead engineer** implementing **rovo**, an open-source (Apache-2.0) food delivery platform for **Mahabubnagar, Telangana, India**. Phase 1 (planning) is complete. It lives in `docs/planning/` of this repository. **Do not redo the planning.** Implement it. Where the plan is silent, choose the simplest option consistent with it, record the choice as a short ADR in `docs/adr/`, and move on.

### 1. Read first, in this order (this is your spec)

1. `docs/planning/README.md`: index and the **precedence rule**.
2. `docs/planning/00-planning-baseline.md`, all of it. §2 vocabulary and canonical states; §3 conventions; §4a environments (user directives); **§8 rulings R1–R26 and §9 review resolutions R27–R48 are binding** and override anything else.
3. `32-final-plan-summary.md`: the condensed plan.
4. `27-implementation-backlog.md` and `28-milestones-dependencies.md`: what to build, and in what order.
5. Reference docs, read as each story needs them:
   - schema → `10`
   - API → `11`; the OpenAPI excerpts there seed `openapi/`
   - state machines and all timers → `13`
   - zones, fees and ETA → `16`
   - payments and ledger → `14`
   - notifications → `15`
   - auth and RBAC → `12`
   - threats and SEC requirements → `19`
   - frontend → `17`; PWA → `18`
   - testing → `20`; CI → `21`; deployment → `22`; DR → `23`; observability → `24`; cost → `25`; repo layout → `26`
   - UX flows → `04`–`07`; requirements and acceptance criteria → `01`/`02`

**Precedence when documents disagree:** `00` §9 › `00` §8 › `00` §4a › the parameter-owning doc (timers → `13`; seeds/`app_config` → `10`; fees/commission → `16`; load model → `20`; cost → `25`) › everything else. If a conflict still remains, note it in `docs/planning/PHASE2-NOTES.md` and pick the option closer to the rulings.

### 2. Hard constraints

- **Local Docker only for development.** Do not sign up for, configure or depend on any cloud or free-tier service that needs a credit card. Do not deploy anything. Everything must run with `make dev` on a laptop using **fakes**: fake OTP/SMS sink, fake payment aggregator with a webhook simulator, fake push, MinIO for S3, Mailpit for email, a local Grafana LGTM stack for telemetry. The full golden flow must pass offline.
- **IaC is authored but never applied** (OpenTofu under `deploy/terraform/`, two profiles: `closed-pilot` and `public-launch`). Only `tofu fmt`, `validate`, `tflint` and `checkov` run in CI. Applying it to AWS needs explicit human approval at milestone M6.
- **The repo is public:** no secrets, keys or real personal data in it. Use `.env.example` and the test seed only.
- **No proprietary assets:** do not copy Zomato, Swiggy or DoorDash branding, copy, icons or screens.
- **Stay in V1 scope.** Do not build anything on the deferred or cut lists (`02` out-of-scope; `00` §9 cuts C1–C20). That includes:
  - no surge pricing, WhatsApp OTP, build-time prerendering, identity-aware proxy, ClamAV, hash-chained audit log, day-one partitioning, outbox table or fan-out hop, or Redis at launch;
  - no live GPS map, multi-city operations, AI recommendations, loyalty, subscriptions, native apps, gRPC, Kubernetes or microservices.
- **Money:** integer paise only; floats are banned in money code by a lint rule. Every ledger journal must balance, enforced by a DB trigger and by tests.
- **Time:** use the injected clock everywhere. Never use `time.Now()` directly in domain code, and never use SQL `now()` for business deadlines (R19).
- **Testability hooks from sprint 1 (R19):**
  - injected clock;
  - seedable IDs and order codes;
  - provider fakes;
  - a `/_test/*` control API, compiled only with the `testhooks` build tag, on a separate port, behind an environment guard;
  - CI proves the production image has no test hooks.

### 3. Architecture to build (summary; details in the docs)

- **Backend:** one Go module `backend/`, one binary `rovo`, with subcommands `api | worker | all | migrate | seed | admin | version`.
  - **Structure:** a modular monolith with modules in `internal/modules/<module>`: identity, users, geo, catalog, pricing, promotions, ordering, payments, dispatch, ledger, ratings, notifications, support, adminaudit, reporting. Each module has a nested `internal/` to block cross-module imports. Enforce boundaries with go-arch-lint and depguard. Vendor SDKs are allowed only in `internal/adapters/`.
  - **HTTP and API:** stdlib `net/http` router. The REST API is **OpenAPI-first** in `openapi/`; generate Go strict-server stubs with oapi-codegen and the TypeScript client with openapi-typescript + openapi-fetch.
    - Errors use problem+json (RFC 9457).
    - `Idempotency-Key` is required on creating POSTs.
    - Optimistic concurrency uses a `version` column / ETag.
    - Pagination is cursor-based.
    - Locale comes from `Accept-Language` (en/te).
    - The base path is `/api/v1`, same-origin on every app host (R15, R27).
  - **Data:** PostgreSQL 17 (18 if available) with **PostGIS**, plus pgx, **sqlc** and **goose**. Extensions are limited to postgis, btree_gist, citext, pg_trgm and pgcrypto. IDs are UUIDv7 generated in the app. Translatable fields are `*_i18n` JSONB, and the API returns `nameI18n` + `displayName` (R17).
  - **Async:** **River** jobs. The publisher inserts one job per subscriber in the same transaction (`InsertManyTx`); that is the outbox (R22, R42). Periodic jobs have catch-up semantics (M11).
  - **Real-time:** a single SSE endpoint `GET /api/v1/stream`.
    - Heartbeat every 20 s, plus a `reauth` event.
    - There is no replay buffer; clients refetch on reconnect.
    - Postgres LISTEN/NOTIFY fans events out between processes, with a queue-usage watchdog (M12).
    - Web Push (VAPID) covers backgrounded apps.
  - **Rate limits and OTP counters** are backed by Postgres (R21). Redis interfaces exist, but only in-memory and Postgres implementations are built at V1.
  - **Auth (doc 12):**
    - Phone OTP via an SMS provider interface, with a fake provider in dev. Turnstile is optional and off in dev.
    - Access tokens are Ed25519 JWTs, 10 min (5 min for admin).
    - Refresh tokens are opaque, hashed and rotated, with family revocation on reuse.
    - Restaurant and rider sessions are bound to the device (R44).
    - Admins use email + argon2id password + **mandatory TOTP** (passkeys are P1).
    - Cookies are host-only, `SameSite=Lax` (`Strict` for admin).
    - CSRF protection: Go `CrossOriginProtection`, an `X-Rovo-Client` header, JSON-only bodies, and no CORS.
    - RBAC permission matrix plus ownership checks in domain services; admin roles are scoped by city.
    - Maker-checker covers only the five families in R31.
- **Frontend:** a pnpm workspace in `web/` with **four Vite + React + TypeScript PWAs**: `customer`, `restaurant`, `rider` and `admin` (admin has no service worker).
  - **Shared packages:** `ui` (Tailwind with Radix/shadcn-style components), `api-client` (generated), `i18n` (i18next, en + te, `Intl` en-IN), `utils` (money/phone/address), `config`, `testing` (MSW).
  - **Libraries and assets:** TanStack Router and TanStack Query; react-hook-form + zod; vite-plugin-pwa; MapLibre GL with OpenFreeMap tiles in dev (self-hosted PMTiles later); Noto Sans Telugu subset.
  - **Budgets:** customer initial JS ≤ ~170–200 KB gzip, LCP < 2.5 s on a Moto-G-class device over 4G, and WCAG 2.2 AA.
- **Local stack** (`deploy/compose`): Postgres + PostGIS. Build a multi-arch image yourself, because the official image is amd64-only. Also MinIO, Mailpit, Grafana LGTM, the fake provider service(s), and optionally Valkey behind a profile.
- **Production target** (author only, do not apply): AWS ap-south-1 with DR to ap-south-2.
  - **Compute and data:** ECS Fargate ARM with separate `api` and `worker` services; RDS PostgreSQL + PostGIS (Single-AZ for the closed pilot, Multi-AZ before Gate B).
  - **Edge:** S3 + CloudFront, with `/api/*` routed to the ALB on each app host; WAF.
  - **Security and observability:** Secrets Manager, KMS, Grafana Cloud, and a 180-day CERT-In log archive in India.
  - **Network:** no NAT in the closed pilot, with compensating controls; one NAT before Gate B. GitHub OIDC for deploys.

### 4. Domain rules you must implement exactly

Canonical names (`00` §2):

- **Order statuses:** `PENDING_PAYMENT, PLACED, ACCEPTED, PREPARING, READY_FOR_PICKUP, PICKED_UP, DELIVERED` + `PAYMENT_FAILED, REJECTED, CANCELLED, UNDELIVERABLE`.
- **Delivery statuses:** `UNASSIGNED, OFFERED, ASSIGNED, AT_RESTAURANT, PICKED_UP, AT_DROP, DELIVERED` + `FAILED, CANCELLED`.
- **Offer statuses:** `PENDING, ACCEPTED, DECLINED, EXPIRED, REVOKED` (R16).

Rules (full transition tables and timers in doc 13):

- **R1 — restaurant accept window, 180 s:**
  - re-alert every 30 s; owner SMS at 60 s; ops flag at 90 s;
  - at 180 s → `CANCELLED` by `SYSTEM`, reason `RESTAURANT_UNRESPONSIVE`; refund; 30-minute auto-pause;
  - 2 consecutive misses → paused until the owner resumes;
  - no device heartbeat for 3 min → auto-pause.
- **R2 — customer cancel:** free while `PLACED` or within 60 s of placing; after that, via support only.
- **R3 — preparing:** `ACCEPTED → PREPARING` after 60 s or on the restaurant's tap.
- **R4 — pickup:** allowed from `PREPARING` (with a flag).
- **R5 — undeliverable:** `UNDELIVERABLE` needs support approval.
- **R6 — COD:**
  - per-order cap ₹1,000 (₹600 on a first order);
  - a rider gets a COD offer only if cash in hand + the order ≤ ₹2,000;
  - 2 COD strikes → COD disabled for that customer.
- **R7 — dispatch:**
  - the delivery is created at `ACCEPTED`;
  - the offer is timed at `max(0, prep − approach − buffer)`;
  - 45 s offer timeout, then cascade to the next rider;
  - tiers: fresh location ≤ 3 min, or stale ≤ 15 min reached via push (R34);
  - admin can assign manually;
  - one active delivery per rider.
- **R8 — billing:**
  - menu prices exclude 5% GST, which is shown as its own line;
  - fee GST is configurable;
  - the payable amount is rounded to the rupee with a `ROUND_OFF` line.
- **Fees** (doc 16 owns them): slabs on road distance (straight-line × 1.3), lower bound inclusive and upper exclusive — 0–2 km ₹20 · 2–4 ₹30 · 4–6 ₹40 · 6–8 ₹50 · 8–10 ₹60. Serviceability: the customer is inside an active zone and within 7 km straight-line of the restaurant. Platform fee ₹5; small-cart fee ₹15 below ₹149. Commission defaults to 15%, effective-dated per restaurant. **No surge.**
- **R12 — cart:** no server cart. `POST /api/v1/cart/quote` returns a signed `quoteId` with a 10-minute TTL. Order creation needs `quoteId` + `Idempotency-Key`; a stale quote → 409 with a diff.
- **R29 / R9 — COD compensation:** the customer chooses a manual UPI refund (UTR recorded) or a single-user coupon. There is no wallet.
- **R39 — delivery OTP:** on for prepaid orders ≥ ₹300.
- **R40 — restaurant cancel after accept:** ops-mediated only.
- **R13 — address:** landmark + map pin required; building/street optional; PIN code 6 digits; phones E.164 +91, Indian mobile numbers only.
- **Payments:**
  - `PaymentProvider` interface with a fake adapter (dev/CI), then a Cashfree or Razorpay sandbox adapter — decided at PA selection (R25), keep both possible;
  - webhook HMAC verification plus fetch-to-confirm;
  - daily reconciliation job;
  - full or partial refunds to source;
  - double-entry ledger with weekly settlement;
  - manual payouts recorded with UTR;
  - design supports both PA split settlement and collect-and-payout (R35).

### 5. How to work

- **Order of work:** follow the milestones in `28` (development milestones run entirely on local Docker).
  1. **M0 Foundations:** repo bootstrap; Make targets; Compose stack; CI skeleton; OpenAPI 3.1-subset tooling spike (R20, fall back to 3.0.3); testhooks; platform packages (config, logging/OTel, db/migrations, River, problem+json, idempotency, rate limit, clock/ID).
  2. **M1** identity + geo + catalog + discovery.
  3. **M2** ordering + quote + COD, end to end on local.
  4. **M3** dispatch and delivery, completing the golden flow.
  5. **M4** online payments (fake PA, then sandbox adapter), ledger, settlement, payouts, COD cash.
  6. **M5** admin/ops, notifications hardening, ratings, support, reports, audit.
  7. **M6** onward (cloud staging, security/performance/DR, pilot, launch) needs human approval and cloud accounts. **Stop and ask before M6.**
- **Workflow:**
  - Work in small vertical slices: migration → sqlc queries → domain logic with table-driven tests → OpenAPI path → handler → generated TS client → UI screen → E2E step.
  - Every PR keeps `make lint test test-integration` green.
  - Golden-flow Playwright E2E (customer, restaurant, rider and admin contexts) is added at M3 and must stay green afterwards.
- **Tests (doc 20):**
  - unit tests;
  - property tests (`rapid`) for pricing, ledger and slab coverage;
  - testcontainers integration tests with PostGIS;
  - contract tests against OpenAPI (kin-openapi validation in test mode, plus oasdiff);
  - an authorisation matrix test and IDOR tests;
  - Vitest, Testing Library, MSW, axe;
  - Playwright E2E; k6 scripts.
  - Coverage: ≥ 90% for pricing, ledger and state machines; ≥ 70% for Go overall.
- **CI (doc 21) on GitHub-hosted runners only:**
  - **Go checks:** gofmt, golangci-lint, go-arch-lint, govulncheck, tests with race detection, sqlc diff, squawk, migration up/down.
  - **API checks:** Redocly lint, oasdiff, generated-code drift check.
  - **Web checks:** typecheck, eslint, vitest, build, bundle budget.
  - **Security and images:** Playwright, gitleaks, osv-scanner, CodeQL, Trivy, multi-arch image build, SBOM.
  - **IaC:** tofu validate/tflint/checkov.
- **Seeds:** Mahabubnagar city, localities and one zone polygon. These are tagged `[ASSUMPTION]` in `10`/`16`; keep them clearly marked as demo data. Also seed demo restaurants with menus (variants and add-ons), riders, an admin, and coupons.
- **Docs to keep up to date:** `docs/adr/` (split ADRs out of `09`), `docs/runbooks/`, `README.md` quick-start, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`.
- **Commits:** conventional commits on short-lived branches. Open PRs only when the human asks.

### 6. Definition of done for V1 development (end of M5, local)

1. `make dev` on a clean machine brings up all four apps, the API and the worker with seeded Mahabubnagar data.
2. Playwright passes the **golden flow**: customer browses → quotes → checks out (COD and fake-online) → restaurant accepts → preparing → ready → rider offered → accepts → picked up → delivered (with OTP for prepaid ≥ ₹300) → customer rates → ledger journals balance → weekly settlement and payout batch with maker-checker → reports export.
3. Failure flows pass:
   - restaurant reject (refund);
   - 180 s timeout;
   - no rider (admin manual assign);
   - offer timeout cascade;
   - duplicate, out-of-order or delayed webhooks;
   - customer cancel in and out of the grace window;
   - COD cash-limit block;
   - `UNDELIVERABLE` with support approval.
4. Security checks pass: the authorisation matrix and IDOR tests; OTP abuse limits; webhook replay rejected; no secrets in the repo; SEC-xxx P0 requirements from `19` have tests.
5. Telugu (te) catalogs are complete (CI check). The pseudo-locale renders. axe reports no serious violations.
6. k6 runs locally (smoke level). The capacity test at 3× the R45 design point is scheduled for staging at M6.
7. OpenTofu for both profiles validates, and the runbooks exist.

### 7. Decisions that need humans (do not block on them; use the stated default)

| Decision | Default until decided |
|---|---|
| Payment aggregator (Cashfree vs Razorpay; written UPI rate ≤ 1% blended target) | Fake PA; build a sandbox adapter for whichever account exists first |
| Restaurant money flow: split settlement vs collect-and-payout (legal opinion by Phase 2 week 4) [LEGAL] | Model both in the ledger; collect-and-payout path used in tests |
| GST presentation on fees; discount treatment; TDS section [LEGAL] | Configurable tax lines; seeds as in `10`/`14` |
| Real zone polygon and locality list (ops) | `[ASSUMPTION]` fixtures |
| Domain name and brand assets | `rovo.example` placeholders; neutral icon set |
| SMS provider and sender-registration templates | Fake SMS provider; template keys in `15` |
| Rider minimum guarantee, accident insurance, gig-worker registration specifics [LEGAL] | `MG_TOPUP` adjustment type; gig-worker fields + export implemented |

Non-code critical-path items for the humans, started in Phase 2 week 1 (doc 28): legal entity, GST e-commerce-operator registration, payment-aggregator onboarding/KYC, SMS sender (DLT) registration, privacy policy/T&C/grievance officer, restaurant and rider recruitment, counter-device procurement, support phone line.

### 8. Reporting

At the end of each milestone, write a short status in `docs/planning/PHASE2-NOTES.md`: what was built, deviations with ADR links, open issues, and what's next. Then stop and summarise for the human.

---
