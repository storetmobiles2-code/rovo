# 01 — Product Requirements Document (PRD)

| Field | Value |
|---|---|
| **Purpose** | Define *what* rovo V1 must do and *why*: vision, market problem, goals, success metrics, every functional requirement (with stable IDs, priority, acceptance criteria), non-functional requirements, business rules and regulatory requirements. This is the source of truth that UX (04–07), Backend (10, 11, 13, 16), Payments (14), Notifications (15), Security (12, 19), QA (20) and Release (27–30) trace back to. |
| **Owner** | Product Architect |
| **Status** | Draft v1.1 — reconciled with review (31) and rulings R1–R48, 2026-10-04 |
| **Depends on** | `00-planning-baseline.md` (vocabulary, states, roles, money/geo/i18n conventions, §4a environment & hosting strategy, commercial defaults, §8 rulings R1–R26, §9 rulings R27–R48 + accepted scope cuts C1–C20 + missing items M1–M17). Availability/RPO/RTO targets are owned by doc 23; cost by doc 25 §15; load model by doc 20 §12.1; timers by doc 13 §5. |
| **Feeds** | `02-v1-scope.md`, `03-user-personas.md`, 04–07 (journeys/workflows), 10–16 (data, API, state machines, payments, notifications, zones), 19 (threat model), 20 (testing), 27 (backlog), 29 (readiness), 30 (risks). |

**Changes in v1.1**
- Restaurant accept ladder per **R1** (repeat every 30 s, owner SMS 60 s, ops 90 s, `CANCELLED`/`SYSTEM`/`RESTAURANT_UNRESPONSIVE` at 180 s, 30-min auto-pause, device-heartbeat auto-pause at 3 min now P0): M-02, RES-ORD-004, RES-HOUR-005, BR-TIME-001…003, NOT-002/003.
- Customer grace cancel within 60 s of placement (**R2**); auto `ACCEPTED→PREPARING` after 60 s (**R3**); pickup from `PREPARING` with `restaurant_skipped_ready` (**R4**); dispatch timing (**R7**); no server cart (**R12**); address = landmark + pin required, building/street optional (**R13**); four apps/hosts (**R14**); `*_i18n` names (**R17**); fee slabs to 10 km road, `[lo,hi)` (**R18**).
- Restaurant cannot self-cancel after accept — ops-mediated (**R40**, RES-ORD-006); COD compensation = customer's choice of manual UPI refund or coupon (**R29**); no surge, rider peak bonus via ledger adjustment (**R30**); maker-checker limited to the five **R31** families; rider dispatch tiers 3 min / 15 min (**R34**); delivery OTP stored for display (**R39**); voice escalation P1 (**R43**); device-bound sessions (**R44**).
- Load (**R45** → doc 20), cost (**R46** → doc 25 §15) and other tunables (**R48**) now referenced, not restated (NFR-PERF-006, BR-COST-002, ADM-CFG-001).
- Scope cuts applied: RES-ANLY-002 → P2 (C13), ADM-COUP-004 → P2 (C16), review moderation queue removed (C14), hash-chained audit → V1.1 (C6), KYC uploads images-only (R38/C5).
- New requirements for accepted missing items: RES-ONB-008 counter-device provisioning (M3), RDR-ONB-007 gig-worker fields + ADM-RPT-005 export (M5), RDR-EARN-007 / BR-RPAY-006 minimum guarantee (M6), RES-MENU-009 ops bulk CSV import → P0 for Gate B (M7), ADM-ORD-011 ops-assisted phone ordering P1 (M8), CUS-SUPP-003 staffed support phone line (M9), X-006 device sessions (M10/R44), NOT-016 voice escalation (R43), ADM-AUTH-004 passkeys P1 (R37).
- Additional rulings §9.1: NFR-AVAIL-001 per **R49**; M-02 redefined per **R54**; payout days and cash-ageing thresholds are doc 10 `app_config` keys (**R53**: riders Monday, restaurants Tuesday, cash alert 24 h / COD block 48 h).
- Register fixes (31 §14): rows 1–5, 21, 23, 30, 31, 35, 36, 48, 49, 52, 55–57, 59–62, 69–72; M-41 → thumbs-up share; reject reason `ITEMS_OUT_OF_STOCK`; coupon codes globally unique; commission bounds 0–3000 bps; rider cancel pay per doc 13 §6.2.

**Conventions used in this document**

- Vocabulary, order statuses (`PLACED`, `ACCEPTED`, …), delivery statuses (`UNASSIGNED`, `OFFERED`, …), offer statuses and roles (`CUSTOMER`, `RESTAURANT_OWNER`, …) are exactly as defined in the baseline §2. This PRD never introduces a new status; where it needs a reason/sub-type it uses a *reason code*, to be owned by `13-order-state-machine.md`.
- Money is stated in rupees for readability; the system stores integer paise (baseline §3).
- **Priority:** **P0** = must ship in V1 (launch blocker); **P1** = should ship in V1 (launch can proceed without it with a documented workaround); **P2** = later (V1.1+; listed here so IDs are stable and the design does not block it).
- **Tags:** `[ASSUMPTION]` needs validation, `[OPEN]` needs a decision from the named owner, `[LEGAL]` needs legal/tax review before launch.
- Requirement IDs are **stable**: never renumber; deprecate with ~~strikethrough~~ and a note.
- "Configurable" means a per-city (and where stated per-zone/per-restaurant) setting editable by `ADMIN_SUPER`/`ADMIN_OPS` via admin config (ADM-CFG), audit-logged, with a default given here.
- **Parameter ownership (R48).** Every tunable has one owning doc: timers/thresholds → doc 13 §5 (`app_config` keys such as `ordering.accept_window_s`, `ordering.customer_cancel_grace_s`, `ordering.auto_preparing_after_s`, `dispatch.offer_ttl_s`, `restaurant.device_offline_pause_s`); seeds/`app_config` defaults and `approvals.thresholds` → doc 10; load model → doc 20 §12.1; cost → doc 25 §15; fee/commission/rider-pay defaults → doc 16 §6. Numbers that still appear in this PRD state *product intent*; **if they ever differ from the owning doc, the owning doc wins** and this PRD is corrected.
- **Apps (R14):** four frontend apps/hosts — `app.` (customer), `restaurant.`, `rider.`, `admin.` — each calling `/api/v1` same-origin. "Partner" remains the UI name for restaurant users and riders, not an app name.

---

## Table of contents

1. Vision
2. Problem statement — Mahabubnagar
3. Goals and non-goals
4. Success metrics and V1 targets
5. Functional requirements — Customer (CUS-*)
6. Functional requirements — Restaurant partner (RES-*)
7. Functional requirements — Delivery partner (RDR-*)
8. Functional requirements — Admin (ADM-*)
9. Cross-cutting functional requirements (NOT-*, X-*)
10. Non-functional requirements (NFR-*)
11. Business rules (BR-*)
12. Regulatory requirements (LEG-*) [LEGAL]
13. Open questions
14. Sources

---

## 1. Vision

> **rovo makes ordering food from your own town's restaurants as easy as a UPI payment — in Telugu or English, on any Android phone, with fair, transparent fees for customers, restaurants and delivery partners.**

rovo is an open-source (Apache-2.0) food-delivery platform built first for **Mahabubnagar (Palamuru), Telangana**, and designed so that any small Indian city can run it. Its differentiators versus national aggregators are not features but **fit**:

1. **Local-first:** Telugu as a first-class language, landmark-based addressing, COD that is actually reconciled, restaurant onboarding that a tiffin-centre owner can complete with help.
2. **Fair and transparent economics:** every fee shown before payment; commission configurable per restaurant from a published default; riders see exactly how each payout is computed.
3. **Light-weight:** runs on a low-end Android browser over patchy 4G; in production it runs on standard managed cloud services in an India region (baseline §4a) at a running cost per order the business can carry (BR-COST).
4. **Open:** a city operator (a local entrepreneur, co-operative or municipality-backed entity) can fork and operate it.

The V1 golden flow that must work end-to-end, every time:
`Customer → Restaurant → Order → Delivery Partner → Pickup → Delivery → Payment/Settlement → Rating`.

## 2. Problem statement — Mahabubnagar

### 2.1 Market context

| Factor | Reality in Mahabubnagar | Product consequence |
|---|---|---|
| **City size & shape** | District HQ ~100 km south of Hyderabad on NH-44; compact urban area of roughly 2–2.5 lakh people [ASSUMPTION – baseline]; most trips within 2–6 km. Mix of old town lanes, newer colonies (e.g. New Town, Padmavathi Colony, Christianpally, Yenugonda, Boyapally, Shasab Gutta) [ASSUMPTION – locality list to be compiled by ops]. | Single city, 1–3 zones at launch; straight-line distance × road factor is good enough (baseline §3); delivery promise 30–45 min is realistic. |
| **Vehicles** | Two-wheelers (100–125 cc motorcycles, scooters); summer peaks ~44–45 °C (Apr–May), monsoon rain (Jun–Sep). | Rider flows must be operable with one hand/gloves and in sunlight (high contrast, big targets); weather can slow deliveries — ETA buffer configurable. |
| **Language** | Telugu is the dominant language; English widely read by students/professionals; Urdu/Dakhni spoken by a significant minority [ASSUMPTION]. Many users read Telugu script better than English menus. | `en` + `te` from day one; menu items get optional Telugu names; language switch always reachable; Urdu as a post-V1 locale [OPEN]. |
| **Payments** | UPI (PhonePe, Google Pay, Paytm) is the default for students and professionals; cash-on-delivery still demanded by families and older users and by first-time users who do not yet trust a new brand. | UPI-first checkout with COD available within limits; COD must be reconciled to the rupee. |
| **Price sensitivity** | Typical order values ₹150–₹600; customers abandon at fee shock; national apps' platform fees/surge are a known irritant. | Fees shown up-front on listings and in the cart; small, predictable fee slabs; no surge pricing in V1. |
| **Restaurant digital maturity** | Many small family restaurants, tiffin centres, bakeries and biryani points; owner often runs the counter; one shared Android phone; menus on a wall board or printed card; prices change informally; some not GST-registered; FSSAI registration/licence varies in currency. | Assisted onboarding by rovo ops (admin can create menus on their behalf); order alert must be impossible to miss; ops calls the restaurant when an order is not acknowledged. |
| **Connectivity** | 4G generally available but patchy indoors, in kitchens and in some outskirts; frequent switching between 4G/3G; prepaid data budgets. | Small bundles, cached shell (PWA), idempotent retries, clear "offline / retrying" states; no heavy maps on customer screens. |
| **Devices** | Low-to-mid range Android (2–4 GB RAM, Android 10+ typical), often with low storage; few iPhones. | Performance budgets set for low-end Android (NFR-PERF); PWA not native apps (baseline P7). |
| **Trust** | New brand vs. known national apps; word-of-mouth and the restaurant's own recommendation matter. | Visible FSSAI number per restaurant, transparent bill, responsive local support number, refunds that actually arrive. |

### 2.2 Problems we solve

**Customers:** limited choice of local restaurants on national apps; high and opaque fees; no Telugu interface; support that cannot resolve local issues; COD sometimes unavailable.
**Restaurants:** high commissions (often 18–30% on national platforms [ASSUMPTION]); complex partner apps; delayed or opaque settlements; no help in getting online.
**Riders:** need flexible local income without moving to Hyderabad; unclear pay computation; cash-in-hand disputes.
**City operator:** needs a low-cost, auditable system they can run with a 3–6 person ops team.

## 3. Goals and non-goals

### 3.1 Goals (V1)

| # | Goal | Measured by (§4) |
|---|---|---|
| G1 | The golden flow works reliably for both **UPI/online** and **COD** orders. | Order completion rate, cancellation rate |
| G2 | Orders are delivered fast and predictably in a compact city. | Delivery time p50/p90, ETA accuracy |
| G3 | Restaurants with low digital literacy can accept and fulfil orders without missing them. | Acceptance rate, time-to-accept, unresponsive-cancel rate (M-02) |
| G4 | Money is right: every rupee (online, COD, refunds, commissions, payouts) reconciles. | COD reconciliation accuracy, settlement accuracy |
| G5 | Usable by Telugu-first and low-end-device users. | Crash-free sessions, performance budgets, accessibility audit, Telugu usage share |
| G6 | Compliant at launch with Indian consumer, food-safety, tax, privacy and telecom rules. | Legal checklist (LEG-*) closed |
| G7 | Operable by a small team on standard managed cloud (India region) at a controlled cost per order, without blocking multi-city later. | Availability/RPO/RTO targets, ops tooling coverage, M-60/M-61 cost per order |

### 3.2 Non-goals (V1) — see `02-v1-scope.md` for the full list and justifications

Multi-city operations; AI recommendations; loyalty, subscriptions, referrals; advanced analytics/BI; complex route optimisation and order batching; live GPS fleet/customer map tracking; native mobile apps; microservices; scheduled, group or multi-restaurant orders; wallets/stored value; tipping (V1.1); masked calling (V1.1); automated payouts (V1.1); grocery/pharmacy/pick-and-drop; dine-in; alcohol; surge pricing (R30); sponsored listings; WhatsApp OTP/notifications; the other accepted scope cuts C1–C20 (`02-v1-scope.md` §2.3).

## 4. Success metrics and V1 targets

Definitions are normative: Analytics/Backend must compute them exactly as defined so that dashboards (ADM-DASH, ADM-RPT) agree. "Launch" = public launch after the closed pilot. Targets apply to the period **4–12 weeks after public launch** unless stated. All targets `[ASSUMPTION]` until validated in the pilot; Release Architect owns re-baselining after pilot.

### 4.1 Operational quality (primary)

| ID | Metric | Definition | Pilot gate | V1 target |
|---|---|---|---|---|
| M-01 | **Restaurant acceptance rate** | `ACCEPTED / (orders that reached PLACED − orders cancelled by customer while PLACED)` | ≥ 90% | **≥ 95%** |
| M-02 | **Accepted within the window (R54)** | Share of orders reaching `PLACED` that the restaurant (or ops on its behalf) accepts within 180 s; system cancellations with `RESTAURANT_UNRESPONSIVE` (R1) count as misses. Miss share (`RESTAURANT_UNRESPONSIVE` / `PLACED`) reported alongside. | ≥ 90% (misses ≤ 4%) | **≥ 95% (misses ≤ 2%)** [ASSUMPTION] |
| M-03 | **Time-to-accept** | `ACCEPTED.at − PLACED.at` | p50 ≤ 90 s | **p50 ≤ 60 s, p90 ≤ 150 s** |
| M-04 | **Rider assignment time** | first `OFFERED` → `ASSIGNED` for the delivery | p90 ≤ 8 min | **p50 ≤ 90 s, p90 ≤ 5 min** |
| M-05 | **Offer acceptance rate** | `ACCEPTED offers / (ACCEPTED+DECLINED+EXPIRED offers)` | ≥ 60% | **≥ 70%** |
| M-06 | **Delivery time (customer-perceived)** | `DELIVERED.at − PLACED.at` (COD) or `− payment-confirmed PLACED.at` (online) | p50 ≤ 40, p90 ≤ 60 min | **p50 ≤ 35 min, p90 ≤ 50 min** |
| M-07 | **Last-mile time** | `DELIVERED.at − PICKED_UP.at` | — | **p50 ≤ 12 min, p90 ≤ 20 min** |
| M-08 | **ETA accuracy** | % delivered at or before the upper bound of the ETA range shown at order placement | ≥ 70% | **≥ 85%** |
| M-09 | **Cancellation rate (all causes)** | `(CANCELLED + REJECTED + UNDELIVERABLE) / orders reaching PLACED` | ≤ 10% | **≤ 6%** |
| M-10 | of which restaurant-caused | `REJECTED` + `CANCELLED` with fault attribution `restaurant` (incl. `RESTAURANT_UNRESPONSIVE` and ops-mediated `RESTAURANT_CANNOT_FULFIL`, R40) | ≤ 5% | **≤ 2.5%** |
| M-11 | of which no-rider-caused | `CANCELLED` with reason `NO_RIDER_AVAILABLE` | ≤ 3% | **≤ 1%** |
| M-12 | **Order completion rate** | `DELIVERED / orders reaching PLACED` | ≥ 88% | **≥ 93%** |

### 4.2 Money correctness

| ID | Metric | Definition | V1 target |
|---|---|---|---|
| M-20 | **COD reconciliation accuracy** | % of `DELIVERED` COD orders with a matching rider cash-collected ledger entry equal to the order's `total_payable` | **100%, same day** |
| M-21 | COD variance | Sum of unexplained (no reason-coded adjustment) differences between expected and deposited/netted rider cash at weekly close | **₹0** |
| M-22 | Rider cash ageing | % of riders with cash-in-hand older than the ageing-alert threshold (`app_config`, doc 10; default 24 h, R53) | **≤ 5%** |
| M-23 | **Online payment success rate** | `payments captured / payment attempts initiated` (excludes attempts the user closed before choosing a method) | **≥ 92%** |
| M-24 | Orphan payments | Captured payments with no `PLACED` order (late webhook, failed order) auto-refunded within 24 h | **100%** |
| M-25 | Refund turnaround | Refund decision → refund initiated with PA | **p95 ≤ 24 h** |
| M-26 | **Settlement accuracy** | Restaurant/rider payouts whose amount equals the statement amount | **100%**; payout disputes ≤ 2% of payee-weeks |

### 4.3 App quality and experience

| ID | Metric | Definition | V1 target |
|---|---|---|---|
| M-30 | **Crash-free sessions** | % of sessions (per app: customer, restaurant, rider, admin — R14) without a fatal client error (unhandled exception or error boundary render that blocks the current flow) as reported to frontend error tracking (Grafana Faro, R36) | **≥ 99.5%** customer, restaurant & rider; ≥ 99% admin |
| M-31 | Order placement error rate | Checkout submissions returning 5xx or failing client-side after retries | **≤ 0.5%** |
| M-32 | Core Web Vitals (customer app, field data, p75, mobile) | LCP / INP / CLS | **LCP ≤ 2.5 s, INP ≤ 200 ms, CLS ≤ 0.1** |
| M-33 | Availability (production) | See NFR-AVAIL-001/002 | **≥ 99.9% ordering critical path; ≥ 99.5% other surfaces** (monthly) |
| M-34 | Telugu usage share | % sessions with `te` locale | tracked, no target (informs content investment) |
| M-35 | Accessibility | Automated (axe) + manual audit of P0 flows | **0 critical / 0 serious issues** |

