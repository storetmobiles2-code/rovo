# 30 — Risks, Assumptions, Decisions & Open Questions (RAID)

| Field | Value |
|---|---|
| **Purpose** | One consolidated register for Phase 2: (a) risks, (b) significant assumptions from docs 01–26 with how to validate them, (c) decisions log (ADRs, Lead rulings R1–R48, user directives, scope cuts), (d) unresolved decisions/open questions (deduplicated `[OPEN]` and `[LEGAL]` items incl. those raised by review doc 31) with owner and decide-by milestone, (e) remaining cross-document conflicts for reconciliation. |
| **Owner** | Release Architect (register); each row has an owner role |
| **Status** | Draft v1 (2026-10-04) — includes baseline §9 (R27–R48, cuts C1–C20, missing items M1–M17) |
| **Depends on** | `00` (§4a, §8, §9), `01`–`26`, `27`–`29`, `31-review-report.md` |
| **Feeds** | Weekly triage (doc 02 §6), Go/No-Go (doc 29 §15), `32`, `33` |

**Scales.** Likelihood (L) and Impact (I): **H**igh / **M**edium / **L**ow. **Owner roles:** FDR founder(s) · PL product/ops lead · RA release architect · LEAD lead architect · BE backend lead · FE frontend lead · DO DevOps owner · SEC security owner · FIN finance · OPS ops lead · SUP support lead · LEGAL counsel · CA chartered accountant. **Milestones** per doc 28 (M0–M8). Review cadence: weekly during build, daily during pilot and hypercare.

---

## (a) Risk register

### Product & market

| ID | Risk | L | I | Mitigation | Owner | Trigger / early warning |
|---|---|---|---|---|---|---|
| RK-01 | **Restaurant adoption too slow** to reach ≥ 10 (Gate A) and ≥ 40 (Gate B) live outlets; owners wary of a new brand or already tied to national apps | H | H | LOIs from M3; ops-assisted onboarding + bulk CSV menu import (E05-S08); promotional 0% commission window (ADM-COMM-003); counter device supplied (E29-S09); transparent fee story | OPS | < 2 restaurants/week onboarded in M6; LOI conversion < 50% |
| RK-02 | **Rider supply/earnings not viable** at pilot volumes (≈ 30–80 orders/day) — riders leave | H | H | `MG_TOPUP` minimum guarantee per peak slot (R47); demand-sized rider pool (≈ 1 per 3 peak-hour orders) instead of a fixed 35; transparent pay; insurance decision | PL + FIN | Offer acceptance < 60%; > 20% of riders inactive for a week |
| RK-03 | **Competition from Swiggy/Zomato** in Mahabubnagar (discounts, exclusivity, rider poaching) | M | H | Local-first value: Telugu UX, lower commission, fast settlements, local support; avoid exclusivity fights; measure share of multi-homing restaurants | FDR | Restaurants asking to delist; competitor promotions in town |
| RK-04 | Customer trust/demand weaker than assumed (new brand; COD demand; price sensitivity) | M | H | Visible fees and FSSAI, COD within caps, reliable refunds, launch coupons budgeted, word of mouth via restaurants | PL | Pilot repeat rate < 30%; checkout drop-off > 50% |
| RK-05 | Restaurants miss orders (shared cheap phone, PWA backgrounded, noisy kitchen) → acceptance below 90% | H | H | R1 ladder, device-bound sessions (R44), heartbeat auto-pause, supplied counter devices, ops manual call at 90 s; voice escalation P1 if > 5% reach 90 s (R43) | OPS | PP-05 > 5%; M-01 < 90% in pilot |
| RK-06 | Scope creep after scope freeze delays launch | M | M | Doc 02 §6 one-in-one-out; scope freeze at M4 exit; cuts C1–C20 enforced | RA | > 2 change requests/month accepted after M4 |

### Technical

| ID | Risk | L | I | Mitigation | Owner | Trigger |
|---|---|---|---|---|---|---|
| RK-10 | **Schedule overrun**: effort (≈ 481 ideal days M0–M5) consumes all capacity; no slack in M0–M5 | H | H | M0 velocity checkpoint; P1 parts cut first; 2-engineer cut list (doc 28 §3.3); launch buffer 3 weeks | RA | Velocity < 10 ideal days/week at M0 exit; any milestone > 1 week late |
| RK-11 | OpenAPI 3.1 tooling gaps (codegen/mocks) | M | M | Week-1 spike, fallback to 3.0.3 (R20) | BE | Spike fails any of the four tools |
| RK-12 | **Fake clock vs River DB time**: River selects due jobs by DB time; `/_test` "run due" may need rescheduling; ledger cut-offs must use app clock (RV-061) | M | H | E00-S09 proves timer firing in M0; app-clock accounting dates (R19); fallback: reschedule-on-advance helper | BE | M0 exit criterion 4 fails |
| RK-13 | Money bugs (rounding, GST, refunds, payouts) → paise errors, double captures | M | H | Golden tables incl. CA-signed invoices (M14), property tests, ledger invariants nightly, two reviewers on T1 code | BE | Any invariant failure; recon exception not explained in 24 h |
| RK-14 | SSE behind CloudFront/ALB drops or LISTEN/NOTIFY stalls → restaurants miss orders | M | H | 20 s heartbeat, refetch on reconnect, polling fallback, LISTEN watchdog + queue-usage alert (M12), 2 h/8 h edge soaks | BE + DO | C-9b failure; A18 SSE drop alert |
| RK-15 | Web Push/PWA limits on low-end Android and iOS (no looping sound, killed tabs) | H | M | Layered alert design (doc 18 §7), device lab, ops call, voice escalation P1 | FE | Device matrix failures; PP-05 |
| RK-16 | Rider location spoofing / stale GPS (residual M–H, doc 19 T-45) | M | M | Plausibility checks, soft geofence flags, two staleness tiers (R34), delivery OTP for prepaid ≥ ₹300 (R39) | SEC | Disputes about "delivered" > 1% of orders |
| RK-17 | CloudFront flat-rate request allowance exceeded (chatty pings/heartbeats/SSE) | L | M | Batched pings, 60 s heartbeat, SSE presence as heartbeat (R27); request dashboard (M13); pay-as-you-go fallback | DO | > 70% of monthly allowance by day 20 |
| RK-18 | Local-only development hides cloud-specific issues (IAM, PostGIS version, pooler/LISTEN, LB timeouts) until M6 | M | M | Same images and PG major locally; IaC authored and policy-checked in M5; M6 has 7 weeks incl. slack; chaos/soak in staging | DO | M6 week 3 staging still not green |

