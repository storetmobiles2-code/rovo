# Phase 2 — Implementation Charter (binding for all implementation agents)

Source of truth: `docs/planning/` (precedence: `00` §9/§9.1 › `00` §8 › `00` §4a › owning doc › others).
This charter fixes the **V1 implementation slice**, the toolchain, conventions, ownership and the few deliberate implementation-level deviations (each also recorded as an ADR in `docs/adr/`).

## 1. Workflow

Implement → Test → Review → Fix → Security check → Refine → Validate. Every PR-sized change keeps `make check` green (lint + unit + integration + contract tests). No feature is "done" without tests, validation, error handling, authz checks, logging, and docs.

## 2. Toolchain (pinned)

| Area | Choice |
|---|---|
| Go | `go 1.25` (toolchain auto-download), stdlib `net/http` ServeMux, `log/slog` JSON |
| DB | PostgreSQL 17 + PostGIS 3.5 (`postgis/postgis:17-3.5` locally; RDS in prod) |
| DB access | `jackc/pgx/v5` (pgxpool), **hand-written parameterised SQL** in per-module `store` packages (ADR-P2-002) |
| Migrations | `pressly/goose/v3`, SQL files embedded in the binary (`backend/migrations`), run by `rovo migrate` |
| Jobs / outbox | `riverqueue/river` + `riverpgxv5`; transactional `InsertTx` per subscriber (R22/R42) |
| Auth | `golang-jwt/jwt/v5` EdDSA (Ed25519), opaque refresh tokens (SHA-256 hashed), `golang.org/x/crypto/argon2` (argon2id), `pquerna/otp` TOTP |
| IDs | `google/uuid` v7; order codes `RV-XXXXXX` (Crockford base32) |
| Contract | OpenAPI 3.1 at `openapi/openapi.yaml` (source of truth); Go responses validated against it in tests with `getkin/kin-openapi`; TS types generated with `openapi-typescript` → `web/packages/api-client` (ADR-P2-001) |
| Observability | OTel tracing (`otelhttp`) with OTLP exporter when `OTEL_EXPORTER_OTLP_ENDPOINT` is set; Prometheus `/metrics` on the internal port; slog JSON with `trace_id`, PII redaction |
| Web | pnpm workspace `web/`: Vite + React 19 + TypeScript strict, TanStack Router + TanStack Query, Tailwind CSS v4, Radix primitives, i18next (`en`, `te`), react-hook-form + zod, `vite-plugin-pwa`, `openapi-fetch` |
| Tests | Go `testing` (+ `-race`), integration tests against real PostGIS (`ROVO_TEST_DATABASE_URL`, fresh DB per package from a migrated template), Vitest + Testing Library + MSW, Playwright (Chromium at `/opt/pw-browsers`), k6 optional |
| Local stack | Docker Compose (`deploy/compose`): postgis, minio, mailpit, api, worker, caddy (serves 4 SPAs on 4 host ports and routes `/api/*` same-origin) |

## 3. V1 implementation slice (what "Phase 2 done" means here)

Golden flow end-to-end: **Customer → Restaurant → Order → Delivery Partner → Pickup → Delivery → Payment/Settlement → Rating**, plus:

| Area | Included |
|---|---|
| Identity & RBAC | Phone OTP (fake SMS provider, India mobiles only, hashed OTP, attempts/TTL, rate limits in Postgres), access JWT (EdDSA, 10 min / admin 5 min) + rotating refresh with reuse detection, host-only cookies per app, admin email+argon2id+**TOTP**, roles CUSTOMER / RESTAURANT_OWNER / RESTAURANT_STAFF / RIDER / ADMIN_SUPER / ADMIN_OPS / ADMIN_SUPPORT / ADMIN_FINANCE + SYSTEM, ownership checks, audience derived from Host |
| Geo | City, localities, zone polygon (PostGIS), serviceability (zone ∩ 7 km straight-line), road distance ×1.3, fee slabs R18 |
| Catalog | Restaurant onboarding application → admin approval; profile, operating hours, pause/resume/open/close; menu categories, items (veg/non-veg/egg, `*_i18n`), variants, add-on groups (min/max), availability toggle; public discovery (filters: veg, cuisine, sort), restaurant + menu pages, search (pg_trgm) |
| Cart & checkout | Stateless `POST /cart/quote` → signed `quoteId` (10 min, stored), bill lines (item total, packaging, delivery fee, platform fee, small-cart fee, discount, GST 5% on food, ROUND_OFF), coupons (one per order; flat/percent, min order, max discount, per-user/global limits, platform/restaurant funded), `POST /orders` with `Idempotency-Key`, 409 diff on stale quote, COD caps (R6) |
| Orders | Full state machine (doc 13, R1–R5, R40), timers via River (payment timeout, 180 s accept ladder, 60 s auto-PREPARING), status history, cancellation matrix, restaurant inbox |
| Payments | `PaymentProvider` interface; **fake PA** (hosted-checkout page + HMAC-signed webhook simulator); webhook verify + fetch-to-confirm + idempotency; refunds (full/partial) |
| Dispatch & delivery | Rider application → admin approval; online/offline + location pings; two-tier candidate search (R34); offers with 45 s timeout cascade (River), `REVOKED`; admin manual assign; delivery steps AT_RESTAURANT → PICKED_UP → AT_DROP → DELIVERED; delivery OTP for prepaid ≥ ₹300 (R39); COD collection + cash-in-hand limit (R6); undeliverable request → support approval |
| Ledger & settlement | Double-entry journals (DB trigger enforces balance), postings on delivery / COD / refund / coupon; commission; rider pay (₹25 + ₹6/km > 2 km + waiting); weekly settlement run → payout batch → maker-checker approval → mark paid with UTR; COD deposits |
| Notifications | Notification inbox, SSE `GET /api/v1/stream` (20 s heartbeat, 30-min cap, `reauth`), LISTEN/NOTIFY fan-out, fake SMS sink, Web Push (VAPID) adapter with fake in dev, restaurant alert ladder |
| Ratings & reviews | Restaurant stars + tags + optional text (profanity filter, admin hide); rider 👍/👎; 7-day window; restaurant rating aggregates ("New" until 5) |
| Coupons | Admin CRUD, validation, redemption accounting |
| Analytics | Restaurant today/week summary; admin dashboard KPIs; CSV reports (orders, settlements) |
| Audit | Append-only `audit_logs` (DB grants + trigger), all admin & auth events, viewer in admin |
| Apps | `customer`, `restaurant`, `rider`, `admin` PWAs (admin without service worker), en/te, mobile-first |