### 4.4 Satisfaction and support

| ID | Metric | Definition | V1 target |
|---|---|---|---|
| M-40 | Restaurant rating (platform avg) | mean of restaurant ratings, last 30 days | **≥ 4.0 / 5** |
| M-41 | Delivery rating (platform) | share of delivery ratings that are 👍 (thumbs + tags, CUS-RATE-001), last 30 days | **≥ 90% thumbs-up** |
| M-42 | Rating submission rate | rated orders / delivered orders | **≥ 25%** |
| M-43 | Support first response | ticket created → first human response, in service hours | **p90 ≤ 10 min** — measured in the pilot, not a launch gate; depends on the staffed support line (M9, CUS-SUPP-003) |
| M-44 | Grievance acknowledgement | complaints acknowledged within 48 h (LEG-CP-003) | **100%** |
| M-45 | Ticket resolution | created → resolved | **p90 ≤ 72 h**; 100% ≤ 30 days (legal) |
| M-46 | Order-issue rate | orders with a ticket of category missing/wrong/quality | **≤ 3%** |

### 4.5 Growth and supply health (directional; not launch gates)

| ID | Metric | V1 target `[ASSUMPTION]` |
|---|---|---|
| M-50 | Orders/day | Closed pilot ≈ 30; Month 1 avg ≥ 80; Month 3 avg ≥ 250 (planning volumes per R45; doc 20 §12.1 owns the load model) |
| M-51 | 30-day customer repeat rate | ≥ 35% of first-time customers order again within 30 days |
| M-52 | Sign-up → first order conversion | ≥ 40% |
| M-53 | Weekly active restaurants | ≥ 85% of live restaurants receive ≥ 1 order/week |
| M-54 | Rider earnings per online hour (median) | ≥ ₹100/h at peaks (rider retention lever); pilot floor via minimum guarantee (RDR-EARN-007, M6) |
| M-55 | Rider 30-day retention | ≥ 60% of approved riders active in their 5th week |
| M-56 | Item-unavailability rejections | `REJECTED` with `ITEMS_OUT_OF_STOCK` (doc 13 §6.3) / `PLACED` ≤ 1% |

### 4.6 Running cost (business line item)

Production runs on paid managed cloud (baseline §4a), so infrastructure is a real cost of goods sold, tracked like rider pay. Definitions in BR-COST; **budget figures are owned by doc 25 §15 (R46), quoted ex-GST**, with a `closed-pilot` and a `public-launch` infrastructure profile (R32). Pilot cloud cost per order is expected to exceed the M-60 target until volume reaches month-3 levels (31 §2.1); M-60 is therefore a month-3 target, not a pilot gate.

| ID | Metric | Definition | V1 target `[ASSUMPTION]` |
|---|---|---|---|
| M-60 | **Cloud cost per delivered order** | (production + staging cloud bill for the month, incl. DB, compute, CDN, storage, logs, WAF, egress, backups) / delivered orders in the month | **≤ ₹6 by month 3** (≈ 7,500 orders/month); ≤ ₹3 at 1,000 orders/day |
| M-61 | **Total tech opex per delivered order** | M-60 + SMS/OTP + PA fees not recovered from customers + error tracking/observability SaaS + telephony + maps/tiles + domains/email | **≤ ₹10 by month 3** |
| M-62 | Cloud budget adherence | Actual monthly cloud spend vs approved budget (doc 25 §15) | ≤ 100%; alerts at 50/80/100% (doc 22/24) |

---

## 5. Functional requirements — Customer (CUS-*)

Customer app = `customer` PWA served at the `app.` host (baseline P7, R14). Role `CUSTOMER`.

### 5.1 CUS-AUTH — Registration, login, profile, account

- **CUS-AUTH-001 (P0) Phone + OTP sign-in/sign-up.** A single flow handles new and returning users: enter Indian mobile → receive 6-digit OTP by SMS → verify → signed in. New users are created on first successful verification.
  - AC: *Given* a valid Indian mobile (10 digits, starts 6–9) *when* the user requests an OTP *then* an SMS is sent via a DLT-registered template within 30 s p95 and the screen shows a resend timer (30 s).
  - AC: OTP valid 5 min, max 5 verification attempts per OTP, max 5 OTP sends per phone per hour and 10 per day; exceeding shows a localised, non-technical message (rate limits owned by doc 12).
  - AC: Web OTP API / SMS auto-read used where the browser supports it; manual entry always works.
  - AC: Phone stored in E.164; leading `0`/`+91` pasted forms are normalised.
- **CUS-AUTH-002 (P0) Minimal profile.** After first verification ask only for **name** (required) and **email** (optional, for receipts). No date of birth, gender, etc.
- **CUS-AUTH-003 (P0) Consent & notices at sign-up.** Show privacy notice (en/te) and terms; a separate, unticked opt-in for marketing messages; user confirms they are 18+ (see LEG-DPDP-006). Notice, consent and 18+ declaration are presented on **one combined screen** to keep first checkout short (RV-057). Consent record (version, timestamp, channel) stored.
- **CUS-AUTH-004 (P0) Language choice.** On first launch, default to device language if `te`/`en`, else `en`; a language switcher is reachable from the header/profile on every screen; choice persists on the account and device.
- **CUS-AUTH-005 (P0) Session persistence.** Stay signed in on the device (refresh-token rotation per baseline P9) for up to 30 days of inactivity [ASSUMPTION – doc 12 owns]; sign-out available; "sign out of all devices" (P1).
- **CUS-AUTH-006 (P0) Browse before login.** Discovery, menus and cart work without login; login is required at checkout (cart preserved through login).
- **CUS-AUTH-007 (P0) Account deletion.** User can request deletion in-app; confirmation by OTP; account deactivated immediately, PII erased/anonymised within 30 days except data under legal retention (NFR-RET); blocked while an order is in progress.
- **CUS-AUTH-008 (P1) Change phone number.** Verify new number by OTP; old number receives notification.
- **CUS-AUTH-009 (P1) Data access request.** User can download a summary of their personal data (profile, addresses, orders) as JSON/CSV or request it via support (DPDP right to information).
- **CUS-AUTH-010 (P2) WhatsApp OTP channel** as alternative to SMS (deferred to V1.1+, C2; V1 fallback is a secondary SMS aggregator).

### 5.2 CUS-ADDR — Location and addresses

- **CUS-ADDR-001 (P0) Set delivery location on first use.** Options: "Use my current location" (browser geolocation, permission prompt explained beforehand) or "Search locality" (from seeded `locality` list) or "Pick on map".
  - AC: *Given* location permission is denied *then* the user can still proceed by choosing a locality and then dropping a pin.
- **CUS-ADDR-002 (P0) Address form (India) — R13.** Fields per baseline §3: house/flat no. (required), building/street (**optional**), locality (required; pick from list, "Other" allowed with free text), **landmark (required)**, PIN (6 digits, prefilled from locality), label (Home/Work/Other), contact name/phone override (optional, "ordering for someone else"), **map pin (required)**.
  - AC: An address cannot be saved without lat/lng from a pin; pin defaults to geolocation or locality centroid; the user must confirm the pin ("Move the pin to your gate").
  - AC: There is **no pinless address path**. If map tiles fail, the user can set the pin with "Use my current location" (GPS fix) or on a lightweight fallback (static locality image); a locality centroid alone is never saved as a delivery pin (register row 30).
- **CUS-ADDR-003 (P0) Serviceability check.** On selecting/saving an address, show whether it is serviceable (inside an active zone). Non-serviceable → clear message "We don't deliver here yet" + option to register interest (stored, P1).
- **CUS-ADDR-004 (P0) Saved addresses.** List, add, edit, delete, set default. Max 10 per user. Editing an address does not alter past orders (orders snapshot the address).
- **CUS-ADDR-005 (P0) Delivery instructions.** Optional free text per address (≤ 200 chars, e.g. "Call on arrival, 2nd floor, blue gate") shown to the rider.
- **CUS-ADDR-006 (P1) Locality-level browsing without a pin** (show restaurants for the locality centroid; pin still required at checkout).
- **CUS-ADDR-007 (P2) Reverse geocoding / address autocomplete** (deferred: no paid geocoding in V1, baseline P13).

### 5.3 CUS-DISC — Restaurant discovery

- **CUS-DISC-001 (P0) Home listing.** For the selected location, list restaurants that are approved, not suspended, serviceable to the location (zone + restaurant max radius, baseline §3), ordered: open first, then by a simple deterministic rank (distance asc, then rating desc, then recent order volume) [ASSUMPTION – no personalisation in V1].
- **CUS-DISC-002 (P0) Restaurant card.** Shows name (en/te), cuisine tags, rating (or "New" if < 5 ratings), ETA range, **delivery fee for this location**, distance, pure-veg badge if all items are veg, offer badge if a restaurant-scoped coupon is displayable, and open/closed/paused state with next opening time.
  - AC: The delivery fee shown on the card equals the fee the cart will show for the same address (fee transparency, LEG-CP-005).
- **CUS-DISC-003 (P0) Closed/paused restaurants.** Shown greyed below open ones with "Opens at 6:00 PM" / "Not accepting orders right now"; menu browsable; add-to-cart disabled.
- **CUS-DISC-004 (P0) Platform service hours.** Outside city service hours (configurable, default 08:00–23:30 IST [ASSUMPTION]) the app shows a banner and disables ordering.
- **CUS-DISC-005 (P1) Curated collections.** Admin-defined lists ("Biryani", "Tiffins", "Under ₹150", "Pure veg") on home (ADM-CONT-001).
- **CUS-DISC-006 (P1) Promotional banners** managed by admin (image + target collection/restaurant/coupon), max 5.
- **CUS-DISC-007 (P2) Personalised ranking / "recommended for you"** — deferred (AI recommendations are a non-goal).

### 5.4 CUS-SRCH — Search, filters, categories

- **CUS-SRCH-001 (P0) Search.** One search box across restaurant names, cuisine tags and dish names (en and te), within the serviceable set; results grouped "Restaurants" / "Dishes" (dish result links to the restaurant menu scrolled to the item).
  - AC: Prefix and typo-tolerant matching (e.g. "biriyani", "biryani", "బిర్యానీ" all find biryani dishes) — trigram similarity in Postgres [ASSUMPTION – Backend confirms].
  - AC: Results return p95 ≤ 500 ms server-side.
- **CUS-SRCH-002 (P0) Filters.** Pure veg; rating 4.0+; delivery time ≤ 30 min; offers available; cost for one (bands); open now. Filters combinable, persisted for the session.
- **CUS-SRCH-003 (P0) Sort.** Relevance (default), delivery time, rating, distance, cost low→high.
- **CUS-SRCH-004 (P0) Cuisine/category chips** on home (admin-managed list with icons, en/te labels).
- **CUS-SRCH-005 (P1) Recent searches** (device-local) and popular searches (admin-curated).
- **CUS-SRCH-006 (P2) Transliteration search** (Latin-typed Telugu, e.g. "pulihora").

### 5.5 CUS-MENU — Restaurant and menu page

- **CUS-MENU-001 (P0) Restaurant header.** Name, cuisines, locality, rating + count, ETA range, delivery fee, cost for two, open hours today, **FSSAI licence/registration number** (LEG-FSSAI-002), pure-veg badge.
- **CUS-MENU-002 (P0) Menu.** Categories in restaurant-defined order with sticky category navigator; each item shows name (te shown under en, or vice versa per locale, when a Telugu name exists — stored in `name_i18n` JSONB, exposed as `nameI18n` + resolved `displayName`, R17), description, price, **dietary marker (VEG / EGG / NON_VEG)** per BR-MENU-001, image if present, "Customisable" hint, bestseller tag (P1), out-of-stock state.
- **CUS-MENU-003 (P0) Veg-only toggle** on the menu page.
- **CUS-MENU-004 (P0) Out-of-stock items** are shown greyed with "Not available now" (not hidden) and cannot be added; items in hidden categories or outside their availability window are not shown.
- **CUS-MENU-005 (P0) Price display.** Item prices displayed as entered by the restaurant, which are **exclusive of GST** (BR-FEE-007); the menu header carries a one-line note "Prices exclude 5% GST; packaging charges may apply".
- **CUS-MENU-006 (P1) In-menu search.**
- **CUS-MENU-007 (P1) Item images lazy-loaded**, responsive sizes, placeholder for items without images (no stock food photos implying the dish).
- **CUS-MENU-008 (P2) Item-level discount (strike-through price).**

### 5.6 CUS-CUST — Item customisation

- **CUS-CUST-001 (P0) Variants.** An item may have one variant group (e.g. Half/Full, Regular/Family pack) with exactly one required selection; each variant has its own absolute price.
- **CUS-CUST-002 (P0) Add-on groups.** An item may have 0..n add-on groups (e.g. "Extra", "Choose your gravy") each with `min_select`, `max_select` (0 ≤ min ≤ max ≤ options), options with price (≥ ₹0) and dietary marker.
  - AC: *Given* group "Choose 1 gravy" min=1 max=1 *when* the user taps Add without selecting *then* the button stays disabled and the group shows "Required".
  - AC: *Given* max=3 *when* 3 options are selected *then* other options are disabled until one is deselected.
  - AC: The running item price updates on every selection.
- **CUS-CUST-003 (P0) Dietary integrity.** If an add-on is NON_VEG/EGG, a veg item with that add-on displays the stricter marker in cart and to the restaurant.
- **CUS-CUST-004 (P0) Customised lines.** Same item with different customisations = separate cart lines; identical customisation increments quantity. Quantity per line 1..20 [ASSUMPTION].
- **CUS-CUST-005 (P0) "Repeat last / choose new"** when tapping + on an item already in cart with customisations.
- **CUS-CUST-006 (P1) Special cooking request** per order (≤ 140 chars, e.g. "less spicy"); not per item in V1. Restaurant may ignore; shown as non-binding.

### 5.7 CUS-CART — Cart

- **CUS-CART-001 (P0) Single-restaurant cart rule.** A cart holds items from exactly one restaurant.
  - AC: *Given* a cart with items from Restaurant A *when* the user adds an item from Restaurant B *then* a dialog "Replace cart? Your cart has items from A" offers **Replace** (clears and adds) or **Cancel**; no silent replacement.
- **CUS-CART-002 (P0) Persistence.** Cart persists **on the device only** (survives reload/offline); there is **no server-side cart** (R12). Price/availability are re-validated on open and at checkout through `POST /api/v1/cart/quote`.
- **CUS-CART-003 (P0) Bill preview.** Cart shows the full bill breakdown per BR-FEE-008 (item total, packaging, GST on food, delivery fee, platform fee, small-cart fee if any, discount, round-off, **To pay**) for the selected address before payment.
- **CUS-CART-004 (P0) Change detection.** If an item became unavailable or a price/fee changed since being added, the cart flags the line ("Price updated from ₹120 to ₹130" / "No longer available — remove") and blocks checkout until acknowledged.
- **CUS-CART-005 (P0) Small-cart nudge.** If subtotal < small-cart threshold, show "Add ₹X more to avoid ₹15 small-cart fee".
- **CUS-CART-006 (P0) Restaurant closing guard.** If the restaurant closes/pauses while items are in cart, checkout is blocked with a clear message.
- **CUS-CART-007 (P1) Suggested add-ons** ("Add a drink?") — restaurant-chosen items, max 4, no algorithmic recommendation.

### 5.8 CUS-CHK — Checkout

- **CUS-CHK-001 (P0) Checkout screen** = address selector (serviceable addresses only), delivery instructions, coupon field, payment method selector, full bill, cancellation/refund policy summary link (LEG-CP-004), and a single "Place order"/"Pay ₹X" button whose amount equals **To pay**.
- **CUS-CHK-002 (P0) Server-side quote (R12).** The client never computes authoritative prices; `POST /api/v1/cart/quote` returns a priced quote with a signed `quoteId` (stored quote, 10-min validity); order placement needs `quoteId` + `Idempotency-Key`; a stale or changed quote is rejected with `409` and a machine-readable diff for user confirmation.
- **CUS-CHK-003 (P0) Idempotent placement.** Order placement carries a client idempotency key; retries after network loss never create duplicate orders or duplicate charges.
  - AC: *Given* the user taps "Place order" and the connection drops *when* the app retries (automatically or on tap) *then* exactly one order exists and the user lands on its tracking page.
- **CUS-CHK-004 (P0) Pre-placement validation** (server): restaurant open & accepting; platform within service hours; address serviceable for this restaurant; items available; quantities within limits; coupon valid; COD eligibility (BR-COD-*); order total ≥ ₹1.
- **CUS-CHK-005 (P0) ETA shown before placing** as a range (e.g. "30–40 min"), computed per BR-TIME-005.
- **CUS-CHK-006 (P0) Order code.** On placement, show the human order code (`RV-XXXXXX`).
- **CUS-CHK-007 (P1) "Ordering for someone else"** — recipient name + phone for this order (rider sees recipient).
- **CUS-CHK-008 (P2) Scheduled delivery time** — out of scope V1.

### 5.9 CUS-PAY — Payments

- **CUS-PAY-001 (P0) Online payment via the payment aggregator (PA)** using the PA's hosted/standard checkout; methods enabled: **UPI (intent/collect/QR)** first, then cards, net banking (PA-dependent). UPI is pre-selected.
  - AC: Order is created in `PENDING_PAYMENT`; moves to `PLACED` only on server-verified capture (webhook or verified status poll), never on a client redirect alone.
  - AC: If the user returns without completing, the order stays `PENDING_PAYMENT` and the app offers "Retry payment" / "Switch to Cash on Delivery" (if COD eligible) for up to 15 min, after which it becomes `PAYMENT_FAILED` (BR-PAY-001).
