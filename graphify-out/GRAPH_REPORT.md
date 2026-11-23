# Graph Report - rovo  (2026-11-16)

## Corpus Check
- 430 files · ~502,938 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 27 file(s) not represented in the graph (top: (none) 9, .css 6, .caddy 5)

## Summary
- 5667 nodes · 11008 edges · 355 communities (290 shown, 65 thin omitted)
- Extraction: 97% EXTRACTED · 3% INFERRED · 0% AMBIGUOUS · INFERRED: 327 edges (avg confidence: 0.87)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `eda0500f`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- schema.d.ts
- ui/src/index.ts
- testing.T
- money.ts
- shell/src/index.ts
- 10 — Database Schema (logical + physical)
- api-client/package.json
- react.tsx
- auth/package.json
- rider
- app.go
- Middleware
- 12 — Authentication & Authorisation (Auth / RBAC)
- 14 — Payment Architecture
- NewDB
- 16 — Delivery Zone & Geography Architecture
- 06 — Delivery Partner (Rider) Workflow
- web/package.json
- Env
- data.go
- 18 — Mobile & PWA Strategy
- i18n/package.json
- customer/package.json
- restaurant/package.json
- rider/package.json
- github.com/google/uuid.UUID
- time.Time
- 13 — Order & Delivery State Machines
- money.ts
- Module
- 07 — Admin Console Workflow
- 05 — Restaurant Partner Workflow
- context.Context
- ui/package.json
- Bus
- 22 — Deployment Architecture
- 09 — Architecture Decision Records
- 2. Backlog by epic
- admin/package.json
- dependencies
- time.Duration
- net/http.Request
- Audience
- Client
- Run
- New
- obs.go
- 24 — Observability Strategy
- 2. Milestones and exit criteria
- 31 — Review Report (Red Team)
- gen-brand-assets.py
- testing/package.json
- compilerOptions
- 29 — Production Readiness Checklist (Go-Live)
- devDependencies
- 02 — V1 Scope
- 15 — Notification Architecture
- 21 — CI/CD Strategy
- 23 — Backup & Disaster Recovery
- (a) Risk register
- client.ts
- time.Duration
- catalog_integration_test.go
- devDependencies
- devDependencies
- devDependencies
- auth/src/index.ts
- newIssuer
- .UpdateMe
- utils/package.json
- minio-init
- Middleware
- 11. Business rules (BR-*)
- Env
- 03 — User Personas
- 3.2 Module catalogue
- lib.sh
- 01 — Product Requirements Document (PRD)
- 5. Functional requirements — Customer (CUS-*)
- 8. Functional requirements — Admin (ADM-*)
- config/package.json
- gen-api.mjs
- Publisher
- 8. Privacy and regulatory (DPDP, IT Act, CERT-In, RBI/PCI, Aadhaar) — `[LEGAL]` throughout
- 9. Security requirements checklist (testable)
- Publisher
- shell/src/index.ts
- eslint.js
- dependencies
- devDependencies
- 26 — Repository Structure
- 4. Tokens, sessions, cookies, CSRF, CORS, SSE
- 4. Tokens, sessions, cookies, CSRF, CORS, SSE
- vite.ts
- Deps
- Contributing to rovo
- 00 — Planning Baseline (Lead Architect Brief)
- 04 — Customer Journey (End-to-End UX)
- tokens.ts
- 6. Focused mitigation designs
- check-i18n.mjs
- customer/src/sw.ts
- dependencies
- dependencies
- dependencies
- dependencies
- environment
- rovo — Session Handoff
- 1. Conventions
- Clock
- 20 — Testing Strategy
- Part A — Free tiers for local / CI / dev-preview / demo
- 32 — Final Plan Summary (Phase 1 close-out)
- devDependencies
- 00003_catalog.sql
- github.com/jackc/pgx/v5/pgxpool.Pool
- 5. Authorisation model
- shell/package.json
- 08 — System Architecture
- 00008_ledger.sql
- 5. Threat table
- Part B — Production cloud comparison (staging + production)
- users
- 4. Daily operations
- 10. Non-functional requirements (NFR-*)
- 12. Regulatory requirements (LEG-*) [LEGAL]
- 19 — Security Threat Model (STRIDE), Security Requirements & Incident Response
- 7. Cloud infrastructure, database, cryptography, logging and supply chain
- 6. Go integration tests
- 14. Service-by-service comparison (India regions)
- planning/README.md
- exports
- Phase 2 — Implementation Charter (binding for all implementation agents)
- rovo — System Knowledge Graph
- 7. Functional requirements — Delivery partner (RDR-*)
- 1. UX principles (apply to all rovo apps: customer, restaurant, rider, admin — R14)
- 15. Cost estimates (on-demand, excl. GST; ₹95/USD) — single source of cost (R46)
- aws_cloudfront_distribution.this
- smoke.sh
- react.json
- catalog
- Runbook: Incident response
- rovo brand assets
- 4. Success metrics and V1 targets
- 8. Checkout and payment
- 5. Key runtime sequences
- 4. Go unit tests
- 10. Capacity model (reference only — the load model is owned by doc 20)
- config/tsconfig.json
- customer/tsconfig.json
- restaurant/tsconfig.json
- rider/tsconfig.json
- node.json
- PostGIS image for local dev and CI
- 13. Help and support for an order (C-21)
- 9. Deployment topology and scaling path
- with
- 10. Incident response outline
- database/main.tf
- 14. Security testing
- 15. Data, migration and backup-restore testing
- 7. Contract tests
- 9. Frontend tests
- 00007_dispatch.sql
- utils/tsconfig.json
- verify-restore.sh
- scripts
- admin/tsconfig.json
- scripts
- scripts
- scripts
- auth/tsconfig.json
- shell/tsconfig.json
- testing/tsconfig.json
- ui/tsconfig.json
- PROMPT — rovo Phase 2: implement V1
- rovo infrastructure (OpenTofu)
- ADR-P2-001: OpenAPI is the contract; no Go server code generation in the first slice
- 4. Home / discovery (C-05)
- 7. Consistency, idempotency and concurrency
- cities
- storage/main.tf
- 8. Authorisation tests
- Store
- steps
- 7. CI/CD, DNS/CDN/WAF, monitoring, email
- dev-keys.sh
- .prettierrc.json
- 6. Login (deferred) and profile
- 7. Restaurant page, customisation, cart
- 9. Order tracking (C-16)
- react
- NotImplemented
- .golangci.yml
- 15. Environment config, build and deploy outputs
- .golangci.yml
- 3. PostgreSQL (PostGIS mandatory)
- 9. Dev / preview / demo recommendation
- tsconfig.node.json
- backup.sh
- bootstrap.sh
- dev.sh
- ADR-P2-003: Production-mode Docker Compose is the deployment target for this phase
- api-client/tsconfig.json
- config/package.json
- Registry
- i18n/tsconfig.json
- rovo
- 3. Admin authentication
- customer/src/router.ts
- 00009_ratings_support_notifications.sql
- 00010_admin_platform.sql
- 13. Accessibility, analytics, error tracking, feature flags, testing hooks
- x-app-env
- test-integration.sh
- tf-validate.sh
- scripts
- Module
- Point
- jobs
- phone.ts
- services
- react
- scripts
- minio-init.sh
- rovo
- knowledge-graph.sh
- restore.sh
- updates
- Export
- github.com/storetmobiles2-code/rovo/backend
- github.com/storetmobiles2-code/rovo/deploy/docker/healthcheck
- 00004_pricing_promotions.sql
- Sequence
- name
- 1. Conventions
- orders
- uuidv7
- lint-meta
- rateLimitSweepWorker
- docker
- minio
- api
- attributes
- go
- Ticket
- worker
- web
- .Validate
- 6. Fees and rider pay
- migrate
- 12. Performance and load tests
- 1. Framework evaluation (challenge of P7)
- 3. Routing maps
- grafana
- caddy
- postgis
- global
- rules
- args
- options
- grafana
- global
- body
- ci.yml
- 10. End-to-end tests
- lint-meta
- postgis
- rules
- healthcheck
- services
- mailpit
- providers
- datasources
- web
- outputs
- healthcheck
- minio-init
- mailpit
- prometheus
- minio
- groups
- prod/.terraform.lock.hcl
- staging/.terraform.lock.hcl
- Competitor "paved path" review — Zomato, Swiggy, Toing
- 5. Search (C-06)
- .PresignPut
- 20. Testability hooks required from backend and frontend
- valkey
- .checkov.yaml
- 12. Taxes, invoices and compliance `[LEGAL]`
- 1. Top-level layout
- contact_links
- 23. Traceability
- 10. Ledger design (double-entry-lite)
- pull_request_template.md
- 2. Quality goals and risk-based prioritisation
- scripts
- openapi.yaml
- @radix-ui/react-slot
- 11. COD flows
- 19. Test environments and provider fakes
- 13. Scope cut list and missing items
- @radix-ui/react-visually-hidden
- @testing-library/user-event@catalog:
- @types/react@catalog:
- @types/react-dom@catalog:
- 16. Reconciliation
- @rovo/auth@workspace:*
- @testing-library/react@catalog:
- workbox-cacheable-response@catalog:
- pnpm-workspace.yaml
- @rovo/auth@workspace:*
- CLAUDE.md
- .claude/CLAUDE.md
- @testing-library/react@catalog:
- workbox-cacheable-response@catalog:
- @rovo/api-client@workspace:*
- @rovo/auth@workspace:*
- @rovo/utils@workspace:*
- @testing-library/react@catalog:
- workbox-cacheable-response@catalog:
- vite@catalog:
- @testing-library/jest-dom@catalog:
- @testing-library/jest-dom@catalog:
- @types/react@catalog:
- typescript@catalog:
- vite-plugin-pwa@catalog:
- vitest@catalog:
- @typescript/native@catalog:
- workbox-routing@catalog:
- workbox-window@catalog:
- vite@catalog:
- @testing-library/jest-dom@catalog:
- @types/react@catalog:
- lucide-react@catalog:
- vite-plugin-pwa@catalog:
- vitest@catalog:
- @testing-library/jest-dom@catalog:
- workbox-routing@catalog:
- workbox-window@catalog:
- @types/react@catalog:
- @testing-library/jest-dom@catalog:
- @types/react@catalog:
- vite-plugin-pwa@catalog:
- vitest@catalog:
- workbox-routing@catalog:
- workbox-window@catalog:
- vitest@catalog:

## God Nodes (most connected - your core abstractions)
1. `Deps` - 83 edges
2. `users` - 55 edges
3. `catalog` - 55 edges
4. `with` - 47 edges
5. `cn()` - 46 edges
6. `NotImplemented()` - 44 edges
7. `New()` - 40 edges
8. `cities` - 34 edges
9. `rider` - 31 edges
10. `x-app-env` - 31 edges

## Surprising Connections (you probably didn't know these)
- `0. Decision summary` --references--> `ApprovalRequest`  [INFERRED]
  docs/planning/11-api-specification.md → backend/internal/modules/adminaudit/module.go
- `2.3 Customer` --references--> `Order`  [INFERRED]
  docs/planning/11-api-specification.md → backend/internal/modules/ordering/module.go
- `Foundation — done & verified (2026-11-16)` --references--> `NotImplemented()`  [INFERRED]
  docs/implementation/01-progress.md → backend/internal/platform/apperr/apperr.go
- `10.2 Time control in E2E` --references--> `Clock`  [INFERRED]
  docs/planning/20-testing-strategy.md → backend/internal/platform/clock/clock.go
- `Consequences` --references--> `DecodeJSON()`  [INFERRED]
  docs/adr/ADR-P2-001-openapi-contract-without-server-codegen.md → backend/internal/platform/httpx/respond.go

## Import Cycles
- 3-file cycle: `backend/internal/app/app.go -> backend/internal/modules/catalog/catalog_integration_test.go -> backend/internal/app/apptest/apptest.go -> backend/internal/app/app.go`
- 3-file cycle: `backend/internal/app/app.go -> backend/internal/modules/geo/geo_integration_test.go -> backend/internal/app/apptest/apptest.go -> backend/internal/app/app.go`
- 3-file cycle: `backend/internal/app/app.go -> backend/internal/modules/identity/identity_integration_test.go -> backend/internal/app/apptest/apptest.go -> backend/internal/app/app.go`
- 4-file cycle: `backend/internal/app/app.go -> backend/internal/modules/catalog/handlers.go -> backend/internal/modules/geo/geo_integration_test.go -> backend/internal/app/apptest/apptest.go -> backend/internal/app/app.go`
- 4-file cycle: `backend/internal/app/app.go -> backend/internal/modules/catalog/module.go -> backend/internal/modules/geo/geo_integration_test.go -> backend/internal/app/apptest/apptest.go -> backend/internal/app/app.go`