### Security & privacy

| ID | Risk | L | I | Mitigation | Owner | Trigger |
|---|---|---|---|---|---|---|
| RK-20 | OTP abuse / SMS pumping (cost and availability) | M | M | Bot challenge on every OTP request, shared rate limits, budget breaker with strict mode, provider daily caps | SEC | Send→verify conversion < 50%; SMS spend > 2× baseline |
| RK-21 | Coupon multi-accounting and refund abuse (residual M) | M | M | Account + phone + device/address-cell caps, budget caps, claim-history auto-resolve limits, weekly fraud review | SUP | Coupon cost > budget; claim ratio > 10% |
| RK-22 | COD cash theft / leakage by riders | M | M | ₹2,000 cash limit gating (R6), ageing blocks, daily COD reconciliation, deposit confirmation, police-complaint guidance [LEGAL] | FIN | Unexplained COD variance > ₹0 for a weekly close |
| RK-23 | Insider PII browsing; admin account phishing (TOTP only in V1, no IAP) | M | H | Masking + reveal with reason + audit, step-up, WAF rate + India geo on admin host (R37), passkeys P1, access reviews | SEC | Reveal-rate alert; admin TOTP failure bursts |
| RK-24 | Cloud operator credential compromise (small team) | L | H | SSO + FIDO2, no IAM users, OIDC pinned, SCPs, GuardDuty, Object-Lock backups | DO | GuardDuty high finding; root login |
| RK-25 | Accepted descopes weaken detection: no hash-chained audit/WORM anchor (SEC-127/128), no PDF/malware scanning (images only), no Sentry | L | M | Append-only grants; images re-encoded; Faro + logs for errors; revisit after pilot | SEC | Any audit-integrity question in an incident |
| RK-26 | Personal-data breach with CERT-In 6 h / DPB reporting obligations | L | H | IR plan, templates, tabletop, 180-day India log archive (M1), PII redaction | SEC + LEGAL | Any suspected breach |

### Legal & regulatory

| ID | Risk | L | I | Mitigation | Owner | Trigger |
|---|---|---|---|---|---|---|
| RK-30 | **Money-flow model ruled unlawful or needs split settlement** (RBI PA directions) | M | H | Legal opinion before Phase-2 week 4 (R35); interface supports both models (R25); linked-account KYC planned | LEGAL | Opinion not received by week 4 |
| RK-31 | GST characterisation (delivery fee §9(5) vs own supply, fee presentation, discounts) wrong → tax exposure | M | H | CA opinion + CA-signed golden invoices (M14); data-driven tax rules | CA | Opinion not received by M3 exit |
| RK-32 | **PA onboarding/KYC delays** (entity, GSTIN, website policies, marketplace approval) push Gate A | M | H | Start sandbox in M3, KYC at M5 start, ≈ 3–5 weeks slack | FIN | KYC not approved 4 weeks before Gate A |
| RK-33 | **DLT registration/template rejection** → no real OTP at Gate A | M | H | PE/header early (after entity); copy freeze M5 week 4; simple English OTP template; aggregator help | PL | Template rejected twice |
| RK-34 | FSSAI e-commerce licence delay | M | M | Apply right after incorporation; 6–10 weeks lead | LEGAL | Not issued by M5 |
| RK-35 | Gig-worker obligations (Social Security Code, Telangana law) add cost/process; insurance mandated | M | M | Counsel opinion, registration fields + export (M5), insurance decision before recruitment | LEGAL | Notification of contribution date or state act |
| RK-36 | Consumer-protection non-compliance (drip pricing, dark patterns, grievance SLA) | L | H | Counsel review of checkout screens, grievance clock, all-fee display | LEGAL | Complaint to authorities |
| RK-37 | DPDP Rules timeline/interpretation shifts (full enforcement ≈ May 2027) | M | M | Comply from day one; erasure map (M15); retention schedule by counsel | LEGAL | Rule amendments |

### Financial

