# 29 — Production Readiness Checklist (Go-Live)

| Field | Value |
|---|---|
| **Purpose** | The single checklist that must be satisfied, with evidence, before the **closed pilot (Gate A)** and the **public launch in Mahabubnagar (Gate B)**, plus the Go/No-Go decision procedure, rollback plan, launch-day plan and hypercare. |
| **Owner** | Release Architect (procedure); item owners as listed |
| **Status** | Draft v1 (2026-10-04) — conforms to baseline §8 (R1–R26) and §9 (R27–R48: launch gates per R47, load model per R45, cost per R46, two IaC profiles per R32) |
| **Depends on** | `01` (NFR, LEG, M-metrics), `02` §5 (product gates, amended by R47), `12`, `19` (SEC-xxx), `20` §22.3 (release gates G1–G12), `21`–`25`, `27` (stories), `28` (milestones), `31` (review findings). |
| **Feeds** | Go/No-Go meetings at Gate A and Gate B; `32-final-plan-summary.md`. |

---

## 0. How to use

- **Gate** column: **A** = required before the closed pilot (production on the `closed-pilot` profile, invited users, ≥ 10 restaurants, ≥ 10 riders, 1 zone — R47). **B** = additionally required before public launch. **A+B** = required at A and re-verified at B.
- **Evidence** is a link (doc, CI run, dashboard snapshot, signed PDF) stored in the private ops repository; the checklist row records the link and the date verified. "Verified" means within **14 days** of the gate unless the row says otherwise.
- **Owner roles:** PL product/ops lead · RA release architect · BE backend lead · FE frontend lead · DO DevOps (full-stack engineer) · QA (rotating engineer + QA owner) · SEC security owner (lead engineer) · FIN finance admin · LEGAL counsel · CA chartered accountant · OPS ops lead · SUP support lead · FDR founder(s).
- A row can be **waived** only per §15 (written, named risk owner, expiry date). Rows marked **[NO WAIVER]** cannot be waived.

---

## 1. Product & operations readiness

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| PR-01 | Supply at Gate A: ≥ 10 restaurants live (approved, menu complete, FSSAI valid, payout account verified, tested counter device) in 1 active zone (R47) | A | OPS | Supply dashboard export; device test log per outlet |
| PR-02 | Supply at Gate B: ≥ 40 restaurants live; category coverage B2 (≥ 8 biryani/non-veg, ≥ 6 tiffin/meals, ≥ 5 fast food/Chinese, ≥ 4 bakery/desserts/juices, ≥ 8 pure-veg); ≥ 10 open until ≥ 23:00; menu quality B4 (100% markers & prices, ≥ 50% photos, top-20 items with Telugu names) | B | OPS | Supply dashboard; menu-quality query output |
| PR-03 | Riders at Gate A: ≥ 10 approved, inducted, kitted, insured per decision; at Gate B: rider pool sized by demand model ≈ 1 online rider per 3 peak-hour orders (R47), demonstrated at lunch and dinner peaks on ≥ 5 pilot days | A / B | OPS | Rider roster; peak online counts from dashboard |
| PR-04 | Real zone polygon(s) and locality list (en/te, PINs) drawn and field-validated; Gate B zone coverage B7 (≥ 70% of zone area has ≥ 15 serviceable restaurants) | A / B | OPS | Zone publish audit entry; B7 query |
| PR-05 | Pilot rider minimum guarantee (`MG_TOPUP`) amount, slots and budget approved and configured (R47) | A | FDR + FIN | Signed decision; config audit |
| PR-06 | Ops desk staffed during all service hours for the 90 s manual call (R43); call script in Telugu rehearsed | A+B | OPS | Rota; rehearsal log |
| PR-07 | Ops runbooks rehearsed in staging: unaccepted order, no rider, undeliverable, payment pending/failed, refund, COD deposit, restaurant pause, zone pause/rain, PA outage → COD-only, OTP outage, incident banner | A | OPS + RA | Runbook links + rehearsal records |
| PR-08 | Training delivered (Telugu + English): restaurant partner guide, rider induction (incl. food hygiene, no personal UPI, SOS), support playbook | A | OPS | Attendance sheets; materials links |
| PR-09 | ≥ 2 named people able to approve money actions (R31); break-glass procedure with 24 h post-review documented | A **[NO WAIVER]** | FIN + RA | Admin role list; procedure doc |
| PR-10 | Finance operating calendar: weekly statements Monday, restaurant payouts by Wednesday, rider payouts by the doc-13/10-owned day (R48), daily COD deposit review; labour plan | A | FIN | Calendar; finance runbook signed off by CA |
| PR-11 | Feature flags set for launch (COD on, delivery OTP on for prepaid ≥ ₹300 (R39), heartbeat gating on, coupons, ratings display) and documented | A+B | PL | Config export with audit IDs |

