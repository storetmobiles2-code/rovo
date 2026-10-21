# Graph Report - rovo  (2026-10-21)

## Corpus Check
- 324 files · ~547,487 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 25 file(s) not represented in the graph (top: (none) 7, .css 6, .caddy 5)

## Summary
- 4068 nodes · 6766 edges · 256 communities (234 shown, 22 thin omitted)
- Extraction: 98% EXTRACTED · 2% INFERRED · 0% AMBIGUOUS · INFERRED: 117 edges (avg confidence: 0.89)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `39af6436`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- schema.d.ts
- ui/src/index.ts
- restaurant/src/router.tsx
- money.ts
- customer/src/router.tsx
- 10 — Database Schema (logical + physical)
- api-client/package.json
- react.tsx
- auth/package.json
- 17 — Frontend Architecture
- deps.go
- i18n/src/index.ts
- 12 — Authentication & Authorisation (Auth / RBAC)
- 14 — Payment Architecture
- testing.T
- 16 — Delivery Zone & Geography Architecture
- 06 — Delivery Partner (Rider) Workflow
- web/package.json
- ref_react
- password.go
- 18 — Mobile & PWA Strategy
- i18n/package.json
- customer/package.json
- restaurant/package.json
- rider/package.json
- migrate.go
- time.Time
- 13 — Order & Delivery State Machines
- apperr.go
- config.go
- 07 — Admin Console Workflow
- 05 — Restaurant Partner Workflow
- context.Context
- ui/package.json
- Bus
- 22 — Deployment Architecture
- 09 — Architecture Decision Records
- 2. Backlog by epic
- admin/package.json
- shell/package.json
- auth.go
- net/http.ResponseWriter
- middleware.go
- github.com/google/uuid.UUID
- totp.go
- money.go
- obs.go
- 24 — Observability Strategy
- 2. Milestones and exit criteria
- 31 — Review Report (Red Team)
- gen-brand-assets.py
- testing/package.json
- compilerOptions
- 29 — Production Readiness Checklist (Go-Live)
- ref_vitest
- 02 — V1 Scope
- 15 — Notification Architecture
- 21 — CI/CD Strategy
- 23 — Backup & Disaster Recovery
- (a) Risk register
- client.ts
- Audience
- compute/main.tf
- devDependencies
- devDependencies
- devDependencies
- session.tsx
- keys.go
- ref_lucide_react
- utils/package.json
- 1. Conventions
- module.compute
- 11. Business rules (BR-*)
- auth/src/index.ts
- 03 — User Personas
- 3.2 Module catalogue
- lib.sh
- 01 — Product Requirements Document (PRD)
- 5. Functional requirements — Customer (CUS-*)
- 8. Functional requirements — Admin (ADM-*)
- config/package.json
- gen-api.mjs
- index.tsx
- 8. Privacy and regulatory (DPDP, IT Act, CERT-In, RBI/PCI, Aadhaar) — `[LEGAL]` throughout
- 9. Security requirements checklist (testable)
- Toast.tsx
- restaurant/src/router.tsx
- dependencies
- dependencies
- devDependencies
- 26 — Repository Structure
- network/main.tf
- api-client/src/index.ts
- vite.ts
- devDependencies
- module.compute
- 00 — Planning Baseline (Lead Architect Brief)
- 04 — Customer Journey (End-to-End UX)
- tokens.ts
- 6. Focused mitigation designs
- load
- customer/src/sw.ts
- dependencies
- dependencies
- dependencies
- dependencies
- dependencies
- Phase 2 — Implementation Progress
- 11 — API Specification (REST, OpenAPI 3.1)
- 10. End-to-end tests
- 20 — Testing Strategy
- Part A — Free tiers for local / CI / dev-preview / demo
- 32 — Final Plan Summary (Phase 1 close-out)
- devDependencies
- 00003_catalog.sql
- db/db.go
- aws_cloudfront_distribution.this
- 6. Functional requirements — Restaurant partner (RES-*)
- 08 — System Architecture
- cities
- 5. Threat table
- Part B — Production cloud comparison (staging + production)
- users
- Dialog.tsx
- 10. Non-functional requirements (NFR-*)
- 12. Regulatory requirements (LEG-*) [LEGAL]
- 19 — Security Threat Model (STRIDE), Security Requirements & Incident Response
- 7. Cloud infrastructure, database, cryptography, logging and supply chain
- 6. Go integration tests
- 14. Service-by-service comparison (India regions)
- 27 — Implementation Backlog (Phase 2)
- exports
- Phase 2 — Implementation Charter (binding for all implementation agents)
- rovo — System Knowledge Graph
- 7. Functional requirements — Delivery partner (RDR-*)
- 1. UX principles (apply to all rovo apps: customer, restaurant, rider, admin — R14)
- 15. Cost estimates (on-demand, excl. GST; ₹95/USD) — single source of cost (R46)
- PROMPT — rovo Phase 2: implement V1
- smoke.sh
- react.json
- dependencies
- devDependencies
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
- 2. Phone OTP authentication (members)
- 10. Incident response outline
- 11. Mobile / PWA tests
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
- database/main.tf
- rovo infrastructure (OpenTofu)
- 3. First open
- 4. Home / discovery (C-05)
- 7. Consistency, idempotency and concurrency
- 00006_payments.sql
- 21. CI gating, coverage, flaky tests, runtime budgets
- 8. Authorisation tests
- iam/main.tf
- 4. Metrics
- 7. CI/CD, DNS/CDN/WAF, monitoring, email
- dev-keys.sh
- .prettierrc.json
- 6. Login (deferred) and profile
- 7. Restaurant page, customisation, cart
- 9. Order tracking (C-16)
- 2. Containers (C4 level 2)
- 4. Module boundary rules
- 6. Real-time design
- storage/main.tf
- 22. Definition of Done, bug severity, triage SLAs, release gates
- 3. PostgreSQL (PostGIS mandatory)
- 9. Dev / preview / demo recommendation
- tsconfig.node.json
- backup.sh
- bootstrap.sh
- dev.sh
- EventStream
- api-client/tsconfig.json
- devDependencies
- Telugu translations — native review queue
- i18n/tsconfig.json
- 5. Search (C-06)
- 3. Components: the Go modular monolith (C4 level 3)
- customer/src/router.tsx
- 00009_ratings_support_notifications.sql
- 00010_admin_platform.sql
- 2. Quality goals and risk-based prioritisation
- observability/main.tf
- test-integration.sh
- tf-validate.sh
- scripts
- peerDependencies
- scripts
- exports
- peerDependencies
- scripts
- peerDependencies
- scripts
- minio-init.sh
- README.md
- knowledge-graph.sh
- restore.sh
- exports
- github.com/storetmobiles2-code/rovo/backend
- github.com/storetmobiles2-code/rovo/deploy/docker/healthcheck
- 00004_pricing_promotions.sql
- idgen.go
- 4. Tokens, sessions, cookies, CSRF, CORS, SSE
- respond.go
- orders
- uuidv7
- phone.ts
- 5. Authorisation model
- 18. Test data management
- (a) Risk register
- 3. Admin authentication
- 3. Logs
- 1. Identity model
- 4. Network
- 2. Instrumentation
- prod/.terraform.lock.hcl
- staging/.terraform.lock.hcl
- terraform/versions.tf

## God Nodes (most connected - your core abstractions)
1. `users` - 49 edges
2. `cn()` - 44 edges
3. `cities` - 34 edges
4. `09 — Architecture Decision Records` - 31 edges
5. `2. Backlog by epic` - 31 edges
6. `20 — Testing Strategy` - 29 edges
7. `14 — Payment Architecture` - 24 edges
8. `aws_cloudfront_distribution.this` - 23 edges
9. `10 — Database Schema (logical + physical)` - 22 edges
10. `compilerOptions` - 21 edges

## Surprising Connections (you probably didn't know these)
- `10.2 Time control in E2E` --references--> `Clock`  [INFERRED]
  docs/planning/20-testing-strategy.md → backend/internal/platform/clock/clock.go
- `18.6 Deterministic clocks and IDs` --references--> `Clock`  [INFERRED]
  docs/planning/20-testing-strategy.md → backend/internal/platform/clock/clock.go
- `1. Decisions summary` --references--> `Clock`  [INFERRED]
  docs/planning/20-testing-strategy.md → backend/internal/platform/clock/clock.go
- `6.5 River jobs and timers (fake clock)` --references--> `Clock`  [INFERRED]
  docs/planning/20-testing-strategy.md → backend/internal/platform/clock/clock.go