| ID | Risk | L | I | Mitigation | Owner | Trigger |
|---|---|---|---|---|---|---|
| RK-40 | **PA fees exceed the ₹5 platform fee** (2% on ₹300–360 baskets) | H | M | Effective blended rate ≤ 1% as PA go/no-go (R46); UPI-heavy mix; negotiate launch offers | FIN | Effective rate > 1.2% after month 1 |
| RK-41 | Cloud cost per order too high at pilot volume (₹14–17k/month vs ≈ 30 orders/day) | H | M | `closed-pilot` profile (R32), staging stopped when idle, budgets/alerts (R46), AWS Activate credits | FIN + DO | Cost per order (M-60) not trending to ≤ platform fee by month 3 |
| RK-42 | COD leakage and goodwill/compensation costs | M | M | Caps (R6), strikes (R5), R29 policy, R31 approvals > ₹150 goodwill | FIN | Compensation > 2% of GMV |
| RK-43 | `MG_TOPUP` and launch coupon costs exceed budget | M | M | Slot-limited guarantee, weekly review, coupon budget caps | FDR | Weekly spend > plan by 25% |
| RK-44 | Manual finance labour (payout batches, deposits, reconciliation) overwhelms a part-time finance admin | M | M | Finance calendar, labour plan, automated payouts in V1.1 trigger (> 100 payees/week) | FIN | > 2 h/day finance ops |

### Operational

| ID | Risk | L | I | Mitigation | Owner | Trigger |
|---|---|---|---|---|---|---|
| RK-50 | **Tiny team on-call** for 108+ service hours/week; burnout; slow incident response | H | H | Service-hours on-call only (overnight infra P1 only), primary + secondary rota, one incident channel, runbooks, business alerts routed to ops desk, SLO 99.5% in pilot | RA | Two P1s acknowledged late in a month |
| RK-51 | Support staffing (Telugu-speaking) insufficient at launch | M | M | Phone line + rota (M9), canned responses, help centre | SUP | Ticket ack SLA misses |
| RK-52 | Monsoon/festival peaks disrupt pilot or launch (rain, Dasara, Diwali) | M | M | Zone pause + rider peak bonus (R30), launch after Diwali 2027, ETA weather buffer | OPS | IMD heavy-rain alerts |
| RK-53 | Key-person dependency (one DevOps-capable engineer) | M | H | Runbooks, IaC only, pairing on M6, second engineer shadows on-call | RA | Planned leave in M6–M8 |
| RK-54 | Supply/ops readiness slips Gate B even when software is ready | H | M | Supply is tracked as critical path (doc 28 §4.2); 3-week buffer; bar not lowered | OPS | < 30 restaurants live 2 weeks before Gate B |

---

## (b) Assumptions register (significant `[ASSUMPTION]`s from docs 01–26)

| ID | Assumption | Source | How / when to validate | Owner |
|---|---|---|---|---|
| AS-01 | Urban population ≈ 2–2.5 lakh (agglomeration ≈ 3 lakh); compact city, most trips 2–6 km | 00 §1, 01 §2.1, 16 §11 | Census/Wikipedia check done (16 §11); pilot trip distances | PL |
| AS-02 | Named localities and PIN codes beyond 509001; single ~5 km core zone covers most demand | 01 §2.1, 16 §11, 10 §15 | Ops field mapping (E29-S04) before M6 | OPS |
| AS-03 | Planning volumes: pilot ≈ 30/day, month 1 ≈ 80/day, month 3 ≈ 250/day; design 2,000/day, 500/h peak (R45) | 20 §12, 00 §9 | Pilot data; re-baseline at M8 | PL |
| AS-04 | AOV ≈ ₹300–361; 70% UPI share among online payments | 09 ADR-012, 14 §3, 31 §2 | Pilot data | FIN |
| AS-05 | Restaurants commonly pay 18–30% commission to national apps; 15% default is attractive | 01 §2.2 | Recruitment interviews (≥ 20) in M3–M4 | OPS |
| AS-06 | ≥ 80% of restaurants run on one Android phone with the PWA; counter device on 2–3 GB RAM Android 9+/10+ | 02 §4, 05 §1 | Recruitment survey; device lab (E26-S06) | OPS |
| AS-07 | Low-end Android + Chrome ≥ 120 dominates; iOS share small (≈ 4%) | 18 §11, 20 | Pilot participants' devices; Faro RUM | FE |
| AS-08 | Customers read Telugu UI well; Western digits used in both locales; Urdu demand modest | 01 §2.1, 04 §1.3, 06 §1 | Native review; pilot survey (OQ-09) | PL |
| AS-09 | Restaurant accept within 180 s is achievable with alerts + ops calls | 01 BR-TIME-001, R1 | Pilot PP-01/PP-05 | OPS |
| AS-10 | Rider approach ≈ 8 min, speed ≈ 18–20 km/h, road factor 1.3, rain factor 0.8 | 13 §5, 16 §5, 01 BR-TIME-005 | First 2 weeks of pilot data | OPS |
| AS-11 | Delivery OTP threshold ₹300 prepaid; COD caps ₹1,000 / ₹600 first order; cash limit ₹2,000 | 01 BR-OTP-001, R6, R39 | Pilot fraud/dispute data (OQ-04) | PL |
| AS-12 | Rider waiting pay ₹1/min after 10 min (cap ₹20); first-mile unpaid | 01 BR-RPAY-002/003 | Rider feedback; offer acceptance ≥ 70% | OPS |
| AS-13 | Refund timelines (UPI 2–5, cards 5–7 business days) and PA late-capture behaviour | 01 BR-REF-003, 04 | PA documentation at onboarding | FIN |
| AS-14 | PCI scope SAQ-A-equivalent with hosted checkout | 14 §0 | PA confirmation at onboarding | SEC |
| AS-15 | PA merchant KYC checklist and 1–3 days per restaurant linked account | 14 | PA onboarding call (M3) | FIN |
| AS-16 | DLT PE registration ₹5,900 and templates approved in 1–7 working days; header `ROVOIN` available | 15 §5–6, 01 LEG-TRAI | Registration (E28-S06) | PL |
| AS-17 | SMS/WhatsApp per-OTP ≈ ₹0.12–0.20; messaging ≈ ₹1.5–3k/month at month 3 | 15 §5, §9 | Written quotes (N-1) | FIN |
| AS-18 | FSSAI verification is manual on FoSCoS (no API) | 05, 07 | Ops procedure test | OPS |
| AS-19 | Indian operators recycle numbers after ~90 days; 180-day inactivity detach flow is adequate | 12 §2 | Counsel/telecom check | SEC |
| AS-20 | Grafana Cloud free/Pro limits and `ap-south-1` residency hold at pilot volume; telemetry ≈ 21 GB logs/month | 24 §1, §11 | Verify pricing at M6; usage dashboard | DO |
| AS-21 | RDS PostgreSQL 17 + PostGIS 3.5 in `ap-south-1`; PITR granularity ≈ 5 min; cross-Region backup lag ≤ 30 min | 23 §1, 25 §0 | Restore drills (E25-S03) | DO |
| AS-22 | Fargate ARM + CloudFront flat-rate Pro prices; INR at ₹95/USD | 25 §15 | AWS calculator at M6; budgets | DO |
| AS-23 | No NAT needed for the closed pilot (public subnets + controls) | 22 §4.3, R28 | Provider IP allow-listing needs (PA/SMS) | DO |
| AS-24 | ≤ 300 orders/day in first 3 months keeps manual payouts and deposit checks sustainable | 02 §4 | Finance time tracking (RK-44) | FIN |
| AS-25 | Launch team: 1 ops lead + 2–3 ops/support agents + part-time finance + 1–2 field staff | 02 §4 | Hiring plan by M5 | FDR |
| AS-26 | Phase 2 team of 3 engineers with Claude Code at ≈ 12 ideal days/week; size weights S/M/L = 1/2.5/5 | 28 §1 | M0 velocity checkpoint | RA |
| AS-27 | Service hours 08:00–23:30 IST; late-night demand limited | 01 BR-TIME-008 | Pilot (OQ-12) | OPS |
| AS-28 | OSM/OpenFreeMap coverage adequate for pin-drop; tile terms permit commercial use | 16 §11, ADR-015 | Terms confirmation (E28-S11); field check | PL |
| AS-29 | Two-tier dispatch reaches backgrounded riders via push within the offer window | R34, 06 | Field test FN-06 | QA |
| AS-30 | Packaging caps ₹30/item, ₹50/order; menu price bounds ₹0–10,000 | 01 BR-FEE-004, BR-MENU-002 | Restaurant interviews | OPS |
| AS-31 | Holidays and festival dates in the plan (Diwali 2026/2027, Sankranti, Dasara) | 28 §1 | Verify calendar at M0 | RA |