## 2. Functional completeness

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| FN-01 | Golden flow **COD** passes on staging (E2E E26-S01) in en and te, 3 consecutive nightly runs | A+B **[NO WAIVER]** | QA | CI run links |
| FN-02 | Golden flow **online (PA sandbox)** passes on staging in en and te, 3 consecutive nightly runs | A+B **[NO WAIVER]** | QA | CI run links |
| FN-03 | Production smoke: one real COD order and one real UPI order placed by the team on production with real restaurant and rider, delivered, rated, and visible in the next statement | A | RA | Order codes; ledger extract |
| FN-04 | Failure scenarios F-1…F-14 (doc 20 §10.4) green on staging, with F-2 asserting **R1** (180 s → `CANCELLED`/`SYSTEM`/`RESTAURANT_UNRESPONSIVE`, 30-min pause) | A+B | QA | CI run; test report tagged by requirement |
| FN-05 | Restaurant alert ladder (R1/R43) verified on real counter devices: repeat 30 s, owner SMS 60 s, ops flag 90 s, cancel 180 s; device heartbeat loss pauses outlet in 3 min | A | QA + OPS | Device test video/log |
| FN-06 | Dispatch tiers (R34) verified on real phones: fresh ≤ 3 min ranked; stale ≤ 15 min reached by push; auto-offline at 15 min | A | QA | Field test log |
| FN-07 | All P0 stories of doc 27 done (DoD), traceability matrix shows every P0 requirement covered by ≥ 1 passing test | A+B | RA | `tools/trace` artifact |
| FN-08 | 0 open S1/S2 defects (S2 waiver only by Product + Engineering lead in writing) (G2) | A+B | QA | Issue tracker query |
| FN-09 | Maker-checker on the five R31 families only, each tested (self-approval blocked, break-glass alert) | A | BE | Integration test report |
| FN-10 | Admin interventions (accept on behalf, status on behalf, manual assign, cancel with refund/fault, ops-mediated restaurant cancel R40) work in production admin | A | OPS | Rehearsal log on prod test outlet |

## 3. Security (maps to doc 19 SEC-xxx)