## Communities (355 total, 65 thin omitted)

### Community 0 - "schema.d.ts"
Cohesion: 0.02
Nodes (99): components, $defs, HeaderETag, HeaderIdempotentReplayed, HeaderLocation, HeaderRateLimit, HeaderRetryAfter, operations (+91 more)

### Community 1 - "ui/src/index.ts"
Cohesion: 0.05
Nodes (81): Language, Props, State, cn(), BillBreakdown(), BillBreakdownLine, BillBreakdownProps, ButtonProps (+73 more)

### Community 2 - "testing.T"
Cohesion: 0.13
Nodes (34): New(), findItem(), paise(), setup(), TestAvailabilityStates(), TestCuisines(), TestDiscoverFiltersAndSort(), TestDiscoverOutsideServiceArea() (+26 more)

### Community 3 - "money.ts"
Cohesion: 0.17
Nodes (20): useFormatters(), AddressLike, clean(), formatAddress(), cache, DateInput, dtf(), formatDate() (+12 more)

### Community 4 - "shell/src/index.ts"
Cohesion: 0.05
Nodes (52): App(), router, App(), updateSW, router, App(), updateSW, router (+44 more)

### Community 5 - "10 — Database Schema (logical + physical)"
Cohesion: 0.07
Nodes (28): 10 — Database Schema (logical + physical), 10. DDL — ratings, support, notifications, 11. DDL — admin, platform, 12. PII classification (DPDP Act 2023 & encryption decisions), 13.1 Erasure map (DPDP, M15, RV-031) [LEGAL], 13. Data retention policy (per table), 14. Retention jobs and partitioning triggers (C8), 15.1 Reference seeds (all environments) (+20 more)

### Community 6 - "api-client/package.json"
Cohesion: 0.05
Nodes (50): openapi-fetch, openapi-typescript, dependencies, openapi-fetch@catalog:, @rovo/utils@workspace:*, description, devDependencies, jsdom@catalog: (+42 more)

### Community 7 - "react.tsx"
Cohesion: 0.06
Nodes (31): 5.1 Endpoint and events (aligned with R10, doc 08 §Real-time and doc 12 §4.7), 5.2 Client (`createEventStream` in `packages/api-client`), 5. Real-time integration (SSE), ListFilters, RestaurantFilters, ApiClientContext, healthListeners, realtimeHealth (+23 more)

### Community 8 - "auth/package.json"
Cohesion: 0.06
Nodes (46): dependencies, @rovo/api-client@workspace:*, @rovo/utils@workspace:*, @tanstack/react-query@catalog:, description, devDependencies, jsdom@catalog:, react@catalog: (+38 more)

### Community 9 - "rider"
Cohesion: 0.06
Nodes (31): 11.3 BR-CAN — Cancellation policy (disclosed at checkout, LEG-CP-004), ADR-009: Frontend: Vite React SPA/PWA (not Next.js), Appendix: challenges to the baseline (summary for the Lead Architect), 0. Summary of decisions, 10. Internationalisation (`packages/i18n`), 11. Images, 12. Performance budgets, 15.1 Environments, origins and same-origin API (decision F4; follows `00` §4a and doc 12 AUTH-D03) (+23 more)

### Community 10 - "app.go"
Cohesion: 0.06
Nodes (50): ExpireApprovalsArgs, Claims, principalKey, Scopes, TOTPEnrolment, policyFor(), IsCode(), TestTOTP() (+42 more)

### Community 11 - "Middleware"
Cohesion: 0.11
Nodes (24): Forbidden(), ParseAppHosts(), TestOriginVerify(), TestRequireClientHeader(), TestResolveAudienceAndHelpers(), AccessLog(), BodyLimit(), Chain() (+16 more)

### Community 12 - "12 — Authentication & Authorisation (Auth / RBAC)"
Cohesion: 0.06
Nodes (30): 0. Decision summary, 10. Sources (accessed 2026-10-04), 11. Challenges to baseline (summary; full list in report), 12 — Authentication & Authorisation (Auth / RBAC), 1.1 Entities, 1.2 Roles and scopes, 1.3 Can a person be both customer and rider on one phone? (decision), 1.4 Phone-number recycling (India-specific risk) (+22 more)

### Community 13 - "14 — Payment Architecture"
Cohesion: 0.10
Nodes (21): 13. Commission and fee computation (quote engine), 14. COD risk controls, 14 — Payment Architecture, 15. Refunds, 17. Settlement cycle and payouts, 18. Failure and timeout handling (payments-specific), 19. Security and privacy, 1. Principles (+13 more)

### Community 14 - "NewDB"
Cohesion: 0.17
Nodes (19): New(), Up(), New(), adminExec(), ensureTemplate(), DB, migrationsHash(), NewDB() (+11 more)

### Community 15 - "16 — Delivery Zone & Geography Architecture"
Cohesion: 0.05
Nodes (36): 0. Decision summary, 10.1 Mahabubnagar sample zone (fixture, NOT the real service area), 10.2 Fixture points and expected results, 10.3 Scenario tests, 10. Test fixtures, 11. Mahabubnagar specifics: verified vs. assumed, 12. Not doing in V1, 13. Open items (+28 more)

### Community 16 - "06 — Delivery Partner (Rider) Workflow"
Cohesion: 0.06
Nodes (35): 06 — Delivery Partner (Rider) Workflow, 10.1 Earnings (D-20, D-21), 10.2 Cash-in-hand and deposit (D-22, D-23), 10.3 Delivery history (D-24), 10. Earnings, cash-in-hand, deposits, history, 11. Safety and SOS (D-30), 12. Delivery status ↔ order status map, 13. Edge cases (+27 more)

### Community 17 - "web/package.json"
Cohesion: 0.06
Nodes (37): eslint, prettier, description, devDependencies, eslint@catalog:, @eslint/js@catalog:, prettier@catalog:, @rovo/config@workspace:* (+29 more)

### Community 18 - "Env"
Cohesion: 0.16
Nodes (4): Client, Env, Response, env

### Community 19 - "data.go"
Cohesion: 0.36
Nodes (8): addon, addonGroup, category, coupon, item, ledgerAccount, restaurant, variant

### Community 20 - "18 — Mobile & PWA Strategy"
Cohesion: 0.06
Nodes (34): 0. Key constraints that shape product scope, 10.1 Step 1b: Trusted Web Activity (Bubblewrap) for restaurant and rider apps, 10.2 Criteria for going native (V2 triggers), 10.3 React Native (Expo) vs Flutter, 10.4 How V1 prepares for native, 10. Native path, 11. Device and browser support matrix, 12. Low-end device test plan (+26 more)

### Community 21 - "i18n/package.json"
Cohesion: 0.07
Nodes (37): i18next, react-i18next, dependencies, i18next@catalog:, react-i18next@catalog:, @rovo/utils@workspace:*, description, devDependencies (+29 more)

### Community 22 - "customer/package.json"
Cohesion: 0.17
Nodes (11): description, @rovo/testing@workspace:*, vite@catalog:, workbox-precaching@catalog:, @rovo/testing, vite, workbox-precaching, name (+3 more)

### Community 23 - "restaurant/package.json"
Cohesion: 0.17
Nodes (11): description, @rovo/testing@workspace:*, vite@catalog:, workbox-precaching@catalog:, @rovo/testing, vite, workbox-precaching, name (+3 more)

### Community 24 - "rider/package.json"
Cohesion: 0.17
Nodes (11): description, @rovo/testing@workspace:*, vite@catalog:, workbox-precaching@catalog:, @rovo/testing, vite, workbox-precaching, name (+3 more)

### Community 25 - "github.com/google/uuid.UUID"
Cohesion: 0.04
Nodes (45): ExecuteApprovalArgs, New(), nullUUID(), New(), NotImplemented(), Money, TOTPAAD(), Addon (+37 more)

### Community 26 - "time.Time"
Cohesion: 0.06
Nodes (30): hhmm(), isoWeekday(), openAt(), TestOpenAt(), i18nMap(), scanItem(), minuteOfDay(), TestPreviousISOWeek() (+22 more)

### Community 27 - "13 — Order & Delivery State Machines"
Cohesion: 0.06
Nodes (32): 0. Decision summary, 10. Open items, 11. Challenges / deviations recorded, 13 — Order & Delivery State Machines, 1.1 Order transition table, 1.2 Admin "on behalf" commands, 1. Order state machine, 2.1 Delivery transition table (+24 more)

### Community 28 - "money.ts"
Cohesion: 0.32
Nodes (14): intlLocale(), addPaise(), assertPaise(), currencyFormatter(), formatMoney(), formatNumber(), formatterCache, money() (+6 more)

### Community 29 - "Module"
Cohesion: 0.08
Nodes (25): Register, RegisterJobs, New(), New(), New(), New(), New(), Module (+17 more)

### Community 30 - "07 — Admin Console Workflow"
Cohesion: 0.06
Nodes (31): 07 — Admin Console Workflow, 10. Coupons (A-50, A-51), 11.1 Lifecycle, 11.2 Priority and SLAs (defaults, configurable), 11.3 Ticket workspace (A-71), 11. Disputes and tickets (A-70, A-71), 12.1 Weekly payout run, 12. Commissions and settlements (A-80 … A-83) (+23 more)

### Community 31 - "05 — Restaurant Partner Workflow"
Cohesion: 0.09
Nodes (22): 05 — Restaurant Partner Workflow, 10. Screen inventory (`restaurant` app), 11. Requirements for Backend / Frontend and challenges to the baseline, 1.1 Indicative Telugu labels (key actions) `[ASSUMPTION — native-speaker review required]`, 1. Design constraints specific to restaurants, 2. Roles and permissions (restaurant side), 3.1 Flow overview, 3.2 Lead form (P-01, public, no account needed) (+14 more)

### Community 32 - "context.Context"
Cohesion: 0.08
Nodes (16): ReasonCode, RateLimitSweepArgs, rateLimitSweepWorker, RandomToken(), Connect(), Querier, MarkProcessed(), Sweep() (+8 more)

### Community 33 - "ui/package.json"
Cohesion: 0.20
Nodes (9): @radix-ui/react-select, @radix-ui/react-select@catalog:, description, exports, name, private, sideEffects, type (+1 more)

### Community 34 - "Bus"
Cohesion: 0.13
Nodes (6): Bus, Start(), Frame, Hub, Subscription, Writer

### Community 35 - "22 — Deployment Architecture"
Cohesion: 0.08
Nodes (28): 10. Local development (Docker Compose), 11. Dev / demo environment (local only, R24), 12. Infrastructure as Code layout (OpenTofu; Terraform-compatible), 13. Resource allocation summary, 14. Scaling path, 15. Cost (single source: `25` §15, R46), 16. Security baseline checklist (deployment-level; threat model in `19`), 1. Environments (+20 more)

### Community 36 - "09 — Architecture Decision Records"
Cohesion: 0.08
Nodes (25): 09 — Architecture Decision Records, ADR-001: Modular monolith instead of microservices, ADR-002: Go for the backend; HTTP routing with net/http ServeMux, ADR-003: REST/JSON with OpenAPI spec-first and generated code, ADR-004: No gRPC in V1, ADR-006: Asynchronous processing with River as the transactional outbox, ADR-008: Real-time via Server-Sent Events, ADR-010: Monorepo (Go module + pnpm workspace in one repository) (+17 more)

### Community 37 - "2. Backlog by epic"
Cohesion: 0.07
Nodes (29): 2. Backlog by epic, E02 — Identity, auth & RBAC, E03 — Geo: cities, zones, localities, serviceability, E04 — Partner onboarding & KYC, E05 — Catalog & menu, E06 — Pricing, quote engine, tax & coupons, E07 — Ordering & order state machine, E08 — Payments (+21 more)

### Community 38 - "admin/package.json"
Cohesion: 0.20
Nodes (9): @rovo/auth@workspace:*, @tanstack/react-router@catalog:, description, @rovo/auth, @tanstack/react-router, name, private, type (+1 more)

