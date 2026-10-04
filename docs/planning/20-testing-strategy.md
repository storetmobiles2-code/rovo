# 20 — Testing Strategy

| Field | Value |
|---|---|
| **Purpose** | Define how rovo proves that it is correct, safe and fast enough to run real money and real food through it in Mahabubnagar: quality goals, risk-based priorities, the test pyramid with concrete tools and conventions, test data, environments and fakes, CI gating, Definition of Done, bug severity, release quality gates and requirement traceability. |
| **Owner** | QA Architect |
| **Status** | Draft v1 (Phase 1 — planning only; no test code exists yet) |
| **Date** | 2026-10-04 |
| **Depends on** | `00-planning-baseline.md` (vocabulary, provisional decisions P1–P17, commercial defaults) |
| **Consumes (when available)** | 01 product requirements (requirement IDs), 10 DB schema, 11 API spec, 12 auth/RBAC (role × operation policy), 13 order state machine (transition tables), 14 payment architecture (PA choice, ledger accounts), 15 notifications (OTP/SMS/push providers), 16 delivery zones (serviceability rules), 17/18 frontend + PWA, 19 threat model, 21 CI/CD, 22 deployment, 23 backup/DR, 24 observability, 25 hosting |
| **Consumed by** | 21 CI/CD (pipeline stages and budgets), 26 repository structure (test folders, fakes, seeds), 27 backlog (test stories), 28 milestones (quality gates), 29 production readiness (release gates), 33 Phase 2 prompt |

> None of documents 01–26 existed in `docs/planning/` when this draft was written (checked 2026-10-04). Everything that depends on them is written against the baseline and marked `[OPEN]` where the owning doc must confirm. The Release/Lead Architect should reconcile in the review pass.

Tags used: `[ASSUMPTION]` unverified premise, `[OPEN]` decision pending elsewhere, `[LEGAL]` needs legal/tax review, `[VERIFY]` external fact to re-check at implementation time.

---

## Table of contents