---

## (c) Decisions log

### User directives

| ID | Decision | Date |
|---|---|---|
| UD-1 | rovo is production-ready and cloud-deployable; production must not use non-standard/free-tier hosting (baseline §4a) | 2026-10-04 |
| UD-2 | Development uses local Docker Compose only; no free tier that requires card details (R24) | 2026-10-04 |

### Architecture Decision Records (doc 09)

| ADR | Decision (one line) |
|---|---|
| 001 | Modular monolith in Go (api + worker modes), not microservices |
| 002 | Go backend with stdlib `net/http` ServeMux |
| 003 | REST/JSON, OpenAPI spec-first, oapi-codegen + openapi-typescript (3.1 gate, R20) |
| 004 | No gRPC in V1 |
| 005 | PostgreSQL + PostGIS; pgx + sqlc; goose (PG 17 on RDS per R22) |
| 006 | River as transactional outbox (no relay; refined by R42) |
| 007 | No Redis at launch; Redis-protocol adapter by config later |
| 008 | Real-time via SSE (R10 rules) |
| 009 | Vite React SPA/PWA (no Next.js); four apps (R14) |
| 010 | Monorepo (Go module + pnpm workspace) |
| 011 | Hybrid EdDSA JWT + server-side session state |
| 012 | PA chosen on written UPI rate behind `PaymentProvider`; split settlement preferred pending legal (R25, R35) |
| 013 | Integer paise + double-entry ledger |
| 014 | Rule-based dispatch with offer cascade |
| 015 | MapLibre + OpenFreeMap; no paid geocoding |
| 016 | Managed containers on a hyperscaler for staging/prod; Compose for local |
| 017 | Observability via OpenTelemetry (backend per R36) |
| 018 | i18n en + te; `*_i18n` JSONB (R17) |
| 019 | UUIDv7 IDs generated in the app |
| 020 | Multi-city via shared schema with `city_id` |
| 021 | Search: Postgres FTS + pg_trgm |
| 022 | Uploads via presigned URLs to object storage (KYC images only, R38) |
| 023 | API conventions: `/api/v1` (R15), problem+json, cursor pagination, Idempotency-Key, ETag |
| 024 | CI/CD on GitHub Actions with OIDC to the cloud |
| 025 | Cloud portability through standard interfaces only |
| 026 | Infrastructure as Code with OpenTofu (`deploy/terraform`) |

### Lead rulings (baseline §8, §9)