### Community 39 - "dependencies"
Cohesion: 0.13
Nodes (15): dependencies, lucide-react@catalog:, @rovo/api-client@workspace:*, @rovo/auth@workspace:*, @rovo/i18n@workspace:*, @rovo/ui@workspace:*, @rovo/utils@workspace:*, @tanstack/react-query@catalog: (+7 more)

### Community 40 - "time.Duration"
Cohesion: 0.19
Nodes (8): maxAge(), Offset, RateLimit(), Limiter, Options, ADR-007: Redis at launch? **No.** Managed Redis later, by configuration, E01 — Platform foundations, Decision

### Community 41 - "net/http.Request"
Cohesion: 0.07
Nodes (28): fieldErr(), restaurantID(), Locale(), PointFromQuery(), As(), PrincipalFrom(), writeJSON(), AudienceFrom() (+20 more)

### Community 42 - "Audience"
Cohesion: 0.04
Nodes (54): FieldError, TokenInput, cooldownFor(), memberAudience(), newOTPCode(), nullIfEmpty(), claimsFor(), devicePlatform() (+46 more)

### Community 43 - "Client"
Cohesion: 0.30
Nodes (3): Client, InsertOpts, InsertParam

### Community 44 - "Run"
Cohesion: 0.12
Nodes (17): Argon2Params, TestFieldCrypto(), TestPasswordHash(), FieldKeyring, NewFieldKeyring(), PasswordHasher, NewPasswordHasher(), parsePHC() (+9 more)

### Community 45 - "New"
Cohesion: 0.10
Nodes (25): Options, main(), runMigrate(), runSeed(), runServers(), serverRole(), New(), TestAllModulesWire() (+17 more)

### Community 46 - "obs.go"
Cohesion: 0.09
Nodes (7): NewMetrics(), redactAttr(), SetupTracing(), statusClass(), ctxKey, Metrics, Tracing

### Community 47 - "24 — Observability Strategy"
Cohesion: 0.07
Nodes (28): 10. Alert routing & on-call (tiny team), 11. Data-volume budget (production; volumes per R45 / doc 20, requests per `25` §15.7), 12. What we are not doing in V1, 1.1 Cost comparison for production (pilot volumes from `25` §10), 1.2 CERT-In log archive (M1, R36) — compliance copy, separate from Grafana, 1. Decision: telemetry backends per environment, 24 — Observability Strategy, 2.1 Backend (Go) (+20 more)

### Community 48 - "2. Milestones and exit criteria"
Cohesion: 0.09
Nodes (22): 1. Planning assumptions (state them, then challenge them at M0 exit), 28 — Milestones & Dependencies (Phase 2), 2. Milestones and exit criteria, 3.1 Summary, 3.2 Gantt (indicative), 3.3 Team-size variants, 3. Calendar and capacity, 4.1 The path (+14 more)

### Community 49 - "31 — Review Report (Red Team)"
Cohesion: 0.11
Nodes (17): 0. Executive summary: top 10 issues, 11. V1 scope creep and essential gaps, 12. Disagreements requiring Lead decision, 14. Cross-doc inconsistency register, 15. Sources spot-checked (accessed 2026-10-04), 1. Architecture choices (over- or under-engineered?), 2.1 Per-order unit economics (Reviewer model, `[ASSUMPTION]` inputs), 2.2 Findings (+9 more)

### Community 50 - "gen-brand-assets.py"
Cohesion: 0.17
Nodes (11): best_font(), contain(), load_mark(), main(), make_icon(), make_logo(), rounded_bg(), save_png() (+3 more)

### Community 51 - "testing/package.json"
Cohesion: 0.05
Nodes (54): axe-core, msw, dependencies, axe-core@catalog:, jsdom@catalog:, msw@catalog:, @rovo/api-client@workspace:*, @rovo/i18n@workspace:* (+46 more)

### Community 52 - "compilerOptions"
Cohesion: 0.09
Nodes (22): compilerOptions, allowImportingTsExtensions, esModuleInterop, exactOptionalPropertyTypes, forceConsistentCasingInFileNames, isolatedModules, lib, module (+14 more)

### Community 53 - "29 — Production Readiness Checklist (Go-Live)"
Cohesion: 0.10
Nodes (21): 0. How to use, 10. Resilience tests, 11. Mobile / PWA device matrix, 12. Accessibility & i18n, 13. Support processes, 14. Rollback plan, 15. Go/No-Go decision procedure, 16.1 Launch day (public launch, target Monday 2027-11-08) (+13 more)

### Community 54 - "devDependencies"
Cohesion: 0.13
Nodes (15): devDependencies, @rovo/config@workspace:*, @rovo/testing@workspace:*, @testing-library/react@catalog:, @testing-library/user-event@catalog:, @types/react@catalog:, @types/react-dom@catalog:, vitest@catalog: (+7 more)

### Community 55 - "02 — V1 Scope"
Cohesion: 0.10
Nodes (20): 02 — V1 Scope, 1.1 Capability map (P0 = launch blocker), 1.2 P1 items committed to V1 (built before launch), 1.3 Golden-flow coverage check, 1. In scope for V1, 2.1 Mandatory deferrals (baseline §1), 2.2 Additional scope-creep decisions, 2.3 Accepted scope cuts from the review (31 §13.1, baseline §9) — all deferred to V1.1+ (+12 more)

### Community 56 - "15 — Notification Architecture"
Cohesion: 0.07
Nodes (26): 0. Summary, 10. Provider abstraction (Go sketch), 11. Delivery receipts and observability, 12. Failure modes, 13. Open questions, 15 — Notification Architecture, 1. Channels, 2.1 Customer (+18 more)

### Community 57 - "21 — CI/CD Strategy"
Cohesion: 0.07
Nodes (26): 10. Release process, 11. Dependency updates, 12. Branch protection (repository rulesets), 13. CI time budget, 1. Principles, 21 — CI/CD Strategy, 2. Workflow map, 3.1 Go backend (`backend/**`, `openapi/**`) (+18 more)

### Community 58 - "23 — Backup & Disaster Recovery"
Cohesion: 0.10
Nodes (20): 10. Dev/preview & local, 1. Targets, 23 — Backup & Disaster Recovery, 2. What must be protected, 3.1 Layers, 3.2 Cost, 3. PostgreSQL backup design, 4. Object storage protection (+12 more)

### Community 59 - "(a) Risk register"
Cohesion: 0.10
Nodes (20): 30 — Risks, Assumptions, Decisions & Open Questions (RAID), (a) Risk register, Architecture Decision Records (doc 09), (b) Assumptions register (significant `[ASSUMPTION]`s from docs 01–26), (c) Decisions log, (d) Unresolved decisions / open questions (deduplicated), (e) Remaining cross-document conflicts (for reconciliation by doc owners), Financial (+12 more)

### Community 60 - "client.ts"
Cohesion: 0.05
Nodes (34): absolute(), acceptLanguage(), API_BASE_URL, ApiClient, ApiClientOptions, FetchResultLike, idempotencyHeaders(), RovoClient (+26 more)

### Community 61 - "time.Duration"
Cohesion: 0.40
Nodes (3): TestCookies(), Cookies, sameSite()

### Community 62 - "catalog_integration_test.go"
Cohesion: 0.07
Nodes (72): aws_appautoscaling_policy.api_cpu, aws_appautoscaling_target.api, aws_ecs_cluster_capacity_providers.this, aws_ecs_cluster.this, aws_ecs_service.api, aws_ecs_service.worker, aws_ecs_task_definition.api, aws_ecs_task_definition.migrate (+64 more)

### Community 63 - "devDependencies"
Cohesion: 0.12
Nodes (17): devDependencies, @rovo/config@workspace:*, @testing-library/user-event@catalog:, @types/react-dom@catalog:, typescript@catalog:, @typescript/native@catalog:, workbox-core@catalog:, workbox-expiration@catalog: (+9 more)

### Community 64 - "devDependencies"
Cohesion: 0.12
Nodes (17): devDependencies, @rovo/config@workspace:*, @testing-library/user-event@catalog:, @types/react-dom@catalog:, typescript@catalog:, @typescript/native@catalog:, workbox-core@catalog:, workbox-expiration@catalog: (+9 more)

### Community 65 - "devDependencies"
Cohesion: 0.12
Nodes (17): devDependencies, @rovo/config@workspace:*, @testing-library/user-event@catalog:, @types/react-dom@catalog:, typescript@catalog:, @typescript/native@catalog:, workbox-core@catalog:, workbox-expiration@catalog: (+9 more)

### Community 66 - "auth/src/index.ts"
Cohesion: 0.06
Nodes (58): renderAt(), server, createAdminRouter(), renderAt(), server, createAppRouter(), renderAt(), server (+50 more)

### Community 67 - "newIssuer"
Cohesion: 0.21
Nodes (11): Keyset, newIssuer(), TestJWTRejectsForgeries(), TestJWTRoundTrip(), TestKeyRotationVerifiesPrevious(), Issuer, NewIssuer(), KID() (+3 more)

### Community 68 - ".UpdateMe"
Cohesion: 0.21
Nodes (7): deref(), Me, MaskPhone(), New(), MePatch, RoleGrant, Service

### Community 69 - "utils/package.json"
Cohesion: 0.12
Nodes (19): description, devDependencies, @rovo/config@workspace:*, typescript@catalog:, @typescript/native@catalog:, vitest@catalog:, exports, @rovo/config (+11 more)

### Community 70 - "minio-init"
Cohesion: 0.22
Nodes (9): minio, condition, depends_on, entrypoint, image, networks, restart, volumes (+1 more)

### Community 71 - "Middleware"
Cohesion: 0.08
Nodes (50): Terraform module: deploy/terraform/envs/prod, local.app_environment, local.app_hosts, local.name_prefix, local.tags, module.compute, module.database, module.edge (+42 more)

### Community 72 - "11. Business rules (BR-*)"
Cohesion: 0.12
Nodes (16): 11.10 BR-COUP — Coupons and stacking, 11.11 BR-CONT — Contact & privacy rules (V1 without masked calling), 11.12 BR-OTP — Delivery OTP, 11.13 BR-RES — Restaurant operating rules, 11.14 BR-PAYOUT — Payout cycles, 11.15 BR-MENU — Menu & dietary rules, 11.15a BR-COST — Running cost as a business line item, 11.1 BR-FEE — Customer-facing fees and pricing (+8 more)

### Community 73 - "Env"
Cohesion: 0.12
Nodes (27): Loading(), LoginRedirect(), Protected(), RequireScreenRole(), ConsoleFrame(), ICONS, isActive(), NAV (+19 more)

### Community 74 - "03 — User Personas"
Cohesion: 0.12
Nodes (15): 03 — User Personas, Anti-personas (we do not design V1 for), Cross-persona summary, Device & connectivity matrix (feeds doc 18 device lab and doc 20 test matrix), P1 — Sravani, the budget-conscious student, P2 — Ravi & Lakshmi, the working family ordering dinner, P3 — Narasimha Reddy, elderly, Telugu-first, less tech-savvy, P4 — Venkatesh, small family restaurant owner, shared phone (+7 more)

### Community 75 - "3.2 Module catalogue"
Cohesion: 0.11
Nodes (19): 3.1 Module map and dependency direction, 3.2 Module catalogue, 3.3 Event catalogue conventions, 3. Components: the Go modular monolith (C4 level 3), admin & audit, catalog (restaurants, menus), dispatch (delivery), geo (cities, zones, localities, serviceability) (+11 more)

### Community 76 - "lib.sh"
Cohesion: 0.18
Nodes (9): compose_pg_container(), die(), fail(), need(), _pg_run(), pg_tool(), pg_tool_nodb(), psql_q() (+1 more)

### Community 77 - "01 — Product Requirements Document (PRD)"
Cohesion: 0.08
Nodes (24): 01 — Product Requirements Document (PRD), 13. Open questions, 14. Sources (accessed 2026-10-04), 1. Vision, 2.1 Market context, 2.2 Problems we solve, 2. Problem statement — Mahabubnagar, 3.1 Goals (V1) (+16 more)

