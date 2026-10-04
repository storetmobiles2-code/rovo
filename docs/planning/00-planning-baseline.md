# 00 — Planning Baseline (Lead Architect Brief)

> Status: **Phase 1 — Planning only.** No application code is written in this phase.
> Every planning agent reads this file first. It fixes the vocabulary and the *provisional* decisions so that 30 documents written in parallel stay consistent. Any agent may challenge a provisional decision, but must do so explicitly (section "Challenges to baseline" in its report) rather than silently diverge.

## 1. Product framing

- **Name:** rovo (lowercase in text, `rovo` in code). Open source, Apache-2.0.
- **Launch market:** Mahabubnagar (Mahbubnagar / Palamuru), Telangana, India. Single city at V1. Pop. roughly 2–2.5 lakh urban area [ASSUMPTION – verify]. Small compact city, two-wheeler deliveries, high Android share, price sensitive, UPI-heavy but with real COD demand.
- **Inspiration:** UX patterns and workflows of Zomato, Swiggy, DoorDash. No proprietary code, branding, assets, copy or screen designs are copied.
- **V1 golden flow (must work end-to-end):**
  `Customer → Restaurant → Order → Delivery Partner → Pickup → Delivery → Payment/Settlement → Rating`
- **Explicitly deferred:** multi-city operations, AI recommendations, loyalty, subscriptions, advanced analytics, complex route optimisation, live GPS fleet tracking, native mobile apps, unnecessary microservices.

## 2. Canonical vocabulary (use these exact names)

| Concept | Code name | UI name (en) |
|---|---|---|
| End customer | `customer` | Customer |
| Restaurant | `restaurant` (a.k.a. outlet; one row per physical outlet) | Restaurant |
| Restaurant owner / staff | `restaurant_owner`, `restaurant_staff` | Partner |
| Delivery partner | `rider` | Delivery Partner |
| Platform operator | `admin` with sub-roles | Admin |
| City | `city` | City |
| Delivery zone (polygon) | `zone` | Zone / Area |
| Locality / neighbourhood (named, for addresses & search) | `locality` | Locality |
| Delivery job for an order | `delivery` (task) | Delivery |
| Offer of a delivery to a rider | `delivery_offer` | Order request |

**Roles (RBAC):** `CUSTOMER`, `RESTAURANT_OWNER`, `RESTAURANT_STAFF`, `RIDER`, `ADMIN_SUPER`, `ADMIN_OPS`, `ADMIN_SUPPORT`, `ADMIN_FINANCE`. Admin roles are scoped by `city_id` (nullable = all cities).

**Order status (canonical, `orders.status`):**
`PENDING_PAYMENT → PLACED → ACCEPTED → PREPARING → READY_FOR_PICKUP → PICKED_UP → DELIVERED`
Terminal/side states: `PAYMENT_FAILED`, `REJECTED` (restaurant), `CANCELLED` (customer/admin/system, with `cancel_reason` + `cancelled_by`), `UNDELIVERABLE` (rider could not deliver).
COD orders skip `PENDING_PAYMENT` and go straight to `PLACED`.

**Delivery status (canonical, `deliveries.status`), separate from order status:**
`UNASSIGNED → OFFERED → ASSIGNED → AT_RESTAURANT → PICKED_UP → AT_DROP → DELIVERED`
Side states: `FAILED`, `CANCELLED`. Offers have their own status: `PENDING | ACCEPTED | DECLINED | EXPIRED`.

Backend Architect owns the full transition tables; other docs must reference these names.

## 3. Cross-cutting conventions

