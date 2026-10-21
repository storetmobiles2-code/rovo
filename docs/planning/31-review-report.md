# 31 — Review Report (Red Team)

| Field | Value |
|---|---|
| **Purpose** | Independent critical review of planning docs 01–26 against the baseline (`00`) and the Lead's binding rulings R1–R26. Finds errors, gaps, contradictions, over- and under-engineering. Lists cross-doc conflicts with the value the rulings require. Proposes scope cuts and missing items, and lists the points the Lead must decide. |
| **Owner** | Reviewer |
| **Status** | Draft v1 (2026-10-04) |
| **Dependencies** | `00-planning-baseline.md` (§2 vocabulary, §4a hosting directive, §5 commercial defaults, §8 rulings R1–R26); docs `01`–`26` as of 2026-10-04 (docs 27–30 not yet written, so not reviewed). |
| **Feeds** | `32-final-plan-summary.md` (Lead), `30-risks-assumptions-decisions.md` (Release), reconciliation edits to `01`–`26` by their owners. |

**How to read this.** Finding IDs are `RV-001`…`RV-0xx`. Severity:
- **Blocker**: will cause a wrong build, a compliance failure, or an outage-class defect if it is not fixed before Phase 2 starts.
- **Major**: material cost, risk or rework. Fix during reconciliation.
- **Minor**: a consistency or clarity fix.

Evidence uses `doc §section` (and line-level quotes where useful). "Owner" is the architect who should change the doc. Numbers marked `[ASSUMPTION]` are the Reviewer's own estimates.

---

## 0. Executive summary: top 10 issues

| # | Issue | Sev | Where | One-line fix |
|---|---|---|---|---|
| 1 | **The API edge topology contradicts R14 and breaks the auth design.** Doc 22 serves `api.` from the ALB directly (to stay under the CloudFront Pro 10M-request allowance) and has no `/api/*` behaviour on the app distributions. Doc 24 then asks for CORS on `api.`. Docs 12, 11 and 17 assume same-origin `/api/*` through the CDN, host-only cookies, **no CORS**, and an edge-injected `X-Rovo-Audience` header that the ALB path cannot set. | Blocker | 22 §3.1; 24 §2.2; 12 AUTH-D03/§4.1; 11 §1.1; R14 | Route `/api/*` on every app host through CloudFront (same-origin). Derive the audience from `Host` plus the origin-verify secret, never from a client header. Re-cost the CDN (RV-015). |
| 2 | **The restaurant accept-timeout ladder is still wrong in six docs.** They say `REJECTED`/`RESTAURANT_TIMEOUT`, a 240 s or 6 min timeout, ops at 120 s or 3 min, and owner SMS at 90 s. R1 says 180 s → `CANCELLED`/`SYSTEM`/`RESTAURANT_UNRESPONSIVE`, owner at 60 s, ops at 90 s. | Blocker | 01 M-02/RES-ORD-004/BR-TIME-002; 04 §15 E6; 05 §4.3/§11; 08 §3.2/§5.3/§11/§14; 15 §7 | Apply R1 verbatim. Only doc 13 is correct today. |
| 3 | **CERT-In log retention is not implemented.** CERT-In requires 180 days of logs for all ICT systems, kept in India. Doc 19 SEC-134 also asks for ≥ 1 year of security logs. Doc 22/24 keep CloudWatch for **3 days** and Loki for **14 days**, and VPC flow logs hold rejects only. | Blocker | 22 §5/§6.3; 24 §1/§3; 19 §8.10; 01 NFR-RET | Add an India-region 180-day (security events 1-year) log archive: CloudWatch Logs infrequent-access class or S3 with Object Lock. Cost it. |
| 4 | **Fee slabs and the radius are defined inconsistently, so some orders cannot be priced.** The doc 10 default/seed slabs stop at 8,000 m road distance and the cap is `max_serviceable_distance_m=8000`. Doc 16 applies the 7 km radius to *road* distance with upper-inclusive slabs. R18 says a 7 km straight-line radius (≈ 9.1 km road), slabs to 10 km, lower bound inclusive. As written, a 7 km straight-line order has no slab. | Blocker | 10 §5.2/§15.1; 16 GEO-D02/§6.1/§4.2; 01 BR-FEE-001; 07 §9; R18 | Re-seed slabs to `[2000,4000,6000,8000,10000]` with ₹20/30/40/50/60, `[lo,hi)` semantics, straight-line radius check. Add golden tests at 1,999/2,000/9,100 m. |
| 5 | **Production cost is sized for 500–2,000 orders/day, but the plan expects 80–250.** At the recommended ₹34.2k/month (prod + staging, ex-GST), cloud cost per order is about ₹38 in the closed pilot, ₹14 in month 1 and ₹4.6 in month 3. That misses M-60's ≤ ₹6 until roughly month 3. Several cost lines are missing: ClamAV, an IAP for admin, DR pre-provisioning, the 180-day log archive, devices. | Major | 25 §10/§15; 22 §13/§15; 01 M-50/M-60/BR-COST-002 | Stage the infrastructure: a "closed-pilot" profile (single-AZ t4g.small, ≈ ₹15k) and a public-launch profile set by the load test (§2). |
| 6 | **Rider supply economics and dispatch eligibility are not viable.** Gate B wants ≥ 35 riders, but month-1 volume is ≈ 80 orders/day. That is about 2.3 trips per rider per day, or ≈ ₹80/day, against the persona's ₹18–22k/month. Separately, the staleness rules (90 s–3 min) exclude exactly the backgrounded riders that Web Push is supposed to reach. | Major | 02 §5.2 B5/B6; 03 P7; 01 M-54; 06 §4; 13 T-RIDER-STALE; 16 §7.2; 18 §8 | Add a pilot minimum-guarantee per peak slot. Size the fleet to demand. Make "online but stale ≤ 15 min" riders a second dispatch tier reached by push. |
| 7 | **Restaurant alert reliability depends on a foregrounded Chrome tab on a shared, cheap phone.** Web Push cannot loop a sound. The fallback is SMS at 60 s (easy to miss in a noisy kitchen). Devices are not budgeted. The counter device session expires after 7 days idle / 30 days absolute. | Major | 05 §1/§4.1/§9 R1; 18 §7; 12 §3.4; 15 §7 | Add an automated **voice-call** escalation at 60 s. Budget pre-configured counter devices. Give registered order-receiver devices a device-bound session that never expires mid-service. |
| 8 | **The design is over-engineered for a 2–3 developer team.** It has about 200 API operations, 81 tables, maker-checker on 13 action types with a "≥ 2 SUPER + ≥ 1 FINANCE" go-live gate, a per-row hash-chained audit log, a ClamAV service, an IAP option, build-time prerendering on a release-candidate framework, quarterly region game days, five observability tools, day-one partitioning, and two custom CI tools. | Major | 11 §2.9; 10 §1.5; 12 §5.5; 19 §6.10; 17 F1; 23 §7; 24 §1 | Apply the scope cut list (§13). |
| 9 | **The authoritative schema (doc 10) is missing tables that the rulings and other docs depend on, and has drifted on names.** Missing: `rate_limit_buckets` (R21), `transfers`/`pa_settlements`/`recon_exceptions` (14, R25), the erasure ledger (23 §9), gig-worker export (LEG-GIG-001), leads/waitlist, SOS events, call-tap log. `delivery_offers` has no `REVOKED` (R16). Ledger account codes differ between 10 and 14. Ticket statuses exist in three versions. | Major | 10 §1.5/§8.2/§9; 14 §5/§10.2; 12 §2.2; 13 §4.1; 01/07/10 tickets | Backend owner reconciles the schema against the §14 register. Add the missing tables. |
| 10 | **Scope creep and policy drift contradict the PRD.** Manual surge pricing is designed in V1 (16 GEO-D05, 11 surge endpoints, 10 surge columns). WhatsApp OTP is a V1 fallback (15). Server-cart remnants remain (01, 04, 08). "COD orders never get money refunds" (10 §7) contradicts BR-REF-004 and is a consumer-law risk. | Major | 16 §6.2; 11 §2.6; 15 §5.2; 01 BR-FEE-006/BR-REF-004/CUS-CART-002; 10 §7; 02 §2.2 | Cut surge and WhatsApp OTP from V1. Remove server-cart text. Lead decides the COD refund policy (D3). |

**Counts:** 86 findings (RV-001…RV-086) — **5 Blocker** (RV-001, RV-025, RV-083, RV-084, RV-085), **49 Major**, **32 Minor**. The cross-doc inconsistency register (§14) has 72 rows plus 11 minor drift notes. The Lead must decide 16 contested points (§12). Top-10 issues map to: #1 → RV-001/025, #2 → RV-083, #3 → RV-085, #4 → RV-084, #5 → RV-012–014, #6 → RV-055/070, #7 → RV-053/054, #8 → §13.1, #9 → RV-043/045, #10 → RV-047/078/079.

---

## 1. Architecture choices (over- or under-engineered?)