### Community 78 - "5. Functional requirements — Customer (CUS-*)"
Cohesion: 0.14
Nodes (14): 5.10 CUS-TRK — Order tracking and status, 5.11 CUS-HIST — Order history and reorder, 5.12 CUS-RATE — Ratings and reviews, 5.13 CUS-COUP — Coupons and offers, 5.14 CUS-SUPP — Support and help for an order, 5.1 CUS-AUTH — Registration, login, profile, account, 5.3 CUS-DISC — Restaurant discovery, 5.4 CUS-SRCH — Search, filters, categories (+6 more)

### Community 79 - "8. Functional requirements — Admin (ADM-*)"
Cohesion: 0.13
Nodes (15): 8.10 ADM-RPT — Reports and CSV export [view per role], 8.11 ADM-AUDIT — Audit logs [view: S; F for money events], 8.12 ADM-ZONE — Zones and fees configuration [act: S/O; F for fee changes], 8.13 ADM-PAYO — Payouts and settlement [act: F/S], 8.14 ADM-CFG / ADM-CONT — Platform configuration and content, 8.1 ADM-AUTH — Admin access, 8.2 ADM-DASH — Dashboard, 8.3 ADM-USER — Users (customers) [view: all; act: S/O/Sp] (+7 more)

### Community 80 - "config/package.json"
Cohesion: 0.10
Nodes (26): eslint-plugin-jsx-a11y, eslint-plugin-react-refresh, @fontsource/noto-sans-telugu, tailwindcss, @tailwindcss/vite, @vitejs/plugin-react, workbox-build, dependencies (+18 more)

### Community 81 - "gen-api.mjs"
Cohesion: 0.10
Nodes (14): html, bin, out, pkg, placeholder, real, checkCatalogs(), flatten() (+6 more)

### Community 82 - "Publisher"
Cohesion: 0.23
Nodes (8): Publisher, NewPublisher(), Enqueuer, 4.1 The rules, 4.2 Compiler-enforced encapsulation (cheap and strong), 4.3 CI checks for what the compiler can't see, 4. Module boundary rules, Notifier

### Community 83 - "8. Privacy and regulatory (DPDP, IT Act, CERT-In, RBI/PCI, Aadhaar) — `[LEGAL]` throughout"
Cohesion: 0.14
Nodes (14): 8.10 CERT-In Directions (28 Apr 2022), 8.11 RBI Payment Aggregator directions and card data, 8.12 PCI-DSS scope minimisation, 8.13 Aadhaar, 8.1 DPDP Act 2023 and DPDP Rules 2025: status as of 2026-10-04, 8.2 DPDP obligations → rovo design, 8.3 Notice and consent UX (for 04/05/06), 8.4 Data-principal rights workflows (+6 more)

### Community 84 - "9. Security requirements checklist (testable)"
Cohesion: 0.14
Nodes (14): 9.10 Privacy and compliance, 9.11 Incident readiness, 9.12 Cloud platform (added for 00 §4a), 9.13 Added in v1.1, 9.1 Authentication (OTP and admin), 9.2 Sessions, tokens, CSRF, CORS, SSE, 9.3 Authorisation, 9.4 Web and content security (+6 more)

### Community 85 - "Publisher"
Cohesion: 0.24
Nodes (9): InTx(), WithTx(), WithTxOptions(), TxBeginner, ADR-P2-002: pgx with hand-written parameterised SQL instead of sqlc, Consequences, Context, Decision (+1 more)

### Community 86 - "shell/src/index.ts"
Cohesion: 0.05
Nodes (65): LoginScreen(), NotFound(), Loading(), LoginRedirect(), Protected(), LoginScreen(), NotFound(), guarded() (+57 more)

### Community 87 - "eslint.js"
Cohesion: 0.20
Nodes (7): eslint-plugin-react-hooks, globals, typescript-eslint, rovoEslintConfig(), eslint-plugin-react-hooks@catalog:, globals@catalog:, typescript-eslint@catalog:

### Community 88 - "dependencies"
Cohesion: 0.11
Nodes (19): clsx, @radix-ui/react-checkbox, @radix-ui/react-dialog, @radix-ui/react-radio-group, @radix-ui/react-switch, @radix-ui/react-toast, dependencies, clsx@catalog: (+11 more)

### Community 89 - "devDependencies"
Cohesion: 0.11
Nodes (19): devDependencies, jsdom@catalog:, @rovo/config@workspace:*, @rovo/testing@workspace:*, @testing-library/react@catalog:, @testing-library/user-event@catalog:, @types/react-dom@catalog:, typescript@catalog: (+11 more)

### Community 90 - "26 — Repository Structure"
Cohesion: 0.17
Nodes (11): 10. Community and governance files to create in Phase 2, 11. CI path filters (summary for doc 21), 12. Open items, 26 — Repository Structure, 3. Code ownership (CODEOWNERS), 4. Branching, commits, versioning, releases, 5. Generated code policy, 6. Task runner: `Makefile` targets (+3 more)

### Community 91 - "4. Tokens, sessions, cookies, CSRF, CORS, SSE"
Cohesion: 0.14
Nodes (41): aws_eip.nat, aws_flow_log.all, aws_internet_gateway.this, aws_nat_gateway.this, aws_route.app_nat, aws_route.public_igw, aws_route_table.app, aws_route_table_association.app (+33 more)

### Community 92 - "4. Tokens, sessions, cookies, CSRF, CORS, SSE"
Cohesion: 0.18
Nodes (11): 4.10 Signing-key management and rotation, 4.1 Host topology (decision AUTH-D03, refines P7), 4.2 Access token (JWT), 4.3 Refresh tokens: rotation and reuse detection, 4.4 Cookies, 4.5 CSRF strategy (AUTH-D06), 4.6 CORS policy, 4.7 SSE authentication (+3 more)

### Community 93 - "vite.ts"
Cohesion: 0.25
Nodes (4): defineRovoApp(), RovoAppName, RovoAppOptions, RovoPwaOptions

### Community 94 - "Deps"
Cohesion: 0.06
Nodes (23): service, App, registerPlatformJobs(), Deps, Mux, AddWorker(), Registry, NewRegistry() (+15 more)

### Community 95 - "Contributing to rovo"
Cohesion: 0.08
Nodes (53): Terraform module: deploy/terraform/envs/staging, local.app_environment, local.app_hosts, local.name_prefix, local.tags, module.compute, module.database, module.edge (+45 more)

### Community 96 - "00 — Planning Baseline (Lead Architect Brief)"
Cohesion: 0.10
Nodes (20): 00 — Planning Baseline (Lead Architect Brief), 1. Product framing, 2. Canonical vocabulary (use these exact names), 3. Cross-cutting conventions, 4. Provisional architecture decisions (agents may challenge), 4a. Environment & hosting strategy (USER DIRECTIVE, 2026-10-04 — overrides P17), 5. Indicative commercial defaults (all configurable per city/zone/restaurant; validate with local market), 6. Deliverable map and owners (+12 more)

### Community 97 - "04 — Customer Journey (End-to-End UX)"
Cohesion: 0.17
Nodes (11): 04 — Customer Journey (End-to-End UX), 10. Cancellation rules (UX), 11. Delivered → rating (C-17), 12. Order history and reorder (C-18, C-19), 14. Notification touchpoints, 15. Edge cases and resilience, 16. Screen inventory (customer app), 17. Requirements for Backend / Frontend / other docs (+3 more)

### Community 98 - "tokens.ts"
Cohesion: 0.31
Nodes (8): appThemeColor, channel(), contrastRatio(), contrastRules, palette, PaletteToken, relativeLuminance(), themeCss

### Community 99 - "6. Focused mitigation designs"
Cohesion: 0.17
Nodes (12): 6.10 Upload pipeline (T-54..T-58) — decision (v1.1, R38), 6.11 DoS, scraping and edge configuration (T-59..T-63), 6.1 OTP abuse (T-01..T-07), 6.2 XSS, CSP and security headers (T-24..T-30), 6.3 Server-authoritative pricing and quotes (T-31, T-32), 6.4 Coupon abuse (T-33, T-34), 6.5 Payment webhooks (T-35..T-38), 6.6 Refund fraud (T-39, T-40) (+4 more)

### Community 100 - "check-i18n.mjs"
Cohesion: 0.40
Nodes (5): 12.1 Load model — canonical (R45; this doc owns it per R48), 12.2 SLOs asserted as k6 thresholds, 12.3 k6 scenarios, 12.4 Capacity test on the managed production shape, 12. Performance and load tests

### Community 101 - "customer/src/sw.ts"
Cohesion: 0.32
Nodes (3): NEVER_CACHE, NEVER_CACHE, NEVER_CACHE

### Community 102 - "dependencies"
Cohesion: 0.13
Nodes (15): dependencies, lucide-react@catalog:, react@catalog:, react-dom@catalog:, @rovo/i18n@workspace:*, @rovo/shell@workspace:*, @rovo/ui@workspace:*, @tanstack/react-query@catalog: (+7 more)

### Community 103 - "dependencies"
Cohesion: 0.10
Nodes (21): dependencies, lucide-react@catalog:, react@catalog:, react-dom@catalog:, @rovo/api-client@workspace:*, @rovo/i18n@workspace:*, @rovo/shell@workspace:*, @rovo/ui@workspace:* (+13 more)

### Community 104 - "dependencies"
Cohesion: 0.10
Nodes (21): dependencies, lucide-react@catalog:, react@catalog:, react-dom@catalog:, @rovo/api-client@workspace:*, @rovo/i18n@workspace:*, @rovo/shell@workspace:*, @rovo/ui@workspace:* (+13 more)

### Community 105 - "dependencies"
Cohesion: 0.10
Nodes (21): dependencies, lucide-react@catalog:, react@catalog:, react-dom@catalog:, @rovo/api-client@workspace:*, @rovo/i18n@workspace:*, @rovo/shell@workspace:*, @rovo/ui@workspace:* (+13 more)

### Community 106 - "environment"
Cohesion: 0.12
Nodes (41): environment, environment, OTEL_EXPORTER_OTLP_ENDPOINT, ROVO_ACME_EMAIL, ROVO_ADMIN_BOOTSTRAP_PASSWORD, ROVO_API_HEALTH_UPSTREAM, ROVO_API_UPSTREAM, ROVO_APP_HOSTS (+33 more)

### Community 107 - "rovo — Session Handoff"
Cohesion: 0.14
Nodes (12): Foundation — done & verified (2026-11-16), Known small items (not blocking; fold into feature phase), M1 — started (2026-11-16, branch `feat/m1-identity-geo-catalog`), Next: feature phase (see `docs/planning/28` milestones, `27` backlog), Phase 2 — Implementation Progress, Environment setup for a new session (the container is ephemeral), Guardrails (do not relearn the hard way), Honest state as of this handoff (+4 more)

### Community 108 - "1. Conventions"
Cohesion: 0.06
Nodes (33): 0. Decision summary, 11 — API Specification (REST, OpenAPI 3.1), 1.10 Rate limiting, 1.11 Localization, 1.12 Auth (detail in 12), 1.13 Versioning and deprecation, 1.14 Caching, 1.1 Hosts, audiences, prefixes (+25 more)

### Community 109 - "Clock"
Cohesion: 0.18
Nodes (12): Clock, IdempotencyStore, NewIdempotencyStore(), 18.1 Principles, 18.2 Seed profiles, 18.3 Mahabubnagar `e2e` dataset (illustrative), 18.4 Factories / builders, 18.5 Production-derived data and anonymisation (DPDP) (+4 more)

### Community 110 - "20 — Testing Strategy"
Cohesion: 0.12
Nodes (15): 13. Resilience / chaos-lite, 16.1 Telugu localisation QA, 16.2 Accessibility audit (manual, pre-release), 16. Localisation QA and accessibility audit, 17. UAT, pilot, dogfooding, bug bash, 20 — Testing Strategy, 24. What we are deliberately not doing, 25. Open questions (+7 more)

### Community 111 - "Part A — Free tiers for local / CI / dev-preview / demo"
Cohesion: 0.18
Nodes (11): 11. Sources (all accessed 2026-10-04), 12. Why free tiers are not acceptable for production (risk summary), 1. Requirements recap (from baseline), 2.1 Comparison table, 2.2 Low-cost paid VM reference (preview fallback only), 2. Backend compute, 4. Redis / cache (P6 check), 5. Object storage (+3 more)