| ID | Item | SEC | Gate | Owner | Evidence |
|---|---|---|---|---|---|
| SE-01 | OTP controls: CSPRNG, TTL, attempts, HMAC+pepper, constant-time, shared rate limits on `rate_limit_buckets`, bot challenge, budget breaker | SEC-001..009, SEC-013 | A | SEC | Test report; breaker drill |
| SE-02 | Sessions/tokens/CSRF incl. device-bound counter sessions (R44); no CORS on any host; `api.` not publicly routed (R27) | SEC-026..040, R27, R44 | A | SEC | Test report; external probe |
| SE-03 | Admin auth: argon2id, **mandatory TOTP for all admins** (R37), lockout, step-up, bootstrap CLI guard; WAF rate + India geo rules on admin host | SEC-014..022, SEC-024, R37 | A **[NO WAIVER]** | SEC | Test report; WAF config |
| SE-04 | Authorisation: every operation has `x-rovo-permission`; generated matrix + IDOR + city-scope suites green | SEC-041..049 | A+B **[NO WAIVER]** | SEC | CI run |
| SE-05 | Money/fraud controls: no client amounts, quote binding, coupon caps under concurrency, webhook HMAC + dedupe + fetch-to-confirm, refund ≤ captured, payouts only to verified destinations with cool-off | SEC-076..093 | A **[NO WAIVER]** | BE | Test report |
| SE-06 | KYC images only (R38), private bucket, SSE-KMS, audited streaming view; field-level encryption for bank accounts and TOTP secrets | SEC-102, SEC-103, SEC-106, SEC-107, SEC-120, R38 | A | SEC | Bucket policy scan; DB dump sample |
| SE-07 | Web content security: CSP, security headers, framing refused, XSS corpus inert, Trusted Types on admin | SEC-061..068 | A | FE | Header sweep; E2E report |
| SE-08 | Edge: WAF W1–W8 in Block mode in prod (after Count in staging); ALB reachable only from CloudFront; `/api` never cached | SEC-108, SEC-112, SEC-171 | A | DO | External probe; `X-Cache` test |
| SE-09 | Cloud guardrails: 4-account org, SCP region lock, no IAM users/keys, OIDC trust pinned per environment (fork negative test), GuardDuty on, CloudTrail to Object-Lock | SEC-172..178, SEC-182, SEC-184 | A | DO | Guardrail scan; negative-test run |
| SE-10 | Containers non-root, read-only rootfs; no SSH; ECS Exec break-glass only; DB/cache private with TLS forced; DB roles enforced (app cannot change audit/ledger rows) | SEC-115..119 | A | DO | Config evidence; role tests |
| SE-11 | Secrets only in Secrets Manager; prod refuses dev keys/fakes/testhooks; no prod data or secrets in local/staging | SEC-121, SEC-125, SEC-185 | A **[NO WAIVER]** | DO | Startup guard test; secret scan |
| SE-12 | Supply chain: SBOM, signed images, deploy by digest with verification, Actions pinned, Dependabot, no unwaived High/Critical in SCA/SAST/image scans | SEC-136..147 | A+B | DO | Release artifacts; scan reports |
| SE-13 | External penetration test on staging: 0 open High/Critical; ASVS L2 self-assessment filed (G12) | NFR-SEC-001 | A | SEC | Pentest report + retest |
| SE-14 | Incident response: IR plan approved; CERT-In 6 h and DPB templates; contacts registry; tabletop done; JWT key / webhook secret / OTP pepper rotation drilled | SEC-031, SEC-138, SEC-166..170 | A | SEC + LEGAL | Tabletop report; drill log |
| SE-15 | Accepted security descopes recorded with risk owner: hash-chained audit/WORM anchor (SEC-127/128), PDF scanning (SEC-104), WebAuthn (SEC-025, P1) | — | A | SEC | Doc 30 risk entries |