- `15.2 Configuration` --references--> `env()`  [INFERRED]
  docs/planning/17-frontend-architecture.md → backend/internal/platform/config/config_test.go

## Import Cycles
- None detected.

## Communities (256 total, 22 thin omitted)

### Community 0 - "schema.d.ts"
Cohesion: 0.02
Nodes (99): components, $defs, HeaderETag, HeaderIdempotentReplayed, HeaderLocation, HeaderRateLimit, HeaderRetryAfter, operations (+91 more)

### Community 1 - "ui/src/index.ts"
Cohesion: 0.07
Nodes (58): cn(), Button, ButtonProps, ButtonSize, ButtonVariant, IconButton, IconButtonProps, sizes (+50 more)

### Community 2 - "restaurant/src/router.tsx"
Cohesion: 0.09
Nodes (36): Protected(), LoginRedirect(), ScreenSkeleton(), guarded(), NotFound(), Loading(), Protected(), LoginRedirect() (+28 more)

### Community 3 - "money.ts"
Cohesion: 0.12
Nodes (33): useFormatters(), AddressLike, clean(), formatAddress(), cache, DateInput, dtf(), formatDate() (+25 more)

### Community 4 - "customer/src/router.tsx"
Cohesion: 0.11
Nodes (11): App(), App(), updateSW, router, App(), updateSW, router, App() (+3 more)

### Community 5 - "10 — Database Schema (logical + physical)"
Cohesion: 0.04
Nodes (47): 10 — Database Schema (logical + physical), 10. DDL — ratings, support, notifications, 11. DDL — admin, platform, 12. PII classification (DPDP Act 2023 & encryption decisions), 13.1 Erasure map (DPDP, M15, RV-031) [LEGAL], 13. Data retention policy (per table), 14. Retention jobs and partitioning triggers (C8), 15.1 Reference seeds (all environments) (+39 more)

### Community 6 - "api-client/package.json"
Cohesion: 0.04
Nodes (48): openapi-fetch, openapi-typescript, dependencies, openapi-fetch, @rovo/utils, description, devDependencies, jsdom (+40 more)

### Community 7 - "react.tsx"
Cohesion: 0.08
Nodes (30): 5.1 Endpoint and events (aligned with R10, doc 08 §Real-time and doc 12 §4.7), 5.2 Client (`createEventStream` in `packages/api-client`), 5. Real-time integration (SSE), ListFilters, RestaurantFilters, ApiClientContext, healthListeners, realtimeHealth (+22 more)

### Community 8 - "auth/package.json"
Cohesion: 0.04
Nodes (44): dependencies, @rovo/api-client, @rovo/utils, @tanstack/react-query, description, devDependencies, jsdom, react (+36 more)

### Community 9 - "17 — Frontend Architecture"
Cohesion: 0.05
Nodes (43): 0. Summary of decisions, 10. Internationalisation (`packages/i18n`), 11. Images, 12. Performance budgets, 13.1 Accessibility (WCAG 2.2 AA), 13.2 Analytics (privacy-friendly, DPDP), 13.3 Error tracking, 13.4 Feature flags and remote config (+35 more)

### Community 10 - "deps.go"
Cohesion: 0.11
Nodes (14): Module, Register, RegisterJobs, Claims, Scopes, TestDecodeJSON(), TestErrorRenders9457(), TestErrorRetryAfter() (+6 more)

### Community 11 - "i18n/src/index.ts"
Cohesion: 0.09
Nodes (29): ADR-026: Infrastructure as Code with OpenTofu (Terraform-compatible), i18next, Catalog, lazyCatalogBackend, loadCatalog(), loaders, Language, LANGUAGE_STORAGE_KEY (+21 more)

### Community 12 - "12 — Authentication & Authorisation (Auth / RBAC)"
Cohesion: 0.18
Nodes (10): 0. Decision summary, 10. Sources (accessed 2026-10-04), 11. Challenges to baseline (summary; full list in report), 12 — Authentication & Authorisation (Auth / RBAC), 6.1 What is collected (proposed; Product/Legal to finalise), 6.2 Storage and access, 6. Rider & restaurant KYC data access, 7. API surface (auth module; detail in 11) (+2 more)

### Community 13 - "14 — Payment Architecture"
Cohesion: 0.05
Nodes (39): 0. Summary, 10.1 Structure, 10.2 Chart of accounts (V1), 10.3 Worked example (prepaid order) — redone 2026-10-04 per R8 (RV-046), 10.4 Other events, 10. Ledger design (double-entry-lite), 11.1 Sequence, 11.2 Rules (+31 more)

### Community 14 - "testing.T"
Cohesion: 0.13
Nodes (24): NewPool(), New(), NewBus(), TestBusTransactionalNotify(), NewHub(), TestCapFrame(), TestHubDropsSlowConsumer(), TestHubRoutesByTopic() (+16 more)

### Community 15 - "16 — Delivery Zone & Geography Architecture"
Cohesion: 0.05
Nodes (36): 0. Decision summary, 10.1 Mahabubnagar sample zone (fixture, NOT the real service area), 10.2 Fixture points and expected results, 10.3 Scenario tests, 10. Test fixtures, 11. Mahabubnagar specifics: verified vs. assumed, 12. Not doing in V1, 13. Open items (+28 more)

### Community 16 - "06 — Delivery Partner (Rider) Workflow"
Cohesion: 0.06
Nodes (35): 06 — Delivery Partner (Rider) Workflow, 10.1 Earnings (D-20, D-21), 10.2 Cash-in-hand and deposit (D-22, D-23), 10.3 Delivery history (D-24), 10. Earnings, cash-in-hand, deposits, history, 11. Safety and SOS (D-30), 12. Delivery status ↔ order status map, 13. Edge cases (+27 more)

### Community 17 - "web/package.json"
Cohesion: 0.06
Nodes (35): eslint, prettier, description, devDependencies, eslint, @eslint/js, prettier, @rovo/config (+27 more)

### Community 18 - "ref_react"
Cohesion: 0.16
Nodes (13): LoginScreen(), AdminLoginScreen(), submitCredentials(), submitTotp(), AdminLoginScreenProps, BrandLogo(), BrandLogoProps, Props (+5 more)

### Community 19 - "password.go"
Cohesion: 0.11
Nodes (8): Argon2Params, TestFieldCrypto(), TestPasswordHash(), FieldKeyring, NewFieldKeyring(), PasswordHasher, NewPasswordHasher(), parsePHC()

### Community 20 - "18 — Mobile & PWA Strategy"
Cohesion: 0.06
Nodes (34): 0. Key constraints that shape product scope, 10.1 Step 1b: Trusted Web Activity (Bubblewrap) for restaurant and rider apps, 10.2 Criteria for going native (V2 triggers), 10.3 React Native (Expo) vs Flutter, 10.4 How V1 prepares for native, 10. Native path, 11. Device and browser support matrix, 12. Low-end device test plan (+26 more)

### Community 21 - "i18n/package.json"
Cohesion: 0.06
Nodes (34): react-i18next, dependencies, i18next, react-i18next, @rovo/utils, description, devDependencies, jsdom (+26 more)

### Community 22 - "customer/package.json"
Cohesion: 0.06
Nodes (34): description, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/config, @rovo/i18n (+26 more)

### Community 23 - "restaurant/package.json"
Cohesion: 0.06
Nodes (34): description, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/config, @rovo/i18n (+26 more)

### Community 24 - "rider/package.json"
Cohesion: 0.06
Nodes (34): description, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/config, @rovo/i18n (+26 more)

### Community 25 - "migrate.go"
Cohesion: 0.12
Nodes (5): New(), Up(), TestMigrationsUpDown(), Runner, Status

### Community 26 - "time.Time"
Cohesion: 0.10
Nodes (15): TestPreviousISOWeek(), AccountingDate(), PreviousISOWeek(), StartOfDay(), StartOfISOWeek(), NewSequence(), TestSequenceDeterministic(), PresignedPut (+7 more)

### Community 27 - "13 — Order & Delivery State Machines"
Cohesion: 0.06
Nodes (32): 0. Decision summary, 10. Open items, 11. Challenges / deviations recorded, 13 — Order & Delivery State Machines, 1.1 Order transition table, 1.2 Admin "on behalf" commands, 1. Order state machine, 2.1 Delivery transition table (+24 more)

### Community 28 - "apperr.go"
Cohesion: 0.19
Nodes (19): As(), BadRequest(), Conflict(), Forbidden(), Error, FieldError, Internal(), IsCode() (+11 more)

### Community 29 - "config.go"
Cohesion: 0.08
Nodes (19): TestRefreshToken(), HashToken(), LooksLikeRefreshToken(), NewRefreshToken(), decodeB64(), Config, Load(), parseBool() (+11 more)