### Community 112 - "32 — Final Plan Summary (Phase 1 close-out)"
Cohesion: 0.18
Nodes (11): 32 — Final Plan Summary (Phase 1 close-out), A. Recommended Architecture, B. Recommended Hosting Stack, C. V1 Feature List, D. Database Design Summary, E. API Summary, G. Testing Strategy, H. Deployment Strategy (+3 more)

### Community 113 - "devDependencies"
Cohesion: 0.13
Nodes (15): devDependencies, @rovo/config@workspace:*, @rovo/testing@workspace:*, @testing-library/jest-dom@catalog:, @testing-library/react@catalog:, typescript@catalog:, @typescript/native@catalog:, vitest@catalog: (+7 more)

### Community 114 - "00003_catalog.sql"
Cohesion: 0.11
Nodes (37): addon_groups, addons, cuisines, item_variants, ix_addons__group, ix_item_variants__item, ix_leads__queue, ix_menu_item_addon_groups__group (+29 more)

### Community 115 - "github.com/jackc/pgx/v5/pgxpool.Pool"
Cohesion: 0.11
Nodes (11): New(), New(), NewBus(), TestBusTransactionalNotify(), NewHub(), TestCapFrame(), TestHubDropsSlowConsumer(), TestHubRoutesByTopic() (+3 more)

### Community 116 - "5. Authorisation model"
Cohesion: 0.12
Nodes (28): aws_cloudfront_cache_policy.api, aws_cloudfront_cache_policy.static, aws_cloudfront_distribution.this, aws_cloudfront_function.router, aws_cloudfront_origin_access_control.media, aws_cloudfront_origin_access_control.web, aws_cloudfront_origin_request_policy.api, aws_cloudfront_response_headers_policy.security (+20 more)

### Community 117 - "shell/package.json"
Cohesion: 0.22
Nodes (8): description, exports, ./runtime-config, name, private, sideEffects, type, version

### Community 118 - "08 — System Architecture"
Cohesion: 0.11
Nodes (17): 08 — System Architecture, 0. Summary (one screen), 10. Multi-city expansion path, 11. Failure modes and graceful degradation, 12. What we are explicitly *not* doing in V1 (and why), 13. Version baseline (verified 2026-10-04), 14. Open items raised by this document, 1. System context (C4 level 1) (+9 more)

### Community 119 - "00008_ledger.sql"
Cohesion: 0.13
Nodes (29): assert_journal_balanced(), cod_deposits, invoice_sequences, invoices, ix_cod_deposits__queue, ix_cod_deposits__rider, ix_invoices__order, ix_ledger_journals__date (+21 more)

### Community 120 - "5. Threat table"
Cohesion: 0.20
Nodes (10): 5.1 Authentication & sessions, 5.2 Authorisation & data access, 5.3 Web application & content, 5.4 Commerce, payments & fraud, 5.5 Admin & insider (application), 5.6 Uploads, storage & media, 5.7 Availability, abuse & cost, 5.8 Cloud control plane, infrastructure, data, secrets & supply chain (+2 more)

### Community 121 - "Part B — Production cloud comparison (staging + production)"
Cohesion: 0.20
Nodes (9): 0. Executive summary, 13. Production requirements (from baseline §4a), 16.1 Scoring (weights reflect §4a non-negotiables), 16.2 Decision, 16. Production recommendation, 17. Production-cloud sources (all accessed 2026-10-04), 18. Challenges to baseline, 25 — Hosting Comparison: Free Tiers (dev/preview) & Production Cloud (+1 more)

### Community 122 - "users"
Cohesion: 0.10
Nodes (37): admin_credentials, admin_recovery_codes, customer_addresses, customer_profiles, devices, enforce_user_role_rules(), erasure_requests, ix_customer_addresses__user (+29 more)

### Community 123 - "4. Daily operations"
Cohesion: 0.25
Nodes (8): 4.1 Start of day — "Start taking orders" (P-20), 4.2 Outlet status: Open / Busy (paused) / Closed (P-21), 4.3 New order alert and accept / reject (P-22, P-23), 4.4 Customer cancels during the grace window, 4.5 Preparing → Ready (P-24), 4.6 Rider arrival and handover (P-25), 4.7 Order history (P-26), 4. Daily operations

### Community 124 - "10. Non-functional requirements (NFR-*)"
Cohesion: 0.22
Nodes (9): 10.1 NFR-PERF — Performance budgets, 10.2 NFR-AVAIL — Availability and resilience (production on managed cloud, baseline §4a), 10.3 NFR-A11Y — Accessibility (WCAG 2.2 AA), 10.4 NFR-I18N — Localisation, 10.5 NFR-PRIV — Privacy (DPDP) — see also LEG-DPDP-*, 10.6 NFR-SEC — Security (doc 12/19 own details), 10.7 NFR-AUD — Auditability, 10.8 NFR-RET — Data retention [LEGAL – retention schedule to be confirmed by counsel/CA] (+1 more)

### Community 125 - "12. Regulatory requirements (LEG-*) [LEGAL]"
Cohesion: 0.22
Nodes (9): 12.1 Food safety — FSSAI, 12.2 GST and income-tax, 12.3 Privacy — DPDP Act 2023 and DPDP Rules 2025, 12.4 Telecom — TRAI DLT, 12.5 Consumer protection — Consumer Protection Act 2019, E-Commerce Rules 2020, Dark Patterns Guidelines 2023, 12.6 Payments — RBI, 12.7 Gig & platform workers, 12.8 Other (+1 more)

### Community 126 - "19 — Security Threat Model (STRIDE), Security Requirements & Incident Response"
Cohesion: 0.22
Nodes (8): 11. Top risks after mitigation (for 30), 13. Challenges to baseline, 19 — Security Threat Model (STRIDE), Security Requirements & Incident Response, 1. Scope, method and assumptions, 2. Assets and data classification, 3. Threat agents, 4.1 Trust boundaries and entry points, 4. Data-flow diagram and trust boundaries

### Community 127 - "7. Cloud infrastructure, database, cryptography, logging and supply chain"
Cohesion: 0.22
Nodes (9): 7.1 Database roles and hardening (managed PostgreSQL), 7.2 Accounts, network and operator access, 7.3 Encryption and key management (KMS envelope encryption), 7.4 Backups (summary; owned by 23), 7.5 Secrets management, 7.6 Logging, audit and PII redaction, 7.7 Supply chain and CI/CD (for 21), 7.8 Open-source-specific measures (+1 more)

### Community 128 - "6. Go integration tests"
Cohesion: 0.25
Nodes (8): 6.1 Database provisioning and isolation, 6.2 sqlc queries and repositories, 6.3 Migrations, 6.4 PostGIS serviceability with Mahabubnagar fixtures, 6.6 Concurrency and race tests, 6.7 Per-subscriber jobs (River as outbox, R22/R42) and invariants, 6.8 SSE, 6. Go integration tests

### Community 129 - "14. Service-by-service comparison (India regions)"
Cohesion: 0.22
Nodes (9): 14.1 Region availability & DR pairing, 14.2 Managed containers (always-on worker + SSE), 14.3 Managed PostgreSQL + PostGIS, 14.4 Cache (only when P6 triggers), 14.5 Object storage, CDN, WAF, secrets, KMS, LB, NAT, 14.6 Observability (native vs Grafana Cloud), 14.7 Startup credits (all fetched 2026-10-04), 14.8 Portability / lock-in (+1 more)

### Community 130 - "planning/README.md"
Cohesion: 0.22
Nodes (8): 0. How to read this backlog, 1. Epic overview, 27 — Implementation Backlog (Phase 2), Appendix A — Requirement coverage map (P0 + committed P1), Appendix B — Effort roll-up (engineering stories only), Appendix C — Not in this backlog (by design), Appendix D — Stretch P1 handling, Appendix E — Lead's missing items (baseline §9, M1–M17) → stories

### Community 131 - "exports"
Cohesion: 0.22
Nodes (9): exports, ./eslint, ./fonts.css, ./theme.css, ./tokens, ./tsconfig/base.json, ./tsconfig/node.json, ./tsconfig/react.json (+1 more)

### Community 132 - "Phase 2 — Implementation Charter (binding for all implementation agents)"
Cohesion: 0.25
Nodes (7): 1. Workflow, 2. Toolchain (pinned), 3. V1 implementation slice (what "Phase 2 done" means here), 4. Conventions, 5. Agent ownership (avoid edit collisions), 6. Deliberate implementation deviations (ADRs), Phase 2 — Implementation Charter (binding for all implementation agents)

### Community 133 - "rovo — System Knowledge Graph"
Cohesion: 0.22
Nodes (8): 1. System graph (actors → apps → API → modules → stores), 2. Modules ↔ owned tables (verified from migrations), 3. Domain‑event graph (the golden flow), 4. Order ↔ delivery state machines (reference), 5. API surface (OpenAPI tag groups → module), 6. External systems (trust boundaries), 7. Graphify (machine-queryable graph), rovo — System Knowledge Graph

### Community 134 - "7. Functional requirements — Delivery partner (RDR-*)"
Cohesion: 0.25
Nodes (8): 7.1 RDR-ONB — Registration and KYC, 7.2 RDR-AVAIL — Availability (online/offline), 7.3 RDR-ASSIGN — Order offers and assignment, 7.4 RDR-FLOW — Pickup and delivery workflow, 7.5 RDR-CONT — Contacting customer and restaurant, 7.6 RDR-HIST — Delivery history, 7.7 RDR-EARN — Earnings and cash-in-hand, 7. Functional requirements — Delivery partner (RDR-*)

### Community 135 - "1. UX principles (apply to all rovo apps: customer, restaurant, rider, admin — R14)"
Cohesion: 0.25
Nodes (8): 1.1 Mobile-first at 360 px, 1.2 Low bandwidth and low-end devices, 1.3 Telugu typography and bilingual content, 1.4 Accessibility — WCAG 2.2 AA, 1.5 Offline and poor-network messaging, 1.6 Fee transparency (non-negotiable) `[LEGAL]`, 1.7 Tone and trust, 1. UX principles (apply to all rovo apps: customer, restaurant, rider, admin — R14)

### Community 136 - "15. Cost estimates (on-demand, excl. GST; ₹95/USD) — single source of cost (R46)"
Cohesion: 0.25
Nodes (8): 15.1 AWS production profiles (R32 / M17), 15.2 AWS: 10× growth (≈ 5,000–20,000 orders/day, ~1,000 riders, ~15k SSE connections), 15.3 GCP alternative (asia-south1), 15.4 Azure (Central India), for completeness, 15.5 Other cost lines: non-AWS, one-time, and cut items (RV-014), 15.6 Per-order cloud cost by phase (R45 volumes), 15.7 CDN request-volume model and request reduction (M13, R27), 15. Cost estimates (on-demand, excl. GST; ₹95/USD) — single source of cost (R46)

### Community 137 - "aws_cloudfront_distribution.this"
Cohesion: 0.12
Nodes (10): 1. Read first, in this order (this is your spec), 2. Hard constraints, 33 — Phase 2 Implementation Prompt, 4. Domain rules you must implement exactly, 5. How to work, 6. Definition of done for V1 development (end of M5, local), 7. Decisions that need humans (do not block on them; use the stated default), 8. Reporting (+2 more)

### Community 138 - "smoke.sh"
Cohesion: 0.47
Nodes (8): checkfail(), checkpass(), env_default(), hdr(), req(), smoke.sh script, want_header(), want_status()

### Community 139 - "react.json"
Cohesion: 0.25
Nodes (7): compilerOptions, jsx, lib, types, extends, ./base.json, $schema

### Community 140 - "catalog"
Cohesion: 0.04
Nodes (55): catalog, axe-core, clsx, eslint, @eslint/js, eslint-plugin-jsx-a11y, eslint-plugin-react-hooks, eslint-plugin-react-refresh (+47 more)

### Community 141 - "Runbook: Incident response"
Cohesion: 0.05
Nodes (41): After restoring real (non-synthetic) data, Alert A16 — no successful backup in 26 h, Restore a backup, Runbook: Backup & restore, Take a backup, Verify (restore drill D0), Cloud (AWS ECS/Fargate) — summary, If it goes wrong (+33 more)

### Community 142 - "rovo brand assets"
Cohesion: 0.29
Nodes (6): Accessibility rules (WCAG 2.2 AA), Colour tokens, Licensing note, Outstanding brand work, rovo brand assets, Usage