- **CUS-PAY-002 (P0) Cash on Delivery.** Selectable when eligible (BR-COD-001..005); order goes directly to `PLACED`. Checkout explains "Pay ₹X in cash to the delivery partner. Please keep change ready".
- **CUS-PAY-003 (P0) Payment status clarity.** Distinct screens for *paid*, *pending confirmation* (spinner + "we're confirming with your bank, don't pay again"), and *failed* (no money debited / money debited will be auto-refunded within N days).
- **CUS-PAY-004 (P0) Late capture safety.** A payment captured after the order was failed/cancelled is auto-refunded (M-24) and the customer is notified.
- **CUS-PAY-005 (P0) Receipt / tax invoice.** After delivery, an itemised receipt/tax invoice is available in order details (LEG-GST-004) and emailed if email is on file (P1).
- **CUS-PAY-006 (P1) Switch payment method on retry** (online → COD) without rebuilding the cart.
- **CUS-PAY-007 (P2) Pay-on-delivery via dynamic UPI QR** shown on rider's phone — V1.1 candidate (reduces cash handling).

### 5.10 CUS-TRK — Order tracking and status

- **CUS-TRK-001 (P0) Live status timeline.** Milestones mapped from canonical statuses: *Order placed* (`PLACED`), *Restaurant accepted* (`ACCEPTED`), *Being prepared* (`PREPARING`), *Ready, delivery partner on the way to restaurant / at restaurant* (`READY_FOR_PICKUP` + delivery `ASSIGNED`/`AT_RESTAURANT`), *Picked up — on the way* (`PICKED_UP`), *Arriving* (delivery `AT_DROP`), *Delivered* (`DELIVERED`). Side states show reason and next steps (refund status).
  - AC: Status changes reach an open tracking page within 5 s p95 (SSE; polling fallback every 15 s, baseline P3).
- **CUS-TRK-002 (P0) ETA updates.** ETA range recalculated at each milestone; if the order is running late by > 10 min vs the original upper bound, show an apology line and a "Get help" entry.
- **CUS-TRK-003 (P0) Rider details.** Once a rider is `ASSIGNED`: first name, vehicle type, photo (if provided), rating band not shown. Phone call button shown from `PICKED_UP` until `DELIVERED` + 15 min (BR-CONT-001).
- **CUS-TRK-004 (P0) Delivery OTP display** (when BR-OTP applies — prepaid ≥ ₹300, R39): 4-digit code visible on the tracking page and in the order push notification; the server stores the code (encrypted) so the app can re-display it.
- **CUS-TRK-005 (P0) Cancel button** visible only while cancellation is self-service — `PLACED`, or within 60 s of placement even if `ACCEPTED`/`PREPARING` (R2, BR-CAN-001) — with the refund consequence stated.
- **CUS-TRK-006 (P0) Background notifications** via Web Push (if permitted) for: accepted, rejected, picked up, delivered, cancelled, refund initiated (see NOT-*).
- **CUS-TRK-007 (P2) Live rider map** — deferred (baseline P12).

### 5.11 CUS-HIST — Order history and reorder

- **CUS-HIST-001 (P0) Order list** (most recent first, paginated): restaurant, date/time, items summary, total, status, rating state.
- **CUS-HIST-002 (P0) Order details**: snapshot of items/customisations, address, bill, payment method & status, refund status/reference, timeline, invoice download, "Get help".
- **CUS-HIST-003 (P0) Reorder.** "Reorder" re-adds the same items/customisations to a cart for the same restaurant, re-priced at current prices, flagging unavailable items (CUS-CART-004); prompts cart replacement if needed (CUS-CART-001).
- **CUS-HIST-004 (P1) Favourite restaurants.**

### 5.12 CUS-RATE — Ratings and reviews

- **CUS-RATE-001 (P0) Separate ratings.** After `DELIVERED`, customer can rate **restaurant/food** (1–5 stars + optional tags + optional text ≤ 500 chars) and **delivery** (👍/👎 + optional tags, e.g. "Polite", "Late", "Food spilled", "Asked for extra cash") independently; either may be skipped. (Delivery rating is thumbs + tags, matching docs 04/10/11; M-41 uses thumbs-up share.)
- **CUS-RATE-002 (P0) Rating window** — BR-RATE-001 (7 days). One rating per order per target; editable until the window closes.
- **CUS-RATE-003 (P0) Prompt.** Rating prompt on next app open after delivery (dismissible, not blocking), and within order details.
- **CUS-RATE-004 (P0) Low-rating follow-up.** Rating ≤ 2 with tag "Missing item"/"Wrong item"/"Quality" offers "Report a problem" (CUS-SUPP-001) prefilled.
- **CUS-RATE-005 (P1) Public text reviews** on the restaurant page (latest 20), shown after the automatic filter in BR-RATE-004 with admin hide/unhide; **no moderation queue and no review replies in V1** (C14).
- **CUS-RATE-006 (P2) Restaurant replies to reviews; photo reviews; dish-level ratings.**

### 5.13 CUS-COUP — Coupons and offers

- **CUS-COUP-001 (P0) Apply a coupon** by code or from "Available offers" list at checkout; **at most one coupon per order** (BR-COUP-001).
  - AC: *Given* a coupon is applied *when* the user applies another *then* the new one replaces the old after confirmation.
  - AC: Invalid coupons show the specific reason (min order not met by ₹X, expired, already used, not valid at this restaurant, not valid for COD, etc.).
- **CUS-COUP-002 (P0) Offer list** shows eligible and ineligible-with-reason coupons for the current cart, with terms.
- **CUS-COUP-003 (P0) Discount line** in bill; savings summary on success screen.
- **CUS-COUP-004 (P1) Best-offer hint** ("Apply WELCOME50 to save ₹50") — deterministic best value, no ML.
- **CUS-COUP-005 (P2) Auto-applied offers, referral codes, loyalty points** — deferred.

### 5.14 CUS-SUPP — Support and help for an order

- **CUS-SUPP-001 (P0) Order help.** From any order (active or past ≤ 7 days), "Get help" offers categories: Order is late; Cancel my order; Missing item(s); Wrong item(s); Food quality/spilled; Payment issue (charged but no order / refund not received); Delivery partner behaviour; Other. Creates a ticket (ADM-TKT) linked to the order.
  - AC: Missing/wrong/quality categories require selecting affected items and allow 1–3 photos (client-compressed ≤ 300 KB each); must be raised within 24 h of delivery [ASSUMPTION].
  - AC: Creation shows a ticket number and the acknowledgement promise; auto-acknowledgement notification sent immediately (LEG-CP-003).
- **CUS-SUPP-002 (P0) Ticket status** visible in app (Open / In progress / Awaiting your reply / Resolved) with agent replies (async messages, not live chat). Customer-facing labels map onto the canonical ticket statuses of doc 10 (`OPEN`, `IN_PROGRESS`, `AWAITING_REQUESTER`, `AWAITING_APPROVAL`, `RESOLVED`, `CLOSED`, `REOPENED`).
- **CUS-SUPP-003 (P0) Help centre.** Static FAQs (en/te) for fees, refunds, cancellation, COD, privacy; **support phone line** (tap-to-call; a business number/IVR staffed by Telugu- and English-speaking agents across all service hours in ≥ 2 shifts, call-recording consent announced — M9) and service hours; **grievance officer name, designation and contact** (LEG-CP-002).
- **CUS-SUPP-004 (P1) Customer can reopen** a resolved ticket within 7 days.
- **CUS-SUPP-005 (P2) Live chat / WhatsApp support integration.**

---

## 6. Functional requirements — Restaurant partner (RES-*)

Restaurant UI is the `restaurant` app at the `restaurant.` host (R14). Roles `RESTAURANT_OWNER` (full) and `RESTAURANT_STAFF` (orders + availability only). One `restaurant` row = one physical outlet; an owner may own several outlets (multi-outlet switcher P1).

### 6.1 RES-ONB — Onboarding and approval

- **RES-ONB-001 (P0) Self-registration.** Owner signs up with phone OTP, enters restaurant basics (name, address with pin, locality, cuisines, contact numbers, owner name) and starts an application in `DRAFT`.
- **RES-ONB-002 (P0) Assisted onboarding.** `ADMIN_OPS` can create the restaurant, owner account and full menu on the owner's behalf (field-sales mode), with the owner later confirming by OTP login. Every assisted edit is audit-logged with the admin identity.
- **RES-ONB-003 (P0) KYC documents.**
  | Document | Required | Captured |
  |---|---|---|
  | FSSAI licence/registration | **Yes** | 14-digit number, type (registration/state/central licence), expiry date, certificate image |
  | PAN (owner or entity) | **Yes** | PAN number, name as per PAN, image |
  | Bank account | **Yes** (or UPI ID, see below) | Account holder name, account no. (entered twice), IFSC, cancelled cheque/passbook image |
  | UPI ID for payouts | Optional alternative | VPA |
  | GSTIN | Optional | 15-char GSTIN, legal name (validated format; PAN embedded in GSTIN must match PAN if both given) |
  | Shop photo (front) | Yes | image |
  | Signed agreement | **Yes** | click-accept of partner terms incl. commission %, timestamp, IP, version |
  - AC: Uploads accept **images only** (JPEG/PNG/WebP; the client converts PDFs/photos to images and compresses them), re-encoded by the server, SSE-KMS at rest, audited streaming view via short-TTL signed URLs (R38; no PDF/ClamAV path, C5); documents stored privately (never public URLs), visible only to the owner and admins with KYC permission.
  - AC: FSSAI number format validated (14 digits); expired FSSAI blocks submission.
- **RES-ONB-004 (P0) Application status.** `DRAFT → SUBMITTED → UNDER_REVIEW → APPROVED | CHANGES_REQUESTED | REJECTED`; owner sees status and admin comments; notified on change. (Status names here are onboarding-application statuses, not order statuses; doc 10 owns the enum.)
- **RES-ONB-005 (P0) Go-live checklist.** A restaurant can go live only when: approved, ≥ 1 category with ≥ 5 available items [ASSUMPTION], operating hours set, prep time set, commission/terms effective, payout account verified (manually in V1), **an order-receiver device registered and its sound/push test passed** (RES-ONB-008), and — only if the PA split-settlement model is chosen (R25/R35) — **PA linked-account KYC complete** [LEGAL].
- **RES-ONB-006 (P1) Bank account verification by penny-drop** via PA API.
- **RES-ONB-007 (P1) FSSAI expiry reminders** at 30/15/7 days; on expiry the restaurant is automatically paused (BR-RES-004).
- **RES-ONB-008 (P0) Counter-device provisioning (M3).** Restaurants without a suitable dedicated device get a **pre-configured budget Android counter device** from rovo ops (app installed, notifications and sound tested, battery-optimisation exemptions set, registered as order-receiver with a device-bound session per X-006). Ops records the device (model, serial) against the outlet and can revoke it. Device and device-lab budget owned by docs 25/29.

### 6.2 RES-PROF — Profile

- **RES-PROF-001 (P0)** Edit display name (en/te), description, cuisines (from admin list), cost for two, contact numbers, cover image, logo. Changes to legal/KYC fields (PAN, bank, FSSAI, GSTIN, address/pin) require admin re-approval and do not take effect until approved.
- **RES-PROF-002 (P0)** Max delivery radius (≤ city max, default 7 km) and pure-veg flag (if set, system blocks adding EGG/NON_VEG items).
- **RES-PROF-003 (P0)** Packaging charge mode: per item (set on each item) **or** per order (flat), per BR-FEE-004.
- **RES-PROF-004 (P1)** Multiple users: owner adds staff by phone number (`RESTAURANT_STAFF`), removes them; multiple simultaneous devices for the same account are allowed in P0 (owner phone + counter phone both ring).

### 6.3 RES-HOUR — Operating hours, holidays, temporary pause

- **RES-HOUR-001 (P0) Weekly hours.** Per day of week, 0..3 time slots (e.g. 07:00–11:00, 12:00–15:30, 18:30–23:00) in local wall-clock time (baseline §3); slots may cross midnight (e.g. 19:00–01:00).
- **RES-HOUR-002 (P0) Holidays/closures.** Date (or date range) closures with optional note ("Closed for Bathukamma").
- **RES-HOUR-003 (P0) Temporary pause** ("Stop taking orders") for 15 / 30 / 60 min / until next slot / until I resume; auto-resumes; status visible to admin.
  - AC: *Given* restaurant pauses for 30 min *then* it immediately disappears from orderable state for customers (closed card), existing in-progress orders are unaffected, and it auto-resumes at T+30 min with a notification to the restaurant.
- **RES-HOUR-004 (P0) Last-order buffer.** Ordering closes N minutes before slot end (default 15, configurable per restaurant).
- **RES-HOUR-005 (P0) Effective open state** = within a slot AND not holiday AND not paused AND not suspended AND FSSAI valid AND platform within service hours AND an order-receiver device alive. **No restaurant-device heartbeat for 3 min while open → auto-pause** (`DEVICE_OFFLINE`, R1; threshold `restaurant.device_offline_pause_s`, doc 13 T-DEVICE-HB). Heartbeat every 60 s; an open SSE connection counts as presence (R27).

### 6.4 RES-MENU — Menu management

- **RES-MENU-001 (P0) Categories.** Create/rename/reorder/hide categories (en + optional te name). Optional category time window (e.g. "Breakfast 07:00–11:00").
- **RES-MENU-002 (P0) Items.** Fields: name (en, required), name (te, optional), description (en/te optional, ≤ 300 chars), category, base price (₹, > 0) **or** variants, dietary marker (VEG/EGG/NON_VEG, required), image (optional), packaging charge (if per-item mode), bestseller flag (P1), display order, "serves" (P2), tax category fixed to restaurant service 5% in V1 (BR-FEE-007).
- **RES-MENU-003 (P0) Variants.** One variant group per item, 2..10 variants, each with name (en/te) and absolute price; exactly one default.
- **RES-MENU-004 (P0) Add-on groups.** Reusable add-on groups (name, min, max, options with price ≥ 0 and dietary marker) attachable to many items; validation `0 ≤ min ≤ max ≤ #options`, max 10 groups per item, 30 options per group [ASSUMPTION].
- **RES-MENU-005 (P0) Images.** Upload from camera/gallery; client resizes to ≤ 1200 px and compresses; server generates thumbnails (WebP); max 2 MB upload; restaurant attests they own the image rights.
- **RES-MENU-006 (P0) Telugu names.** Every user-visible menu text has an optional Telugu value (`*_i18n` JSONB, R17); UI shows completeness ("12 of 40 items have Telugu names"). Admin can bulk-edit Telugu names (assisted translation).
- **RES-MENU-007 (P0) Price change rules.** Price changes take effect immediately for new carts; carts with old prices are re-validated (CUS-CART-004); in-flight orders keep their snapshot price. Price history retained (audit).
- **RES-MENU-008 (P0) Admin moderation.** New items and price increases > 30% [ASSUMPTION] go live immediately but are flagged for admin review; admin can hide an item with a reason.
- **RES-MENU-009 (P0 for Gate B — ops-side; owner-side P1) Menu bulk import** via CSV template (categories, items, variants, add-ons, prices, dietary markers, Telugu names), with dry-run validation report before commit and audit (M7). Ops uses it to digitise paper menus during assisted onboarding (≈ 1 restaurant per ops person per day).
- **RES-MENU-010 (P1) Duplicate item / copy menu to another outlet.**
- **RES-MENU-011 (P2) Combos/meal builders, item-level schedules beyond category windows, nutritional/allergen info** (allergen text field P1 as free text).

### 6.5 RES-AVAIL — Item availability

- **RES-AVAIL-001 (P0) Quick stock toggle.** One tap to mark an item (or variant, or add-on option) **Out of stock** with auto-restore choice: "Back in 2 hours" / "Back tomorrow (next opening)" / "Until I turn it on". Default "Back tomorrow".
  - AC: Toggling is reflected for customers within 30 s and is available from the order screen (RES-ORD-004).
- **RES-AVAIL-002 (P0) Category-level out-of-stock.**
- **RES-AVAIL-003 (P1) "Out of stock" list view** with one-tap restore all.

### 6.6 RES-ORD — Order management

- **RES-ORD-001 (P0) New-order alert that cannot be missed.** On `PLACED`: full-screen new-order card, **looping loud sound** (until acknowledged), vibration, and Web Push when backgrounded, **repeated every 30 s** until accepted/rejected (R1; escalation per BR-TIME-002); works with screen locked if the PWA is open and push is allowed. Alert content: order code, items with customisations (big text, te names when present), dietary markers, special request, order total and **payment type label ("Prepaid" — restaurant never collects cash)**, customer first name only.
  - AC: *Given* the restaurant app is open on the orders screen *when* an order is `PLACED` *then* the alert renders within 5 s p95.