| ID | Ruling (one line) |
|---|---|
| R1 | Accept window 180 s: repeat 30 s, owner SMS 60 s, ops 90 s, then `CANCELLED`/`SYSTEM`/`RESTAURANT_UNRESPONSIVE`, refund, 30-min pause; 2 misses → paused; 3-min heartbeat loss → pause |
| R2 | Customer free cancel while `PLACED` or within 60 s of placement; later via support |
| R3 | `ACCEPTED → PREPARING` after 60 s or on tap; prep time chosen at accept |
| R4 | Rider "Picked up" from `PREPARING` or `READY_FOR_PICKUP` (`restaurant_skipped_ready`) |
| R5 | `UNDELIVERABLE` needs support approval; 2 customer-fault COD failures → COD disabled |
| R6 | COD cap ₹1,000 (₹600 first order); COD offered only if cash-in-hand + order ≤ ₹2,000 |
| R7 | Delivery created at `ACCEPTED`; offer at `max(0, prep − approach − buffer)` |
| R8 | Payable rounded to rupee with `ROUND_OFF`; menu GST-exclusive; fee GST presentation configurable [LEGAL] |
| R9 | Goodwill/compensation as single-user coupons; no wallet (refined by R29) |
| R10 | Single SSE stream, 20 s heartbeat, `reauth`, no replay; LB/CDN idle ≥ 120 s |
| R11 | Add `approval_requests`, `reason_codes`, `restaurant_devices`, `rider_availability`, ticket statuses |
| R12 | No server cart; signed 10-min `quoteId`; order needs `quoteId` + `Idempotency-Key` |
| R13 | Landmark + map pin required; building/street optional |
| R14 | Four apps/hosts (`app.`, `restaurant.`, `rider.`, `admin.`), same-origin `/api`, workspace under `web/` |
| R15 | API base path `/api/v1` everywhere |
| R16 | Offer status `REVOKED` added |
| R17 | Translatable fields as `*_i18n` JSONB; API `nameI18n` + `displayName` |
| R18 | Fee slabs on road-adjusted distance to 10 km, `[lo, hi)`; radius 7 km straight-line |
| R19 | Injected app clock; testhooks mandatory from sprint 1 |
| R20 | OpenAPI 3.1 subset, week-1 spike, fallback 3.0.3 |
| R21 | Rate limits and OTP counters in Postgres until Redis |
| R22 | River insert is the outbox; PostgreSQL 17 on RDS (18 if PostGIS available) |
| R23 | AWS Mumbai primary, Hyderabad DR; ECS Fargate ARM, RDS, S3 + CloudFront, WAF, Secrets Manager, KMS; OpenTofu |
| R24 | Development on local Docker Compose only |
| R25 | PA chosen on written rates (Cashfree vs Razorpay); settlement model with counsel; design supports both |
| R26 | Admins separate identities; `RIDER` ⟂ `RESTAURANT_*`; `SYSTEM` principal |
| R27 | `/api/*` same-origin via CloudFront flat-rate Pro (PAYG fallback); no public `api.`/CORS; reduce chatty traffic |
| R28 | Pilot: public subnets with compensating controls; one NAT before Gate B |
| R29 | COD compensation by customer choice: UPI refund (UTR) or coupon; never coupon-only |
| R30 | No surge in V1; zone pause + rider peak bonus |
| R31 | Maker-checker only for five action families; break-glass with 24 h review; ≥ 2 money approvers |
| R32 | Pilot RDS Single-AZ t4g.small; Multi-AZ before Gate B or > 100 orders/day; IaC profiles `closed-pilot` / `public-launch` |
| R33 | No build-time prerender; static OG + Go `/r/{slug}` share pages (P1) |
| R34 | Dispatch tiers: fresh ≤ 3 min, stale ≤ 15 min via push; auto-offline at 15 min |
| R35 | Legal opinion on money flow before Phase-2 week 4; linked-account KYC on path if split |
| R36 | Grafana Cloud incl. Faro; CERT-In 180-day India log archive, security events ≥ 1 year; no Sentry |
| R37 | Mandatory TOTP for all admins; passkeys P1; WAF rate + geo on admin; no IAP |
| R38 | KYC images only, SSE-KMS, audited view; no ClamAV / app-layer file encryption; field encryption for bank accounts + TOTP secrets |
| R39 | Delivery OTP on for prepaid ≥ ₹300, stored for customer display, off for COD |
| R40 | No restaurant self-cancel after accept; ops-mediated |
| R41 | Cross-module FKs to `orders` allowed on money paths |
| R42 | No event/outbox table; publisher inserts one River job per subscriber (`InsertManyTx`) |
| R43 | Alert escalation: push/SSE repeats, owner SMS 60 s, ops manual call 90 s; voice escalation P1 if > 5% reach 90 s |
| R44 | Device-bound restaurant sessions 30-day idle / 90-day absolute; riders 30-day sliding |
| R45 | Load model owned by doc 20: pilot 30/day … design 2,000/day, 500/h; test 1,500/h + 3,000 SSE |
| R46 | Cost owned by doc 25 §15: pilot ≈ ₹14–17k/month prod, launch ≈ ₹30k; PA effective rate ≤ 1% target |
| R47 | Gate A ≥ 10 restaurants, ≥ 10 riders, 1 zone; Gate B ≥ 40 restaurants, demand-sized riders; `MG_TOPUP` |
| R48 | Each tunable owned by one doc (13 timers, 10 seeds, 20 load, 25 cost, 16 fees) |

### Scope and release decisions

