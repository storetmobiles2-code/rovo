# rovo — System Knowledge Graph

| | |
|---|---|
| **Updated** | 2026-10-21 |
| **Source** | Generated from the live artifacts: `backend/migrations/*` (94 tables), `openapi/openapi.yaml` (API tag groups), and the module boundaries in `docs/implementation/00-implementation-charter.md` + planning docs 08/10/11/13. |
| **Scope** | The nodes and edges of V1: actors → apps → API groups → domain modules → owned tables → domain events → external systems. |

This is a hand‑verifiable knowledge graph (not an auto‑generated black box): every table below is grepped from the migrations, every API group from the OpenAPI tags. Regenerate the mappings with `scripts/knowledge-graph.sh` (greps the same sources).

## 1. System graph (actors → apps → API → modules → stores)

```mermaid
flowchart LR
  CUS([Customer]):::actor
  RES([Restaurant]):::actor
  RID([Rider]):::actor
  ADM([Admin]):::actor

  subgraph Apps["PWAs (web/apps)"]
    AC[customer]:::app
    AR[restaurant]:::app
    ARi[rider]:::app
    AA[admin]:::app
  end

  EDGE[[CDN / Caddy · same-origin /api/v1 · SSE]]:::edge
  API[[Go API · net/http + OpenAPI contract]]:::svc
  WORK[[Go Worker · River jobs + timers]]:::svc
  PG[("PostgreSQL 17 + PostGIS<br/>94 tables · River queue · LISTEN/NOTIFY")]:::db
  S3[(S3 / MinIO<br/>media + KYC)]:::db

  PA[[Payment Aggregator]]:::ext
  SMS[[SMS / OTP]]:::ext
  PUSH[[Web Push]]:::ext

  CUS-->AC; RES-->AR; RID-->ARi; ADM-->AA
  AC & AR & ARi & AA --> EDGE --> API
  API <--> PG
  WORK <--> PG
  API --> S3
  WORK --> PA & SMS & PUSH
  PA -. webhooks .-> API
  API -. SSE/NOTIFY .-> EDGE

  classDef actor fill:#0C1825,color:#fff,stroke:#0C1825;
  classDef app fill:#F6F4EC,color:#0C1825,stroke:#C2410C;
  classDef edge fill:#fff,color:#0C1825,stroke:#64748B;
  classDef svc fill:#C2410C,color:#fff,stroke:#0C1825;
  classDef db fill:#1E2A3A,color:#fff,stroke:#0C1825;
  classDef ext fill:#fff,color:#0C1825,stroke:#F85000,stroke-dasharray:4 3;
```

## 2. Modules ↔ owned tables (verified from migrations)

Each module owns its tables; cross‑module references are logical except FKs to `cities`/`users` and money‑path FKs to `orders` (ruling R41).

```mermaid
flowchart TB
  subgraph identity["identity"]
    i1[users]; i2[user_roles]; i3[otp_challenges]; i4[sessions]; i5[refresh_tokens]; i6[admin_credentials]; i7[devices]; i8[rate_limit_buckets]
  end
  subgraph users["users"]
    u1[customer_profiles]; u2[customer_addresses]; u3[user_consents]; u4[erasure_requests]; u5[waitlist]
  end
  subgraph geo["geo"]
    g1[cities]; g2[localities]; g3[zones]
  end
  subgraph catalog["catalog"]
    c1[restaurants]; c2[restaurant_users]; c3[restaurant_devices]; c4[restaurant_kyc_documents]; c5[restaurant_operating_hours]; c6[menu_categories]; c7[menu_items]; c8[item_variants]; c9[addon_groups]; c10[addons]
  end
  subgraph pricing["pricing"]
    p1[fee_configs]; p2[tax_rules]; p3[quotes]
  end
  subgraph promotions["promotions"]
    pr1[coupons]; pr2[coupon_targets]; pr3[coupon_redemptions]
  end
  subgraph ordering["ordering"]
    o1[orders]; o2[order_items]; o3[order_charges]; o4[order_status_history]
  end
  subgraph payments["payments"]
    pm1[payments]; pm2[payment_events]; pm3[refunds]; pm4[transfers]; pm5[pa_settlements]; pm6[recon_exceptions]
  end
  subgraph dispatch["dispatch"]
    d1[riders]; d2[rider_availability]; d3[rider_location_pings]; d4[deliveries]; d5[delivery_offers]; d6[delivery_status_history]; d7[sos_events]
  end
  subgraph ledger["ledger"]
    l1[ledger_accounts]; l2[ledger_journals]; l3[ledger_postings]; l4[commission_plans]; l5[payout_batches]; l6[payouts]; l7[cod_deposits]; l8[invoices]
  end
  subgraph ratings["ratings"]
    r1[ratings]; r2[reviews]; r3[rating_aggregates]
  end
  subgraph support["support"]
    s1[support_tickets]; s2[ticket_messages]
  end
  subgraph notifications["notifications"]
    n1[notification_templates]; n2[notifications]; n3[notification_deliveries]; n4[push_subscriptions]
  end
  subgraph adminaudit["adminaudit"]
    a1[approval_requests]; a2[reason_codes]; a3[app_config]; a4[feature_flags]; a5[audit_logs]; a6[idempotency_keys]; a7[file_objects]
  end
  subgraph reporting["reporting"]
    rp1[report_exports]
  end
```