### Community 30 - "07 — Admin Console Workflow"
Cohesion: 0.06
Nodes (31): 07 — Admin Console Workflow, 10. Coupons (A-50, A-51), 11.1 Lifecycle, 11.2 Priority and SLAs (defaults, configurable), 11.3 Ticket workspace (A-71), 11. Disputes and tickets (A-70, A-71), 12.1 Weekly payout run, 12. Commissions and settlements (A-80 … A-83) (+23 more)

### Community 31 - "05 — Restaurant Partner Workflow"
Cohesion: 0.06
Nodes (30): 05 — Restaurant Partner Workflow, 10. Screen inventory (`restaurant` app), 11. Requirements for Backend / Frontend and challenges to the baseline, 1.1 Indicative Telugu labels (key actions) `[ASSUMPTION — native-speaker review required]`, 1. Design constraints specific to restaurants, 2. Roles and permissions (restaurant side), 3.1 Flow overview, 3.2 Lead form (P-01, public, no account needed) (+22 more)

### Community 32 - "context.Context"
Cohesion: 0.06
Nodes (22): Store, Clock, Offset, NewOffset(), TestFakeAndOffset(), Querier, IsNoRows(), MarkProcessed() (+14 more)

### Community 33 - "ui/package.json"
Cohesion: 0.06
Nodes (31): clsx, @radix-ui/react-checkbox, @radix-ui/react-radio-group, @radix-ui/react-select, @radix-ui/react-slot, @radix-ui/react-switch, @radix-ui/react-visually-hidden, description (+23 more)

### Community 34 - "Bus"
Cohesion: 0.13
Nodes (6): Bus, Start(), Frame, Hub, Subscription, Writer

### Community 35 - "22 — Deployment Architecture"
Cohesion: 0.09
Nodes (22): 10. Local development (Docker Compose), 11. Dev / demo environment (local only, R24), 12. Infrastructure as Code layout (OpenTofu; Terraform-compatible), 13. Resource allocation summary, 14. Scaling path, 15. Cost (single source: `25` §15, R46), 16. Security baseline checklist (deployment-level; threat model in `19`), 1. Environments (+14 more)

### Community 36 - "09 — Architecture Decision Records"
Cohesion: 0.08
Nodes (26): 09 — Architecture Decision Records, ADR-001: Modular monolith instead of microservices, ADR-002: Go for the backend; HTTP routing with net/http ServeMux, ADR-003: REST/JSON with OpenAPI spec-first and generated code, ADR-004: No gRPC in V1, ADR-006: Asynchronous processing with River as the transactional outbox, ADR-008: Real-time via Server-Sent Events, ADR-009: Frontend: Vite React SPA/PWA (not Next.js) (+18 more)

### Community 37 - "2. Backlog by epic"
Cohesion: 0.07
Nodes (27): 2. Backlog by epic, E02 — Identity, auth & RBAC, E03 — Geo: cities, zones, localities, serviceability, E04 — Partner onboarding & KYC, E06 — Pricing, quote engine, tax & coupons, E07 — Ordering & order state machine, E08 — Payments, E09 — Ledger, settlement, payouts, COD cash (+19 more)

### Community 38 - "admin/package.json"
Cohesion: 0.07
Nodes (26): description, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/config, @rovo/i18n (+18 more)

### Community 39 - "shell/package.json"
Cohesion: 0.08
Nodes (25): description, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/config, @rovo/i18n (+17 more)

### Community 40 - "auth.go"
Cohesion: 0.11
Nodes (15): principalKey, Role, TokenInput, Audience, Principal, PrincipalFrom(), SystemPrincipal(), WithPrincipal() (+7 more)

### Community 41 - "net/http.ResponseWriter"
Cohesion: 0.17
Nodes (6): writeJSON(), BearerToken(), JSON(), NoContent(), Checker, statusRecorder

### Community 42 - "middleware.go"
Cohesion: 0.08
Nodes (29): ParseAppHosts(), AudienceFrom(), Authenticate(), mapTokenError(), TestRequireClientHeader(), TestResolveAudienceAndHelpers(), AccessLog(), BodyLimit() (+21 more)

### Community 43 - "github.com/google/uuid.UUID"
Cohesion: 0.12
Nodes (15): Deps, Publisher, NewPublisher(), NewRegistry(), AddWorker(), Enqueuer, New(), NewRegistry() (+7 more)

### Community 44 - "totp.go"
Cohesion: 0.24
Nodes (8): TOTPEnrolment, TestMAC(), TestTOTP(), EqualMAC(), MAC(), NewTOTPSecret(), TOTPCode(), ValidateTOTP()

### Community 45 - "money.go"
Cohesion: 0.17
Nodes (16): Format(), groupIndian(), InclusiveTax(), INR(), MulDiv(), RoundToRupee(), Rupees(), SplitHalves() (+8 more)

### Community 46 - "obs.go"
Cohesion: 0.09
Nodes (9): NewMetrics(), redactAttr(), SetupTracing(), statusClass(), TraceID(), WithRequestID(), ctxKey, Metrics (+1 more)

### Community 47 - "24 — Observability Strategy"
Cohesion: 0.15
Nodes (12): 10. Alert routing & on-call (tiny team), 11. Data-volume budget (production; volumes per R45 / doc 20, requests per `25` §15.7), 12. What we are not doing in V1, 1.1 Cost comparison for production (pilot volumes from `25` §10), 1.2 CERT-In log archive (M1, R36) — compliance copy, separate from Grafana, 1. Decision: telemetry backends per environment, 24 — Observability Strategy, 5. Traces & sampling (+4 more)

### Community 48 - "2. Milestones and exit criteria"
Cohesion: 0.09
Nodes (22): 1. Planning assumptions (state them, then challenge them at M0 exit), 28 — Milestones & Dependencies (Phase 2), 2. Milestones and exit criteria, 3.1 Summary, 3.2 Gantt (indicative), 3.3 Team-size variants, 3. Calendar and capacity, 4.1 The path (+14 more)

### Community 49 - "31 — Review Report (Red Team)"
Cohesion: 0.09
Nodes (22): 0. Executive summary: top 10 issues, 10. Vendor lock-in and the portability ADR, 11. V1 scope creep and essential gaps, 12. Disagreements requiring Lead decision, 13.1 Scope cut list (defer to V1.1+ unless noted), 13.2 Missing items (add), 13.3 Cross-document consistency findings, 13. Scope cut list and missing items (+14 more)

### Community 50 - "gen-brand-assets.py"
Cohesion: 0.17
Nodes (11): best_font(), contain(), load_mark(), main(), make_icon(), make_logo(), rounded_bg(), save_png() (+3 more)

### Community 51 - "testing/package.json"
Cohesion: 0.09
Nodes (21): description, jsdom, react, react-dom, @rovo/api-client, @rovo/config, @rovo/i18n, @tanstack/react-query (+13 more)

### Community 52 - "compilerOptions"
Cohesion: 0.09
Nodes (22): compilerOptions, allowImportingTsExtensions, esModuleInterop, exactOptionalPropertyTypes, forceConsistentCasingInFileNames, isolatedModules, lib, module (+14 more)

### Community 53 - "29 — Production Readiness Checklist (Go-Live)"
Cohesion: 0.10
Nodes (21): 0. How to use, 10. Resilience tests, 11. Mobile / PWA device matrix, 12. Accessibility & i18n, 13. Support processes, 14. Rollback plan, 15. Go/No-Go decision procedure, 16.1 Launch day (public launch, target Monday 2027-11-08) (+13 more)

### Community 54 - "ref_vitest"
Cohesion: 0.12
Nodes (14): msw, ApiClientProvider(), server, wrapper(), SessionProvider(), API, at(), authHandlers() (+6 more)

### Community 55 - "02 — V1 Scope"
Cohesion: 0.10
Nodes (20): 02 — V1 Scope, 1.1 Capability map (P0 = launch blocker), 1.2 P1 items committed to V1 (built before launch), 1.3 Golden-flow coverage check, 1. In scope for V1, 2.1 Mandatory deferrals (baseline §1), 2.2 Additional scope-creep decisions, 2.3 Accepted scope cuts from the review (31 §13.1, baseline §9) — all deferred to V1.1+ (+12 more)

### Community 56 - "15 — Notification Architecture"
Cohesion: 0.08
Nodes (25): 0. Summary, 10. Provider abstraction (Go sketch), 12. Failure modes, 13. Open questions, 15 — Notification Architecture, 1. Channels, 2.1 Customer, 2.2 Restaurant (owner/staff of the outlet) (+17 more)