### Community 143 - "4. Success metrics and V1 targets"
Cohesion: 0.29
Nodes (7): 4.1 Operational quality (primary), 4.2 Money correctness, 4.3 App quality and experience, 4.4 Satisfaction and support, 4.5 Growth and supply health (directional; not launch gates), 4.6 Running cost (business line item), 4. Success metrics and V1 targets

### Community 144 - "8. Checkout and payment"
Cohesion: 0.29
Nodes (7): 8.1 Address add / select (C-11, C-12), 8.2 Checkout screen (C-13), 8.3 Place order and payment sequence, 8.4 Price-change handling (C-13b "What changed" sheet), 8.5 Payment failure and retry (C-14), 8.6 Order placed confirmation (C-15), 8. Checkout and payment

### Community 145 - "5. Key runtime sequences"
Cohesion: 0.29
Nodes (7): 5.1 Place order: online payment (UPI / card / netbanking), 5.2 COD order, 5.3 Restaurant accept (with the R1 accept ladder), 5.4 Dispatch offer cascade with timeout, 5.5 Pickup → delivery → settlement entries, 5.6 Refund on rejection (prepaid order), 5. Key runtime sequences

### Community 146 - "4. Go unit tests"
Cohesion: 0.29
Nodes (7): 4.1 Conventions, 4.2 Order and delivery state machines (T1), 4.3 Pricing / quote engine golden tables (T1), 4.4 Commission, rider pay, payouts (T1), 4.5 Ledger (T1), 4.6 Other T1/T2 unit areas, 4. Go unit tests

### Community 147 - "10. Capacity model (reference only — the load model is owned by doc 20)"
Cohesion: 0.29
Nodes (7): 10.1 Load assumptions [ASSUMPTION — superseded by doc 20 §12.1 / R45], 10.2 Compute, 10.3 Database size growth, 10.4 Object storage, 10.5 Backups, 10.6 Bandwidth, logs, metrics, traces, CI, 10. Capacity model (reference only — the load model is owned by doc 20)

### Community 148 - "config/tsconfig.json"
Cohesion: 0.29
Nodes (6): ./tsconfig/node.json, compilerOptions, allowJs, checkJs, extends, include

### Community 149 - "customer/tsconfig.json"
Cohesion: 0.29
Nodes (6): compilerOptions, lib, types, extends, include, @rovo/config/tsconfig/react.json

### Community 150 - "restaurant/tsconfig.json"
Cohesion: 0.29
Nodes (6): compilerOptions, lib, types, extends, include, @rovo/config/tsconfig/react.json

### Community 151 - "rider/tsconfig.json"
Cohesion: 0.29
Nodes (6): compilerOptions, lib, types, extends, include, @rovo/config/tsconfig/react.json

### Community 152 - "node.json"
Cohesion: 0.29
Nodes (6): compilerOptions, lib, types, extends, ./base.json, $schema

### Community 153 - "PostGIS image for local dev and CI"
Cohesion: 0.33
Nodes (5): Alternative considered: build our own, Decision: `imresamu/postgis` for multi-arch, official on amd64, How it is selected, PostGIS image for local dev and CI, The problem: the official image is amd64-only

### Community 154 - "13. Help and support for an order (C-21)"
Cohesion: 0.33
Nodes (6): 13.1 Entry points, 13.2 Order-specific issue tree (self-serve first), 13.3 Refund messaging (standard block), 13.4 Ticket UI (C-22), 13.5 Ops-assisted phone ordering (P1, M8; 01 ADM-ORD-011), 13. Help and support for an order (C-21)

### Community 155 - "9. Deployment topology and scaling path"
Cohesion: 0.33
Nodes (6): 9.1 Local and CI, 9.2 Production at launch: two IaC profiles (R32), 9.3 Scale horizontally (same architecture), 9.4 Scale the data tier, 9.5 Selective extraction (only if needed), 9. Deployment topology and scaling path

### Community 156 - "with"
Cohesion: 0.05
Nodes (41): with, build-args, cache, cache-dependency-path, cache-from, cache-to, context, directory (+33 more)

### Community 157 - "10. Incident response outline"
Cohesion: 0.33
Nodes (6): 10.1 Roles, 10.2 Severity, 10.3 Timeline, 10.4 Playbooks (written in the private ops repo before launch), 10.5 Contacts registry (fill before launch), 10. Incident response outline

### Community 158 - "database/main.tf"
Cohesion: 0.33
Nodes (6): 11.1 Emulated low-end device profiles (Playwright + Chromium CDP), 11.2 Offline and flaky-network behaviour, 11.3 Service-worker update flow, 11.4 Web Push, 11.5 Real-device smoke checklist (pre-release, manual, ~45 min per device), 11. Mobile / PWA tests

### Community 159 - "14. Security testing"
Cohesion: 0.33
Nodes (6): 14.1 OWASP ASVS 5.0 Level 2 mapping, 14.2 Automated scanning in CI, 14.3 Authorisation, 14.4 Rate-limit tests, 14.5 Webhook replay and spoofing, 14. Security testing

### Community 160 - "15. Data, migration and backup-restore testing"
Cohesion: 0.33
Nodes (6): 15.1 Migrations (goose), 15.2 Data integrity checks, 15.3 Backup restore verification (R50: weekly automated, monthly manual drill), 15.4 Erasure map tests (M15, RV-031) `[LEGAL]`, 15.5 CERT-In log archive verification (M1), 15. Data, migration and backup-restore testing

### Community 161 - "7. Contract tests"
Cohesion: 0.33
Nodes (6): 7.1 OpenAPI as the enforced contract, 7.2 Provider adapter architecture for testability, 7.3 Payment webhook contract tests, 7.4 OTP / SMS / push provider contracts, 7.5 Golden documents and PA files (M14, RV-063, RV-066), 7. Contract tests

### Community 162 - "9. Frontend tests"
Cohesion: 0.33
Nodes (6): 9.1 Unit / component (Vitest + Testing Library), 9.2 API mocking (MSW), 9.3 i18n completeness, 9.4 Accessibility (automated part), 9.5 Visual regression (decision), 9. Frontend tests

### Community 163 - "00007_dispatch.sql"
Cohesion: 0.14
Nodes (28): contact_tap_log, deliveries, delivery_offers, delivery_status_history, ix_contact_tap_log__delivery, ix_deliveries__dispatch_queue, ix_deliveries__rider_history, ix_deliveries__undeliverable (+20 more)

### Community 164 - "utils/tsconfig.json"
Cohesion: 0.33
Nodes (5): @rovo/config/tsconfig/base.json, compilerOptions, lib, extends, include

### Community 165 - "verify-restore.sh"
Cohesion: 0.47
Nodes (4): check_fail(), meta_get(), PG_CONTAINER, verify-restore.sh script

### Community 166 - "scripts"
Cohesion: 0.33
Nodes (6): scripts, build, dev, preview, test, typecheck

### Community 167 - "admin/tsconfig.json"
Cohesion: 0.33
Nodes (5): compilerOptions, types, extends, include, @rovo/config/tsconfig/react.json

### Community 168 - "scripts"
Cohesion: 0.33
Nodes (6): scripts, build, dev, preview, test, typecheck

### Community 169 - "scripts"
Cohesion: 0.33
Nodes (6): scripts, build, dev, preview, test, typecheck

### Community 170 - "scripts"
Cohesion: 0.33
Nodes (6): scripts, build, dev, preview, test, typecheck

### Community 171 - "auth/tsconfig.json"
Cohesion: 0.33
Nodes (5): compilerOptions, types, extends, include, @rovo/config/tsconfig/react.json

### Community 172 - "shell/tsconfig.json"
Cohesion: 0.33
Nodes (5): compilerOptions, types, extends, include, @rovo/config/tsconfig/react.json

### Community 173 - "testing/tsconfig.json"
Cohesion: 0.33
Nodes (5): compilerOptions, types, extends, include, @rovo/config/tsconfig/react.json

### Community 174 - "ui/tsconfig.json"
Cohesion: 0.33
Nodes (5): compilerOptions, types, extends, include, @rovo/config/tsconfig/react.json

### Community 175 - "PROMPT — rovo Phase 2: implement V1"
Cohesion: 0.15
Nodes (24): aws_db_instance.this, aws_db_parameter_group.this, aws_db_subnet_group.this, aws_security_group.db, Terraform module: deploy/terraform/modules/database, output.db_name, output.endpoint, output.master_user_secret_arn (+16 more)

### Community 176 - "rovo infrastructure (OpenTofu)"
Cohesion: 0.40
Nodes (4): Applying later (M6, when AWS exists), Layout, rovo infrastructure (OpenTofu), Validate (never apply)

### Community 177 - "ADR-P2-001: OpenAPI is the contract; no Go server code generation in the first slice"
Cohesion: 0.29
Nodes (6): ADR-P2-001: OpenAPI is the contract; no Go server code generation in the first slice, Consequences, Context, Decision, Lint policy, Revisit when

### Community 178 - "4. Home / discovery (C-05)"
Cohesion: 0.40
Nodes (5): 4.1 Layout (top → bottom at 360 px), 4.2 Sort options, 4.3 Filters, 4.4 Restaurant availability states in lists, 4. Home / discovery (C-05)

### Community 179 - "7. Consistency, idempotency and concurrency"
Cohesion: 0.40
Nodes (5): 7.1 Idempotency-Key on POST, 7.2 Concurrency control, 7.3 Transactional events (outbox) detail — amended 2026-10-04 per R22/R42, 7.4 Periodic jobs: catch-up semantics (M11), 7. Consistency, idempotency and concurrency

### Community 180 - "cities"
Cohesion: 0.17
Nodes (25): ix_pa_settlement_lines__entity, ix_pa_settlement_lines__unmatched, ix_payment_events__payment, ix_payments__order, ix_payments__pending, ix_recon_exceptions__queue, ix_refunds__open, ix_refunds__order (+17 more)

### Community 181 - "storage/main.tf"
Cohesion: 0.40
Nodes (5): 21.1 What runs when, 21.2 Coverage targets (measured, gated by module), 21.3 Runtime budgets and how we keep them, 21.4 Flaky-test policy, 21. CI gating, coverage, flaky tests, runtime budgets

### Community 182 - "8. Authorisation tests"
Cohesion: 0.40
Nodes (5): 8.1 Generated role × operation matrix, 8.2 IDOR suites per resource (hand-written, T1), 8.3 Session and token tests, 8.4 Maker-checker and audit tests (doc 12 §5.5, §5.7), 8. Authorisation tests

### Community 183 - "Store"
Cohesion: 0.18
Nodes (23): aws_iam_openid_connect_provider.github, aws_iam_role.deploy, aws_iam_role.ecr_push, aws_iam_role_policy.deploy, aws_iam_role_policy.ecr_push, data.aws_caller_identity.current, data.aws_iam_policy_document.deploy, data.aws_iam_policy_document.deploy_trust (+15 more)

### Community 184 - "steps"
Cohesion: 0.05
Nodes (37): name, permissions, runs-on, steps, strategy, concurrency, cancel-in-progress, group (+29 more)

### Community 185 - "7. CI/CD, DNS/CDN/WAF, monitoring, email"
Cohesion: 0.40
Nodes (5): 7.1 CI/CD and registry, 7.2 DNS / CDN / WAF / Zero-Trust (Cloudflare Free), 7.3 Observability & alerting, 7.4 Transactional email (brief, Solution Architect owns `15`), 7. CI/CD, DNS/CDN/WAF, monitoring, email

### Community 186 - "dev-keys.sh"
Cohesion: 0.60
Nodes (3): gen_for(), rand_b64(), dev-keys.sh script

### Community 187 - ".prettierrc.json"
Cohesion: 0.40
Nodes (4): printWidth, semi, singleQuote, trailingComma

### Community 188 - "6. Login (deferred) and profile"
Cohesion: 0.50
Nodes (4): 6.1 Phone + OTP (C-08, C-09), 6.2 Guest cart merge, 6.3 Profile (C-20), 6. Login (deferred) and profile

### Community 189 - "7. Restaurant page, customisation, cart"
Cohesion: 0.50
Nodes (4): 7.1 Restaurant page (C-07), 7.2 Item customisation sheet (C-07a), 7.3 Cart (C-10), 7. Restaurant page, customisation, cart