## 4. Privacy & legal [LEGAL]

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| LG-01 | Operating entity, current account, PAN/TAN | A **[NO WAIVER]** | FDR | Certificates |
| LG-02 | GST registration as ECO under §9(5) in Telangana; invoice series configured; **CA-signed golden invoices/statements** match system output (M14) | A **[NO WAIVER]** | CA + FIN | GSTIN; signed golden set; test run |
| LG-03 | FSSAI central licence for e-commerce FBO displayed; every listed restaurant has valid FSSAI verified on FoSCoS | A **[NO WAIVER]** | LEGAL + OPS | Licence; verification log |
| LG-04 | TRAI DLT: PE, header, en/te templates approved incl. whitelisted URLs/numbers; real OTP delivered on Jio, Airtel, Vi, BSNL | A **[NO WAIVER]** | PL | DLT portal screenshots; test log |
| LG-05 | Written legal opinion on money flow (split settlement vs collect-and-payout, COD restaurant share) obtained before Phase-2 week 4 and implemented (R35) | A **[NO WAIVER]** | LEGAL | Opinion letter; config |
| LG-06 | Privacy notice, terms, refund/cancellation policy, partner and rider agreements published en + te after counsel review; consent records with versions | A | LEGAL + PL | Published URLs; consent sample |
| LG-07 | Grievance officer appointed and published; legal name, address, customer care number displayed; ticket SLA timers (48 h ack, 1 month redressal) live | A **[NO WAIVER]** | LEGAL + SUP | Help-centre screenshot; SLA test |
| LG-08 | DPDP: data inventory, processor register with locations, DPAs signed (PA, SMS, AWS, Grafana Cloud), erasure map per table/bucket (M15), deletion and access flows tested, retention jobs running | A | LEGAL + BE | Register; erasure-map review; job logs |
| LG-09 | 18+ declaration in sign-up; no Aadhaar stored; no card data anywhere | A | SEC | CI scans; UI evidence |
| LG-10 | CERT-In: PoC registered; **180-day India log archive** and ≥ 1-year security events (M1/R36) | A **[NO WAIVER]** | DO + LEGAL | Retention config; archive query sample |
| LG-11 | Consumer protection: total price incl. all fees before payment, no pre-ticked paid add-ons, seller details and cancellation policy disclosed at checkout, ranking parameters disclosed | A | PL + LEGAL | Screen review signed by counsel |
| LG-12 | Gig workers: rider agreement (contractor, transparent pay, no arbitrary deductions); insurance decision implemented; registration fields + export ready (M5) | A | LEGAL + OPS | Agreement; policy; export sample |
| LG-13 | Restricted items blocklist active (alcohol/tobacco); open-source licence compliance (NOTICE, map attribution, fonts); tile provider terms confirmed | A | RA | Lint results; terms confirmation |
| LG-14 | TDS 194-O / income-tax treatment decided by CA and implemented or recorded as not applicable | B | CA | Opinion; config |
| LG-15 | All P0 LEG-* items closed or risk-accepted in writing by counsel/CA (doc 02 B16) | B **[NO WAIVER]** | LEGAL | Signed LEG register |

## 5. Payments & reconciliation

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| PY-01 | PA selected on written rates; **effective blended rate ≤ 1% target** or explicit founder acceptance (R46) | A | FDR + FIN | Quotes; decision record |
| PY-02 | PA live mode; webhooks registered to production; first real settlement to rovo's bank observed and reconciled | A **[NO WAIVER]** | FIN | Settlement report; recon run |
| PY-03 | Split settlement (if chosen): restaurant linked accounts KYC'd for every live restaurant; transfers on hold released on cycle in staging | A (if Model A) | FIN + BE | Linked-account list; staging run |
| PY-04 | Reconciliation: PA settlement import → ledger matching with `recon_exceptions` queue; nightly invariants (Σ debit = Σ credit, order = capture = ledger) clean for 7 days on staging (A) and production (B) | A / B | FIN + BE | Job logs; dashboard |
| PY-05 | Refunds: automatic (reject/timeout/no-rider/late capture) within 5 min, manual with R31 thresholds; customer sees amount, date, reference | A | BE | E2E report; prod test refund |
| PY-06 | COD controls: per-order caps (R6), cash limit gating, ageing reminders/blocks, deposit confirmation, daily COD reconciliation; COD compensation by UPI refund or coupon (R29) | A | FIN | Test report; first-week reconciliation |
| PY-07 | Settlement & payouts: weekly statements, catch-up + missed-settlement alert (M11), payout batch maker-checker, UTR capture; dry run with real bank format | A | FIN | Dry-run batch; alert test |
| PY-08 | Degraded mode: PA down → COD-only banner flag tested in staging | A | BE | Drill log |
| PY-09 | Pilot money results: 2 consecutive weekly restaurant and rider payout cycles with 0 statement-vs-payout mismatches (M-26); COD reconciliation 100% with ₹0 unexplained variance for 2 weekly closes (M-20/M-21) | B **[NO WAIVER]** | FIN | Weekly close reports |

