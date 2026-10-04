# 03 — User Personas

| Field | Value |
|---|---|
| **Purpose** | Give every architect and designer a shared, concrete picture of who uses rovo in Mahabubnagar, what they are trying to get done, and what that implies for design. Requirements in doc 01 and scope decisions in doc 02 cite these personas; UX journeys (04–07) are written from their point of view. |
| **Owner** | Product Architect |
| **Status** | Draft v1.1 — reconciled with review (31) and rulings R1–R48, 2026-10-04 |
| **Depends on** | `00-planning-baseline.md` (roles, vocabulary, §8–§9 rulings R1–R48); `01-product-requirements.md` (requirement IDs referenced in "Design implications"). |
| **Feeds** | 04–07 (journeys/workflows), 17–18 (frontend & PWA device targets), 20 (test personas & device matrix), 29 (UAT participants). |

**Changes in v1.1**
- Primary apps renamed to the four apps/hosts of **R14** (`app.`, `restaurant.`, `rider.`, `admin.`).
- P3: ops-assisted phone ordering (M8) and staffed support line (M9); combined consent screen.
- P4: accept ladder per **R1** (repeat 30 s, owner SMS 60 s, ops call 90 s, cancel 180 s, 30-min pause), mandatory device-heartbeat auto-pause, rovo-provisioned counter device (M3), device-bound session (**R44**), voice escalation P1 (**R43**).
- P5: 7/30-day trends deferred to V1.1 (C13) — noted as a retention risk.
- P6/P7: dispatch tiers (**R34**), pilot minimum guarantee (M6), manual peak bonus (**R30**), fleet sized to demand (**R47**), gig-worker registration [LEGAL] (M5), insurance decision.
- P8/P9: maker-checker narrowed to the **R31** families, COD compensation choice (**R29**), bulk menu import (M7), bank-statement matching, finance FTE from Gate B.

**How these personas were built.** They are *proto-personas*: synthesised from knowledge of small-city Telangana markets, national food-delivery patterns and the constraints in the baseline. **They are not yet validated by research** `[ASSUMPTION]`. Before Gate A (doc 02 §5.1), ops should run ≥ 5 interviews per persona group (customers, restaurants, riders) in Mahabubnagar and update this document. Names are illustrative; any resemblance to real people is unintended.

**Why nine personas** (brief suggested 6–8): each maps to a distinct RBAC role or a distinct accessibility profile that changes design decisions; merging any two would hide a conflicting need (e.g. part-time vs full-time riders differ on payouts and alerts; ops vs finance differ on tools and permissions).

| # | Persona | Role | Primary app | Short label |
|---|---|---|---|---|
| P1 | Sravani | `CUSTOMER` | customer app (`app.`) | Budget student, UPI-native |
| P2 | Ravi & Lakshmi | `CUSTOMER` | customer app (`app.`) | Family dinner orderers |
| P3 | Narasimha Reddy | `CUSTOMER` | customer app (`app.`) + support phone line | Elderly, Telugu-first, low tech |
| P4 | Venkatesh | `RESTAURANT_OWNER` | restaurant app (`restaurant.`) | Small family restaurant, shared phone |
| P5 | Farhan | `RESTAURANT_OWNER` / manager | restaurant app (`restaurant.`) + desktop | Established biryani house, multi-outlet-ready |
| P6 | Kiran | `RIDER` | rider app (`rider.`) | Part-time student rider |
| P7 | Mahesh | `RIDER` | rider app (`rider.`) | Full-time rider |
| P8 | Anusha | `ADMIN_OPS` + `ADMIN_SUPPORT` | admin app (`admin.`) | Ops & support agent |
| P9 | Suresh | `ADMIN_FINANCE` | admin app (`admin.`) | Finance admin |

---

## P1 — Sravani, the budget-conscious student