- **RES-ORD-002 (P0) Accept with prep time.** Accept button with prep-time choice (default = restaurant default prep time; quick options 10/15/20/30/45 min).
- **RES-ORD-003 (P0) Reject with reason.** Reasons (doc 13 §6.3 `ORDER_REJECT`): `ITEMS_OUT_OF_STOCK` (must select the items → they are auto-marked out of stock), `TOO_BUSY`, `CLOSING_SOON`, `KITCHEN_ISSUE`, `RESTAURANT_OTHER` (text). Rejection triggers full refund (BR-REF-001).
- **RES-ORD-004 (P0) Timeout (R1).** If not accepted within the 180 s accept window (BR-TIME-001) the order is **auto-cancelled**: `CANCELLED`, `cancelled_by=SYSTEM`, reason `RESTAURANT_UNRESPONSIVE` (not `REJECTED`), prepaid fully refunded; escalation per BR-TIME-002; auto-pause per BR-TIME-003.
- **RES-ORD-005 (P0) Order board.** Tabs/columns: New, Preparing, Ready, Picked up (today), Past; each card shows elapsed time vs promised prep time, rider status (`not assigned` / name + "arriving in ~N min" (estimate) / "at restaurant").
- **RES-ORD-006 (P0) Restaurant cannot-fulfil after accept — ops-mediated (R40).** The restaurant **cannot self-cancel** after accepting. "Can't make this order" raises an **urgent ops issue** (ticket, priority P2) with a reason; ops contacts the customer and cancels via ADM-ORD-006 with reason `RESTAURANT_CANNOT_FULFIL` and fault `restaurant` (`cancelled_by=ADMIN`), full refund; counted against restaurant quality metrics (M-10).
- **RES-ORD-007 (P0) Handover.** Restaurant sees rider name and the **order code** to match; rider marks picked up (RDR-FLOW-004). Restaurant may mark "Handed over" too (either side completes pickup; first wins; conflicts logged).
- **RES-ORD-008 (P0) No customer contact details.** Restaurant sees customer first name and order notes only; no phone or address (data minimisation, LEG-DPDP-004); issues go via rider or support.
- **RES-ORD-009 (P1) Printable KOT** (browser print, 58/80 mm-friendly layout).
- **RES-ORD-010 (P2) Bluetooth thermal printer integration; partial order edit (remove an unavailable item with partial refund).**

### 6.7 RES-PREP — Preparation status and prep time

- **RES-PREP-001 (P0) Status progression.** On accept the order is `ACCEPTED`; it moves to `PREPARING` on a restaurant tap **or automatically 60 s after accept** (R3; `ordering.auto_preparing_after_s`); prep time is chosen at accept; then `READY_FOR_PICKUP` ("Food is ready"). If the restaurant never marks ready, the rider's pickup is allowed from `PREPARING` and the order is flagged `restaurant_skipped_ready` (R4).
- **RES-PREP-002 (P0) Prep-time extension.** Restaurant can add +5/+10 min once (max 2 extensions per order, total ≤ +20 min); customer ETA updates; extension reason optional.
- **RES-PREP-003 (P0) Default prep time** per restaurant (default 15 min) and per-order override at accept.
- **RES-PREP-004 (P0) Ready reminder.** If prep time elapsed + 5 min and not `READY_FOR_PICKUP` while rider is `AT_RESTAURANT`, prompt the restaurant "Is the food ready?".

### 6.8 RES-ANLY — Basic analytics

- **RES-ANLY-001 (P0) Today/this week summary.** Orders (delivered, rejected, cancelled), gross food sales, net payable estimate, average rating (last 30 days).
- **RES-ANLY-002 (P2 — was P1; deferred to V1.1 per C13) 7/30-day trends.** Orders per day, revenue per day, top 10 items by quantity and revenue, rejection & timeout counts, average accept time, average prep time vs promised, ratings distribution with recent comments.
- **RES-ANLY-003 (P1) CSV export** of orders for a date range (no customer PII).
- **RES-ANLY-004 (P2) Peak-hour heatmaps, customer repeat analytics, menu engineering** — deferred (advanced analytics).

### 6.9 RES-PAYO — Payouts and statements

- **RES-PAYO-001 (P0) Statement view.** Per payout cycle (BR-PAYOUT-001): list of delivered orders with per-order breakdown: item total, packaging, restaurant-funded discount, commission (%, amount), GST on commission, TDS (if applicable), adjustments (refund recoveries, compensations), **net payable**; cycle total; payout status (`PENDING`, `PROCESSING`, `PAID` with UTR/reference, `ON_HOLD` with reason).
  - AC: Every number on the statement is derivable from ledger entries (ADM-PAYO-002); the statement total equals the payout amount (M-26).
- **RES-PAYO-002 (P0) Download** statement CSV; (P1) PDF statement and monthly commission tax invoice.
- **RES-PAYO-003 (P0) Payout account** shown masked (last 4 digits); change requires re-verification (RES-PROF-001).
- **RES-PAYO-004 (P1) Raise payout dispute** from a statement line (creates ticket).

---

## 7. Functional requirements — Delivery partner (RDR-*)

Rider UI is the `rider` app at the `rider.` host (R14). Role `RIDER`. Optimised for one-handed use outdoors (NFR-A11Y-005).

### 7.1 RDR-ONB — Registration and KYC