## 6. Data & backups

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| DB-01 | RDS automated backups + PITR on; cross-Region automated backups to `ap-south-2`; deletion protection; nightly logical dump to Object-Locked bucket in `audit/backup` account | A | DO | Console/IaC evidence; dump job log |
| DB-02 | Restore drill (PITR clone or dump restore) executed **within 7 days of the gate**, with verification SQL incl. ledger balance and PostGIS; measured RTO/RPO within doc 23 targets (G5) | A+B **[NO WAIVER]** | DO | Drill record `docs/ops/drills/` |
| DB-03 | Post-restore PA reconciliation procedure run once on a staging clone (D5) | A | FIN + DO | Drill record |
| DB-04 | Migrations rehearsed on prod-shape data; expand/contract only; rollback/forward-fix plan for the release (G9) | A+B | BE | Rehearsal log |
| DB-05 | Multi-AZ enabled (`public-launch` profile) and AZ-failover test (C-3b) documented with no lost committed writes (R32) | B **[NO WAIVER]** | DO | Failover record |
| DB-06 | Retention and erasure jobs running with catch-up; first run audited | A | BE | Job audit rows |

## 7. Infrastructure

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| IN-01 | Staging and production built only from OpenTofu (`deploy/terraform`); no console drift (nightly drift check clean 7 days) | A+B | DO | Drift job runs |
| IN-02 | `closed-pilot` profile for Gate A: Single-AZ db.t4g.small, tasks in public subnets with compensating controls (SG from ALB only, egress allow-list, VPC endpoints), no NAT (R28, R32) | A | DO | Plan output; SG review |
| IN-03 | `public-launch` profile for Gate B: Multi-AZ, one NAT, private app subnets, ≥ 2 API + ≥ 2 workers (R28, R32) | B **[NO WAIVER]** | DO | Plan/apply output |
| IN-04 | CloudFront per app host with same-origin `/api/*` (R14, R27); flat-rate Pro allowance monitored with fallback plan | A | DO | Config; request dashboard |
| IN-05 | TLS everywhere; certificates auto-renewing; DNS records audited (no dangling) | A | DO | SSL scan; DNS review |
| IN-06 | Secrets Manager + KMS keys with rotation; key admins ≠ users | A | DO | IAM review |
| IN-07 | Data residency: all data stores, logs and backups in India regions; Grafana Cloud stack in `ap-south-1` | A **[NO WAIVER]** | DO | Residency inventory |
| IN-08 | Budgets and cost anomaly alerts per doc 25 §15 (R46): closed-pilot ≈ ₹14–17k/month prod + stoppable staging; public-launch ≈ ₹30k/month (ex-GST); autoscaling max caps set | A / B | DO + FIN | Budget config; alert test |
| IN-09 | CI/CD: OIDC, protected environments, signed digests, staging-pass gate, auto-rollback verify window | A | DO | Workflow runs |