**Explicitly out of this slice** (backlog, not cut): real PA sandbox adapters (interface + Razorpay-style signature code only), real SMS/DLT, KYC document upload pipeline beyond a basic presigned upload, passkeys, bulk CSV menu import, ops phone ordering, voice escalation, OTel collector deployment, AWS apply. Everything in `00` §9 cut list stays cut.

## 4. Conventions

- Module layout: `backend/internal/modules/<module>/{module.go (public API), store.go, service.go, http.go, jobs.go, *_test.go}`; platform in `backend/internal/platform/*`. No module imports another module's internals; cross-module calls go through the exported service interface; events go through River jobs.
- HTTP: base path `/api/v1`; JSON camelCase; errors RFC 9457 `application/problem+json` with stable `code`; money `{amountPaise, currency}`; RFC 3339 UTC timestamps; cursor pagination `{items, nextCursor}`; `Idempotency-Key` required on POST /orders and payment-initiation; `If-Match`/`version` for state commands where specified.
- Every handler: decode with size limit (1 MiB default) + `DisallowUnknownFields`, validate, authorise (role + ownership), call service, map errors, log with request id.
- Clock: `platform/clock.Clock` injected everywhere; no `time.Now()` in modules; no SQL `now()` for business deadlines (R19).
- Test hooks: `/_test/*` only with build tag `testhooks` (clock advance, run due jobs, read fake OTP/SMS/push, reset/seed), separate listener, env guard.
- Commits: conventional commits; small and frequent; never commit secrets (`.env` is git-ignored; `.env.example` documents vars).

## 5. Agent ownership (avoid edit collisions)

| Agent | Owns |
|---|---|
| Architect / Database Engineer | `openapi/`, `backend/migrations/`, `backend/internal/platform/`, `backend/cmd/`, module skeletons & interfaces, seed data |
| Backend Engineer A | modules `identity`, `geo`, `catalog`, `adminaudit` (+ admin endpoints for users/restaurants/riders approval) |
| Backend Engineer B | modules `pricing`, `promotions`, `ordering`, `payments` |
| Backend Engineer C | modules `dispatch`, `ledger`, `ratings`, `notifications` (SSE/push/SMS), `reporting`, `support` |
| Frontend Engineer(s) | `web/` (customer; restaurant + rider; admin; shared packages) |
| DevOps/SRE | `deploy/`, `.github/`, `Makefile`, `scripts/`, Dockerfiles, backup/restore |
| QA Engineer | `e2e/` (Playwright), k6, cross-module integration tests |
| Security Engineer / Code Reviewer / UX Reviewer | review reports in `docs/implementation/reviews/`; fixes go back to the owning agent |
| Release Engineer | `docs/implementation/release/`, smoke tests, deployment verification, release notes |

## 6. Deliberate implementation deviations (ADRs)

- **ADR-P2-001** OpenAPI is the contract; enforced by response-validation contract tests and TS type generation. Go strict-server code generation (oapi-codegen) is deferred — hand-written handlers on stdlib ServeMux keep the build simple; the spec cannot drift because every handler's responses are validated against it in tests.
- **ADR-P2-002** `pgx` with hand-written parameterised SQL instead of `sqlc` generation for the first slice (fewer generation steps across parallel engineers); all SQL is parameterised (no string concatenation of user input), reviewed by the Security Engineer. Revisit sqlc when the schema stabilises.
- **ADR-P2-003** Deployment target for this phase is the approved **dev/demo path** (Phase 1 §4a, R24): production-mode Docker Compose stack, optionally exposed through a card-free tunnel. AWS (the approved production target) needs accounts, a payment method and explicit approval at M6; IaC is authored and validated, not applied.