- **RDR-ONB-001 (P0) Self-registration** with phone OTP; profile: full name, photo (selfie, required), date of birth (must be ≥ 18), emergency contact, preferred language, vehicle type (`MOTORCYCLE`, `SCOOTER`, `EV_SCOOTER`; bicycles not in V1 [OPEN]).
- **RDR-ONB-002 (P0) KYC documents.**
  | Document | Required | Captured |
  |---|---|---|
  | Driving licence | **Yes** | DL number, expiry, class covers two-wheeler (MCWG/MCWOG), front/back images |
  | Vehicle RC | **Yes** | registration number, owner name, image (RC need not be in rider's name; then a declaration of permission) |
  | Vehicle insurance | Yes (P1: expiry tracking) | policy expiry, image |
  | PAN | **Yes** | PAN, image |
  | Bank account and/or UPI ID | **Yes** (one of) | holder name, account no., IFSC / VPA |
  | Aadhaar | **Not collected** | — (never store full Aadhaar, baseline §3; DL serves as identity + address proof) |
  | Police verification certificate | [OPEN – ops/legal decide; recommended P1] | image |
  - AC: Documents stored privately as images only (as RES-ONB-003, R38); low-speed EVs not requiring DL/RC are out of scope V1.
- **RDR-ONB-003 (P0) Application review** by `ADMIN_OPS`: `SUBMITTED → UNDER_REVIEW → APPROVED | CHANGES_REQUESTED | REJECTED`; approval requires completed **in-person or video induction** (checkbox + date recorded by admin) covering app usage, COD handling, food safety & hygiene, road safety.
- **RDR-ONB-004 (P0) Rider agreement** click-accept (independent partner terms, pay structure version, COD responsibilities, code of conduct), with version and timestamp.
- **RDR-ONB-005 (P1) Document expiry tracking** (DL, insurance) with reminders; expired → rider blocked from going online.
- **RDR-ONB-006 (P1) Kit issuance record** (bag, T-shirt, deposit amount if any) [OPEN – ops].
- **RDR-ONB-007 (P0) Gig-worker registration data (M5) [LEGAL].** Capture the rider fields the Social Security Code aggregator portal requires (LEG-GIG-001), e.g. gender, permanent address with state, date of joining/last active date, bank account, and an e-Shram/UAN number if the rider has one — **never full Aadhaar**. Exact field list confirmed by counsel; exported via ADM-RPT-005.

### 7.2 RDR-AVAIL — Availability (online/offline)

- **RDR-AVAIL-001 (P0) Go online/offline toggle.** Going online requires: approved, not suspended, documents valid, location permission granted and a fresh location fix, and (for COD eligibility) cash-in-hand within limit (BR-COD-006). Status persists; availability states per doc 13 §4.3 (`OFFLINE`, `AVAILABLE`, `ON_DELIVERY`; location freshness and `cod_blocked` are flags, not states; no `ON_BREAK`, C12). Rider sessions use a 30-day sliding expiry (R44).
- **RDR-AVAIL-002 (P0) Foreground location while online.** The app sends location every 30–60 s while open and online (baseline P12); it shows a persistent "Keep rovo open while online" banner and uses Screen Wake Lock where available.
  - AC (R34 dispatch tiers): location ≤ 3 min old → **tier 1** (ranked by distance); stale but ≤ 15 min → **tier 2**, still offered orders, reached via Web Push (`Urgency: high`) + SSE and ranked after tier 1; *given* no ping/heartbeat for 15 min *then* the rider is set offline with a notification. Thresholds are `app_config` keys owned by doc 13.
- **RDR-AVAIL-003 (P0) Auto-offline on ignored offers.** 3 consecutive **expired** (not declined) offers → rider set offline with a push "We set you offline because you missed 3 orders" (BR-DISP-004).
- **RDR-AVAIL-004 (P1) Planned shifts / slot booking** — P2 (deferred); V1 uses free online/offline.

### 7.3 RDR-ASSIGN — Order offers and assignment

- **RDR-ASSIGN-001 (P0) Offer card.** Shows: restaurant name & locality, pickup distance from rider (approx.), drop locality (not exact address) and approx. drop distance, **estimated earning for this delivery**, payment type (**COD: collect ₹X** / Prepaid), prep status/ready ETA, countdown timer.
- **RDR-ASSIGN-002 (P0) Offer timeout and cascade.** Offer expires after 45 s (baseline P11; `dispatch.offer_ttl_s`, doc 13 §4.2); an offer withdrawn by system/admin becomes `REVOKED` (R16); one rider offered at a time; on decline/expiry the next best rider is offered (BR-DISP-001..003).
  - AC: *Given* an offer is `PENDING` *when* 45 s elapse without response *then* the offer becomes `EXPIRED`, the rider's screen dismisses it, and the next candidate receives an offer within 5 s.
- **RDR-ASSIGN-003 (P0) Accept/decline.** Accept → delivery `ASSIGNED`, full pickup details revealed; Decline with optional reason (Too far, Vehicle issue, Ending shift, Other). Declines are not penalised in V1 but tracked (M-05).
- **RDR-ASSIGN-004 (P0) One active delivery per rider** (baseline P11). No batching.
- **RDR-ASSIGN-005 (P0) Manual assignment by admin** (ADM-ORD-005) appears to the rider as an assigned job with a notification (not an offer) — rider can "Report problem" to return it.
- **RDR-ASSIGN-006 (P0) Offer alert** with sound + vibration + Web Push; must render while app is in foreground within 3 s p95 of creation.
- **RDR-ASSIGN-007 (P1) Rider-initiated unassign before pickup** (emergency) with reason → back to dispatch, flagged to ops.

### 7.4 RDR-FLOW — Pickup and delivery workflow

- **RDR-FLOW-001 (P0) Navigate to restaurant.** "Navigate" opens the device's map app via a geo/maps URL with the restaurant pin (no in-app routing, no paid API).
- **RDR-FLOW-002 (P0) Arrived at restaurant** button → delivery `AT_RESTAURANT` (soft geofence: warn if > 300 m from restaurant pin; allow with reason). Starts waiting-time clock (BR-RPAY-003).
- **RDR-FLOW-003 (P0) Order verification at pickup.** Rider sees order code + item count + item list to match with the restaurant's packed order.
- **RDR-FLOW-004 (P0) Picked up** button → delivery `PICKED_UP` and order `PICKED_UP`; allowed when the order is `READY_FOR_PICKUP` **or `PREPARING`** (R4) — in the latter case with a confirmation "Restaurant handed over food but didn't mark ready" and the order flagged `restaurant_skipped_ready` (actor = rider, logged; exact transition per doc 13).
- **RDR-FLOW-005 (P0) Drop details revealed after pickup**: customer/recipient first name, full address incl. landmark, delivery instructions, pin; "Navigate" to drop pin; call button (BR-CONT-001).
- **RDR-FLOW-006 (P0) Arrived at drop** → delivery `AT_DROP` (soft geofence 300 m); notifies the customer "Your delivery partner has arrived".
- **RDR-FLOW-007 (P0) Delivery OTP (when required, BR-OTP-001 / R39: prepaid ≥ ₹300, off for COD).** Rider enters the customer's 4-digit OTP; 5 attempts; fallback "Customer can't find OTP" → shows OTP is in customer's app/push; second fallback "Call support" where an `ADMIN_OPS`/`ADMIN_SUPPORT` can authorise completion (audit-logged).
- **RDR-FLOW-008 (P0) Mark delivered.** For COD, rider must first confirm **"Collected ₹X cash"** (exact total; no partial in V1); then `DELIVERED`. Records timestamp + location.
  - AC: *Given* COD order total ₹405 *when* rider taps Delivered *then* the app requires the confirmation "I collected ₹405" before completing, and a cash-collected ledger entry of ₹405 is written atomically with the status change.
- **RDR-FLOW-009 (P0) Undeliverable flow.** Rider can mark "Can't deliver" only from `AT_DROP` after **≥ 10 min waiting and ≥ 2 call attempts** (logged as call-button taps) [ASSUMPTION], choosing reason (`CUSTOMER_UNREACHABLE`, `CUSTOMER_REFUSED`, `WRONG_ADDRESS`, `UNSAFE_LOCATION`, `OTHER`); requires **support/ops approval** (ops tries calling the customer) before order → `UNDELIVERABLE`, delivery → `FAILED` (R5); there is **no rider self-mark** — an unanswered request escalates to the ops lead after 5 min (doc 13 T-UNDELIV-SLA). Food disposition instruction shown (default: rider does not return to restaurant; follow ops instruction) [OPEN – ops].
- **RDR-FLOW-010 (P0) Resilient actions.** Every rider action (arrived, picked up, delivered, collected cash) is retried automatically on network failure with an idempotency key and shows a "Syncing…" state; the device timestamp is sent and stored alongside server time.
- **RDR-FLOW-011 (P1) Offline action queue** (actions persisted locally and replayed in order when back online).
- **RDR-FLOW-012 (P1) Proof-of-delivery photo** optional for prepaid orders without OTP ("left at door" on customer instruction).
- **RDR-FLOW-013 (P1) SOS button**: one tap to dial 112 and alert ops with last location (stored as `sos_events`, M4).

### 7.5 RDR-CONT — Contacting customer and restaurant

V1 approach (masked calling deferred; see BR-CONT-*): **direct calling via `tel:` links, numbers revealed only to the parties of an active delivery, only for the window they need it, and every call-button tap is logged** (`contact_tap_log`, M4). A `tel:` link discloses the number in the dialler and call log; the privacy notice says so plainly (RV-035).
- **RDR-CONT-001 (P0)** Rider → restaurant call button from `ASSIGNED` until `PICKED_UP` + 15 min (restaurant's business phone, not owner's personal number unless that is the only one).
- **RDR-CONT-002 (P0)** Rider → customer call button from `PICKED_UP` until `DELIVERED`/`FAILED` + 15 min; the number is the order's recipient/override phone; the app shows the number only inside the dialler handoff (not as copyable text on screen) — a deterrent, not a guarantee [ASSUMPTION].
- **RDR-CONT-003 (P0)** Rider → support call button always (ops helpline).
- **RDR-CONT-004 (P0)** After the window closes, past deliveries show locality only — no name, phone or address (LEG-DPDP-004).
- **RDR-CONT-005 (P2)** Masked calling via a cloud telephony provider; canned in-app messages ("I'm at the gate").

### 7.6 RDR-HIST — Delivery history

- **RDR-HIST-001 (P0)** List of completed/failed deliveries by day: order code, restaurant, drop locality, distance (computed), earning, COD collected, status.
- **RDR-HIST-002 (P1)** Ratings summary (average over last 50 rated deliveries, tag counts) — individual ratings never attributed to customers.

### 7.7 RDR-EARN — Earnings and cash-in-hand

- **RDR-EARN-001 (P0) Earnings view.** Today / this week / payout cycle: per-delivery pay (base + distance + waiting) per BR-RPAY-*, adjustments, total.
- **RDR-EARN-002 (P0) Cash-in-hand view.** Current cash-in-hand (sum of COD collected − deposits − netting), limit, and per-order list of cash collected since last settlement; prominent warning when ≥ 80% of limit.
- **RDR-EARN-003 (P0) Deposit recording.** Rider can declare a deposit (UPI transfer to the company collection account or cash handed at ops desk) with reference/UTR; it is **pending** until finance/ops confirms (ADM-PAYO-005); confirmed deposits reduce cash-in-hand.
- **RDR-EARN-004 (P0) Payout statement** per cycle: earnings − cash-in-hand netted (BR-COD-008) = net payout (or net amount due from rider), status and UTR.
- **RDR-EARN-005 (P1) On-demand payout request** (min ₹200, max 1/day), processed manually by finance within 1 business day.
- **RDR-EARN-006 (P2) Incentives (automated peak-hour/milestone bonus schemes), tips** — deferred. V1 exception: ops may grant a **manual rider peak bonus** as a ledger adjustment when supply is short (R30, ADM-PAYO-006).
- **RDR-EARN-007 (P0 — ops) Pilot minimum guarantee (M6, R47).** For scheduled pilot peak slots, a rider who was online for the slot and met the agreed conditions (acceptance, no rider-fault failures) is topped up to a guaranteed amount via an `MG_TOPUP` ledger adjustment, shown as its own line in earnings and the payout statement. Amount, slots and eligibility are a business decision [OPEN]; posted through ADM-PAYO-006 (maker-checker above threshold, R31).

---

## 8. Functional requirements — Admin (ADM-*)

Admin UI = `admin` web app at the `admin.` host (R14; desktop-first, works on tablet). Roles: `ADMIN_SUPER`, `ADMIN_OPS`, `ADMIN_SUPPORT`, `ADMIN_FINANCE`, city-scoped (baseline §2). Product-level permission intent per area is given as **[S/O/Sp/F]** = which roles may *act* (all admin roles may *view* unless stated). Doc 12 owns the authoritative RBAC matrix.

### 8.1 ADM-AUTH — Admin access

- **ADM-AUTH-001 (P0)** Email + password + mandatory TOTP for all admins (baseline P9, R37); admin accounts are **separate identities** from consumer/partner accounts (R26); admin accounts created only by `ADMIN_SUPER` (role grant needs a second approver, R31 family 5); no self-signup. WAF rate + geo (India) rules on the admin host; no identity-aware proxy in V1 (C4).
- **ADM-AUTH-002 (P0)** Role and city scope assignment; deactivation takes effect within 1 min (sessions revoked).
- **ADM-AUTH-003 (P0)** Sensitive data reveal (full phone, KYC document view, bank details) requires a reason prompt and is audit-logged.
- **ADM-AUTH-004 (P1) Passkeys (WebAuthn)** for admins, first for `ADMIN_SUPER`/`ADMIN_FINANCE` (R37).

### 8.2 ADM-DASH — Dashboard

- **ADM-DASH-001 (P0) Live ops summary** (auto-refresh ≤ 30 s): orders by status now; orders today (placed/delivered/cancelled); riders online / busy / idle; restaurants open / paused / unexpectedly offline; SLA breaches (unaccepted > 90 s, unassigned > 5 min, late > 10 min); open tickets by age.
- **ADM-DASH-002 (P0) Today's KPIs**: M-01, M-03, M-06 (p50/p90), M-09, GMV, COD share, online payment success.
- **ADM-DASH-003 (P1) 7/30-day trend charts** of the §4 metrics.

### 8.3 ADM-USER — Users (customers) [view: all; act: S/O/Sp]

- **ADM-USER-001 (P0)** Search customers by phone, name, order code; view profile, addresses, order history, tickets, COD status.
- **ADM-USER-002 (P0)** Block/unblock a customer (reason required); toggle COD eligibility (BR-COD-004).
- **ADM-USER-003 (P0)** Process account-deletion and data-access requests (if not self-served) with status tracking (LEG-DPDP-003).
- **ADM-USER-004 (P0)** Admin user management (ADM-AUTH-001/002) [S].

### 8.4 ADM-REST — Restaurants [act: S/O]

- **ADM-REST-001 (P0)** Application queue with document viewer; approve / request changes (comment per field) / reject (reason); approval sets commission and effective date.
- **ADM-REST-002 (P0)** Suspend / reinstate (reason required; suspension immediately makes the restaurant unorderable; in-flight orders continue unless admin cancels).
- **ADM-REST-003 (P0)** Edit any restaurant profile/menu/hours on behalf (assisted onboarding RES-ONB-002), audit-logged.
- **ADM-REST-004 (P0)** Restaurant health view: acceptance rate, timeouts, rejections by reason, cancellations, rating, last-seen-online, FSSAI expiry.
- **ADM-REST-005 (P0)** Force pause/unpause (e.g. restaurant phoned to say they are closed).
- **ADM-REST-006 (P1)** Menu moderation queue (RES-MENU-008).

### 8.5 ADM-RDR — Delivery partners [act: S/O]

- **ADM-RDR-001 (P0)** Application queue with document viewer; approve (requires induction recorded) / request changes / reject.
- **ADM-RDR-002 (P0)** Suspend / reinstate (reason; suspended riders forced offline; active delivery must be reassigned first).
- **ADM-RDR-003 (P0)** Rider list with live state (offline / available tier 1 fresh / available tier 2 stale / on delivery + delivery status, R34), last location time and distance to a chosen restaurant (no map required), cash-in-hand, today's deliveries.
- **ADM-RDR-004 (P0)** Rider performance: offers accepted/declined/expired, deliveries, avg last-mile time, ratings, undeliverables, cash ageing.
- **ADM-RDR-005 (P2)** Map of riders' last-known positions (static, not live tracking).

### 8.6 ADM-ORD — Orders: live board and interventions [act: S/O; Sp limited]

- **ADM-ORD-001 (P0) Live order board.** All active orders with columns by stage (Placed → Accepted/Preparing → Ready → Picked up → At drop), each card showing order code, restaurant, locality, payment type, elapsed time vs SLA with colour state, rider, flags (late, unaccepted, unassigned, OTP issue, undeliverable request). Filters (zone, restaurant, rider, flag), search by order code/phone. Updates live (SSE).
- **ADM-ORD-002 (P0) Order detail** with full timeline (every status change: from→to, actor, timestamp, reason), payment events, offers history, call-tap log, notes, tickets.
- **ADM-ORD-003 (P0) Accept on behalf of restaurant** after phoning the restaurant — the order is flagged at 90 s and ops call manually (R1/R43; ops desk staffed during service hours) (reason "Confirmed by phone" required; prep time required; audited) — essential for low-digital-maturity partners. [S/O]
- **ADM-ORD-004 (P0) Update status on behalf** (mark ready, picked up, delivered, undeliverable) with mandatory reason — exception handling only, audit-logged, flagged in reports. [S/O]
- **ADM-ORD-005 (P0) Manual assign / reassign rider.** Choose from eligible riders sorted by distance-to-restaurant with their state; reassigning cancels the current rider's assignment (rider notified, reason recorded) and, if pickup already happened, requires confirmation of handover between riders. [S/O]
- **ADM-ORD-006 (P0) Cancel order** with reason code (doc 13 §6.3) and **refund decision** (full / partial amount / none, per BR-CAN and BR-REF), fault attribution (`customer`, `restaurant`, `rider`, `platform`) driving settlement (BR-REF-005). This is also the path for restaurant "can't fulfil" after accept (R40). For COD orders needing compensation, the customer chooses a manual UPI refund or a coupon (R29, BR-REF-004). [S/O; Sp up to refund limit]
- **ADM-ORD-007 (P0) Refund (post-delivery)** full/partial/by item, with reason; refunds above ₹500 [ASSUMPTION] require a second approver (`ADMIN_FINANCE` or `ADMIN_SUPER`) — maker-checker. [Sp/O create; F/S approve above limit]
- **ADM-ORD-008 (P0) Internal notes** on orders (visible to admins only).
- **ADM-ORD-009 (P0) Contact shortcuts**: call customer / restaurant / rider (`tel:`), each tap logged.
- **ADM-ORD-010 (P1) Bulk actions in an incident** (e.g. pause all restaurants in a zone; disable COD city-wide; city-wide in-app banner, ADM-CFG-003). No push/SMS admin broadcasts in V1 (C20).
- **ADM-ORD-011 (P1) Ops-assisted phone ordering (M8).** An `ADMIN_OPS`/`ADMIN_SUPPORT` agent places a **COD** order for a customer who calls the support line (e.g. persona P3): finds or creates the customer by phone (OTP to the customer's phone or recorded verbal consent [LEGAL]), selects a restaurant and items, confirms the address pin with the customer, reads out the full bill, and places the order through the same quote/placement rules (BR-COD limits apply). The customer receives an SMS with the order code; the order is tagged `assisted` and audited.

### 8.7 ADM-COMM — Commissions [act: S/F]

- **ADM-COMM-001 (P0)** Per-restaurant commission % with effective-from date (history retained; never retroactive to past orders); city default (15%); allowed range 0–30% hard bounds (`commission_bps` CHECK 0–3000, doc 10). Changes need a second approver (R31 family 3).
- **ADM-COMM-002 (P0)** Commission snapshot stored on each order at placement (the rate in effect at `PLACED`).
- **ADM-COMM-003 (P1)** Promotional commission (e.g. 0% for first 30 days for new restaurants) via effective-dated rate.

### 8.8 ADM-COUP — Coupons [act: S/O; F for funding approval]

- **ADM-COUP-001 (P0) Create/edit/deactivate coupons** with: code (globally unique, uppercase alnum 4–16; RV-052), title & terms (en/te), type (`FLAT`, `PERCENT` with max cap, `FREE_DELIVERY`), value, min subtotal, validity window, total redemption cap, per-user cap, first-order-only flag, payment-method restriction (any/online/COD), scope (all restaurants / list; zones), funding (`PLATFORM` | `RESTAURANT` with restaurant consent record; no `SHARED` funding, no cuisine/user targets, no per-day budgets in V1 — C16), display on listing flag.
- **ADM-COUP-002 (P0) Coupon report**: redemptions, discount given, orders, by restaurant; CSV.
- **ADM-COUP-003 (P0) Validation preview** ("test this coupon against cart X").
- **ADM-COUP-004 (P2 — was P1; deferred per C16) Bulk unique codes** (e.g. 500 single-use codes for a college fest). Single-user goodwill coupons (R9) remain V1.

### 8.9 ADM-TKT — Disputes and tickets [act: Sp/O; F for payment tickets]

- **ADM-TKT-001 (P0) Ticket inbox**: sources customer (CUS-SUPP), restaurant (RES-PAYO-004, P1), rider (P1), admin-created (phone calls logged as tickets). Fields: ticket no., category, order link, priority (auto: payment issues & active orders = high), status (doc 10 set: `OPEN`, `IN_PROGRESS`, `AWAITING_REQUESTER`, `AWAITING_APPROVAL`, `RESOLVED`, `CLOSED`, `REOPENED`), assignee, SLA timers (acknowledge 48 h legal max; internal first-response target 10 min), conversation thread, attachments, resolution code.
- **ADM-TKT-002 (P0) Resolution actions** from the ticket: refund (via ADM-ORD-007), goodwill coupon (single-user; above ₹150 needs a second approver, R31 family 1), COD compensation as the customer's choice of manual UPI refund (UTR recorded) or coupon (R29), restaurant/rider warning note, no action — each with customer-visible message.
- **ADM-TKT-003 (P0) Grievance escalation**: tickets flagged "grievance" route to the grievance officer queue with the 1-month redressal clock (LEG-CP-003).
- **ADM-TKT-004 (P1) Canned responses** (en/te).
- **ADM-TKT-005 (P1) Fraud flags**: customers with > 2 missing-item claims in 30 days surfaced to agent.

### 8.10 ADM-RPT — Reports and CSV export [view per role]

- **ADM-RPT-001 (P0) CSV exports** (date range, city/zone filters): orders (with status timestamps), order items, payments & refunds, COD collections & deposits, restaurant settlements, rider earnings, coupons, tickets, ratings. Exports of PII-bearing data restricted to S/F and audit-logged; default exports pseudonymise customer phone (last 4 digits).
- **ADM-RPT-002 (P0) GST working report** for finance: per order taxable values and GST by component (restaurant service under §9(5), delivery fee, platform fee, commission) to support returns preparation by the CA (LEG-GST-*).
- **ADM-RPT-003 (P0) Daily ops report** (auto-generated, emailed P1): §4 metrics for the day.
- **ADM-RPT-004 (P2) Scheduled reports, BI integration, cohort analysis** — deferred.
- **ADM-RPT-005 (P0) Gig-worker registration export (M5) [LEGAL].** CSV of rider registration fields (RDR-ONB-007) and new/active/exited riders per period, in the format required by the designated portal (LEG-GIG-001); restricted to S/O, audit-logged.

### 8.11 ADM-AUDIT — Audit logs [view: S; F for money events]

- **ADM-AUDIT-001 (P0)** Every admin/partner action that changes state, money, permissions, configuration, prices, KYC or reveals sensitive data writes an append-only audit record (append-only table, app role has no UPDATE/DELETE grants; no per-row hash chain in V1, C6): actor id + role, city, action, entity type/id, before/after (diff, with secrets redacted), reason, timestamp, IP, user agent, request/trace id.
- **ADM-AUDIT-002 (P0)** Audit viewer with filters (actor, entity, action, date) and CSV export; no edit/delete capability for anyone.
- **ADM-AUDIT-003 (P0)** Order/delivery state transitions are captured in the order event log (ADM-ORD-002), distinct from the admin audit log, both retained per NFR-RET.

### 8.12 ADM-ZONE — Zones and fees configuration [act: S/O; F for fee changes]

- **ADM-ZONE-001 (P0) Zones**: draw/edit polygons on a map (MapLibre), name, active flag (GeoJSON import/export, impact dry-run and heat maps deferred, C15); overlapping zones disallowed in V1 [ASSUMPTION – doc 16 decides]; localities list per city (name en/te, PIN, centroid).
- **ADM-ZONE-002 (P0) Fee configuration** per city with optional per-zone override, effective-dated: delivery fee slabs, max radius, road factor, platform fee, small-cart fee & threshold, COD max order value, COD enabled flag, rider pay parameters, cash limit, service hours. Defaults owned by doc 16 §6 (R48). **Fee-config and commission changes need a second approver** (R31 family 3). Tax rules are seeded by migration — no tax-rule CRUD UI in V1 (C20).
  - AC: Changing a fee never alters already-placed orders; new quotes use the new config from its effective time; change is audit-logged with the previous value.
- **ADM-ZONE-003 (P0) Zone pause** (e.g. heavy rain/flooding in a locality): stops new orders to addresses inside, banner explains.
- **ADM-ZONE-004 (P2) Weather/peak surcharge** — not V1 (BR-FEE-006, R30, C1).

### 8.13 ADM-PAYO — Payouts and settlement [act: F/S]

- **ADM-PAYO-001 (P0) Settlement run** per cycle: system computes restaurant and rider payables from the ledger (restaurant: BR-COMM; rider: BR-RPAY & BR-COD netting) into a **payout batch** in `DRAFT`; finance reviews, approves (maker-checker: creator ≠ approver), exports a bank-upload/UPI CSV, records transfers with UTR/reference per payee, and marks `PAID`; failures marked with reason and carried to next cycle.
- **ADM-PAYO-002 (P0) Ledger**: double-entry style internal ledger with accounts per restaurant, rider, customer refunds, platform revenue, GST payable, PA clearing, cash-in-transit; every order financial event posts balanced entries (doc 14 owns design). Finance can view a payee's ledger with running balance.
- **ADM-PAYO-003 (P0) PA reconciliation**: import PA settlement report (CSV) and match captured payments/refunds/fees to ledger; mismatches listed for action (`recon_exceptions`, M4); golden tests against the PA's sample settlement files (M14).
- **ADM-PAYO-004 (P0) Holds**: put a payee's payout on hold with reason (e.g. KYC issue, dispute).
- **ADM-PAYO-005 (P0) Rider cash deposits**: confirm/reject rider-declared deposits against bank/UPI statement — **bank-statement CSV import with automatic UTR matching** (RV-068); record cash received at ops desk with receipt number; daily cash-in-hand report with ageing (M-22).
- **ADM-PAYO-006 (P0) Manual adjustments** (credit/debit to a payee with reason; second approver above the threshold in `approvals.thresholds`, R31 family 1) — e.g. restaurant compensation for customer-fault cancellation, rider minimum-guarantee top-up `MG_TOPUP` (RDR-EARN-007), manual rider peak bonus (R30), rider damage recovery (rider recoveries require written rider acknowledgement [LEGAL]).
- **ADM-PAYO-007 (P0) COD refunds to customers** (no original payment instrument): when the customer chooses money over a coupon (R29), finance records the manual UPI/bank refund to the customer-provided VPA with UTR and a ledger entry (`refunds.provider='MANUAL'`).
- **ADM-PAYO-008 (P1) TDS/TCS computation & reports** per LEG-TAX (if confirmed applicable).
- **ADM-PAYO-009 (P2) Automated payouts** via PA payout/route API — V1.1 candidate.

### 8.14 ADM-CFG / ADM-CONT — Platform configuration and content

- **ADM-CFG-001 (P0)** City settings: name (en/te), timezone, service hours, support phone, grievance officer details, feature flags (COD enabled, delivery OTP policy), timers and thresholds (keys and defaults owned by doc 13 §5 / doc 10 `app_config`, e.g. `ordering.accept_window_s`, `dispatch.offer_ttl_s`, `payments.pending_timeout_s`, `restaurant.device_offline_pause_s`), cash limit; all effective-dated & audit-logged. [S]
- **ADM-CFG-002 (P0)** Legal documents management: terms, privacy notice, partner agreements, refund policy — versioned (en/te); users re-consent on material change.
- **ADM-CONT-001 (P1)** Cuisine list, collections, banners, FAQs (en/te).
- **ADM-CFG-003 (P1)** City-wide customer banner (incident message).

---

## 9. Cross-cutting functional requirements

### 9.1 NOT — Notifications (doc 15 owns channels/templates)

| ID | Pri | Event | Customer | Restaurant | Rider | Admin |
|---|---|---|---|---|---|---|
| NOT-001 | P0 | OTP | SMS | SMS | SMS | — (TOTP) |
| NOT-002 | P0 | Order `PLACED` | in-app + push | **loud alert + push, repeated every 30 s; owner SMS + push at 60 s** (R1) | — | board; flagged red + sound at 90 s for ops call |
| NOT-003 | P0 | Order accepted / rejected / cancelled as unresponsive (180 s) | push (+SMS if rejected/cancelled & prepaid, P1) | owner SMS on miss + auto-pause notice | — | board |
| NOT-004 | P0 | Delivery offer | — | — | **loud alert + push** | — |
| NOT-005 | P0 | Rider assigned / at restaurant | in-app | in-app | — | — |
| NOT-006 | P0 | Picked up / arrived / delivered | push | in-app | — | — |
| NOT-007 | P0 | Cancelled / undeliverable / refund initiated / refund completed | push + SMS (refund initiated, P1) | push | push | board |
| NOT-008 | P0 | Ticket acknowledged / reply / resolved | push + in-app (+email if present, P1) | — | — | inbox |
| NOT-009 | P0 | Restaurant/rider application status change | — | push + SMS | push + SMS | — |
| NOT-010 | P0 | Payout paid | — | push | push | — |
| NOT-011 | P1 | Cash-in-hand ≥ 80% of limit / limit reached | — | — | push | — |
| NOT-012 | P1 | FSSAI/DL expiry reminders | — | push + SMS | push + SMS | list |

- **NOT-016 (P1) Automated voice-call escalation (R43).** Automated IVR call to the counter device number and the owner for unaccepted orders; enabled only if the pilot shows > 5% of orders reaching the 90 s mark. V1 relies on repeated push/SSE alarm + owner SMS at 60 s + ops manual call at 90 s.
- **NOT-013 (P0)** All user-facing notifications localised to the recipient's language; SMS only via DLT-approved templates (LEG-TRAI-001).
- **NOT-014 (P0)** Marketing notifications only with marketing consent (CUS-AUTH-003); transactional messages need no marketing consent.
- **NOT-015 (P0)** Push permission is requested in context (e.g. after placing first order: "Get updates when your food is on the way?"), never on first page load.

### 9.2 X — Other cross-cutting

- **X-001 (P0) Human order code** `RV-XXXXXX` shown everywhere the order is referenced (support calls, restaurant, rider).
- **X-002 (P0) Time display** in IST, 12-hour format with AM/PM in `en`; Telugu equivalents (ఉ./సా. via `Intl` `te-IN`) [ASSUMPTION – UX confirms]; durations as "12 min".
- **X-003 (P0) Currency display** `₹1,23,456.50` (en-IN grouping); paise shown on bills and statements, totals per BR-FEE-009.
- **X-004 (P0) Error messages** are localised, human, and suggest an action; never show raw codes except a short support reference.
- **X-005 (P0) Feature flags** per city for: COD, delivery OTP, coupons, ratings display. (Device-heartbeat auto-pause is mandatory per R1, not a flag.)
- **X-006 (P0) Device sessions (R44, M10).** Registered restaurant order-receiver devices get device-bound long-lived sessions: sliding 30-day idle, 90-day absolute, revocable by owner/admin, re-auth scheduled outside service hours; rider sessions 30-day sliding. Doc 12 owns the mechanics.

---

## 10. Non-functional requirements (NFR-*)

### 10.1 NFR-PERF — Performance budgets

Reference device: **low-end Android** (≈ Moto E/Redmi A-series class, 2–3 GB RAM, Chrome stable) on **"Slow 4G"** (Lighthouse mobile throttling) [ASSUMPTION – Frontend Architect may refine].

| ID | Pri | Requirement |
|---|---|---|
| NFR-PERF-001 | P0 | Customer app: initial route JS ≤ 170 KB gzip, CSS ≤ 30 KB gzip; total home first-load transfer ≤ 500 KB including images above the fold; repeat visit ≤ 100 KB (PWA cache). |
| NFR-PERF-002 | P0 | Core Web Vitals p75 (field): LCP ≤ 2.5 s, INP ≤ 200 ms, CLS ≤ 0.1 (customer, restaurant and rider apps). Lab: TTI ≤ 5 s on reference device. |
| NFR-PERF-003 | P0 | API latency (server-side, excluding PA/SMS): p95 ≤ 300 ms reads, ≤ 500 ms writes; quote + place order p95 ≤ 800 ms. |
| NFR-PERF-004 | P0 | Real-time: status event → client render p95 ≤ 5 s (customer), ≤ 5 s (restaurant new order), ≤ 3 s (rider offer). |
| NFR-PERF-005 | P0 | Menu images: list thumbnails ≤ 30 KB (WebP/AVIF), detail ≤ 120 KB; lazy-loaded. |
| NFR-PERF-006 | P0 | Capacity per the **single load model in doc 20 §12.1 (R45)**: planning volumes ≈ 30 orders/day closed pilot, ≈ 80 month 1, ≈ 250 month 3; design point 2,000 orders/day with a 500 orders/h peak; load-tested at 3× (1,500 orders/h) plus 3,000 concurrent SSE connections, including a CGNAT scenario. Infrastructure is sized to the phase (`closed-pilot` / `public-launch` profiles, R32) and capacity is proven by test; scaling beyond is a configuration change, not a re-architecture. Doc 20 wins on any number. |
| NFR-PERF-007 | P1 | Partner app usable after cold start in ≤ 3 s on reference device when cached. |

### 10.2 NFR-AVAIL — Availability and resilience (production on managed cloud, baseline §4a)

Targets apply to the **production** environment (standard hyperscaler, India region, managed services). Local/dev/preview environments (Docker, free tiers) carry **no** availability target.

| ID | Pri | Requirement |
|---|---|---|
| NFR-AVAIL-001 | P0 | **Ordering critical path availability (R49): 99.5% monthly during the closed pilot (Single-AZ); 99.9% monthly from Gate B (Multi-AZ)**, with ops as first responders using runbooks (pause zone, COD-only, accept on behalf) — measured by synthetic checks every 1 min from outside the cloud: browse/menu, quote, place order, payment confirmation (webhook ingest), order status/SSE, restaurant accept, rider offer accept/status updates. (Error budget ≈ 43 min/month.) |
| NFR-AVAIL-002 | P0 | **Other surfaces ≥ 99.5% monthly**: admin app, reports/exports, partner analytics, onboarding/KYC uploads, statements. |
| NFR-AVAIL-003 | P0 | **Pilot exception (R32):** during the closed pilot only, a Single-AZ database (`closed-pilot` IaC profile) with PITR + cross-region automated backups is acceptable with target ≥ 99.5% for the critical path; **Multi-AZ is mandatory before Gate B (public launch) or when orders exceed 100/day, whichever first**. |
| NFR-AVAIL-004 | P0 | **Data protection — targets owned by doc 23 §1** (adopted here, register row 70): logical corruption RPO ≤ 5 min / RTO ≤ 2 h (PITR); AZ failure RPO 0 / RTO minutes once Multi-AZ (R32); region outage RPO ≤ 30 min / RTO ≤ 4 h via cross-region backups into `ap-south-2` + IaC rebuild (no DR pre-provisioning or region game days before Gate B, C10). PA webhooks are replayable and reconciliation (ADM-PAYO-003) detects gaps. |
| NFR-AVAIL-005 | P0 | Zero-downtime deploys for API and worker (rolling, health-checked); schema migrations backward-compatible (expand/contract). Planned maintenance that needs downtime only 01:00–06:00 IST, announced in admin ≥ 24 h ahead, and counted against the error budget. |
| NFR-AVAIL-006 | P0 | Degraded modes: PA down → hide online payment, COD-only banner (if COD enabled); SMS provider down → existing sessions continue, ops alerted (secondary OTP provider P1); SSE down → polling fallback; object storage/CDN degraded → menus render without images. |
| NFR-AVAIL-007 | P0 | All timers that matter (restaurant accept window, offer expiry, payment-pending expiry, auto-resume of pauses) are server-side and survive process restarts/redeploys (persisted jobs, baseline P4); ≥ 2 API replicas and ≥ 1 always-on worker (≥ 2 for public launch) in production. |
| NFR-AVAIL-008 | P0 | Alerting: on-call (ops/dev rota) paged when critical-path SLO burn rate indicates budget exhaustion within 6 h, and on any of: order placement errors > 2% for 5 min, unaccepted-order backlog, webhook backlog, worker queue lag > 60 s, a missed periodic run such as the weekly settlement (catch-up jobs, M11), and Postgres notification-queue usage (M12) (doc 24). |
| NFR-AVAIL-009 | P1 | Status banner mechanism for incidents (ADM-CFG-003); public status page P2. |

### 10.3 NFR-A11Y — Accessibility (WCAG 2.2 AA)

| ID | Pri | Requirement |
|---|---|---|
| NFR-A11Y-001 | P0 | All four apps (R14) conform to **WCAG 2.2 Level AA** for P0 flows; verified by automated checks in CI and a manual audit (TalkBack on Android + keyboard on admin) before launch. |
| NFR-A11Y-002 | P0 | Touch targets ≥ 44×44 CSS px for customer, restaurant and rider apps (exceeds 2.5.8's 24 px minimum), ≥ 8 px spacing between primary actions. |
| NFR-A11Y-003 | P0 | Dietary markers never rely on colour alone: shape + text label ("Veg", "Egg", "Non-veg") + accessible name. |
| NFR-A11Y-004 | P0 | Text scales to 200% without loss of function; Telugu text uses a font with full conjunct support (e.g. Noto Sans Telugu) and line-height ≥ 1.5 to avoid clipped vowel signs. |
| NFR-A11Y-005 | P0 | Rider and restaurant critical actions (Accept, Picked up, Delivered) are full-width, high-contrast (≥ 4.5:1, aim 7:1 for outdoor legibility), and use **slide-to-confirm or confirm dialogs** to prevent accidental taps for irreversible steps. |
| NFR-A11Y-006 | P0 | Alerts (new order/offer) are multi-modal: sound + vibration + visual; sound volume independent of media mute where the platform allows; visual flashing respects 2.3.1 (≤ 3 flashes/s). |
| NFR-A11Y-007 | P0 | Timeouts (offer countdown, OTP) are announced to screen readers; payment pending page doesn't auto-redirect without notice. |
| NFR-A11Y-008 | P1 | Plain-language copy (en ≤ grade 8; te reviewed by native speakers for colloquial clarity, avoiding formal/Sanskritised terms where common words exist). |

### 10.4 NFR-I18N — Localisation

| ID | Pri | Requirement |
|---|---|---|
| NFR-I18N-001 | P0 | 100% of UI strings for P0 screens available in `en` and `te` at launch, from string catalogs (no hard-coded text); translations reviewed by a native Telugu speaker. |
| NFR-I18N-002 | P0 | Locale-aware number/currency/date formatting via `Intl` (`en-IN`, `te-IN`); digits displayed as Western Arabic numerals in both locales [ASSUMPTION – common usage]. |
| NFR-I18N-003 | P0 | User-generated content: Telugu fields optional; fallback to English; search indexes both. |
| NFR-I18N-004 | P0 | Server-generated content (SMS, push, emails, invoices, statements) localised to recipient language; invoices bilingual or English with mandatory fields per GST rules [LEGAL]. |
| NFR-I18N-005 | P0 | Layouts tolerate +40% text expansion (Telugu strings are often longer). |
| NFR-I18N-006 | P2 | Additional locales (`ur`, `hi`) — architecture must allow adding without code change (RTL readiness for Urdu flagged to Frontend). |

### 10.5 NFR-PRIV — Privacy (DPDP) — see also LEG-DPDP-*

| ID | Pri | Requirement |
|---|---|---|
| NFR-PRIV-001 | P0 | Data minimisation: collect only fields listed in this PRD; no contacts/SMS-inbox/background-location permissions requested. |
| NFR-PRIV-002 | P0 | Role-based visibility of PII: restaurant never sees customer phone/address; rider sees customer contact only during the active-delivery window (RDR-CONT); support sees masked phone by default (reveal logged). |
| NFR-PRIV-003 | P0 | KYC documents encrypted at rest, served via short-lived signed URLs to authorised roles only; never cached in CDN. |
| NFR-PRIV-004 | P0 | Rider location used only for dispatch, ETA and safety; raw pings retained per NFR-RET; never shown to customers in V1. |
| NFR-PRIV-005 | P0 | No third-party ad/tracking SDKs; product analytics (if any) first-party or privacy-preserving, no PII in analytics events or logs (phones/addresses redacted in logs). |
| NFR-PRIV-006 | P0 | **Production personal data (database, object storage, backups, logs) resides in an India cloud region** (baseline §4a); cross-border processing limited to processors with DPDP-aligned contracts (observability on Grafana Cloud India region incl. Faro for frontend errors; no Sentry in V1, R36) [LEGAL – DPDP permits transfer except to notified restricted countries; counsel confirms]. Dev/preview environments use **synthetic data only** — no production PII on free tiers. |

### 10.6 NFR-SEC — Security (doc 12/19 own details)

- **NFR-SEC-001 (P0)** OWASP ASVS L2 for the API; TLS everywhere; secrets not in repo; rate limiting on OTP, login, coupon validation, search.
- **NFR-SEC-002 (P0)** PA webhooks verified by signature; amounts and order IDs cross-checked server-side.
- **NFR-SEC-003 (P0)** Admin: TOTP mandatory (R37), passkeys P1 (ADM-AUTH-004); WAF rate + India geo rules on the admin host; session timeout 30 min idle [ASSUMPTION]; IP/device logged.
- **NFR-SEC-004 (P0)** Payment card data never touches rovo servers (PA hosted checkout ⇒ PCI-DSS scope SAQ-A).
- **NFR-SEC-005 (P0)** Security incident reporting process: CERT-In report within 6 hours of noticing specified incidents; DPDP breach intimation (LEG-DPDP-007) [LEGAL].

### 10.7 NFR-AUD — Auditability

- **NFR-AUD-001 (P0)** Every money movement is traceable from order → ledger entries → payout batch → UTR, and from PA settlement → ledger.
- **NFR-AUD-002 (P0)** Every order has a complete, immutable event history (status transitions, actor, reason, timestamps, client timestamps for rider actions).
- **NFR-AUD-003 (P0)** Config, price, commission, fee and coupon changes are effective-dated and never overwrite history.
- **NFR-AUD-004 (P0)** Admin audit log per ADM-AUDIT-001; tamper-evidence = append-only table with no UPDATE/DELETE grants for the app role. Hash-chained/sealed entries deferred to V1.1+ (C6).
- **NFR-AUD-005 (P0)** Time sources: server UTC with NTP; all reports state timezone.

### 10.8 NFR-RET — Data retention [LEGAL – retention schedule to be confirmed by counsel/CA]

| Data | Retention | Basis / note |
|---|---|---|
| Orders, invoices, payments, refunds, ledger, payouts, GST/TDS records | **8 years** from end of financial year | GST Act requires books for 72 months from due date of annual return; Companies Act 8 years — take the longer [LEGAL] |
| Customer profile & addresses | Until account deletion; then erased/anonymised within 30 days; orders retained with pseudonymised customer reference | DPDP purpose limitation |
| Inactive customer accounts | Notify and erase after 3 years of inactivity [ASSUMPTION – adopt DPDP Third-Schedule-style practice voluntarily] | DPDP Rules |
| Restaurant/rider KYC documents | Duration of relationship + 8 years (financial linkage) [LEGAL]; images may be reduced to extracted fields + hash after 3 years [OPEN] | Tax, disputes |
| Rider raw location pings | **30 days**, then deleted (keep only per-delivery computed distances) | Minimisation |
| OTP send logs | 1 year | Abuse investigation; DLT disputes |
| Application/security logs and traffic logs | **180-day archive in India for all ICT-system logs** (app, LB/edge, WAF, VPC flow, DB, cloud audit — CloudWatch Logs/S3 in `ap-south-1`); security events **≥ 1 year** (M1, R36) [LEGAL] | DPDP Rules log retention; CERT-In 2022 |
| Admin audit logs | 8 years | Financial actions traceability |
| Support tickets & attachments | 3 years after closure; photos 1 year | Consumer disputes |
| Ratings/reviews | Life of restaurant listing; anonymised on author deletion | — |
| Marketing consent records | Life of account + 3 years | Proof of consent |

---

## 11. Business rules (BR-*)

All amounts configurable per city (and per zone/restaurant where stated) via ADM-ZONE-002 / ADM-CFG-001 with effective dates. Defaults from baseline §5 unless marked as a product decision.

### 11.1 BR-FEE — Customer-facing fees and pricing

- **BR-FEE-001 Delivery fee (R18)** by **road-adjusted** distance restaurant→customer = haversine × road factor 1.3: 0–2 km ₹20 · 2–4 km ₹30 · 4–6 km ₹40 · 6–8 km ₹50 · 8–10 km ₹60, **lower bound inclusive, upper exclusive** (`[lo,hi)`, so 2,000 m → ₹30). Serviceability uses the **straight-line** max radius (default 7 km ≈ 9.1 km road), so every serviceable point has exactly one slab. Fee is **GST-inclusive** (product decision, BR-FEE-007). Defaults owned by doc 16 §6.1 (R48).
- **BR-FEE-002 Platform fee** ₹5 per order, flat, GST-inclusive.
- **BR-FEE-003 Small-cart fee** ₹15 when item total (after item-level, before coupon) < ₹149; GST-inclusive. Not charged if a FREE_DELIVERY or restaurant offer explicitly waives it (coupon config flag).
- **BR-FEE-004 Packaging charge** set by restaurant: per item (≤ ₹30/item [ASSUMPTION]) or per order (≤ ₹50 [ASSUMPTION]); taxed with food at 5% as part of the restaurant supply [LEGAL]; passed 100% to restaurant, **no commission** on packaging (product decision).
- **BR-FEE-005 No minimum order value** in V1 (small-cart fee instead). Restaurant-level minimum order P2.
- **BR-FEE-006 No surge/rain/peak fees** in V1 (R30). Bad weather / rider shortage is handled by zone pause (ADM-ZONE-003) plus a manual rider peak bonus posted as a ledger adjustment (ADM-PAYO-006); customers never pay a surcharge.
- **BR-FEE-007 Tax-inclusive vs exclusive display (product decision):**
  - Menu item, variant, add-on and packaging prices are **GST-exclusive** (matches how local restaurants print menus); GST on food is shown as its own bill line "GST on food (5%)".
  - Delivery fee, platform fee and small-cart fee are configured and displayed **GST-inclusive** (round, predictable numbers for a price-sensitive market); tax is back-calculated for the invoice.
  - [OPEN – Finance/CA to confirm presentation on the tax invoice; LEGAL].
- **BR-FEE-008 Bill structure (order of lines):** Item total → Packaging → GST on food & packaging → Delivery fee (with distance) → Platform fee → Small-cart fee (if any) → Coupon discount (negative) → Round off → **To pay**. Every line has an info tooltip explaining it (dark-pattern compliance, LEG-CP-005).
- **BR-FEE-009 Rounding:** all line amounts computed in paise (round half-up per line, baseline §3); **To pay is rounded to the nearest whole rupee (half-up) with an explicit "Round off" line (±₹0.01–0.50)** — product decision closing baseline OPEN; helps COD change-handling. Round-off is a platform income/expense ledger line, not a fee.
- **BR-FEE-010 Worked example** (illustrative; tax treatment [LEGAL]):

  | Line | Amount (₹) | Note |
  |---|---|---|
  | Chicken Biryani (Full) ×1 | 220.00 | |
  | Paneer 65 ×1 | 160.00 | |
  | **Item total** | **380.00** | |
  | Packaging (2 × ₹10) | 20.00 | per-item mode |
  | GST on food & packaging @5% | 20.00 | ECO pays under §9(5) |
  | Delivery fee (3.1 km) | 30.00 | GST-incl. (taxable 25.42 + GST 4.58) |
  | Platform fee | 5.00 | GST-incl. (taxable 4.24 + GST 0.76) |
  | Coupon WELCOME50 (20% up to ₹50, platform-funded) | −50.00 | on item total only |
  | Round off | 0.00 | |
  | **To pay** | **405.00** | |

  Restaurant settlement for this order (commission 15% on item total): item total 380.00 + packaging 20.00 − commission 57.00 − GST on commission @18% 10.26 − TDS @0.1% of gross 0.40 [LEGAL] = **net 332.34**. Rider pay (3.1 km): 25 + 6 × 1.1 = **₹31.60**. Whether GST on food is computed before or after a *platform-funded* discount is [LEGAL]; the example assumes before (discount does not reduce the restaurant's supply value).

### 11.2 BR-COMM — Commission

- **BR-COMM-001** Commission = restaurant's effective rate × (item total − restaurant-funded discount). Excludes packaging, GST and all customer fees.
- **BR-COMM-002** Default 15%, contract range 10–25% (hard bounds 0–30%), per restaurant, effective-dated, snapshotted on the order.
- **BR-COMM-003** Platform-funded discounts never reduce the restaurant's commission base or payout.
- **BR-COMM-004** GST @18% is charged to the restaurant on commission (and on any other platform service to the restaurant) [LEGAL]; monthly commission invoice (P1).
- **BR-COMM-005** On orders cancelled/rejected before `PICKED_UP` with full refund and fault ≠ customer, no commission is charged.

### 11.3 BR-CAN — Cancellation policy (disclosed at checkout, LEG-CP-004)

| Order state at request | Who can cancel | Customer refund (online) / COD consequence | Restaurant | Rider |
|---|---|---|---|---|
| `PENDING_PAYMENT` | Customer (abandon) / system after 15 min | Any late capture auto-refunded in full | — | — |
| `PLACED` (not yet accepted) | **Customer self-service in app, free** | **100%** | nothing owed | no delivery yet |
| `ACCEPTED`, `PREPARING` | Customer **self-service free within 60 s of placement** (R2), otherwise **via support only**; restaurant only via an ops-mediated issue (R40); admin | Within 60 s of placement: **100%**. After that, customer-fault: refund = To pay − cancellation fee, fee = item total + packaging + GST on food if the restaurant confirms preparation started, else **0** [product decision]; restaurant/platform-fault: **100%** | Customer-fault after preparation: paid item total + packaging − commission (online); for COD: platform pays 50% of item total, capped ₹300, as compensation [ASSUMPTION – OPEN commercial] | if assigned and arrived at restaurant: base pay + waiting pay |
| `READY_FOR_PICKUP`, `PICKED_UP` | **No customer cancellation**; admin only for exceptional cases | If customer then refuses/unreachable → `UNDELIVERABLE`: online = **no refund**; COD = nothing collected, customer gets a COD strike (BR-COD-004) | Paid as delivered (online); COD: compensation as above | Full delivery pay |
| Restaurant reject / accept timeout (`RESTAURANT_UNRESPONSIVE`, 180 s, R1) | Restaurant / system | **100%** | nothing owed; quality metric | if assigned: cancellation pay (doc 13 §6.2) |
| No rider available (`NO_RIDER_AVAILABLE`) | Admin/system after 20 min unassigned past ready [ASSUMPTION] | **100%** | If food prepared: paid as delivered by platform (platform-fault) | — |
| Platform/tech fault | Admin | **100%** + goodwill coupon optional | made whole | made whole |

- **BR-CAN-001** The in-app Cancel button exists in `PENDING_PAYMENT`, `PLACED`, and within **60 s of placement** even if already `ACCEPTED`/`PREPARING` (R2; `ordering.customer_cancel_grace_s`). Past that, "Get help → Cancel my order" creates a high-priority ticket and support/admin cancels with fault attribution.
- **BR-CAN-002** Every cancellation stores `cancel_reason` (code), `cancelled_by` (`CUSTOMER`, `ADMIN`, `SYSTEM` — restaurants never cancel directly, R40; `SYSTEM` is the internal principal, R26) and fault attribution (`customer`, `restaurant`, `rider`, `platform`) — doc 13 owns codes.
- **BR-CAN-003** Coupon usage is restored when an order ends in any non-delivered terminal state, **except** customer-fault `UNDELIVERABLE`, which burns the coupon (RV-048).

### 11.4 BR-REF — Refunds

- **BR-REF-001** Online-paid refunds go **to the original payment instrument via the PA** only; no wallet/credits (stored value is RBI-regulated; out of scope).
- **BR-REF-002** Refund initiated within 24 h of the decision (M-25); system-triggered refunds (reject, `RESTAURANT_UNRESPONSIVE`, no rider, late capture) are initiated automatically within 5 min.
- **BR-REF-003** Customer is shown the refund amount, date initiated and reference (PA refund ID/ARN when available) and the expected timeline (UPI typically 2–5 business days, cards/net banking 5–7 business days [ASSUMPTION – PA-dependent]).
- **BR-REF-004 (R29)** COD orders needing a refund or compensation (e.g. missing item after cash paid) — **the customer chooses**: manual UPI/bank refund to a customer-provided VPA, recorded by finance with UTR and a ledger entry (ADM-PAYO-007), **or** a single-user coupon of equal or higher value. Never coupon-only; no wallet (R9) [LEGAL].
- **BR-REF-005** Refund cost allocation follows fault: restaurant-fault (missing/wrong item, quality) is recovered from restaurant payout (up to the item value incl. GST); rider-fault (spillage, delay caused by rider) borne by platform in V1 (no rider deductions without due process) [ASSUMPTION]; platform-fault borne by platform.
- **BR-REF-006** Partial refunds by item (value of item incl. its share of GST and packaging); maximum refund ≤ amount paid.
- **BR-REF-007** Second approver above ₹500 per refund and above ₹150 per goodwill coupon (R31 family 1; thresholds in `approvals.thresholds`, doc 10).

### 11.5 BR-COD — Cash on Delivery

- **BR-COD-001** COD available only if: city COD flag on; **To pay ≤ ₹1,000** (configurable `cod_max_order`); customer not COD-blocked; customer has at least one verified phone (always true via OTP).
- **BR-COD-002** First-time customers: COD limit ₹600 on their first order [ASSUMPTION – fraud/no-show protection; configurable].
- **BR-COD-003** COD orders are dispatched only to riders eligible for that cash amount (BR-COD-006).
- **BR-COD-004** COD block: **2 customer-fault COD failures** (`UNDELIVERABLE` with `CUSTOMER_UNREACHABLE`/`CUSTOMER_REFUSED`, or customer-fault cancellations after `PREPARING`) within 90 days → COD disabled for that customer; message explains and offers online payment; admin can re-enable.
- **BR-COD-005** Rider collects exactly **To pay** (rounded to rupee, BR-FEE-009); partial payment or changing to online at the door is not supported in V1 (dynamic UPI QR is V1.1, CUS-PAY-007). A customer paying the rider via the rider's personal UPI is **prohibited** in rider terms (cash-equivalent mismatch risk) [product decision].
- **BR-COD-006** Rider cash limit (default ₹2,000): a rider is offered a COD order only if `cash_in_hand + order_to_pay ≤ cash_limit`; at or above the limit the rider can still take prepaid orders. (Refinement of baseline §5 "blocked after limit" — prevents overshoot.)
- **BR-COD-007** Cash-collected ledger entry written atomically with `DELIVERED` (RDR-FLOW-008).
- **BR-COD-008** Settlement netting: at each rider payout cycle, `net = earnings − confirmed cash_in_hand`; if negative, the rider must deposit the difference within the cash-ageing window (BR-COD-009); deposits confirmed by finance (ADM-PAYO-005).
- **BR-COD-009 (R53)** Cash ageing: thresholds are `app_config` keys owned by doc 10 — defaults: cash-in-hand older than **24 h → alert/reminder**; older than **48 h → no new COD offers** (prepaid offers continue) until the deposit is confirmed.
- **BR-COD-010** Daily COD reconciliation: every delivered COD order must have a cash entry (M-20); exceptions (rider claims customer paid less, counterfeit notes, theft) recorded as reason-coded adjustments with approval.

### 11.6 BR-TIME — Timeouts and time rules

- **BR-TIME-001 Restaurant accept window: 180 s** from `PLACED` (R1; `ordering.accept_window_s`, doc 13 §5 T-ACC-TIMEOUT).
- **BR-TIME-002 Escalation (R1, R43):** 0 s alert + push; **alarm/push repeats every 30 s**; **60 s** owner SMS + push; **90 s** order flagged red on the admin live board so ops phones the restaurant and may accept on behalf (ADM-ORD-003, audited) and the customer sees "Taking a little longer…"; **180 s** → `CANCELLED`, `cancelled_by=SYSTEM`, reason `RESTAURANT_UNRESPONSIVE` (not `REJECTED`), prepaid fully refunded, customer notified with "Try another restaurant" suggestions. Automated voice escalation is P1 (NOT-016).
- **BR-TIME-003 Restaurant auto-pause (R1):** each miss → outlet auto-paused **30 min**; **2 consecutive misses** → paused until the owner resumes; owner notified by push + SMS, ops alerted. Separately, no order-receiver heartbeat for 3 min while open → auto-pause (RES-HOUR-005).
- **BR-TIME-004 Payment pending expiry:** 15 min in `PENDING_PAYMENT` → `PAYMENT_FAILED` (PA order expired/cancelled).
- **BR-TIME-005 ETA (shown as a 10-min range):** `max(prep_time, rider_to_restaurant_est) + travel_est + buffer`; speeds/buffers are owned by the ETA model in doc 16 §5 (R48). Lower bound = estimate, upper = estimate + 10, rounded to 5 min.
- **BR-TIME-006 Dispatch start (R7):** the delivery is created at `ACCEPTED`; the first offer is timed at `max(0, prep_time − rider_approach_estimate − buffer)` after acceptance (`dispatch.lead_buffer_min`, doc 13 §4.2).
- **BR-TIME-007 Unassigned escalation:** delivery still `UNASSIGNED/OFFERED` 5 min after dispatch start → admin board flag; 10 min after `READY_FOR_PICKUP` → high-priority flag; 20 min after ready → ops decides cancel (`NO_RIDER_AVAILABLE`) or continue with customer consent.
- **BR-TIME-008 Platform service hours** default 08:00–23:30 IST [ASSUMPTION]; restaurants' hours are intersected with it.

### 11.7 BR-DISP — Dispatch (baseline P11; doc 16/13 own algorithm detail)

- **BR-DISP-001 Eligibility:** rider `AVAILABLE`, approved, not suspended, no active delivery, within the dispatch radius steps (doc 13 §4.2 `dispatch.radius_steps_m`), COD-eligible if COD (BR-COD-006), not previously declined/expired this delivery. **Tiers (R34):** tier 1 = location ≤ 3 min old, ranked by distance; tier 2 = stale ≤ 15 min, reached via push + SSE and ranked after tier 1; offline at 15 min without ping.
- **BR-DISP-002 Ranking:** nearest to restaurant (road-factor distance) first; tie-break by longest idle time (fairness), then higher acceptance rate.
- **BR-DISP-003 Offer timeout 45 s**, one rider at a time, max offers then admin flag (continue cascading in a second round including riders who expired, not those who declined) — parameters in doc 13 §4.2.
- **BR-DISP-004** 3 consecutive expired offers → rider auto-offline.

### 11.8 BR-RPAY — Rider pay

- **BR-RPAY-001** Per delivery: base ₹25 + ₹6/km for road-factor distance restaurant→drop beyond 2 km (pro-rated per 0.1 km).
- **BR-RPAY-002** Pickup distance (rider→restaurant) is unpaid in V1 [ASSUMPTION – flag: riders on national platforms often get first-mile pay; revisit if offer acceptance < 70%].
- **BR-RPAY-003** Waiting pay ₹1/min beyond 10 min from `AT_RESTAURANT` to `PICKED_UP`, cap ₹20 [ASSUMPTION – amount not set in baseline].
- **BR-RPAY-004** Cancelled after rider reached restaurant (not rider-fault): cancellation pay per doc 13 §6.2 (50% of base + waiting pay). Undeliverable (not rider-fault): full pay.
- **BR-RPAY-005** Pay amount shown in the offer equals final pay except waiting-pay additions.
- **BR-RPAY-006 Pilot minimum guarantee (M6, R47):** `MG_TOPUP` ledger adjustment per scheduled peak slot (RDR-EARN-007); amount and conditions [OPEN — business decision].
- **BR-RPAY-007 Manual peak bonus (R30):** ops may add a per-delivery or per-slot rider bonus as a ledger adjustment during bad weather or shortage; never charged to customers.

### 11.9 BR-RATE — Ratings

- **BR-RATE-001** Rating window: **7 days** after `DELIVERED`; only delivered orders can be rated; edits allowed within window.
- **BR-RATE-002** Restaurant displayed rating = mean of restaurant ratings in the last 180 days, shown to one decimal with count; **"New" until ≥ 5 ratings**.
- **BR-RATE-003** Delivery ratings (👍/👎 + tags) are private (rider sees aggregates only; admin sees all).
- **BR-RATE-004** Text reviews: auto-hide if containing phone numbers, URLs or a blocklist of abusive terms (en/te) (profanity filter); admin can hide/unhide with reason; no moderation queue and no restaurant replies in V1 (C14); restaurants cannot delete reviews. Reviews from orders with a refund still count (honest signal).
- **BR-RATE-005** Delivery rating ≤ 2 with tags "Asked for extra cash"/"Rude behaviour" auto-creates an ops review item.

### 11.10 BR-COUP — Coupons and stacking

- **BR-COUP-001 One coupon per order. No stacking** of coupons with each other. Restaurant "offers" in V1 *are* coupons (displayed on listing); hence also not stackable.
- **BR-COUP-002** Discount applies to item total only (never to GST, packaging or fees), except `FREE_DELIVERY` which zeroes the delivery fee (and optionally small-cart fee).
- **BR-COUP-003** `PERCENT` coupons always have a max cap; `FLAT` discount ≤ item total; result never negative.
- **BR-COUP-004** Usage counted at `PLACED`; restored per BR-CAN-003.
- **BR-COUP-005** Per-user limits are enforced on customer account **and** phone number; first-order-only means "no prior `DELIVERED` order on this account/phone".
- **BR-COUP-006** Restaurant-funded coupons require the restaurant's recorded consent (in-app accept or admin-recorded) and are deducted from the restaurant's payout; platform-funded coupons are platform expense.
- **BR-COUP-007** Coupons are validated server-side at quote and at placement; a coupon that became invalid between quote and placement causes re-quote (CUS-CHK-002).

### 11.11 BR-CONT — Contact & privacy rules (V1 without masked calling)

- **BR-CONT-001** Customer ↔ rider calling via `tel:` only during the active window (rider: `PICKED_UP` → end + 15 min; customer: same window). Rider ↔ restaurant during `ASSIGNED` → `PICKED_UP` + 15 min. All call-button taps logged with timestamp (not call content).
- **BR-CONT-002** Restaurants never receive customer phone numbers. Customers never receive restaurant owners' personal numbers (they contact support).
- **BR-CONT-003** Rider terms prohibit saving or contacting customers outside an active delivery; violations → suspension.
- **BR-CONT-004** Customers can provide an alternate number per order (CUS-CHK-007).

### 11.12 BR-OTP — Delivery OTP

- **BR-OTP-001 (R39)** Delivery OTP (4 digits, generated at `PICKED_UP`) is **on for prepaid orders with To pay ≥ ₹300** and **off for COD** (cash handover is the confirmation). The code is stored (encrypted) so the customer app can display it (CUS-TRK-004); seed `dispatch.delivery_code_required` must match (doc 10).
- **BR-OTP-002** OTP visible to the customer in app and push; read out by phone is acceptable; admin override per RDR-FLOW-007.

### 11.13 BR-RES — Restaurant operating rules

- **BR-RES-001** A restaurant must have a valid FSSAI licence/registration to be orderable; expiry auto-pauses.
- **BR-RES-002** Pure-veg restaurants cannot list EGG/NON_VEG items.
- **BR-RES-003** Restaurant-caused cancellations + rejections > 10% over a rolling 7 days (min 20 orders) → ops review; > 20% → auto-pause pending review [ASSUMPTION].
- **BR-RES-004** FSSAI expiry → auto-pause until updated document approved.

### 11.14 BR-PAYOUT — Payout cycles

- **BR-PAYOUT-001 (R53)** Restaurants: weekly cycle Mon 00:00 – Sun 23:59 IST of `DELIVERED` orders (plus adjustments); settlement day is an `app_config` key owned by doc 10 (default **Tuesday**; manual transfer, V1).
- **BR-PAYOUT-002 (R53)** Riders: weekly cycle same window; payout day is an `app_config` key owned by doc 10 (default **Monday** for the previous Mon–Sun); on-demand P1 (RDR-EARN-005).
- **BR-PAYOUT-003** Payouts are made only to verified accounts in the payee's own name (or registered entity name); account changes trigger a 1-cycle hold unless verified by penny-drop [ASSUMPTION – fraud control].
- **BR-PAYOUT-004** Minimum payout ₹100; below carries forward.

### 11.15a BR-COST — Running cost as a business line item

- **BR-COST-001** Cloud and third-party technology costs are budgeted per month in INR and reported per delivered order (M-60, M-61) in the monthly finance pack alongside commission revenue, fees, rider pay, discounts and refunds — i.e. **contribution margin per order is computed after tech cost**.
- **BR-COST-002 Monthly budget envelope — single source is doc 25 §15 (R46).** This PRD no longer restates cost numbers. Doc 25 §15 owns the `closed-pilot` and `public-launch` profile estimates (ex-GST), including the lines added by the review: CERT-In 180-day log archive (M1), counter devices and device lab (M3), support telephony (M9), DLT registration, and the NAT Gateway before Gate B (R28). SMS costs are owned by doc 15; PA fees by doc 14.

- **BR-COST-003** Product choices that drive cost need a cost note in their requirement: e.g. SMS only where push cannot do the job (NOT-*); images thumbnailed and CDN-cached (NFR-PERF-005); rider location pings ≤ 1/30 s, batched, and raw pings retained 30 days (NFR-RET); no paid maps/geocoding in V1; telephony limited to the support phone line (M9) and, only if triggered, the P1 voice escalation (R43) — masked calling deferred; Redis only when needed (baseline P6).
- **BR-COST-004** Fees are **not** raised to cover tech cost during V1 without a product decision; the platform fee (BR-FEE-002, ₹5) is the reference point: target M-60 ≤ platform fee by month 3. The **PA effective rate is a go/no-go criterion at PA selection** (target ≤ 1% blended; UPI as low as negotiable — R46); Product revisits the ₹5 platform fee once the written rate is known.
- **BR-COST-005** Any new V1 scope request must state its recurring cost impact (see `02-v1-scope.md` §6 scope-creep guard).

### 11.15 BR-MENU — Menu & dietary rules

- **BR-MENU-001 Dietary markers.** `VEG` = green square outline with filled green circle; `NON_VEG` = brown square outline with filled brown triangle (FSSAI Labelling & Display Regulations 2020 symbol) [LEGAL – verify]; `EGG` = shown with a distinct "Egg" label and the non-veg symbol family, because under FSSAI definitions food containing egg is non-vegetarian [LEGAL – verify]. The "Pure veg" filter excludes EGG and NON_VEG.
- **BR-MENU-002** Prices must be > ₹0 and ≤ ₹10,000 per item [ASSUMPTION]; variants priced absolutely.
- **BR-MENU-003** Menu price parity with in-store is encouraged but not enforced (no clause in V1) [OPEN – commercial].

---

## 12. Regulatory requirements (LEG-*) [LEGAL]

All items require review by qualified counsel / chartered accountant before launch. Sources in §14.

### 12.1 Food safety — FSSAI

- **LEG-FSSAI-001 [LEGAL] (P0)** rovo's operating entity, as an e-commerce food business operator, obtains its own FSSAI licence (central licence for e-commerce FBOs) before launch; its number is displayed in app footer/help.
- **LEG-FSSAI-002 [LEGAL] (P0)** Every listed restaurant has a valid FSSAI licence/registration captured and verified (RES-ONB-003), displayed on the restaurant page (CUS-MENU-001) and printed on the customer receipt/invoice; no restaurant is listed without it (FSSAI e-commerce guidance; reiterated in Dec 2024 advisory).
- **LEG-FSSAI-003 [LEGAL] (P1)** Display hygiene rating if the restaurant has one (field on profile).
- **LEG-FSSAI-004 [LEGAL] (P0)** Rider induction includes food hygiene (closed bags, no tampering); tamper-evident packaging encouraged for restaurants [ASSUMPTION].
- **LEG-FSSAI-005 [LEGAL] (P0)** Veg/non-veg symbols per BR-MENU-001.

### 12.2 GST and income-tax

- **LEG-GST-001 [LEGAL] (P0)** **Restaurant service supplied through rovo is taxed under CGST §9(5)**: rovo, as the e-commerce operator (ECO), is liable to pay GST (5%, no ITC) on restaurant services supplied through it — **regardless of whether the restaurant is GST-registered** (in force since 1 Jan 2022). Hence GSTIN is optional for restaurants but GST on food is charged on every order.
- **LEG-GST-002 [LEGAL] (P0)** **Delivery fee**: following the 56th GST Council decisions (effective 22 Sep 2025), *local delivery services* supplied through an ECO are notified under §9(5) at **18%**. If rovo itself is the supplier of delivery (riders contracted by rovo), rovo charges 18% GST on the delivery fee in its own right; either way the delivery fee carries 18% GST. Counsel to confirm which characterisation applies to rovo's rider model.
- **LEG-GST-003 [LEGAL] (P0)** Platform fee and small-cart fee: rovo's own service, 18% GST. Commission to restaurants: 18% GST (restaurants may claim ITC if registered — though restaurants under 5% no-ITC scheme generally cannot use it).
- **LEG-GST-004 [LEGAL] (P0)** Tax invoice/bill of supply issued to the customer per order with mandatory particulars (supplier/ECO GSTIN, invoice number series, HSN/SAC, taxable values, tax by component CGST/SGST since intra-state Telangana, FSSAI number of restaurant). Invoice numbering must be sequential per financial year (doc 10/14 to design).
- **LEG-GST-005 [LEGAL] (P0)** rovo registers for GST in Telangana before launch (ECO under §9(5) must register irrespective of turnover).
- **LEG-GST-006 [LEGAL] (P1)** TCS under CGST §52 — not applicable to supplies on which the ECO pays tax under §9(5); confirm applicability to any other supplies.
- **LEG-TAX-001 [LEGAL] (P1)** Income-tax TDS by e-commerce operators on payments to e-commerce participants (historically §194-O, **0.1% from 1 Oct 2024**) — confirm current section/rate under the Income-tax Act, 2025 regime in force from 1 Apr 2026, threshold applicability to individual/HUF restaurant owners, and whether it applies to riders (likely not as they are service providers to rovo, possibly other TDS sections apply).

### 12.3 Privacy — DPDP Act 2023 and DPDP Rules 2025

The DPDP Rules were notified on 13–14 Nov 2025 with phased commencement; most data-fiduciary obligations become enforceable **18 months later (≈ 13 May 2027)**. rovo will **comply from day one** (cheaper to build in than retrofit).
- **LEG-DPDP-001 [LEGAL] (P0)** Itemised, plain-language privacy notice in English and Telugu at the point of collection (CUS-AUTH-003, partner onboarding), listing data, purposes, rights, grievance contact, and how to withdraw consent.
- **LEG-DPDP-002 [LEGAL] (P0)** Consent records (what, when, notice version); withdrawal as easy as giving (marketing toggle; account deletion).
- **LEG-DPDP-003 [LEGAL] (P0)** Data principal rights: access (CUS-AUTH-009), correction (profile edit), erasure (CUS-AUTH-007), grievance redressal (contact published), nomination (P2 – via support); response within the period prescribed by the Rules [LEGAL].
- **LEG-DPDP-004 [LEGAL] (P0)** Purpose limitation & minimisation enforced by product rules (RES-ORD-008, RDR-CONT-004, NFR-PRIV-*).
- **LEG-DPDP-005 [LEGAL] (P0)** Reasonable security safeguards (encryption, access control, logging, ≥ 1-year log retention) — NFR-SEC/NFR-RET.
- **LEG-DPDP-006 [LEGAL] (P0)** Children: rovo does not knowingly process data of persons under 18 — sign-up requires an 18+ declaration; verifiable parental consent flows are out of scope [LEGAL – confirm adequacy; note student persona aged 17 at junior college may attempt to sign up].
- **LEG-DPDP-007 [LEGAL] (P0)** Personal-data breach: intimate affected users and the Data Protection Board without delay, detailed report within 72 h (per Rules) — process in doc 19/24.
- **LEG-DPDP-008 [LEGAL] (P0)** Processors (PA, SMS provider, hosting, observability incl. Grafana Cloud/Faro) under contracts with DPDP-aligned clauses; no PII sent to error tracking. Erasure is driven by a per-table/bucket erasure map (M15).

### 12.4 Telecom — TRAI DLT

- **LEG-TRAI-001 [LEGAL] (P0)** All SMS (OTP, transactional, service) sent via a DLT-registered principal entity (rovo's entity), approved header (sender ID, e.g. `ROVOIN` [ASSUMPTION]) and approved content templates (en and te — Unicode templates count more characters per segment), through a DLT-compliant provider; marketing SMS only to consenting users and with promotional headers (marketing SMS out of scope V1).
- **LEG-TRAI-002 [LEGAL] (P0)** URLs/callback numbers inside SMS must be whitelisted in templates per current TRAI directions.

### 12.5 Consumer protection — Consumer Protection Act 2019, E-Commerce Rules 2020, Dark Patterns Guidelines 2023

- **LEG-CP-001 [LEGAL] (P0)** Display legal name, geographic address of HQ, customer care and grievance officer contact on the platform (help/footer).
- **LEG-CP-002 [LEGAL] (P0)** Appoint a **grievance officer**; publish name, designation and contact (CUS-SUPP-003).
- **LEG-CP-003 [LEGAL] (P0)** **Acknowledge every consumer complaint within 48 h and redress within one month** (ticketing SLA timers, ADM-TKT-001/003; auto-acknowledgement).
- **LEG-CP-004 [LEGAL] (P0)** Clear disclosure, before purchase, of cancellation, refund and return policy (BR-CAN/BR-REF) and of seller (restaurant) details: name, address, FSSAI no., GSTIN if any; no cancellation charges unless equivalent charges are borne by the platform/seller when they cancel unilaterally [LEGAL – check symmetry requirement against BR-CAN].
- **LEG-CP-005 [LEGAL] (P0)** **Total price display including all fees** before payment; fees shown on listings (delivery fee) and itemised in cart; no drip pricing, basket sneaking (nothing pre-added — e.g. no pre-ticked donation/tip/insurance), false urgency, confirm-shaming or subscription traps (CCPA Dark Patterns Guidelines, 30 Nov 2023).
- **LEG-CP-006 [LEGAL] (P0)** Ratings/reviews are not manipulated: no paid/fake reviews, no selective suppression of negative reviews (BR-RATE-004); BIS IS 19000:2022 online consumer review practices considered [ASSUMPTION – voluntary standard].
- **LEG-CP-007 [LEGAL] (P0)** No discrimination or ranking manipulation undisclosed: ranking parameters for listings disclosed in help (BR: CUS-DISC-001 ranking rule).
- **LEG-CP-008 [LEGAL] (P0)** Refunds for cancellations processed within a reasonable period (BR-REF-002/003).

### 12.6 Payments — RBI

- **LEG-RBI-001 [LEGAL] (P0)** All online customer payments flow through an RBI-authorised payment aggregator; rovo does not hold customer funds outside the PA's nodal/escrow arrangements; settlements to rovo's current account, then payouts to restaurants/riders (baseline P10). Counsel to confirm that rovo collecting the full order value and settling restaurants does not make rovo itself a payment aggregator (marketplace exemption) [LEGAL]. The design supports both PA split settlement and collect-and-payout (R25); **the legal opinion is required before Phase-2 week 4** — if split settlement, PA linked-account KYC joins restaurant onboarding (RES-ONB-005) (R35).
- **LEG-RBI-002 [LEGAL] (P0)** No wallet/stored value (PPI) in V1.

### 12.7 Gig & platform workers

- **LEG-GIG-001 [LEGAL] (P0)** **Code on Social Security, 2020** (bulk in force from 21 Nov 2025): aggregators may be required to contribute 1–2% of annual turnover (capped at 5% of amounts paid to gig workers) once the contribution date is notified, and to **register each new gig/platform worker on the designated portal in real time or daily** (Social Security (Central) Rules 2026, as reported) — product must capture the rider data fields the portal needs (RDR-ONB-007) and produce an export (ADM-RPT-005) (M5, P0) [LEGAL – confirm applicability thresholds for a small aggregator].
- **LEG-GIG-002 [LEGAL] (P1)** **Telangana Gig and Platform Workers Bill 2025** (draft April 2025): welfare-board registration of aggregators/workers, welfare fee, data sharing — confirm enacted status as of launch; design rider data export accordingly.
- **LEG-GIG-003 [LEGAL] (P0)** Rider agreement establishes independent-contractor relationship, transparent pay computation (BR-RPAY shown in app), grievance mechanism for riders, and no arbitrary deductions (BR-REF-005).
- **LEG-GIG-004 [LEGAL] (P1)** Accident insurance cover for riders while on delivery (common market practice; may be mandated by state law) — decision required before Gate A (R47; group accident cover from day one recommended by the review) [OPEN – ops/finance].

### 12.8 Other

- **LEG-OTH-001 [LEGAL] (P0)** Alcohol, tobacco and other restricted items cannot be listed (menu moderation blocklist).
- **LEG-OTH-002 [LEGAL] (P0)** Open-source licence compliance (Apache-2.0 NOTICE, third-party licences, map tile attribution, font licences).
- **LEG-OTH-003 [LEGAL] (P0)** Map/tile provider terms permit commercial use at our volume (baseline P13).
- **LEG-OTH-004 [LEGAL] (P1)** Accessibility: Rights of Persons with Disabilities Act 2016 accessibility standards for digital services — WCAG 2.2 AA (NFR-A11Y-001) expected to satisfy [ASSUMPTION].
- **LEG-OTH-005 [LEGAL] (P0)** IT Act intermediary obligations (if rovo is treated as an intermediary for reviews/UGC): grievance mechanism, takedown handling — covered by ADM-TKT and BR-RATE-004.

---

## 13. Open questions

| # | Question | Owner | Needed by |
|---|---|---|---|
| OQ-01 | Confirm GST characterisation of delivery fee (ECO §9(5) vs rovo own supply), treatment of platform-funded discounts on §9(5) taxable value, and invoice format for GST-inclusive fees (BR-FEE-007/010). | Finance + CA [LEGAL] | Before doc 14 freeze |
| OQ-02 | Does collecting full order value and settling restaurants keep rovo outside PA licensing (marketplace model)? | Counsel [LEGAL] | Before PA contract |
| OQ-03 | Compensation to restaurants for customer-fault COD cancellations (50% capped ₹300) — acceptable cost? | Product + Finance | Pilot |
| OQ-04 | Delivery OTP threshold (≥ ₹300 prepaid, decided by R39) — validate with pilot fraud data. | Product + Ops | Pilot |
| OQ-05 | Police verification for riders: mandatory pre-approval or within 30 days? | Ops + Counsel | Rider onboarding start |
| OQ-06 | Bicycle / low-speed EV riders without DL: allow in V1? | Ops | Rider onboarding start |
| OQ-07 | ~~Heartbeat gating default~~ — **resolved by R1**: no heartbeat for 3 min while open → auto-pause (P0). | — | closed |
| OQ-08 | ~~Auto-move `ACCEPTED → PREPARING`~~ — **resolved by R3**: after 60 s or on tap. | — | closed |
| OQ-09 | Urdu locale demand — measure in pilot (survey). | Product | Post-pilot |
| OQ-10 | Rider first-mile pay & waiting-pay amount (BR-RPAY-002/003). | Ops + Finance | Pilot |
| OQ-11 | Applicability thresholds of Social Security Code aggregator obligations and Telangana Gig Workers Act for a small aggregator. | Counsel [LEGAL] | Before launch |
| OQ-12 | Service hours 08:00–23:30 — validate late-night demand (biryani points open past midnight). | Ops | Pilot |
| OQ-13 | Approve monthly cloud budget (doc 25 §15, R46); Multi-AZ trigger decided by R32. | Finance + DevOps | Before staging build-out |
| OQ-14 | Pilot rider minimum-guarantee amount, slots and eligibility (RDR-EARN-007, R47). | Ops + Finance [OPEN] | Before Gate A |
| OQ-15 | Rider accident insurance (LEG-GIG-004, R47). | Ops + Counsel [LEGAL] | Before Gate A |
| OQ-16 | Restaurant money flow: PA split settlement vs collect-and-payout (R25/R35). | Counsel [LEGAL] | Before Phase-2 week 4 |

## 14. Sources (accessed 2026-10-04)

- GST on local delivery via ECO under §9(5) at 18%, effective 22 Sep 2025: Deccan Herald, "Online food delivery charges to rise when new GST rules take effect" — https://www.deccanherald.com/amp/story/business%2Feconomy%2Fonline-food-delivery-charges-to-rise-when-new-gst-rules-take-effect-3713486 ; Inc42, "Zomato, Swiggy deliveries to get costlier with new 18% GST" — https://inc42.com/buzz/zomato-swiggy-deliveries-to-get-costlier-with-new-18-gst ; BDO India, "Swiggy, Zomato & the GST 2.0 squeeze" — https://www.bdo.in/en-gb/news/2025/swiggy,-zomato-the-gst-2-0-squeeze
- DPDP Rules notified Nov 2025, phased over 12–18 months (to ≈ 13 May 2027): The Week (PTI), 14 Nov 2025 — https://www.theweek.in/wire-updates/business/2025/11/14/del148-biz-dpdp-rules-ld-govt.html ; Mondaq, "DPDP Act compliance mandate" — https://www.mondaq.com/india/privacy-protection/1708830/dpdp-act-compliance-mandate
- Consumer Protection (E-Commerce) Rules 2020 — grievance officer, 48 h acknowledgement, 1 month redressal: IndiaLaw summary — https://www.indialaw.in/blog/consumer-protection-e-commerce-rules/ ; Mondaq — https://mondaq.com/india/dodd-frank-consumer-protection-act/985606/the-consumer-protection-e-commerce-rules-2020
- CCPA Guidelines for Prevention and Regulation of Dark Patterns 2023 (drip pricing, basket sneaking): SCC Online, 4 Dec 2023 — https://www.scconline.com/blog/post/2023/12/04/ccpa-notifies-guidelines-for-prevention-and-regulation-of-dark-patterns-2023-legal-news/ ; JSA — https://www.jsalaw.com/newsletters-and-updates/ccpa-issues-guidelines-for-prevention-and-regulation-of-dark-patterns-2023/
- FSSAI e-commerce FBO obligations (display seller FSSAI licence, Dec 2024 advisory): FSSAI advisory PDF — https://fssai.gov.in/upload/advisories/2024/12/674efa161d756Adobe%20Scan%203%20Dec%202024.pdf ; Lexplosion — https://lexplosion.in/fssai-re-iterates-compliance-norms-for-e-commerce-food-platforms/
- TDS §194-O reduced to 0.1% from 1 Oct 2024: ClearTax — https://cleartax.in/s/section-194o ; TaxGuru — https://taxguru.in/income-tax/section-194-o-amendment-lower-tds-rate-e-commerce-payments.html
- Code on Social Security 2020 aggregator obligations (in force from 21 Nov 2025; 1–2% turnover contribution; worker registration): Taxmann — https://www.taxmann.com/post/blog/analysis-aggregator-obligations-code-on-social-security/ ; K&S Partners — https://ksandk.com/labour-employment/gig-workers-epf-code-social-security-2026/
- Draft Telangana Gig and Platform Workers Bill 2025: PRS India — https://prsindia.org/bills/state-legislative-briefs/the-draft-telangana-gig-and-platform-workers-registration-social-security-and-welfare-bill-2025

Not independently verified in this pass (flagged [LEGAL]/[ASSUMPTION] inline): FSSAI symbol specifics for non-veg/egg; CERT-In 6-hour reporting and 180-day log retention (CERT-In Directions, 28 Apr 2022); DPDP Rules 72-hour breach report and 1-year log retention; GST record-keeping periods; Income-tax Act 2025 section renumbering.