### Community 57 - "21 — CI/CD Strategy"
Cohesion: 0.07
Nodes (26): 10. Release process, 11. Dependency updates, 12. Branch protection (repository rulesets), 13. CI time budget, 1. Principles, 21 — CI/CD Strategy, 2. Workflow map, 3.1 Go backend (`backend/**`, `openapi/**`) (+18 more)

### Community 58 - "23 — Backup & Disaster Recovery"
Cohesion: 0.10
Nodes (20): 10. Dev/preview & local, 1. Targets, 23 — Backup & Disaster Recovery, 2. What must be protected, 3.1 Layers, 3.2 Cost, 3. PostgreSQL backup design, 4. Object storage protection (+12 more)

### Community 59 - "(a) Risk register"
Cohesion: 0.14
Nodes (13): 30 — Risks, Assumptions, Decisions & Open Questions (RAID), Architecture Decision Records (doc 09), (b) Assumptions register (significant `[ASSUMPTION]`s from docs 01–26), (c) Decisions log, (d) Unresolved decisions / open questions (deduplicated), (e) Remaining cross-document conflicts (for reconciliation by doc owners), Lead rulings (baseline §8, §9), Legal / tax / regulatory [LEGAL] (+5 more)

### Community 60 - "client.ts"
Cohesion: 0.08
Nodes (26): absolute(), acceptLanguage(), API_BASE_URL, ApiClientOptions, createApiClient(), FetchResultLike, idempotencyHeaders(), RovoClient (+18 more)

### Community 61 - "Audience"
Cohesion: 0.35
Nodes (4): TestCookies(), Cookies, maxAge(), sameSite()

### Community 62 - "compute/main.tf"
Cohesion: 0.07
Nodes (72): aws_appautoscaling_policy.api_cpu, aws_appautoscaling_target.api, aws_ecs_cluster_capacity_providers.this, aws_ecs_cluster.this, aws_ecs_service.api, aws_ecs_service.worker, aws_ecs_task_definition.api, aws_ecs_task_definition.migrate (+64 more)

### Community 63 - "devDependencies"
Cohesion: 0.11
Nodes (19): devDependencies, @rovo/config, @rovo/testing, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event, @types/react, @types/react-dom (+11 more)

### Community 64 - "devDependencies"
Cohesion: 0.11
Nodes (19): devDependencies, @rovo/config, @rovo/testing, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event, @types/react, @types/react-dom (+11 more)

### Community 65 - "devDependencies"
Cohesion: 0.11
Nodes (19): devDependencies, @rovo/config, @rovo/testing, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event, @types/react, @types/react-dom (+11 more)

### Community 66 - "session.tsx"
Cohesion: 0.17
Nodes (14): 5.2 Permission catalogue and role matrix, 11. Delivery receipts and observability, unwrap(), ApiError, isRecord(), qk, useApiClient(), AdminMfaChallenge (+6 more)

### Community 67 - "keys.go"
Cohesion: 0.26
Nodes (10): Keyset, newIssuer(), TestJWTRejectsForgeries(), TestJWTRoundTrip(), TestKeyRotationVerifiesPrevious(), Issuer, NewIssuer(), KID() (+2 more)

### Community 68 - "ref_lucide_react"
Cohesion: 0.22
Nodes (13): RootLayout(), TABS, RootLayout(), TABS, RootLayout(), TABS, BottomNav(), BottomNavItem (+5 more)

### Community 69 - "utils/package.json"
Cohesion: 0.11
Nodes (17): description, devDependencies, @rovo/config, typescript, vitest, exports, @rovo/config, typescript (+9 more)

### Community 70 - "1. Conventions"
Cohesion: 0.12
Nodes (17): title(), ADR-018: Internationalisation (en + te), 1.10 Rate limiting, 1.11 Localization, 1.12 Auth (detail in 12), 1.13 Versioning and deprecation, 1.14 Caching, 1.1 Hosts, audiences, prefixes (+9 more)

### Community 71 - "module.compute"
Cohesion: 0.08
Nodes (49): Terraform module: deploy/terraform/envs/prod, local.app_environment, local.app_hosts, local.name_prefix, local.tags, module.compute, module.database, module.edge (+41 more)

### Community 72 - "11. Business rules (BR-*)"
Cohesion: 0.12
Nodes (17): 11.10 BR-COUP — Coupons and stacking, 11.11 BR-CONT — Contact & privacy rules (V1 without masked calling), 11.12 BR-OTP — Delivery OTP, 11.13 BR-RES — Restaurant operating rules, 11.14 BR-PAYOUT — Payout cycles, 11.15 BR-MENU — Menu & dietary rules, 11.15a BR-COST — Running cost as a business line item, 11.1 BR-FEE — Customer-facing fees and pricing (+9 more)

### Community 73 - "auth/src/index.ts"
Cohesion: 0.23
Nodes (17): Me, RoleName, evaluateGuard(), GuardOutcome, RequireAuth(), RequireAuthProps, roleSetFor(), appAudienceKey() (+9 more)

### Community 74 - "03 — User Personas"
Cohesion: 0.12
Nodes (15): 03 — User Personas, Anti-personas (we do not design V1 for), Cross-persona summary, Device & connectivity matrix (feeds doc 18 device lab and doc 20 test matrix), P1 — Sravani, the budget-conscious student, P2 — Ravi & Lakshmi, the working family ordering dinner, P3 — Narasimha Reddy, elderly, Telugu-first, less tech-savvy, P4 — Venkatesh, small family restaurant owner, shared phone (+7 more)

### Community 75 - "3.2 Module catalogue"
Cohesion: 0.12
Nodes (16): 3.2 Module catalogue, admin & audit, catalog (restaurants, menus), dispatch (delivery), geo (cities, zones, localities, serviceability), identity (auth), ledger & settlement, notifications (+8 more)

### Community 76 - "lib.sh"
Cohesion: 0.18
Nodes (9): compose_pg_container(), die(), fail(), need(), _pg_run(), pg_tool(), pg_tool_nodb(), psql_q() (+1 more)

### Community 77 - "01 — Product Requirements Document (PRD)"
Cohesion: 0.13
Nodes (14): 01 — Product Requirements Document (PRD), 13. Open questions, 14. Sources (accessed 2026-10-04), 1. Vision, 2.1 Market context, 2.2 Problems we solve, 2. Problem statement — Mahabubnagar, 3.1 Goals (V1) (+6 more)

### Community 78 - "5. Functional requirements — Customer (CUS-*)"
Cohesion: 0.13
Nodes (15): 5.10 CUS-TRK — Order tracking and status, 5.11 CUS-HIST — Order history and reorder, 5.12 CUS-RATE — Ratings and reviews, 5.13 CUS-COUP — Coupons and offers, 5.14 CUS-SUPP — Support and help for an order, 5.1 CUS-AUTH — Registration, login, profile, account, 5.2 CUS-ADDR — Location and addresses, 5.3 CUS-DISC — Restaurant discovery (+7 more)

### Community 79 - "8. Functional requirements — Admin (ADM-*)"
Cohesion: 0.13
Nodes (15): 8.10 ADM-RPT — Reports and CSV export [view per role], 8.11 ADM-AUDIT — Audit logs [view: S; F for money events], 8.12 ADM-ZONE — Zones and fees configuration [act: S/O; F for fee changes], 8.13 ADM-PAYO — Payouts and settlement [act: F/S], 8.14 ADM-CFG / ADM-CONT — Platform configuration and content, 8.1 ADM-AUTH — Admin access, 8.2 ADM-DASH — Dashboard, 8.3 ADM-USER — Users (customers) [view: all; act: S/O/Sp] (+7 more)

### Community 80 - "config/package.json"
Cohesion: 0.10
Nodes (20): eslint-plugin-jsx-a11y, eslint-plugin-react-hooks, eslint-plugin-react-refresh, @fontsource/noto-sans-telugu, globals, tailwindcss, typescript-eslint, workbox-build (+12 more)

### Community 81 - "gen-api.mjs"
Cohesion: 0.11
Nodes (11): bin, out, pkg, placeholder, real, flatten(), root, TARGETS (+3 more)

### Community 82 - "index.tsx"
Cohesion: 0.14
Nodes (12): axe-core, ApiClient, AppId, Namespace, AppProvidersProps, axeViolations(), createTestProviders(), Providers() (+4 more)