The core choices hold up for this team and city: a Go modular monolith, Postgres + PostGIS, River, SSE, a static SPA plus CDN, and ECS Fargate. The problems are in the **ceremony around** those choices, and in a few under-engineered money and real-time paths.

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-001 | Blocker | API edge topology violates R14 (see top-10 #1). Doc 22 puts `app./partner./admin.` on CloudFront with only S3 origins, and `api.` on the ALB because "API volume (≈ 40M req/mo) would exceed the flat-rate Pro request allowance". Doc 24 configures Sentry `tracePropagationTargets: api.` and "API CORS must allow traceparent…". Doc 12 forbids cookies on `api.` and relies on the CDN overwriting `X-Rovo-Audience`. An ALB cannot inject arbitrary request headers, so the API either trusts a client-settable header or breaks. | 22 §3.1, §3 diagram; 24 §2.2; 12 §4.1 ("CDN or LB adds `X-Rovo-Audience`"); 11 §1.1; R14 | Every app host gets a CloudFront behaviour for `/api/*` (caching off, all cookies forwarded) to the ALB. The ALB accepts only the CloudFront prefix list plus a secret origin header. The API derives the audience from `Host` and **ignores** `X-Rovo-Audience` unless the origin secret is present. `api.` stays reserved for future native clients and PA webhooks only. Remove CORS from doc 24. | DevOps + Security |
| RV-002 | Major | **The event pipeline has one hop too many and two stores for one fact.** `events.Record` writes `event_log`/`outbox_events` **and** a single `event.fanout` job, which then inserts one job per subscriber. Online-order placement becomes webhook job → fanout → ordering handler → `OrderPlaced` → fanout → notifications job → push: about 4 River fetch cycles before the restaurant rings, against a 5 s p95 target (NFR-PERF-004). It also adds a monthly-partitioned table plus a 90-day archive job. | 08 §7.3; 09 ADR-006; 10 §14; 01 NFR-PERF-004 | Insert subscriber jobs directly with `InsertManyTx` from the static subscription table at emit time (River is still the outbox per R22). Drop `outbox_events` (log the event in job args, plus `processed_events`). Add a latency budget test: commit → restaurant SSE ≤ 2 s p95, commit → push sent ≤ 3 s p95. | Solution Architect |
| RV-003 | Major | **`pg_notify` inside the business transaction is a hidden availability risk.** If any LISTEN backend stalls (a connection that is open but not reading), the async notification queue fills (8 GB default). After that, *every* transaction that calls `pg_notify` fails, including order placement. Doc 08 treats NOTIFY as free. | 08 §6.2 ("`SELECT pg_notify` inside same tx") | Alert on `pg_notification_queue_usage() > 0.1`. Keep LISTEN connections read-only with a watchdog that reconnects when lag grows. Optionally move NOTIFY to a post-commit step (lose-able hint) so business commits never depend on it. | Solution Architect |
| RV-004 | Major | **River OSS periodic jobs are not durable** (doc 10's own source notes "durable periodic jobs" are a River Pro feature). The weekly settlement run (Mon 06:00), daily PA recon (10:30), retention sweeps and partition maintenance all rely on leader-elected periodic jobs. A leader restart at the tick skips that run silently. | 10 §20 sources; 14 §16.1/§17; 10 §14 | Make every periodic job a **catch-up** job: it runs hourly, checks "has the run for period P completed?", and runs if not. Alert on "no settlement run for last week by Mon 09:00". | Backend Architect |
| RV-005 | Major | **R19 (no SQL `now()` in business logic) is contradicted in four places.** (a) Doc 08 §11: "All timers are DB-side (`now()`)". (b) Doc 14 §18: "Our timers use DB time". (c) Doc 16 §7.2: candidate search `last_location_at > now() - interval '3 minutes'`. (d) Doc 10 payout cut-off is "postings `created_at < period_end`", where `created_at DEFAULT now()`. River itself picks scheduled jobs by DB time, so E2E fake-clock tests need the doc 20 §10.2 spike. | 08 §11; 14 §18; 16 §7.2; 10 §9 payouts; 20 §10.2; R19 | Pass `:now` from the injected clock into every business query. Base payout cut-off on `ledger_journals.accounting_date`/`occurred_at`, set from the app clock. Run the River fake-clock spike in week 1 with an explicit fallback (short real timeouts in the E2E profile). | Backend + QA |
| RV-006 | Major | **The no-cross-module-FK policy (only `cities`/`users`) gives up integrity on money paths for a speculative extraction.** `payments.order_id`, `refunds.order_id`, `deliveries.order_id`, `ledger_postings.order_id`, `coupon_redemptions.order_id` and `invoices.order_id` have no FKs, and a nightly orphan check is the backstop. No microservice extraction is planned (08 §9.5, 02 §2.1). | 10 DB-D04; 08 §4.1 rule 1 | Allow FKs inside the single database for `order_id` references from payments, refunds, deliveries and invoices. Keep the import-boundary rules in Go. If a module is ever extracted, dropping an FK is a one-line migration. | Backend + Solution |
| RV-007 | Minor | **Two custom CI tools** (`tools/tableowner`, `tools/eventschema`) on top of go-arch-lint, sqlc vet and oasdiff. This is real build effort for a 2–3 person team before any feature ships. | 08 §4.3; 26 §1.5 | V1 keeps go-arch-lint/depguard and a `table_ownership.yaml` review checklist. Build `tableowner` only after the first cross-module SQL incident. Replace `eventschema` with JSON fixtures per event (already in 20 §6.7). | Solution Architect |
| RV-008 | Minor | **REST + OpenAPI 3.1 spec-first is right, but about 200 operations** (40% admin) is a lot to spec, generate, authz-matrix-test and keep drift-free. | 11 §2.9 | Freeze the V1 spec at the golden-flow and ops-critical endpoints (≈ 110). Admin "nice-to-have" endpoints (zone import/export, broadcasts, `geo/live`, tax-rule CRUD, surge) move to V1.1. | Backend Architect |
| RV-009 | Minor | **Four apps (R14) are accepted**, but the cost is understated ("one extra Vite build"). Each app host is also one CDN distribution with `/api/*` routing, a WAF association, an S3 bucket, a manifest, a service-worker update flow, Lighthouse budgets and a Playwright profile. Docs 21/22/24 still budget three. | 17 §1.4; 22 §6.3 (3 web buckets); 21 §3.3 | Keep four apps. Update 21/22/24/26 to four hosts and buckets (§14). Put all four hosts on **one** distribution with host-based behaviours if the flat-rate plan is per distribution (22 §3.1 [OPEN]). | DevOps |
| RV-010 | Major | **Under-engineered: the PA split-settlement path (preferred in ADR-012) has no data model and no ops plan.** Doc 14 relies on `transfers`, `ReleaseHold` and `ReverseTransfer`. Doc 10 has no `transfers` table. Every restaurant needs PA linked-account KYC ("1–3 days per vendor"), on top of the 40 restaurants needed for Gate B. COD-heavy restaurant shares are still paid from rovo's own account, so the "Model B" legal question exists either way. | 09 ADR-012; 14 §3.1/§4/§17; 10 (absent); R25 | Add `transfers` (and `pa_settlements`, `pa_settlement_lines`, `recon_exceptions`) to doc 10 now. Put "PA linked-account KYC per restaurant" in the onboarding checklist (05 §3.5) and the critical path (28). Get the legal opinion (PAY-1) before Phase-2 week 4, because it changes payout code. | Solution + Backend + Release |
| RV-011 | Minor | **SSE stream lifetime is set three different ways**: 30 min server close (08 §6.2), 55 min per-stream deadline (22 §3.3), and `reauth` at access-token expiry, which is every **10 min** (11 §4.1, 12 §3.4). Every restaurant tablet therefore reconnects and refetches every 10 min. | 08 §6.2; 22 §3.3; 11 §4.1; 12 §3.4 | Keep one rule: the stream continues past JWT expiry while the server-side session is valid (partner sessions are checked per request anyway). Close at 30 min for rebalancing. | Solution + Security |

---

## 2. Cost assumptions and unit economics

### 2.1 Per-order unit economics (Reviewer model, `[ASSUMPTION]` inputs)

Inputs:
- Average item total ₹300 (14 §3), packaging ₹10, average road distance 3.3 km, online share 70%.
- PA 1.95% (Cashfree list price; Razorpay 2%).
- Launch promotions average ₹10/order (platform-funded); refund/goodwill leakage ₹3/order.
- Fees GST-inclusive per R8/BR-FEE-007. GST on cloud and PA fees treated as ITC-recoverable, so ex-GST amounts are used.

| Line (per delivered order) | ₹ |
|---|---|
| Commission 15% × ₹300 | +45.00 |
| Delivery fee ₹30 incl. 18% GST → net | +25.42 |
| Platform fee ₹5 incl. GST → net | +4.24 |
| **Platform revenue (ex-GST)** | **+74.66** |
| Rider pay: ₹25 + ₹6 × 1.3 km + avg wait ₹2 | −34.80 |
| PA fee: 70% × 1.95% × ₹361 payable | −4.93 |
| Platform-funded promotions (launch average) | −10.00 |
| Refund/goodwill leakage | −3.00 |
| **Contribution before tech** | **≈ +21.9** |
| SMS/OTP (15 §9: ≈ ₹1.4k / 7,500 orders) | −0.19 |

Cloud cost per order (doc 22 §15 recommended: prod ₹30.6k + staging ₹3.8k = **₹34.2k ex-GST/month**):

| Phase (orders/day → /month) | Cloud ₹/order | Contribution after cloud + SMS |
|---|---|---|
| Closed pilot (30/day → 900) | **38.0** | **−16.3** |
| Month 1 (80/day → 2,400) — M-50 | **14.3** | **+7.4** |
| Month 3 (250/day → 7,500) — M-50 | **4.6** (M-60 target ≤ 6 met) | +17.1 |
| 1,000/day (30,000) | 1.1 (M-60 ≤ 3 met) | +20.6 |

Interpretation:
- The tech stack is affordable from month 3 onward. In the pilot it costs more per order than the platform fee and delivery margin earn.
- The cost model sizes infrastructure for **500–2,000 orders/day** (25 §10.1, 08 §9.2). The PRD's own targets are 80 → 250/day (01 M-50).
- People cost dominates anyway. Four ops/support staff at about ₹18k each is about ₹72k/month `[ASSUMPTION]`, or ₹9.6/order at month 3.

### 2.2 Findings

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-012 | Major | **Infrastructure is sized for the wrong phase.** Doc 25's pilot load model (500–2,000 orders/day, 20,000 sessions/day, ≈ 8% of the city's population daily) drives Multi-AZ db.t4g.medium, 2 API + 2 worker + an OTel gateway, and AWS WAF on the ALB, all from day one. | 25 §10.1, §15.1; 22 §13; 01 M-50 | Use two IaC profiles. **closed-pilot**: RDS db.t4g.small **Single-AZ** with PITR and cross-region backups (allowed by NFR-AVAIL-003), 2 api at 0.25 vCPU/0.5 GB, 1 worker, no Alloy gateway (export OTLP directly), and the CloudFront-included WAF only. Reviewer estimate ≈ $150–165/month ≈ ₹14–16k `[ASSUMPTION]`. **public-launch**: Multi-AZ, with the size set by the K-2/K-6 load test. | DevOps |
| RV-013 | Major | **The cloud budget has three different numbers.** BR-COST-002 says ₹28k–54k/month. Docs 22/25 say ₹16.9k–30.6k prod + ₹3.8–6.1k staging. Doc 22 §15 sets the AWS Budget at ₹40k "incl. GST". M-60 is defined to include staging. | 01 BR-COST-002, M-60; 22 §15; 25 §0 | Single source: doc 25 §15 owns the numbers, doc 01 references them. State clearly whether figures are ex-GST (ITC-recoverable) or incl. GST. | Product + DevOps |
| RV-014 | Major | **Missing cost lines.** None of these is in docs 22/25 §15: ClamAV service (2 GB task; 19 §6.10); an identity-aware proxy for admin (12 §3.5 option A, AWS Verified Access — Reviewer believes about $0.27/app-hour ≈ ₹18k/month, **unverified**); the CERT-In 180-day log archive (RV-085); DR pre-provisioning (ECR replication, Secrets Manager replicas, multi-region KMS keys; 23 §2); GuardDuty and Config (22 §16 "UNVERIFIED"); public IPv4 for ≈ 7 addresses (≈ $25/month); DLT ₹5,900; Google Play developer account (TWA, 18 §10.1); counter devices and a device lab (RV-062); support telephony number. | 22 §15; 19 §6.10; 12 §3.5; 23 §2; 18 §10.1/§12 | Add every line to doc 25 §15, or cut the item (§13). | DevOps |
| RV-015 | Major | **The CloudFront flat-rate assumption needs care.** Verified on 2026-10-04: Pro is $15/month for 10M requests and 50 TB, with **no overage charges**. A first spike up to 3× the allowance is accommodated, after which "excess usage is evaluated over multiple months" and performance may be reduced or an upgrade required. Business is $200/month for 125M requests. Flat-rate plans "may not be combined with any other offers, promotions, or discounts", so AWS Activate credits (25 §0) may not offset them. Same-origin `/api/*` (RV-001) puts API traffic on the plan. Reviewer estimate for month 3: ≈ 6–12M requests/month (sessions + rider pings + 30 s restaurant heartbeats + polling) — borderline Pro. | 17 §1.2; 22 §3.1; 25 §0 [C12]; sources §15 | Before launch, model requests per month from the K-1/K-2 mix. Reduce chatty traffic: batch rider pings (11 already allows 1–10 points), heartbeat restaurants every 60 s and count SSE presence. Fallback: CloudFront pay-as-you-go for the app distributions, not Business. Do **not** solve it by moving the API off the CDN (RV-001). | DevOps |
| RV-016 | Major | **PA fees are the largest variable tech cost, and the "UPI first" strategy makes them worse.** At 2% + GST on a ₹361 basket, the fee is ≈ ₹7.2, which exceeds the ₹5 platform fee. The cost is noted (09 ADR-012, 14 §3) but no customer-side lever is planned. | 14 §3; 09 ADR-012; 01 BR-COST-004 | (a) Make the written PA rate a go/no-go criterion (target ≤ 1%). (b) Use the Cashfree ₹20 lakh 0% allowance for the pilot if Cashfree wins. (c) Product decides whether the platform fee stays at ₹5 once the PA rate is known (BR-COST-004 freezes it). | Product + Lead |
| RV-017 | Minor | **SMS/WhatsApp costs are realistic** (₹1.3–3k/month) but carry hidden fixed items: DLT scrubbing fees "unverified", a voice-OTP option, and Meta business verification. The WebOTP per-host line ("`@app.` vs `@partner.`") now needs **four** hosts (R14), which means either a variable in the DLT template (15 N-6 [OPEN]) or four templates. | 15 §5.2, §9; 12 §2.2 | Settle the DLT template design in week 1 (template approval takes 1–7 days). Budget ₹3k/month plus a one-time ₹5.9k. | Solution Architect |
| RV-018 | Minor | **The 10× cost (₹1.34 lakh/month at 5–20k orders/day) is fine per order**, but it assumes Grafana stays cheap at 10× (24 §11 "tighten to 5%"). | 25 §15.2 | Leave as is. Re-check at S2 of the scaling path. | DevOps |

---

## 3. Free-tier and development-workflow dependencies (R24: local Docker only, no card-requiring tiers)

**What must work fully offline with fakes:**
- OTP (fake sink)
- PA checkout, webhooks, refunds and settlement report (fakepay)
- Web Push (log sink)
- SMS/WhatsApp
- email (Mailpit)
- object storage (MinIO)
- map tiles (local PMTiles or stub)
- bot challenge (fake verifier)
- telemetry (local LGTM)
- clock (`testhooks`)
- seed data for one city, 8 restaurants and 4 riders

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-019 | Major | **Stale free-tier and preview text contradicts R24.** Doc 02 §4: "dev/preview may use free tiers". Doc 22 §1 preview row: "Stakeholders via Cloudflare Access", "Oracle option"; §7 "Preview: .env on the VM"; §11 full Oracle VM hardening. Doc 21 §7.5 `preview.yml` deploys to the Oracle VM with `wrangler`. Doc 23 §10 "Preview (Oracle free)… R2". Doc 24 §1 preview on Grafana Cloud Free + Sentry. Doc 08 §9.1 "free-tier dev/preview environments are allowed for demos". | 02 §4; 22 §1/§7/§11; 21 §7.5; 23 §10; 24 §1; 08 §9.1; R24 | Delete these or mark them historical. The only demo path is the local Compose stack plus a card-free quick tunnel. Remove the `preview.yml` workflow from 21. | DevOps + Product |
| RV-020 | Major | **Turnstile breaks the offline dev loop.** Cloudflare's always-pass test keys still load the widget script from Cloudflare, so local dev without internet cannot log in. | 12 AUTH-D07; 20 §10.1 | Ship a `BotChallenge=fake` adapter (accepts any token) for local, CI and E2E. Use real Turnstile test keys only in a staging smoke test. | Security + Frontend |
| RV-021 | Minor | **Map tiles in local dev depend on OpenFreeMap.** CI has a "local map tiles stub" (20 §10.1); local Compose does not. | 08 §2.3; 15/16 maps; 20 §10.1 | Add a small Mahabubnagar PMTiles extract (or a blank-style stub) served by MinIO in Compose. Make `tileStyleUrl` configurable (11 §2.1 already exposes it). | Frontend + DevOps |
| RV-022 | Minor | **The PA sandbox in local dev** (26 §8, 08 §2.3) needs a PA account. Test mode is normally available on sign-up before KYC `[ASSUMPTION — verify per PA]`, but the account needs a legal entity. Tunnels: `ngrok` requires an account; a Cloudflare quick tunnel does not. | 26 §8, §1.5 `tunnel.sh` | Keep the PA sandbox optional and staging-only. Make `cloudflared` quick tunnel the documented default. The fakepay contract suite must cover every scenario in 14 §20, so nobody needs the sandbox to develop. | DevOps + Solution |
| RV-023 | Minor | **The local web dev servers don't agree.** Doc 22 §10 shows 3 Vite servers (customer, partner, admin). Doc 08 §2.3 and doc 26 §8 show 4 (5173–5176). Local Postgres image: 22 uses a custom `rovo-postgres:17-3.5`; 20 §10.1 says "Postgres 18". | 22 §10; 08 §2.3; 26 §8; 20 §10.1 | Four dev servers. PG **17** + PostGIS 3.5 everywhere (R22). | DevOps + QA |
| RV-024 | Minor | **The WhatsApp Cloud API test number** (26 §8) and Grafana Cloud/Sentry projects for dev are optional SaaS sign-ups. They are not card-requiring, but they are not offline either. | 26 §8; 24 §1 | State explicitly: optional, never required for the golden flow. | DevOps |

---

## 4. Security gaps

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-025 | Blocker | **Audience/header trust when the CDN is bypassed** (follows RV-001). The design makes the server derive the audience from `X-Rovo-Audience` "set by the edge". Any request path that reaches the ALB without CloudFront (doc 22's `api.` host today, or a misconfigured security group) can forge it and reach `partner`/`admin` cookie semantics. | 12 §4.1; 11 §1.1; 22 §3.1 | The audience comes from the `Host` header **and** requires the origin-verify secret header. Reject any request that has `X-Rovo-Audience` but no secret. Add an authz test. | Security |
| RV-085 | Blocker | **CERT-In / DPDP log retention is not implemented** (top-10 #3). CERT-In requires ICT-system logs for a rolling 180 days within India (verified, §15). Doc 19 SEC-134 asks for app, LB/edge, WAF, VPC-flow, DB and cloud-audit logs ≥ 180 days (security events ≥ 1 year). Doc 22 keeps CloudWatch for 3 days, ALB/CloudFront/flow logs for 90 days (flow logs rejects only). Doc 24 keeps Loki 14 days and says "never rely on Loki for compliance". Nothing in the design holds 180 days. | 22 §5, §6.3 `rovo-prod-logs` 90 d; 24 §1, §3 (line "14 days on Grafana Free"); 19 §8.10, SEC-134; 01 NFR-RET | Add a `log-archive` pipeline: app JSON logs, ALB/CloudFront/WAF logs, VPC flow (all, not rejects only), RDS logs and CloudTrail → S3 in `ap-south-1` with lifecycle 400 days (Object Lock for the security subset), or CloudWatch Logs infrequent-access class. Cost it in 25 (≈ 25–40 GB/month at month 3 `[ASSUMPTION]`). Make it a Gate A item in 29. | DevOps + Security |
| RV-026 | Major | **The no-NAT, public-subnet decision (22 §4.3) contradicts the threat model.** Doc 19 §7.2 has workloads in private subnets with no public IPs, "public subnets hold only the LB and NAT", plus a DNS/egress allowlist (TB2→TB5). Doc 22 runs tasks with public IPv4 and unrestricted egress to save ≈ $41/AZ/month. Neither doc acknowledges the other. | 22 §4.1/§4.3; 19 §4.1 TB2→TB5, §7.2 | Lead decision D2. Minimum if accepted for the closed pilot: security groups with no inbound except from the ALB; egress limited to 443 and 5432 (internal); an app-level outbound host allowlist; GuardDuty Runtime Monitoring on ECS; doc 19 amended with an explicit time limit (move before Gate B or when a provider requires IP allowlisting). | DevOps + Security |
| RV-027 | Major | **Admin phishing resistance is planned too late, and the admin gate is unrealistic for this team.** TOTP can be phished, and doc 19 ranks T-08 admin phishing #8. WebAuthn is V1.1. The "recommended" IAP (Verified Access / IAP / Cloudflare Access) is not costed or deployed (22). An IP allowlist does not suit ops staff working from phones on dynamic mobile IPs (07 §1). | 12 AUTH-D08/§3.5; 19 §11; 07 §1; 22 | Bring **passkeys (WebAuthn) into V1** for `ADMIN_SUPER`/`ADMIN_FINANCE` (any admin with step-up for money). No IAP in V1. WAF rate rules on `/auth/admin/*` plus geo-IN on the admin host. | Security |
| RV-028 | Major | **The hash-chained audit log serialises all audited writes.** `hash = SHA-256(prev_hash ‖ row)` needs a global ordering lock in the insert transaction. Doc 12 also audits `auth.otp.requested`/`verified`, so OTP traffic contends on one row. | 12 §5.7; 10 §1.5 `audit_logs` partitioned | V1: append-only grants and triggers (already planned) are enough. If tamper evidence is required, seal in batches: an hourly job computes a Merkle or chain hash over the new rows and writes the anchor to the WORM bucket. Keep auth-event telemetry out of the business audit table. | Security + Backend |
| RV-029 | Major | **The KYC malware pipeline is heavy.** ClamAV `clamd` (≈ 2 GB task, freshclam egress, which fails without NAT/egress per RV-026) plus PDF parsing in a subprocess plus app-level envelope encryption on top of SSE-KMS. ClamAV is not in doc 22's topology or cost. | 19 §6.10; 12 §6.2; 22 §5 | V1: accept JPEG/PNG only for KYC, and convert PDF uploads to images in the browser (or rasterise server-side with a resource-limited library), so admins never open a PDF. Drop ClamAV. Keep bucket SSE-KMS with access audit. Decide whether **app-layer** encryption of KYC images is worth the KMS calls and complexity (Lead D12). | Security + DevOps |
| RV-030 | Major | **Raw PA webhook payloads are kept for 8 years with customer contact data.** `payment_events.payload` (raw JSON, 8 years) includes the PA's customer email/contact and masked VPA. Doc 14 says 180 days for the same raw body. Under DPDP minimisation, PII should not ride along with the financial-retention class. | 10 §7, §12 (8-year retention), §13; 14 §5 (`raw_body` 180 d) | Redact contact/email fields at ingest. Keep the full raw body 180 days for disputes; keep the normalised fields 8 years. | Backend + Security |
| RV-031 | Major | **DPDP erasure coverage is incomplete.** The erasure ledger (23 §9) has no table in doc 10. Free-text PII in `ticket_messages`, `customer_note`, `delivery_instructions`, `outbox_events`/job args, audit "before/after" diffs, the exports bucket and `notification_deliveries` has no erasure rule. | 23 §9; 10 §13; 12 §5.7 | Add `erasure_requests` (or an erasure ledger) table. Add an "erasure map" per table/bucket with action = delete / anonymise / retain (legal basis). Run a restore drill that replays erasures (23 D1). | Backend + Security |
| RV-032 | Major | **Cross-border processors.** Sentry SaaS stores data in US/EU regions (no India region `[ASSUMPTION — verify]`). Grafana Cloud's `ap-south-1` stack is claimed (24 §1 [S73]). Customer apps send browser errors with URL paths that contain order IDs. | 24 §1, §2.2; 19 §8.9 T-91 | Keep Sentry only with strict scrubbing (no breadcrumbs carrying URLs with IDs, IP collection off). Alternatively send frontend errors to Grafana Faro (India stack) and drop Sentry (D10). Record the processor in the DPA register. | Security + DevOps |
| RV-033 | Minor | **Supply chain.** cosign signing and provenance are "optional → recommended". GitHub Actions are not pinned by SHA. `pull_request_target` is proposed for infra plans on forks (21 §6), which is a classic token-exfiltration path on a public repo. | 21 §4, §6 | Make signature verification mandatory for production deploys. Pin third-party actions by commit SHA (Dependabot updates them). No `pull_request_target` workflows in V1. | DevOps |
| RV-034 | Minor | **OTP abuse controls are thorough but over-layered for V1** (12 per-key rules plus Turnstile on every request plus device cookies plus /24 limits plus a global breaker). Turnstile on *every* OTP request adds a third-party script and latency to the most important conversion step on low-end phones. | 12 §2.2; 11 §1.10 | Run Turnstile invisibly and only at risk (soft IP limit exceeded, new device plus high velocity, or global strict mode). Keep the per-phone limits and the global SMS budget breaker. | Security |
| RV-035 | Minor | **The rider-to-customer phone reveal is weaker than stated.** A `tel:` link reveals the number in the dialler and call log. The policy window also differs: 01 (from `PICKED_UP`, +15 min) vs 06 (from `ASSIGNED`, +60 min). | 01 BR-CONT-001; 06 §9 | Adopt doc 01's window. State plainly that the number is disclosed. Masked calling stays the V1.1 trigger (02 §2.2). | UX + Security |
| RV-036 | Minor | **CERT-In clock sync and incident response.** The PoC registration, 6 h reporting and tabletop are well covered (19 §8.10, §10). But the incident commander is "on-call engineer → Security lead", and the team has no security lead. | 19 §10.1 | Name roles by person in 29. One founder is the CERT-In PoC. | Release Architect |

---

## 5. Scalability and performance risks

The V1 volumes are small: ≤ 300 orders/hour peak, under 1,000 SSE connections, under 10 River jobs/s. Throughput is not the risk. Correctness of the real-time paths and failure behaviour are.

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-037 | Major | **The real-time latency budget isn't allocated across the async chain** (RV-002). River's job fetch interval, the fanout hop, notification job scheduling, and Web Push via FCM (Doze-delayed on Xiaomi/Vivo/Oppo, 18 §6.3) all stack up. The NFR is ≤ 5 s restaurant and ≤ 3 s rider offer. | 01 NFR-PERF-004; 08 §5.1; 18 §6.3 | Allocate the budget per hop. Use River's LISTEN-based fast fetch and a short fetch poll for the `realtime` queue. Measure in K-2. The rider offer push has TTL 45 s and high urgency (good). | Solution + QA |
| RV-038 | Major | **Reconnect storms.** All restaurant tablets and riders reconnect on every deploy (graceful drain), on every 10-min `reauth` (RV-011), and on RDS failover. Each reconnect refetches every active query. | 08 §6.2; 11 §4.1; 17 §5.2 | Add jitter to `retry:` (2–10 s). Refetch only active order/inbox queries on reconnect. Include a "deploy during K-2" scenario in the load test. | Frontend + QA |
| RV-039 | Minor | **Day-one partitioning** of `rider_location_pings` (daily) and `outbox_events` (monthly) at ≈ 30–170k rows/day adds a partition-maintenance job, DEFAULT-partition alerts and PK constraints for no V1 benefit. A missed maintenance run sends rows to the DEFAULT partition, which then blocks creating that partition later. | 10 DB-D11, §14 | V1: plain tables with a nightly batched `DELETE` by an indexed timestamp. Partition `audit_logs` only if hash sealing stays per-row (RV-028). Revisit at 20M rows (10 §14 thresholds). | Backend |
| RV-040 | Minor | **The rider-ping evidence plan doesn't match PWA reality.** Doc 10 stores "**every** ping during an active delivery (evidence for 'rider never came' disputes)". Docs 06/18 say pings stop while the rider is in Google Maps, which is most of the active delivery. | 10 §8.1; 06 §4; 18 §8 | Rely on location captured at each status tap (already in `delivery_status_history`). Drop the "every ping" evidence claim. Keep idle pings sparse. | Backend |
| RV-041 | Minor | **PostGIS discovery and search are fine at 50 restaurants.** Telugu search needs expression trigram indexes on `name_i18n->>'te'` plus a normalised romanisation column. That isn't specified (10 §4 uses `*_i18n` JSONB). | 04 §5.2; 10 DB-D05 | Add the index definitions. Keep synonym-table search. The romanisation of Telugu names (04 §5.2 step 3) is P2 (01 CUS-SRCH-006), so defer it. | Backend |
| RV-042 | Minor | **The single DB holds everything:** River, sessions, rate limits, idempotency keys, heartbeats every 30 s per device, pings. WAL is ≈ 300–500 MB/day, dominated by rider updates (25 §10.5). That is fine, but `rider_availability` and `restaurant_devices` hot updates need `fillfactor` (done for one table only) and autovacuum tuning. | 10 §8.1; 25 §10.5 | Set `fillfactor=70` on `restaurant_devices` too. Add autovacuum thresholds for River and hot tables to the migration checklist. | Backend |

---

## 6. Data-model problems

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-043 | Major | **Missing tables required elsewhere:** `rate_limit_buckets` (R21; 08 §3.2, 12 §2.2); `transfers`, `pa_settlements`, `pa_settlement_lines`, `recon_exceptions` (14); `erasure_requests` (23 §9); restaurant **leads** and customer **area waitlist** (05 §3.2, 07 §7.1, 01 CUS-ADDR-003, 04 C-04); `staff_invites`/`partner_applications` (12 §2.3); SOS events (06 §11, committed P1); call-tap log with timestamps (01 BR-CONT-001 requires each tap logged; 10 keeps only `call_attempts` count); gig-worker registration export fields (LEG-GIG-001 P0). | 10 §1.5 (81 tables listed); refs in cell | Add these, or map each to an existing table with a note. Grow 10 §1.5 to the true count. | Backend |
| RV-044 | Major | **81 tables is more than V1 needs**, while the list above is missing. Cut or merge: `rider_shifts` and the `ON_BREAK` state (slot booking is V2+); `customer_profiles` (merge into `users`); `devices` vs `restaurant_devices` (one table); `feature_flags` + `app_config` (one); `rating_aggregates` (compute nightly into `restaurants`/`riders`); `coupon_targets` `CUISINE`/`USER` types; `reviews` moderation workflow (P1 stretch); `phone_change_requests` (P1 stretch); `outbox_events` (RV-002). | 10 §1.5; 02 §1.2 stretch list | Target ≈ 70 tables including the missing ones. Fewer aggregates means fewer sqlc packages and fewer authz rows. | Backend |
| RV-045 | Major | **The ledger chart of accounts differs between 10 and 14.** 10 uses `PG_CLEARING`, `EXPENSE_PG_FEES`, `GST_OUTPUT_PAYABLE`, `TDS_PAYABLE`, `GOODWILL_LIABILITY`. 14 uses `PA_CLEARING`, `EXPENSE_PA_FEES`, `GST_OUTPUT_9_5_RESTAURANT/_DELIVERY/_OWN`, `GST_INPUT_CREDIT`, `TDS_194O_PAYABLE`, `REFUNDS_PAYABLE`, `EXPENSE_PROMOTIONS`. 14 has `gstin_state` and `normal_side` columns; 10 doesn't. GST by §9(5) category is needed for GSTR-3B reporting (14 §12.1). | 10 §9; 14 §10.1–10.2 | Doc 14's chart wins (it is the finance design). Doc 10 adopts its codes and the `gstin_state` dimension. | Backend + Solution |
| RV-046 | Major | **The worked examples contradict R8.** 14 §10.3 and §13 compute fee GST *on top* (₹30 + ₹5.40) and total ₹366.80 (not whole rupees). 14 §12.4 says "not rounded to the rupee unless Product decides". R8 and BR-FEE-007 say fees are displayed GST-inclusive (configurable) and payable is rounded with `ROUND_OFF`. The contribution example (₹40.46) therefore overstates platform revenue by ≈ ₹5.3/order. | 14 §10.3, §12.4, §13; 01 BR-FEE-007/009/010; R8 | Redo 14's examples with inclusive fees and round-off. Keep tax-exclusive *ledger bases* (correct). Formula: `fee_taxable = round(fee_incl × 100/118)`. | Solution Architect |
| RV-047 | Major | **The COD refund policy conflicts across docs and with consumer law.** Doc 10 §7: "**COD orders never get money refunds** (ruling 9)". Doc 13 §6.2 offers goodwill coupons only. Doc 01 BR-REF-004 and ADM-PAYO-007 offer a manual UPI/bank refund, with coupon by customer choice and "coupon never forced". R9 only says compensation is *issued as* coupons and there is no wallet. Forcing store credit for a paid, missing item is a consumer-protection risk `[LEGAL]`. | 10 §7; 13 §6.2; 01 BR-REF-004; R9 | Lead decision D3. Recommend: customer choice, with a manual UPI refund recorded with UTR (`refunds.provider='MANUAL'` already exists) or a coupon. | Lead + Backend + Legal |
| RV-048 | Major | **Coupon accounting has holes.** (a) Usage is counted at `PLACED` (BR-COUP-004), but redemptions on COD orders that end `UNDELIVERABLE` with customer fault are never released, while the budget burns. (b) The GST valuation of a platform-funded discount is assumed "before discount" (01 BR-FEE-010, 14 §10.4) and only flagged `[LEGAL]`. If the CA disagrees, every invoice, the ledger rule and the restaurant statement change. | 01 BR-COUP-004, BR-CAN-003; 14 §10.4; 10 §5.4 | (a) Release rules: release on any non-delivered terminal state except customer-fault `UNDELIVERABLE`, which burns the coupon. Write that explicitly. (b) Put OQ-01/PAY-3 on the critical path: CA sign-off before quote-engine code freeze. | Product + Solution |
| RV-049 | Major | **The delivery OTP default contradicts the PRD.** BR-OTP-001 (P0) requires a delivery OTP for prepaid orders with payable ≥ ₹300. Doc 10 seeds `dispatch.delivery_code_required=false`. 08 §14 and 10 §19 call it `[OPEN]`. `delivery_code_hmac` stores only an HMAC, yet the customer app must re-display the code (CUS-TRK-004), so it needs deterministic derivation or encrypted storage. | 01 BR-OTP-001, CUS-TRK-004; 10 §6, §15.1; 08 §14 | Lead decision D13. If on: store the code encrypted (or derive it as HMAC(K, order_id)[:4]) so the customer app can show it. Seed the flag to match. | Backend + Product |
| RV-050 | Minor | **Rider ratings: stars vs thumbs.** 01 CUS-RATE-001 (1–5 stars) and M-41 ("≥ 4.3/5") vs 04 §11, 10 `ratings.thumbs_up` and 11 (thumbs). M-41 cannot be computed from thumbs. | 01 M-41; 10 §10; 11 §2.3 | Keep thumbs + tags (simpler). Change M-41 to "≥ 90% thumbs-up". | Product |
| RV-051 | Minor | **Bounds disagree.** Commission: 01 hard bound 0–30% vs 10 `commission_bps 0–5000`. Quantity per line: 01 1–20 vs 10/11 1–50. Prep time: 13 5–90 vs 10 5–120. Price-increase flag: 01 > 30% vs 05 > 25%. | refs | Product picks one value each. 10 encodes them as CHECKs. | Product + Backend |
| RV-052 | Minor | **Multi-city readiness is good** (`city_id` everywhere, per-state GST series). One gap: coupon codes are global-unique (10 §17) while doc 01 says "unique per city". | 10 §17; 01 ADM-COUP-001 | Keep global and fix 01. | Product |

---

## 7. UX gaps

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-053 | Major | **Restaurant alert escalation is too weak for kitchens.** The chain is in-app loop → push every 30 s → SMS/WhatsApp at 60 s → ops call at 90 s. An SMS to the owner's pocket in a noisy kitchen is the weakest link. The system can't place a voice call; ops must dial manually. | 05 §4.3; 15 §7; 13 T-ACC-* | Add an **automated voice call** (IVR "You have a new rovo order, press 1 to hear the code") to the counter phone and the owner at 60 s, via the SMS aggregator's voice API (≈ ₹0.3–0.6/call per 15 §5.1, unverified). Cost at 10% of orders is under ₹500/month. Mark it P0 for launch. | UX + Solution |
| RV-054 | Major | **Counter device sessions and lifecycle.** Partner sessions expire at 7 days idle / **30 days absolute** (12 §3.4), so counter tablets will be logged out mid-service monthly. A Chrome restart needs a tap to unlock audio (05 §4.1). OEM battery killers are only covered by "illustrated steps". | 12 §3.4; 05 §3.6, §4.1; 18 §7.2 | Device-bound long-lived refresh for `restaurant_devices.is_order_receiver` (rotated, revocable, re-auth scheduled outside service hours). The TWA (18 §10.1) with a high-importance notification channel becomes a **launch item** for restaurant devices. Ops ships restaurants pre-configured devices (RV-062). | Security + Frontend |
| RV-055 | Major | **Backgrounded riders vs dispatch eligibility** (top-10 #6). Doc 06 marks riders `ONLINE_STALE` after 90 s hidden. Docs 13/16 exclude locations older than 3 min. Yet doc 18 relies on Web Push to reach backgrounded riders. Riders waiting at a stand in 44 °C heat will not keep the screen on for hours. | 06 §4; 13 T-RIDER-STALE; 16 §7.2; 18 §8 | Two-tier dispatch: tier 1 is fresh (≤ 3 min); tier 2 is online with location ≤ 15 min, contacted by push with `Urgency: high`. Treat the rider's location as approximate and weight the score. Lead decision D8. | Backend + UX |
| RV-056 | Major | **The address-quality rules conflict with R13.** Doc 04 §8.1 and C-12 have a "Skip map" path with a locality centroid and `pin_approximate=true`. Doc 08 §11 says "Pin optional for that session". Doc 01 CUS-ADDR-002 requires house no. *and* building/street. R13 says landmark + map pin required, building/street optional. | 04 §8.1; 08 §11; 01 CUS-ADDR-002; R13 | No pinless addresses. If tiles fail, show a lightweight offline pin on a static locality image, or use "Use my current location" as the pin. Remove "building/street required" and house no. stays required per R13 baseline. | UX + Product |
| RV-057 | Major | **First checkout is too long for low-literacy users.** Turnstile, OTP, name, consent screen, 18+ declaration, address with a pin, then the UPI app hand-off. Each step can fail on patchy 4G. There is no assisted-ordering path (call-to-order by ops) for users like Narasimha (P3). | 04 §6–8; 03 P3; 12 §2.3 | Combine the consent and 18+ declaration into one screen. Run Turnstile only at risk (RV-034). Add ops-assisted phone ordering (admin places an order for a customer, COD) to V1 as a low-cost P1. | UX + Product |
| RV-058 | Minor | **Telugu.** OTP SMS is English-only (fine). Restaurant UI labels are pending native review (05 §1.1). There are no **audio** cues for illiterate staff and riders: the alarm is a tone, not words. | 05 §1.1; 06 §1 (Telugu audio deferred) | Add a pre-recorded Telugu voice line to the new-order and new-offer alarms ("కొత్త ఆర్డర్ వచ్చింది") — one precached ~20 KB asset each. | UX |
| RV-059 | Minor | **Support depends on async tickets plus a "responsive local support number"** (01 §2.1), but there is no telephony plan. M-43 (p90 first response ≤ 10 min) needs staffing across 15.5 service hours, 7 days a week. | 01 M-43, CUS-SUPP-003; 02 §4 | Budget a business phone line or IVR number (with call recording consent). Staff ≥ 2 shifts. Make M-43 a pilot measurement, not a gate. | Product + Release |
| RV-060 | Minor | **The payment-failure UX is good** (04 §8.5, E2/E3). One gap: after the UPI app returns, low-RAM Android often kills the Chrome tab. On reopen, the app must land on the pending order from the persisted idempotency key (04 E1 covers placement but not the PA return). | 04 E1–E3; 18 §9.1 | On start, any order in `PENDING_PAYMENT` < 15 min old opens C-14 automatically. | Frontend |

---

## 8. Testing gaps

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-061 | Major | **The golden-flow E2E depends on an unproven fake-clock spike.** River selects due jobs by DB time. Doc 20 admits `run-due` may have to re-schedule jobs (§10.2 [OPEN]). Ledger cut-offs use `created_at DEFAULT now()` (RV-005). The 4-context Playwright flow can't exercise weekly payouts without that spike. | 20 §10.2; 10 §9 | Make the spike a week-1 exit criterion. Define the fallback now (E2E profile: offer 5 s, accept 10 s; settlement triggered via `/_test/settlement/run?periodEnd=`). | QA + Backend |
| RV-062 | Major | **Device lab and restaurant devices aren't budgeted or owned.** Doc 18 §12 lists D1–D7 real devices including iPhones. The pre-release real-device checklist is 45 min per device. Restaurants are assumed to own a dedicated phone, but persona P4 (Venkatesh) has one shared phone. | 18 §12; 20 §11.5; 03 P4; 05 §1 | Buy 5 budget Android devices (Redmi/Realme/Vivo/Samsung A-series) for the lab, plus about 20 counter devices for restaurants without one, ≈ ₹6–8k each `[ASSUMPTION]`. Add to doc 25 and 29. | QA + Release |
| RV-063 | Major | **PA reconciliation can't be fully tested in sandbox.** Sandboxes rarely produce realistic settlement reports with fees, holds and Route transfers. The 14 §16 recon job and the split-settlement `ReleaseHold`/`ReverseTransfer` paths would be first exercised in production. | 14 §16, §20; 20 §7.3 | Get sample settlement and recon files from the chosen PA during onboarding. Golden-file tests against them. A "₹1 live transaction" test in production before Gate A (02 A4 already requires a real settlement; extend it to a refund and a transfer release). | QA + Solution |
| RV-064 | Minor | **Load-test numbers differ:** peak 300/h (01 NFR-PERF-006) vs 500/h used by 20 §12.1 vs 3–5/min (08 §9.2) vs 3.3/min (25 §10.1). Design load 900 vs 1,500/h. The load test also ignores WAF behaviour under **CGNAT** (many customers behind one Jio IP), which W4 (3,000 per 5 min per IP) may block at peak. | 01; 20 §12.1; 08; 25; 19 §6.11 W4 | One load model owned by doc 20, referenced by 01/08/25. Add a CGNAT scenario (500 sessions from 5 IPs). | QA + DevOps |
| RV-065 | Minor | **DLT template conformity isn't testable offline.** Operators scrub non-matching SMS. Staging needs a real DLT test template and real phones on Jio/Airtel/Vi/BSNL (02 A4). | 15 §6; 02 A4 | Add a staging "SMS template conformance" check per release: the template mirror in the repo is diffed against the provider export. | QA |
| RV-066 | Minor | **The coverage gate (90% statements on money modules) is fine.** But there is no **CA-reviewed golden invoice set**. Invoice tests compare PDF text against our own expectations. | 20 §4.3; 14 §20 | The CA signs off 10 golden invoices and statements (prepaid, COD, coupon, partial refund, cancellation). These become fixtures. | QA + Solution |

---

## 9. Operational risks

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-067 | Major | **On-call is unrealistic for 2–3 developers covering 108 service hours a week.** Paging relies on Telegram plus Grafana IRM Free (phone escalation "UNVERIFIED"). 99.9% critical-path availability is measured 24×7 (01 NFR-AVAIL-001). | 24 §10; 01 NFR-AVAIL-001; 19 §1 ("24×7 on-call is not realistic") | Pilot SLO 99.5% measured during service hours. Dev on-call 08:00–23:30, with ops as first responders using runbooks (pause zone, COD-only, accept on behalf). Use a paid phone-call escalation (UptimeRobot or Grafana IRM paid) for P1. Raise to 99.9% after 3 months of data. | DevOps + Release |
| RV-068 | Major | **Manual money operations have no labour budget.** Every week, finance must: build payout batches with maker-checker by a second finance/super; enter UTRs; verify rider COD deposits against an uploaded bank statement daily (M-20 is "100% same day"); reconcile PA files; handle 48 h holds after bank changes. Split settlement adds release-hold operations. | 07 §12.1; 14 §11, §16, §17; 01 M-20 | A part-time finance admin is not enough at 250 orders/day. Plan one finance FTE from Gate B. Automate bank-statement matching in V1 (CSV import + UTR match) — it's already designed (14 §16.1); make it P0. | Release + Product |
| RV-069 | Major | **Restaurant onboarding throughput is underestimated.** Gate B needs 40 live restaurants. Each needs: KYC; FSSAI checked manually on FoSCoS; bank verification; PA linked-account KYC (split model); ops digitising the menu from photos (≈ 80 items each, with Telugu names); device setup and sound test; a test order. Bulk CSV import (RES-MENU-009) is only a stretch item. | 02 §1.2, §5.2 B1/B4; 05 §3; 14 §3.1 | Commit RES-MENU-009 (CSV import, ops-side) to V1. Plan ≈ 1 restaurant per ops person per day. Start onboarding 6 weeks before Gate A. | Product + Release |
| RV-070 | Major | **Rider supply** (top-10 #6). Riders are expected to be independent contractors with no earnings floor. Insurance is OPEN (LEG-GIG-004). Social Security Code worker registration (LEG-GIG-001 P0) has no data/export design (RV-043). | 01 M-54/55, LEG-GIG-*; 02 B5/B6 | Pilot: a minimum guarantee per scheduled peak slot (ledger adjustment type `MG_TOPUP`). Group accident cover from day one. Design the gig-worker export (fields + CSV) in 10/11. | Product + Release |
| RV-071 | Minor | **Runbooks** (02 A6, 19 §10.4 "written in the private ops repo") have no owner or format. DR drills are monthly (D1/D2), quarterly (D3–D6), plus weekly backup verification (20 §15.3). That is heavy for this team. | 23 §7; 20 §15.3 | Pilot: monthly D1 *or* D2 alternating, D4 automated. D3 game day once before Gate B, then semi-annual. | DevOps |
| RV-072 | Minor | **Maker-checker staffing.** Doc 12 §5.5 makes "≥ 2 active ADMIN_SUPER and ≥ 1 ADMIN_FINANCE" a go-live gate. Doc 07 §3 proposes break-glass. Both assume people the launch team (02 §4) may not have. | 12 §5.5; 07 §3; 02 §4 | See D5. | Lead |
| RV-073 | Minor | **The incident-response comms channel is Telegram.** Ops may also use WhatsApp groups. Choose one. | 24 §10 | Choose one, document it in 29. | Release |

---

## 10. Vendor lock-in and the portability ADR

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-074 | Major | **ADR-025 delivers portability of application code only.** Infrastructure is AWS-only (OpenTofu modules, CloudFront flat-rate, WAF rules, ECS, RDS cross-region backups, KMS envelope encryption via Tink/AWS KMS). DR-4's "stand up on GCP from the L4 dump" (RTO ≤ 8 h) has no GCP IaC, no tested GCS S3-interop adapter, and no rehearsal. | 09 ADR-025/026; 23 §1, DR-4; 25 §14.8 | State the realistic claim: "app portable; infra rebuild on another cloud = weeks". Drop the 8 h RTO for DR-4, or fund a one-off GCP restore rehearsal before Gate B. | DevOps + Solution |
| RV-075 | Major | **PA lock-in is understated.** With Route/Easy Split, every restaurant is a KYC'd linked account at that PA. Switching PA means re-KYC for all restaurants and migrating held transfers. Saved PA tokens and webhook semantics differ too. | 09 ADR-012; 14 §4 | Record it in doc 30 as a decision with a switching cost. Keep `transfers` keyed by provider. The `Provider` interface can't hide this. | Solution |
| RV-076 | Minor | **SMS is low lock-in** (the DLT PE registration is operator-level and templates are reusable across aggregators). **Grafana is low** (OTLP; dashboards as code). **Sentry is medium** (SDK). **Turnstile is a Cloudflare dependency inside an AWS stack**, behind a `BotChallenge` interface (good). **CloudFront flat-rate** is a pricing dependency, not a technical one. | 15 §5; 24 §1; 12 §2.2 | No change beyond RV-034/RV-032. | — |
| RV-077 | Minor | **River pre-1.0** (v0.48) is a core dependency for queue, timers and outbox. | 08 §13; 09 ADR-006 | Pin exactly. Run the River upgrade only between releases with the full integration suite. The `platform/queue` interface exists (good). | Solution |

---

## 11. V1 scope creep and essential gaps

Detailed cut and missing lists are in §13. Summary findings:

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-078 | Major | **Manual surge pricing is in V1** in 16 GEO-D05/§6.2 (zone surge fee, rider bonus, rain factor), 11 (`PUT/DELETE /admin/zones/{id}/surge`, `surge` in restaurant list and earnings), and 10 (`surge_fee_paise`, `SURGE_FEE` tax component). It contradicts 01 BR-FEE-006 ("No surge/rain/peak fees in V1"), 02 §2.2 (V2+) and 02 §6.4. | refs | Cut. Keep zone pause plus a *rider-side* peak bonus as a ledger adjustment if supply is short. | Backend + Product |
| RV-079 | Major | **WhatsApp OTP and WhatsApp utility messages are in V1** (15 §0, §5.2, §7, §9). Doc 12 §2.6 defers WhatsApp OTP to V1.1 and doc 01 CUS-AUTH-010 says P2. It needs Meta business verification, template approval and a BSP/Cloud API integration. | 15; 12 §2.6; 01 | Cut to V1.1. V1 fallback is a secondary SMS aggregator. | Solution |
| RV-080 | Major | **Build-time prerendering of public pages** (17 F1/§1.3) uses TanStack Start (RC in 2025), a nightly GitHub Actions rebuild that calls the production API, and a `repository_dispatch` on profile change. The stated benefits are SEO and WhatsApp link previews. In a city of 2–2.5 lakh people, acquisition is word of mouth and WhatsApp shares, which need **OG tags only**. | 17 §1.1–1.3 | V1: the Go API serves a tiny server-rendered HTML for `/{city}/r/{slug}` share links (title, description, OG image), then redirects or boots the SPA. No build-time prerender, no CI dependency on production data. Revisit per 17's own SSR triggers. | Frontend |
| RV-081 | Major | **Maker-checker breadth:** 13 action types in 12 §5.5, plus fee configs (10 `fee_configs.approval_id`, 11 ⚖ on fee-configs and tax-rules). Doc 07 §3 deliberately keeps fee config out of maker-checker. | 12 §5.5; 07 §3; 10 §5.2; 11 §2.6 | V1 maker-checker covers: payout batch release; refunds above ₹500; payout-destination change; admin role grant/TOTP reset; manual ledger adjustment. Everything else is audit + preview + a next-day review report. | Security + Lead |
| RV-082 | Major | **Missing essentials** (details in §13.2): voice-call escalation, counter devices, CERT-In log archive, gig-worker export, rider minimum guarantee, bulk menu import, ops-assisted ordering, the support phone line, the erasure ledger, the schema tables in RV-043. | — | Add to the backlog (27) as P0/P1 as marked. | Release |

---

## 12. Disagreements requiring Lead decision

| # | Contested point | Options | Reviewer recommendation |
|---|---|---|---|
| D1 | **API edge and CDN cost** (R14 vs 22 §3.1) | (a) `/api/*` on every app host through CloudFront flat-rate Pro; monitor the allowance, reduce chatty requests. (b) Same, but on CloudFront pay-as-you-go. (c) Keep the API on an `api.` host with CORS (doc 22/24). | **(a)**, with (b) as the fallback if the allowance is exceeded for 2 months. **Reject (c)**: it breaks doc 12's cookie and CSRF model and R14. |
| D2 | **No NAT, tasks in public subnets** (22 §4.3 vs 19 §7.2) | (a) Accept for the closed pilot with compensating controls (RV-026). (b) One NAT Gateway in one AZ (≈ $41+/month). (c) NAT per AZ (≈ $82+/month). | **(a) until Gate B**, then (b), unless a provider requires IP allowlisting earlier. Doc 19 is amended to match. |
| D3 | **COD compensation: money refund vs coupon only** (10 §7, 13 §6.2 vs 01 BR-REF-004) | (a) Coupon only. (b) Customer choice: manual UPI refund (UTR) or coupon. | **(b)** — consumer-law safer `[LEGAL]`, and the manual flow is already designed (ADM-PAYO-007). |
| D4 | **Surge in V1** (16 vs 01/02) | (a) Manual zone surge with expiry. (b) No surge; zone pause plus rider peak bonus via adjustment. | **(b)**. |
| D5 | **Maker-checker staffing gate** (12 §5.5 vs 07 §3) | (a) Go-live needs ≥ 2 SUPER + ≥ 1 FINANCE. (b) Break-glass self-approval with 24 h post-review. (c) Narrow maker-checker list (RV-081) plus (b). | **(c)**. Gate: ≥ 2 named people who can approve money actions. |
| D6 | **Pilot DB topology** (22: Multi-AZ t4g.medium from day 1; 01 NFR-AVAIL-003: single-AZ allowed for pilot; 08 §9.2: different trigger) | (a) Multi-AZ from day 1. (b) Single-AZ t4g.small for the closed pilot; Multi-AZ mandatory at Gate B. | **(b)**. One trigger everywhere: "before Gate B (public launch) or > 100 orders/day". |
| D7 | **Build-time prerendering** (17 F1) | (a) TanStack Start prerender with nightly rebuild. (b) API-served OG share pages only. | **(b)**. |
| D8 | **Rider staleness and dispatch tiers** (06: 90 s; 01: 2–3 min; 13: 3/10 min; 16: 3 min) | (a) Fresh only (≤ 3 min). (b) Two tiers: ≤ 3 min fresh, ≤ 15 min stale via push. | **(b)**. Thresholds: unavailable at 3 min for tier 1, auto-offline at 15 min. |
| D9 | **Restaurant money flow** (ADR-012 split settlement vs collect-and-payout) | (a) Split settlement mandatory. (b) Collect-and-payout with a legal opinion. (c) Both supported; pick per counsel (R25). | **(c)**, but get the opinion **before Phase-2 week 4**. If (a), add PA linked-account KYC to the onboarding critical path now. |
| D10 | **Observability vendors and residency** (Grafana Cloud + Sentry + CloudWatch + UptimeRobot + Faro) | (a) As designed. (b) Grafana Cloud (metrics, traces, logs, Faro) + CloudWatch Logs as the 180-day compliance archive + UptimeRobot; no Sentry. (c) (a) plus a separate compliance archive. | **(b)** — fewer vendors, India residency, solves CERT-In. Sentry only if Faro's error triage is inadequate after the pilot. |
| D11 | **Admin edge gate** (12 §3.5 IAP option A) | (a) Identity-aware proxy (Verified Access/IAP). (b) WAF IP allowlist. (c) Passkeys in V1 + WAF rate/geo rules. | **(c)**. |
| D12 | **KYC file protection** (19 §6.10, 12 §6.2) | (a) ClamAV + PDF parsing + app-layer envelope encryption. (b) Images only (client converts), SSE-KMS, audited streaming view, no app-layer encryption. (c) (b) + GuardDuty Malware Protection for S3. | **(b)** for the pilot; (c) if PDFs must be accepted. |
| D13 | **Delivery OTP default** (01 BR-OTP-001 P0 vs 10 seed `false`) | (a) Off at launch. (b) On for prepaid ≥ ₹300. | **(b)**, as the PRD says. Store the code so the customer app can display it (RV-049). |
| D14 | **Restaurant cancel after accept** (01 RES-ORD-006 self-cancel vs 05 §4.5 / 11 ops-mediated) | (a) Restaurant self-cancels. (b) Restaurant raises an urgent issue; ops cancels. | **(b)** — prevents silent cancellations. Update 01. |
| D15 | **Cross-module FKs on money paths** (10 DB-D04) | (a) No FKs except `cities`/`users`. (b) Allow FKs to `orders` from payments, refunds, deliveries and invoices. | **(b)** (RV-006). |
| D16 | **Event fan-out** (08 §7.3) | (a) `event_log` + fanout job + subscriber jobs. (b) Direct `InsertManyTx` of subscriber jobs; no event table. | **(b)** — still River-as-outbox (R22), one less hop. |

---

## 13. Scope cut list and missing items

### 13.1 Scope cut list (defer to V1.1+ unless noted)

| # | Cut / defer | From | Why | Saves |
|---|---|---|---|---|
| C1 | Manual surge (zone surge fee, rain factor, rider surge bonus, surge endpoints and columns) | 16 §6.2; 11 §2.6; 10 | Contradicts PRD; new money path | Pricing code, tests, admin UI |
| C2 | WhatsApp OTP and WhatsApp utility notifications | 15 | PRD P2; Meta onboarding | 1 provider integration |
| C3 | Build-time prerender (TanStack Start), nightly rebuild jobs | 17 F1 | OG share page is enough | Framework risk, CI coupling |
| C4 | Identity-aware proxy for admin | 12 §3.5 | Cost, ops; passkeys instead | ≈ ₹18k/month (unverified) |
| C5 | ClamAV service and PDF KYC uploads | 19 §6.10 | Images only | 1 service, egress |
| C6 | Per-row hash-chained audit log + daily WORM anchor | 12 §5.7 | Append-only grants are enough for V1; hourly sealing later | Write contention |
| C7 | Maker-checker beyond the 5 money/privilege actions | 12 §5.5; 10/11 fee & tax ⚖ | Team size | Approval flows |
| C8 | Day-one partitioning of pings and outbox; `outbox_events` table and fanout hop | 10 §14; 08 §7.3 | Volume tiny | Maintenance job |
| C9 | `tools/tableowner`, `tools/eventschema` | 08 §4.3 | Use go-arch-lint + review | Tooling weeks |
| C10 | Quarterly region game day; DR pre-provisioning in `ap-south-2` (ECR, secrets replicas, MRKs, ALB cert) before Gate B | 23 §7, DR-3 | Pilot keeps cross-region backups + IaC only | Cost, drills |
| C11 | Grafana Faro **or** Sentry (keep one); Alloy gateway task at pilot | 24 §1–2 | Vendor count | ≈ $31/month + ops |
| C12 | Rider shifts table, `ON_BREAK` state, rider offline-outbox beyond pickup/deliver | 10 §8.1; 13 §4.3; 18 §4.3 | Shifts are V2+; keep the narrow outbox | States, tests |
| C13 | Restaurant 7/30-day trend analytics (committed P1) → keep only today/week (RES-ANLY-001) | 02 §1.2 | Reporting queries; build once data exists | Admin/partner UI |
| C14 | Public text reviews moderation queue; review replies | 01 CUS-RATE-005; 10 `reviews` | Moderation load | 1 workflow |
| C15 | Zone GeoJSON import/export, impact dry-run, last-30-day drop heat map | 16 §8; 11 §2.6 | One zone at launch | Admin tooling |
| C16 | Coupon `SHARED` funding, `CUISINE`/`USER` targets, per-day budgets, bulk unique codes | 10 §5.4; 07 §10; 01 ADM-COUP-004 | Keep platform/restaurant funding only | Settlement complexity |
| C17 | Telugu romanisation of `name_te` | 04 §5.2 step 3 | PRD P2 (CUS-SRCH-006); synonym table covers it | Search code |
| C18 | Six-account AWS organisation | 19 §7.2; 22 §2 | Keep 4 accounts: mgmt, prod, nonprod, audit/backup | Account ops |
| C19 | Static-QR COD-UPI at the door (14 §11.3) | 14; 06 §7 | 02 says V1.1 (dynamic QR); fake-UTR risk | New money path |
| C20 | Admin broadcasts, `geo/live` map endpoint, tax-rule CRUD UI (seed by migration) | 11 §2.6 | Not needed at one city | Endpoints |

### 13.2 Missing items (add)

| # | Item | Priority | Owner |
|---|---|---|---|
| M1 | 180-day India log archive (CERT-In), ≥ 1-year security events; cost line | P0 | DevOps |
| M2 | Automated voice-call escalation for unaccepted orders (counter + owner) | P0 | Solution + UX |
| M3 | Counter-device provisioning (supply + setup) and device lab budget | P0 | Release + QA |
| M4 | `rate_limit_buckets` (R21), `transfers`, `pa_settlements(_lines)`, `recon_exceptions`, `erasure_requests`, leads, waitlist, `staff_invites`, SOS events, call-tap log in doc 10 | P0 | Backend |
| M5 | Gig-worker registration data fields + export (LEG-GIG-001) | P0 `[LEGAL]` | Backend + Product |
| M6 | Rider minimum-guarantee mechanism for the pilot (`MG_TOPUP` adjustment) + accident insurance decision | P0 (ops) | Product |
| M7 | Bulk menu CSV import, ops-side (RES-MENU-009 → committed) | P0 for Gate B | Product + Backend |
| M8 | Ops-assisted phone ordering (admin places a COD order for a customer) | P1 | Product |
| M9 | Support phone line / IVR number + staffing plan | P0 | Release |
| M10 | Device-bound long-lived session for order-receiver devices | P0 | Security |
| M11 | Catch-up semantics for all periodic jobs (RV-004) + "missed settlement" alert | P0 | Backend |
| M12 | `pg_notification_queue_usage` alert and LISTEN watchdog (RV-003) | P0 | Solution + DevOps |
| M13 | Request-volume model for the CDN allowance and request-reduction (batched pings, heartbeat cadence) | P1 | DevOps |
| M14 | CA-signed golden invoices and statements; PA sample settlement files | P0 | QA + Solution |
| M15 | Erasure map per table and bucket (RV-031) | P0 `[LEGAL]` | Backend + Security |
| M16 | Legal-entity, GST (ECO), FSSAI e-commerce licence, DLT, PA onboarding on the critical path with lead times | P0 | Release |
| M17 | Pilot "closed-pilot" IaC profile (RV-012) | P1 | DevOps |

---

### 13.3 Cross-document consistency findings

| ID | Sev | Finding | Evidence | Recommendation | Owner |
|---|---|---|---|---|---|
| RV-083 | Blocker | **The restaurant accept-timeout ladder contradicts R1 in six docs** (register rows 1–14). Phase-2 implementers reading 05/08/15 will build `REJECTED` + 240 s / 6 min. That breaks M-02, refund rules, quality metrics and the ops board. | 01, 04, 05, 07, 08, 15 (register §14 #1–14); 13 SM-D02 correct | Owners apply R1 verbatim. Doc 13 §5 timer table becomes the single source; other docs link to it instead of restating numbers. | Product, UX, Solution |
| RV-084 | Blocker | **Fee slabs and the serviceable radius can't be implemented consistently** (register rows 36–39). Seed slabs end at 8,000 m road, while the R18 radius (7 km straight-line ≈ 9.1 km road) produces orders with no slab. Doc 16's golden fixtures assert the opposite of R18 at boundaries and for R2→C1. | 10 §5.2/§15.1; 16 §4.2/§6.1; 01 BR-FEE-001; 07 §9; R18 | Re-seed slabs to 10 km; `[lo,hi)`; straight-line radius check; regenerate 16 §4.2 golden values; add property test "every serviceable point has exactly one slab". | Backend + Product |
| RV-086 | Major | **Several docs restate rulings with drifting numbers** (staleness, thresholds, payout day, cash ageing, peak load). Each later change will drift again. | §14 rows 51–58, 69–72 | Each parameter gets one owning doc (13 timers, 10 seeds/`app_config`, 20 load model, 25 cost). Others reference by key, not by value. | Lead |

## 14. Cross-doc inconsistency register

Correct values follow the §8 rulings. Where no ruling exists, the Reviewer's proposal is marked *(proposal)* and should be confirmed in D-items or by the owner.

| # | Doc | Location | Current text | Correct value |
|---|---|---|---|---|
| 1 | 01 | §4.1 M-02 | "Orders `REJECTED` with reason `RESTAURANT_TIMEOUT`" | `CANCELLED`, `cancelled_by=SYSTEM`, reason `RESTAURANT_UNRESPONSIVE` (R1); metric counts these |
| 2 | 01 | RES-ORD-004 | "auto-rejected (`RESTAURANT_TIMEOUT`)" | auto-cancelled `RESTAURANT_UNRESPONSIVE`, 180 s (R1) |
| 3 | 01 | BR-TIME-002 | "60 s re-alert… 90 s flagged… 180 s auto-reject with `RESTAURANT_TIMEOUT`" | repeat every 30 s; owner SMS at 60 s; ops at 90 s; cancel at 180 s (R1) |
| 4 | 01 | BR-TIME-003 | "2 consecutive timeouts → paused until I resume" | 1st miss → pause 30 min; 2 consecutive → paused until owner resumes (R1) |
| 5 | 01 | RES-HOUR-005, OQ-07 | heartbeat gating "within last 10 min" (P1) | no device heartbeat for **3 min** while open → auto-pause (R1), P0 |
| 6 | 04 | §15 E6 | "auto-cancel time (default 6 min, doc 05)… From minute 3…" | 180 s; "taking longer" message at 90 s (R1) |
| 7 | 05 | §4.3 diagram + text | T+90 s owner SMS; T+3 min ops; T+6 min cancel; `RESTAURANT_NO_RESPONSE`; red at T+2 min | 60 s / 90 s / 180 s; `RESTAURANT_UNRESPONSIVE` (R1) |
| 8 | 05 | §11 req 3 | "T+90 s owner SMS, T+3 min ops alert, T+6 min system cancel" | as R1 |
| 9 | 08 | §3.2 ordering "Consumes" | `RestaurantAcceptTimeout` → `REJECTED` with `cancel_reason=RESTAURANT_TIMEOUT` | → `CANCELLED`/`RESTAURANT_UNRESPONSIVE` (R1) |
| 10 | 08 | §5.3 | `accept_timeout at +N min [ASSUMPTION N=4]`; "CAS PLACED→REJECTED" | +180 s; CAS → `CANCELLED` (R1) |
| 11 | 08 | §11 row "Restaurant not responding"; §14 | "escalate to ops after 2 min → auto-reject at 4 min"; "proposed 4 min with escalation at 2 min" | ops 90 s; cancel 180 s (R1) |
| 12 | 15 | §7 diagram + timings | `alert.ops_s=120`, `accept_timeout_s=240`, "PLACED → REJECTED (RESTAURANT_TIMEOUT)", "2 in a day → auto-pause" | `alert.ops_s=90`, `accept_timeout_s=180`, `CANCELLED`/`RESTAURANT_UNRESPONSIVE`, pause per R1 |
| 13 | 15 | §7 "Restaurant device health" | warn after 5 min, auto-pause after 15 min [OPEN] | auto-pause after 3 min (R1) |
| 14 | 07 | §4 dashboard | "Unaccepted > N min: PLACED for more than 3 min" | flag at 90 s (R1) |
| 15 | 05 | §4.3 Accept | "system immediately auto-advances it to `PREPARING`" | auto-advance after **60 s** or on tap (R3) |
| 16 | 05 | §9 R2 | flag `ready_marked_by=implicit` | `restaurant_skipped_ready` flag (R4) |
| 17 | 06 | §8.1 | rider may self-mark undeliverable if support doesn't respond in 5 min | support approval required (R5); escalate at 5 min, no self-mark |
| 18 | 04 | §10 row PICKED_UP | "2 refusals → COD disabled `[OPEN]`" | decided: 2 customer-fault COD failures → COD disabled (R5) |
| 19 | 14 | §14 | "New-customer COD cap: first order ≤ ₹500" | ₹600 first order, ₹1,000 cap (R6) |
| 20 | 06 / 08 / 10 | 06 §3 `BLOCKED_COD` at ≥ limit; 08 §5.5 "if cash_in_hand ≥ limit"; 10 `cod_blocked` "cash_in_hand ≥ limit (ruling 6)" | offer COD only if cash-in-hand + order payable ≤ ₹2,000 (R6) |
| 21 | 01 | BR-TIME-006 | "launch delay cap 0 (immediate)" | offer at `max(0, prep − rider_approach − buffer)` from `ACCEPTED` (R7) |
| 22 | 14 | §10.3, §13, §12.4 | fees GST added on top; total ₹366.80; "not rounded unless Product decides" | payable rounded to whole rupee with `ROUND_OFF`; fee GST presentation configurable, inclusive default (R8, BR-FEE-007) |
| 23 | 01 | CUS-CART-002 | "server-side for 24 h" | device cart only; signed `quoteId` (R12) |
| 24 | 04 | §6.2 | "if the server already has a cart for this account (from another device)" | no server cart (R12) |
| 25 | 08 | §3.2 cart & pricing; §5.1 | server cart `carts`, `cart_lines`, `UpsertCart`; `POST /api/v1/quotes` | no cart tables; `POST /api/v1/cart/quote` (R12) |
| 26 | 17 | §4.3 | `409 QUOTE_STALE`; `POST /orders` takes "quote_id (or the full cart plus address)" | `409 QUOTE_CHANGED`/`QUOTE_EXPIRED`; `quoteId` + `Idempotency-Key` only (R12, 11 §1.8) |
| 27 | 04 | §1.5, E11, §17 item 4 | "resumes SSE with `Last-Event-ID`", "Event IDs per order stream", heartbeat ≤ 25 s | no replay; refetch snapshot on reconnect; heartbeat 20 s (R10) |
| 28 | 05 | §11 req 1 | "`Last-Event-ID` replay of the last 15 min" | no server replay (R10) |
| 29 | 08 / 22 / 11 | 08 §6.2 30-min stream cap; 22 §3.3 55-min deadline; 11 §4.1 `reauth` at access-token expiry (10 min) | *(proposal)* one rule: 30-min cap, `reauth` only on session revocation (RV-011) |
| 30 | 01 / 04 / 08 | 01 CUS-ADDR-002 building/street required; 04 §8.1 "Skip map" pinless; 08 §11 "Pin optional for that session" | landmark + map pin required; building/street optional (R13) |
| 31 | 11 / 12 / 22 / 21 / 24 / 01–06 | 11 §1.1 & 12 §4.1 `partner.<domain>`; 22 §3.1 `partner.rovo.example`, 3 web buckets; 21 §3.3 build "customer/partner/admin"; 24 §2.2 "3 SPAs"; 01–06 "partner PWA" | four hosts `app.`, `restaurant.`, `rider.`, `admin.`; four buckets; four builds (R14) |
| 32 | 22 / 24 | 22 §3.1 `api.` ALB direct for apps; 24 §2.2 CORS and `tracePropagationTargets: api.` | `/api/*` same-origin via CDN on each app host; no CORS; `api.` reserved for native + webhooks (R14) |
| 33 | 12 / 09 / 24 | 12 §4.1 `https://api.rovo.in/v1/…`; ADR-008 `GET /v1/stream`; 24 §2.1 span `GET /v1/orders/{id}` | `/api/v1` on every host (R15) |
| 34 | 10 / 13 | 10 §8.2 `delivery_offers.status` CHECK 4 values; 13 §4.1 revocation folded into `EXPIRED` | add `REVOKED` (R16); `close_reason` keeps detail |
| 35 | 01 / 17 | 01 CUS-MENU-002, RES-MENU-006 `name_te`; 17 §10, §3.2, §4.2 `name_te` | `*_i18n` JSONB; API `nameI18n` + `displayName` (R17) |
| 36 | 01 | BR-FEE-001 | slabs "0–2… >6–8 km ₹50 (upper bounds inclusive)"; max radius 7 km road | road-adjusted slabs to 10 km incl. 8–10 ₹60; `[lo,hi)`; 7 km straight-line radius (R18) |
| 37 | 10 | §5.2 default `delivery_fee_slabs`, `max_serviceable_distance_m=8000`; §15.1 seed "to 8 km" | slabs `[2000,4000,6000,8000,10000]` ₹20–60; cap 7,000 m straight-line (R18) |
| 38 | 16 | GEO-D02, §6.1, §4.2 golden R2→C1 | radius applied to road distance; `≤ 2 km` inclusive (2,000 m → ₹20); R2→C1 7,033 m road = TOO_FAR | radius on straight-line (5,410 m is serviceable); 2,000 m → ₹30 (R18) |
| 39 | 07 | §9 fee slabs | "0–2 ₹20 · 2–4 ₹30 · 4–6 ₹40 · 6–8 ₹50" | add 8–10 ₹60 (R18) |
| 40 | 08 / 14 / 16 | 08 §11 "timers are DB-side (`now()`)"; 14 §18 "Our timers use DB time"; 16 §7.2 `now() - interval '3 minutes'` | injected app clock; no SQL `now()` in business logic (R19) |
| 41 | 10 | §9 payouts "cut-off: postings created_at < period_end" | cut-off on journal `accounting_date`/`occurred_at` from the app clock (R19) |
| 42 | 10 | §1.5 module tables | no `rate_limit_buckets` | add Postgres rate-limit table (R21) |
| 43 | 08 / 09 / 20 | 08 §2.1/§13 & ADR-005 "PG 18 preferred"; 20 §10.1 "Postgres 18 + PostGIS" | PostgreSQL **17** on RDS (18 only if available with PostGIS); same major in local/CI (R22) |
| 44 | 24 | §2.1 "Outbox rows store traceparent; the relay creates a span" | no relay; trace context in River job metadata (R22) |
| 45 | 22 / 21 | 22 §12 `infra/`; 21 §2 `infra/**` paths | `deploy/terraform/` (R23) |
| 46 | 02 / 22 / 21 / 23 / 24 / 08 | free-tier and Oracle preview text (see RV-019) | local Docker Compose only (R24) |
| 47 | 14 | §21 PAY-7 and §3 | "Build Razorpay adapter first" (fine); selection on written rates | Cashfree vs Razorpay on written rates; settlement model with counsel (R25) |
| 48 | 01 | §8.1 ADM-AUTH, §6/§7 roles | admins not explicitly separate identities | admins separate identities; `RIDER` ⟂ `RESTAURANT_*`; `SYSTEM` principal (R26) |
| 49 | 01 / 07 / 10 | ticket statuses: 01 ADM-TKT-001 `OPEN…AWAITING_CUSTOMER…`; 07 §11.1 `NEW…WAITING_ON_REQUESTER/INTERNAL…`; 10 `OPEN, IN_PROGRESS, AWAITING_REQUESTER, AWAITING_APPROVAL, RESOLVED, CLOSED, REOPENED` | adopt doc 10 set (R11 ticket statuses) |
| 50 | 06 / 10 / 13 / 11 | rider states: 06 `ONLINE_IDLE/ONLINE_STALE/ON_DELIVERY/BLOCKED_COD`; 10/13/11 `OFFLINE/AVAILABLE/ON_BREAK/ON_DELIVERY` | 10/13 set (R11 `rider_availability`); `cod_blocked` and "stale" are flags; `ON_BREAK` cut (C12) |
| 51 | 01 / 06 / 08 / 13 / 16 | staleness: 01 ≤ 2 min & 3/15 min; 06 90 s/15 min; 08 < 3 min; 13 3/10 min; 16 3 min | *(proposal, D8)* tier-1 ≤ 3 min; offline at 15 min |
| 52 | 01 / 07 / 10 / 11 / 13 | goodwill coupon: 01 ADM-TKT-002 capped ₹100; others maker-checker > ₹150 | *(proposal)* ₹150 threshold; drop "₹100 cap" from 01 |
| 53 | 07 vs 10 / 11 / 12 | coupon budget maker-checker: 07 > ₹10,000; others > ₹5,000 | *(proposal)* ₹10,000 (or remove, C7) |
| 54 | 07 vs 10 / 11 / 12 | fee-config changes: 07 §3 "no checker"; 10/11/12 maker-checker | *(proposal)* no checker; audit + preview + next-hour effect (RV-081) |
| 55 | 01 vs 05 | price-increase moderation flag: > 30% vs > 25% | *(proposal)* 30% |
| 56 | 01 vs 10 | commission hard bounds 0–30% vs `CHECK 0–5000 bps` | 0–3000 bps |
| 57 | 01 vs 10 / 11 | line quantity 1..20 vs 1–50 | *(proposal)* 1–20 |
| 58 | 13 vs 10 | prep time 5–90 vs 5–120 | *(proposal)* 5–90 |
| 59 | 01 vs 04 / 10 / 11 | rider rating 1–5 stars, M-41 ≥ 4.3/5 vs thumbs | thumbs + tags; M-41 → ≥ 90% thumbs-up (RV-050) |
| 60 | 01 vs 05 / 11 | restaurant self-cancel after accept (RES-ORD-006) vs ops-mediated issue | ops-mediated (D14) |
| 61 | 01 vs 08 / 13 | reject reason `ITEM_OUT_OF_STOCK` vs `ITEM_UNAVAILABLE` (08 §5.6) | doc 13 §6.3 catalogue |
| 62 | 01 vs 06 | contact window: from `PICKED_UP` to +15 min vs `ASSIGNED`…+60 min | 01 window (RV-035) |
| 63 | 08 vs 11 | webhook path `/webhooks/pa/razorpay` vs `/webhooks/payments/{provider}` | `/webhooks/payments/{provider}` |
| 64 | 15 vs 11 | `POST /api/v1/restaurant/orders/{id}/ack` vs no ack endpoint; partner paths `/partner/restaurants/{rid}/orders/...` | add `POST /partner/restaurants/{rid}/orders/{id}/ack` to 11 |
| 65 | 08 vs 11 vs 06 | SSE topic `restaurant:{id}:inbox` vs `inbox:{rid}`; event `offer.new` vs `offer.created` | doc 11 §4.2 names |
| 66 | 14 vs 10 | `payment_intents`, `webhook_events`, `account_balances`, ledger codes (`PA_CLEARING`…) vs `payments`, `payment_events`, `ledger_account_balances`, `PG_CLEARING`… | table names per 10; **account codes per 14** (RV-045) |
| 67 | 08 vs 10 | ownership: rider profile in `users`, `rider_locations` in `geo`, `push_subscriptions` in `users` vs `riders`/pings in `dispatch`, `push_subscriptions` in `notifications` | doc 10 §1.5 (update 08 §3.2) |
| 68 | 14 vs 10 | raw webhook body retention 180 days vs `payment_events` 8 years | raw 180 d (redacted); normalised fields 8 y (RV-030) |
| 69 | 01 / 08 / 22 / 23 | Multi-AZ trigger: 01 "before public launch or > 100/day"; 08 "first payout cycle or GMV > ₹10 lakh"; 22 Multi-AZ from day 1; R23 "Multi-AZ" | *(proposal, D6)* single-AZ closed pilot; Multi-AZ before Gate B or > 100/day |
| 70 | 01 vs 23 | RPO/RTO: 01 NFR-AVAIL-004 AZ RTO ≤ 1 h, region RTO ≤ 24 h / RPO ≤ 1 h; 23 AZ 1–3 min, region RPO ≤ 30 min / RTO ≤ 4 h | adopt 23 targets in 01 (stricter, designed) |
| 71 | 01 / 08 / 20 / 25 | peak: 300/h; 3–5/min; 500/h; 3.3/min; design 900 vs 1,500/h | one load model in 20, referenced everywhere (RV-064) |
| 72 | 01 vs 14 vs 19 | rider cash ageing: 01 48 h reminder / 72 h block; 14 24 h SLA / 48 h suspend; 19 24/48 h alerts, 72 h suspension. Rider payout day: 01 Tuesday vs 14 Wednesday. COD off 23:00–06:00 (14) vs service hours to 23:30 (01) | *(proposal)* 01 values (48/72 h, Tuesday riders); COD window = service hours |

Additional minor drift (fix during reconciliation, no ruling needed):
- 01 BR-COST-002 cloud ₹28–54k vs 22/25 (RV-013); SMS ₹1.5–5k vs 15 ₹1.3–1.5k.
- 21 §3.1/§3.2 paths `api/openapi/**`, `db/**` vs 26 `openapi/`, `backend/migrations/`.
- 17 §2 workspace at repo root vs 26 `web/`.
- 05 §8 "FSSAI expiry → auto-*suspended*" vs 01 BR-RES-004 "auto-*paused*".
- 01 BR-RPAY-004 rider cancel pay "base ₹25 + waiting" vs 10/13 "50% of base".
- 05 §2 staff cannot edit hours vs 11 `PUT operating-hours` OWN, STAFF.
- 06 §13 rider speed 18 km/h vs 01 BR-TIME-005 20 km/h vs 16 band speeds.
- 11 §1.10 OTP per phone 3/15 min vs 12 §2.2 5/hour, 10/day.
- 13 §6.2 rider fault "per HR policy" vs 01 BR-REF-005 "no rider deductions without due process; riders are contractors".
- 08 §8 "No server-side image pipeline" vs 17 F9 resize worker vs 01 RES-MENU-005 server WebP thumbnails. Note that Go has no stdlib WebP *encoder*, so a server pipeline needs cgo/libvips. *(proposal)* the client produces WebP variants, and the server validates and re-encodes JPEG only.
- 01 BR-DISP-001 eligibility "within 5 km" vs 08/13 radius steps 2→4→7 km.

---

## 15. Sources spot-checked (accessed 2026-10-04)

- **CloudFront flat-rate plans** — Free $0 (1M requests, 100 GB), Pro $15 (10M, 50 TB), Business $200 (125M), Premium $1,000 (500M). "No additional overage charges"; plans "may not be combined with any other offers, promotions, or discounts": [AWS CloudFront pricing](https://aws.amazon.com/cloudfront/pricing/).
- **Behaviour above the allowance** — first spike up to 3× accommodated, then evaluated over multiple months; performance may be reduced or an upgrade required: [AWS Networking blog — flat-rate plans new features](https://aws.amazon.com/blogs/networking-and-content-delivery/amazon-cloudfront-flat-rate-pricing-plans-new-features-and-expanded-capabilities); [conductatlas summary of AWS service terms](https://conductatlas.com/platform/aws-bedrock/aws-service-terms/provision/CA-P-052334/cloudfront-flat-rate-plan-overages-may-trigger-throttling/).
- **CERT-In Directions, 28 Apr 2022** — logs of all ICT systems for a rolling 180 days within Indian jurisdiction; incidents reported within 6 hours; effective 27 Jun 2022: [TaxGuru](https://taxguru.in/corporate-law/cert-in-issues-directions-relating-information-security-practices.html); [PSA Legal](https://psalegal.com/new-cert-in-directions-overview-and-implications/); [Mondaq](https://mondaq.co.uk/india/it-and-internet/1270332/6-hours-to-report-cyber-incidents-to-cert-in).
- **Not verified by the Reviewer** (marked inline): AWS Verified Access pricing, Sentry data regions, PA test-mode availability before KYC, voice-OTP/IVR per-call price, device prices.