| | |
|---|---|
| **Age / life** | 20, B.Sc. student; lives in a girls' hostel near the college area (e.g. near Palamuru University / Yenugonda) [ASSUMPTION – locality illustrative]; from a village in the district |
| **Device** | Redmi/Realme budget phone, 4 GB RAM, Android 13, storage nearly full (photos, WhatsApp media) — uninstalls apps she rarely uses |
| **Connectivity** | Jio prepaid 1.5 GB/day; hostel Wi-Fi unreliable; data often exhausted by night |
| **Language** | Telugu at home, reads English comfortably; types Telugu words in English letters ("biryani", "pulihora") |
| **Payment** | UPI only (PhonePe/Google Pay); occasional failed UPI because of bank server issues; rarely carries cash |
| **Spend** | ₹120–₹250 per order; orders 2–3×/week, often with roommates splitting via UPI |

> *"If the delivery fee is more than the chai, I'm not ordering."*

**Goals:** cheap, filling food fast; find deals; order late in the evening when the mess food is bad.
**Frustrations:** fees that appear only at checkout; small-cart fees; UPI payment "pending" with money debited and no order; apps that eat storage and data; minimum order values.
**Jobs-to-be-done**
- When the hostel dinner is bad, I want to find something under ₹200 delivered in 30 min, so I can eat without going out at night.
- When my UPI payment hangs, I want to know immediately whether money left my account and whether I'll get it back, so I don't pay twice.
- When there's a coupon, I want to see whether my cart qualifies and by how much, so I can add a cheap item to cross the threshold.

**Design implications**
- PWA, no install required; tiny bundle and offline-capable shell (NFR-PERF-001, -002); "Add to Home screen" prompted only after a successful order.
- Fees visible on restaurant cards and cart (CUS-DISC-002, BR-FEE-008); small-cart nudge (CUS-CART-005); filter "cost for one" and sort by cost (CUS-SRCH-002/003).
- Coupon eligibility with exact shortfall message (CUS-COUP-001/002).
- Clear payment pending/failed/refund screens and auto-refund of late captures (CUS-PAY-003/004, M-24); switch to COD on retry if allowed (CUS-PAY-006).
- Typo-tolerant, Latin-script search via trigram + synonym table (CUS-SRCH-001); romanisation/transliteration later (CUS-SRCH-006, C17).
- Age: may be 17 in junior college — sign-up requires 18+ declaration (LEG-DPDP-006), shown on the same single consent screen as the privacy notice (CUS-AUTH-003); a real policy risk noted for counsel.

---

## P2 — Ravi & Lakshmi, the working family ordering dinner

| | |
|---|---|
| **Age / life** | Ravi 37, bank officer; Lakshmi 34, government school teacher; two children (8, 12); live in a newer colony (e.g. Padmavathi Colony / New Town) [ASSUMPTION]; Ravi's parents live nearby (see P3) |
| **Device** | Ravi: Samsung Galaxy M-series, 6 GB RAM; Lakshmi: Vivo mid-range. Both on Airtel postpaid + home fibre |
| **Connectivity** | Good at home; spotty on the move |
| **Language** | Both bilingual; Lakshmi prefers Telugu UI; kids read English |
| **Payment** | UPI mostly; COD when ordering for the grandparents or when the order is large and they're unsure of the restaurant |
| **Spend** | ₹500–₹1,200 per order, 1–2×/week (weekend dinner, festival days, guests) |

> *"Biryani for six people should be simple. Tell me when it'll come and don't make me call anyone."*

**Goals:** reliable, on-time family meals; know the food is from a trusted, hygienic place; veg and non-veg in one order for mixed preferences; sometimes send food to parents.
**Frustrations:** late deliveries with no explanation; missing items (raita, extra gravy) with painful refunds; confusing customisation (half/full, gravies); having to explain the address every time.
**Jobs-to-be-done**
- When guests come, I want to order a large mixed veg/non-veg meal with the right portions, so everyone eats at the same time.
- When something is missing, I want to report it with a photo and get the money back without a phone fight.
- When my parents need dinner, I want to order to their address with their phone number as the contact.