### Community 83 - "8. Privacy and regulatory (DPDP, IT Act, CERT-In, RBI/PCI, Aadhaar) — `[LEGAL]` throughout"
Cohesion: 0.14
Nodes (14): 8.10 CERT-In Directions (28 Apr 2022), 8.11 RBI Payment Aggregator directions and card data, 8.12 PCI-DSS scope minimisation, 8.13 Aadhaar, 8.1 DPDP Act 2023 and DPDP Rules 2025: status as of 2026-10-04, 8.2 DPDP obligations → rovo design, 8.3 Notice and consent UX (for 04/05/06), 8.4 Data-principal rights workflows (+6 more)

### Community 84 - "9. Security requirements checklist (testable)"
Cohesion: 0.14
Nodes (14): 9.10 Privacy and compliance, 9.11 Incident readiness, 9.12 Cloud platform (added for 00 §4a), 9.13 Added in v1.1, 9.1 Authentication (OTP and admin), 9.2 Sessions, tokens, CSRF, CORS, SSE, 9.3 Authorisation, 9.4 Web and content security (+6 more)

### Community 85 - "Toast.tsx"
Cohesion: 0.21
Nodes (13): @radix-ui/react-toast, dismissToast(), emit(), items, listeners, toast(), Toaster(), ToastInput (+5 more)

### Community 86 - "restaurant/src/router.tsx"
Cohesion: 0.10
Nodes (21): LoginScreen(), LoginScreen(), analytics, full(), hours, inbox, layoutRoute, loginRoute (+13 more)

### Community 87 - "dependencies"
Cohesion: 0.14
Nodes (14): dependencies, @eslint/js, eslint-plugin-jsx-a11y, eslint-plugin-react-hooks, eslint-plugin-react-refresh, @fontsource/noto-sans-telugu, globals, tailwindcss (+6 more)

### Community 88 - "dependencies"
Cohesion: 0.14
Nodes (14): dependencies, clsx, lucide-react, @radix-ui/react-checkbox, @radix-ui/react-dialog, @radix-ui/react-radio-group, @radix-ui/react-select, @radix-ui/react-slot (+6 more)

### Community 89 - "devDependencies"
Cohesion: 0.14
Nodes (14): devDependencies, jsdom, react, react-dom, @rovo/config, @rovo/testing, @testing-library/jest-dom, @testing-library/react (+6 more)

### Community 90 - "26 — Repository Structure"
Cohesion: 0.10
Nodes (18): 10. Community and governance files to create in Phase 2, 11. CI path filters (summary for doc 21), 12. Open items, 1.1 `backend/` (Go module), 1.2 `openapi/`, 1.3 `web/` (pnpm workspace; structure from doc 17 §2), 1.4 `deploy/`, 1.5 `tools/` and `scripts/` (+10 more)

### Community 91 - "network/main.tf"
Cohesion: 0.14
Nodes (41): aws_eip.nat, aws_flow_log.all, aws_internet_gateway.this, aws_nat_gateway.this, aws_route.app_nat, aws_route.public_igw, aws_route_table.app, aws_route_table_association.app (+33 more)

### Community 92 - "api-client/src/index.ts"
Cohesion: 0.22
Nodes (6): Json, OtpChallenge, OtpVerifyResult, RequestBody, Responses, SuccessBody

### Community 93 - "vite.ts"
Cohesion: 0.21
Nodes (6): @tailwindcss/vite, @vitejs/plugin-react, defineRovoApp(), RovoAppName, RovoAppOptions, RovoPwaOptions

### Community 94 - "devDependencies"
Cohesion: 0.15
Nodes (13): devDependencies, react, react-dom, @rovo/config, @rovo/testing, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event (+5 more)

### Community 95 - "module.compute"
Cohesion: 0.12
Nodes (37): Terraform module: deploy/terraform/envs/staging, local.app_environment, local.app_hosts, local.name_prefix, local.tags, module.compute, module.database, module.edge (+29 more)

### Community 96 - "00 — Planning Baseline (Lead Architect Brief)"
Cohesion: 0.15
Nodes (12): 00 — Planning Baseline (Lead Architect Brief), 1. Product framing, 2. Canonical vocabulary (use these exact names), 3. Cross-cutting conventions, 4. Provisional architecture decisions (agents may challenge), 4a. Environment & hosting strategy (USER DIRECTIVE, 2026-10-04 — overrides P17), 5. Indicative commercial defaults (all configurable per city/zone/restaurant; validate with local market), 6. Deliverable map and owners (+4 more)

### Community 97 - "04 — Customer Journey (End-to-End UX)"
Cohesion: 0.17
Nodes (11): 04 — Customer Journey (End-to-End UX), 10. Cancellation rules (UX), 11. Delivered → rating (C-17), 12. Order history and reorder (C-18, C-19), 14. Notification touchpoints, 15. Edge cases and resilience, 16. Screen inventory (customer app), 17. Requirements for Backend / Frontend / other docs (+3 more)

### Community 98 - "tokens.ts"
Cohesion: 0.24
Nodes (10): 2.6 Provider fallback and alternative channels, 4.1 RED (HTTP, per route template), appThemeColor, channel(), contrastRatio(), contrastRules, palette, PaletteToken (+2 more)

### Community 99 - "6. Focused mitigation designs"
Cohesion: 0.17
Nodes (12): 6.10 Upload pipeline (T-54..T-58) — decision (v1.1, R38), 6.11 DoS, scraping and edge configuration (T-59..T-63), 6.1 OTP abuse (T-01..T-07), 6.2 XSS, CSP and security headers (T-24..T-30), 6.3 Server-authoritative pricing and quotes (T-31, T-32), 6.4 Coupon abuse (T-33, T-34), 6.5 Payment webhooks (T-35..T-38), 6.6 Refund fraud (T-39, T-40) (+4 more)

### Community 100 - "load"
Cohesion: 0.18
Nodes (11): 12.1 Load model — canonical (R45; this doc owns it per R48), 12.2 SLOs asserted as k6 thresholds, 12.3 k6 scenarios, 12.4 Capacity test on the managed production shape, 12. Performance and load tests, 6.2 sqlc queries and repositories, E00 — Repository bootstrap & tooling, E05 — Catalog & menu (+3 more)

### Community 101 - "customer/src/sw.ts"
Cohesion: 0.32
Nodes (3): NEVER_CACHE, NEVER_CACHE, NEVER_CACHE

### Community 102 - "dependencies"
Cohesion: 0.17
Nodes (12): dependencies, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/i18n, @rovo/shell (+4 more)

### Community 103 - "dependencies"
Cohesion: 0.17
Nodes (12): dependencies, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/i18n, @rovo/shell (+4 more)

### Community 104 - "dependencies"
Cohesion: 0.17
Nodes (12): dependencies, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/i18n, @rovo/shell (+4 more)

### Community 105 - "dependencies"
Cohesion: 0.17
Nodes (12): dependencies, lucide-react, react, react-dom, @rovo/api-client, @rovo/auth, @rovo/i18n, @rovo/shell (+4 more)

### Community 106 - "dependencies"
Cohesion: 0.17
Nodes (12): dependencies, axe-core, jsdom, msw, @rovo/api-client, @rovo/i18n, @tanstack/react-query, @testing-library/dom (+4 more)

### Community 107 - "Phase 2 — Implementation Progress"
Cohesion: 0.18
Nodes (10): API contract (`openapi/`), Backend (`backend/`), DevOps (`deploy/`, `.github/`), Frontend (`web/`), Known risks / notes, Phase 2 — Implementation Progress, Remaining before the foundation is "done", Snapshot (+2 more)