## 8. Observability & on-call

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| OB-01 | Metrics, traces, logs and Faro RUM in Grafana Cloud (R36); CloudWatch last-resort alarms for worker down, DB storage, backups, security | A | DO | Dashboard links |
| OB-02 | Alerts A1–A21 configured with runbook links; **each fired once in staging** (fault injection) and reached the right channel | A | DO + OPS | Alert test log |
| OB-03 | LISTEN/NOTIFY watchdog + `pg_notification_queue_usage` alert (M12); outbox/River lag alert; missed-periodic-job alert (M11) | A | BE | Drill log |
| OB-04 | SLOs and burn-rate alerts live; pilot critical-path SLO 99.5% during service hours (pending Lead decision on NFR-AVAIL-001 99.9% 24×7, doc 30) | A / B | RA + DO | SLO dashboard |
| OB-05 | External uptime checks (card-free tool) on all four hosts; synthetic golden-flow order in staging every 15 min; read-only synthetics in prod | A | DO | Monitor list |
| OB-06 | On-call rota (primary + secondary) published for service hours; one incident channel; P1 ack ≤ 5 min tested; escalation path to founders | A **[NO WAIVER]** | RA | Rota; test page |
| OB-07 | No PII in logs/telemetry (redaction fixture tests; Faro scrubbing verified) | A | SEC | Test report |
| OB-08 | Cost-per-order dashboard (M-60/M-61) and SMS spend live | B | FIN + DO | Dashboard |

## 9. Performance

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| PF-01 | Load test on `public-launch` shape (staging scaled) at **3× design peak = 1,500 orders/h** with full lifecycle + **3,000 concurrent SSE** for 8 h through CloudFront/ALB (R45); k6 thresholds (doc 20 §12.2) met | B **[NO WAIVER]** | QA + DO | Capacity report |
| PF-02 | Pilot-shape run (`closed-pilot` profile) at ≥ 3× pilot volume with SLOs met | A | QA | k6 report |
| PF-03 | Offer timeout fires at 45 s + < 5 s under dispatch storm (K-4); OTP/coupon abuse (K-7) contained by rate limits | A | QA | k6 report |
| PF-04 | Frontend budgets: customer initial JS ≤ 170 KB gz, CSS ≤ 30 KB; Lighthouse CI on Slow 4G; LCP/INP/CLS p75 from Faro within NFR-PERF-002 after 7 days of pilot | A / B | FE | CI; RUM dashboard |
| PF-05 | API latency p95 ≤ 300 ms reads, ≤ 500 ms writes, quote+place ≤ 800 ms in production during pilot peaks | B | BE | Dashboard snapshot |

## 10. Resilience tests

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| RS-01 | Local chaos C-1…C-9 green (worker kill, API kill mid-create, DB restart, PA timeouts, webhook outage, OTP provider failure, disk, clock jump, proxy idle) | A | QA | CI chaos run |
| RS-02 | Staging: RDS reboot/failover (C-3b), task kill + rolling deploy under load (C-3c), 2 h real-edge SSE soak (C-9b) | A | DO | Test records |
| RS-03 | Degraded modes demonstrated: PA down → COD-only; SMS provider down → sessions continue + ops alert; SSE down → polling; media down → menus without images | A | QA | Drill log |
| RS-04 | Region DR: cross-Region backups restorable in `ap-south-2` and IaC plan for DR region validates (no pre-provisioning or game day before Gate B, per cut list); first region game day scheduled within 3 months after launch | B | DO | Restore-in-region record; schedule |

## 11. Mobile / PWA device matrix

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| DV-01 | Device lab owned (≥ 1 per class A–E); counter-device model used by restaurants included (M3) | A | QA | Inventory |
| DV-02 | Real-device checklist passed for classes A (entry Android Chrome), B (Samsung Internet), D (budget tablet/counter device), and E (iOS Safari ≥ 16.4, customer only) (G11) | A+B | QA | Checklist per release |
| DV-03 | PWA install, Web Push (backgrounded and screen-locked with PWA open), wake lock, audio unlock, camera capture, geolocation, offline banners verified per app | A | QA | Test videos/logs |
| DV-04 | Service-worker update flow forces reload on breaking API change; min-client-version enforced | A | FE | E2E report |
| DV-05 | Every Gate A/B restaurant's actual order-receiver device passed sound test + test order at go-live | A / B | OPS | Device register (`restaurant_devices`) |