**Design implications**
- Variants (Half/Full, Family pack) and add-on groups with min/max made obvious (CUS-CUST-001/002); veg/egg/non-veg markers per item and add-on (CUS-CUST-003, BR-MENU-001).
- Accurate ETA range and proactive late message (CUS-CHK-005, CUS-TRK-002, M-08).
- FSSAI number and ratings visible (CUS-MENU-001, LEG-FSSAI-002).
- Missing/wrong item flow with photos and item-level refund (CUS-SUPP-001, BR-REF-006); for COD orders the customer chooses a manual UPI refund or a coupon (BR-REF-004, R29).
- Saved addresses with required landmark + pin (building/street optional, R13) and per-address instructions; "ordering for someone else" (CUS-ADDR-004/005, CUS-CHK-007).
- COD up to ₹1,000 covers most family orders (BR-COD-001) — validate in pilot; larger orders nudged to UPI.

---

## P3 — Narasimha Reddy, elderly, Telugu-first, less tech-savvy

| | |
|---|---|
| **Age / life** | 66, retired Telugu teacher; lives with his wife in the old town (e.g. near the clock tower area) [ASSUMPTION]; son (Ravi, P2) set up his phone |
| **Device** | Older Android (3 GB RAM, Android 10/11), large font setting enabled, cracked screen protector |
| **Connectivity** | Wi-Fi from son's connection at home; mobile data rarely on |
| **Language** | Telugu only for comfort; can read some English slowly; prefers voice calls to typing |
| **Payment** | Cash (COD) strongly preferred; PhonePe installed by son but he fears "wrong UPI payments" |
| **Spend** | ₹150–₹350; orders tiffin/meals occasionally, more when wife is unwell |

> *"అన్నీ తెలుగులో ఉంటే నేనే ఆర్డర్ చేస్తాను."* ("If everything is in Telugu, I'll order myself.")

**Goals:** order familiar food (idli, meals, pesarattu) himself without bothering his son; pay cash; talk to a human if confused.
**Frustrations:** small text; English-only menus; OTP screens that time out; icons without words; being rushed by countdowns; unexpected charges.
**Jobs-to-be-done**
- When I want lunch delivered, I want to find "meals" in Telugu and order with cash, so I don't need my son's help.
- When I can't find the OTP or a screen confuses me, I want to call someone who speaks Telugu.

**Design implications**
- Telugu UI complete and natural (NFR-I18N-001, NFR-A11Y-008); Telugu dish names shown (RES-MENU-006, CUS-MENU-002).
- Large text support to 200%, 44 px targets, icons with labels, colour + shape markers (NFR-A11Y-002/003/004).
- OTP with auto-read, 5-min validity and generous resend (CUS-AUTH-001); no aggressive timers on the customer side.
- COD with clear "keep ₹X ready" message and rupee-rounded total (CUS-PAY-002, BR-FEE-009).
- Delivery OTP not required for COD (BR-OTP-001) — avoids a confusing step at the door.
- Support phone line, staffed in all service hours with Telugu-speaking agents (CUS-SUPP-003, M9); help centre in Telugu.
- **Ops-assisted phone ordering** (ADM-ORD-011, P1, M8): he can call the support line and an agent places a COD order for him, confirming the bill aloud; he gets the order code by SMS.
- Automated voice ordering deferred (doc 02) but research-worthy for this persona.

---

## P4 — Venkatesh, small family restaurant owner, shared phone