| ID | Decision | Source |
|---|---|---|
| SD-1 | Scope cuts C1–C20 deferred to V1.1+ (surge, WhatsApp, prerender, IAP, ClamAV/PDF KYC, hash-chained audit, extra maker-checker, partitioning/outbox table, custom tools, DR pre-provisioning/game days pre-Gate B, Sentry, rider shifts/`ON_BREAK`, restaurant trend analytics, review moderation/replies, zone GeoJSON/heat maps, coupon extras incl. bulk codes, Telugu romanisation, six-account org, static-QR COD-UPI, admin broadcasts/live map/tax-rule UI) | 00 §9, 31 §13.1 |
| SD-2 | Missing items M1–M17 added to the backlog (doc 27 Appendix E) | 00 §9, 31 §13.2 |
| SD-3 | Scope freeze at M4 exit; feature complete at M5 exit; copy freeze M5 week 4 | 28 §6 |
| SD-4 | AWS accounts opened on M6 day 1; IaC authored in M5 without apply | 28, UD-2 |
| SD-5 | Public launch placed after Diwali 2027 with a 3-week buffer after Gate B | 28 §3 |
| SD-6 | Go/No-Go rules: [NO WAIVER] rows, written waivers with expiry, founder final call | 29 §15 |
| SD-7 | Pilot SLO 99.5% measured during service hours (final public SLO pending OQ-30) | 29 OB-04, 31 RV-067 |

---

## (d) Unresolved decisions / open questions (deduplicated)

Sources: doc 01 §13 (OQ-01…13), doc 14 §21 (PAY-1…9), doc 15 §13 (N-1…8), docs 04–26 `[OPEN]`/`[LEGAL]` tags, doc 31 (§12 points not settled by §9 and register rows marked *(proposal)*). Items settled by R1–R48 are not repeated.

### Legal / tax / regulatory [LEGAL]

| ID | Question | Sources | Owner | Decide by |
|---|---|---|---|---|
| OQ-L1 | Money-flow model: split settlement vs collect-and-payout; restaurant share of COD cash | 01 OQ-02, 14 PAY-1, R25, R35 | LEGAL | **Phase-2 week 4 (M0)** |
| OQ-L2 | GST: delivery-fee characterisation (§9(5) vs own supply), fee-inclusive presentation on invoice, platform-funded discount valuation, SAC codes, invoice format/bilingual | 01 OQ-01, BR-FEE-007, 14 PAY-2/3, 10, 11, 16 | CA | M3 exit |
| OQ-L3 | TDS 194-O mapping under Income-tax Act 2025; rider TDS; TCS §52 confirmation | 01 LEG-TAX-001, LEG-GST-006, 14 PAY-4 | CA | M3 exit |
| OQ-L4 | Gig-worker obligations: Social Security Code thresholds, registration field list, Telangana act status, accident insurance (mandated?) | 01 OQ-11, LEG-GIG-001/002/004, 06, R47 | LEGAL | M4 exit |
| OQ-L5 | Police verification for riders (pre-approval vs 30 days); bicycles/low-speed EVs without DL | 01 OQ-05/06, 06 §2.2 | LEGAL + OPS | M4 exit |
| OQ-L6 | Retention schedule (8-year tax vs DPDP erasure, KYC images after 3 years, raw webhook bodies), erasure map sign-off | 01 NFR-RET, 10 §13, 19 §8.5, 23 §9, M15 | LEGAL | M5 mid |
| OQ-L7 | Rider recoveries for damage: allowed with acknowledgement? Rider fault policy wording ("HR policy" vs contractors) | 01 ADM-PAYO-006, 13 §6.2, 06 D8 | LEGAL | M4 exit |
| OQ-L8 | Children: is an 18+ declaration adequate given student users aged 17? | 01 LEG-DPDP-006, 19 §8.6 | LEGAL | M5 |
| OQ-L9 | Cancellation-fee policy compliance (CP E-commerce rules: symmetric charges) and COD compensation wording (R29) | 01 LEG-CP-004, 13 §6 | LEGAL | M4 exit |
| OQ-L10 | CERT-In reportability thresholds (e.g. DDoS) and DPB filing channel once operational | 19 §10.4–10.5 | LEGAL + SEC | M6 |
| OQ-L11 | Seller/menu data: real restaurant names/menus in staging dogfooding need written consent | 26 §9, 20 §17 | LEGAL + OPS | M6 |

### Product / business