- **Money:** integer **paise** (`BIGINT`), column suffix `_paise`; every monetary row also carries `currency CHAR(3) DEFAULT 'INR'` where it is a ledger/order-level amount. Never floats. Rounding rule: round half up to the nearest paisa at line level; displayed totals round to the rupee only if product decides (OPEN).
- **IDs:** UUIDv7 primary keys generated in the app. Human-facing order code: short base32 code e.g. `RV-7K3P9Q`.
- **Time:** `timestamptz` stored in UTC; city has `timezone` (`Asia/Kolkata`); restaurant operating hours stored as local wall-clock times + day-of-week, evaluated in city timezone.
- **Phone numbers:** E.164 (`+91XXXXXXXXXX`), validated as Indian mobile (10 digits starting 6–9) at V1; library-based validation (libphonenumber port) so other countries can be added.
- **Addresses (India):** house/flat no., building/street, area/locality (FK to `locality` where possible), landmark (strongly encouraged — Indian addressing relies on landmarks), city, state, PIN code (6 digits), lat/lng from map pin (required for delivery), label (Home/Work/Other), contact name/phone override.
- **Geo:** PostgreSQL + **PostGIS**. Zones are polygons; serviceability = customer pin inside an active zone AND within restaurant's max delivery radius. Distances V1 = straight-line (haversine) × configurable road-factor (default 1.3). No paid routing API in V1.
- **Multi-city readiness:** `cities` table from day one; `city_id` on zones, localities, restaurants, riders, orders, pricing configs, coupons, admin scopes. Single city seeded at launch. No hard-coded city strings in code.
- **i18n:** English + Telugu (`en`, `te`) from day one in UI string catalogs; user-generated content (menu names) has optional `name_te` (translations JSONB or companion columns — Backend decides). Number/currency formatting via `Intl` with `en-IN` / `te-IN` (₹, lakh grouping).
- **Regulatory flags (must be addressed by relevant docs, legal review required before launch):** FSSAI licence number required for each restaurant; GST — food delivery via e-commerce operator under CGST §9(5) and current GST rates on platform/delivery fees (verify current rates); TDS/TCS obligations of e-commerce operators (verify); DPDP Act 2023 + DPDP Rules (consent, purpose limitation, retention, grievance officer, breach notification); TRAI DLT registration for SMS; RBI rules for payment aggregators (we use a licensed PA, we do not hold funds outside it); Aadhaar must not be stored in full.

## 4. Provisional architecture decisions (agents may challenge)

| # | Decision | Lean | Rationale (short) |
|---|---|---|---|
| P1 | Overall style | **Modular monolith** in Go, one binary with `api` and `worker` run modes | Team of few, single city, low ops budget. Module boundaries enforced by package layout + no cross-module table access. |
| P2 | API style | **REST/JSON, OpenAPI 3.1 spec-first**; codegen server interfaces (Go) and typed TS client | One contract for web + future native apps. **No gRPC in V1** (no internal network boundary yet). |
| P3 | Real-time | **Server-Sent Events** for order/delivery status, restaurant order inbox, rider offers; polling fallback; Web Push for background alerts | Simpler than WebSockets, works through proxies/CDNs, one-way is enough. |
| P4 | Async/events | **Transactional outbox in Postgres + Postgres-backed job queue (River)**; in-process domain events | No Kafka/NATS/RabbitMQ in V1. Exactly-once-ish via idempotent handlers. |
| P5 | Database | **PostgreSQL (17+) with PostGIS**, `pgx` + `sqlc`, migrations with `goose` | Relational integrity for money and orders; PostGIS for zones. |
| P6 | Redis | **Not required at V1 launch.** Interfaces (`RateLimiter`, `PubSub`, `Cache`) with in-memory/Postgres implementations; Redis-protocol adapter (managed ElastiCache/Memorystore/Azure Cache, or Valkey locally) enabled when >1 API replica or proven need | Fewer moving parts; production cloud makes adding it a config change. Solution Architect to confirm or overturn. |
| P7 | Frontend | **React + TypeScript SPA/PWA built with Vite**, deployed as static assets to object storage + CDN (cloud-native, e.g. S3+CloudFront / GCS+Cloud CDN; Cloudflare acceptable in front); three apps: `customer`, `partner` (restaurant + rider), `admin`, in a pnpm workspace with shared packages. **Next.js evaluated and deferred** | No server runtime for frontend → free, portable, no Vercel Hobby non-commercial restriction, no edge-CPU limits. SSR/SEO revisited later. Frontend Architect to confirm or overturn with evidence. |
| P8 | UI stack | TanStack Router + TanStack Query, Tailwind CSS, Radix-based components (shadcn/ui style, copied in), i18next, vite-plugin-pwa (Workbox), openapi-typescript + openapi-fetch | Mature, small, typed. |
| P9 | Auth | Phone + OTP for customer/rider/restaurant users; admin = email + password + mandatory TOTP. Short-lived access JWT (≈15 min) + rotating opaque refresh token (hashed in DB), httpOnly Secure cookies for web; bearer tokens for future native apps | OTP delivery provider is a paid line item (DLT); provider abstracted. |
| P10 | Payments | **COD + one Indian payment aggregator (Razorpay lean; Cashfree/PhonePe PG as alternatives)** via provider interface; webhook-driven confirmation; platform collects, then settles restaurants/riders from an internal **ledger** (weekly payouts, manual bank/UPI transfer in V1, automated split/route later) | Keep regulated money movement with a licensed PA. |
| P11 | Dispatch | Rule-based: offer to best available rider (online, fresh location, nearest to restaurant, fewest active jobs), 45 s offer timeout, cascade to next; admin manual assign fallback; one active delivery per rider in V1 (batching deferred) | No route optimisation in V1. |
| P12 | Rider location | Foreground-only geolocation from rider PWA while online (≈ every 30–60 s) used for dispatch; customer sees **status milestones**, not a live map | PWAs cannot track in background; live GPS deferred. |
| P13 | Maps | MapLibre GL JS with a free/open tile source (verify terms) for pin-drop; no paid geocoding in V1 (pin + structured fields + locality list) | Cost. Google Maps Platform kept as option. |
| P14 | Containers | Docker images for everything (OCI, multi-arch amd64+arm64); 12-factor config; health/readiness endpoints; graceful shutdown. **Local dev = Docker Compose.** **Production = managed container platform on a standard hyperscaler** (see §4a); Kubernetes-ready (stateless API, separate worker deployment, migrations as a one-off job) but K8s only if the chosen managed service needs it | Same images run on a laptop and in the cloud. |
| P15 | Observability | OpenTelemetry SDK (traces + metrics), `slog` JSON logs with trace IDs, exported via OTLP. **Production:** the cloud's native stack (e.g. CloudWatch/X-Ray via ADOT, or Cloud Logging/Trace/Monitoring) or Grafana Cloud — decided by DevOps; error tracking (Sentry lean) + external uptime checks. Dev: local Grafana LGTM container or free tiers | Vendor-neutral via OTLP so the backend can change. |
| P16 | CI/CD | GitHub Actions: lint, test, build, scan, push images to the cloud's registry (and/or GHCR), deploy to staging automatically and production on approved tag; **OIDC federation to the cloud (no long-lived cloud keys)**; infrastructure as code (Terraform/OpenTofu) | Standard, auditable. |
| P17 | Hosting | **Superseded by §4a (user directive).** Production runs on a standard, mainstream cloud with managed services in an India region. Free tiers / local Docker are for development, CI, preview and demo only. | Production must not depend on non-standard or free-tier hosting. |