| | |
|---|---|
| **Age / business** | 45; runs "Sri Venkateswara Tiffins & Meals" (illustrative) with his wife and one helper; 25 seats; breakfast tiffins 6:30–11:00, meals 12:00–15:00, evening tiffins 17:30–21:30 |
| **Device** | One Samsung Galaxy A-series phone (Android 12) shared between him, his wife and his son (who handles "online things" after college); kept at the cash counter, often in a pocket |
| **Connectivity** | Jio 4G; weak signal inside the kitchen; no Wi-Fi |
| **Language** | Telugu; limited English reading; uses WhatsApp voice notes and YouTube |
| **Digital maturity** | Low; has been approached by national aggregators but found onboarding and commissions hard; FSSAI *registration* (not licence) obtained via an agent; not GST-registered |
| **Money** | Cash and UPI QR (soundbox) at the counter; checks bank balance by SMS; distrusts delayed settlements |

> *"Order vasthe nenu chusthanu, kani phone lo emaina kashtam aithe evaru chepthaaru?"* ("I'll see the order when it comes — but if the phone gives trouble, who will tell me?")

**Goals:** extra orders in the slow afternoon/evening hours without extra staff; understand exactly how much he'll receive and when; not be punished for a missed notification during a rush.
**Frustrations:** complicated menus to set up; documents and forms in English; notifications that don't ring; money deducted for reasons he doesn't understand; changing prices in an app.
**Jobs-to-be-done**
- When an order arrives during the breakfast rush, I want a loud, obvious alert and one big button to accept, so I don't lose it.
- When idli batter runs out, I want to mark idli "not available" in one tap until tomorrow.
- When Monday comes, I want to see a simple statement in Telugu that shows what I earned, what rovo kept and when the money arrives.

**Design implications**
- **Assisted onboarding:** ops creates restaurant, menu and Telugu names on his behalf, digitising his printed menu with the ops-side bulk CSV import (RES-ONB-002, ADM-REST-003, RES-MENU-009, M7); FSSAI registration accepted, documents photographed as images (RES-ONB-003, R38); GSTIN optional because rovo pays GST under §9(5) (LEG-GST-001).
- Looping alert sound, vibration, push repeated every 30 s, full-screen card; one large Accept with default prep time (RES-ORD-001/002, NFR-A11Y-005/006).
- Escalation per R1: owner SMS at 60 s, ops manual call at 90 s with **accept on behalf** (BR-TIME-002, ADM-ORD-003); at 180 s the order is cancelled as `RESTAURANT_UNRESPONSIVE` (not counted as his rejection, but as a miss); one miss pauses the outlet 30 min, two consecutive misses pause it until he resumes (BR-TIME-003). An automated voice call is a P1 option if pilot data shows it is needed (R43).
- If his shared phone is unsuitable, rovo provides a pre-configured counter device (RES-ONB-008, M3) with a device-bound session that does not expire mid-service (X-006, R44).
- Multiple devices on one account so the son's phone also rings (RES-PROF-004).
- One-tap out-of-stock with "back tomorrow" default (RES-AVAIL-001).
- Restaurant never handles cash or customer phone numbers (RES-ORD-001/008).
- Statement per order with plain-language labels in Telugu; masked account; weekly payout by Wednesday (RES-PAYO-001, BR-PAYOUT-001).
- Mandatory device-heartbeat auto-pause: if no order device is alive for 3 min while open, the outlet pauses so customers don't order into a dead phone (RES-HOUR-005, R1).

---

## P5 — Farhan, manager of an established biryani restaurant

| | |
|---|---|
| **Age / business** | 33; operations manager of a well-known family biryani & kebab restaurant near the main road (illustrative); 60 seats; ~150 dine-in/takeaway covers/day; second outlet planned in New Town |
| **Device** | Dedicated Android tablet at the dispatch counter + his own phone; desktop PC with Tally in the back office |
| **Connectivity** | Broadband + Wi-Fi; 4G backup |
| **Language** | English, Telugu, Urdu/Dakhni [ASSUMPTION]; comfortable with apps |
| **Digital maturity** | High; already on national aggregators; GST-registered; FSSAI state licence; prints KOTs |
| **Money** | Watches margins closely; reconciles aggregator payouts in Excel; resents high commissions and opaque deductions |