| ID | Question | Sources | Owner | Decide by |
|---|---|---|---|---|
| OQ-P1 | Pilot rider minimum guarantee (`MG_TOPUP`) amount, slots, budget | R47, 31 RV-070 | FDR + FIN | M5 |
| OQ-P2 | Customer-fault refund at `PREPARING` (delivery fee refunded if rider assigned?); restaurant compensation (50% capped ₹300?) and whether commission is waived; penalty for restaurant-caused cancellations | 01 OQ-03, 13 §10 | PL + FIN | M2 |
| OQ-P3 | Commission basis includes packaging? Rider cancellation compensation (BR-RPAY-004 "base + waiting" vs doc 10/13 "50% of base") | 10 §open, 13 §10 | FIN | M3 |
| OQ-P4 | Rider first-mile pay and waiting-pay rate; waiting pay at drop for undeliverable | 01 OQ-10, 06, 13 | OPS + FIN | M7 (pilot data) |
| OQ-P5 | Payment retry semantics: retry on the same order vs new order; PA order expiry alignment with 15-min pending | 20 F-4d, 14 PAY-5, 04 | PL + BE | M4 start |
| OQ-P6 | Default customer locale for Mahabubnagar (te vs en) and which SMS need Telugu variants | 15 N-2 | PL | M5 |
| OQ-P7 | Variant default selection (cheapest vs none + required); "Egg" display rules | 04 §7.2, 05 | PL | M1 |
| OQ-P8 | Restaurant "menu editor" staff flag; can staff edit hours (05 vs 11) | 05 §2, 31 §14 | PL | M1 |
| OQ-P9 | Price-increase moderation threshold (30% vs 25%); line quantity bound (1–20 vs 1–50); prep-time range (5–90 vs 5–120); coupon budget approval threshold | 31 §14 rows 53, 55, 57, 58 | PL | M1 |
| OQ-P10 | Rider rating as thumbs + tags and M-41 redefinition (≥ 90% thumbs-up) | 31 §14 row 59 | PL | M3 |
| OQ-P11 | Cash ageing thresholds (48/72 h vs 24/48 h), rider payout day (Tuesday vs Wednesday), COD hours (off 23:00–06:00 vs service hours to 23:30) | 31 §14 row 72 | FIN + OPS | M3 |
| OQ-P12 | "New" rating badge threshold (< 5 per BR-RATE-002 vs < 20 in doc 04) | 01, 04 | PL | M1 |
| OQ-P13 | Urdu locale demand; late-night service hours | 01 OQ-09/12 | PL | M8 (post-pilot) |
| OQ-P14 | ONDC integration as strategic V2 | 02 §2.2 | FDR | Post-launch |
| OQ-P15 | On-demand rider payout limits (min ₹500, 1/day) | 06 | FIN | M4 |
| OQ-P16 | Women riders' "calls via support only" option; masked-calling upgrade trigger | 06 §9 | PL | M5 |
| OQ-P17 | Public status page; admin separate registrable domain; production domain name (`rovo.in` vs placeholder) | 01 NFR-AVAIL-009, 17, 22 | FDR + DO | M1 (domain) / M6 |
| OQ-P18 | Pilot order floor if volume < 300 orders in last 14 days | 29 §15 | LEAD | Gate A |

### Technical / operational

| ID | Question | Sources | Owner | Decide by |
|---|---|---|---|---|
| OQ-T1 | Final PA selection (written quotes; effective rate ≤ 1% target) and penny-drop provider | 14 PAY-7/9, R46 | FDR + FIN | M3 exit |
| OQ-T2 | SMS aggregator (MSG91 vs Gupshup), secondary provider, voice OTP need, WebOTP line across four hosts in DLT templates | 15 N-1/N-6/N-7, 31 RV-017 | PL + DO | M5 week 4 (copy freeze) |
| OQ-T3 | Bot-challenge provider (Turnstile vs WAF bot control) | 15 N-5, 12 | SEC | M1 |
| OQ-T4 | Lock-screen notification content (privacy) | 15 N-8 | PL + SEC | M2 |
| OQ-T5 | SSE stream max duration and `reauth` rule (30-min cap; reauth only on revocation?) | 31 §14 row 29, 08, 11, 22 | BE | M2 |
| OQ-T6 | Image pipeline: client-generated WebP + server JPEG re-encode (no cgo) vs libvips worker | 31 §14 minor, 08, 17 | BE + FE | M1 |
| OQ-T7 | Zone draw library (Terra Draw vs mapbox-gl-draw) | 16 §13, 17 | FE | M1 |
| OQ-T8 | Dependabot vs Renovate; release-please vs git-cliff; merge queue | 21, 26 | DO | M0 |
| OQ-T9 | DCO vs CLA; signed commits | 26 §12 | FDR | M0 |
| OQ-T10 | OTP rate-limit numbers (3/15 min vs 5/h, 10/day) — owner doc 12/13 per R48 | 31 §14 minor | SEC | M0 |
| OQ-T11 | Rider speed parameter (18 vs 20 km/h vs band speeds) — owner doc 13/16 | 31 §14 minor | BE | M1 |
| OQ-T12 | External pentest budget and vendor; AWS support plan level | 20 G12, 19 §10.5 | FDR | M5 |
| OQ-T13 | Mutation testing; visual regression blocking date; SSE soak harness (xk6-sse vs Go) | 20 §1, §2.2 | QA | M2 |
| OQ-T14 | Grafana IRM phone escalation availability; one incident channel (Telegram vs WhatsApp) | 24 §10, 31 RV-073 | RA | M5 |
| OQ-T15 | Counter-device supply model (rovo-provided on deposit vs restaurant-owned) and device budget | M3, 31 | OPS + FIN | M5 |
| OQ-T16 | Network egress filtering beyond SG/app allow-list | 19 SEC-176 | DO | M6 |
| OQ-T17 | Copy of weekly dump to a non-AWS store for org-level loss (DR-4) | 23 §8 | DO | M6 |

### Planning-level decisions for the Lead / user

| ID | Question | Owner | Decide by |
|---|---|---|---|
| OQ-30 | Public-launch SLO: keep NFR-AVAIL-001 99.9% 24×7 or adopt 99.5% service-hours (pilot) / 99.9% service-hours (launch) given a 3-person on-call | LEAD | M5 |
| OQ-31 | Is a ≈ 13-month timeline to public launch (Nov 2027) acceptable, or should the team grow to 4 engineers / cut further? | FDR | M0 exit |
| OQ-32 | Would the user allow staging on AWS before M6 (e.g. from M4) to de-risk cloud integration? (currently forbidden by R24 reading) | User via LEAD | M3 |
| OQ-33 | Committed P1 RES-ANLY-002 (restaurant 7/30-day trends) is removed by the C-list — confirm and update doc 02 §1.2 | PL + LEAD | M1 |