1. [Decisions summary](#1-decisions-summary)
2. [Quality goals and risk-based prioritisation](#2-quality-goals-and-risk-based-prioritisation)
3. [Test pyramid overview](#3-test-pyramid-overview)
4. [Go unit tests](#4-go-unit-tests)
5. [Property-based tests and fuzzing](#5-property-based-tests-and-fuzzing)
6. [Go integration tests (Postgres + PostGIS, River, outbox, SSE)](#6-go-integration-tests)
7. [Contract tests (OpenAPI, payment & OTP providers)](#7-contract-tests)
8. [Authorisation tests (role matrix + IDOR)](#8-authorisation-tests)
9. [Frontend tests (Vitest, MSW, i18n, a11y, visual)](#9-frontend-tests)
10. [End-to-end tests (Playwright multi-actor)](#10-end-to-end-tests)
11. [Mobile / PWA tests](#11-mobile--pwa-tests)
12. [Performance and load tests (k6)](#12-performance-and-load-tests)
13. [Resilience / chaos-lite](#13-resilience--chaos-lite)
14. [Security testing](#14-security-testing)
15. [Data, migration and backup-restore testing](#15-data-migration-and-backup-restore-testing)
16. [Localisation QA and accessibility audit](#16-localisation-qa-and-accessibility-audit)
17. [UAT, pilot, dogfooding, bug bash](#17-uat-pilot-dogfooding-bug-bash)
18. [Test data management](#18-test-data-management)
19. [Test environments and provider fakes](#19-test-environments-and-provider-fakes)
20. [Testability hooks required from backend and frontend](#20-testability-hooks-required-from-backend-and-frontend)
21. [CI gating, coverage, flaky tests, runtime budgets](#21-ci-gating-coverage-flaky-tests-runtime-budgets)
22. [Definition of Done, bug severity, triage SLAs, release gates](#22-definition-of-done-bug-severity-triage-slas-release-gates)
23. [Traceability](#23-traceability)
24. [What we are deliberately not doing](#24-what-we-are-deliberately-not-doing)
25. [Open questions](#25-open-questions)
26. [Challenges to the baseline](#26-challenges-to-the-baseline)
27. [Sources](#27-sources)

---

## 1. Decisions summary

| # | Decision | Choice | Why |
|---|---|---|---|
| QA-1 | Go test framework | stdlib `testing` + `github.com/stretchr/testify/require` for assertions + `github.com/google/go-cmp/cmp` for diffs. Table-driven by default. | Ubiquitous, no magic, contributors already know it. |
| QA-2 | Property-based testing | `pgregory.net/rapid` (generics, automatic shrinking, `StateMachine` support) for pricing, ledger and state-machine invariants. | Finds the 1-paisa bugs example tables miss. |
| QA-3 | Fuzzing | Go native fuzzing (`go test -fuzz`) for parsers/verifiers (webhook payloads + signatures, phone numbers, order codes, PIN codes, cursor tokens). Nightly, time-boxed. | Built-in, no extra tool. |
| QA-4 | DB for tests | **Real PostgreSQL + PostGIS** (same major versions as prod) via `testcontainers-go` locally and in CI; **template-database per test** isolation; transaction-per-test only for pure read/query tests. No SQLite, no mocks of the DB layer for integration tests. | PostGIS, `SKIP LOCKED`, River and outbox semantics cannot be faked. |
| QA-5 | Integration test selection | Go build tag `integration` (`//go:build integration`). `go test ./...` = unit only (no Docker); `go test -tags=integration ./...` = everything. | Fast inner loop, explicit CI stage. |
| QA-6 | Contract | OpenAPI 3.1 file is the contract. Every API integration/E2E request and response is validated against it by a `kin-openapi/openapi3filter` middleware in **test mode** (fail on mismatch). `oasdiff` breaking-change gate on PRs. Codegen drift check for Go server + TS client. | Spec-first only works if the spec is enforced. |
| QA-7 | Authorisation | **Generated** role × operation matrix test from an OpenAPI extension `x-rovo-authz` (proposed) + hand-written IDOR suites per resource. Test fails if any operation lacks a policy. | Missing authz checks are the #1 API risk; generation makes "forgot to add a test" impossible. |
| QA-8 | Frontend unit/component | Vitest + Testing Library + `user-event`; MSW for HTTP; `vitest-axe` (or `jest-axe`-compatible matcher) for component a11y. | Vite-native, fast. |
| QA-9 | E2E | **Playwright** (TypeScript), Chromium as primary (Android-Chrome-like), WebKit smoke for iOS Safari PWA; multi-actor tests with 4 browser contexts; against a docker-compose stack with wire-level fake providers. | One tool for E2E, a11y (`@axe-core/playwright`), screenshots, device emulation and CDP throttling. |
| QA-10 | Visual regression | Playwright `toHaveScreenshot` on a curated ~30-screen set (en + te, 360×800), run inside the pinned Playwright Docker image; **nightly, non-blocking** for the first 2 milestones, then blocking on `main`. No paid SaaS. | Free; catches Telugu layout breakage. |
| QA-11 | Load | **k6** (OSS) scripts in repo; SSE soak via the `xk6-sse` extension or, if unsuitable, a small Go SSE soak harness `[OPEN]`. Load is generated from outside the target VM. | Free, scriptable, thresholds as code. |
| QA-12 | Chaos-lite | Scripted experiments with `docker compose kill/pause/restart` + **Toxiproxy** for latency/timeouts to PA/OTP fakes and to Postgres. No chaos platform. | Proportionate to a single-VM V1. |
| QA-13 | DAST | **ZAP baseline (passive) scan** on every `main` merge against the ephemeral E2E stack; ZAP **API scan** (active, using the OpenAPI file) weekly against the ephemeral stack only, never against prod. | Free; passive scan is cheap and safe. |
| QA-14 | Supply chain | `govulncheck`, `osv-scanner` (Go + pnpm lockfile), Trivy image scan, `gitleaks`, CodeQL (free for public repos), Dependabot/Renovate `[OPEN: 21]`. | Free for public repos. |
| QA-15 | Time & IDs | All business time via an injected `Clock`; River gets the same clock via `river.Config.Test.Time` in tests; UUIDv7 and order-code generators injectable and seedable. **No business deadline may be computed with SQL `now()`.** | Timeouts (45 s offer, restaurant accept) must be testable without sleeping. |
| QA-16 | Test-control surface | Non-prod-only `/_test/*` API (OTP sink, fake-PA control, clock, seed/reset, push sink) compiled only with build tag `testhooks`, on a separate listener, plus env guard and shared secret. Prod image is built without the tag, and a CI test proves `/_test/*` is absent from it. | E2E needs control; prod must be provably free of it. |
| QA-17 | Coverage | ≥ 90 % statements for pricing/quote, ledger, order & delivery state machines, coupons, commission/rider pay, payouts; 100 % of transition-table rows; 100 % operations in authz matrix; ≥ 70 % overall Go; ≥ 85 % frontend shared packages (money, cart, i18n utils), ≥ 60 % per frontend app. | Meaningful where money and state live; not vanity elsewhere. |

---

## 2. Quality goals and risk-based prioritisation

### 2.1 Quality goals (what "good" means for V1)

| ID | Goal | Measurable target (V1 pilot) |
|---|---|---|
| QG-1 | **Money is exact** | 0 paise discrepancy between customer bill, PA capture/refund, ledger and payout statements; every journal balances (Σ debit = Σ credit) — enforced by test, DB constraint/trigger `[OPEN: 10/14]` and nightly reconciliation job. |
| QG-2 | **Order and delivery state never corrupt** | No order/delivery reaches a state not allowed by the canonical transition tables; no order is assigned to two riders; no terminal state is left. |
| QG-3 | **Dispatch is reliable** | Every `READY_FOR_PICKUP`-bound order either gets an `ASSIGNED` rider or raises an admin alert within the configured window; offer timeouts fire within 45 s ± 5 s under load. |
| QG-4 | **Access is correctly scoped** | 0 IDOR / cross-tenant reads in authz suites; 100 % of operations covered by matrix. |
| QG-5 | **Serviceability is right** | Customers outside active zones or beyond restaurant radius can never place an order; boundary behaviour matches doc 16. |
| QG-6 | **Usable in Telugu and English on entry-level Android** | 100 % `te` key coverage, native-speaker sign-off, no truncation in the curated screen set; customer app interactive in < 5 s on emulated low-end device + Slow 4G `[ASSUMPTION: final budget in 18]`. |
| QG-7 | **Fast enough** | p95 < 300 ms for read APIs, p95 < 800 ms for order create (excluding PA round-trip), SSE event delivered < 2 s after commit, at 3× expected peak. |
| QG-8 | **Recoverable** | Backup restore verified weekly; worker/DB restarts lose no outbox events or timers. |

### 2.2 Risk register driving test intensity

Likelihood (L) and impact (I) on 1–5; Score = L × I. Tier decides how much testing a module gets.

| Risk ID | Risk | L | I | Score | Tier | Primary defences (sections) |
|---|---|---|---|---|---|---|
| R-MONEY-1 | Rounding/fee/GST computation off by paise; totals disagree between quote, order, invoice, ledger | 4 | 5 | 20 | **T1** | Golden tables (§4.3), rapid invariants (§5), E2E ledger assertions (§10) |
| R-MONEY-2 | Duplicate/out-of-order/missing PA webhook → double capture credit, order stuck `PENDING_PAYMENT`, refund twice | 4 | 5 | 20 | **T1** | Webhook contract + idempotency tests (§7.3), E2E failure scenarios F-4 (§10.4), reconciliation job tests |
| R-MONEY-3 | Refund exceeds captured amount / refund on COD order / partial refunds miscounted | 3 | 5 | 15 | **T1** | Invariant INV-L4 (§5), unit + E2E |
| R-MONEY-4 | COD cash-in-hand mis-tracked; rider limit not enforced | 3 | 4 | 12 | **T1** | Ledger unit tests, E2E F-7 |
| R-STATE-1 | Illegal transition (e.g., `PICKED_UP` → `CANCELLED` without refund path), lost update under concurrency | 4 | 5 | 20 | **T1** | Exhaustive transition tests, rapid state machine, concurrent integration tests (§6.6) |
| R-DISP-1 | Offer timeout not firing (worker down, clock bugs) → orders stranded at `READY_FOR_PICKUP` | 3 | 5 | 15 | **T1** | River job tests with stubbed time, chaos C-1, E2E F-3/F-8 |
| R-DISP-2 | Double assignment (two riders accept concurrently; late accept after expiry) | 3 | 5 | 15 | **T1** | Race tests (§6.6), DB uniqueness constraints `[OPEN: 10]` |
| R-AUTH-1 | IDOR: customer reads other customer's order/address; restaurant staff sees another outlet; admin scope leaks across city | 4 | 5 | 20 | **T1** | Generated matrix + IDOR suites (§8), ZAP |
| R-AUTH-2 | OTP brute force / SMS pumping (cost attack) | 4 | 4 | 16 | **T1** | Rate-limit tests (§14.4), k6 abuse scenario |
| R-SVC-1 | Serviceability wrong at zone boundary / polygon holes / radius vs road distance mismatch | 3 | 4 | 12 | **T2** | PostGIS fixture tests (§6.4) |
| R-RT-1 | SSE dropped behind CDN/proxy; restaurant misses new order | 3 | 4 | 12 | **T2** | SSE integration, staging-through-CDN test, polling fallback E2E, soak |
| R-PWA-1 | Stale service worker serves old app against new API (breaking change) | 3 | 4 | 12 | **T2** | SW update-flow E2E (§11.3), oasdiff gate |
| R-I18N-1 | Missing/wrong Telugu strings, overflow, unreadable fonts on low-end devices | 4 | 3 | 12 | **T2** | Key parity, pseudo-locale, visual, native review (§16) |
| R-PERF-1 | Free VM saturates at dinner peak | 2 | 4 | 8 | **T2** | k6 capacity test (§12) |
| R-DATA-1 | Migration breaks prod data / irreversible | 2 | 5 | 10 | **T2** | Migration up/down + prod-shape snapshot test (§15) |
| R-DATA-2 | Backups unrestorable | 2 | 5 | 10 | **T2** | Weekly restore verification (§15.3) |
| R-PRIV-1 | Personal data leaks into logs/test envs (DPDP) | 3 | 4 | 12 | **T2** | Log-redaction tests, anonymisation rules (§18.5) |
| R-UX-1 | Admin/ops UI defects (reports, filters) | 3 | 2 | 6 | **T3** | Component tests, manual exploratory |
| R-CONTENT-1 | Menu image upload edge cases | 2 | 2 | 4 | **T3** | Unit + one E2E |

**Tier policy**

| Tier | Required | Coverage target | Review |
|---|---|---|---|
| T1 (money, state, dispatch, authz) | Unit (table + golden) **and** property-based **and** integration **and** ≥ 1 E2E path; mutation testing sampled weekly `[OPEN]` | ≥ 90 % statements; 100 % transition rows / matrix cells | Two reviewers, one with domain ownership (Backend or Finance-ops); test changes reviewed as carefully as code |
| T2 | Unit + integration; E2E where user-visible | ≥ 75 % | One reviewer |
| T3 | Unit/component; manual exploratory | ≥ 60 % (module) | One reviewer |

---

## 3. Test pyramid overview

```mermaid
flowchart TB
  subgraph Manual["Manual / human (pre-release, pilot)"]
    UAT["UAT & pilot with Mahabubnagar restaurants + riders"]
    RD["Real-device smoke checklist"]
    L10N["Telugu native review, a11y audit, bug bash"]
  end
  subgraph E2E["E2E (Playwright, compose stack, wire-level fakes) ~40-60 tests"]
    GF["Golden flow multi-actor (COD + online)"]
    FS["Failure scenarios F-1..F-10"]
    PWA["PWA: SW update, offline, throttled devices"]
  end
  subgraph Mid["Service / contract / integration (Go + real Postgres/PostGIS) ~400-800 tests"]
    INT["Repos, sqlc, migrations, PostGIS, River jobs, outbox, SSE"]
    CON["OpenAPI response validation, oasdiff, PA/OTP contracts"]
    AUTHZ["Generated authz matrix + IDOR"]
  end
  subgraph Unit["Unit (Go + Vitest) ~1500+ tests, seconds"]
    GU["State machines, pricing golden tables, coupons, ledger, commission, rider pay"]
    PBT["rapid property tests + fuzz"]
    FU["Components, hooks, money/i18n utils, MSW-backed screens"]
  end
  Unit --> Mid --> E2E --> Manual
  NFR["Cross-cutting: k6 load/soak, chaos-lite, ZAP + SCA, restore drills"] -.-> Mid
  NFR -.-> E2E
```

| Layer | Tool(s) | Runs where | Typical count (V1) | Budget |
|---|---|---|---|---|
| Go unit | `testing`, testify/require, go-cmp, golden files | every PR | 1,000+ | < 90 s |
| Property / fuzz | `pgregory.net/rapid`, `go test -fuzz` | PR (100 checks default), nightly (10,000 checks; fuzz 5 min/target) | 30–50 properties, 8–12 fuzz targets | PR < 60 s |
| Go integration | testcontainers-go (PostGIS image), pgx, sqlc, goose, `rivertest` | every PR (sharded ×2) | 400–800 | < 8 min |
| Contract | kin-openapi `openapi3filter`, oasdiff, codegen drift, recorded webhook fixtures | every PR; PA/OTP sandbox nightly | – | < 2 min |
| Authz matrix | generated Go tests | every PR | (ops × 9 principals) ≈ 1,000+ cases | < 2 min |
| Frontend unit/component | Vitest, Testing Library, MSW, vitest-axe | every PR (affected apps) | 500+ | < 4 min |
| E2E | Playwright | PR: smoke (golden COD + online); main/nightly: full | 40–60 | PR < 10 min; nightly < 40 min |
| Visual | Playwright screenshots | nightly | ~60 snapshots (30 screens × 2 locales) | < 10 min |
| Load | k6 (+ xk6-sse) | weekly + pre-release on staging/prod-shape | 7 scenarios | 15 min – 8 h |
| Chaos-lite | compose + Toxiproxy | nightly subset; pre-release full | 8 experiments | < 30 min |
| Security | ZAP baseline/API scan, govulncheck, osv-scanner, Trivy, gitleaks, CodeQL | PR (SCA, secrets), main (ZAP baseline), weekly (ZAP API scan) | – | < 10 min |

---

## 4. Go unit tests

### 4.1 Conventions

- **Location:** `_test.go` next to the code, same package for white-box tests of pure functions; `package x_test` for API-level tests of a module's public surface. Shared helpers in `internal/testutil` (clock, ID generator, golden-file helper, `require` wrappers for money). Test fixtures in `testdata/` per package. `[OPEN: 26 confirms layout]`
- **Table-driven** with named cases (`name` field, `t.Run(tc.name, …)`), `t.Parallel()` wherever there is no shared mutable state.
- **Pure core, thin shell:** state transitions, pricing, coupon evaluation, commission, rider pay, journal construction are **pure functions** taking explicit inputs (including `now time.Time` and config snapshots) and returning values + domain events. They do not touch the DB, the clock or random sources. This is a design requirement on backend (see §20) because it makes T1 logic unit-testable in microseconds.
- **Money helpers:** a `money.Paise` type (int64) with no float conversion; tests use `require.Equal(t, money.Paise(49280), got)` — never compare formatted strings for money.
- **Naming:** `TestQuote_DeliveryFeeSlabs`, `TestOrderTransition_AcceptFromPlaced`. Requirement tags in a comment (`// REQ: CUS-031, PAY-004`) — parsed by the traceability script (§23).
- **No sleeps** in unit tests. Ever. Time is injected.
- **Golden files:** `testdata/*.golden.yaml` (human-reviewable) with a `-update` flag (`go test ./internal/pricing -run Golden -update`). CI fails if `-update` would change anything (`git diff --exit-code`). Changes to golden files require a Finance-ops/Product reviewer in CODEOWNERS for `**/pricing/testdata/**` and `**/ledger/testdata/**`.

### 4.2 Order and delivery state machines (T1)

Doc 13 owns the transition tables. Tests consume them **as data**:

1. The transition table is defined once in Go as a declarative table (`[]Transition{From, Event, Guard, To, Actor, SideEffects}`).
2. **Exhaustive matrix test:** for every `(state ∈ all order states) × (event ∈ all events) × (actor ∈ roles+system)` assert either the documented target state or a typed `ErrIllegalTransition`. With 11 order states × ~15 events × 9 actors ≈ 1,500 cases — generated, runs in milliseconds. Same for 9 delivery states and offer states.
3. **Doc parity test:** a test renders the Go table to Mermaid `stateDiagram-v2` and compares to a committed snapshot; doc 13 embeds the generated diagram, so code and docs cannot drift.
4. **Side-effect assertions:** each transition returns the domain events/outbox messages it must emit (e.g., `ACCEPTED` → `order.accepted` event + schedule prep reminder; `REJECTED` on a prepaid order → `refund.requested`). Tests assert the event list exactly.
5. **Cross-machine consistency** (order ↔ delivery): table of allowed combinations, e.g. `orders.status = PICKED_UP` ⇒ `deliveries.status ∈ {PICKED_UP, AT_DROP}`; `orders.status = DELIVERED` ⇔ `deliveries.status = DELIVERED`. Checked in unit tests of the coordinator and by a SQL invariant query in integration/E2E (`assertInvariants()` helper, §6.7).
6. **Guards** tested at boundaries: customer cancel windows (e.g., free cancel before `ACCEPTED`, conditional after — values from doc 13 `[OPEN]`), COD skips `PENDING_PAYMENT`, `UNDELIVERABLE` only from `PICKED_UP`/`AT_DROP`.

### 4.3 Pricing / quote engine golden tables (T1)

The quote engine is a pure function: `Quote(cart, restaurantPricing, cityPricingConfig, coupon, distance, now) → QuoteResult{lines[], taxLines[], totals}`. It must produce the same numbers the order, invoice and ledger use (single source).

**Golden table format** (`internal/pricing/testdata/quote_cases.golden.yaml`), one case per row, reviewed by a human:

```yaml
- id: Q-001
  req: [CUS-030]
  desc: "Typical lunch order, 2.6 km road distance, no coupon"
  config_ref: mbnr-default-2026-10     # frozen config snapshot in testdata
  cart:
    - { item: chicken_biryani_full, unit_paise: 18000, qty: 2 }
    - { item: mirchi_bajji,        unit_paise:  6000, qty: 1 }
  packaging_paise: 1000                # per order, set by restaurant
  straight_line_m: 2000                # × road_factor 1.3 = 2600 m
  expect:
    food_subtotal_paise:     42000
    packaging_paise:          1000
    delivery_fee_paise:       3000     # slab 2–4 km
    platform_fee_paise:        500
    small_cart_fee_paise:        0     # subtotal >= 14900
    tax_lines:                         # RATES ARE PLACEHOLDERS [LEGAL]
      - { base: food+packaging, rate_bp: 500,  amount_paise: 2150 }
      - { base: platform_fee,   rate_bp: 1800, amount_paise:   90 }
      - { base: delivery_fee,   rate_bp: 1800, amount_paise:  540 }
    total_paise:             49280
```

> The GST rates and which party/section each line falls under are **placeholders** pending doc 14 and tax review `[LEGAL]`. The golden file references a config snapshot, so when rates are confirmed only the config and expected values change — the test structure does not.

**Mandatory case families** (each family = several rows):

| Family | Cases (examples) |
|---|---|
| Delivery-fee slabs | road distance 0, 1,999, 2,000, 2,001, 3,999, 4,000, 6,000, 7,999, 8,000 m and > max; **boundary inclusivity pinned** (`[OPEN: 16/14]` — is 2.000 km slab 1 or 2?) |
| Max radius | straight-line 6,999 / 7,000 / 7,001 m (serviceable / boundary / reject) — and interaction with road factor (see Challenge CH-3) |
| Small-cart fee | subtotal 14,899 / 14,900 / 14,901 paise; after coupon discount (is threshold pre- or post-discount? `[OPEN: 01]`) |
| Rounding | items with GST producing x.5 paise (e.g., base 1,010 paise × 5 % = 50.5 → 51 half-up), many lines each rounding up (sum-of-rounded vs rounded-sum difference pinned), quantity 1 vs 99 |
| Coupons | flat, percent with cap, min-order, first-order-only, restaurant-funded vs platform-funded, expired (evaluated in `Asia/Kolkata` at 23:59:59 / 00:00:00 IST), usage-limit reached, coupon making total negative (must clamp at 0, delivery fee handling) |
| Free delivery campaign | threshold just below/at/above |
| Packaging | per item vs per order, zero |
| Variants & add-ons | half/full biryani, add-on with price 0, add-on quantity, removed item |
| Config effective dates | commission/pricing config change at midnight IST — order quoted 23:59:59 uses old, 00:00:00 uses new |
| Re-quote | quote at cart time vs order create — price change in between must surface a "price changed" error, not silently charge |

### 4.4 Commission, rider pay, payouts (T1)

- **Commission:** golden rows for 10 %, 15 %, 25 %, effective-date switch mid-week, base = food subtotal only (excludes packaging? `[OPEN: 14]`), GST on commission `[LEGAL]`, rounding per order (not per payout) — and a test proving payout = Σ per-order nets exactly.
- **Rider pay:** `₹25 base + ₹6/km beyond 2 km + waiting pay after 10 min`: boundaries at 2,000 m, fractional km (per-metre pro-rata vs per-started-km `[OPEN: 06/14]`), waiting 9:59 / 10:00 / 10:01 min, cancelled-after-pickup pay rules.
- **Payout run:** weekly window boundaries in IST (Mon 00:00 IST), orders delivered at 23:59:59 Sunday IST included in the right week, adjustments/clawbacks, negative balance carried forward, minimum payout amount.

### 4.5 Ledger (T1)

Doc 14 owns the chart of accounts; tests assert structure, not account names:

- **Every journal balances**: `Σ debits == Σ credits` per journal entry, per currency — unit-tested on every journal builder function (and property-tested, §5).
- **Journal per business event** (golden): `order.paid_online`, `order.cod_collected`, `order.delivered` (revenue recognition), `order.refunded` (full/partial), `commission.accrued`, `rider_pay.accrued`, `cod.deposit_recorded`, `payout.recorded`, `pa_fee.settled`, `adjustment.manual` (admin, with reason + maker-checker `[OPEN: 07/14]`).
- **Immutability:** no update/delete of posted entries; corrections are reversing entries — tested in integration (DB-level trigger/permissions `[OPEN: 10]`).
- Illustrative golden journal for Q-001 (online payment) — account names provisional:

| Entry | Dr (paise) | Cr (paise) |
|---|---|---|
| PA clearing | 49,280 | |
| Restaurant payable (food + packaging) | | 43,000 |
| GST payable – food (e-commerce operator, §9(5)) `[LEGAL]` | | 2,150 |
| Platform-fee revenue | | 500 |
| GST payable – platform fee `[LEGAL]` | | 90 |
| Delivery-fee revenue | | 3,000 |
| GST payable – delivery fee `[LEGAL]` | | 540 |
| **Σ** | **49,280** | **49,280** |
| Restaurant payable (commission 15 % of 42,000 + 18 % GST) | 7,434 | |
| Commission revenue | | 6,300 |
| GST payable – commission `[LEGAL]` | | 1,134 |
| Rider cost (₹25 + ₹6 × 0.6 km) | 2,860 | |
| Rider payable | | 2,860 |

→ Restaurant net for this order = 43,000 − 7,434 = **35,566 paise**; this exact figure is asserted again in E2E (§10.3).

### 4.6 Other T1/T2 unit areas

| Area | Notes |
|---|---|
| Dispatch ranking | Pure `RankRiders(candidates, restaurant, now, cfg)`: online, location freshness (stale > N s excluded), distance, active jobs, COD-limit exclusion, previously-declined/expired riders excluded for this delivery; deterministic tie-break (by rider ID) so tests are stable. |
| Serviceability decision (pure part) | Given distances and zone membership flags → decision + reason codes (shown to user in en/te). |
| Operating hours | Local wall-clock + day-of-week in `Asia/Kolkata`: open/close boundaries, overnight hours (18:00–02:00), holiday override, "closing in 15 min" cutoff. |
| Phone/OTP | E.164 normalisation (`9876543210`, `+91 98765 43210`, `09876543210`), reject numbers not starting 6–9; OTP hashing, expiry, attempt counter. |
| Order code | `RV-` + base32, no ambiguous chars, collision handling with seeded generator. |
| Idempotency keys | Same key + same body → same response; same key + different body → 409/422. |
| Redaction | Log redaction helper masks phones (`+91******3210`), OTPs, tokens, addresses — table tests (DPDP). |
| i18n server strings | SMS/push template rendering in `en`/`te`, placeholders filled, Telugu SMS length/segment count (Unicode → 70 chars/segment) asserted for DLT templates `[OPEN: 15]`. |

---

## 5. Property-based tests and fuzzing

### 5.1 Invariants (rapid)

Generators produce random carts (1–30 lines, unit price 0–₹5,000, qty 1–20, random add-ons), random configs within allowed ranges (fee slabs, commission 10–25 %, coupon shapes), random event sequences (pay, accept, reject, cancel at any point, partial refunds, duplicate webhooks).

| ID | Invariant | Module |
|---|---|---|
| INV-P1 | `total = Σ lines + Σ fees + Σ taxes − discounts`, computed independently by a naïve reference implementation (oracle) in the test, using `math/big` | pricing |
| INV-P2 | `total ≥ 0`; no line or tax is negative; discount ≤ discountable base | pricing |
| INV-P3 | Monotonicity: adding an item never decreases food subtotal; increasing distance never decreases delivery fee (within the slab table) | pricing |
| INV-P4 | Quote is deterministic: same inputs → identical result (run twice, compare deeply) | pricing |
| INV-P5 | Rounding drift bounded: `|Σ rounded lines − round(Σ exact lines)| ≤ number_of_lines × 0.5 paise` and matches the documented rule | pricing |
| INV-L1 | **Every journal balances** (Σ Dr = Σ Cr) for any sequence of events | ledger |
| INV-L2 | **Σ customer bill = Σ ledger credits** for that order's revenue/payable/tax accounts at order-paid time (before commission/rider entries) | ledger + pricing |
| INV-L3 | Σ of all account balances across the whole ledger = 0 after any sequence (trial balance) | ledger |
| INV-L4 | **Refund never exceeds captured amount**: for any sequence of partial/full refunds and duplicate refund webhooks, Σ refunded ≤ Σ captured, and COD orders never produce a PA refund journal | payments + ledger |
| INV-L5 | Duplicate webhook/event delivery is idempotent: applying event list `E` and `E` with random duplicates/reordering yields identical ledger and order state | payments |
| INV-L6 | Payout for a party over a period = Σ that party's payable movements in the period (no order counted twice across weeks) | payouts |
| INV-L7 | Rider COD cash-in-hand = Σ COD collected − Σ deposits ≥ 0; rider is blocked iff cash-in-hand ≥ limit | ledger + dispatch |
| INV-S1 | **Model-based state machine** (`rapid.StateMachine`): random commands (place, pay-webhook, accept, reject, ready, offer, offer-expire, rider-accept, pickup, deliver, cancel by each actor) applied to the real aggregate and to a simplified model; after each step the canonical status pair is in the allowed set, terminal states are absorbing, and at most one rider is `ASSIGNED` per delivery | state |
| INV-D1 | Dispatch ranking never selects an offline, stale-location, already-busy, COD-blocked or previously-expired rider | dispatch |

Run with default 100 checks on PR; nightly with `-rapid.checks=10000`; failing seeds are committed as regression cases (rapid prints a reproducible seed; we copy the minimised failure into a table test).

### 5.2 Fuzz targets (Go native, nightly, 5 min each)

Webhook payload parser + signature verifier; phone normaliser; address/PIN validator; pagination cursor decoder; order code parser; coupon code normaliser; i18n template renderer (no panics, no unescaped HTML); JWT/refresh-token parser. Corpus seeds from recorded fixtures; crashers become unit tests.

---

## 6. Go integration tests

### 6.1 Database provisioning and isolation

**Decision:** `testcontainers-go` with the Postgres module, using the **same `postgis/postgis` image tag as production** (`[OPEN: 22]` pins PG 17+/PostGIS 3.x). The harness honours `ROVO_TEST_DATABASE_URL`: if set (e.g., a CI service container or a developer's local DB), it is used instead of starting a container.

```mermaid
sequenceDiagram
  participant M as TestMain (per package)
  participant C as Postgres+PostGIS container (shared per CI job)
  participant T as Each test
  M->>C: start once (or reuse via ROVO_TEST_DATABASE_URL)
  M->>C: CREATE DATABASE rovo_template; goose up; seed reference data (city, zones fixtures)
  M->>C: mark template (datistemplate = true), no connections
  T->>C: CREATE DATABASE t_<random> TEMPLATE rovo_template  (~tens of ms)
  T->>T: run test with its own pgx pool + River client
  T->>C: DROP DATABASE t_<random> (t.Cleanup)
```

| Strategy | Use for | Why |
|---|---|---|
| **Template DB per test (default)** | Anything that commits: outbox relay, River jobs, SSE fan-out via `LISTEN/NOTIFY`, concurrency/race tests, multi-tx flows | Real commit semantics; tests run in parallel safely |
| **Transaction-per-test (rollback)** | Read-heavy sqlc query tests, PostGIS lookup tests | Fastest; but invalid where code commits or uses `LISTEN/NOTIFY` |
| Fresh migrations from zero | Migration tests only (§15) | Verifies the migration chain itself |

- Migrations run **once per CI job** into the template; the template hash (hash of migration files) is cached so local re-runs reuse it.
- Container tuned for tests (`fsync=off`, `synchronous_commit=off`, `full_page_writes=off`) — **only** in tests.
- `testcontainers-go`'s Postgres module also offers snapshot/restore; we prefer explicit templates because they allow parallel tests. (Source: golang.testcontainers.org modules/postgres, accessed 2026-10-04.)

### 6.2 sqlc queries and repositories

- Every sqlc query has at least one integration test exercising it (a CI script lists queries from `sqlc` output and fails if a query name does not appear in any `_test.go` — cheap "query coverage").
- Constraint tests: unique active delivery per order, one `ASSIGNED` rider per delivery, one active delivery per rider (V1), `CHECK (amount_paise >= 0)`, FK cascades, `currency = 'INR'`.
- **Query-count assertions** for hot endpoints (menu fetch, order list) via a pgx tracer counting queries — e.g., menu fetch ≤ 3 queries regardless of item count (N+1 guard).
- `EXPLAIN` snapshot tests for the 10 hottest queries on a seeded `load` dataset: assert index usage (no `Seq Scan` on `orders`, `deliveries`, `menu_items` beyond small tables) — nightly.

### 6.3 Migrations

See §15.1 (up/down/up round trip, schema snapshot, destructive-change lint).

### 6.4 PostGIS serviceability with Mahabubnagar fixtures

Fixture polygons are **synthetic** test geometry near Mahabubnagar's centre (≈ 16.74° N, 77.99° E `[ASSUMPTION — fixture only; real zones come from ops in doc 16]`), committed as GeoJSON in `testdata/geo/mbnr_zones.geojson` and loaded into the template DB.

| Fixture | Purpose |
|---|---|
| `Z-CENTRAL` (active), `Z-NORTH` (active), `Z-SOUTH` (inactive) | Membership, inactive zone rejection |
| `Z-NORTH` with an interior **hole** (e.g., a lake/cantonment-style exclusion) | Point in hole = not serviceable |
| Two zones sharing an edge | Point exactly on shared boundary → deterministic single zone (`ST_Covers` vs `ST_Contains` choice pinned) `[OPEN: 16]` |
| Restaurant R1 centre, R2 near zone edge, R3 with 3 km custom radius | Radius checks |
| Customer points: inside, outside, on boundary, in hole, 6,999 m / 7,000 m / 7,001 m from R1 (computed with geodesic distance), across the anti-meridian-free but SRID-sensitive cases (4326 vs geography) | Distance and SRID correctness |
| Invalid polygon (self-intersecting) upload | Admin zone editor rejects (`ST_IsValid`) |

Assertions: serviceability result + reason code; computed straight-line distance within ±1 m of a reference haversine value; road-factor distance used by the quote equals the one stored on the order.

### 6.5 River jobs and timers (fake clock)

River supports stubbing time via `river.Config.Test.Time` (`rivertype.TimeGenerator`) and provides `rivertest` helpers (`RequireInserted[Tx]`, `RequireNotInserted[Tx]`, `RequireManyInserted[Tx]`, `NewWorker(...).Work/WorkJob`). (Sources: River `client.go` on GitHub and pkg.go.dev `rivertest`, accessed 2026-10-04.)

Test patterns:

1. **Scheduling assertions:** placing an online order inserts `restaurant_accept_timeout` with `ScheduledAt = placed_at + cfg.accept_timeout`; creating an offer inserts `delivery_offer_timeout` at `offered_at + 45 s` — asserted with `RequireInsertedTx` + `RequireInsertedOpts{ScheduledAt: …}`.
2. **Execution with fake clock:** set the app `Clock` to `offered_at + 44 s` → run the job via `rivertest.NewWorker(...).Work` → asserts **no-op** (job re-checks the deadline against the clock; idempotent). Set to `+45 s` → offer `EXPIRED`, next-best rider offered, new timeout job inserted.
3. **Late action vs timeout race:** rider accept at `+44.9 s` committed first, timeout job runs after → no-op; reverse order → accept fails with `OFFER_EXPIRED` (409). Both orders exercised deterministically via explicit sequencing, plus a randomised goroutine race test (§6.6).
4. **Cascade exhaustion:** N riders all expire → delivery returns to `UNASSIGNED`, admin alert outbox event emitted, retry backoff job scheduled (values from doc 13/06 `[OPEN]`).
5. **Retries & dead jobs:** job handler errors are retried with River's policy; poisoned jobs land in discarded state and raise an alert metric — tested with injected failures.
6. **Uniqueness:** duplicate timeout scheduling is prevented by unique job options (`rivertest.Worker` disables uniqueness by default — integration tests that rely on uniqueness use a real client instead).

### 6.6 Concurrency and race tests

- Run all Go tests with `-race` in CI.
- Dedicated tests launch K goroutines (K = 2..20) performing conflicting commands on one aggregate: two riders accept the same offer; restaurant accepts while customer cancels; duplicate payment webhooks arrive simultaneously; admin manual assign while auto-dispatch assigns. Assert exactly one winner, losers get typed conflict errors, and invariants (§6.7) hold. Repeated 50× in nightly (`-count=50`).

### 6.7 Outbox and invariants

- **Atomicity:** domain change + outbox row in the same transaction — test forces a failure after the outbox insert and asserts neither persisted.
- **Relay:** at-least-once delivery, per-aggregate ordering preserved, crash between "handled" and "marked sent" leads to redelivery, handler idempotency (dedupe by event ID) prevents double side-effects (double SMS, double ledger post).
- **Lag metric** emitted and asserted in tests (doc 24 consumes).
- **`assertInvariants(t, db)` helper**, called at the end of every integration and E2E test (via the test API in E2E), runs SQL checks: all journals balance; trial balance = 0; no order/delivery status combination outside the allowed set; no delivery with two accepted offers; no rider with > 1 active delivery; no `PENDING_PAYMENT` order older than the payment expiry without a reconciliation job scheduled; refunds ≤ captures per order. These same queries become a nightly production **reconciliation job** (doc 14/24).

### 6.8 SSE

- `httptest` server + real SSE handler + real Postgres `LISTEN/NOTIFY` (or in-process pub-sub per P6): a transition commits → event received by subscribed client within 1 s; client not authorised for that order receives nothing (authz on stream subscription).
- `Last-Event-ID` resume after disconnect returns missed events (or a "resync" signal) — per doc 11/17 design `[OPEN]`.
- Heartbeat/comment frames at the configured interval (needed to survive CDN/proxy idle timeouts; see CH-6).
- Connection cleanup: N connects/disconnects → goroutine count and DB listeners return to baseline (leak test).

---

## 7. Contract tests

### 7.1 OpenAPI as the enforced contract

| Check | How | When |
|---|---|---|
| Spec lint | `vacuum` or Spectral with a rovo ruleset: operationId required, every operation has `x-rovo-authz`, error responses use the shared problem+json schema, money fields are `integer` with `_paise` suffix, no `number` for money, enums match canonical status names | PR |
| Response/request validation | Test-only middleware wraps the real router using `kin-openapi/openapi3filter.ValidateRequest` + `ValidateResponse` (kin-openapi lists OpenAPI 3.1 support — README accessed 2026-10-04). Any response with an undocumented status code, missing required field, extra field where `additionalProperties: false`, or wrong enum value **fails the test** | All Go API integration tests + E2E stack (middleware enabled via `testhooks` build, logs + fails the request with 500 `CONTRACT_VIOLATION` so Playwright notices) |
| Breaking changes | `oasdiff breaking` (supports OpenAPI 3.0/3.1/3.2; GitHub Action `oasdiff/oasdiff-action/breaking` — oasdiff.com accessed 2026-10-04) comparing PR spec vs `main`; breaking change requires label `api-breaking` + versioning note | PR |
| Codegen drift | Regenerate Go server interfaces and TS client; `git diff --exit-code` | PR |
| TS client typecheck | `tsc --noEmit` across all three apps against the regenerated client — a renamed field breaks the frontend build in the same PR | PR |
| Generator compatibility fixture | A small "kitchen-sink" spec using every 3.1 construct we allow (nullable via type arrays, `oneOf` with discriminator, `const`, `examples`) is run through oapi-codegen, openapi-typescript and kin-openapi in CI; constructs that any tool mishandles are banned by lint (see CH-1) | PR (spec/tooling changes) |
| Examples valid | Every `example`/`examples` in the spec validates against its schema (they also feed MSW fixtures) | PR |

**Not doing:** Pact/consumer-driven contracts — single repo, single team, spec-first codegen on both sides already gives compile-time consumer checks.

### 7.2 Provider adapter architecture for testability

```
Domain ──► PaymentProvider interface ──► razorpayAdapter (real wire protocol) ──► HTTP ──► { real PA sandbox | fakepa }
                                    └──► inMemoryProvider (unit tests only)
```

- **Unit tests** use the in-memory provider.
- **Integration and E2E** use the **real adapter** talking HTTP to **`fakepa`**, a small Go service that speaks the chosen PA's wire protocol (orders, payments, refunds, webhooks with the PA's signature scheme). This exercises our serialisation, signature verification and error mapping, which an interface-level fake would skip.
- The **same contract suite** (`paymentcontract.Run(t, provider)`) runs against `fakepa` on every PR and against the **PA sandbox** nightly (secrets only on `main`/scheduled runs, never on fork PRs). If sandbox behaviour diverges from `fakepa`, the contract suite fails and `fakepa` is fixed — keeping the fake honest.

### 7.3 Payment webhook contract tests

- **Recorded fixtures:** one sanitised JSON per webhook event type we consume (payment authorised/captured/failed, refund created/processed/failed, order paid, dispute if applicable `[OPEN: 14]`), recorded from the PA sandbox, stored in `testdata/pa/<provider>/<version>/`. A re-record script refreshes them; diffs are reviewed.
- **Signature verification:** valid signature → accepted; tampered body (1 byte), wrong secret, missing header, signature over re-serialised JSON (whitespace differences), old secret during rotation window (accepted if dual-secret supported `[OPEN: 14]`) → rejected with 400/401 and **no state change**, security log event emitted.
- **Idempotency & ordering:** same event twice → one effect; `captured` before `authorized`; `failed` after `captured` (ignored + alert); refund webhook before our refund API response returns; webhook for unknown order (logged, 200 to stop retries, alert).
- **Amount/currency mismatch:** webhook amount ≠ order total → order not marked paid, flagged for finance review.
- **Reconciliation poll:** when webhook never arrives, the reconciliation job queries the PA (fake) and converges to the same state.
- **Replay:** captured real-shaped webhook replayed after N minutes → idempotent no-op; if the PA provides timestamps, outside-tolerance replays rejected (§14.5).

### 7.4 OTP / SMS / push provider contracts

Same pattern: real adapter → `fakeotp`/`fakesms` (wire-level) in PR, provider sandbox/test mode nightly where one exists `[OPEN: 15]`. Tests cover DLT template ID mapping per language, provider error codes → retry/fallback decisions, delivery-report webhooks, and the fallback path (secondary provider or voice/WhatsApp OTP `[OPEN: 15]`).

---

## 8. Authorisation tests

### 8.1 Generated role × operation matrix

**Proposal (needs doc 11/12 agreement):** every OpenAPI operation declares its policy in an extension:

```yaml
x-rovo-authz:
  roles: [CUSTOMER]                  # who may call it at all
  ownership: order.customer_id       # resource-ownership rule id (doc 12)
  scope: none                        # or city / restaurant / rider
```

A Go test generator reads the spec and produces, for **every operation × every principal** (`anonymous`, 8 roles, plus `ADMIN_OPS` scoped to another city), a case:

| Principal kind | Expected |
|---|---|
| anonymous on non-public op | 401 |
| role not in `roles` | 403 (or 404 where doc 12 chooses to hide existence — pinned per op) |
| role in `roles`, owns resource | 2xx (with valid fixture request from spec examples) |
| role in `roles`, does not own resource | 404/403 per doc 12 |
| admin scoped to city B on city-A resource | 403/404 |

- Fixtures: a seeded "authz world" with two cities (the second is a test-only city `TESTCITY` to prove scoping; this also validates multi-city readiness), two restaurants per city each with owner + staff, two riders, two customers, orders in each state.
- **Completeness gates:** test fails if an operation lacks `x-rovo-authz`, or if the generator cannot build a valid request (forces examples in the spec).
- The matrix is also emitted as a Markdown/CSV artifact for security review (doc 19).

### 8.2 IDOR suites per resource (hand-written, T1)

| Resource | Attack | Expected |
|---|---|---|
| Order | Customer A `GET/PATCH/cancel` order of customer B (UUID and human code `RV-…`) | 404, no data in body/timing-insensitive message |
| Address | Customer A reads/edits/uses B's address id in checkout | 404 / 422 |
| Payment | A initiates payment for B's order; A requests refund on B's order | 404 |
| Restaurant inbox | Staff of outlet R1 accepts/rejects/reads orders of R2 (same owner and different owner) | 403/404 |
| Menu | Owner of R1 edits R2's menu items, uploads image to R2 | 403 |
| Delivery | Rider X accepts offer addressed to rider Y; marks Y's delivery picked/delivered; reads customer phone after delivery completed (masking window `[OPEN: 12]`) | 403/404 |
| Rider earnings & COD | Rider X reads Y's earnings/cash-in-hand | 404 |
| Payout | Restaurant owner reads another restaurant's payout statement | 404 |
| Admin | `ADMIN_SUPPORT` performs finance-only actions (manual adjustment, payout record); `ADMIN_OPS` of city A edits zone in city B | 403 |
| SSE streams | Subscribe to another order's/restaurant's/rider's stream | 403, no events |
| Files | Fetch invoice/proof URLs of another order (signed URL expiry, path guessing) | 403/expired |
| Mass assignment | Customer PATCH sets `status`, `total_paise`, `restaurant_id`; rider PATCH sets `pay_paise` | ignored or 422 (spec `readOnly` enforced) |

### 8.3 Session and token tests

Expired access JWT → 401 + refresh succeeds; refresh-token rotation and **reuse detection** (old refresh token reused → whole family revoked); logout revokes; admin login without TOTP blocked; TOTP replay within window rejected; OTP attempt limit + lockout; cookie flags (`HttpOnly`, `Secure`, `SameSite`) asserted; CSRF defence for cookie-authenticated mutating endpoints (doc 12 `[OPEN]`).

---

## 9. Frontend tests

### 9.1 Unit / component (Vitest + Testing Library)

- Environment: `jsdom` (or `happy-dom` if faster and compatible — measure `[OPEN: 17]`). `@testing-library/react` + `@testing-library/user-event`; query by role/label first (doubles as an a11y check), `data-testid` only for non-semantic hooks (§20.2).
- What to test: cart logic and money display (shared package, `Intl.NumberFormat('en-IN'|'te-IN', {style:'currency', currency:'INR'})` with lakh grouping — **display only**; amounts arrive in paise from the API, the UI never recomputes totals), form validation (phone, PIN, address), checkout state handling (price-changed, unserviceable, payment pending), restaurant inbox sound/notification toggle logic, rider offer countdown component (fake timers via `vi.useFakeTimers()`), order status timeline mapping canonical statuses → en/te labels (exhaustive over the enum — a new status without a label fails).
- TanStack Query hooks tested with a fresh `QueryClient` per test.
- **Router-level screen tests** render a route with MSW-backed API to test loading/error/empty states.

### 9.2 API mocking (MSW)

- MSW handlers are **typed from the generated OpenAPI types** (e.g., `openapi-msw` over `openapi-typescript` paths `[VERIFY at implementation]`), so a handler returning a shape that is not in the spec fails `tsc`.
- Response fixtures come from **shared TS factories** (§18.3) and from spec `examples`.
- `@mswjs/source`'s `fromOpenApi()` can generate handlers at runtime from a spec; its docs state OpenAPI 2.0 and 3.0 support (source.mswjs.io, accessed 2026-10-04) — **3.1 support unconfirmed**, so it is optional for dev-mode mocking only `[OPEN]`.
- The same handlers power **dev mode without a backend** (`pnpm dev:mock`) for UI work and for Storybook-less component review.

### 9.3 i18n completeness

| Check | Tool / rule | Gate |
|---|---|---|
| Key parity | Script compares `en` and `te` catalogs per app + shared: missing keys, extra keys, empty strings, untranslated (identical to `en` unless allow-listed, e.g., "UPI", "OTP") | PR — blocking for `missing`/`empty`; untranslated allowed only with `te-pending` allow-list that must be empty at release |
| Placeholder parity | Interpolation variables and ICU plural/select branches identical across locales; ICU syntax valid | PR |
| No hard-coded strings | ESLint rule (`i18next/no-literal-string` or equivalent) on JSX text | PR |
| Pseudo-localisation | Build-time `en-XA` locale: accents + brackets + ~40 % expansion; Playwright runs the golden flow screens in `en-XA` and screenshots; layout overflow detection script flags elements whose `scrollWidth > clientWidth` | nightly |
| Telugu render | Visual snapshots in `te` (§9.5) with Noto Sans Telugu (or chosen font) loaded; font subsetting doesn't drop conjuncts (render test string set incl. ర్ర, క్ష, శ్రీ) | nightly |
| Server-side strings | SMS/push templates per language parity (Go test) | PR |

### 9.4 Accessibility (automated part)

- Component level: `vitest-axe` assertions on every shared component and key screens.
- E2E level: `@axe-core/playwright` `AxeBuilder` on each main screen of each app, tags `wcag2a, wcag2aa, wcag21a, wcag21aa` (+ `wcag22aa` where axe supports) — **0 serious/critical violations** to pass (Playwright docs, accessed 2026-10-04). Automated checks catch only part of WCAG; manual audit in §16.2.
- Specific checks: touch targets ≥ 48 CSS px on partner app (riders with gloves/helmets), colour contrast of status chips, focus order in checkout, live-region announcement for order status updates and new-order alert.

### 9.5 Visual regression (decision)

- **Tool:** Playwright `expect(page).toHaveScreenshot()` — free, no SaaS. Rendered inside the pinned `mcr.microsoft.com/playwright` image so fonts/antialiasing are deterministic; dynamic regions (timestamps, maps) masked.
- **Scope:** ~30 high-value screens × `en` + `te` at 360×800 (entry Android viewport) and admin at 1366×768. Baselines committed with Git LFS or plain PNG (`[OPEN: 26]` — size estimate ~6 MB).
- **Gate:** nightly, non-blocking until M2 `[OPEN: 28]`, then blocking on `main`. Baseline updates via a `update-snapshots` workflow triggered by PR label, reviewed by a human.

---

## 10. End-to-end tests

### 10.1 Stack and conventions

- **Playwright Test** (TypeScript) in `e2e/` (`[OPEN: 26]`), run against `compose.e2e.yml` (planning reference; DevOps owns): `api` + `worker` built with `testhooks` tag, PostGIS, Caddy (same reverse-proxy config as prod so SSE buffering/headers are realistic), three PWAs as **production builds** served statically, `fakepa`, `fakeotp`/`fakesms`, `fakepush`, Toxiproxy (idle unless a chaos test arms it), local map tiles stub (no external tile requests in CI).
- Each spec starts from `POST /_test/reset?profile=e2e` (truncate + reseed deterministic data, ~1–2 s) or uses **unique per-test entities** created by factory endpoints so specs can run in parallel workers without reset `[decide in Phase 2 by measurement]`.
- Logins use the OTP sink: UI enters phone → test reads OTP via `GET /_test/otp?phone=…` → UI enters it. Admin TOTP uses a seeded TOTP secret and computes the code in the test (`otpauth` lib).
- Selectors: role/label first, `data-testid` for dynamic items (order cards, offer card) — §20.2.
- **Every spec ends with** `GET /_test/invariants` (runs §6.7 checks) and fails on any violation; **contract-validation middleware** is on, so any spec-violating response fails the run.
- Artifacts on failure: Playwright trace, video, HAR, API/worker logs, `fakepa` webhook log, DB dump of the affected order.
- Retries: `retries: 1` on CI only; a pass-on-retry is reported as **flaky** (§21.4), never silently green.

### 10.2 Time control in E2E

The golden flow needs a 45 s offer timeout, restaurant accept timeout, cancel windows and weekly payouts without waiting.

- **Primary:** a controllable clock in `testhooks` builds: `POST /_test/clock {advance: "46s"}` updates a shared offset (single-row table read by the `Clock` implementation in both `api` and `worker`), then `POST /_test/jobs/run-due` asks the worker to run River jobs whose `scheduled_at ≤ fake now`. Whether River's job fetcher honours the stubbed `Config.Test.Time` for scheduling, or whether `run-due` must re-schedule due jobs to "now", is a **Phase 2 spike** `[OPEN]`.
- **Fallback:** E2E profile config with short real timeouts (`offer_timeout=5s`, `accept_timeout=10s`) — same code path, real waiting, slower and slightly flakier; used only if the spike fails.

### 10.3 Golden flow (multi-actor, 4 browser contexts)

```mermaid
sequenceDiagram
  autonumber
  participant C as Customer ctx (customer app, 360x800)
  participant R as Restaurant ctx (partner app, staff, 800x1280)
  participant D as Rider ctx (partner app, rider, 360x800, geolocation mocked)
  participant A as Admin ctx (admin app, 1366x768, ADMIN_OPS + ADMIN_FINANCE)
  participant P as fakepa / test API
  C->>C: OTP login (sink), pick locality + pin in Z-CENTRAL
  C->>C: browse restaurant R1 (te locale for half the runs), add variants/add-ons
  C->>C: checkout: quote shows Q-001 numbers exactly
  alt Online
    C->>P: pay on fakepa checkout page (Success)
    P-->>C: webhook payment.captured → order PLACED
  else COD
    C->>C: order PLACED directly
  end
  R-->>R: SSE: new order alert in inbox < 2 s
  R->>R: Accept (prep time 20 min) → ACCEPTED → PREPARING
  R->>R: Mark ready → READY_FOR_PICKUP (dispatch may start earlier per doc 13)
  D-->>D: SSE: offer card with countdown
  D->>D: Accept offer → delivery ASSIGNED
  D->>D: Arrived → AT_RESTAURANT; Picked up (order code check) → PICKED_UP
  C-->>C: status timeline updates via SSE
  D->>D: Arrived at drop → AT_DROP; Delivered (COD: confirm cash ₹492.80) → DELIVERED
  C->>C: Rate restaurant + rider
  A->>A: Order detail shows full timeline + audit log
  A->>P: GET /_test/ledger?order=… → assert journals (§4.5)
  A->>P: clock → next Monday 00:00 IST; run payout job
  A->>A: Payout run lists R1 net 35,566 paise, rider 2,860 paise; mark paid with UTR ref
  A->>P: GET /_test/invariants → all pass
```

**Assertions per step** (excerpt — full list lives with the spec):

| Step | UI assertion | Backend assertion (test API) |
|---|---|---|
| Checkout | Bill lines and total match golden Q-001 in `en` and `te` (₹492.80 / te-IN formatting) | Order `total_paise = 49280`, quote id linked |
| Payment (online) | "Payment successful" screen; no double order on double-click | One PA order, one capture, order `PLACED` |
| Restaurant accept | Inbox card moves to "Preparing"; alert sound element triggered (asserted via `data-alert-played` flag) | `ACCEPTED` → `PREPARING`; accept-timeout job cancelled |
| Offer | Countdown visible, starts ≤ 45 s | One `PENDING` offer, timeout job scheduled |
| Pickup | Rider cannot mark picked up before `AT_RESTAURANT` | `orders.status = PICKED_UP`, `deliveries.status = PICKED_UP` |
| Delivered (COD) | Cash-to-collect amount equals order total | Rider cash-in-hand += 49,280 |
| Rating | Rating saved; second rating blocked | One rating per order per target |
| Ledger | – | Journals of §4.5 posted; INV-L1..L3 hold |
| Payout | Statement totals | Payout journal; payable balances → 0 for the week |

Runs: COD and online variants; locale `en` and `te` variants alternate; one variant on WebKit (iOS Safari proxy) as smoke.

### 10.4 Failure scenarios

| ID | Scenario | Trigger (test control) | Expected outcome (states + money) |
|---|---|---|---|
| F-1 | **Restaurant rejects** prepaid order | Restaurant clicks Reject with reason | `REJECTED`; refund requested via fakepa; `refund.processed` webhook → customer sees "Refund initiated/processed"; ledger reversing entries; INV-L4 |
| F-2 | **Restaurant accept timeout** | Advance clock past accept timeout; run due jobs | Order auto-`CANCELLED` (`cancelled_by=system`, reason `RESTAURANT_TIMEOUT`) **or** escalated to admin first per doc 13 `[OPEN]`; prepaid refund; restaurant flagged |
| F-3 | **No rider available** | No riders online (or all decline/expire) | Delivery `UNASSIGNED` after cascade; admin dashboard alert; **admin manual assign** to a rider → `ASSIGNED`; rider sees assignment |
| F-4a | **Payment webhook delayed** | `fakepa` holds webhook 2 min | Customer sees "confirming payment"; order stays `PENDING_PAYMENT`; reconciliation poll or late webhook → `PLACED` exactly once |
| F-4b | **Webhook duplicated** (×3, concurrent) | `fakepa` duplicate mode | One state change, one journal |
| F-4c | **Webhook out of order** (`failed` after `captured`, `captured` before `authorized`) | `fakepa` reorder mode | Final state per doc 14 rules; alert on contradictory events; no ledger double-post |
| F-4d | **Payment fails then retried** | fakepa "Fail" then "Success" | `PAYMENT_FAILED` → retry path (new attempt on same order or new order per doc 14 `[OPEN]`); never two captures |
| F-4e | **Webhook never arrives, customer closed browser** | fakepa drop mode | Reconciliation job resolves within configured window; or expiry → `CANCELLED`/`PAYMENT_FAILED` + no charge |
| F-5 | **Customer cancel windows** | Cancel at `PLACED`, `ACCEPTED`, `PREPARING`, `PICKED_UP` | Allowed/blocked and refund amount per doc 13 policy; UI shows correct en/te explanation |
| F-6 | **Admin cancels** after pickup | Admin action with reason | `CANCELLED` + rider pay rules + refund; audit log entry |
| F-7 | **COD cash limit block** | Rider cash-in-hand set to ₹1,950; deliver a ₹100 COD order | After delivery rider is blocked from COD offers (still eligible for prepaid `[OPEN: 06]`); admin records deposit → unblocked |
| F-8 | **Offer timeout cascade** | Rider 1 ignores (advance 46 s), rider 2 declines, rider 3 accepts | Offers `EXPIRED`, `DECLINED`, `ACCEPTED`; rider 1's late accept → "offer expired" UI; one assignment |
| F-9 | **Unserviceable address** | Pin in `Z-SOUTH` (inactive) / hole / 7.1 km | Checkout blocked with reason; no order row |
| F-10 | **Restaurant closes during cart** | Admin/owner toggles closed; or clock passes closing time | Checkout blocked; cart preserved |
| F-11 | **Price change between quote and order** | Owner edits price after quote | Order create returns price-changed; UI re-quotes; customer must confirm |
| F-12 | **Undeliverable** | Rider marks customer unreachable at drop | `UNDELIVERABLE`; COD: no cash, rider pay per policy; prepaid: refund policy per doc 13 `[OPEN]` |
| F-13 | **SSE loss** | Toxiproxy cuts SSE for 60 s | UI falls back to polling and shows current state; no missed new-order alert for restaurant |
| F-14 | **Double submit / back button** | Double-click "Place order", browser back after payment | Idempotency key → one order |

---

## 11. Mobile / PWA tests

### 11.1 Emulated low-end device profiles (Playwright + Chromium CDP)

| Profile | Viewport / DPR | CPU throttle | Network | Use |
|---|---|---|---|---|
| `entry-android` | 360×800 @2 | 4× (`Emulation.setCPUThrottlingRate`) | "Slow 4G" ≈ 400 ms RTT, 1.6 Mbps down, 750 kbps up (`Network.emulateNetworkConditions`) `[ASSUMPTION: align with 18]` | customer golden flow, partner rider flow |
| `very-low-end` | 360×720 @1.5 | 6× | 3G-like ≈ 300 kbps down, 400 ms RTT | smoke: app shell + menu + checkout |
| `restaurant-tablet` | 800×1280 @1.5 | 4× | Slow 4G | partner inbox long-running (SSE soak in browser 2 h, memory growth check via `performance.memory` where available) |

Throttling is Chromium-only (CDP session via `page.context().newCDPSession(page)`); WebKit runs unthrottled.

**Budgets** (Lighthouse CI mobile preset on built PWAs, nightly; values owned by doc 18 `[OPEN]`): initial JS of customer app ≤ ~170 KB gzip `[ASSUMPTION]`, LCP ≤ 2.5 s (Lighthouse simulated mobile), TBT ≤ 300 ms, CLS ≤ 0.1; in Playwright `entry-android` profile: menu interactive ≤ 5 s cold, ≤ 2 s warm (SW cache).

### 11.2 Offline and flaky-network behaviour

| Case | Expected |
|---|---|
| Customer app offline at launch (after first install) | App shell loads from SW; offline banner (en/te); cached restaurant list may show with "may be outdated" |
| Offline at checkout | "Place order" disabled; **no queued payments or orders** (never background-sync money actions) |
| Connection drops after "Place order" sent | On reconnect, app queries order by idempotency key and shows real state |
| Rider offline at drop | "Delivered" action retried with idempotency key on reconnect `[OPEN: 06/18 — allowed offline queue for rider status?]`; UI clearly shows "pending sync" |
| Network switch Wi-Fi ↔ 4G (simulated by context.setOffline toggles) | SSE reconnects with `Last-Event-ID`, state consistent |

### 11.3 Service-worker update flow

Automated E2E: build v1 and v2 of an app (v2 with a visible build marker); serve v1, load and install; switch server to v2; assert the "Update available" prompt (vite-plugin-pwa `prompt` mode `[OPEN: 18]`); accept → reload → v2 marker visible; no mixed v1/v2 chunks (no `ChunkLoadError`); **an open checkout is not reloaded mid-payment** (update deferred while on payment screens). Also: old SW + API with additive change keeps working (backward-compat guarantee tested by running v1 frontend against v2 API in the E2E smoke for one release window).

### 11.4 Web Push

- Automated: notification permission granted in Playwright context; assert subscription is created and registered with the backend (`fakepush` receives subscription); backend events (new order for restaurant, new offer for rider, order out for delivery for customer) produce the right payload in `fakepush` in the right language; SW `push` handler unit-tested with a mocked `ServiceWorkerGlobalScope` (payload → `showNotification` args, click → correct deep link).
- **Not automatable reliably:** real delivery through FCM/Mozilla/Apple push services, OS-level notification display under battery saver — covered by the real-device checklist.

### 11.5 Real-device smoke checklist (pre-release, manual, ~45 min per device)

**Device classes** typical for the market `[ASSUMPTION — validate with pilot participants' actual phones; Product to collect during recruitment]`:

| Class | Example models (any one per class) | Why |
|---|---|---|
| A. Entry Android, 3–4 GB RAM, Android 12–14 | Xiaomi Redmi A-series / Redmi 12C–14C class; Realme C-series / Narzo N-series; Samsung Galaxy A0x/M1x entry | Majority of customers and riders `[ASSUMPTION]` |
| B. Android Go / 2 GB RAM | Any Android Go edition device (e.g., older Redmi A / Samsung A0x Go) | Worst-case memory/WebView |
| C. Mid Android | Samsung Galaxy M3x/A3x, Redmi Note series, Vivo/Oppo Y-series | Restaurant owners/staff phones |
| D. Budget Android tablet (8") | Lenovo/Realme/Samsung entry tablet | Restaurant counter device `[ASSUMPTION]` |
| E. iPhone (older, iOS 16.4+) | iPhone SE 2nd/3rd gen or iPhone 11 | Safari PWA install + Web Push (home-screen apps only) |

**Checklist per device:** install PWA (A2HS) in Chrome/Samsung Internet; OTP login (SMS autofill/WebOTP where supported); location permission + pin drop; browse/checkout in `te`; UPI intent flow to PhonePe/GPay/Paytm via PA sandbox; restaurant inbox alert **sound with screen locked/app backgrounded** (autoplay and background limits!) and wake-lock; rider offer accept within 45 s on Jio, Airtel and Vi networks; 4G↔Wi-Fi switch; battery saver on; low storage (< 500 MB free); dark mode; system font size "Large"; Telugu font rendering; app update prompt; push notification received when app closed; reopen after 24 h (token refresh). Results recorded in a release checklist (doc 29).

---

## 12. Performance and load tests

### 12.1 Load model `[ASSUMPTION — all numbers to be revisited after 4 weeks of pilot data]`

| Parameter | Value | Derivation |
|---|---|---|
| Orders/day (target at 6–12 months) | 2,000 | Baseline brief |
| Peak-hour share | 25 % → **500 orders/h** (≈ 0.14 orders/s) | Lunch 12:30–13:30 or dinner 20:00–21:00 IST |
| Design load | **3× peak = 1,500 orders/h**; spike 5× for 5 min | Headroom for growth/campaigns |
| Browse sessions per order | 10 (10 % conversion) → 5,000 sessions/peak h | `[ASSUMPTION]` |
| API reads per session | 25 → ~35 rps at peak, **~105 rps at design load**, 175 rps spike | |
| Concurrent SSE | customers tracking (~350) + restaurant devices (~300) + riders (~120) + admins (~10) ≈ 800 → **test 2,500** | 40-min avg order lifecycle |
| Rider location pings | 120 riders / 30 s ≈ 4 rps → test 15 rps | P12 |
| Order transitions | ~8 per order → ~1 event/s peak → test 5/s | |

### 12.2 SLOs asserted as k6 thresholds

| Endpoint class | p95 | p99 | Error rate |
|---|---|---|---|
| Read APIs (restaurant list, menu, order status) | **< 300 ms** | < 800 ms | < 0.5 % |
| Order create (excluding PA round-trip; fakepa with 0 latency) | **< 800 ms** | < 1.5 s | < 0.5 % |
| Quote | < 400 ms | < 1 s | < 0.5 % |
| State-change commands (accept, pickup…) | < 400 ms | < 1 s | < 0.5 % |
| SSE: commit → client receive | < 2 s | < 5 s | 0 dropped events (with resume) |
| Offer timeout firing | 45 s + < 5 s | 45 s + < 10 s | 0 missed |
| Outbox lag | < 2 s | < 5 s | – |

(Doc 24 owns production SLOs; these are pre-release test thresholds and must be ≤ production SLOs.)

### 12.3 k6 scenarios

| ID | Scenario | Shape | Duration | Cadence |
|---|---|---|---|---|
| K-1 | **Menu browsing read load** | ramping-arrival-rate to 105 rps, mix: list 30 %, menu 40 %, search 15 %, quote 15 % | 30 min | weekly, pre-release |
| K-2 | **Lunch/dinner peak** | orders at 1,500/h with full lifecycle driven by API (restaurant accept, rider accept, deliver) + K-1 background | 60 min | pre-release |
| K-3 | **SSE connection soak** | 2,500 concurrent streams (`xk6-sse` `[VERIFY suitability]` or Go harness) + K-2 at 1× | 2 h (8 h before launch) | weekly (2 h) |
| K-4 | **Dispatch storm** | 100 orders become `READY_FOR_PICKUP` within 5 min, 40 riders online, scripted accept/decline/ignore mix | 20 min | pre-release |
| K-5 | **Spike** | 5× read load for 5 min | 15 min | pre-release |
| K-6 | **Stress to break** | step up until SLO breach; record max sustainable orders/h | ≤ 60 min | pre-release (capacity report) |
| K-7 | **Abuse** | OTP send/verify flood from many phones/IPs; coupon brute-force | 10 min | pre-release (validates rate limits under load) |

Data: `load` seed profile (§18.2) with 150 restaurants, 6,000 menu items, 300 riders, 50,000 customers, 100,000 historical orders (to make indexes realistic).

### 12.4 Capacity test on the target free VM

- Target shape per P17/doc 25 (e.g., an Always-Free ARM VM) `[VERIFY: 25]`. Because free-tier capacity may not allow a second identical VM, the **capacity run is executed on the production VM before launch** (empty of real data), and afterwards only on staging or in announced maintenance windows (see CH-5).
- Load generator runs **off-box** (GitHub-hosted runner or developer machine; must respect the hosting provider's acceptable-use policy and Cloudflare rate limits — run against origin with an allow-listed IP or a staging hostname bypassing CDN for raw API, and once through CDN for realistic SSE/proxy behaviour).
- Report: max orders/h with SLOs met, CPU/RAM/disk IO, Postgres connections, River queue latency, outbox lag, SSE memory per connection, GC pauses. Pass criterion: design load (3× peak) sustained 60 min at < 70 % CPU.

---

## 13. Resilience / chaos-lite

Run against the compose stack (nightly subset C-1, C-3, C-4; full set pre-release) with `/_test/invariants` checked after each experiment.

| ID | Experiment | Method | Expected |
|---|---|---|---|
| C-1 | **Kill worker mid-dispatch** | `docker compose kill worker` right after an offer is created; restart after 60 s | River rescues the job; offer expires/ cascades correctly once worker is back; **no double offer or double assignment**; alert fired for worker down (doc 24) |
| C-2 | **Kill API during order create** | kill between DB commit and response | Client retries with idempotency key → same order; no duplicate |
| C-3 | **DB restart** | `docker compose restart db` during K-2 at 1× | API readiness → 503 during outage; pools reconnect; no lost outbox events; River resumes; SSE clients reconnect and resync |
| C-4 | **PA timeouts** | Toxiproxy latency 30 s / reset on fakepa | Order create returns retryable error within our timeout budget; no orphan "paid but no order"; reconciliation fixes stragglers |
| C-5 | **PA webhook endpoint unreachable for 30 min** | block inbound from fakepa | Reconciliation poll converges; fakepa retries succeed idempotently |
| C-6 | **OTP provider failure** | fakeotp returns 5xx / times out | Fallback provider or channel engaged per doc 15; user sees en/te message; rate limits not consumed by failed sends `[OPEN: 15]` |
| C-7 | **Disk nearly full / slow disk** | fill volume to 95 %; IO latency via cgroup throttle | Alerts fire; API degrades gracefully; no corrupt state |
| C-8 | **Clock jump** | advance fake clock 10 min abruptly | Timeout jobs fire in order; no negative durations; SLA metrics sane |
| C-9 | **SSE proxy idle timeout** | Caddy/CDN-like proxy with 100 s idle timeout via Toxiproxy | Heartbeats keep streams alive; reconnect logic works |

---

## 14. Security testing

Doc 19 owns the threat model; this section owns **verification**.

### 14.1 OWASP ASVS 5.0 Level 2 mapping

ASVS 5.0.0 is current (owasp.org project page; chapter list from the v5.0.0 CSV on GitHub, accessed 2026-10-04). Target: **L2 for all apps**, with a requirement-level checklist maintained by Security Architect `[OPEN: 19]`.

| ASVS 5.0 chapter | Verification in rovo |
|---|---|
| V1 Encoding & Sanitization | Fuzz targets (§5.2); React auto-escaping + lint ban on `dangerouslySetInnerHTML`; SQL only via sqlc (parameterised) |
| V2 Validation & Business Logic | OpenAPI request validation; business-logic abuse tests: negative qty, coupon stacking, price tampering, cancel/refund abuse, COD limit bypass (§10.4) |
| V3 Web Frontend Security | CSP/HSTS/frame-ancestors/`X-Content-Type-Options` header tests on Caddy + CDN; ZAP baseline |
| V4 API & Web Service | Contract validation, mass-assignment tests (§8.2), HTTP method tests, rate limits |
| V5 File Handling | Menu image upload: type sniffing, size limits, EXIF stripping, no SVG script, storage path traversal |
| V6 Authentication | OTP brute force & lockout, OTP expiry, admin password + TOTP, credential stuffing rate limits |
| V7 Session Management | Refresh rotation + reuse detection, logout/revocation, cookie flags, idle/absolute timeouts |
| V8 Authorization | Generated matrix + IDOR suites (§8) |
| V9 Self-contained Tokens | JWT `alg` pinning (reject `none`/HS↔RS confusion), `exp/nbf/aud/iss` checks, key rotation |
| V10 OAuth & OIDC | N/A in V1 (no OAuth login) — documented as not applicable |
| V11 Cryptography | OTP/refresh-token hashing, TOTP secret encryption at rest, no custom crypto (code review checklist) |
| V12 Secure Communication | TLS config scan (e.g., `testssl.sh` on staging, pre-release) |
| V13 Configuration | `/_test/*` absent from prod image (test), debug endpoints off, secrets not in images (gitleaks + Trivy secret scan) |
| V14 Data Protection | Log redaction tests; PII fields encrypted/masked per doc 12/19; data export/deletion (DPDP) flows tested |
| V15 Secure Coding & Architecture | govulncheck, osv-scanner, Trivy, CodeQL, dependency pinning |
| V16 Security Logging & Error Handling | Security events (auth failures, authz denials, webhook signature failures) emitted — asserted in tests; no stack traces in responses |
| V17 WebRTC | N/A |

### 14.2 Automated scanning in CI

| Tool | Scope | Cadence | Gate |
|---|---|---|---|
| `govulncheck` | Go modules (reachable vulns) | PR | Block on reachable vuln with fix available |
| `osv-scanner` | Go + pnpm lockfiles | PR | Block on High/Critical with fix; waiver file with expiry |
| CodeQL (Go + JS/TS) | SAST | PR + weekly | Block on High |
| `gitleaks` | Secrets in diff/history | PR | Block |
| Trivy | Built images (OS + language pkgs), IaC/compose misconfig | main + nightly | Block release on Critical/High with fix |
| **ZAP baseline** (passive; spider default 1 min; exit 1 on FAIL, 2 on WARN — zaproxy.org docs accessed 2026-10-04) | Each PWA + API via Caddy on the ephemeral E2E stack, authenticated context for customer app | every `main` merge | Rules file: selected rules = FAIL (missing CSP, cookie flags, info leaks), rest WARN; FAIL blocks |
| ZAP API scan (active) using OpenAPI | API on ephemeral stack only | weekly | Findings triaged within 5 working days |
| `testssl.sh` | Staging TLS | pre-release | No High |

**Decision on ZAP:** Yes — baseline on `main` (not PR, to save minutes and because fork PRs can't run the full stack reliably), active API scan weekly. Never scan production actively. External penetration test before public launch `[OPEN: budget, 19/29]`.

### 14.3 Authorisation

§8 (generated matrix + IDOR) is the primary control; ZAP and manual review are secondary.

### 14.4 Rate-limit tests

Integration tests against the rate limiter implementation (in-memory/Postgres per P6) with the fake clock: OTP send per phone (e.g., 3/10 min), per IP/device (e.g., 10/h), OTP verify attempts (5 then lock 15 min), login (admin), coupon validation, order create per customer, webhook endpoint (no limit that would drop legit PA retries — allow-listed by signature, not IP). Assert `429` + `Retry-After`, counters reset after window, limits per city config. k6 K-7 validates under load. When a Redis-backed limiter is introduced (>1 replica), the same suite runs against it (shared contract suite for the `RateLimiter` interface).

### 14.5 Webhook replay and spoofing

Replay of a valid signed webhook (idempotent no-op), replay with modified amount (signature fail), webhook from unexpected source with valid-looking body but no signature, very large body (size limit), slowloris on webhook endpoint (server timeouts).

---

## 15. Data, migration and backup-restore testing

### 15.1 Migrations (goose)

- **Round trip** on every PR touching `migrations/`: empty DB → `up` all → schema dump A → `down` to previous release tag → `up` → schema dump B → `A == B`. (Down migrations are required for rollback rehearsal but prod rollback policy is forward-fix by default `[OPEN: 21/22]`.)
- **Schema snapshot:** `pg_dump --schema-only` (normalised) committed; CI fails on drift without a migration.
- **Destructive change lint:** flag `DROP COLUMN`, `ALTER TYPE`, non-concurrent index creation on large tables, `NOT NULL` without default on populated tables — a migration linter (e.g., Squawk `[VERIFY]`) or a custom script; requires `migration-reviewed` label.
- **Data migrations** tested with a fixture dataset containing edge rows (nulls, old enum values, Telugu text, max paise values).
- **Prod-shape rehearsal (pre-release):** apply pending migrations to a restored **anonymised** copy of the latest production backup (or the `load` dataset before launch); measure lock times and duration; must be < 30 s blocking per migration `[ASSUMPTION]`.

### 15.2 Data integrity checks

The `assertInvariants` SQL set (§6.7) doubles as the nightly reconciliation in production and as the post-restore check.

### 15.3 Backup restore verification (weekly, automated)

Scheduled workflow (doc 23 owns backup mechanics): fetch latest encrypted backup from object storage → restore into an ephemeral PostGIS container on a runner → `goose status` equals expected version → run invariants + row-count sanity vs. production metrics snapshot → verify PITR to a timestamp if WAL archiving is used `[OPEN: 23]` → record **measured restore time** (RTO evidence) → destroy. Failure pages the on-call. Backup data never leaves the runner and is never used in lower environments without anonymisation (§18.5).

---

## 16. Localisation QA and accessibility audit

### 16.1 Telugu localisation QA

- **Native-speaker review** of every `te` string before each release (paid or volunteer reviewer from Mahabubnagar/Telangana; Telangana Telugu register, not overly formal/Sanskritised) using a review kit: screenshots per screen (generated by Playwright in `te`) + string table with context notes `[OPEN: 01/17 owner and budget]`.
- **Glossary:** fixed translations for domain words (order, delivery partner, COD, refund, OTP, UPI, cart) — many stay in English/transliterated as users expect (e.g., "ఆర్డర్"); the glossary is the reference for reviewers and SMS templates.
- **Text expansion & layout:** pseudo-locale run (§9.3) + `te` visual snapshots; buttons must wrap, not truncate, for actions (Accept, Reject, Picked up).
- **Fonts:** Noto Sans Telugu (OFL) or chosen font; check conjunct rendering on device classes A/B, line-height for vowel signs (no clipping), font loading strategy (FOUT acceptable, invisible text not) and font size of `te` ≥ `en` equivalent.
- **Numbers/dates:** Western digits in `te` UI `[ASSUMPTION — confirm with users]`; `te-IN` date/time formatting; currency ₹ with Indian grouping.
- **Mixed content:** Telugu UI with English restaurant/menu names and vice versa; search works for transliterated queries? (deferred — V1 searches `name` + `name_te` substring `[OPEN: 01]`).
- **SMS in Telugu:** DLT template registration in Telugu, Unicode segment cost check (§4.6).

### 16.2 Accessibility audit (manual, pre-release)

WCAG 2.2 AA target `[OPEN: 17]`. Manual pass per app: TalkBack on Android device class A (customer checkout, rider offer accept), keyboard-only for admin, zoom 200 %, system large font, colour contrast of status colours, motion reduction, alert/notification not conveyed by sound only (restaurant inbox also flashes/vibrates). Findings tracked as bugs with severity per §22.2.

---

## 17. UAT, pilot, dogfooding, bug bash

| Phase | Who | Where | Duration | Entry criteria | Exit criteria |
|---|---|---|---|---|---|
| **Internal dogfooding** | Team + friends (10–20 people) | Staging with fakepa + 2–3 real Mahabubnagar restaurants' menus (with permission) and staff playing riders | 2 weeks | All release gates except pilot-specific; golden flow green nightly 5 days | 0 open S1/S2; ≥ 50 test orders incl. all F-scenarios manually |
| **Closed pilot** | 5–10 real restaurants, 5–8 riders, 50–100 invited customers | Production, 1–2 zones, limited hours (e.g., 11:00–22:00 IST), COD + online, caps on order value `[ASSUMPTION]` | 4 weeks | Production readiness checklist (doc 29) passed; on-call rota; support WhatsApp/phone line in Telugu | Order success rate ≥ 95 %; 0 money discrepancies in reconciliation; restaurant accept median < 3 min; dispatch: < 2 % orders needing manual assign; partner NPS/feedback reviewed |
| **Beta cohort** | Open sign-up, capacity-capped, wider zones | Production | 4–8 weeks | Pilot exit + fixes | Launch go/no-go |
| **Bug bash** | Whole team + partners' staff | Staging, 2 h per milestone | Each milestone | Feature complete for milestone | Bugs triaged within 2 days |

- **Partner onboarding as test:** training sessions double as usability tests (observe restaurant staff accepting orders on their own phones; riders accepting offers while on a bike — safety: only when stopped).
- **Pilot instrumentation:** feature flag for enhanced logging, daily reconciliation report reviewed by finance-ops, daily standup with ops on stuck orders.
- **Feedback channels:** in-app "Report a problem" (attaches order ID + app version), WhatsApp group per partner type, call-back for S1 issues. Telugu-speaking support is mandatory.
- **UAT scripts** derived from doc 04–07 journeys; each script references requirement IDs (§23).
- **Legal/consent:** pilot participants informed of pilot status; DPDP consent flows exercised for real `[LEGAL]`.