**Full table counts by migration** (94 total): geo/users/identity 18 · catalog 16 · pricing/promotions 6 · ordering 4 · payments 8 · dispatch 9 · ledger 12 · ratings/support/notifications 9 · admin/platform 9 · (+ extensions/reference/grants).

## 3. Domain‑event graph (the golden flow)

Events are River jobs inserted transactionally (the outbox, R22/R42). Edges = "emits → consumed by".

```mermaid
flowchart LR
  O1{{order.placed}}:::ev --> NOTIF[notifications]
  O1 --> PAY[payments]
  PAY --> O2{{payment.captured}}:::ev --> LED[ledger]
  O3{{order.accepted}}:::ev --> DISP[dispatch]
  ordering([ordering]) --> O1 & O3
  DISP --> E1{{offer.sent}}:::ev --> NOTIF
  E1 --> E2{{offer.expired}}:::ev --> DISP
  DISP --> E3{{delivery.picked_up}}:::ev --> NOTIF
  DISP --> E4{{delivery.delivered}}:::ev
  E4 --> LED
  E4 --> RAT[ratings]
  E4 --> NOTIF
  CODc{{cod.collected}}:::ev --> LED
  LED --> SET{{settlement.run}}:::ev --> PAYOUT[payout batch → maker-checker]
  classDef ev fill:#F85000,color:#fff,stroke:#0C1825;
```

**Reads:** placing an order notifies + starts payment; capture posts to the ledger; acceptance creates a delivery and starts the offer cascade (45 s → expire → next rider); delivery completion posts rider/commission entries, opens the rating window, and notifies; COD collection posts cash‑in‑hand; the weekly settlement run produces payout batches that require two‑person approval.

## 4. Order ↔ delivery state machines (reference)

Canonical names (`00` §2); full transition tables in planning doc 13.

```mermaid
stateDiagram-v2
  direction LR
  [*] --> PENDING_PAYMENT
  PENDING_PAYMENT --> PLACED: paid / COD
  PENDING_PAYMENT --> PAYMENT_FAILED
  PLACED --> ACCEPTED: restaurant accepts (≤180s)
  PLACED --> CANCELLED: customer / timeout(SYSTEM)
  ACCEPTED --> PREPARING: +60s / tap
  PREPARING --> READY_FOR_PICKUP
  READY_FOR_PICKUP --> PICKED_UP
  PREPARING --> PICKED_UP: rider (skipped-ready)
  PICKED_UP --> DELIVERED
  DELIVERED --> [*]
  ACCEPTED --> REJECTED
```

## 5. API surface (OpenAPI tag groups → module)

| API group(s) | Module(s) |
|---|---|
| Public, Config | geo, catalog |
| Auth, Me, Customer Addresses | identity, users |
| Checkout | pricing, promotions, ordering |
| Orders, Support | ordering, support |
| Payments, Webhooks | payments |
| Partner Onboarding/Restaurant/Menu/Orders | catalog, ordering |
| Rider, Rider Deliveries | dispatch |
| Admin Dashboard/Orders/Users/Restaurants/Riders/Finance/Coupons/Geography/Support/Approvals/Reports/Audit | adminaudit + all modules |
| Realtime | realtime (SSE) |
| Health | platform/health |

## 6. External systems (trust boundaries)

| Node | Direction | Notes |
|---|---|---|
| Payment Aggregator | out + inbound webhook | HMAC‑verified, fetch‑to‑confirm; `api.` host receives webhooks only (R55) |
| SMS / OTP | out | India mobiles, rate‑limited, fake in dev |
| Web Push | out | VAPID, fake in dev |
| S3 / MinIO | out | media (public) + KYC (private, SSE‑KMS) |
| OTel / Grafana | out | traces/metrics/logs |

---

*Generated by the rovo implementation team from the repo's own migrations and OpenAPI spec. Re‑run `scripts/knowledge-graph.sh` after schema or contract changes to refresh the tables in §2 and §5.*