## 4a. Environment & hosting strategy (USER DIRECTIVE, 2026-10-04 — overrides P17)

> "We should consider this as a production ready app that can be deployed to cloud as an option once development is complete so we don't use non-standard hosting for production and live app. For development we can use a free env or even local docker until it is fully developed and ready for going live."

| Environment | Where | Purpose |
|---|---|---|
| **local** | Docker Compose on the developer machine (Postgres+PostGIS, optional Valkey, MinIO for S3-compatible storage, Mailpit, fake OTP & fake payment providers, local OTel/Grafana LGTM) | Day-to-day development; must run the full golden flow offline with fakes |
| **ci** | GitHub Actions ephemeral containers | Tests, builds, scans |
| **dev/preview (optional)** | **Local Docker Compose only** (user directive 2026-10-04: no free tier that asks for credit-card details). Shared demos: run the local stack and expose it temporarily through a card-free tunnel | Demos, stakeholder previews |
| **staging** | Same cloud, same IaC as production, scaled down (can be stopped when idle) | Pre-prod validation, PA sandbox, UAT |
| **production** | **Standard hyperscaler, India region** (AWS ap-south-1 Mumbai / ap-south-2 Hyderabad, GCP asia-south1 Mumbai / asia-south2 Delhi, or Azure Central India / South India — DevOps recommends one primary + one alternative) using **managed services**: managed container runtime, managed PostgreSQL with PostGIS, managed Redis-compatible cache (if needed), object storage, CDN, secrets manager, KMS, managed backups/PITR, WAF | Live business with real money and personal data |

Rules that follow from this:
1. **Cloud-portable by construction:** the app depends only on standard interfaces — PostgreSQL wire protocol + PostGIS, S3-compatible object storage API, Redis protocol (optional), OTLP, OCI containers, env-var config, secrets injected at runtime. No provider-specific SDK in domain code; provider adapters live behind interfaces.
2. **Infrastructure as Code** (Terraform/OpenTofu) for staging and production; local uses Compose. The same container images flow local → CI → staging → production.
3. **Production non-negotiables:** managed Postgres with automated backups + PITR and Multi-AZ (or a documented single-AZ-at-pilot decision with upgrade trigger), TLS everywhere, private networking for DB/cache, secrets manager, KMS-backed encryption, WAF/rate limiting at the edge, centralised logs/metrics/traces with alerting, data residency in India (DPDP-friendly), cost budgets & alerts.
4. The free-hosting comparison (doc 25) is still produced, but **scoped to dev/preview/demo use and kept for reference**: development uses **local Docker only** (no card-requiring free tiers, user directive 2026-10-04), and paired with a production cloud comparison and a monthly cost estimate in INR.
5. Background workers (River) and SSE need always-on compute in production — choose services accordingly (e.g. ECS Fargate services, GKE Autopilot, Cloud Run with instance-based billing/min instances, Azure Container Apps with min replicas) and document the trade-offs.