### Community 108 - "11 — API Specification (REST, OpenAPI 3.1)"
Cohesion: 0.10
Nodes (20): 0. Decision summary, 11 — API Specification (REST, OpenAPI 3.1), 2.1 Public catalog and config (`/public`, `/config`), 2.2 Auth, session, me (`/auth`, `/me`), aligned with 12 §7, 2.3 Customer, 2.4 Restaurant partner (`/partner/…`, partner host), 2.5 Rider (`/rider/…`, `rider.` host), 2.6 Admin (`/admin/…`, admin host; every list is filtered by the caller's city scope) (+12 more)

### Community 109 - "10. End-to-end tests"
Cohesion: 0.18
Nodes (10): 20. Testing hooks (for doc 20), 10.1 Stack and conventions, 10.2 Time control in E2E, 10.3 Golden flow (multi-actor, 4 browser contexts), 10.4 Failure scenarios, 10. End-to-end tests, 19.1 Environments (baseline §4a), 19.2 Provider fakes (wire-level, shipped as small Go services in the repo) (+2 more)

### Community 110 - "20 — Testing Strategy"
Cohesion: 0.09
Nodes (22): 13. Resilience / chaos-lite, 16.1 Telugu localisation QA, 16.2 Accessibility audit (manual, pre-release), 16. Localisation QA and accessibility audit, 17. UAT, pilot, dogfooding, bug bash, 1. Decisions summary, 20.1 Backend, 20.2 Frontend (+14 more)

### Community 111 - "Part A — Free tiers for local / CI / dev-preview / demo"
Cohesion: 0.18
Nodes (11): 11. Sources (all accessed 2026-10-04), 12. Why free tiers are not acceptable for production (risk summary), 1. Requirements recap (from baseline), 2.1 Comparison table, 2.2 Low-cost paid VM reference (preview fallback only), 2. Backend compute, 4. Redis / cache (P6 check), 5. Object storage (+3 more)

### Community 112 - "32 — Final Plan Summary (Phase 1 close-out)"
Cohesion: 0.18
Nodes (11): 32 — Final Plan Summary (Phase 1 close-out), A. Recommended Architecture, B. Recommended Hosting Stack, C. V1 Feature List, D. Database Design Summary, E. API Summary, G. Testing Strategy, H. Deployment Strategy (+3 more)

### Community 113 - "devDependencies"
Cohesion: 0.18
Nodes (11): devDependencies, @rovo/config, @rovo/testing, @testing-library/jest-dom, @testing-library/react, @testing-library/user-event, @types/react, @types/react-dom (+3 more)

### Community 114 - "00003_catalog.sql"
Cohesion: 0.11
Nodes (37): addon_groups, addons, cuisines, item_variants, ix_addons__group, ix_item_variants__item, ix_leads__queue, ix_menu_item_addon_groups__group (+29 more)

### Community 115 - "db/db.go"
Cohesion: 0.22
Nodes (10): Connect(), InTx(), IsCheckViolation(), isCode(), IsExclusionViolation(), IsUniqueViolation(), PgError(), WithTx() (+2 more)

### Community 116 - "aws_cloudfront_distribution.this"
Cohesion: 0.12
Nodes (28): aws_cloudfront_cache_policy.api, aws_cloudfront_cache_policy.static, aws_cloudfront_distribution.this, aws_cloudfront_function.router, aws_cloudfront_origin_access_control.media, aws_cloudfront_origin_access_control.web, aws_cloudfront_origin_request_policy.api, aws_cloudfront_response_headers_policy.security (+20 more)

### Community 117 - "6. Functional requirements — Restaurant partner (RES-*)"
Cohesion: 0.20
Nodes (10): 6.1 RES-ONB — Onboarding and approval, 6.2 RES-PROF — Profile, 6.3 RES-HOUR — Operating hours, holidays, temporary pause, 6.4 RES-MENU — Menu management, 6.5 RES-AVAIL — Item availability, 6.6 RES-ORD — Order management, 6.7 RES-PREP — Preparation status and prep time, 6.8 RES-ANLY — Basic analytics (+2 more)

### Community 118 - "08 — System Architecture"
Cohesion: 0.20
Nodes (9): 08 — System Architecture, 0. Summary (one screen), 10. Multi-city expansion path, 11. Failure modes and graceful degradation, 12. What we are explicitly *not* doing in V1 (and why), 13. Version baseline (verified 2026-10-04), 14. Open items raised by this document, 1. System context (C4 level 1) (+1 more)

### Community 119 - "cities"
Cohesion: 0.14
Nodes (31): cities, assert_journal_balanced(), cod_deposits, commission_plans, invoice_sequences, invoices, ix_cod_deposits__queue, ix_cod_deposits__rider (+23 more)

### Community 120 - "5. Threat table"
Cohesion: 0.20
Nodes (10): 5.1 Authentication & sessions, 5.2 Authorisation & data access, 5.3 Web application & content, 5.4 Commerce, payments & fraud, 5.5 Admin & insider (application), 5.6 Uploads, storage & media, 5.7 Availability, abuse & cost, 5.8 Cloud control plane, infrastructure, data, secrets & supply chain (+2 more)

### Community 121 - "Part B — Production cloud comparison (staging + production)"
Cohesion: 0.20
Nodes (9): 0. Executive summary, 13. Production requirements (from baseline §4a), 16.1 Scoring (weights reflect §4a non-negotiables), 16.2 Decision, 16. Production recommendation, 17. Production-cloud sources (all accessed 2026-10-04), 18. Challenges to baseline, 25 — Hosting Comparison: Free Tiers (dev/preview) & Production Cloud (+1 more)

### Community 122 - "users"
Cohesion: 0.13
Nodes (28): customer_addresses, customer_profiles, erasure_requests, ix_customer_addresses__user, ix_erasure_requests__due, ix_localities__aliases, ix_localities__centroid, ix_localities__name_trgm (+20 more)

### Community 123 - "Dialog.tsx"
Cohesion: 0.33
Nodes (9): @radix-ui/react-dialog, BaseDialogProps, BottomSheet(), CloseButton(), Description(), Dialog(), DialogClose, Sheet() (+1 more)

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
Nodes (8): 6.1 Database provisioning and isolation, 6.3 Migrations, 6.4 PostGIS serviceability with Mahabubnagar fixtures, 6.5 River jobs and timers (fake clock), 6.6 Concurrency and race tests, 6.7 Per-subscriber jobs (River as outbox, R22/R42) and invariants, 6.8 SSE, 6. Go integration tests

### Community 129 - "14. Service-by-service comparison (India regions)"
Cohesion: 0.22
Nodes (9): 14.1 Region availability & DR pairing, 14.2 Managed containers (always-on worker + SSE), 14.3 Managed PostgreSQL + PostGIS, 14.4 Cache (only when P6 triggers), 14.5 Object storage, CDN, WAF, secrets, KMS, LB, NAT, 14.6 Observability (native vs Grafana Cloud), 14.7 Startup credits (all fetched 2026-10-04), 14.8 Portability / lock-in (+1 more)

### Community 130 - "27 — Implementation Backlog (Phase 2)"
Cohesion: 0.13
Nodes (9): 0. How to read this backlog, 1. Epic overview, 27 — Implementation Backlog (Phase 2), Appendix A — Requirement coverage map (P0 + committed P1), Appendix B — Effort roll-up (engineering stories only), Appendix C — Not in this backlog (by design), Appendix D — Stretch P1 handling, Appendix E — Lead's missing items (baseline §9, M1–M17) → stories (+1 more)

### Community 131 - "exports"
Cohesion: 0.22
Nodes (9): exports, ./eslint, ./fonts.css, ./theme.css, ./tokens, ./tsconfig/base.json, ./tsconfig/node.json, ./tsconfig/react.json (+1 more)

### Community 132 - "Phase 2 — Implementation Charter (binding for all implementation agents)"
Cohesion: 0.25
Nodes (7): 1. Workflow, 2. Toolchain (pinned), 3. V1 implementation slice (what "Phase 2 done" means here), 4. Conventions, 5. Agent ownership (avoid edit collisions), 6. Deliberate implementation deviations (ADRs), Phase 2 — Implementation Charter (binding for all implementation agents)

### Community 133 - "rovo — System Knowledge Graph"
Cohesion: 0.25
Nodes (7): 1. System graph (actors → apps → API → modules → stores), 2. Modules ↔ owned tables (verified from migrations), 3. Domain‑event graph (the golden flow), 4. Order ↔ delivery state machines (reference), 5. API surface (OpenAPI tag groups → module), 6. External systems (trust boundaries), rovo — System Knowledge Graph

### Community 134 - "7. Functional requirements — Delivery partner (RDR-*)"
Cohesion: 0.25
Nodes (8): 7.1 RDR-ONB — Registration and KYC, 7.2 RDR-AVAIL — Availability (online/offline), 7.3 RDR-ASSIGN — Order offers and assignment, 7.4 RDR-FLOW — Pickup and delivery workflow, 7.5 RDR-CONT — Contacting customer and restaurant, 7.6 RDR-HIST — Delivery history, 7.7 RDR-EARN — Earnings and cash-in-hand, 7. Functional requirements — Delivery partner (RDR-*)

### Community 135 - "1. UX principles (apply to all rovo apps: customer, restaurant, rider, admin — R14)"
Cohesion: 0.25
Nodes (8): 1.1 Mobile-first at 360 px, 1.2 Low bandwidth and low-end devices, 1.3 Telugu typography and bilingual content, 1.4 Accessibility — WCAG 2.2 AA, 1.5 Offline and poor-network messaging, 1.6 Fee transparency (non-negotiable) `[LEGAL]`, 1.7 Tone and trust, 1. UX principles (apply to all rovo apps: customer, restaurant, rider, admin — R14)

### Community 136 - "15. Cost estimates (on-demand, excl. GST; ₹95/USD) — single source of cost (R46)"
Cohesion: 0.25
Nodes (8): 15.1 AWS production profiles (R32 / M17), 15.2 AWS: 10× growth (≈ 5,000–20,000 orders/day, ~1,000 riders, ~15k SSE connections), 15.3 GCP alternative (asia-south1), 15.4 Azure (Central India), for completeness, 15.5 Other cost lines: non-AWS, one-time, and cut items (RV-014), 15.6 Per-order cloud cost by phase (R45 volumes), 15.7 CDN request-volume model and request reduction (M13, R27), 15. Cost estimates (on-demand, excl. GST; ₹95/USD) — single source of cost (R46)

### Community 137 - "PROMPT — rovo Phase 2: implement V1"
Cohesion: 0.22
Nodes (9): 1. Read first, in this order (this is your spec), 2. Hard constraints, 33 — Phase 2 Implementation Prompt, 4. Domain rules you must implement exactly, 5. How to work, 6. Definition of done for V1 development (end of M5, local), 7. Decisions that need humans (do not block on them; use the stated default), 8. Reporting (+1 more)

### Community 138 - "smoke.sh"
Cohesion: 0.64
Nodes (7): checkfail(), checkpass(), hdr(), req(), smoke.sh script, want_header(), want_status()

### Community 139 - "react.json"
Cohesion: 0.25
Nodes (7): compilerOptions, jsx, lib, types, extends, ./base.json, $schema

### Community 140 - "dependencies"
Cohesion: 0.25
Nodes (8): dependencies, lucide-react, @rovo/api-client, @rovo/auth, @rovo/i18n, @rovo/ui, @rovo/utils, @tanstack/react-query

### Community 141 - "devDependencies"
Cohesion: 0.25
Nodes (8): devDependencies, react, react-dom, @rovo/config, @types/react, @types/react-dom, typescript, vite

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

### Community 156 - "2. Phone OTP authentication (members)"
Cohesion: 0.33
Nodes (6): 2.1 Flow, 2.2 Parameters (defaults, config per city in `auth_settings`), 2.3 Account creation, 2.4 Phone-number change, 2.5 SIM-swap and account takeover controls, 2. Phone OTP authentication (members)

### Community 157 - "10. Incident response outline"
Cohesion: 0.33
Nodes (6): 10.1 Roles, 10.2 Severity, 10.3 Timeline, 10.4 Playbooks (written in the private ops repo before launch), 10.5 Contacts registry (fill before launch), 10. Incident response outline

### Community 158 - "11. Mobile / PWA tests"
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

### Community 175 - "database/main.tf"
Cohesion: 0.15
Nodes (24): aws_db_instance.this, aws_db_parameter_group.this, aws_db_subnet_group.this, aws_security_group.db, Terraform module: deploy/terraform/modules/database, output.db_name, output.endpoint, output.master_user_secret_arn (+16 more)

### Community 176 - "rovo infrastructure (OpenTofu)"
Cohesion: 0.40
Nodes (4): Applying later (M6, when AWS exists), Layout, rovo infrastructure (OpenTofu), Validate (never apply)

### Community 177 - "3. First open"
Cohesion: 0.40
Nodes (5): 3.1 Language pick (C-01), 3.2 Location (C-02, C-03), 3.3 Serviceability check, 3.4 "Not serviceable yet" state + waitlist (C-04), 3. First open

### Community 178 - "4. Home / discovery (C-05)"
Cohesion: 0.40
Nodes (5): 4.1 Layout (top → bottom at 360 px), 4.2 Sort options, 4.3 Filters, 4.4 Restaurant availability states in lists, 4. Home / discovery (C-05)

### Community 179 - "7. Consistency, idempotency and concurrency"
Cohesion: 0.40
Nodes (5): 7.1 Idempotency-Key on POST, 7.2 Concurrency control, 7.3 Transactional events (outbox) detail — amended 2026-10-04 per R22/R42, 7.4 Periodic jobs: catch-up semantics (M11), 7. Consistency, idempotency and concurrency

### Community 180 - "00006_payments.sql"
Cohesion: 0.17
Nodes (25): ix_pa_settlement_lines__entity, ix_pa_settlement_lines__unmatched, ix_payment_events__payment, ix_payments__order, ix_payments__pending, ix_recon_exceptions__queue, ix_refunds__open, ix_refunds__order (+17 more)

### Community 181 - "21. CI gating, coverage, flaky tests, runtime budgets"
Cohesion: 0.40
Nodes (5): 21.1 What runs when, 21.2 Coverage targets (measured, gated by module), 21.3 Runtime budgets and how we keep them, 21.4 Flaky-test policy, 21. CI gating, coverage, flaky tests, runtime budgets

### Community 182 - "8. Authorisation tests"
Cohesion: 0.40
Nodes (5): 8.1 Generated role × operation matrix, 8.2 IDOR suites per resource (hand-written, T1), 8.3 Session and token tests, 8.4 Maker-checker and audit tests (doc 12 §5.5, §5.7), 8. Authorisation tests

### Community 183 - "iam/main.tf"
Cohesion: 0.18
Nodes (23): aws_iam_openid_connect_provider.github, aws_iam_role.deploy, aws_iam_role.ecr_push, aws_iam_role_policy.deploy, aws_iam_role_policy.ecr_push, data.aws_caller_identity.current, data.aws_iam_policy_document.deploy, data.aws_iam_policy_document.deploy_trust (+15 more)

### Community 184 - "4. Metrics"
Cohesion: 0.40
Nodes (5): 4.2 USE (resources), 4.3 Async / pipeline, 4.4 Business KPIs (emitted on state transitions; city label for multi-city readiness), 4.5 Cardinality budget (Grafana Free: 10k active series [S71]), 4. Metrics

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

### Community 191 - "2. Containers (C4 level 2)"
Cohesion: 0.50
Nodes (4): 2.1 Logical containers (same in every environment), 2.2 Production / staging shape (managed cloud, provider-neutral), 2.3 Local development shape (Docker Compose), 2. Containers (C4 level 2)

### Community 192 - "4. Module boundary rules"
Cohesion: 0.50
Nodes (4): 4.1 The rules, 4.2 Compiler-enforced encapsulation (cheap and strong), 4.3 CI checks for what the compiler can't see, 4. Module boundary rules

### Community 193 - "6. Real-time design"
Cohesion: 0.50
Nodes (4): 6.1 Channels and topics, 6.2 Delivery path: worker → API process, 6.3 Rider location ingestion, 6. Real-time design

### Community 194 - "storage/main.tf"
Cohesion: 0.15
Nodes (22): aws_s3_bucket_lifecycle_configuration.kyc, aws_s3_bucket_lifecycle_configuration.logs, aws_s3_bucket_lifecycle_configuration.media, aws_s3_bucket_lifecycle_configuration.web, aws_s3_bucket_policy.this, aws_s3_bucket_public_access_block.this, aws_s3_bucket_server_side_encryption_configuration.this, aws_s3_bucket.this (+14 more)

### Community 195 - "22. Definition of Done, bug severity, triage SLAs, release gates"
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

### Community 203 - "api-client/tsconfig.json"
Cohesion: 0.50
Nodes (3): extends, include, @rovo/config/tsconfig/react.json

### Community 204 - "devDependencies"
Cohesion: 0.50
Nodes (4): devDependencies, @types/node, typescript, vitest

### Community 205 - "Telugu translations — native review queue"
Cohesion: 0.50
Nodes (3): Glossary used (validate with users, doc 17 §10), Rules for new strings, Telugu translations — native review queue

### Community 206 - "i18n/tsconfig.json"
Cohesion: 0.50
Nodes (3): extends, include, @rovo/config/tsconfig/react.json

### Community 207 - "5. Search (C-06)"
Cohesion: 0.67
Nodes (3): 5.1 Behaviour, 5.2 Telugu and transliteration, 5. Search (C-06)

### Community 208 - "3. Components: the Go modular monolith (C4 level 3)"
Cohesion: 0.67
Nodes (3): 3.1 Module map and dependency direction, 3.3 Event catalogue conventions, 3. Components: the Go modular monolith (C4 level 3)

### Community 209 - "customer/src/router.tsx"
Cohesion: 0.08
Nodes (24): accountRoute, addressEditRoute, addressesRoute, addressNewRoute, cartRoute, checkoutRoute, cityRoute, indexRoute (+16 more)

### Community 210 - "00009_ratings_support_notifications.sql"
Cohesion: 0.14
Nodes (23): ix_notification_deliveries__notification, ix_notifications__inbox, ix_notifications__unread, ix_push_subscriptions__user, ix_ratings__restaurant, ix_ratings__rider, ix_reviews__restaurant_public, ix_support_tickets__order (+15 more)

### Community 211 - "00010_admin_platform.sql"
Cohesion: 0.13
Nodes (20): app_config, approval_requests, audit_logs, feature_flags, file_objects, idempotency_keys, ix_approval_requests__post_review, ix_approval_requests__queue (+12 more)

### Community 212 - "2. Quality goals and risk-based prioritisation"
Cohesion: 0.67
Nodes (3): 2.1 Quality goals (what "good" means for V1), 2.2 Risk register driving test intensity, 2. Quality goals and risk-based prioritisation

### Community 213 - "observability/main.tf"
Cohesion: 0.23
Nodes (15): aws_cloudwatch_log_group.app, aws_cloudwatch_log_group.security, aws_cloudwatch_log_metric_filter.app_events, aws_cloudwatch_metric_alarm.cert_in_delivery, aws_sns_topic.alerts, aws_sns_topic_subscription.email, Terraform module: deploy/terraform/modules/observability, output.alerts_topic_arn (+7 more)

### Community 216 - "scripts"
Cohesion: 0.67
Nodes (3): scripts, test, typecheck

### Community 217 - "peerDependencies"
Cohesion: 0.67
Nodes (3): peerDependencies, react, react-dom

### Community 218 - "scripts"
Cohesion: 0.67
Nodes (3): scripts, test, typecheck

### Community 219 - "exports"
Cohesion: 0.67
Nodes (3): exports, ./msw, ./setup

### Community 220 - "peerDependencies"
Cohesion: 0.67
Nodes (3): peerDependencies, react, react-dom

### Community 221 - "scripts"
Cohesion: 0.67
Nodes (3): scripts, test, typecheck

### Community 222 - "peerDependencies"
Cohesion: 0.67
Nodes (3): peerDependencies, react, react-dom

### Community 223 - "scripts"
Cohesion: 0.67
Nodes (3): scripts, test, typecheck

### Community 232 - "00004_pricing_promotions.sql"
Cohesion: 0.24
Nodes (14): coupon_redemptions, coupon_targets, coupons, fee_configs, ix_coupon_redemptions__user, ix_coupon_targets__target, ix_coupons__live, ix_coupons__owner (+6 more)

### Community 233 - "idgen.go"
Cohesion: 0.21
Nodes (4): Generator, RandomCrockford(), TestDefault(), Default

### Community 234 - "4. Tokens, sessions, cookies, CSRF, CORS, SSE"
Cohesion: 0.18
Nodes (11): 4.10 Signing-key management and rotation, 4.1 Host topology (decision AUTH-D03, refines P7), 4.2 Access token (JWT), 4.3 Refresh tokens: rotation and reuse detection, 4.4 Cookies, 4.5 CSRF strategy (AUTH-D06), 4.6 CORS policy, 4.7 SSE authentication (+3 more)

### Community 235 - "respond.go"
Cohesion: 0.31
Nodes (9): decodeDetail(), DecodeJSON(), Error(), firstNonEmpty(), hasJSONContentType(), kebab(), trimSpace(), RequestID() (+1 more)

### Community 236 - "orders"
Cohesion: 0.40
Nodes (8): assert_order_bill_balanced(), ix_order_charges__order, ix_order_status_history__occurred, order_status_history, orders, trg_order_charges_balanced, trg_orders_bill_balanced, ux_order_status_history__command

### Community 237 - "uuidv7"
Cohesion: 0.25
Nodes (8): RFC-9562, ADR-005: PostgreSQL 18 + PostGIS; pgx + sqlc; goose migrations, ADR-019: UUIDv7 primary keys, 0. Decision summary, 20. Sources (accessed 2026-10-04), useIdempotencyKey(), newIdempotencyKey(), uuidv7()

### Community 238 - "phone.ts"
Cohesion: 0.57
Nodes (6): E164, formatIndianMobile(), isValidIndianMobile(), maskIndianMobile(), normalizeIndianMobile(), toNationalMobile()

### Community 239 - "5. Authorisation model"
Cohesion: 0.29
Nodes (7): 5.1 Principles, 5.3 Resource-level (ABAC) rules (L2 policy functions), 5.4 Where checks live (summary), 5.5 Maker-checker (four-eyes), 5.6 Impersonation (decision AUTH-D11), 5.7 Audit logging, 5. Authorisation model

### Community 240 - "18. Test data management"
Cohesion: 0.29
Nodes (7): 18.1 Principles, 18.2 Seed profiles, 18.3 Mahabubnagar `e2e` dataset (illustrative), 18.4 Factories / builders, 18.5 Production-derived data and anonymisation (DPDP), 18.6 Deterministic clocks and IDs, 18. Test data management

### Community 241 - "(a) Risk register"
Cohesion: 0.29
Nodes (7): (a) Risk register, Financial, Legal & regulatory, Operational, Product & market, Security & privacy, Technical

### Community 242 - "3. Admin authentication"
Cohesion: 0.33
Nodes (6): 3.1 Credentials, 3.2 TOTP (RFC 6238), 3.3 Provisioning, reset, offboarding, 3.4 Sessions, timeouts and step-up, 3.5 Network restrictions for the admin console (R37), 3. Admin authentication

### Community 243 - "3. Logs"
Cohesion: 0.33
Nodes (6): 3.1 Schema (JSON, one line per event), 3.2 Levels, 3.3 PII redaction rules (enforced in code + collector) [LEGAL: DPDP], 3.4 Volume control, 3.5 Audit log ≠ observability, 3. Logs

### Community 245 - "1. Identity model"
Cohesion: 0.40
Nodes (5): 1.1 Entities, 1.2 Roles and scopes, 1.3 Can a person be both customer and rider on one phone? (decision), 1.4 Phone-number recycling (India-specific risk), 1. Identity model

### Community 246 - "4. Network"
Cohesion: 0.40
Nodes (5): 4.1 VPC layout (`ap-south-1`, 2 AZs; a 3rd AZ is reserved in the CIDR plan), 4.2 Security groups, 4.3 Decision: no NAT in `closed-pilot`, one NAT before Gate B (R28), 4.4 Administrative access, 4. Network

### Community 247 - "2. Instrumentation"
Cohesion: 0.50
Nodes (4): 2.1 Backend (Go), 2.2 Frontend (4 SPAs/PWAs: customer, restaurant, rider, admin — R14), 2.3 Collector: none in `closed-pilot`; optional Alloy gateway in `public-launch`, 2. Instrumentation

## Knowledge Gaps
- **238 isolated node(s):** `github.com/storetmobiles2-code/rovo/backend`, `github.com/storetmobiles2-code/rovo/deploy/docker/healthcheck`, `ROVO_TEST_DATABASE_URL`, `PG_CONTAINER`, `@rovo/api-client` (+233 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 2154 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **22 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `09 — Architecture Decision Records` connect `09 — Architecture Decision Records` to `context.Context`, `27 — Implementation Backlog (Phase 2)`, `1. Conventions`, `i18n/src/index.ts`, `uuidv7`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **What connects `github.com/storetmobiles2-code/rovo/backend`, `github.com/storetmobiles2-code/rovo/deploy/docker/healthcheck`, `ROVO_TEST_DATABASE_URL` to the rest of the system?**
  _238 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `schema.d.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.02 - nodes in this community are weakly interconnected._
- **Why does `2. Backlog by epic` connect `2. Backlog by epic` to `context.Context`, `27 — Implementation Backlog (Phase 2)`, `load`?**
  _High betweenness centrality (0.060) - this node is a cross-community bridge._
- **Should `ui/src/index.ts` be split into smaller, more focused modules?**
  _Cohesion score 0.06964006259780908 - nodes in this community are weakly interconnected._
- **Why does `20 — Testing Strategy` connect `20 — Testing Strategy` to `15. Data, migration and backup-restore testing`, `6. Go integration tests`, `7. Contract tests`, `22. Definition of Done, bug severity, triage SLAs, release gates`, `load`, `9. Frontend tests`, `10. End-to-end tests`, `18. Test data management`, `4. Go unit tests`, `2. Quality goals and risk-based prioritisation`, `21. CI gating, coverage, flaky tests, runtime budgets`, `8. Authorisation tests`, `11. Mobile / PWA tests`, `14. Security testing`?**
  _High betweenness centrality (0.058) - this node is a cross-community bridge._
- **Should `restaurant/src/router.tsx` be split into smaller, more focused modules?**
  _Cohesion score 0.08792270531400966 - nodes in this community are weakly interconnected._