### Community 190 - "9. Order tracking (C-16)"
Cohesion: 0.50
Nodes (4): 9.1 Layout, 9.2 What the customer sees for each status combination, 9.3 Tracking flow (customer-visible milestones), 9. Order tracking (C-16)

### Community 191 - "react"
Cohesion: 0.29
Nodes (7): react@catalog:, react-dom@catalog:, react, react-dom, peerDependencies, react@^19.0.0, react-dom@^19.0.0

### Community 192 - "NotImplemented"
Cohesion: 0.12
Nodes (14): ActionType, ActorType, Approval, ApprovalRequest, ApprovalStatus, AuditFilter, AuditPage, ConfigScope (+6 more)

### Community 193 - ".golangci.yml"
Cohesion: 0.33
Nodes (5): linters, settings, staticcheck, checks, version

### Community 194 - "15. Environment config, build and deploy outputs"
Cohesion: 0.15
Nodes (22): aws_s3_bucket_lifecycle_configuration.kyc, aws_s3_bucket_lifecycle_configuration.logs, aws_s3_bucket_lifecycle_configuration.media, aws_s3_bucket_lifecycle_configuration.web, aws_s3_bucket_policy.this, aws_s3_bucket_public_access_block.this, aws_s3_bucket_server_side_encryption_configuration.this, aws_s3_bucket.this (+14 more)

### Community 195 - ".golangci.yml"
Cohesion: 0.50
Nodes (4): 22.1 Definition of Done (story level), 22.2 Bug severity and triage SLAs, 22.3 Release quality gates (feeds doc 29 production readiness), 22. Definition of Done, bug severity, triage SLAs, release gates

### Community 196 - "3. PostgreSQL (PostGIS mandatory)"
Cohesion: 0.50
Nodes (4): 3.1 Comparison table, 3.2 Why self-hosted Postgres on the VM is acceptable for preview (and not for production), 3.3 Why the baseline's "managed free fallback" does not work, 3. PostgreSQL (PostGIS mandatory)

### Community 197 - "9. Dev / preview / demo recommendation"
Cohesion: 0.50
Nodes (4): 9.1 Primary preview stack (₹0/month + domain), 9.2 Fallback preview options, 9.3 Re-verification cadence, 9. Dev / preview / demo recommendation

### Community 198 - "tsconfig.node.json"
Cohesion: 0.50
Nodes (3): @rovo/config/tsconfig/node.json, extends, include

### Community 199 - "backup.sh"
Cohesion: 0.83
Nodes (3): cleanup_snapshot(), backup.sh script, snap_query()

### Community 202 - "ADR-P2-003: Production-mode Docker Compose is the deployment target for this phase"
Cohesion: 0.33
Nodes (5): ADR-P2-003: Production-mode Docker Compose is the deployment target for this phase, Consequences, Context, Decision, Revisit when

### Community 203 - "api-client/tsconfig.json"
Cohesion: 0.50
Nodes (3): extends, include, @rovo/config/tsconfig/react.json

### Community 204 - "config/package.json"
Cohesion: 0.22
Nodes (9): devDependencies, @types/node@catalog:, typescript@catalog:, @typescript/native@catalog:, vitest@catalog:, @types/node, typescript, @typescript/native (+1 more)

### Community 205 - "Registry"
Cohesion: 0.47
Nodes (3): NewRegistry(), Registry, Subscriber

### Community 206 - "i18n/tsconfig.json"
Cohesion: 0.50
Nodes (3): extends, include, @rovo/config/tsconfig/react.json

### Community 207 - "rovo"
Cohesion: 0.67
Nodes (3): 5.1 Behaviour, 5.2 Telugu and transliteration, 5. Search (C-06)

### Community 208 - "3. Admin authentication"
Cohesion: 0.33
Nodes (6): 3.1 Credentials, 3.2 TOTP (RFC 6238), 3.3 Provisioning, reset, offboarding, 3.4 Sessions, timeouts and step-up, 3.5 Network restrictions for the admin console (R37), 3. Admin authentication

### Community 209 - "customer/src/router.ts"
Cohesion: 0.07
Nodes (35): RootLayout(), TABS, accountRoute, addressEditRoute, addressesRoute, addressNewRoute, cartRoute, checkoutRoute (+27 more)

### Community 210 - "00009_ratings_support_notifications.sql"
Cohesion: 0.14
Nodes (23): ix_notification_deliveries__notification, ix_notifications__inbox, ix_notifications__unread, ix_push_subscriptions__user, ix_ratings__restaurant, ix_ratings__rider, ix_reviews__restaurant_public, ix_support_tickets__order (+15 more)

### Community 211 - "00010_admin_platform.sql"
Cohesion: 0.13
Nodes (20): app_config, approval_requests, audit_logs, feature_flags, file_objects, idempotency_keys, ix_approval_requests__post_review, ix_approval_requests__queue (+12 more)

### Community 212 - "13. Accessibility, analytics, error tracking, feature flags, testing hooks"
Cohesion: 0.33
Nodes (6): 13.1 Accessibility (WCAG 2.2 AA), 13.2 Analytics (privacy-friendly, DPDP), 13.3 Error tracking, 13.4 Feature flags and remote config, 13.5 Testing hooks, 13. Accessibility, analytics, error tracking, feature flags, testing hooks

### Community 213 - "x-app-env"
Cohesion: 0.06
Nodes (31): x-app-env, OTEL_EXPORTER_OTLP_ENDPOINT, ROVO_ADMIN_BOOTSTRAP_PASSWORD, ROVO_APP_HOSTS, ROVO_COOKIE_SECURE, ROVO_CUSTOMER_BASE_URL, ROVO_DATABASE_URL, ROVO_DOMAIN (+23 more)

### Community 216 - "scripts"
Cohesion: 0.67
Nodes (3): scripts, test, typecheck

### Community 217 - "Module"
Cohesion: 0.11
Nodes (15): otpSender, SMSSender, New(), Outbox, Channel, FakePush, FakeSMS, InboxPage (+7 more)

### Community 218 - "Point"
Cohesion: 0.07
Nodes (26): containsFold(), containsSubstr(), hoursView(), sortCards(), ServiceabilityView, Coverage, Service, New() (+18 more)

### Community 219 - "jobs"
Cohesion: 0.09
Nodes (24): if, needs, runs-on, if, needs, permissions, runs-on, jobs (+16 more)

### Community 220 - "phone.ts"
Cohesion: 0.19
Nodes (4): Store, New(), ref(), Store

### Community 221 - "services"
Cohesion: 0.09
Nodes (21): internal, name, networks, data, edge, command, image, networks (+13 more)

### Community 222 - "react"
Cohesion: 0.29
Nodes (7): react@catalog:, react-dom@catalog:, react, react-dom, peerDependencies, react@^19.0.0, react-dom@^19.0.0

### Community 223 - "scripts"
Cohesion: 0.67
Nodes (3): scripts, test, typecheck

### Community 225 - "rovo"
Cohesion: 0.06
Nodes (31): Code of Conduct, Enforcement, Our pledge, Scope, Coding standards, Commits and pull requests, Contributing to rovo, Licensing & DCO (+23 more)

### Community 228 - "updates"
Cohesion: 0.09
Nodes (21): patterns, prefix, update-types, actions, go-minor, web-minor, day, interval (+13 more)

### Community 229 - "Export"
Cohesion: 0.23
Nodes (8): New(), Export, ExportRequest, ExportStatus, Period, ReportType, RestaurantSummary, Service

### Community 232 - "00004_pricing_promotions.sql"
Cohesion: 0.21
Nodes (16): cities, coupon_redemptions, coupon_targets, coupons, fee_configs, ix_coupon_redemptions__user, ix_coupon_targets__target, ix_coupons__live (+8 more)

### Community 233 - "Sequence"
Cohesion: 0.22
Nodes (5): Generator, RandomCrockford(), TestDefault(), Default, Sequence

### Community 234 - "name"
Cohesion: 0.22
Nodes (19): steps, steps, steps, steps, GITHUB_TOKEN, steps, steps, steps (+11 more)

### Community 235 - "1. Conventions"
Cohesion: 0.18
Nodes (10): New(), AccountCode, AccountRef, EntryType, Journal, OwnerType, PayoutBatch, Posting (+2 more)

### Community 236 - "orders"
Cohesion: 0.21
Nodes (16): assert_order_bill_balanced(), ix_order_charges__order, ix_order_status_history__occurred, ix_orders__city_created, ix_orders__city_live, ix_orders__customer, ix_orders__restaurant, ix_orders__restaurant_live (+8 more)

### Community 237 - "uuidv7"
Cohesion: 0.18
Nodes (11): RFC-9562, ADR-005: PostgreSQL 18 + PostGIS; pgx + sqlc; goose migrations, ADR-019: UUIDv7 primary keys, 0. Decision summary, 20. Sources (accessed 2026-10-04), 2.1 `packages/api-client`, 2.2 `packages/utils` (selected functions), 2. Workspace layout (pnpm, under `web/` per R14) (+3 more)

### Community 238 - "lint-meta"
Cohesion: 0.57
Nodes (6): E164, formatIndianMobile(), isValidIndianMobile(), maskIndianMobile(), normalizeIndianMobile(), toNationalMobile()

### Community 239 - "rateLimitSweepWorker"
Cohesion: 0.17
Nodes (8): NewFakePA(), ADR-012: Payment aggregator choice and abstraction, 0. Summary, 2. Naming conventions, 10. Vendor lock-in and the portability ADR, FakePA, Provider, ProviderEvent

### Community 240 - "docker"
Cohesion: 0.33
Nodes (6): if, needs, permissions, runs-on, docker, packages

### Community 241 - "minio"
Cohesion: 0.15
Nodes (13): MINIO_BROWSER, MINIO_REGION, MINIO_ROOT_PASSWORD, MINIO_ROOT_USER, command, deploy, environment, image (+5 more)

### Community 242 - "api"
Cohesion: 0.12
Nodes (16): cap_drop, command, depends_on, expose, image, networks, ports, read_only (+8 more)

### Community 243 - "attributes"
Cohesion: 0.12
Nodes (15): description, label, options, placeholder, render, value, body, attributes (+7 more)

### Community 245 - "go"
Cohesion: 0.12
Nodes (16): POSTGRES_DB, POSTGRES_PASSWORD, POSTGRES_USER, ROVO_TEST_DATABASE_URL, env, if, needs, permissions (+8 more)

### Community 246 - "Ticket"
Cohesion: 0.21
Nodes (7): New(), Category, OpenRequest, Resolution, Service, Status, Ticket

### Community 247 - "worker"
Cohesion: 0.13
Nodes (15): migrate, condition, worker, cap_drop, command, depends_on, expose, image (+7 more)

### Community 256 - "migrate"
Cohesion: 0.13
Nodes (15): postgis, disable, cap_drop, command, depends_on, healthcheck, image, networks (+7 more)

### Community 257 - "12. Performance and load tests"
Cohesion: 0.40
Nodes (5): 5.1 Cart decision: device cart + persisted quote (Lead ruling 12), 5.2 Fee configs (versioned per city / zone), 5.3 Tax rules [LEGAL], 5.4 Promotions, 5. DDL — quote & pricing, promotions

### Community 258 - "1. Framework evaluation (challenge of P7)"
Cohesion: 0.40
Nodes (5): 1.1 What the customer app actually needs, 1.2 Options compared, 1.3 Recommendation (firm; amended in v1.1 by R33), 1.4 App split decision, 1. Framework evaluation (challenge of P7)

### Community 259 - "3. Routing maps"
Cohesion: 0.40
Nodes (5): 3.1 Customer (`apps/customer`, origin `app.<domain>`; apex redirects here [OPEN]), 3.2 Restaurant (`apps/restaurant`, origin `restaurant.<domain>`, UI name "rovo Partner"), 3.3 Rider (`apps/rider`, origin `rider.<domain>`, UI name "rovo Delivery Partner"), 3.4 Admin (`apps/admin`, origin `admin.<domain>`), 3. Routing maps

### Community 260 - "grafana"
Cohesion: 0.14
Nodes (14): GF_ANALYTICS_CHECK_FOR_UPDATES, GF_ANALYTICS_REPORTING_ENABLED, GF_AUTH_ANONYMOUS_ENABLED, GF_AUTH_ANONYMOUS_ORG_ROLE, GF_NEWS_NEWS_FEED_ENABLED, GF_SECURITY_ADMIN_PASSWORD, depends_on, environment (+6 more)

