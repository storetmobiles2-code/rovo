# 09 — Architecture Decision Records

| | |
|---|---|
| **Purpose** | Records each significant architecture decision for rovo V1: the context, the options weighed, the decision, its consequences, and the conditions that should make us revisit it. Each baseline provisional decision (P1–P17 in doc 00) is examined critically here and either confirmed or changed. |
| **Owner** | Solution Architect |
| **Status** | Draft v1.1 — reconciled with review (31) and rulings R1–R48, 2026-10-04. Every ADR below is **Proposed**. They become **Accepted** when the Lead Architect signs off in doc 32. |
| **Depends on** | `00-planning-baseline.md` (incl. rulings R1–R48, §8–§9), `31-review-report.md` |
| **Referenced by** | `08-system-architecture.md`, `14-payment-architecture.md`, `15-notification-architecture.md`, `17-frontend-architecture.md`, `21/22/24/25` (DevOps docs), `26-repository-structure.md`, `30-risks-assumptions-decisions.md` |

Format: lightweight MADR. Each ADR is immutable once Accepted. A change of mind gets a new ADR that supersedes it. In Phase 2 these will be split into one file per ADR under `docs/adr/NNNN-title.md` (see doc 26).

Tags: `[ASSUMPTION]`, `[OPEN]`, `[LEGAL]`.

**Changes in v1.1 (2026-10-04).** History is kept: changed ADRs carry an "**Amended 2026-10-04 per R-xx**" block at the end instead of rewritten text.
- Amended: ADR-001 (C9 tooling), 003 (R15/R20 paths), 005 (R22 PG 17), 006 (R22/R42 direct `InsertManyTx`, no event table; M11 catch-up periodic jobs), 008 (R10/R27, M12 NOTIFY safety), 009 (R14 four apps under `web/`, R33 no prerender), 011 (R14/R27/R37/R44), 012 (R25/R35/R46), 014 (R16/R34/R30), 016 (R23/R24/R28/R32 two IaC profiles), 017 (R36 Grafana Cloud + Faro, 180-day archive, no Sentry), 018 (R17), 022 (R38), 023 (R15), 025 (RV-074 realistic portability), 026 (R23/R32, C18).
- New: **ADR-027** cross-module FKs on money paths (R41); **ADR-028** same-origin API edge, no CORS, `api.` reserved (R14/R27).

## Index and baseline verdicts

| ADR | Title | Baseline item | Verdict |
|---|---|---|---|
| 001 | Modular monolith vs microservices | P1 | **Confirmed** |
| 002 | Go for the backend; net/http ServeMux router | P1 | **Confirmed** (router: stdlib, not chi) |
| 003 | REST + OpenAPI spec-first; oapi-codegen + openapi-typescript | P2 | **Confirmed**, with a 3.1 tooling gate |
| 004 | No gRPC in V1 | P2 | **Confirmed** |
| 005 | PostgreSQL 17 + PostGIS; pgx + sqlc; goose | P5 | **Confirmed**; amended per R22 (PG 17 on RDS; 18 only if available with PostGIS) |
| 006 | Async: River-as-outbox | P4 | **Refined**; amended per R22/R42: one River job per subscriber via `InsertManyTx`, no event table, no fan-out hop |
| 007 | No Redis at launch; managed Redis by config later | P6 | **Confirmed** (decided) |
| 008 | SSE for real-time | P3 | **Confirmed** (plus heartbeat and invalidation-hint rules); amended per R10/R27/M12 |
| 009 | Vite SPA/PWA, not Next.js | P7, P8 | **Confirmed**. Four apps under `web/` (R14); no build-time prerender (R33). |
| 010 | Monorepo layout | — | New |
| 011 | Auth tokens: hybrid JWT + server-side session state | P9 | **Confirmed with refinement** (aligned with doc 12 AUTH-D04/D05); amended per R37/R44 |
| 012 | Payment aggregator choice and abstraction | P10 | **Confirmed with change**; amended per R25/R35/R46: Cashfree vs Razorpay on written rates (effective rate is go/no-go); both money-flow models supported; legal opinion before Phase-2 week 4 |
| 013 | Money as integer paise + double-entry ledger | P10, §3 | **Confirmed** |
| 014 | Rule-based dispatch with offer cascade | P11, P12 | **Confirmed**; amended per R34 (two tiers) and R16 (`REVOKED`) |
| 015 | Maps: MapLibre + OpenFreeMap, no paid geocoding | P13 | **Confirmed** (tile source named; OSM.org tiles ruled out) |
| 016 | Deployment: managed containers on a hyperscaler (prod/staging); Compose for local | P14, P17 (§4a) | **Changed by user directive**; amended per R23/R24/R32 (AWS, two IaC profiles) |
| 017 | Observability via OpenTelemetry | P15 | **Confirmed**; amended per R36 (Grafana Cloud incl. Faro, 180-day India archive, no Sentry) |
| 018 | i18n approach | §3 | **Confirmed** |
| 019 | UUIDv7 identifiers | §3 | **Confirmed** |
| 020 | Multi-city data model (shared schema, `city_id`) | §3 | **Confirmed** |
| 021 | Search: Postgres FTS + pg_trgm | — | New |
| 022 | File uploads: presigned direct-to-object-storage, client-side resize | — | New |
| 023 | API conventions: `/v1`, problem+json, cursor pagination, Idempotency-Key, ETag | P2 | New |
| 024 | CI/CD: GitHub Actions, OIDC to cloud, registry push | P16 | **Confirmed** (DevOps owns the details) |
| 025 | Cloud portability via standard interfaces | §4a | New (user directive); amended per RV-074 (app portable, infra rebuild = weeks) |
| 026 | Infrastructure as Code: OpenTofu/Terraform | §4a, P16 | New (user directive); amended per R23/R32 |
| 027 | Cross-module foreign keys on money paths | — | **New** (R41) |
| 028 | API edge: same-origin `/api/*` on every app host; no CORS; `api.` reserved | P7, §4a | **New** (R14/R27) |

---

## ADR-001: Modular monolith instead of microservices