## 12. Accessibility & i18n

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| AX-01 | Manual a11y audit of P0 flows (TalkBack on Android, keyboard on admin): 0 critical/serious open (M-35, B17) | A+B | FE | Audit report |
| AX-02 | axe checks in CI clean (no serious/critical) | A+B | FE | CI |
| AX-03 | Telugu: 100% `te` keys, `te-pending` empty, native-speaker sign-off of UI, SMS templates, push texts, invoices/statements (G7) | A **[NO WAIVER]** | PL | Sign-off record |
| AX-04 | 360 px screens in en and te without truncation (+40% expansion) on the curated set; Noto Sans Telugu conjunct rendering checked on device classes A and D | A | FE | Visual snapshots |
| AX-05 | Number, currency, date formats (`en-IN`/`te-IN`, IST 12-hour) verified on devices | A | QA | Checklist |

## 13. Support processes

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| SP-01 | Support phone line/IVR live, published, staffed all service hours with Telugu speakers (M9) | A **[NO WAIVER]** | SUP | Test call log; rota |
| SP-02 | Ticket inbox with SLA timers; canned responses en/te; grievance queue with 1-month clock; auto-acknowledgement ≤ 48 h | A | SUP | Screenshots; SLA test |
| SP-03 | Resolution policies documented: refunds, goodwill (> ₹150 needs approver), COD compensation (UPI refund or coupon, R29), undeliverable approvals (R5) | A | SUP + FIN | Policy doc |
| SP-04 | Partner support: WhatsApp group or phone channel for restaurants and riders (no PII in group), escalation to ops desk | A | OPS | Channel list |
| SP-05 | Fraud review routine: weekly review of refund claim ratios, coupon multi-accounting flags, COD strikes, rider cash ageing | B | SUP + FIN | First two review notes |

## 14. Rollback plan

| ID | Item | Gate | Owner | Evidence |
|---|---|---|---|---|
| RB-01 | App rollback: `ecs-rollback.sh` to previous task definitions + previous SPA build from versioned bucket; drilled in staging ≤ 10 min (D4) | A+B | DO | Drill record |
| RB-02 | Release with migrations: pre-deploy snapshot; expand/contract only; contract migrations ≥ 1 release after code stops using columns | A+B | BE | Release checklist |
| RB-03 | Data damage: maintenance mode flag → PITR to new instance → selective repair or cut-over → PA reconciliation for the gap (DR-2) | A | DO | Runbook rehearsal |
| RB-04 | Business rollback switches: pause zone, COD off city-wide, online payments off (COD-only), city banner, restaurant/rider bulk pause — each tested in staging | A | OPS | Drill log |
| RB-05 | Launch rollback criteria defined (§16.3) and agreed by founders before launch day | B | RA + FDR | Signed launch plan |

## 15. Go/No-Go decision procedure

**Board:** Founder (chair, final decision), Release Architect (facilitator), Product/Ops lead, Engineering lead (backend), DevOps owner, Security owner, Finance admin; Counsel and CA attend for §4/§5 items (written statements acceptable).

**Timeline per gate:**
1. **T−14 days:** RA circulates the checklist with owners; owners fill evidence links. Freeze on scope; only fixes.
2. **T−7 days:** Restore drill (DB-02) and nightly golden flows (FN-01/02) must be green from here on; pentest retest closed (A).
3. **T−3 days: dry-run review.** Every row is **Green** (done with evidence), **Amber** (done with waiver proposal) or **Red** (not done). RA publishes the list of Reds with recovery dates.
4. **T−1 day: Go/No-Go meeting** (60 min). Agenda: Reds, Ambers/waivers, open S1/S2, risk register top 10 (doc 30), staffing for the first week, weather/festival calendar.
5. **Decision rules:**
   - **No-Go** if any **[NO WAIVER]** row is not Green, any S1 is open, or any Red remains.
   - **Waiver** allowed for other rows only when: a named risk owner accepts it in writing, a compensating control exists, an expiry date (≤ 30 days after the gate) is set, and the waiver is logged in doc 30 §(c).
   - **Go** requires unanimous board agreement that remaining Ambers are acceptable; otherwise the founder decides after hearing objections, and dissent is minuted.