## 5. Indicative commercial defaults (all configurable per city/zone/restaurant; validate with local market)

- Delivery fee slabs (customer): 0–2 km ₹20 · 2–4 km ₹30 · 4–6 km ₹40 · 6–8 km ₹50; default max radius 7 km. Free-delivery threshold optional per campaign.
- Platform fee: ₹5 per order (flat, configurable). Small-cart fee: ₹15 below ₹149 subtotal.
- Packaging charge: set by restaurant per item or per order.
- Restaurant commission: default 15% of food subtotal (contract range 10–25%), configurable per restaurant with effective dates.
- Rider pay: ₹25 base per delivery + ₹6/km beyond 2 km (restaurant→customer), + waiting-time pay after 10 min at restaurant; incentives deferred.
- COD: rider collects cash; rider cash-in-hand tracked in ledger; configurable cash limit (default ₹2,000) after which rider is blocked from new COD orders until deposit is recorded.
- Payouts: weekly (restaurants), weekly or on-demand (riders) — manual transfer in V1, recorded in ledger with reference numbers.

## 6. Deliverable map and owners

| # | File | Owner agent |
|---|---|---|
| 01 | `01-product-requirements.md` | Product Architect |
| 02 | `02-v1-scope.md` | Product Architect |
| 03 | `03-user-personas.md` | Product Architect |
| 04 | `04-customer-journey.md` | UX Architect |
| 05 | `05-restaurant-workflow.md` | UX Architect |
| 06 | `06-delivery-workflow.md` | UX Architect |
| 07 | `07-admin-workflow.md` | UX Architect |
| 08 | `08-system-architecture.md` | Solution Architect |
| 09 | `09-architecture-decision-records.md` | Solution Architect |
| 10 | `10-database-schema.md` | Backend Architect |
| 11 | `11-api-specification.md` | Backend Architect |
| 12 | `12-auth-rbac.md` | Security Architect |
| 13 | `13-order-state-machine.md` | Backend Architect |
| 14 | `14-payment-architecture.md` | Solution Architect |
| 15 | `15-notification-architecture.md` | Solution Architect |
| 16 | `16-delivery-zone-architecture.md` | Backend Architect |
| 17 | `17-frontend-architecture.md` | Frontend Architect |
| 18 | `18-mobile-pwa-strategy.md` | Frontend Architect |
| 19 | `19-security-threat-model.md` | Security Architect |
| 20 | `20-testing-strategy.md` | QA Architect |
| 21 | `21-cicd-strategy.md` | DevOps Architect |
| 22 | `22-deployment-architecture.md` | DevOps Architect |
| 23 | `23-backup-disaster-recovery.md` | DevOps Architect |
| 24 | `24-observability-strategy.md` | DevOps Architect |
| 25 | `25-free-hosting-comparison.md` | DevOps Architect |
| 26 | `26-repository-structure.md` | Solution Architect |
| 27 | `27-implementation-backlog.md` | Release Architect |
| 28 | `28-milestones-dependencies.md` | Release Architect |
| 29 | `29-production-readiness-checklist.md` | Release Architect |
| 30 | `30-risks-assumptions-decisions.md` | Release Architect |
| 31 | `31-review-report.md` | Reviewer |
| 32 | `32-final-plan-summary.md` (A–J) | Lead Architect |
| 33 | `33-phase2-implementation-prompt.md` | Lead Architect |

## 7. Writing rules for all agents

1. Markdown, GitHub-renderable. Diagrams in Mermaid (`erDiagram`, `stateDiagram-v2`, `sequenceDiagram`, `flowchart`).
2. Illustrative snippets (SQL DDL, OpenAPI excerpts, config) are allowed **as design artifacts inside docs**. Do **not** create application source files, build files, Dockerfiles, workflows, or anything outside `docs/planning/`.
3. Tag assumptions `[ASSUMPTION]`, open questions `[OPEN]`, and anything needing legal/tax review `[LEGAL]`.
4. Cite sources (URL + date accessed) for any external factual claim — especially pricing and free-tier terms. If you cannot verify a claim, say so.
5. Prefer the simplest design that satisfies V1 while not blocking multi-city expansion. Say what you are *not* doing and why.
6. Every doc starts with: purpose, owner, status (`Draft v1`), dependencies on other docs.
7. Do not commit to git. The Lead Architect commits.