---

## (e) Remaining cross-document conflicts (for reconciliation by doc owners)

Conflicts already settled by rulings are listed in doc 31 §14 and must be edited into the source docs; the ones below are those the Release Architect found that are **still not explicitly settled**, or settled but with stale text that affects the backlog.

| ID | Conflict | Docs | Resolution used in 27–29 / needed |
|---|---|---|---|
| C-01 | Accept-timeout text (`REJECTED`/`RESTAURANT_TIMEOUT`, 240 s) still in 01 M-02/BR-TIME-002/RES-ORD-004, 08, 15, 20 F-2 | 01, 08, 15, 20 | Backlog uses R1; M-02 definition must be rewritten to count `RESTAURANT_UNRESPONSIVE` cancellations |
| C-02 | Fee slabs to 8 km (01 BR-FEE-001, 07, 10 seeds) vs R18 to 10 km | 01, 07, 10, 16 | R18 used; seeds owned by doc 10/16 (R48) |
| C-03 | Pilot supply targets: doc 02 Gate A ≥ 15/15 and doc 20 §17 5–10 restaurants vs R47 ≥ 10/10; Gate B 35 riders vs demand model | 02, 20 | R47 used |
| C-04 | Surge designed in 07, 10, 11, 16 vs R30 | 07, 10, 11, 16 | No surge in backlog; remove columns/endpoints |
| C-05 | `outbox_events` partitioned table (10 DB-D11) vs R22/R42 | 10 | No outbox table; retention jobs instead of partitioning |
| C-06 | Delivery OTP seed default off (10) vs R39 on | 10 | R39 used |
| C-07 | WhatsApp OTP as V1 fallback (15) vs V1.1 (12) vs P2 (01) vs cut (§9) | 12, 15, 01 | SMS only in V1 |
| C-08 | Restore-drill cadence: weekly (20 QG-8), 7-day gate (20 G5) vs monthly (23, SEC-123) | 20, 23 | Monthly alternating D1/D2 + one within 7 days of each gate |
| C-09 | Observability: Sentry in 17/24 vs R36 (Faro, no Sentry); CloudWatch 3-day vs 180-day archive | 17, 24, 22 | R36 used (E23-S03, E23-S06) |
| C-10 | Availability: NFR-AVAIL-001 99.9% 24×7 vs doc 24 SLO 99.5% | 01, 24 | Pilot 99.5% service hours; OQ-30 for launch |
| C-11 | Hosts `partner.` in 12, 24, 21 builds and doc 12 audience `partner` vs R14 four hosts | 12, 21, 24 | R14 used |
| C-12 | Paths `infra/`, `api/openapi/**`, `db/migrations` (21, 22, 23) vs `deploy/terraform`, `openapi/`, `backend/migrations` (26) | 21, 22, 23 | Doc 26 paths used |
| C-13 | Oracle/free-tier preview (21 `preview.yml`, 22 §11, 24 preview stack, 25 §9) vs R24 | 21, 22, 24, 25 | Local + card-free tunnel only |
| C-14 | Six-account AWS org (22, 19 SEC-172) vs four accounts (§9 cut) | 19, 22 | Four accounts |
| C-15 | DR pre-provisioning and quarterly game days (23) vs cut before Gate B | 23 | Cross-Region backups + IaC; first game day after launch |
| C-16 | Hash-chained audit + WORM anchors (19 SEC-127/128, 12) vs cut | 12, 19 | Append-only grants; SEC rows marked descoped |
| C-17 | KYC PDFs + ClamAV + envelope encryption (12, 19) vs R38 images only | 12, 19 | R38 used |
| C-18 | COD caps in doc 04 (₹1,500) and doc 14 (₹500 first order) vs R6 | 04, 14 | R6 used |
| C-19 | Restaurant self-cancel after accept (01 RES-ORD-006) vs R40 | 01 | R40 used |
| C-20 | SEC-038 heartbeat ≤ 15 s vs R10 20 s | 19 | 20 s used; SEC-038 to be edited |
| C-21 | Maker-checker lists in 07/10/11/12 (13 action types) and go-live gate "≥ 2 SUPER + ≥ 1 FINANCE" vs R31 | 07, 10, 11, 12 | R31 used |
| C-22 | Goodwill coupon cap ₹100 (01 ADM-TKT-002) vs approver threshold ₹150 (R31) | 01 | R31 threshold used; cap to be removed or confirmed (OQ-P9) |
| C-23 | Ticket statuses differ across 01/07/10 | 01, 07, 10 | Doc 10 set used |
| C-24 | Cost figures: BR-COST-002 ₹28–54k vs doc 22/25 vs R46 | 01, 22, 25 | R46 / doc 25 §15 used |
| C-25 | Load model: 300/h (01) vs 500/h (20) vs 3–5/min (08) | 01, 08, 25 | R45 used |
| C-26 | Doc 02 §1.2 still commits RES-ANLY-002 (cut) and ADM-COUP-004 is P1 stretch (cut) | 02 | Update doc 02 (OQ-33) |
| C-27 | Doc 13 folds offer revocation into `EXPIRED` vs R16 `REVOKED` | 13 | R16 used |