6. **Record:** signed decision record (date, attendees, Green/Amber/Red counts, waivers, conditions) stored with the checklist; Gate B record references the Gate A record.
7. **Re-gate:** a No-Go sets a new date ≥ 3 working days later; only failed rows are re-reviewed unless > 14 days have passed.

**Gate B additional pilot-performance criteria** (measured over the **last 14 days** of the pilot, with ≥ 300 orders — achievable at ≈ 30 orders/day per R45; if volume falls short the Lead must approve the floor explicitly):

| ID | Criterion | Target |
|---|---|---|
| PP-01 | Restaurant acceptance rate (M-01) | ≥ 90% |
| PP-02 | Time-to-accept (M-03) | p50 ≤ 90 s |
| PP-03 | Delivery time (M-06) | p50 ≤ 40 min, p90 ≤ 60 min |
| PP-04 | Cancellation rate (M-09) | ≤ 10% |
| PP-05 | Orders reaching the 90 s unaccepted mark | ≤ 5% (else E11-S06 voice escalation required, R43) |
| PP-06 | COD reconciliation / settlement accuracy | = PY-09 |
| PP-07 | Crash-free sessions (M-30) | ≥ 99.5% customer and partner apps |
| PP-08 | Open S1/S2 in golden flow | 0 |

## 16. Launch-day plan & hypercare

### 16.1 Launch day (public launch, target Monday 2027-11-08)

| Time (IST) | Action | Owner |
|---|---|---|
| T−7 d | Deploy freeze except fixes; final release tagged and on production; `public-launch` profile confirmed | RA + DO |
| T−2 d | Supply check: restaurants confirm opening; riders confirm availability for lunch and dinner; counter devices tested | OPS |
| T−1 d | Go/No-Go (Gate B) signed; comms pack approved by counsel; war-room rota | RA |
| 09:00 | War room opens (ops desk + on-call engineer + founder); dashboards on screens | RA |
| 10:00 | Remove invite-only flag; city banner; partner announcements | PL |
| 11:00–15:00 | Lunch peak watch: unaccepted orders, unassigned deliveries, payment success, SSE connections | OPS + DO |
| 15:00 | Mid-day review: incidents, fixes queued (no deploys during peaks) | RA |
| 19:00–22:30 | Dinner peak watch | OPS + DO |
| 23:45 | Day close: reconciliation run, COD collections, incident log, next-day staffing | FIN + RA |

### 16.2 Hypercare (4 weeks)

- Daily 10:00 stand-up (ops, engineering, finance) on the previous day's metrics, stuck orders, reconciliation, tickets.
- Engineering on-call doubled (primary + secondary) during service hours; deploys only 15:00–18:00 or after 23:00.
- Weekly review against doc 01 §4 targets; first two weekly payout cycles reviewed by founders.
- Exit report at day 28: SLOs, incidents, costs per order (M-60), supply health, re-baselined targets, V1.1 priorities.

### 16.3 Launch rollback / pause triggers (pre-agreed)

| Trigger | Action |
|---|---|
| Order placement errors > 5% for 15 min, or payment success < 70% for 30 min | Online payments off (COD-only) and/or app rollback (RB-01) |
| Ledger imbalance or double capture detected | Stop payouts; S1 incident; continue orders only if root cause is contained |
| Unaccepted-order rate > 15% for an hour | Pause affected restaurants; ops calls; consider zone pause |
| No riders online in zone for 20 min during service hours | Zone pause with banner until supply returns |
| Personal-data breach suspected | IR playbook; CERT-In/DPB clocks start; founders decide on full pause |
| Two P1 incidents in 24 h | Return to invite-only mode until fixes verified |