> *"I'll put my best items on rovo if your commission is fair and your statements actually match my bank."*

**Goals:** incremental orders at a lower commission than national apps; accurate statements; data on top items and peak times; minimise rejections and ratings damage; prepare for multi-outlet.
**Frustrations:** orders arriving with wrong rider timing (food waits and gets cold, or rider waits and complains); unexplained refunds charged to the restaurant; inability to export data.
**Jobs-to-be-done**
- When a busy Friday dinner hits, I want to adjust prep time per order and see which rider is coming, so food and rider arrive together.
- When the week closes, I want a CSV of every order with commission, GST and adjustments that ties to the bank credit.
- When I open the second outlet, I want to copy the menu and manage both from one login.

**Design implications**
- Prep-time choice and extensions; rider status on order card (RES-ORD-002/005, RES-PREP-002).
- Dispatch timing tuned to prep time (BR-TIME-006) — waiting pay and food temperature both matter.
- Printable KOT (RES-ORD-009, committed P1).
- Statement with per-order commission/GST/TDS/adjustments; CSV; payout disputes (RES-PAYO-001/002/004); refunds charged to restaurant only when restaurant-fault and visible (BR-REF-005).
- Today/this-week summary in V1 (RES-ANLY-001); 7/30-day trends and top items are **deferred to V1.1** (RES-ANLY-002, C13) — a retention risk for this persona, mitigated by the CSV of orders (RES-ANLY-003, stretch) and ops sharing a monthly summary [ASSUMPTION].
- Multi-outlet: one owner, many `restaurant` rows; outlet switcher and menu copy (RES-MENU-010) — data model must allow from day one.
- GSTIN captured and shown on statements/invoices (RES-ONB-003); monthly commission tax invoice for ITC (BR-COMM-004).

---

## P6 — Kiran, part-time student rider

| | |
|---|---|
| **Age / life** | 21; final-year B.Com student; lives with parents; rides 17:00–22:00 on weekdays, longer on weekends |
| **Vehicle** | Father's Honda Shine (RC in father's name), helmet; valid DL |
| **Device** | Redmi Note-series mid-range (Android 13, 6 GB RAM); cheap phone mount; power bank |
| **Connectivity** | Airtel prepaid 2 GB/day; dead spots in some lanes |
| **Language** | Telugu + English; reads English UI fine |
| **Money** | UPI for everything; wants weekly money for fees and phone recharge |

> *"I just need clear orders, clear pay, and no fights about cash."*

**Goals:** earn ₹4,000–₹6,000/month in flexible hours; avoid long waits at restaurants; get home safely by 22:30.
**Frustrations:** offers that disappear before he can read them while riding; not knowing how far the drop is; COD customers without change; app logging him out or losing location; not being paid for waiting.
**Jobs-to-be-done**
- When an offer arrives while I'm riding, I want enough time and the key facts (pickup distance, drop area, pay, COD/prepaid) to decide safely.
- When I go online for a few hours, I want the app to keep me online without draining my battery.
- When I finish for the day, I want to see exactly what I earned and how much cash I owe.

**Design implications**
- Offer card with pickup distance, drop locality, estimated pay and payment type; 45 s timeout; loud alert (RDR-ASSIGN-001/002/006).
- Gig-worker registration fields captured once at onboarding (RDR-ONB-007, M5) [LEGAL]; 30-day sliding session so he isn't logged out between shifts (X-006, R44).
- RC not in rider's name allowed with declaration (RDR-ONB-002).
- Foreground location every 30–60 s, wake lock, clear "keep app open" banner; when backgrounded he still gets offers as a tier-2 rider via push for up to 15 min (R34); auto-offline only after 15 min without any ping (RDR-AVAIL-002) or 3 consecutive expired offers (RDR-AVAIL-003).
- Waiting pay after 10 min (BR-RPAY-003); per-delivery pay transparency (RDR-EARN-001).
- Exact-amount COD with rupee rounding; cash-in-hand view and limit (BR-FEE-009, RDR-EARN-002, BR-COD-006).
- SOS button for night rides (RDR-FLOW-013).
- One-handed, high-contrast, slide-to-confirm actions (NFR-A11Y-005).