### Community 261 - "caddy"
Cohesion: 0.14
Nodes (14): condition, cap_add, cap_drop, depends_on, image, networks, ports, read_only (+6 more)

### Community 262 - "postgis"
Cohesion: 0.09
Nodes (23): deploy, deploy, resources, POSTGRES_DB, POSTGRES_INITDB_ARGS, POSTGRES_PASSWORD, POSTGRES_USER, TZ (+15 more)

### Community 263 - "global"
Cohesion: 0.14
Nodes (13): env, global, evaluation_interval, external_labels, scrape_interval, service, rule_files, scrape_configs (+5 more)

### Community 264 - "rules"
Cohesion: 0.14
Nodes (13): description, runbook_url, summary, groups, name, rules, severity, team (+5 more)

### Community 265 - "args"
Cohesion: 0.26
Nodes (13): build, BUILD_TAGS, COMMIT, VERSION, args, context, dockerfile, build (+5 more)

### Community 266 - "options"
Cohesion: 0.19
Nodes (13): logging, driver, options, logging, max-file, max-size, logging, x-hardening (+5 more)

### Community 267 - "grafana"
Cohesion: 0.15
Nodes (13): GF_ANALYTICS_CHECK_FOR_UPDATES, GF_ANALYTICS_REPORTING_ENABLED, GF_NEWS_NEWS_FEED_ENABLED, GF_SECURITY_ADMIN_PASSWORD, depends_on, environment, image, networks (+5 more)

### Community 268 - "global"
Cohesion: 0.15
Nodes (12): env, global, evaluation_interval, external_labels, scrape_interval, service, rule_files, scrape_configs (+4 more)

### Community 269 - "body"
Cohesion: 0.15
Nodes (12): description, label, value, body, attributes, id, type, validations (+4 more)

### Community 270 - "ci.yml"
Cohesion: 0.15
Nodes (12): concurrency, cancel-in-progress, group, env, GO_VERSION_FILE, POSTGIS_IMAGE, name, on (+4 more)

### Community 271 - "10. End-to-end tests"
Cohesion: 0.40
Nodes (5): 10.1 Stack and conventions, 10.2 Time control in E2E, 10.3 Golden flow (multi-actor, 4 browser contexts), 10.4 Failure scenarios, 10. End-to-end tests

### Community 272 - "lint-meta"
Cohesion: 0.40
Nodes (5): lint-meta, if, needs, permissions, runs-on

### Community 273 - "postgis"
Cohesion: 0.17
Nodes (12): POSTGRES_DB, POSTGRES_INITDB_ARGS, POSTGRES_PASSWORD, POSTGRES_USER, TZ, command, environment, image (+4 more)

### Community 274 - "rules"
Cohesion: 0.17
Nodes (11): extends, rules, no-identical-paths, no-invalid-media-type-examples, no-unused-components, operation-4xx-response, operation-operationId, operation-operationId-unique (+3 more)

### Community 275 - "healthcheck"
Cohesion: 0.49
Nodes (11): healthcheck, healthcheck, interval, retries, start_period, test, timeout, healthcheck (+3 more)

### Community 276 - "services"
Cohesion: 0.20
Nodes (9): name, services, volumes, grafanadata, miniodata, pgdata, promdata, x-restart (+1 more)

### Community 277 - "mailpit"
Cohesion: 0.20
Nodes (10): MP_DISABLE_VERSION_CHECK, MP_SMTP_AUTH_ACCEPT_ANY, MP_SMTP_AUTH_ALLOW_INSECURE, environment, image, networks, ports, restart (+2 more)

### Community 278 - "providers"
Cohesion: 0.20
Nodes (9): apiVersion, path, providers, allowUiUpdates, disableDeletion, folder, name, options (+1 more)

### Community 279 - "datasources"
Cohesion: 0.20
Nodes (9): apiVersion, datasources, access, editable, isDefault, name, type, uid (+1 more)

### Community 280 - "web"
Cohesion: 0.40
Nodes (5): web, if, needs, permissions, runs-on

### Community 281 - "outputs"
Cohesion: 0.20
Nodes (10): outputs, permissions, runs-on, changes, backend, deploy, docker, openapi (+2 more)

### Community 282 - "healthcheck"
Cohesion: 0.47
Nodes (9): interval, retries, start_period, test, timeout, healthcheck, healthcheck, healthcheck (+1 more)

### Community 283 - "minio-init"
Cohesion: 0.25
Nodes (8): minio, condition, depends_on, entrypoint, image, restart, volumes, minio-init

### Community 284 - "mailpit"
Cohesion: 0.25
Nodes (8): MP_DISABLE_VERSION_CHECK, MP_SMTP_AUTH_ACCEPT_ANY, MP_SMTP_AUTH_ALLOW_INSECURE, environment, image, ports, restart, mailpit

### Community 285 - "prometheus"
Cohesion: 0.25
Nodes (8): command, extra_hosts, image, ports, profiles, restart, volumes, prometheus

### Community 286 - "minio"
Cohesion: 0.17
Nodes (13): MINIO_REGION, MINIO_ROOT_PASSWORD, MINIO_ROOT_USER, ROVO_S3_BUCKET_KYC, ROVO_S3_BUCKET_MEDIA, command, environment, image (+5 more)

### Community 287 - "groups"
Cohesion: 0.29
Nodes (6): groups, interval, name, rules, expr, record

### Community 288 - "prod/.terraform.lock.hcl"
Cohesion: 0.29
Nodes (7): 1.1 Naming, 1.2 Standard columns and triggers, 1.3 Database roles (managed-service friendly: no superuser required), 1.4 Soft delete vs. status policy, 1.5 Module ownership (08 §3), 1.6 Name reconciliation with sibling drafts, 1. Conventions

### Community 289 - "staging/.terraform.lock.hcl"
Cohesion: 0.29
Nodes (7): 2.1 Geo, identity, users, 2.2 Catalog (restaurants and menus), 2.3 Quote, pricing, promotions, ordering, 2.4 Payments, dispatch, 2.5 Ledger, payouts, engagement, support, notifications, platform, 2.6 PA settlement & reconciliation, privacy and growth (added in v1.1), 2. Entity-relationship diagrams (by module)

### Community 290 - "Competitor "paved path" review — Zomato, Swiggy, Toing"
Cohesion: 0.29
Nodes (6): Competitor "paved path" review — Zomato, Swiggy, Toing, Confirmed: our dispatch timing matches Swiggy's published model, Findings to act on, Sources (accessed 2026-11-12), Unverified / limits, Verdict

### Community 291 - "5. Search (C-06)"
Cohesion: 0.33
Nodes (5): New(), Order, PlaceRequest, Service, Status

### Community 293 - "20. Testability hooks required from backend and frontend"
Cohesion: 0.67
Nodes (3): 20.1 Backend, 20.2 Frontend, 20. Testability hooks required from backend and frontend

### Community 294 - "valkey"
Cohesion: 0.33
Nodes (6): valkey, command, image, ports, profiles, restart

### Community 295 - ".checkov.yaml"
Cohesion: 0.33
Nodes (5): compact, directory, framework, quiet, skip-check

### Community 296 - "12. Taxes, invoices and compliance `[LEGAL]`"
Cohesion: 0.33
Nodes (6): 12.1 GST, 12.2 Income-tax TDS §194-O, 12.3 Data model for tax, 12.4 Rounding, 12.5 Invoices per order (customer), 12. Taxes, invoices and compliance `[LEGAL]`

### Community 297 - "1. Top-level layout"
Cohesion: 0.33
Nodes (6): 1.1 `backend/` (Go module), 1.2 `openapi/`, 1.3 `web/` (pnpm workspace; structure from doc 17 §2), 1.4 `deploy/`, 1.5 `tools/` and `scripts/`, 1. Top-level layout

### Community 298 - "contact_links"
Cohesion: 0.33
Nodes (5): blank_issues_enabled, contact_links, about, name, url

### Community 299 - "23. Traceability"
Cohesion: 0.67
Nodes (3): 23.1 Requirement ID prefixes → suites, 23.2 Tagging convention and report, 23. Traceability

### Community 300 - "10. Ledger design (double-entry-lite)"
Cohesion: 0.40
Nodes (5): 10.1 Structure, 10.2 Chart of accounts (V1), 10.3 Worked example (prepaid order) — redone 2026-10-04 per R8 (RV-046), 10.4 Other events, 10. Ledger design (double-entry-lite)

### Community 301 - "pull_request_template.md"
Cohesion: 0.40
Nodes (4): Checklist, Screenshots / recordings, Summary, Type

### Community 302 - "2. Quality goals and risk-based prioritisation"
Cohesion: 0.67
Nodes (3): 2.1 Quality goals (what "good" means for V1), 2.2 Risk register driving test intensity, 2. Quality goals and risk-based prioritisation

### Community 303 - "scripts"
Cohesion: 0.67
Nodes (3): scripts, test, typecheck

### Community 304 - "openapi.yaml"
Cohesion: 0.40
Nodes (4): openapi.yaml, no-unused-components, operation-2xx-response, operation-4xx-response

### Community 306 - "11. COD flows"
Cohesion: 0.50
Nodes (4): 11.1 Sequence, 11.2 Rules, 11.3 Customer pays COD by UPI at the door — **deferred to V1.1 (scope cut C19)**, 11. COD flows

### Community 307 - "19. Test environments and provider fakes"
Cohesion: 0.50
Nodes (4): 19.1 Environments (baseline §4a), 19.2 Provider fakes (wire-level, shipped as small Go services in the repo), 19.3 Post-deploy smoke and synthetic checks, 19. Test environments and provider fakes

### Community 308 - "13. Scope cut list and missing items"
Cohesion: 0.50
Nodes (4): 13.1 Scope cut list (defer to V1.1+ unless noted), 13.2 Missing items (add), 13.3 Cross-document consistency findings, 13. Scope cut list and missing items

### Community 313 - "16. Reconciliation"
Cohesion: 0.67
Nodes (3): 16.1 Daily PA reconciliation job (`payments.recon_daily`, for D-1; catch-up semantics, M11), 16.2 Internal integrity checks (nightly), 16. Reconciliation

## Knowledge Gaps
- **22 isolated node(s):** `github.com/storetmobiles2-code/rovo/backend`, `github.com/storetmobiles2-code/rovo/deploy/docker/healthcheck`, `ROVO_TEST_DATABASE_URL`, `PG_CONTAINER`, `@rovo/config/tsconfig/react.json` (+17 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 2324 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **65 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `rider` connect `rider` to `00 — Planning Baseline (Lead Architect Brief)`, `rovo`, `22 — Deployment Architecture`, `Phase 2 — Implementation Charter (binding for all implementation agents)`, `10 — Database Schema (logical + physical)`, `7. Functional requirements — Delivery partner (RDR-*)`, `Middleware`, `1. Conventions`, `12 — Authentication & Authorisation (Auth / RBAC)`, `8. Functional requirements — Admin (ADM-*)`, `06 — Delivery Partner (Rider) Workflow`, `10. End-to-end tests`, `32 — Final Plan Summary (Phase 1 close-out)`, `data.go`, `08 — System Architecture`, `4. Tokens, sessions, cookies, CSRF, CORS, SSE`, `07 — Admin Console Workflow`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **What connects `github.com/storetmobiles2-code/rovo/backend`, `github.com/storetmobiles2-code/rovo/deploy/docker/healthcheck`, `ROVO_TEST_DATABASE_URL` to the rest of the system?**
  _22 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `schema.d.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.02 - nodes in this community are weakly interconnected._
- **Why does `17 — Frontend Architecture` connect `rider` to `00 — Planning Baseline (Lead Architect Brief)`, `1. Framework evaluation (challenge of P7)`, `3. Routing maps`, `react.tsx`, `uuidv7`, `13. Accessibility, analytics, error tracking, feature flags, testing hooks`?**
  _High betweenness centrality (0.055) - this node is a cross-community bridge._
- **Should `ui/src/index.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.054254711593375214 - nodes in this community are weakly interconnected._
- **Why does `2. Backlog by epic` connect `2. Backlog by epic` to `time.Duration`, `rider`, `planning/README.md`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.13213213213213212 - nodes in this community are weakly interconnected._