**Context.** V1 serves a single city (≈2–2.5 lakh people `[ASSUMPTION]`), built by a small team, with a lean infrastructure budget. Production runs on managed hyperscaler services (baseline §4a), where every additional service costs money and operational attention. The domain has tightly coupled, money-bearing workflows (order ↔ payment ↔ ledger ↔ dispatch) whose consistency matters more than independent scalability. Expected volume is on the order of 10³ orders per day `[ASSUMPTION]`.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| A. Modular monolith (one binary; api/worker modes) | One deploy and one DB, so local ACID transactions are available where money needs them. Refactors are cheap. Lowest operational cost. Easy local dev. Boundaries can still be enforced. | Discipline is required, otherwise it decays into a big ball of mud. Everything scales together. A bad deploy affects everything. |
| B. Microservices (order, payment, dispatch, notification…) | Independent deploy and scale. Fault isolation. | Distributed transactions/sagas around money. Network failure modes. N pipelines, N databases, observability overhead. Needs a broker and service discovery. Far more ops than the team or budget allow. No load justifies it. |
| C. Classic layered monolith (no module boundaries) | Fastest to start | Couples everything. Extraction later is very hard. |
| D. Serverless functions (Cloud Run / Lambda / Workers) | Scale to zero | Long-lived SSE connections and timers fit poorly. Cold starts. Vendor lock-in. Free-tier CPU limits (e.g. 10 ms CPU per request on Workers Free, https://developers.cloudflare.com/workers/platform/limits/, accessed 2026-10-04). |

**Decision.** **A: modular monolith** in Go, with compiler-enforced module encapsulation (nested `internal/`), a CI-enforced dependency DAG, table ownership, and events through a transactional outbox (doc 08 §4).

**Consequences.** + One process to debug. Atomic multi-table money operations. Cheap hosting. − We must invest early in boundary tooling (go-arch-lint, table-ownership check). − Single blast radius, mitigated by separate api/worker processes and fast rollbacks.

**Revisit when.** A module needs a different scaling profile or deploy cadence (notifications, dispatch are the likely first). The team grows beyond ~3 squads stepping on each other. Any §9.5 trigger in doc 08.


**Amended 2026-10-04 per C9 (31 RV-007).** Boundary tooling in V1 is **go-arch-lint and/or golangci-lint depguard** plus a `table_ownership.yaml` review checklist and CODEOWNERS. The custom `tools/tableowner` and `tools/eventschema` programs are **not built in V1**; event compatibility uses JSON fixtures per event (doc 20 §6.7). Build `tableowner` only after the first cross-module SQL incident.

---

## ADR-002: Go for the backend; HTTP routing with net/http ServeMux

**Context.** We need a backend language with low memory use (small, cheap container tasks), good concurrency (thousands of SSE streams, timers), strong typing, simple deployment (static binary, multi-arch for ARM), and a healthy Postgres ecosystem. Contributors to an open-source Indian project are likely to know JS/TS, Java or Go.

**Options (language).**

| Option | Pros | Cons |
|---|---|---|
| **Go** | Single static binary. Small memory footprint (~30–80 MB). Goroutines suit SSE and timers. Excellent pgx/sqlc/River ecosystem. Fast builds. Easy arm64 cross-compile. Readable for newcomers. | Verbose error handling. Less expressive domain modelling (no sum types). Code generation is needed for the DB layer. |
| Node.js / TypeScript (NestJS, Fastify) | One language across front and back. Huge pool of contributors. | Higher memory per process. CPU-bound work blocks the event loop. ORMs (Prisma, TypeORM) are heavier. Dependency churn and supply-chain surface. The type system is unsound at runtime boundaries without Zod-style validation. |
| Java / Kotlin (Spring Boot) | Mature, rich ecosystem (jOOQ, Flyway). Strong typing. Kotlin is expressive. | JVM memory (≥ 256–512 MB per process) raises the per-task memory size, and so the cost, on managed containers. Heavier framework magic. Slower startup (GraalVM native adds complexity). |
| Rust (Axum) | Performance, safety | Learning curve and slower iteration. Smaller contributor pool. Overkill for this load. |

**Options (router).** Go 1.22+ `net/http.ServeMux` supports method and wildcard patterns (`GET /v1/orders/{id}`). oapi-codegen can emit `std-http` servers.

| Option | Pros | Cons |
|---|---|---|
| **stdlib ServeMux** | Zero dependencies. Stable forever. Generated routes plug in directly. Middleware is plain `func(http.Handler) http.Handler`. | No route groups or sub-router helpers. We write ~50 lines of middleware-chaining helpers. |
| chi v5 | Groups and sub-routers, rich middleware set, still `net/http` compatible | An extra dependency for little gain when routes are generated |
| Echo / Gin / Fiber | Familiar to many | Non-standard handler signatures (Fiber is not even net/http). Tighter lock-in. |

**Decision.** **Go 1.27.x** (current stable; 1.27.1 released 2026-09-01, https://go.dev/doc/devel/release, accessed 2026-10-04). Router: **stdlib `net/http.ServeMux`** with oapi-codegen strict-server (`std-http-server`) adapters. Logging: `log/slog`. No web framework.

**Consequences.** + Minimal dependencies. Long-term stability. − Contributors coming from TS must learn Go. We mitigate with a CONTRIBUTING guide and examples (doc 26). − The domain state machine is implemented explicitly with transition tables (doc 13), not with type tricks.

**Revisit when.** Middleware composition becomes painful: switch to chi, which is a drop-in because it is net/http compatible. The team composition becomes overwhelmingly TS-only and Go becomes a hiring blocker.

---

## ADR-003: REST/JSON with OpenAPI spec-first and generated code

**Context.** We have four web clients now and possibly native apps later, plus external integrators (restaurant POS) in future. We want one contract, typed clients, and server stubs that cannot drift from the spec.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| **Spec-first OpenAPI** (hand-written YAML is the source of truth; generate server interfaces and TS client) | The contract is reviewed before code. Drift is caught in CI. Docs come free (Scalar/Redoc). Works for any future client. | YAML authoring is verbose. Codegen tool limitations shape the spec. |
| Code-first (annotations → spec, e.g. huma, swag) | Less duplication | The spec becomes an afterthought. Annotation drift. Harder to review API design separately. |
| GraphQL | Flexible client queries | Caching, auth per field and N+1 complexity. Over-powered for our screens. Needs a server runtime library. Poor fit with HTTP caching and idempotency keys. |
| tRPC / Connect | Great DX inside a TS stack | tRPC is TS-only on the server. Connect is gRPC-flavoured (ADR-004). |

**Codegen choices (Go).**

| Tool | Pros | Cons |
|---|---|---|
| **oapi-codegen v2.8.0** (2026-07-17; "initial OpenAPI 3.1 support"; https://github.com/oapi-codegen/oapi-codegen/releases, accessed 2026-10-04) | Most widely used. Strict-server mode gives typed request/response objects. Supports stdlib `net/http`. Can generate types only. | 3.1 support is new. Some schema constructs (complex `oneOf`, `type: [x, "null"]`) may generate awkward types. |
| ogen v1.24.0 | Generates its own fast router and encoders with no reflection. Strong validation built in. | Opinionated: its own server and optional types. Smaller community. Harder to mix hand-written handlers (SSE, webhooks). |

**TS client:** `openapi-typescript` 7.13.0 (types) + `openapi-fetch` (tiny typed fetch). Zero runtime codegen beyond types, and good 3.1 support.

**Decision.** **Spec-first OpenAPI 3.1**, authored in `/openapi` and split by module (bundled with Redocly CLI). Server: **oapi-codegen strict-server, std-http**. Request validation: generated types plus `kin-openapi` request-validation middleware in non-prod and on selected routes in prod `[OPEN: measure overhead]`. TS client: **openapi-typescript + openapi-fetch** in `packages/api-client`.

**Gate (first week of Phase 2):** a spike generates server and client from a representative spec slice (nullable fields, enums, `oneOf` discriminators, problem+json, cursor pagination). **If oapi-codegen's 3.1 handling is unsatisfactory, author the spec as OpenAPI 3.0.3.** Down-converting is mechanical because we restrict ourselves to a "3.0-compatible subset" style guide (doc 11). Nullable is modelled via `type: [T, "null"]` only, and examples are kept simple.

Hand-written (not generated) endpoints: `GET /v1/stream` (SSE) and `POST /webhooks/pa/*` (raw body needed for signature verification). They are still documented in the spec.

**Consequences.** + Contract reviews in PRs. Breaking-change detection with `oasdiff` in CI. − The spec style guide must be followed.

**Revisit when.** The codegen tool blocks needed constructs, or ogen's ergonomics improve enough to justify the switch.


**Amended 2026-10-04 per R15/R20.** All operations, including the hand-written ones, live under **`/api/v1` on every host**: `GET /api/v1/stream` (SSE) and `POST /api/v1/webhooks/payments/{provider}` (raw body for signature verification). The spec is authored as 3.1 restricted to the subset validated against oapi-codegen, openapi-typescript and the mock generator; the week-1 spike falls back to 3.0.3 if the tooling fails (R20).

---

## ADR-004: No gRPC in V1

**Context.** gRPC shines for internal service-to-service calls with streaming and strict schemas. V1 has no internal network boundary (ADR-001). Browsers can't speak native gRPC (grpc-web/Connect need proxies or special servers).

**Options.** (A) REST only. (B) gRPC internally + REST gateway (grpc-gateway) externally. (C) Connect RPC for both.

**Decision.** **A.** No gRPC or Protobuf in V1. In-process Go interfaces are the "internal API".

**Consequences.** + One protocol, one toolchain. Curl-debuggable. Cloudflare and Caddy friendly. − If services are extracted later, we decide then (gRPC/Connect vs REST). The Go interfaces at module boundaries make that a contained change.

**Revisit when.** The first service is extracted (doc 08 §9.5) and has chatty, latency-sensitive internal calls, or a native app needs bidirectional streaming.

---

## ADR-005: PostgreSQL 18 + PostGIS; pgx + sqlc; goose migrations

**Context.** Orders and money need relational integrity, transactions and constraints. Zones need polygon containment and distance queries. The job queue (ADR-006) and pub/sub (ADR-008) also want Postgres.

**Options (DB).** PostgreSQL+PostGIS; MySQL 8 (spatial support is weaker, no River); MongoDB (no multi-document ACID ergonomics for ledgers, weaker spatial-relational joins); managed serverless Postgres (Neon/Supabase: fine as fallback hosting, doc 25).

**Options (data access).**

| Option | Pros | Cons |
|---|---|---|
| **sqlc + pgx v5** | You write SQL and get type-safe Go. No runtime reflection. PostGIS and CTEs are expressible. Query review is plain SQL. | Dynamic queries (filters) need care (`sqlc.narg`, or a small hand-written query builder for admin search). |
| GORM / ent / bun (ORMs) | Fast CRUD | Hidden queries and N+1. PostGIS awkward. Migrations coupled to the ORM. Harder to enforce table ownership. |
| squirrel / goqu (builders) | Dynamic SQL | Less type safety |

**Options (migrations).**

| Option | Pros | Cons |
|---|---|---|
| **goose v3.28.0** | Plain SQL files, Go migrations when needed, embeddable in our binary (`rovo migrate`), simple | Linear versioning (we use timestamps to reduce conflicts) |
| atlas | Declarative diffing, lint (destructive-change detection) | Larger tool. Some features are commercial. A declarative model can surprise reviewers. |
| golang-migrate | Popular | Fewer features than goose. No Go migrations. |

**Decision.** **PostgreSQL 18.x** (current 18.6; supported until Nov 2030; https://www.postgresql.org/support/versioning/, accessed 2026-10-04) with **PostGIS 3.x**, **provided the chosen managed Postgres service offers PG18 + PostGIS as GA at Phase 2 start**. Otherwise use **17.x** (supported until Nov 2029), as doc 22 currently plans for RDS. The baseline said "17+". 18 is preferred for its longer support window and native `uuidv7()` (ADR-019), but **no code may depend on PG18-only features**: IDs are app-generated, so the `uuidv7()` default is only used if available. Local, CI and prod must run the **same major version**. Data access: **pgx v5 + sqlc**. One sqlc package per module, limited to owned tables. Migrations: **goose**, SQL-first, timestamped, embedded. River's own migrations run through `rivermigrate` inside `rovo migrate`. Optionally use `atlas migrate lint` in CI (free/OSS features only) as a destructive-change detector `[OPEN: DevOps]`.

**Consequences.** + SQL is reviewable, fast and explicit. − Contributors must be comfortable with SQL. − Expand/contract migration discipline is required for zero-downtime deploys (doc 21).

**Revisit when.** Dynamic-query burden grows significantly (consider adding a builder for admin search only).


**Amended 2026-10-04 per R22.** **PostgreSQL 17 on RDS** with PostGIS 3.5 is the decision (18 only if RDS offers it with PostGIS at Phase 2 start). No PG18-only features anywhere (the `uuidv7()` default is not used; IDs are app-generated, ADR-019). Local, CI and production run the same major (17).

---

## ADR-006: Asynchronous processing with River as the transactional outbox

**Context.** State changes must reliably trigger side effects (notifications, dispatch, ledger postings, refunds) without dual-write bugs. Timers are needed (offer expiry 45 s, accept timeout, payment expiry). Volume is small (< 10 jobs/s at peak `[ASSUMPTION]`).

**Options.**

| Option | Pros | Cons |
|---|---|---|
| **Postgres outbox + River (Postgres-backed queue)** | Transactional enqueue (`InsertTx`) gives exactly-once *enqueue* with the business change. Scheduled jobs, retries with backoff, unique jobs, periodic jobs, a web UI (riverui). No extra infra. | Load stays on the main DB. Pre-1.0 library (v0.48.0, 2026-10-01). Throughput is bounded by Postgres (thousands/s is fine). |
| Custom outbox table + poller → in-process handlers | No dependency | We would reinvent retries, scheduling, uniqueness and observability |
| Redis queues (asynq) | Fast, mature | Dual write (DB commit + Redis enqueue are not atomic), so we would still need an outbox. Adds Redis. |
| NATS JetStream / Kafka / RabbitMQ | Scale, fan-out, replay | Extra infra and ops. Still needs an outbox for atomicity. Overkill. |
| pg-boss / Graphile Worker | Same idea | Node-only |

**Decision.** **River** (pin the exact version; v0.48.0 at time of writing, https://github.com/riverqueue/river/releases, accessed 2026-10-04).
**Refinement of P4:** we do **not** build a separate outbox table plus relay. `events.Record(tx, evt)` writes the event to `event_log` (an audit/debug record) **and** inserts a River `event.fanout` job in the same transaction. River's transactional insert *is* the outbox. The fan-out worker enqueues one unique job per subscriber. Handlers are idempotent (doc 08 §7). Delayed work (offer expiry, accept timeout, payment expiry) uses River `ScheduledAt`. Recurring work (settlement, recon, cleanups) uses River periodic jobs with leader election.
All modules access the queue via `platform/queue` (an interface), never River types directly.

**Consequences.** + No dual writes, no broker. − River upgrades must be read carefully (pre-1.0 API changes). − Queue tables need vacuum tuning (River cleans completed jobs; set retention to 24–72 h).

**Revisit when.** Sustained > ~1,000 jobs/s, cross-service events after extraction (then NATS JetStream), or queue-induced DB load over 20% of DB CPU.


**Amended 2026-10-04 per R22 and R42 (31 RV-002, RV-004; scope cut C8; missing item M11).**
1. **No `event_log` / `outbox_events` table and no `event.fanout` hop.** `events.Publish(ctx, tx, evt)` looks up the static subscription table and inserts **one River job per subscriber in the same transaction with `InsertManyTx`** (unique key `(handler, event_id)`; args carry the envelope and trace context). River's transactional insert is still the outbox (R22): no relay, no dual write. Handlers dedupe via `processed_events`. This removes one River fetch cycle from the restaurant-ring path; the latency budget is in doc 08 §7.3.
2. **Periodic jobs use catch-up semantics** because River OSS periodic jobs are not durable across a leader restart: each recurring job runs hourly, checks whether the run for period P has completed, and runs idempotently if not. A "missed settlement" alert fires if no settlement run exists for last week by Monday 09:00 IST (doc 08 §7.4).
3. Pin River exactly (pre-1.0); upgrade only between releases with the full integration suite (RV-077).

---

## ADR-007: Redis at launch? **No.** Managed Redis later, by configuration

**Context.** Redis is commonly used for caching, rate limiting, pub/sub, sessions and queues. At V1 scale, each of these needs is covered by Postgres or in-process memory. In production on a hyperscaler, a managed Redis-compatible cache (ElastiCache/MemoryDB for Valkey or Redis, Memorystore, Azure Cache/Managed Redis) is always *available*. The question is whether to **pay for it and depend on it at launch**. The smallest Multi-AZ managed cache nodes are a fixed monthly cost line `[DevOps to price in doc 22/25]`. Locally, a Valkey container is trivial.

**Needs analysis.**

| Need | V1 solution without Redis | Breaks when |
|---|---|---|
| Job queue | River (ADR-006) | — |
| Pub/sub to SSE hubs | Postgres `LISTEN/NOTIFY` (works across any number of API tasks) | NOTIFY commit-lock contention at roughly thousands/s |
| Rate limiting: OTP/login | Postgres table (`rate_limit_buckets`, upsert with window) + edge WAF rules | > a few hundred req/s on auth (no) |
| Rate limiting: general API | Edge WAF per-IP rules + in-process token bucket per task | Precise global per-user limits are needed across many tasks |
| Sessions | Postgres + in-process cache ≤ 30 s, NOTIFY bust (doc 12) | Very high RPS with many tasks |
| Caching (menus, zones) | In-process LRU with NOTIFY invalidation | Memory per task, or cold caches on many tasks |
| Distributed locks | Postgres advisory locks / row locks | — |

**Options.** (A) no Redis at launch, with the adapter ready; (B) managed Redis from day one; (C) self-managed Valkey container in production (rejected: production must use managed services, §4a).

**Decision.** **A.** Interfaces `ratelimit.Limiter`, `pubsub.Bus`, `cache.Cache` ship with Postgres/in-memory implementations **and** a Redis-protocol adapter (go-redis/valkey-go client), tested in CI against a Valkey container. Switching is **configuration only** (`CACHE_BACKEND`, `RATELIMIT_BACKEND`, `PUBSUB_BACKEND` = `memory|postgres|redis`, plus `REDIS_URL` from the secrets manager). The Terraform `cache` module exists but is disabled (`enable_cache = false`) for prod at launch. Locally, Valkey runs under the Compose profile `cache`, for testing the adapter. The app uses only the Redis protocol (no cloud-specific features), per ADR-025. If self-hosted anywhere, prefer **Valkey** (BSD-licensed) `[ASSUMPTION: re-verify Redis licence status at adoption time]`.

**Consequences.** + No fixed cache cost or extra failure domain at launch. − Slightly more DB load. Per-task general rate limits until enabled.

**Revisit when.** Any trigger in the table fires, or there are ≥ 4 API tasks. Enabling it is a Terraform flag plus env change, verified in staging first.

---

## ADR-008: Real-time via Server-Sent Events

**Context.** Customers need order status updates. Restaurants need instant new-order alerts while the app is open. Riders need offers within seconds. Traffic is server → client. Client → server actions are discrete REST calls that need idempotency. Clients include low-end Android on flaky mobile networks, behind a CDN/WAF and load balancer.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| **SSE** | Plain HTTP. Auto-reconnect built into `EventSource`. Works through CDNs, L7 load balancers and HTTP/2 multiplexing. Trivial in Go. Cookie auth works. | One-way. `EventSource` can't set headers (use cookies, or a fetch-based reader for bearer tokens). Idle-timeout behaviour behind proxies needs heartbeats. |
| WebSockets | Bidirectional, binary | A separate protocol to secure (origin checks, auth on upgrade). Custom reconnect and resubscribe logic. Bidirectionality is unused. More proxy edge cases. |
| Long/short polling only | Simplest | Latency vs load trade-off. Battery and data use. |
| Third-party realtime (Pusher, Ably, Firebase) | Offloads infra | Cost, vendor lock-in, data leaves our stack. Free tiers have connection caps. |

**Decision.** **SSE** on `GET /v1/stream` (one stream per tab, server-derived topics), fed by Postgres `LISTEN/NOTIFY`. Rules:
1. SSE messages are **hints with minimal payloads**. REST is the source of truth. Clients refetch on reconnect.
2. A heartbeat comment every 20 s, shorter than any idle timeout in the chain (LB ≥ 120 s configured; Cloudflare ~100 s if used), plus a 30-min max stream with `retry:` (doc 08 §6.2).
3. Fall back to polling after repeated failures.
4. **Web Push** for backgrounded or closed apps (doc 15). SSE is not a notification system.

**Consequences.** + Simple, cheap, robust. − No server replay buffer, so the client must refetch on reconnect (by design).

**Revisit when.** Two-way features are needed (in-app chat with typing indicators, or high-frequency rider telemetry when a native rider app exists). WebSockets or MQTT then.


**Amended 2026-10-04 per R10, R27 and M12 (31 RV-003, RV-011).**
- Endpoint is `GET /api/v1/stream` on each app host (same-origin, R15/R27). Heartbeat 20 s; `reauth` event; **no server replay buffer**, clients refetch snapshots on reconnect; LB/CDN idle timeout ≥ 120 s (R10). An open stream from a restaurant order-receiver device counts as its heartbeat (R27).
- **NOTIFY safety:** alert on `pg_notification_queue_usage() > 0.1`; LISTEN connections only read and are watchdogged (self-probe NOTIFY, reconnect on lag); fallback is to move NOTIFY to a post-commit best-effort step (doc 08 §6.2).
- Stream lifetime: 30-min cap with jittered `retry:`; continuing past access-token expiry while the session is valid is a *proposal* pending alignment with docs 11/12/22 (31 register row 29).

---

## ADR-009: Frontend: Vite React SPA/PWA (not Next.js)

**Context.** There are four user surfaces. Users are on mostly mid/low-end Android devices over 4G with variable quality. The budget is ~₹0 hosting. The project is open source and self-hostable. A future native app (React Native) is possible. SEO needs are low: a single-city food delivery app gains users through word of mouth, restaurant partnerships and local marketing, not organic search for menus `[ASSUMPTION]`. A few public pages (landing, about, legal policies) must be crawlable and are **required by payment aggregators for merchant onboarding** (terms, privacy, refund/cancellation, contact) `[ASSUMPTION: exact PA checklist to be confirmed at onboarding]`.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| **Vite + React SPA/PWA, static on CDN** | No server runtime, so near-zero hosting cost (object storage + CDN) and fully portable. Fast dev. Mature PWA tooling (vite-plugin-pwa/Workbox). One mental model. | No SSR, so first paint waits for the JS bundle. Needs strict bundle budgets. SEO only for prerendered public pages. |
| Next.js (App Router, SSR/RSC) | SSR/SEO, image optimisation, routing conventions | Needs a Node server or a platform. **Vercel Hobby is "restricted to non-commercial personal use only"** and any payment processing counts as commercial (https://vercel.com/docs/limits/fair-use-guidelines, accessed 2026-10-04), so production on Vercel needs Pro. Cloudflare Workers Free allows **10 ms CPU per request** (https://developers.cloudflare.com/workers/platform/limits/, accessed 2026-10-04), which SSR of React pages commonly exceeds, so we would need the Workers Paid plan. Self-hosting means an extra always-on Node service in our container runtime (cost, ops). RSC complexity. PWA/offline story is weaker. |
| Remix / React Router v7 framework mode | Good data loading. Can run SPA mode. | SSR mode has the same runtime cost. SPA mode ≈ Vite SPA anyway. |
| Astro + React islands | Excellent for content pages | App-like flows (cart, tracking) are mostly islands anyway |
| SvelteKit / SolidStart | Smaller bundles | Smaller ecosystem in India. No React Native sharing. |

**Low-end Android:** SSR helps the first visit but costs a server round-trip per navigation. Repeat visits (most food ordering is repeat) are dominated by the cached app shell, where an SPA/PWA does well. Mitigations: route-level code splitting, a customer initial JS budget ≤ 170 KB gzip `[OPEN: Frontend Architect to set]`, avoid heavy UI libraries, lazy-load MapLibre only on the address screen.

**React Native code sharing:** equal for both options. What is shareable is TS types, the generated API client, i18n catalogs, validation schemas and domain helpers (money formatting), not DOM UI.

**Decision.** **Vite + React + TypeScript SPA/PWA**, served as static assets from **object storage + CDN** in staging/production (P7 and P8 confirmed). Each app host routes `/api/*` to the Go API at the edge (same-origin API, docs 12 and 17). Public marketing/legal pages are **prerendered static HTML** at build time (a tiny static page set in the customer app, or a separate `apps/site`, at Frontend Architect's discretion). This satisfies PA onboarding and basic SEO without a server runtime.

**App split (P7 changed; decided by the Frontend Architect in doc 17 F2, endorsed here):** `partner` is split into two apps, **`restaurant`** and **`rider`**. Each PWA gets its own manifest (name/icon/start_url/orientation), install identity on the home screen, service worker and push subscription scope, and permission prompts (riders: geolocation; restaurants: notification and sound). Bundles also get smaller. Shared code lives in `packages/*`, so the marginal cost is one Vite config. They are hosted on separate subdomains (`restaurant.` / `rider.`), which keeps service worker scopes and push subscriptions cleanly separated.

**Consequences.** + Near-zero frontend hosting cost (object storage + CDN egress). No frontend server runtime to operate. No vendor runtime terms risk. Offline-capable shell. − SEO for restaurant/menu pages is limited (acceptable at V1).

**Revisit when.** Organic search becomes an acquisition goal (e.g. "biryani in Mahabubnagar" landing pages). Then add SSR/SSG for public catalog pages only (Astro or Next on a paid plan), keeping the app as a SPA.


**Amended 2026-10-04 per R14 and R33.** Four apps and hosts: `app.` (customer), `restaurant.`, `rider.`, `admin.`, in a pnpm workspace under **`web/`** (apps under `web/apps/*`, shared code under `web/packages/*`). **No build-time prerendering** (no TanStack Start, no nightly rebuild against production data): each SPA's `index.html` carries static OG/meta tags, PA-required legal pages are static HTML files shipped with the customer app, and restaurant share-preview pages (`/r/{slug}`: HTML with OG tags plus a redirect into the SPA) are served by a small Go handler (P1). SSR is revisited only under the triggers above.

---

## ADR-010: Monorepo (Go module + pnpm workspace in one repository)

**Context.** The OpenAPI spec is shared between Go server and TS clients. Atomic changes (spec + server + client) should land in one PR. The contributor experience should be simple.

**Options.** (A) single monorepo; (B) polyrepo (backend, web, spec); (C) monorepo with Nx/Turborepo/Bazel.

**Decision.** **A: one repository**: `/backend` (Go module), `/openapi`, `/web` (pnpm workspace: `apps/{customer,restaurant,rider,admin}`, `packages/{ui,api-client,i18n,config,...}`), `/deploy`, `/docs`, `/tools`, `/scripts`. Task runner: **Taskfile** (go-task) or Make. Doc 26 picks one. **No Nx/Bazel.** pnpm workspaces plus Turborepo caching may be added if web CI exceeds ~5 min `[OPEN]`. CI uses path filters so a change to web code doesn't run Go tests and vice versa, except for spec changes, which run both.

**Consequences.** + Atomic contract changes. One issue tracker. − Larger clone (fine). Mixed toolchains in CI (handled with path filters).

**Revisit when.** Separate teams need separate release cadences, or a native app repo appears (it may live in the monorepo as `/mobile`).

---

## ADR-011: Authentication tokens: hybrid short-lived JWT + server-side session state

**Context.** Customers, riders and restaurant staff sign in with phone + OTP. Admins sign in with email + password + mandatory TOTP. Each web app calls the API **same-origin** (`<app-host>/api/*`), so cookies are host-only per app. Future native apps need bearer tokens. The Security Architect owns the detailed design (doc 12). This ADR records the architectural trade-off.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| JWT access only (15 min) + rotating opaque refresh (baseline P9) | Stateless verification. Standard. | Revocation is delayed up to the TTL, which is unacceptable for admins and riders handling cash. |
| Opaque server-side session tokens only | Instant revocation. Nothing to sign or rotate. | A DB/cache lookup on every request, including high-volume customer browsing. Service extraction later needs introspection. |
| **Hybrid (doc 12):** EdDSA-signed access JWT (10 min; admin 5 min) + opaque rotating refresh with reuse detection, **plus server-side session-state checks for partner (cached ≤ 30 s) and admin (uncached)** | Cheap for high-volume customer traffic. Fast revocation where privilege is high. Asymmetric keys let future services and edges verify without signing keys. | Two mechanisms to implement and test. Key rotation (`kid`) must be operated. |
| Third-party auth (Firebase/Supabase/Auth0/Cognito) | Phone OTP built in | Per-MAU/SMS cost, lock-in (conflicts with ADR-025), admin TOTP flows are vendor-specific |

The Solution Architect initially proposed opaque-only. It was weighed and **not** chosen, because doc 12's hybrid gives the same revocation guarantee where it matters and avoids per-request lookups for customers.

**Decision.** Adopt **doc 12 AUTH-D03/D04/D05**: per-audience sessions, EdDSA access JWT in a host-only `__Host-` cookie (web) or bearer (native), opaque rotating refresh token hashed in the DB, and session-state checks for partner and admin. The signing key lives in the secrets manager (KMS-encrypted) and is injected at runtime. Rotation uses `kid` with a JWKS kept in config, not a cloud KMS signing API, to stay portable (ADR-025). The webhook and native host `api.<domain>` is bearer-only.

**Consequences.** + Revocation is fast where it matters. The model is ready for native apps. − Key management is an operational task (runbook in doc 12/23).

**Revisit when.** Native apps launch (confirm token storage and DPoP needs), or services are extracted (internal token exchange).


**Amended 2026-10-04 per R14, R26, R27, R37 and R44.**
- Hosts are `app.`, `restaurant.`, `rider.`, `admin.`, each with same-origin `/api/v1` and host-only cookies (ADR-028). `api.<domain>` is reserved for future native bearer clients and server-to-server webhooks; no CORS anywhere in V1.
- Admins: **mandatory TOTP for all admins** at V1; passkeys (WebAuthn) are **P1**; **no identity-aware proxy** in V1; WAF rate and geo (India) rules on the admin host (R37). Admins are separate identities; `RIDER` ⟂ `RESTAURANT_*`; internal `SYSTEM` principal (R26).
- **Device-bound long-lived sessions** for restaurant order-receiver devices (sliding 30-day idle, 90-day absolute, revocable by owner/admin); riders 30-day sliding (R44).

---

## ADR-012: Payment aggregator choice and abstraction

**Context.** We need COD plus online payments (UPI intent/collect, cards, netbanking, wallets) via an RBI-authorised PA, with webhooks, refunds and settlement reports. Ideally also marketplace split settlement. The market is UPI-heavy. Pricing differs materially on UPI. Full analysis and sources are in doc 14.

**Key facts (accessed 2026-10-04; details and URLs in doc 14 §3).**
- **Razorpay Standard:** **2% platform fee on all domestic methods, UPI included**. Route +0.1%. Instant refunds ₹7.99–14.99 each. GST 18% on fees.
- **Cashfree standard TDR:** **1.95%** for UPI, cards, netbanking and wallets (doc dated 2026-08-10). Plus a **festive offer: 0% platform fee on domestic PG transactions up to a one-time ₹20 lakh GMV cap, for new merchants who register between 2026-07-21 and 2027-03-31** (GST still applies; exclusions apply).
- **PhonePe PG and Paytm PG:** both publish a flat **1.99%** standard plan. Third-party sources claim 0% on UPI (Paytm: UPI and RuPay debit at 0 MDR), but the official pricing pages fetched do not itemise by method, so this is **unverified**.
- **Juspay:** an orchestrator layered over PAs (~0.22–0.25% extra, enterprise-negotiated). Not needed at our scale.
- **UPI MDR policy is in flux:** a 2026 amendment bill would let the government notify MDR on UPI P2M, reportedly 0.4% above ₹2,000 for large merchants. Proposed, not enacted as of Aug–Sep 2026. Most of our orders are below ₹2,000.

**Cost sensitivity.** On a ₹350 average order `[ASSUMPTION]` paid via UPI, a 2% + GST fee costs ≈ ₹8.26 per order. That is more than the entire ₹5 platform fee. With 70% UPI share `[ASSUMPTION]`, every 0.5 percentage point of negotiated UPI rate is worth ≈ ₹2 per order. PA fees are the single largest variable technology cost per order in V1, larger than the per-order cloud cost target (doc 01 M-60: ≤ ₹6).

**Options.** (A) Razorpay only; (B) Cashfree only; (C) PhonePe PG / Paytm PG (claimed low/0% UPI); (D) orchestrator (Juspay); (E) provider-abstracted, choose commercially.

**Decision.** **E.** A `payments.Provider` Go interface (doc 14 §9) with:
1. **Reference adapter: Razorpay** (best-documented sandbox, official Go SDK, Route for split settlement, mature webhooks). It is built first so the open-source project has a working default.
2. **Commercial selection for the Mahabubnagar launch:** apply to **Razorpay and Cashfree** (both have marketplace split products and good docs). Also ask PhonePe PG/Paytm PG for written UPI pricing. Choose on (a) onboarding success for our entity type, (b) the **written** UPI rate (target well below 2%, since every PA negotiates), (c) split-settlement support, (d) launch offers such as Cashfree's ₹20 lakh 0% GMV allowance. If the winner isn't Razorpay, write its adapter (target ≤ 1 week, with the contract test suite shared across adapters).
3. **Fake provider** for local dev and E2E tests (simulates success, failure, delayed webhooks).

**Money flow (changes P10's default, subject to `[LEGAL]` opinion):** P10 assumed "platform collects all, then pays restaurants manually from the ledger". Under the RBI Master Direction on Payment Aggregators (15 Sept 2025), collecting customer funds for third-party merchants and settling to them is itself payment aggregation. Doing it outside a PA's escrow risks being treated as unauthorised PA activity `[LEGAL]`. **Preferred:** use the PA's **marketplace split settlement** (Razorpay Route / Cashfree Easy Split / equivalent) so that restaurant shares settle from the PA escrow directly to KYC'd restaurant linked accounts, on hold until delivery and released on our weekly cycle. The platform's own share (commission, fees) settles to the platform. The internal ledger stays the accounting truth and drives the transfer instructions. **Fallback:** collect-and-payout manually only if counsel confirms it is permissible for our structure. Riders are paid from the platform's own funds (contractor payments, not aggregation). COD cash settlement netting is described in doc 14 §11.

**Consequences.** + Cost-optimised, swappable, compliant by design. − Split settlement requires restaurant KYC with the PA (onboarding friction for small restaurants) and adds ~0.1% (Razorpay Route). − Two PA applications in parallel take founder time.

**Revisit when.** Volume above ~₹50 lakh/month GMV (negotiate custom pricing), PA outage history, or the regulatory opinion changes.


**Amended 2026-10-04 per R25, R35 and R46 (31 RV-010, RV-016, RV-075).**
- **Shortlist: Cashfree vs Razorpay**, chosen on **written** UPI/card rates at onboarding, behind the `Provider` interface (R25). The **PA effective rate is a go/no-go criterion** at selection: target ≤ 1% blended, UPI as low as negotiable (R46). If Cashfree wins, use its ₹20 lakh 0% allowance for the pilot.
- **Money flow:** the design supports **both** PA split settlement (model A) and collect-and-payout (model B). "Preferred: split settlement" above is no longer a decision; it is one of two supported options. A **legal opinion is required before Phase-2 week 4** (R35) because it changes payout code. If split settlement is chosen, PA linked-account KYC per restaurant joins the restaurant onboarding critical path. The restaurant share of COD cash is paid from rovo's account in either model and is covered by the same opinion `[LEGAL]`.
- **Lock-in (RV-075):** with split settlement every restaurant is a KYC'd linked account at that PA; switching PA means re-KYC and migrating held transfers. `transfers` are keyed by provider. This switching cost is recorded in doc 30.

---

## ADR-013: Money as integer paise, with a double-entry ledger

**Context.** Fees, GST, commissions, coupons funded by different parties, COD cash, refunds and payouts must reconcile to the paisa with a PA settlement report and a bank statement.

**Options.** (A) `BIGINT` paise + double-entry ledger; (B) `NUMERIC(12,2)` rupees + per-order columns; (C) floats (rejected outright); (D) an external ledger service (e.g. TigerBeetle, Formance, Blnk).

**Decision.** **A.** All amounts are `BIGINT` paise with `_paise` suffix and `currency CHAR(3) DEFAULT 'INR'` (baseline §3). In Go, a `money.Paise int64` type with explicit, tested rounding helpers (half-up at line level, per baseline; GST rounding rule per doc 14 §12). The ledger is an append-only **journal + postings** design with per-journal balance enforced by a deferred constraint trigger. Corrections are made only by reversing journals (doc 14 §10). An external ledger is not adopted: our volume is small, and adding infra conflicts with ADR-001/007.

**Consequences.** + Exact arithmetic, auditable, reconcilable. − Developers must use the `money` package and never `float64`. A linter rule forbids `float` in `ledger`, `payments` and `pricing` (forbidigo/custom analyzer).

**Revisit when.** Multi-currency is needed (only with international expansion).

---

## ADR-014: Rule-based dispatch with an offer cascade

**Context.** This is a compact city with two-wheeler riders and low concurrent order counts. Rider locations are foreground-only and refreshed every 30–60 s (PWA limits, P12). There is no paid routing API.

**Options.** (A) rule-based nearest-available with sequential offers and timeout; (B) broadcast offer to N riders, first-accept wins; (C) optimisation (Hungarian/batching with ETA models); (D) manual dispatch only.

| | Pros | Cons |
|---|---|---|
| **A** | Fair, predictable, easy to explain to riders and ops, easy to test | Sequential latency (45 s per decline) |
| B | Fast assignment | "Race" fatigue. Unfair to slower phones and networks. Many rejections. |
| C | Efficient at scale | Complex. Needs good ETA data we don't have. |
| D | No code | Doesn't scale. Ops burnout. |

**Decision.** **A**, with **manual assign** as the fallback and an option to switch a city to **B with N=2** via config if offer latency hurts `[OPEN: ops to evaluate after pilot]`. Scoring: distance (straight-line × road factor) from rider to restaurant, plus a fairness term (time since last delivery), plus a COD headroom filter. Location freshness < 3 min. One active delivery per rider. Offer expiry via River scheduled jobs. Dispatch starts timed to prep ETA (doc 08 §5.4).

**Consequences.** + Simple, transparent. − Not optimal in peaks (acceptable at V1 volume).

**Revisit when.** Average time-to-assign > 3 min at peak, or > 15% of orders hit `DispatchExhausted`. Then consider batching or a broadcast mode.


**Amended 2026-10-04 per R16, R30 and R34.**
- **Two dispatch tiers (R34):** tier 1 = location fresh ≤ 3 min, ranked by distance; tier 2 = stale ≤ 15 min, reached via Web Push (`Urgency: high`) + SSE. Auto-offline at 15 min without any ping or heartbeat. The values live in `app_config`, owned by doc 13 (R48). "Location freshness < 3 min" above is superseded.
- Offers have a fifth status **`REVOKED`** (withdrawn by system/admin) (R16).
- **No surge pricing** in V1; rider shortage is handled by zone pause plus a manual rider peak bonus (ledger adjustment) (R30).

---

## ADR-015: Maps and geocoding

**Context.** We need a pin-drop for addresses, zone drawing in admin, and a map of restaurant/drop for riders (with a link out to Google Maps app for navigation). Budget ≈ ₹0.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| **MapLibre GL JS + OpenFreeMap tiles** | Open source. OpenFreeMap: MIT, "no limits on the number of map views or requests", no API keys, commercial use allowed, self-hostable (https://openfreemap.org/, accessed 2026-10-04) | Donation-funded public instance with no SLA |
| MapLibre + self-hosted Protomaps PMTiles in object storage | Fully under our control. Cheap (one file covering Telangana) | We maintain extract updates |
| tile.openstreetmap.org | Free | **Not suitable**: the OSMF policy says commercial services' "access may be withdrawn at any point", no SLA, and heavy use is blocked (https://operations.osmfoundation.org/policies/tiles/, accessed 2026-10-04) |
| Google Maps Platform | Best Indian POI and geocoding quality | Paid beyond free credits. Billing account required. Terms restrict caching. |
| Mapbox / MapTiler | Polished | Paid after free tier. Keys. |

**Decision.** **MapLibre GL JS** with **OpenFreeMap** as the primary style and tiles. **Self-hosted PMTiles (Protomaps basemap extract for Telangana) in our object storage behind the CDN** is the configured fallback, switchable by config without a code change. **No geocoding API in V1**: pin drop + locality picker (our `localities` table) + landmark + PIN code. Rider navigation hands off to the device maps app via a `geo:`/Google Maps URL (no API cost). OSM attribution is shown as the ODbL requires.

**Consequences.** + ₹0, no keys. − Address search by free text is weak (mitigated by the locality list and landmarks).

**Revisit when.** Address quality issues cause more than X% failed deliveries `[OPEN: ops metric]`. Then consider Google Places Autocomplete with a session token budget, or self-hosted Nominatim/Photon.

---

## ADR-016: Deployment: managed containers on a hyperscaler (staging/prod); Docker Compose for local

**Context.** User directive (baseline §4a, 2026-10-04): production must run on a **standard, mainstream cloud with managed services in an India region**. Free tiers and local Docker are for development, CI and demos only. The workload needs **always-on compute**: SSE streams in the API, and River workers with timers such as the 45 s offer expiry. Scale-to-zero platforms must therefore be configured with minimum instances. DevOps picks the cloud and service (docs 22/25). This ADR fixes the architectural shape and constraints.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| **Managed container service** (ECS Fargate / Cloud Run with min instances + instance billing / Azure Container Apps with min replicas) | No servers or cluster to patch. Per-task scaling. Native LB, secrets and IAM integration. Runs our OCI images unchanged. | Per-vCPU cost is higher than raw VMs. Platform-specific quirks (request timeouts, idle CPU throttling) must be checked against SSE and workers. |
| Managed Kubernetes (EKS/GKE Autopilot/AKS) | Most portable control plane. Rich ecosystem. | Cluster fee and upgrade burden. Overkill for 2–3 services. |
| VMs with Docker Compose (baseline's original P14/P17) | Cheapest. Simple. | Self-managed patching, backups, failover. **Rejected for production by the user directive.** Kept for local only. |
| PaaS free tiers (Render/Koyeb/Fly/…) | ₹0 | Sleep/time limits break SSE and workers. Non-standard for production. **Dev/preview only** (doc 25). |
| Serverless functions | Scale to zero | Long-lived SSE and timers fit poorly (ADR-001 option D) |

**Decision.**
- **Local:** Docker Compose (`deploy/compose`): Postgres+PostGIS, MinIO, Mailpit, Grafana LGTM, optional Valkey, plus fake providers (doc 08 §2.3).
- **Staging and production:** a **managed container service** on the hyperscaler chosen by DevOps. Three workloads from **one image**: `api` service (≥ 2 tasks behind a managed L7 LB), `worker` service (≥ 1 always-on task) and `migrate` one-off task per release. Managed Postgres + PostGIS (PITR, KMS encryption, private networking). Object storage + CDN for static apps and media. Secrets manager. Edge WAF. All of it is provisioned by IaC (ADR-026).
- **Kubernetes only** if the managed container service fails a hard requirement, or per the criteria in doc 08 §9.5.
- **Deploy flow:** CI builds multi-arch images once → pushes them to the cloud registry (and optionally GHCR for the open-source community) → auto-deploys to staging → production on an approved tag (GitHub environment protection), via OIDC (no long-lived keys). Migrations run first (expand/contract), then a rolling update of `api` and `worker`.
- **Runtime contract for any platform:** listens on `$PORT`; `/healthz` (liveness) and `/readyz` (DB reachable, LISTEN connected, migrations at the expected version); SIGTERM → stop accepting, send `retry:` to SSE clients, drain within 25 s; logs as JSON to stdout; config only from env/files.

**Current instantiation (doc 22):** AWS `ap-south-1`, ECS Fargate (ARM) behind ALB + CloudFront/WAF, RDS PostgreSQL, S3, Secrets Manager/KMS, with DR backups to `ap-south-2`.

**Consequences.** + Production-grade availability, backups and security controls from managed services. + The same images flow from laptop to prod. − A real monthly cloud bill (DevOps estimates it in INR, doc 25). − Platform request-timeout and CPU-throttling behaviour must be validated for SSE and workers in staging.

**Revisit when.** Cost or limits of the managed container service become a problem (then consider managed K8s), or multi-region becomes necessary.


**Amended 2026-10-04 per R23, R24, R28 and R32.**
- **Production: AWS `ap-south-1`** primary, `ap-south-2` for DR backups; ECS Fargate (ARM) `api` + `worker`; RDS PostgreSQL + PostGIS; S3 + CloudFront; WAF; Secrets Manager; KMS (R23). Alternative: GCP.
- **Two IaC profiles (R32):** `closed-pilot` = RDS **Single-AZ** db.t4g.small with PITR + cross-region automated backups, 2 small `api` tasks, 1 `worker`; `public-launch` = **Multi-AZ, mandatory before Gate B or > 100 orders/day, whichever first**, sized by the doc 20 load test.
- Closed pilot: tasks in public subnets with compensating controls (SG ingress only from the ALB, egress allow-list, VPC endpoints for S3/ECR/Secrets/Logs); one NAT Gateway before Gate B (R28).
- **Development: local Docker Compose only**; no card-requiring free tiers; demos through a card-free tunnel (R24). The "PaaS free tiers… dev/preview only" row above is superseded.

---

## ADR-017: Observability via OpenTelemetry

**Context.** We need traces across HTTP → DB → jobs → providers, RED metrics, business metrics (orders/min, time-to-accept, time-to-assign), and structured logs, without vendor lock-in.

**Options.** (A) OTel SDK + OTLP to a backend chosen by DevOps (cloud-native stack via collector, or Grafana Cloud) + Sentry for errors; (B) proprietary vendor agent (Datadog/New Relic); (C) self-hosted LGTM stack in production (ops burden); (D) logs only.

**Decision.** **A**, with the backend chosen by DevOps: the cloud-native stack (via the ADOT/OTel collector) or Grafana Cloud (P15 as updated). The app only speaks **OTLP** (ADR-025). Locally, `grafana/otel-lgtm` runs in Compose. OTel Go SDK v1.47.x: traces (`otelhttp`, pgx tracer, River middleware propagating trace context through job args), metrics (Prometheus-style via OTLP), `slog` JSON logs carrying `trace_id`/`span_id`. Sampling: 100% of errors, 10% head sampling on success in prod `[OPEN: DevOps]`. Keep metric cardinality low: no `order_id` labels; `city_id` is allowed. Sentry Go SDK for panics and errors, plus Sentry browser SDK for PWAs (sample rate tuned to budget). External uptime checks. Backend costs and limits are verified in docs 24/25.

**Consequences.** + Vendor-neutral. − Must watch free-tier quotas (series/log GB).

**Revisit when.** Telemetry cost exceeds budget. Then tune sampling and retention, or switch backend (OTLP makes that a config change).


**Amended 2026-10-04 per R36 (31 RV-032, RV-085; missing item M1).**
- Backend: **Grafana Cloud** for metrics, traces and logs, plus **Grafana Faro** for frontend errors/RUM. **No Sentry in V1** (Go or browser); re-evaluate after the pilot.
- **CERT-In compliance archive:** application, ALB/CloudFront/WAF, VPC flow (all), RDS and CloudTrail logs are kept **180 days in India** in CloudWatch Logs / S3 (`ap-south-1`) with lifecycle; security events ≥ 1 year. Grafana Cloud retention is operational only, never the compliance record.
- External uptime checks from a free, card-free service. Alert escalation (paid phone-call paging for P1) is owned by doc 24.

---

## ADR-018: Internationalisation (en + te)

**Context.** Telugu and English from day one. Menu content may have Telugu names. Formatting uses `en-IN`/`te-IN` (₹, lakh grouping). Notifications must be localised (SMS DLT templates are registered per language).

**Decision.**
- **UI strings:** i18next with ICU message format, catalogs in `packages/i18n/{en,te}/*.json` keyed by namespace. A CI check ensures every `en` key exists in `te` (missing `te` falls back to `en` at runtime, with a warning in dev). Translators work through PRs (or Weblate later `[OPEN]`).
- **Formatting:** `Intl.NumberFormat('en-IN'|'te-IN', {style:'currency', currency:'INR'})` on the client. The server returns paise integers only, never formatted strings, except in notifications.
- **Server-side text** (notifications, invoices, problem+json `title`): the Go side uses message catalogs `backend/internal/platform/i18n/{en,te}.toml` keyed by the same template keys as doc 15. Locale is chosen from the user preference, then `Accept-Language`, then the city default.
- **User-generated content:** `name` + optional `name_te` (or `translations JSONB` — Backend decides in doc 10). Search indexes both.
- **Fonts:** Noto Sans Telugu subset, loaded only when the locale is `te`.
- **Error codes** are stable machine strings (`ORDER_STATE_CONFLICT`). Clients map them to localised messages.

**Revisit when.** A third language is added. Consider a TMS (Weblate/Tolgee self-hosted).


**Amended 2026-10-04 per R17.** User-generated translatable fields are stored as `*_i18n` JSONB (e.g. `name_i18n = {"en": "...", "te": "..."}`). The API exposes `nameI18n` plus a resolved `displayName` chosen by `Accept-Language`. The `name_te` option above is superseded. Telugu romanisation for search is deferred (C17).

---

## ADR-019: UUIDv7 primary keys

**Context.** We need globally unique IDs that can be generated before insert (for events, idempotency, client correlation), with index locality and no information leakage beyond the creation time.

**Options.** `BIGSERIAL` (leaks volume, needs the DB round-trip, merges badly across cities later); UUIDv4 (random index inserts, B-tree bloat); **UUIDv7** (time-ordered, good locality); ULID (similar, non-standard type); Snowflake (needs worker IDs).

**Decision.** **UUIDv7**, generated in the app (`github.com/google/uuid` `NewV7`, via `platform/idgen` for test determinism), stored as the native `uuid` type. On PG18 (if adopted, ADR-005), column defaults can also be `uuidv7()` as a safety net (https://www.postgresql.org/docs/18/functions-uuid.html, accessed 2026-10-04). Human-facing **order code** `RV-XXXXXX`: 6 chars of Crockford base32 (≈ 1.07 billion space), randomly generated with a unique index and retry on collision. It is not derived from the UUID, to avoid leaking timing.

**Consequences.** + Sortable by creation time, so cursor pagination is easy. − UUIDv7 reveals creation time to anyone holding the ID (acceptable; IDs are not secrets and authorization is always checked).

---

## ADR-020: Multi-city data model: shared schema with a `city_id` discriminator

**Context.** One city at launch, more later. Cities are not separate tenants (one operator), but admin access is city-scoped, configuration differs per city, and GST registration is per state.

**Options.** (A) shared tables with a `city_id` column; (B) schema per city; (C) database per city; (D) Postgres Row-Level Security keyed on a session variable.

**Decision.** **A.** `city_id` (FK to `cities`) is on all city-scoped tables (listed in doc 08 §10). All list queries take `city_id` as an explicit parameter. Admin RBAC scope is enforced in middleware (doc 12). Composite indexes lead with `city_id` where queries filter by city. **RLS is not used in V1.** It adds complexity with pooled connections and `SET LOCAL` handling for marginal gain when tenants are not adversarial. Reconsider for admin-scope defence in depth `[OPEN: Security]`. Customers are global (not city-bound). Addresses and orders carry `city_id`.

**Consequences.** + Cross-city reporting is trivial. A new city is a config task. − A forgotten `city_id` filter is a logic bug, not an isolation breach. Mitigated by query review and tests with two seeded cities in CI (doc 20).

**Revisit when.** A city requires data residency separation, or volumes call for sharding.

---

## ADR-021: Search: PostgreSQL full-text search + pg_trgm

**Context.** Customers search restaurants and dishes in English and Telugu, with typos and transliterations ("biryani", "biriyani", "బిర్యానీ").

**Decision.** A Postgres generated `tsvector` (simple config, no stemming, since Telugu has no built-in dictionary) over names and tags, plus a **`pg_trgm`** GIN index for fuzzy matching, plus a small synonyms table for common transliterations `[OPEN: seed list]`. Results are filtered by serviceability first, so the candidate set per location is small (tens to hundreds of restaurants). No Elasticsearch, Meilisearch or Typesense in V1.

**Revisit when.** Catalog > ~50k items per city or relevance complaints. Then consider Meilisearch (self-hosted) fed by events.

---

## ADR-022: File uploads via presigned URLs to object storage

**Context.** We store menu and restaurant images, rider KYC documents (sensitive), FSSAI certificate images, generated invoices and statements, and DB backups.

**Decision.** S3-compatible API (the cloud's object storage in staging/prod; MinIO locally; see ADR-025 for GCS/Azure compatibility). The API issues **presigned PUT URLs** (5 min expiry, content-type and max-size constrained) and records the object key after a client confirmation call that verifies size and type with a `HEAD` request. Two buckets: `public-media` (served via the CDN on `img.<domain>`, immutable versioned keys) and `private-docs` (KYC, invoices; only short-lived presigned GETs for authorised viewers; never public). **Images are resized and compressed client-side** (canvas → WebP/JPEG, max 1200 px, ≤ 300 KB) before upload. That means no server image pipeline and no cgo/libvips. The server rejects files over the limit. Store only masked Aadhaar if any is ever collected (baseline §3; preferably do not collect Aadhaar at all) `[LEGAL]`.

**Revisit when.** We need multiple renditions or AVIF. Then add a worker job with a pure-Go or libvips pipeline, or Cloudflare Images (paid).


**Amended 2026-10-04 per R38.** **KYC files are images only** (JPEG/PNG/WebP; the client converts PDFs and photos). The server re-encodes KYC images, stores them with SSE-KMS at rest, shows them through an audited streaming view with short-TTL signed URLs. **No ClamAV and no app-layer envelope encryption for files** in V1. Field-level encryption for bank account numbers and TOTP secrets stays.

---

## ADR-023: API conventions

**Decision (details in doc 11).**
- Path versioning: `/v1/...` for app APIs, `/admin/v1/...` for admin, `/webhooks/...` for providers. Breaking changes mean `/v2` for the affected resources only. Additive changes are allowed in v1.
- Errors: **RFC 9457 `application/problem+json`** with a stable `code` extension, `trace_id`, and `errors[]` for field validation.
- Pagination: cursor-based (`?cursor=…&limit=…`, the opaque cursor encodes the UUIDv7/time key). No offset pagination on large tables.
- `Idempotency-Key` header on POSTs (doc 08 §7.1). `ETag`/`If-Match` on mutable aggregates (orders, menus).
- Timestamps in RFC 3339 UTC. Money is integer paise plus `currency`. IDs are UUID strings. Enums are UPPER_SNAKE matching DB values.
- Rate-limit headers (`RateLimit-*` per the IETF draft) on throttled endpoints.


**Amended 2026-10-04 per R15.** The base path is **`/api/v1` on every host** (app hosts, admin host and the reserved `api.` host). Admin operations are under `/api/v1/admin/*`; provider webhooks under `/api/v1/webhooks/*`. The `/v1/...`, `/admin/v1/...` and `/webhooks/...` forms above are superseded.

---

## ADR-024: CI/CD on GitHub Actions with OIDC to the cloud

**Context and decision.** P16 (as updated) is confirmed: GitHub Actions (free for public repos) runs lint, test, build, scan, boundary checks (doc 08 §4.3), OpenAPI drift and oasdiff, sqlc vet, migration lint, `tofu plan` on IaC changes, multi-arch image build and push to the cloud registry (and GHCR), auto-deploy to staging, and production deploy on an approved tag. **Cloud authentication via GitHub OIDC federation**, so there are no long-lived cloud keys in GitHub secrets. DevOps owns the details in doc 21. Supply chain: pin actions by SHA, Dependabot/Renovate, `govulncheck`, `pnpm audit`, SBOM (syft) and image scan (trivy or grype).

**Revisit when.** Runner minutes or secrets handling becomes limiting (e.g. if the repo turns private).

---

## ADR-025: Cloud portability through standard interfaces only

**Context.** Baseline §4a requires production on a mainstream cloud, with the choice of cloud left to DevOps. The project is open source, so others will deploy it on different clouds or on-prem. Lock-in usually creeps in through SDK calls in domain code and proprietary services (queues, auth, document DBs).

**Options.** (A) portable: standard protocols only, provider adapters behind interfaces; (B) cloud-native: use the provider's queues, auth, secrets SDKs and serverless directly; (C) a multi-cloud abstraction framework (e.g. Dapr).

**Decision.** **A.** The application depends **only** on:

| Concern | Standard interface | Local | Production examples |
|---|---|---|---|
| Relational DB + geo | PostgreSQL wire protocol, PostGIS extension, `LISTEN/NOTIFY` | `postgis/postgis` container | RDS/Aurora PostgreSQL, Cloud SQL for PostgreSQL, Azure Database for PostgreSQL Flexible Server |
| Queue | River on that same Postgres (no cloud queue) | same | same |
| Object storage | **S3 API** (SigV4, presigned PUT/GET) via `platform/blob` | MinIO | S3, GCS (XML API interoperability + HMAC keys), Azure Blob (needs a native `blob` adapter or gateway `[OPEN: only if Azure chosen]`) |
| Cache / rate limit / pub-sub (optional) | **Redis protocol** (RESP) | Valkey | ElastiCache/MemoryDB, Memorystore, Azure Cache/Managed Redis |
| Telemetry | **OTLP** (gRPC/HTTP) | otel-lgtm | ADOT → CloudWatch/X-Ray, Google Cloud Ops via collector, Azure Monitor via collector, or Grafana Cloud |
| Packaging | **OCI images** (multi-arch) | Docker | any managed container service / K8s |
| Config | **env vars + mounted files** (12-factor). Secrets injected at runtime by the platform. | `.env` | Secrets Manager / Secret Manager / Key Vault references |
| Email | SMTP **or** provider HTTP adapter behind `notify.EmailSender` | Mailpit | SES (SMTP), Resend, Brevo |
| Edge | Plain HTTP semantics: path routing, cache headers | Vite proxy / Caddy | CloudFront, Cloud CDN, Front Door, Cloudflare |

Rules:
1. **No cloud SDK imports** in `internal/modules/**`. Cloud- or vendor-specific code lives only in `internal/adapters/<kind>/<vendor>` packages, wired in `cmd/rovo` from config. The depguard rule bans `github.com/aws/*`, `cloud.google.com/*` and `github.com/Azure/*` outside `internal/adapters/**`.
2. No dependence on proprietary managed features where a standard one exists (e.g. no DynamoDB, Pub/Sub, Cognito or Firebase Auth in V1).
3. IAM-based DB auth or secret-based passwords are handled in `platform/db` config. The rest of the app sees a DSN.
4. CI runs the full integration suite against the **local standard implementations** (Postgres, MinIO, Valkey). Each production adapter gets a contract test that runs in staging.
5. Infrastructure differences live only in Terraform modules (ADR-026).

**Consequences.** + DevOps can pick or switch clouds without app changes. Self-hosters can deploy. − We forgo some managed conveniences (e.g. a managed queue). Minor S3-compatibility gaps must be tested per provider.

**Revisit when.** A proprietary service offers a decisive benefit (e.g. a managed geo-routing API). Add it as an adapter behind an interface, never in domain code.


**Amended 2026-10-04 per 31 RV-074.** The realistic claim is: **the application is portable; rebuilding the infrastructure on another cloud takes weeks**, not hours. The OpenTofu code, CloudFront flat-rate plan, WAF rules, ECS and RDS backups are AWS-specific. A cross-cloud restore (doc 23 DR-4) has no RTO commitment unless a one-off GCP restore rehearsal is funded before Gate B.

---

## ADR-026: Infrastructure as Code with OpenTofu (Terraform-compatible)

**Context.** Staging and production must be reproducible, reviewable and identical in shape (§4a rule 2). Local uses Compose.

**Options.**

| Option | Pros | Cons |
|---|---|---|
| **OpenTofu** (MPL-2.0, Linux Foundation fork of Terraform) | Open-source licence that fits an Apache-2.0 project. HCL- and provider-compatible with Terraform. Supports state encryption. | Slightly smaller ecosystem of commercial tooling |
| Terraform (BSL 1.1 since 2023) | Largest ecosystem | BSL licence restrictions on some uses. Fine for end users, but awkward to mandate in an OSS project. `[ASSUMPTION: licence status as of 2026 not re-verified]` |
| Pulumi (Go/TS) | Real languages | Smaller community for reviewers. State service. |
| Cloud-native (CloudFormation/CDK, Deployment Manager, Bicep) | Deep integration | Single-cloud. Conflicts with ADR-025. |

**Decision.** **OpenTofu**, keeping the code **Terraform-compatible** (no OpenTofu-only syntax unless needed, e.g. state encryption) so either CLI works. Layout `deploy/terraform/` (doc 26):
- `modules/`: `network`, `db`, `cache` (disabled by default), `storage`, `cdn` (+WAF), `compute` (api, worker, migrate), `observability`, `secrets`, `registry`, `dns`. Each module has a provider-specific implementation under `modules/<name>/<cloud>` or is written for the single chosen cloud. DevOps decides `[OPEN]`, with the recommendation to start single-cloud and avoid premature multi-cloud abstraction.
- `envs/staging`, `envs/prod`: thin compositions with tfvars. Remote state in the cloud's object storage with locking and encryption. One state per env.
- CI: `tofu fmt -check`, `validate`, `tflint`, `checkov`/`trivy config` on PRs. `plan` is posted as a PR comment. `apply` to staging on merge, to prod with manual approval. Authentication via OIDC.
- Drift detection: a scheduled `plan` job weekly.
- **Not in IaC:** application config values that change at runtime (pricing, zones), which live in the DB and admin UI. Secrets values are created out-of-band in the secrets manager; IaC creates the secret *containers* only.

**Consequences.** + Reviewable, reproducible environments. Disaster recovery (DR) can rebuild from code (doc 23). − Needs IaC skills on the team.

**Revisit when.** Moving to K8s (add Helm/Kustomize for workloads, keep OpenTofu for cloud resources).


**Amended 2026-10-04 per R23 and R32 (scope cut C18).** The IaC root is **`deploy/terraform/`** (not `infra/`). Profiles `closed-pilot` and `public-launch` are tfvars sets applied to the prod environment (`deploy/terraform/profiles/*.tfvars`), so moving to Multi-AZ is a reviewed plan, not a rewrite. AWS organisation: 4 accounts (mgmt, prod, nonprod, audit/backup), not six.

---

## ADR-027: Cross-module foreign keys on money paths

**Context.** Doc 08 §4.1 originally allowed foreign keys across module boundaries only to `cities` and `users`, so a module could be extracted later without untangling constraints. That gave up referential integrity on money paths (`payments.order_id`, `refunds.order_id`, `deliveries.order_id`, `invoices.order_id`, `ledger_postings.order_id`) for a speculative extraction that no plan foresees (31 RV-006). A nightly orphan check was the backstop.

**Options.** (A) No cross-module FKs except `cities`/`users`; (B) FKs to `orders` from the money-path tables; (C) FKs everywhere.

**Decision (R41).** **B.** FKs are allowed on money paths: `payments`, `refunds`, `deliveries`, `invoices` and ledger references → `orders`. Otherwise FKs exist only within a module and to `cities`/`users`. FKs do not grant query access: the Go import rules and table ownership still apply.

**Consequences.** + The database rejects orphan money rows. − Extracting `ordering` later needs a migration that drops the FKs (one line each). − Migration order must create `orders` before the dependent tables.

**Revisit when.** A module is actually extracted (doc 08 §9.5).

---

## ADR-028: API edge: same-origin `/api/*` on every app host; no CORS; `api.` reserved

**Context.** Doc 22 originally served the API from an `api.` host directly on the ALB (to stay under the CloudFront flat-rate request allowance) and doc 24 then needed CORS on `api.`. Docs 11, 12 and 17 assume same-origin `/api/*`, host-only cookies and no CORS. An ALB cannot inject an audience header, so the API would have to trust a client-settable header (31 RV-001, RV-025).

**Options.** (a) `/api/*` on every app host through CloudFront flat-rate Pro; (b) same, on CloudFront pay-as-you-go; (c) a public `api.` host with CORS.

**Decision (R14, R27).** **(a)**, with **(b)** as the fallback if the request allowance is exceeded two months running. Every app host (`app.`, `restaurant.`, `rider.`, `admin.`) has a CloudFront behaviour for `/api/*` (caching off, cookies forwarded) to the ALB. The ALB accepts only the CloudFront origin-facing prefix list plus a secret origin-verify header. The API derives the audience from `Host` and ignores `X-Rovo-Audience` unless the origin secret is present. **No public `api.` host with CORS in V1**; `api.` stays reserved for future native bearer clients and server-to-server provider webhooks (no cookies). Chatty traffic is reduced: batched rider pings, restaurant heartbeat every 60 s, SSE presence counts as heartbeat.

**Consequences.** + One cookie and CSRF model; no CORS; audience cannot be forged through the edge. − API traffic counts against the CDN request allowance (doc 25 models it, M13). − Each app host needs its own `/api/*` behaviour and WAF association.

**Revisit when.** A native app launches (bearer clients on `api.`), or CDN request volume makes pay-as-you-go cheaper than the flat-rate plan.

---

## Appendix: challenges to the baseline (summary for the Lead Architect)

1. **P4 refined:** River's transactional insert is the outbox. No custom relay (ADR-006).
2. **P5 clarified:** PostgreSQL **18 preferred, 17 acceptable**, matched to managed-service GA availability. No PG18-only code dependencies.
3. **P7 app split:** `restaurant` + `rider` instead of a combined `partner` app (doc 17 F2, endorsed in ADR-009). Static hosting is object storage + CDN with same-origin `/api`.
4. **P9 refined:** hybrid EdDSA JWT + server-side session checks for partner/admin, aligned with doc 12 (ADR-011).
5. **P10 changed:** the PA is selected on the written UPI rate and launch offers (Razorpay 2% vs Cashfree 1.95% with a 0% launch allowance; 0%-UPI claims for PhonePe/Paytm unverified). Restaurant money should flow via PA **split settlement**, not collect-then-manual-payout, pending legal opinion (ADR-012, doc 14).
6. **P13 sharpened:** OpenFreeMap / self-hosted PMTiles. **Not** `tile.openstreetmap.org` (ADR-015).
7. **P14/P17 replaced by user directive §4a:** managed containers + managed Postgres on a hyperscaler for staging/prod, Compose for local (ADR-016). New ADR-025 (portability) and ADR-026 (IaC).
8. **P6 confirmed:** no Redis at launch. Managed Redis is enabled by config only on a trigger (ADR-007).
9. All other provisional decisions are confirmed.
10. **v1.1 (2026-10-04):** the Lead's rulings R1–R48 supersede items 2 (PG 17, R22), 3 (four apps, no prerender, R14/R33) and 5 (both money-flow models supported, legal opinion before Phase-2 week 4, R35). See the amendment blocks on each ADR and the new ADR-027/028.