---

## P7 — Mahesh, full-time rider

| | |
|---|---|
| **Age / life** | 29; married, one child; previously rode for a national platform in Hyderabad, moved back to Mahabubnagar; rides 10–11 h/day, 6 days a week |
| **Vehicle** | Own Bajaj Platina (on EMI) and, occasionally, a rented EV scooter |
| **Device** | Samsung Galaxy M-series (Android 13); dual SIM (Jio + Airtel) to handle dead spots |
| **Language** | Telugu; basic English; prefers Telugu UI |
| **Money** | Daily fuel and EMI pressure; carries cash; wants fast payouts; experienced with cash-deposit disputes on previous platform |

> *"Last company held my money for two weeks over a ₹300 cash mismatch. Show me every rupee."*

**Goals:** maximise deliveries per hour; steady weekly income (target ₹18,000–₹22,000/month [ASSUMPTION]); fair dispatch; no unexplained deductions.
**Frustrations:** idle time while others get orders; being offered far pickups; deposit confirmations that take days; customers not answering at the door; heat and rain.
**Jobs-to-be-done**
- When I'm idle, I want to get the next nearby order fairly, so my hours aren't wasted.
- When I've collected ₹1,800 in cash, I want to deposit it quickly and see my limit reset, so I keep getting COD orders.
- When a customer doesn't answer, I want a clear procedure that protects my pay.

**Design implications**
- Dispatch fairness tie-break by longest idle time (BR-DISP-002); tiered eligibility so a rider waiting at a stand with the screen off is still reachable (BR-DISP-001, R34).
- **Earnings viability:** at pilot volumes (≈ 30–80 orders/day) his target income is not reachable from per-trip pay alone (31 RV-055/070). Mitigations: fleet sized to demand (≈ 1 online rider per 3 peak-hour orders, R47), a pilot **minimum guarantee** per scheduled peak slot (`MG_TOPUP`, RDR-EARN-007, M6) and a manual peak bonus in bad weather (R30); accident-insurance decision [LEGAL] (LEG-GIG-004).
- Projected cash-limit check so he isn't stuck over the limit mid-shift (BR-COD-006); 80% warning (NOT-011); deposit declaration with UTR and fast confirmation (RDR-EARN-003, ADM-PAYO-005); weekly netting (BR-COD-008); on-demand payout (RDR-EARN-005, committed P1).
- Undeliverable flow with waiting/call evidence and ops confirmation; full pay when not rider-fault (RDR-FLOW-009, BR-RPAY-004).
- No deductions without due process (BR-REF-005, LEG-GIG-003).
- Telugu rider UI, readable in sunlight (NFR-A11Y-005).
- History with per-delivery detail for disputes (RDR-HIST-001); personal data of customers hidden after delivery (RDR-CONT-004).

---

## P8 — Anusha, ops & support agent

| | |
|---|---|
| **Age / role** | 26; B.Tech; rovo ops/support agent (roles `ADMIN_OPS` + `ADMIN_SUPPORT`, city-scoped); works shifts covering lunch or dinner peaks |
| **Device** | Office laptop (Chrome) with a second monitor; work Android phone for calls; headset |
| **Connectivity** | Office broadband + phone hotspot backup |
| **Language** | Telugu, English, some Hindi/Urdu |
| **Context** | During peaks handles 10–20 live orders, phone calls with restaurants (accept nudges), rider exceptions, and customer tickets simultaneously |

> *"Show me what's about to go wrong, not what already went wrong."*

**Goals:** keep every order moving; solve customer issues in one touch; avoid mistakes with refunds; hand over cleanly to the next shift.
**Frustrations:** switching between many screens; no context when a customer calls; unclear authority on refunds; repeated typing of the same replies; not knowing which rider is closest.
**Jobs-to-be-done**
- When an order is unaccepted at 90 s, I want it highlighted with the restaurant's phone number, so I can call and accept on their behalf.
- When no rider accepts, I want a list of riders sorted by distance with their status, so I can assign in seconds.
- When a customer reports a missing raita, I want to see the order, the photo and issue a refund for that item within my limit.

**Design implications**
- Live board with SLA colour states and flags; filters; live updates (ADM-ORD-001, ADM-DASH-001).
- Accept on behalf with reason; manual assign/reassign with distance-sorted riders; call shortcuts with logging (ADM-ORD-003/005/009).
- Ticket inbox linked to order with photos, canned responses in en/te, refund limits and a second approver only above ₹500 refunds / ₹150 goodwill (R31) (ADM-TKT-001/002/004, ADM-ORD-007).
- Answers the support phone line during her shift (M9); places assisted COD orders for callers (ADM-ORD-011, M8); imports restaurant menus by CSV during onboarding (RES-MENU-009, M7).
- Order timeline showing every event and actor (ADM-ORD-002); internal notes for shift handover (ADM-ORD-008).
- Masked PII by default; reveal with reason (ADM-AUTH-003, NFR-PRIV-002).
- Desktop-first, keyboard-friendly admin (NFR-A11Y-001).

---

## P9 — Suresh, finance admin

| | |
|---|---|
| **Age / role** | 41; B.Com, semi-qualified accountant; part-time finance admin for rovo (role `ADMIN_FINANCE`) and works with an external CA for GST filing |
| **Device** | Windows desktop, Excel, Tally, net banking with bulk-upload |
| **Language** | English for accounting; Telugu conversationally |
| **Context** | Weekly payout runs (restaurants Wednesday, riders Tuesday), daily COD deposit confirmations, monthly GST working papers, PA settlement reconciliation |

> *"If the ledger, the bank and the payment gateway don't agree, I don't release payouts."*

**Goals:** accurate, auditable payouts; zero unexplained COD variance; GST data ready for the CA; separation of duties.
**Frustrations:** manual matching of UTRs; refunds issued by support without approval; numbers that change after statements are published; exports missing columns.
**Jobs-to-be-done**
- When the weekly cycle closes, I want a draft payout batch whose totals tie to the ledger and the statements, so I can approve and upload one bank file.
- When riders declare deposits, I want to match them to bank credits and confirm in bulk.
- When the month ends, I want a GST working report split by component (§9(5) restaurant service, delivery fee, platform fee, commission).

**Design implications**
- Settlement run → draft batch → maker-checker approval → bank CSV → UTR recording (ADM-PAYO-001); holds and adjustments, with a second approver for the R31 families only (payout release, refunds/goodwill/adjustments above threshold, commission/fee changes, payout-detail changes, role grants) (ADM-PAYO-004/006).
- Records manual UPI refunds with UTR when a COD customer chooses money over a coupon (ADM-PAYO-007, R29); posts minimum-guarantee top-ups (`MG_TOPUP`) and peak bonuses as adjustments.
- Workload: part-time is enough for the pilot; one finance FTE from Gate B (RV-068).
- PA settlement import and reconciliation (ADM-PAYO-003); ledger views with running balance (ADM-PAYO-002).
- Deposit confirmation queue with bank-statement CSV import and automatic UTR matching, plus cash ageing report (ADM-PAYO-005, M-22).
- GST working report and CSV exports with stable columns (ADM-RPT-001/002); TDS report if applicable (ADM-PAYO-008).
- Effective-dated config and immutable statements once published; corrections via adjustments in the next cycle (NFR-AUD-003, RES-PAYO-001).
- Monthly cost-per-order report including cloud bill (BR-COST-001, M-60/61).

---

## Cross-persona summary

### Device & connectivity matrix (feeds doc 18 device lab and doc 20 test matrix)

| Persona | Device class | RAM | Android | Network | Locale | Test priority |
|---|---|---|---|---|---|---|
| P1 Sravani | Budget | 3–4 GB | 12–14 | Jio 4G, data caps | en (te strings) | High |
| P2 Ravi/Lakshmi | Mid-range | 6 GB | 13–14 | Fibre Wi-Fi / Airtel | en / te | Medium |
| P3 Narasimha | Old low-end, large font | 2–3 GB | 10–11 | Home Wi-Fi | **te** | **High (a11y)** |
| P4 Venkatesh | Low-mid, shared | 3–4 GB | 12 | Weak 4G in kitchen | **te** | **High (alerts)** |
| P5 Farhan | Android tablet + desktop | 4 GB+ | 12+ | Wi-Fi | en | Medium |
| P6 Kiran | Mid-range | 6 GB | 13 | Airtel 4G, dead spots | en / te | **High (offers)** |
| P7 Mahesh | Mid-range | 4–6 GB | 13 | Dual-SIM 4G | **te** | **High (COD)** |
| P8 Anusha | Laptop Chrome | — | — | Broadband | en / te | Medium |
| P9 Suresh | Windows desktop | — | — | Broadband | en | Medium |

iOS share is small [ASSUMPTION]; iOS Safari ≥ 16.4 is supported for customers (Web Push only when installed to Home Screen) but not a primary target for the restaurant and rider apps.

### Persona → key requirement map

| Need | P1 | P2 | P3 | P4 | P5 | P6 | P7 | P8 | P9 |
|---|---|---|---|---|---|---|---|---|---|
| Fee transparency (BR-FEE-008, LEG-CP-005) | ● | ● | ● | | | | | | |
| Telugu-complete UI (NFR-I18N-001) | ○ | ● | ● | ● | | ○ | ● | ○ | |
| Accessibility / large text (NFR-A11Y) | | ○ | ● | ● | | ● | ● | ○ | |
| COD (BR-COD) | ○ | ● | ● | | | ● | ● | ● | ● |
| Unmissable alerts (RES-ORD-001, RDR-ASSIGN-006) | | | | ● | ● | ● | ● | | |
| Assisted onboarding / accept on behalf | | | | ● | | | | ● | |
| Statements & reconciliation | | | | ● | ● | ● | ● | | ● |
| Support & refunds (CUS-SUPP, ADM-TKT) | ● | ● | ● | | ○ | | ○ | ● | ○ |
| Low data / low-end perf (NFR-PERF) | ● | | ● | ● | | ● | ● | | |

● primary need · ○ secondary need

### Anti-personas (we do not design V1 for)

- **The bulk caterer** ordering 200 meals for an event (needs quotes/invoices) — out of scope (doc 02 §2.2).
- **The out-of-town user** ordering into Mahabubnagar from Hyderabad for family — supported only through "ordering for someone else" with a local address; no multi-city browsing.
- **The coupon farmer** creating many accounts for first-order discounts — actively defended against (BR-COUP-005, ADM-TKT-005).
- **The cloud kitchen with 10 virtual brands** — allowed as separate restaurants but no brand-group tooling in V1.

### Research plan to validate personas `[OPEN – Ops lead]`

1. 5 customer interviews per customer persona (students, families, elderly) incl. 1 observed order on their own phone.
2. 8 restaurant visits (mix of P4/P5 profiles) observing a lunch rush and testing an alert prototype in a noisy kitchen.
3. 6 rider interviews (part-time and full-time), including a ride-along at dinner peak.
4. Shadow 2 ops/support shifts during the pilot.
5. Update this document and doc 01 assumptions; record changes in doc 30.
