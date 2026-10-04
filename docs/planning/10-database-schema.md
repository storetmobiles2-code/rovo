# 10 — Database Schema (logical + physical)

| | |
|---|---|
| **Purpose** | Authoritative logical and physical data model for rovo V1. It covers tables, columns, types, constraints, indexes, module ownership, PII classification, retention and erasure map, partitioning triggers, seed data and `app_config` defaults, migration rules and multi-city readiness. |
| **Owner** | Backend Architect |
| **Status** | Draft v1.1 — reconciled with review (31) and rulings R1–R48, 2026-10-04 |
| **Depends on** | 00 planning baseline (vocabulary, money, IDs, §4a managed cloud); 08 system architecture (module map, boundary rules, events as River jobs via `InsertManyTx` (R42), SSE via `NOTIFY`); 12 auth/RBAC (identity model, sessions, maker-checker, audit, KYC storage); 13 order state machine (status values, history rows); 14 payment architecture (PA flows, journal templates, invoices: the detail lives there); 16 delivery zone architecture (geo semantics); 19 threat model (encryption, retention) |
| **Consumed by** | 11 API spec, 13, 14, 16, 20 testing, 22/23 deployment/backup, 26 repo structure, 27 backlog |

**Changes in v1.1**
- **Table count 81 → 90.** Removed `rider_shifts` (C12) and `outbox_events` (R42/C8). Added the M4 tables `rate_limit_buckets` (R21), `transfers`, `pa_settlements`, `pa_settlement_lines`, `recon_exceptions` (14, R25), `erasure_requests` (M15), `leads`, `waitlist`, `staff_invites`, `sos_events`, `contact_tap_log` (§1.5).
- **R18 (register row 37):** fee slabs re-seeded to 10 km road distance with `[from, to)` bounds (8–10 km ₹60). `fee_configs.max_serviceable_distance_m` (8,000, ambiguous) → `max_serviceable_radius_m` (7,000, **straight-line**). Fee/commission default values are owned by 16 §6.5 (R48).
- **R16 (row 34):** `delivery_offers.status` adds `REVOKED`, with a status ↔ `close_reason` CHECK.
- **R30 / C1:** surge removed (zone surge columns, `surge_fee_paise`, the `SURGE_FEE` component).
- **R41 / RV-006:** real FKs to `orders` from `payments`, `refunds`, `deliveries`, `invoices`, `transfers`, `ledger_postings` and `payout_items` (DB-D04).
- **R42 / C8 / RV-039:** no `outbox_events` table and **no day-one partitioning**. Retention is by batched-delete jobs; partition a table when it passes ~10 M rows (§14). **C6 / RV-028:** `audit_logs` is append-only through grants and a trigger; no hash chain.
- **R31 / RV-081:** `approval_requests.action_type` limited to the five action families, plus break-glass self-approval with 24 h post-review.
- **R29 / RV-047:** COD compensation by manual UPI refund with UTR (`refunds.channel`) or coupon; the "COD never refunded" rule is removed. **R39 / RV-049:** `orders.requires_delivery_code` + encrypted stored code; seed flag on.
- **R44 / M10:** device-bound sessions (`sessions.binding`, `restaurant_devices.session_id`). **R14:** four app audiences (`customer`, `restaurant`, `rider`, `admin`).
- **RV-045 (row 66):** ledger account codes and the `gstin_state` / `normal_side` columns follow 14 §10.2. **M6:** `ADJUSTMENT` journals carry `adjustment_type` incl. `MG_TOPUP`, `PEAK_BONUS`.
- **M5:** gig-worker registration fields on `riders` + `GIG_WORKER_REGISTRATION` export [LEGAL]. **M15:** erasure map per table/bucket (§13.1) [LEGAL]. **M11:** periodic jobs are catch-up jobs (13 §5.2).
- **C12** `ON_BREAK` removed; **C14** review moderation queue → profanity filter + admin hide; **C16** coupon `SHARED` funding and `CUISINE`/`USER` targets removed; **C20** tax rules seeded by migration (no CRUD/approval).
- **RV-030 (row 68):** PA webhook payloads are PII-redacted at ingest; the raw body goes to a 180-day object-store prefix. **RV-005 (row 41):** payout cut-off uses journal `occurred_at` from the app clock (R19). **R38:** KYC files are images with SSE-KMS, without app-layer envelope encryption. **Rows 56–58:** commission 0–3,000 bps, line quantity 1–20, prep time 5–90.

> **Design artifacts only.** The SQL below is illustrative DDL for review. It is not a migration file. Phase 2 turns it into `goose` migrations. DDL is grouped by module for reading, **not** in executable order. Migrations order it by dependency (e.g. `users` before `zones.paused_by`, `fee_configs` before `quotes`). Deferred FKs are added with `ALTER TABLE`.

---

## 0. Decision summary

| ID | Decision | Why |
|---|---|---|
| DB-D01 | **PostgreSQL 17 + PostGIS 3.4+** (18 only if the provider offers it with PostGIS; no 18-only features, R22), run as a **managed service** in an India region (RDS/Aurora PostgreSQL, Cloud SQL, or Azure Database for PostgreSQL Flexible Server, per §4a). Only these extensions are used: `postgis`, `btree_gist`, `citext`, `pg_trgm`, `pgcrypto`. All five are on the supported-extension lists of the three managed offerings `[ASSUMPTION — DevOps re-verifies on the chosen provider and version; on Azure each extension must be allow-listed in `azure.extensions`]`. **Not used:** `h3-pg`, `pg_partman`, `pg_cron`, `timescaledb`, `pgvector`. They are not universally available on managed services, and we don't need them. There is no day-one partitioning (C8); retention runs as batched-delete jobs (§14). | Portability across clouds, per baseline §4a rule 1. |
| DB-D02 | **UUIDv7 generated in the application** (`platform/idgen`). The schema never relies on PG 18's native `uuidv7()`, which keeps PG 17 viable. No DB defaults are used for PKs. | PG 18's `uuidv7()` exists ([postgresql.org docs](https://www.postgresql.org/docs/18/functions-uuid.html), accessed 2026-10-04), but managed-service PG 18 + PostGIS availability varies `[OPEN — DevOps]`. |
| DB-D03 | **Enums are `text` + `CHECK`**, not PG `ENUM` types. Values are `UPPER_SNAKE` everywhere, including `veg_type` (`VEG`, `NON_VEG`, `EGG`). | Adding or removing values is a cheap `NOT VALID` constraint swap. Uppercase matches the canonical status names. |
| DB-D04 | **One schema (`public`) and module-owned tables.** FKs within a module, plus FKs to `cities` and `users` from anywhere, **plus FKs to `orders` on money paths (R41):** `payments`, `refunds`, `transfers`, `deliveries`, `invoices`, `ledger_postings` and `payout_items`. Other cross-module references are plain `uuid` columns marked `-- ref:` in DDL and checked by a nightly orphan-check job plus integration tests. | Integrity where money is at stake. Go import boundaries still keep modules separate; dropping an FK if a module is ever extracted is a one-line migration. |
| DB-D05 | **Translations in `*_i18n jsonb`** (`{"te": "…"}`) next to a canonical `name` column, which holds English or what the owner typed. This is not `name_te` columns. | Adding Hindi/Urdu/Kannada for the next city needs no migration. sqlc maps it to a typed Go struct via override. |
| DB-D06 | **No server-side cart in V1** (Lead ruling 12, following 17). The cart lives on the device. `POST /api/v1/cart/quote` is stateless for the client: it takes the cart lines and returns a **signed, short-TTL `quoteId`**. The server **persists the priced quote** (`quotes`, TTL 10 min) so that order creation needs only `quoteId` + `Idempotency-Key` and uses exactly what the customer saw (08 §5.1). | One less mutable aggregate, and offline-friendly. Server price authority is kept via the persisted quote. Lost: cross-device cart and abandoned-cart analytics, deferred to V1.1 as an expand-only `carts` table if wanted. |
| DB-D07 | **Orders snapshot everything they need**: restaurant identity, address, item names/prices/addons, fee config, commission rate, tax rules. History never joins back to mutable catalog rows. | Legal invoices, disputes and settlement must not change when a menu changes. |
| DB-D08 | **Status columns over soft delete.** No blanket `deleted_at`. Lifecycle entities have `status`. Catalog rows referenced by history get `archived_at`. Ephemeral rows are hard-deleted. Users are **anonymised**, not deleted (DPDP erasure vs. tax retention, §13). | Soft-delete everywhere leaks into every query and keeps PII forever. |
| DB-D09 | **Double-entry ledger.** Journals are append-only. Every journal balances to zero (deferred constraint trigger). Corrections are made by reversal journals only. `ledger_account_balances` holds locked running balances for gating checks. | Money correctness (08 §7.2, 14). |
| DB-D10 | **Zones are `geometry(MultiPolygon,4326)`; points are `geography(Point,4326)`.** Point-in-zone uses `ST_Covers(zone.boundary, point::geometry)`. Distances use geography or Go haversine (16). | Admins draw on a Web-Mercator map with straight edges, so planar polygons match what was drawn. Geography is used where metres matter. |
| DB-D11 | **No day-one partitioning (C8).** All tables are plain tables. Retention runs as a catch-up job doing batched `DELETE`s by an indexed timestamp (§13, §14). A table is partitioned only when it passes **~10 M rows** (expand/contract recipe in §14). | V1 volume is tiny. A missed partition-maintenance run would push rows into a DEFAULT partition and block later creates (RV-039). |
| DB-D12 | **No brand/chain table in V1.** One `restaurants` row per outlet. A future `brands` table is an expand-only migration (new table + nullable `restaurants.brand_id`). Multi-outlet owners are already modelled through `user_roles`. | YAGNI for a single-city launch. Nothing in V1 blocks it. |
| DB-D13 | Object storage is referenced only by **provider-neutral `(bucket, object_key)`** in `file_objects`, never by URL. URLs (CDN or presigned) are built at request time from config. | Works with S3, GCS (S3 interop), Azure via an S3 gateway, and MinIO locally (§4a). |
| DB-D14 | **No event table (R42).** A domain event is the args of River jobs inserted with `InsertManyTx`, one per subscriber, in the business transaction (River is the outbox, R22). `processed_events` keeps handler idempotency. | One less hop and one less store for the same fact (RV-002). |
| DB-D15 | **Append-only audit without a hash chain (C6).** `audit_logs` has no `UPDATE`/`DELETE` grant for `rovo_app`, plus the `forbid_mutation()` trigger. Tamper evidence (hourly batch sealing to a WORM bucket) is V1.1. | Per-row chaining serialised all audited writes (RV-028). |
| DB-D16 | **Parameter ownership (R48).** This doc owns the seed mechanism and the `app_config` key set (§15). Timer/threshold **values** are owned by 13 §5.1; fee, commission and rider-pay **values** by 16 §6.5. DDL `DEFAULT`s mirror those values for readability only. Business SQL takes `:now` from the app clock; `DEFAULT now()` is only for technical `created_at`/`updated_at` (R19). | One source per value (RV-086). |

---

## 1. Conventions

### 1.1 Naming

| Thing | Rule | Example |
|---|---|---|
| Tables | `snake_case`, plural | `menu_items`, `order_status_history` (history tables keep a singular noun) |
| PK | `id uuid` (UUIDv7, app-generated) | |
| FK / ref column | `<singular>_id` | `restaurant_id` |
| Timestamps | `<event>_at timestamptz` (UTC) | `accepted_at` |
| Local wall-clock time | `time` without tz, evaluated in `cities.timezone` | `opens_at` |
| Money | `<name>_paise bigint` + `currency char(3)` on order/ledger-level rows | `total_paise` |
| Rates | basis points `int`, `_bps` (1% = 100 bps) | `commission_bps` |
| Distances / durations | `_m` metres `int`; `_s` seconds; `_min` minutes | `max_delivery_radius_m` |
| Booleans | `is_` / `has_` / adjective | `is_available` |
| Encrypted columns | `<name>_enc bytea` + `pii_key_id text` (KEK version); search via `<name>_last4` or `<name>_hmac` (blind index) | `account_number_enc` |
| JSONB | `_snapshot` (frozen copy), `_i18n` (translations), `metadata` | `restaurant_snapshot` |
| Indexes | `ix_<table>__<cols>`; unique `ux_`; check `ck_`; exclusion `ex_`; FK `fk_` | `ix_orders__city_active` |

### 1.2 Standard columns and triggers

- `created_at timestamptz NOT NULL DEFAULT now()` on every table. `updated_at` on mutable tables, maintained by trigger `set_updated_at()`.
- **Aggregate roots** (`orders`, `deliveries`, `restaurants`, `menu_items`, `payments`, `coupons`, `zones`, `fee_configs`, `payouts`, `support_tickets`) carry `version int NOT NULL DEFAULT 1`. The API exposes it as `ETag` (11 §1.9). Transitions use CAS on `(status, version)` (13 §7).
- **Append-only tables** (`*_status_history`, `ledger_journals`, `ledger_postings`, `audit_logs`, `payment_events`, `user_consents`) get trigger `forbid_mutation()`, and the app role lacks `UPDATE/DELETE` on them.

```sql
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS btree_gist;   -- uuid/text equality inside EXCLUDE constraints
CREATE EXTENSION IF NOT EXISTS citext;       -- case-insensitive email / coupon codes
CREATE EXTENSION IF NOT EXISTS pg_trgm;      -- fuzzy search on restaurant / dish / locality names
CREATE EXTENSION IF NOT EXISTS pgcrypto;     -- digest() in data migrations/backfills only; NOT used for PII encryption (done in app, KMS-backed)

CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at := now(); RETURN NEW; END $$;

CREATE FUNCTION forbid_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION '% is append-only', TG_TABLE_NAME USING ERRCODE = 'insufficient_privilege'; END $$;
-- attached as: CREATE TRIGGER trg_<t>_immutable BEFORE UPDATE OR DELETE ON <t> FOR EACH ROW EXECUTE FUNCTION forbid_mutation();
```

### 1.3 Database roles (managed-service friendly: no superuser required)

| Role | Grants | Used by |
|---|---|---|
| `rovo_owner` | Owns all objects. `CREATE` on schema. Member of the provider's admin role (`rds_superuser` / `cloudsqlsuperuser` / `azure_pg_admin`) only for `CREATE EXTENSION`. | `rovo migrate` one-off job (goose + River migrations) |
| `rovo_app` | `SELECT, INSERT, UPDATE, DELETE` on module tables. **`SELECT, INSERT` only** on append-only tables. No DDL. | `rovo api`, `rovo worker` |
| `rovo_report` | `SELECT` on `*_report` views and a read replica | reporting exports, BI |

Row-level security is **not** used in V1 (12 §8). City scoping is enforced in repository queries (12 AUTH-D09).

### 1.4 Soft delete vs. status policy

| Category | Policy | Tables |
|---|---|---|
| Lifecycle entities | `status` column with explicit states, never deleted | restaurants, riders, users, coupons, zones, payouts, support_tickets |
| Catalog referenced by history | `archived_at` (hidden from menus, kept for FK integrity inside the module and for analytics) | menu_categories, menu_items, item_variants, addon_groups, addons |
| Customer-owned convenience data | **hard delete** on user request (orders keep snapshots) | customer_addresses, push_subscriptions |
| Financial, order and legal records | never deleted inside the retention period; PII redacted after the dispute window (§9) | orders, payments, refunds, ledger_*, invoices |
| Ephemeral / security | hard delete by TTL sweeper | otp_challenges, idempotency_keys, quotes, processed_events, refresh_tokens (expired), rate_limit_buckets, staff_invites (expired) |

### 1.5 Module ownership (08 §3)

| Module (Go package) | Tables |
|---|---|
| `geo` | cities, localities, zones, **waitlist** |
| `users` | users, customer_profiles, customer_addresses, user_consents, **erasure_requests** |
| `identity` | user_roles, otp_challenges, sessions, refresh_tokens, admin_credentials, admin_recovery_codes, phone_change_requests, devices, **rate_limit_buckets** (shared facility, schema owned here as in 08 §3.2) |
| `catalog` | restaurants, restaurant_users, restaurant_devices, restaurant_kyc_documents, restaurant_operating_hours, restaurant_closures, cuisines, restaurant_cuisines, menu_categories, menu_items, item_variants, addon_groups, addons, menu_item_addon_groups, **staff_invites**, **leads** |
| `pricing` (quote & pricing) | quotes, fee_configs, tax_rules |
| `promotions` | coupons, coupon_targets, coupon_redemptions |
| `ordering` | orders, order_items, order_charges, order_status_history |
| `payments` | payments, payment_attempts, payment_events, refunds, **transfers**, **pa_settlements**, **pa_settlement_lines**, **recon_exceptions** |
| `dispatch` | riders, rider_availability, rider_kyc_documents, rider_location_pings, deliveries, delivery_offers, delivery_status_history, **sos_events**, **contact_tap_log** |
| `ledger` (& settlement) | ledger_accounts, ledger_journals, ledger_postings, ledger_account_balances, commission_plans, payout_accounts, payouts, payout_items, cod_deposits, invoices, invoice_sequences |
| `ratings` | ratings, reviews, rating_aggregates |
| `notifications` | notifications, notification_deliveries, notification_templates, push_subscriptions |
| `support` | support_tickets, ticket_messages |
| `admin` (& audit) | audit_logs, approval_requests, reason_codes, feature_flags, app_config, report_exports |
| `platform` (shared infra, importable by all) | processed_events, idempotency_keys, file_objects |
| River (library-managed, via `rivermigrate`) | river_job, river_leader, river_queue, river_client, river_migration… (not counted) |

**Total: 90 application tables** (plus River's own), counted from the DDL below: v1's 81 − `rider_shifts` (C12) − `outbox_events` (R42) + 11 new tables (M4, in **bold** above). Not added as tables, on purpose: bulk menu CSV import (M7) runs as a synchronous validate-then-apply request over a `file_objects` upload (11 §2.6); ops-assisted orders (M8) are ordinary `orders` with `placed_via = 'OPS_ASSISTED'`; bank-statement uploads reuse `pa_settlements` with `source = 'BANK_STATEMENT'`; 12's `partner_applications` are `restaurants` in `DRAFT/SUBMITTED` and `riders` in `APPLIED/UNDER_REVIEW`.

### 1.6 Name reconciliation with sibling drafts

08 and 12 proposed indicative names. This document is authoritative (both docs say so). Mapping:

| Proposed in | Name there | Canonical name here | Note |
|---|---|---|---|
| 12 | `role_assignments(scope_type, scope_id)` | `user_roles(city_id, restaurant_id)` | Explicit typed scope columns instead of polymorphic `scope_id`. `city_id` gets a real FK. |
| 12 | `recovery_codes` | `admin_recovery_codes` | |
| 12 / 08 | `audit_events` / `audit_log` | `audit_logs` | Columns follow 12 §5.7, **without** the hash chain (C6). |
| 08 | `event_log` / `outbox_events` | — (none) | R42: events are River job args, one job per subscriber via `InsertManyTx`; dedupe in `processed_events`. |
| 08 | `payment_intents` | `payments` | One row per PA order (intent). Attempts are in `payment_attempts`. |
| 08 | `webhook_events` | `payment_events` | Notification-provider DLR webhooks go to `notification_deliveries`. |
| 08 | `ledger_journals`, `account_balances` | `ledger_journals`, `ledger_account_balances` | |
| 08 | `settlement_runs`, `settlement_statements` | `payouts.batch_id` + `payout_items`; statements are generated documents (`report_exports`) | Fewer tables. A statement is a rendering of payout items. |
| 08 | `rider_locations`, `rider_location_samples` | `rider_availability.last_location`, `rider_location_pings` | |
| 08 | `delivery_events` | `delivery_status_history` | |
| 08 | `order_lines`, `order_address_snapshots` | `order_items`, `orders.delivery_address_snapshot` | |
| 08 | `tickets`, `ticket_actions` | `support_tickets`, `ticket_messages` (`kind='ACTION'`) | |
| 12 | `auth_settings` | `app_config` keys `auth.*` (scope CITY) | |
| 12 | `partner_applications` | `restaurants` (`DRAFT`→`SUBMITTED`) / `riders` (`APPLIED`→`UNDER_REVIEW`) | The application *is* the lifecycle row. |
| 12 | `staff_invites` | `staff_invites` | Same name (§4). |
| 14 | `payment_intents`, `webhook_events`, `account_balances` | `payments`, `payment_events`, `ledger_account_balances` | Table names per this doc; **account codes per 14 §10.2** (register row 66). |
| 14 | `transfers`, `pa_settlements`, `pa_settlement_lines`, `recon_exceptions` | same names (§7.1) | Added in v1.1 (M4). |

---

## 2. Entity-relationship diagrams (by module)

Only key columns are shown. The full DDL is in §3–§8. Dashed (non-identifying) relationships across modules are **logical** references without an FK (DB-D04).

### 2.1 Geo, identity, users

```mermaid
erDiagram
    cities ||--o{ localities : contains
    cities ||--o{ zones : contains
    users ||--o{ user_roles : "granted"
    cities |o--o{ user_roles : "admin scope"
    users ||--o{ sessions : has
    sessions ||--o{ refresh_tokens : rotates
    users ||--o| admin_credentials : "kind=STAFF"
    users ||--o{ admin_recovery_codes : "kind=STAFF"
    users ||--o{ otp_challenges : "by phone"
    users ||--o{ devices : uses
    users ||--o| customer_profiles : has
    users ||--o{ customer_addresses : saves
    localities |o..o{ customer_addresses : "area"
    users ||--o{ user_consents : gives
    cities {
        uuid id PK
        text code UK
        text timezone
        char gst_state_code
        geography centroid
        text status
    }
    zones {
        uuid id PK
        uuid city_id FK
        geometry boundary "MultiPolygon 4326"
        text status
        timestamptz paused_until
    }
    localities {
        uuid id PK
        uuid city_id FK
        text name
        jsonb name_i18n
        geography centroid
    }
    users {
        uuid id PK
        text kind "MEMBER|STAFF"
        text phone_e164 UK
        citext email UK
        text status
    }
    user_roles {
        uuid id PK
        uuid user_id FK
        text role
        uuid city_id FK
        uuid restaurant_id "ref"
        timestamptz revoked_at
    }
    customer_addresses {
        uuid id PK
        uuid user_id FK
        text house_no
        text landmark
        char pin_code
        geography location
    }
```

### 2.2 Catalog (restaurants and menus)

```mermaid
erDiagram
    restaurants ||--o{ restaurant_users : staff
    restaurants ||--o{ restaurant_devices : "heartbeat"
    restaurants ||--o{ restaurant_kyc_documents : submits
    restaurants ||--o{ restaurant_operating_hours : opens
    restaurants ||--o{ restaurant_closures : closes
    restaurants ||--o{ restaurant_cuisines : tagged
    cuisines ||--o{ restaurant_cuisines : tags
    restaurants ||--o{ menu_categories : has
    menu_categories ||--o{ menu_items : groups
    menu_items ||--o{ item_variants : "sizes"
    restaurants ||--o{ addon_groups : defines
    addon_groups ||--o{ addons : contains
    menu_items ||--o{ menu_item_addon_groups : offers
    addon_groups ||--o{ menu_item_addon_groups : "attached to"
    restaurants {
        uuid id PK
        uuid city_id FK
        uuid zone_id "ref geo"
        text status
        geography location
        int max_delivery_radius_m
        text fssai_license_no
        timestamptz paused_until
        int menu_version
    }
    menu_items {
        uuid id PK
        uuid restaurant_id FK
        uuid category_id FK
        text veg_type "VEG|NON_VEG|EGG"
        bigint price_paise
        bigint packaging_paise
        bool is_available
        jsonb name_i18n
    }
    addon_groups {
        uuid id PK
        smallint min_select
        smallint max_select
    }
```

### 2.3 Quote, pricing, promotions, ordering

```mermaid
erDiagram
    quotes |o..o| orders : "consumed by"
    orders ||--|{ order_items : contains
    orders ||--|{ order_charges : "bill lines"
    orders ||--|{ order_status_history : "transitions"
    coupons ||--o{ coupon_targets : "targets"
    coupons ||--o{ coupon_redemptions : redeemed
    orders |o..o| coupon_redemptions : "uses"
    fee_configs |o..o{ quotes : "priced with"
    tax_rules |o..o{ order_charges : "tax basis"
    orders {
        uuid id PK
        text code UK "RV-XXXXXX"
        uuid city_id FK
        uuid zone_id "ref"
        uuid customer_id FK
        uuid restaurant_id "ref"
        text status
        int version
        text payment_method
        bigint total_paise
        jsonb restaurant_snapshot
        jsonb delivery_address_snapshot
    }
    order_charges {
        uuid id PK
        uuid order_id FK
        text component
        text tax_type
        bigint amount_paise
    }
    coupons {
        uuid id PK
        citext code
        text discount_type
        text funded_by
        bigint budget_paise
    }
```

### 2.4 Payments, dispatch

```mermaid
erDiagram
    payments ||--o{ payment_attempts : "tries"
    payments ||--o{ refunds : "refunded by"
    payments |o..o{ payment_events : "webhooks"
    riders ||--|| rider_availability : "live state"
    riders ||--o{ rider_kyc_documents : submits
    riders ||--o{ rider_location_pings : reports
    riders ||--o{ sos_events : raises
    deliveries ||--o{ contact_tap_log : "call taps"
    deliveries ||--o{ delivery_offers : "offered via"
    riders ||--o{ delivery_offers : receives
    riders |o--o{ deliveries : "assigned"
    deliveries ||--|{ delivery_status_history : transitions
    payments {
        uuid id PK
        uuid order_id "ref"
        text provider
        text provider_order_id
        text status
        bigint amount_paise
    }
    deliveries {
        uuid id PK
        uuid order_id UK "ref"
        uuid rider_id FK
        text status
        int version
        timestamptz dispatch_after
    }
    delivery_offers {
        uuid id PK
        uuid delivery_id FK
        uuid rider_id FK
        text status "PENDING|ACCEPTED|DECLINED|EXPIRED|REVOKED"
        timestamptz expires_at
    }
    rider_availability {
        uuid rider_id PK
        text state "OFFLINE|AVAILABLE|ON_DELIVERY"
        geography last_location
        timestamptz last_location_at
        bigint cash_in_hand_paise "cached"
    }
```

### 2.5 Ledger, payouts, engagement, support, notifications, platform

```mermaid
erDiagram
    ledger_journals ||--|{ ledger_postings : "balanced lines"
    ledger_accounts ||--o{ ledger_postings : "posted to"
    ledger_accounts ||--|| ledger_account_balances : "running balance"
    payouts ||--|{ payout_items : contains
    ledger_postings ||--o| payout_items : "paid by"
    payout_accounts |o--o{ payouts : "paid to"
    cod_deposits |o..o| ledger_journals : "posts"
    ratings ||--o| reviews : "text"
    support_tickets ||--o{ ticket_messages : thread
    notifications ||--o{ notification_deliveries : "per channel"
    ledger_journals {
        uuid id PK
        text entry_type
        text idempotency_key UK
        uuid order_id "ref"
    }
    ledger_postings {
        uuid id PK
        uuid journal_id FK
        uuid account_id FK
        bigint amount_paise "+Dr / -Cr"
    }
    ratings {
        uuid id PK
        uuid order_id "ref"
        text target_type "RESTAURANT|RIDER"
        smallint stars
        bool thumbs_up
    }
```


### 2.6 PA settlement & reconciliation, privacy and growth (added in v1.1)

```mermaid
erDiagram
    orders ||--o{ payments : "paid by"
    payments ||--o{ transfers : "split to restaurant"
    pa_settlements ||--|{ pa_settlement_lines : contains
    pa_settlement_lines }o--o| payment_attempts : matches
    pa_settlement_lines }o--o| refunds : matches
    pa_settlement_lines }o--o| transfers : matches
    recon_exceptions }o--o| pa_settlement_lines : "raised for"
    users ||--o{ erasure_requests : requests
    cities ||--o{ waitlist : "demand outside zones"
    cities ||--o{ leads : "restaurant/rider leads"
    restaurants ||--o{ staff_invites : invites
    transfers {
        uuid id PK
        uuid order_id FK
        uuid restaurant_id "ref"
        text provider_transfer_id
        bigint amount_paise
        bool on_hold
        text status
    }
    pa_settlements {
        uuid id PK
        text provider
        text source "PA_REPORT|BANK_STATEMENT"
        date report_date
        text status
    }
    recon_exceptions {
        uuid id PK
        text exception_type
        text status
        bigint amount_paise
    }
    erasure_requests {
        uuid id PK
        uuid user_id FK
        text status
        timestamptz due_at
    }
```

---

## 3. DDL — geo, users, identity

### 3.1 geo

```sql
CREATE TABLE cities (
  id                 uuid PRIMARY KEY,
  code               text NOT NULL UNIQUE CHECK (code ~ '^[A-Z]{3,8}$'),          -- 'MBNR'
  slug               text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,40}$'),     -- 'mahabubnagar'
  name               text NOT NULL,
  name_i18n          jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(name_i18n) = 'object'),
  state_name         text NOT NULL,                                               -- 'Telangana'
  gst_state_code     char(2) NOT NULL CHECK (gst_state_code ~ '^[0-9]{2}$'),     -- '36' (place of supply)
  country_code       char(2) NOT NULL DEFAULT 'IN',
  timezone           text NOT NULL DEFAULT 'Asia/Kolkata',                        -- IANA; validated in app
  currency           char(3) NOT NULL DEFAULT 'INR',
  default_locale     text NOT NULL DEFAULT 'en',
  supported_locales  text[] NOT NULL DEFAULT '{en,te}',
  centroid           geography(Point,4326) NOT NULL,
  status             text NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED','LIVE','PAUSED','RETIRED')),
  launched_at        timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE localities (
  id               uuid PRIMARY KEY,
  city_id          uuid NOT NULL REFERENCES cities(id),
  slug             text NOT NULL,
  name             text NOT NULL,
  name_i18n        jsonb NOT NULL DEFAULT '{}',
  search_aliases   text[] NOT NULL DEFAULT '{}',          -- spelling variants: Boyapalle/Boyapally/Boyapalli
  pin_codes        text[] NOT NULL DEFAULT '{}',          -- each validated '^[1-9][0-9]{5}$' in app
  centroid         geography(Point,4326) NOT NULL,
  boundary         geometry(MultiPolygon,4326),           -- optional; V1 uses centroid only
  is_active        boolean NOT NULL DEFAULT true,
  sort_order       int NOT NULL DEFAULT 0,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_localities__city_slug UNIQUE (city_id, slug)
);
CREATE INDEX ix_localities__name_trgm ON localities USING gin (lower(name) gin_trgm_ops);
CREATE INDEX ix_localities__aliases   ON localities USING gin (search_aliases);
CREATE INDEX ix_localities__centroid  ON localities USING gist (centroid);

CREATE TABLE zones (
  id                      uuid PRIMARY KEY,
  city_id                 uuid NOT NULL REFERENCES cities(id),
  code                    text NOT NULL CHECK (code ~ '^[A-Z0-9-]{2,20}$'),      -- 'MBNR-CORE'
  name                    text NOT NULL,
  name_i18n               jsonb NOT NULL DEFAULT '{}',
  boundary                geometry(MultiPolygon,4326) NOT NULL,
  priority                int NOT NULL DEFAULT 0,               -- tie-break if a point is covered by >1 zone (16 §3)
  status                  text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','INACTIVE')),
  -- operational pause (rain, rider shortage) — orthogonal to status
  paused_until            timestamptz,
  pause_reason            text CHECK (pause_reason IN ('RAIN','RIDER_SHORTAGE','LAW_AND_ORDER','FESTIVAL','TECHNICAL','OTHER')),
  pause_message_i18n      jsonb NOT NULL DEFAULT '{}',
  paused_by               uuid REFERENCES users(id),
  -- no surge columns in V1 (R30, C1); manual zone surge is a V1.1 expand-only migration
  version                 int NOT NULL DEFAULT 1,
  created_by              uuid REFERENCES users(id),
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_zones__city_code UNIQUE (city_id, code),
  CONSTRAINT ck_zones__valid     CHECK (ST_IsValid(boundary) AND ST_SRID(boundary) = 4326)
);
CREATE INDEX ix_zones__boundary ON zones USING gist (boundary);
CREATE INDEX ix_zones__city_active ON zones (city_id) WHERE status = 'ACTIVE';

CREATE TABLE waitlist (                             -- "notify me when you deliver here" (01 CUS-ADDR-003, 04 C-04) — M4
  id                 uuid PRIMARY KEY,
  city_id            uuid REFERENCES cities(id),     -- nearest city by centroid; NULL if none within 50 km
  user_id            uuid REFERENCES users(id),      -- NULL for anonymous sign-ups
  phone_e164         text CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),   -- needed to notify anonymous sign-ups
  location           geography(Point,4326) NOT NULL, -- the unserviceable pin
  locality_id        uuid REFERENCES localities(id),
  pin_code           char(6) CHECK (pin_code ~ '^[1-9][0-9]{5}$'),
  consent_notice_version text NOT NULL,              -- DPDP notice shown at sign-up
  status             text NOT NULL DEFAULT 'WAITING' CHECK (status IN ('WAITING','NOTIFIED','UNSUBSCRIBED')),
  notified_at        timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_waitlist__contact CHECK (user_id IS NOT NULL OR phone_e164 IS NOT NULL)
);
CREATE INDEX ix_waitlist__city ON waitlist (city_id, status);
CREATE INDEX ix_waitlist__location ON waitlist USING gist (location) WHERE status = 'WAITING';   -- coverage-expansion analysis
```

Zone overlap among `ACTIVE` zones of a city is **rejected by the admin write path** (16 §8.2) and not by a constraint: `EXCLUDE ... &&` would only compare bounding boxes. Zone → fee config resolution is **by effective date** (`fee_configs.zone_id` + range, §5.4), not by a pointer on `zones`. This keeps price history immutable and lets ops schedule a fee change in advance. This is a deliberate deviation from "zones (… fee config ref)".

### 3.2 users

```sql
CREATE TABLE users (
  id                    uuid PRIMARY KEY,
  kind                  text NOT NULL DEFAULT 'MEMBER' CHECK (kind IN ('MEMBER','STAFF')),   -- 12 AUTH-D01
  phone_e164            text CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),                    -- Indian-mobile rule in app (00 §3)
  phone_verified_at     timestamptz,
  email                 citext,
  email_verified_at     timestamptz,
  full_name             text CHECK (char_length(full_name) <= 100),
  preferred_locale      text NOT NULL DEFAULT 'en' CHECK (preferred_locale ~ '^[a-z]{2}(-[A-Z]{2})?$'),
  home_city_id          uuid REFERENCES cities(id),
  status                text NOT NULL DEFAULT 'ACTIVE'
                          CHECK (status IN ('ACTIVE','BLOCKED','DELETION_PENDING','DETACHED','ANONYMIZED')),
  status_reason         text,
  adult_declared_at     timestamptz,                    -- 18+ self-declaration (riders/owners), 12 §1.1
  last_active_at        timestamptz,                    -- phone-recycling guard (12 §1.4)
  anonymized_at         timestamptz,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_users__identifier CHECK (
       status IN ('ANONYMIZED','DETACHED')
    OR (kind = 'MEMBER' AND phone_e164 IS NOT NULL)
    OR (kind = 'STAFF'  AND email IS NOT NULL))
);
-- a phone/email can belong to only one live account; DETACHED/ANONYMIZED rows release it (12 §1.4)
CREATE UNIQUE INDEX ux_users__phone ON users (phone_e164) WHERE kind = 'MEMBER' AND status NOT IN ('DETACHED','ANONYMIZED');
CREATE UNIQUE INDEX ux_users__email ON users (email)      WHERE kind = 'STAFF'  AND status <> 'ANONYMIZED';
CREATE INDEX ix_users__name_trgm ON users USING gin (lower(full_name) gin_trgm_ops);   -- admin search

CREATE TABLE customer_profiles (
  user_id                  uuid PRIMARY KEY REFERENCES users(id),
  cod_status               text NOT NULL DEFAULT 'ENABLED' CHECK (cod_status IN ('ENABLED','DISABLED')),
  cod_disabled_reason      text,
  cod_strike_count         int NOT NULL DEFAULT 0,       -- approved UNDELIVERABLE COD orders; 2 strikes → COD DISABLED (ruling 5)
  delivered_order_count    int NOT NULL DEFAULT 0,       -- new-user coupon eligibility
  first_order_at           timestamptz,
  default_address_id       uuid,                         -- same module; FK added after customer_addresses
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE customer_addresses (
  id                      uuid PRIMARY KEY,
  user_id                 uuid NOT NULL REFERENCES users(id),
  city_id                 uuid NOT NULL REFERENCES cities(id),
  locality_id             uuid,                           -- ref: geo.localities (logical)
  label                   text NOT NULL CHECK (label IN ('HOME','WORK','OTHER')),
  label_custom            text CHECK (char_length(label_custom) <= 30),
  house_no                text NOT NULL CHECK (char_length(house_no) BETWEEN 1 AND 100),     -- house / flat no.
  building_street         text CHECK (char_length(building_street) <= 150),                  -- building / street (optional, ruling 13)
  area_text               text CHECK (char_length(area_text) <= 100),                        -- free-text area if locality not listed
  landmark                text NOT NULL CHECK (char_length(landmark) BETWEEN 3 AND 150),     -- REQUIRED (ruling 13)
  city_name               text NOT NULL,
  state_name              text NOT NULL,
  pin_code                char(6) NOT NULL CHECK (pin_code ~ '^[1-9][0-9]{5}$'),
  location                geography(Point,4326) NOT NULL,                                  -- map pin (required)
  location_accuracy_m     int CHECK (location_accuracy_m >= 0),
  contact_name            text CHECK (char_length(contact_name) <= 100),                     -- "ordering for someone else"
  contact_phone_e164      text CHECK (contact_phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  delivery_instructions   text CHECK (char_length(delivery_instructions) <= 200),
  is_default              boolean NOT NULL DEFAULT false,
  last_used_at            timestamptz,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_customer_addresses__user ON customer_addresses (user_id, last_used_at DESC);
CREATE UNIQUE INDEX ux_customer_addresses__default ON customer_addresses (user_id) WHERE is_default;
ALTER TABLE customer_profiles ADD CONSTRAINT fk_customer_profiles__default_address
  FOREIGN KEY (default_address_id) REFERENCES customer_addresses(id) ON DELETE SET NULL;

CREATE TABLE user_consents (                     -- append-only consent ledger (DPDP §6); current = latest row
  id               uuid PRIMARY KEY,
  user_id          uuid NOT NULL REFERENCES users(id),
  purpose          text NOT NULL CHECK (purpose IN ('TERMS','PRIVACY_NOTICE','MARKETING_SMS','MARKETING_WHATSAPP',
                                                    'MARKETING_PUSH','PRECISE_LOCATION','PARTNER_AGREEMENT')),
  notice_version   text NOT NULL,                 -- e.g. 'privacy-2026-09-en'
  granted          boolean NOT NULL,
  locale           text NOT NULL,
  source           text NOT NULL,                 -- 'customer-web:checkout', 'partner-web:onboarding'
  ip               inet,
  user_agent       text,
  created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_user_consents__current ON user_consents (user_id, purpose, created_at DESC);

CREATE TABLE erasure_requests (                  -- DPDP erasure ledger (23 §9, M15); holds NO PII beyond the user id
  id                 uuid PRIMARY KEY,
  user_id            uuid NOT NULL REFERENCES users(id),
  source             text NOT NULL CHECK (source IN ('SELF_SERVICE','SUPPORT','GRIEVANCE_OFFICER')),
  status             text NOT NULL DEFAULT 'RECEIVED' CHECK (status IN ('RECEIVED','ON_HOLD','EXECUTING','COMPLETED','REJECTED')),
  hold_reason        text CHECK (hold_reason IN ('OPEN_ORDER','OPEN_PAYOUT','OPEN_DISPUTE','LEGAL_HOLD','CASH_IN_HAND')),
  due_at             timestamptz NOT NULL,       -- received + 30 days [LEGAL]
  executed_at        timestamptz,
  executed_by        text,                       -- 'job:erasure.execute' or admin user id
  steps_completed    jsonb NOT NULL DEFAULT '[]',-- erasure-map rows applied (§13.1), for restore replay (23 §9)
  rejection_reason   text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_erasure_requests__open ON erasure_requests (user_id) WHERE status IN ('RECEIVED','ON_HOLD','EXECUTING');
CREATE INDEX ix_erasure_requests__due ON erasure_requests (due_at) WHERE status IN ('RECEIVED','ON_HOLD');
```

`erasure_requests` is the erasure ledger that a database restore **replays** before serving traffic (23 §9). Execution (catch-up job `erasure.execute`, 13 §5.2) applies the §13.1 map; it is audited and is not maker-checker (R31).

### 3.3 identity

```sql
CREATE TABLE user_roles (                        -- = 12's role_assignments; THE authorisation source of truth
  id               uuid PRIMARY KEY,
  user_id          uuid NOT NULL REFERENCES users(id),
  role             text NOT NULL CHECK (role IN ('CUSTOMER','RESTAURANT_OWNER','RESTAURANT_STAFF','RIDER',
                                                 'ADMIN_SUPER','ADMIN_OPS','ADMIN_SUPPORT','ADMIN_FINANCE')),
  city_id          uuid REFERENCES cities(id),   -- ADMIN_*: NULL = all cities; RIDER: rider's city
  restaurant_id    uuid,                          -- ref: catalog.restaurants (logical) — RESTAURANT_* only
  granted_by       uuid REFERENCES users(id),
  approval_id      uuid,                          -- ref: admin.approval_requests (maker-checker for ADMIN_*)
  granted_at       timestamptz NOT NULL DEFAULT now(),
  revoked_at       timestamptz,
  revoked_by       uuid REFERENCES users(id),
  revoke_reason    text,
  CONSTRAINT ck_user_roles__scope CHECK (
       (role IN ('RESTAURANT_OWNER','RESTAURANT_STAFF') AND restaurant_id IS NOT NULL)
    OR (role LIKE 'ADMIN\_%' AND restaurant_id IS NULL)
    OR (role = 'RIDER'    AND restaurant_id IS NULL AND city_id IS NOT NULL)
    OR (role = 'CUSTOMER' AND restaurant_id IS NULL AND city_id IS NULL))
);
CREATE UNIQUE INDEX ux_user_roles__active ON user_roles (user_id, role, city_id, restaurant_id)
  NULLS NOT DISTINCT WHERE revoked_at IS NULL;
CREATE INDEX ix_user_roles__restaurant ON user_roles (restaurant_id) WHERE revoked_at IS NULL AND restaurant_id IS NOT NULL;
-- Trigger trg_user_roles_exclusive (BEFORE INSERT): rejects RIDER + RESTAURANT_* on the same user (12 AUTH-D02)
-- and ADMIN_* on kind=MEMBER / member roles on kind=STAFF (12 AUTH-D01).

CREATE TABLE otp_challenges (
  id                    uuid PRIMARY KEY,
  phone_e164            text NOT NULL,
  purpose               text NOT NULL CHECK (purpose IN ('LOGIN','PHONE_CHANGE_OLD','PHONE_CHANGE_NEW','STEP_UP','PARTNER_AGREEMENT')),
  audience              text NOT NULL CHECK (audience IN ('customer','restaurant','rider')),   -- R14 hosts
  channel               text NOT NULL CHECK (channel IN ('SMS')),   -- WhatsApp/voice OTP are V1.1 (C2): widen the CHECK then
  code_hmac             bytea NOT NULL,           -- HMAC-SHA-256(pepper, id || code) (12 AUTH-D07)
  attempts              smallint NOT NULL DEFAULT 0,
  max_attempts          smallint NOT NULL DEFAULT 5,
  resend_count          smallint NOT NULL DEFAULT 0,
  expires_at            timestamptz NOT NULL,
  verified_at           timestamptz,
  invalidated_at        timestamptz,
  provider              text,
  provider_message_id   text,
  request_ip            inet,
  device_id             uuid,
  created_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_otp__attempts CHECK (attempts <= max_attempts)
);
CREATE INDEX ix_otp_challenges__phone ON otp_challenges (phone_e164, created_at DESC);   -- per-phone rate limits
CREATE INDEX ix_otp_challenges__ip    ON otp_challenges (request_ip, created_at DESC);   -- per-IP rate limits

CREATE TABLE devices (
  id               uuid PRIMARY KEY,
  install_id       text NOT NULL UNIQUE,       -- random id from the client (rovo_did cookie / native install id)
  user_id          uuid REFERENCES users(id),  -- last signed-in user
  platform         text NOT NULL CHECK (platform IN ('WEB_ANDROID','WEB_IOS','WEB_DESKTOP','ANDROID','IOS')),
  app              text NOT NULL CHECK (app IN ('customer','restaurant','rider','admin')),   -- four apps/hosts (R14)
  app_version      text,
  user_agent       text,
  first_seen_at    timestamptz NOT NULL DEFAULT now(),
  last_seen_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_devices__user ON devices (user_id);

CREATE TABLE sessions (                            -- one row per login = refresh-token family (12 §4.3)
  id                    uuid PRIMARY KEY,           -- = JWT 'sid'
  user_id               uuid NOT NULL REFERENCES users(id),
  audience              text NOT NULL CHECK (audience IN ('customer','restaurant','rider','admin',
                                                     'customer_native','restaurant_native','rider_native')),   -- = host (R14)
  active_context        text,                       -- 'restaurant:<uuid>' (restaurant audience, multi-outlet owners)
  device_id             uuid REFERENCES devices(id),
  binding               text NOT NULL DEFAULT 'STANDARD' CHECK (binding IN ('STANDARD','DEVICE_BOUND')),
                                                    -- DEVICE_BOUND: registered order-receiver device (R44, M10)
  idle_timeout_s        int NOT NULL,               -- sliding window; R44: restaurant device 30 d, rider 30 d; others per 12
  absolute_timeout_s    int NOT NULL,               -- R44: restaurant device 90 d
  device_label          text,
  amr                   text[] NOT NULL,            -- {'otp'} | {'pwd','totp'}
  step_up_at            timestamptz,
  created_ip            inet,
  last_ip               inet,
  last_seen_at          timestamptz NOT NULL DEFAULT now(),
  idle_expires_at       timestamptz NOT NULL,
  absolute_expires_at   timestamptz NOT NULL,
  revoked_at            timestamptz,
  revoke_reason         text CHECK (revoke_reason IN ('LOGOUT','LOGOUT_OTHERS','REFRESH_REUSE','ADMIN_REVOKED',
                                                      'USER_BLOCKED','PHONE_CHANGED','PASSWORD_CHANGED','GLOBAL_NOT_BEFORE',
                                                      'DEVICE_REVOKED')),          -- owner/admin revoked a bound device (R44)
  created_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_sessions__device_bound CHECK (binding = 'STANDARD' OR (device_id IS NOT NULL AND audience = 'restaurant'))
);
CREATE INDEX ix_sessions__user_live ON sessions (user_id) WHERE revoked_at IS NULL;

CREATE TABLE refresh_tokens (
  id              uuid PRIMARY KEY,
  session_id      uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  token_sha256    bytea NOT NULL UNIQUE,
  parent_id       uuid REFERENCES refresh_tokens(id),
  issued_at       timestamptz NOT NULL DEFAULT now(),
  used_at         timestamptz,                  -- set on rotation; reuse >15 s later ⇒ revoke family
  expires_at      timestamptz NOT NULL
);
CREATE INDEX ix_refresh_tokens__session ON refresh_tokens (session_id);

CREATE TABLE admin_credentials (
  user_id               uuid PRIMARY KEY REFERENCES users(id),   -- kind = STAFF only (trigger)
  password_hash         text NOT NULL,                           -- argon2id PHC string (12 AUTH-D08)
  totp_secret_enc       bytea,                                   -- AES-256-GCM, DEK wrapped by KMS KEK
  pii_key_id            text,                                    -- KEK version used
  totp_enrolled_at      timestamptz,
  totp_last_step        bigint,                                  -- replay guard
  failed_attempts       int NOT NULL DEFAULT 0,
  locked_until          timestamptz,
  must_change_password  boolean NOT NULL DEFAULT true,
  password_changed_at   timestamptz NOT NULL DEFAULT now(),
  setup_token_sha256    bytea UNIQUE,                            -- one-time setup link (12 §3.3)
  setup_token_expires_at timestamptz,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_admin_credentials__totp CHECK ((totp_secret_enc IS NULL) = (pii_key_id IS NULL))
);

CREATE TABLE admin_recovery_codes (
  id            uuid PRIMARY KEY,
  user_id       uuid NOT NULL REFERENCES users(id),
  code_hash     text NOT NULL,                   -- argon2id
  used_at       timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_admin_recovery_codes__user ON admin_recovery_codes (user_id) WHERE used_at IS NULL;

CREATE TABLE phone_change_requests (
  id                 uuid PRIMARY KEY,
  user_id            uuid NOT NULL REFERENCES users(id),
  old_phone_e164     text NOT NULL,
  new_phone_e164     text NOT NULL,
  old_otp_id         uuid REFERENCES otp_challenges(id),
  new_otp_id         uuid REFERENCES otp_challenges(id),
  status             text NOT NULL CHECK (status IN ('PENDING','COMPLETED','EXPIRED','CANCELLED','REVIEW')),
  expires_at         timestamptz NOT NULL,
  completed_at       timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_phone_change__pending ON phone_change_requests (user_id) WHERE status = 'PENDING';

CREATE TABLE rate_limit_buckets (                -- R21: Postgres-backed limits shared across replicas (12 §2.2)
  bucket_key      text NOT NULL,                 -- 'otp:phone:<hmac>', 'otp:ip:<ip/24>', 'sms:global', 'api:user:<id>'
  window_start    timestamptz NOT NULL,          -- fixed window start (app clock); sliding = weighted sum of 2 windows
  window_s        int NOT NULL CHECK (window_s > 0),
  hits            int NOT NULL DEFAULT 0,
  expires_at      timestamptz NOT NULL,          -- window_start + 2 × window_s; swept by retention.sweep
  PRIMARY KEY (bucket_key, window_start)
) WITH (fillfactor = 70);
CREATE INDEX ix_rate_limit_buckets__expiry ON rate_limit_buckets (expires_at);
-- INSERT … ON CONFLICT (bucket_key, window_start) DO UPDATE SET hits = rate_limit_buckets.hits + 1 RETURNING hits;
-- Keys never contain raw phone numbers or IPs (HMAC). Moves to the Redis adapter only if/when P6 enables it.
```

Device-bound sessions (R44, M10): a restaurant owner (or ops) **registers** a device as an order receiver. Its session gets `binding = 'DEVICE_BOUND'`, `idle_timeout_s = 30 d` (sliding) and `absolute_timeout_s = 90 d`. The refresh token stays bound to `devices.install_id`, and owners/admins can revoke it (`DEVICE_REVOKED`). Re-authentication is scheduled outside service hours. Rider sessions use a 30-day sliding idle timeout. All other timeouts are owned by 12.

---

## 4. DDL — catalog (restaurants, menus)

```sql
CREATE TABLE restaurants (                       -- one row per physical outlet (00 §2)
  id                          uuid PRIMARY KEY,
  city_id                     uuid NOT NULL REFERENCES cities(id),
  zone_id                     uuid,              -- ref: geo.zones — zone covering `location`, set on approval
  locality_id                 uuid,              -- ref: geo.localities
  slug                        text NOT NULL CHECK (slug ~ '^[a-z0-9-]{3,80}$'),
  name                        text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 80),
  name_i18n                   jsonb NOT NULL DEFAULT '{}',
  description                 text CHECK (char_length(description) <= 500),
  description_i18n            jsonb NOT NULL DEFAULT '{}',
  legal_name                  text,              -- for tax invoices
  status                      text NOT NULL DEFAULT 'DRAFT' CHECK (status IN
                                ('DRAFT','SUBMITTED','CHANGES_REQUESTED','REJECTED','ACTIVE','SUSPENDED','OFFBOARDED')),
  status_reason               text,
  diet_type                   text NOT NULL DEFAULT 'MIXED' CHECK (diet_type IN ('PURE_VEG','MIXED')),   -- "Pure veg" badge
  phone_e164                  text NOT NULL CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  email                       citext,
  address_line                text NOT NULL,
  landmark                    text,
  pin_code                    char(6) NOT NULL CHECK (pin_code ~ '^[1-9][0-9]{5}$'),
  location                    geography(Point,4326) NOT NULL,
  max_delivery_radius_m       int NOT NULL DEFAULT 7000 CHECK (max_delivery_radius_m BETWEEN 500 AND 15000),
                                                 -- STRAIGHT-LINE metres (R18); effective = min(this, fee_configs.max_serviceable_radius_m)
  avg_prep_time_min           smallint NOT NULL DEFAULT 20 CHECK (avg_prep_time_min BETWEEN 5 AND 90),
  min_order_paise             bigint NOT NULL DEFAULT 0 CHECK (min_order_paise >= 0),
  cost_for_two_paise          bigint CHECK (cost_for_two_paise >= 0),
  packaging_mode              text NOT NULL DEFAULT 'PER_ITEM' CHECK (packaging_mode IN ('PER_ITEM','PER_ORDER','NONE')),
  packaging_per_order_paise   bigint NOT NULL DEFAULT 0 CHECK (packaging_per_order_paise BETWEEN 0 AND 10000),
  fssai_license_no            text CHECK (fssai_license_no ~ '^[0-9]{14}$'),     -- 14-digit FSSAI number [LEGAL]
  fssai_valid_until           date,
  gstin                       text CHECK (gstin ~ '^[0-9]{2}[A-Z0-9]{10}[0-9A-Z]Z[0-9A-Z]$'),  -- nullable: unregistered OK under §9(5)
  pan_enc                     bytea,
  pan_last4                   text,
  pii_key_id                  text,
  accepting_orders            boolean NOT NULL DEFAULT false,   -- partner "Open / Closed" master switch
  paused_until                timestamptz,                      -- "Busy" pause (05 §4.2); NULL = not paused; 'infinity' = until owner resumes
  pause_reason                text CHECK (pause_reason IN ('TOO_MANY_ORDERS','STAFF_SHORTAGE','KITCHEN_ISSUE',
                                                           'AUTO_MISSED_ORDERS','DEVICE_OFFLINE','ADMIN','OTHER')),
  paused_by_user_id           uuid REFERENCES users(id),
  consecutive_missed_orders   smallint NOT NULL DEFAULT 0,      -- reset on accept; 1 → pause 30 min, 2 → pause until resumed (13 §5)
  menu_version                int NOT NULL DEFAULT 1,           -- bumped on any menu change → ETag / cache key (08 §8)
  logo_file_id                uuid,                             -- ref: platform.file_objects
  cover_file_id               uuid,
  approved_at                 timestamptz,
  approved_by                 uuid REFERENCES users(id),
  version                     int NOT NULL DEFAULT 1,
  created_at                  timestamptz NOT NULL DEFAULT now(),
  updated_at                  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_restaurants__city_slug UNIQUE (city_id, slug),
  CONSTRAINT ck_restaurants__live_requirements CHECK (
    status <> 'ACTIVE' OR (fssai_license_no IS NOT NULL AND fssai_valid_until IS NOT NULL AND zone_id IS NOT NULL))
);
CREATE INDEX ix_restaurants__live_location ON restaurants USING gist (location) WHERE status = 'ACTIVE';
CREATE INDEX ix_restaurants__city_status   ON restaurants (city_id, status);
CREATE INDEX ix_restaurants__name_trgm     ON restaurants USING gin (lower(name) gin_trgm_ops);
CREATE INDEX ix_restaurants__fssai_expiry  ON restaurants (fssai_valid_until) WHERE status = 'ACTIVE';

CREATE TABLE restaurant_users (                -- staff membership (activated from an accepted staff_invite)
  id                   uuid PRIMARY KEY,
  restaurant_id        uuid NOT NULL REFERENCES restaurants(id),
  user_id              uuid NOT NULL REFERENCES users(id),
  role                 text NOT NULL CHECK (role IN ('RESTAURANT_OWNER','RESTAURANT_STAFF')),
  permissions          text[] NOT NULL DEFAULT '{}',        -- staff narrowing, e.g. {MENU_EDIT} (12 §5.2)
  display_name         text,
  status               text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','REMOVED')),
  staff_invite_id      uuid,                                -- the invite it came from (NULL for the owner)
  user_role_id         uuid,                                -- ref: identity.user_roles row created on activation
  removed_at           timestamptz,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_restaurant_users__member ON restaurant_users (restaurant_id, user_id) WHERE status = 'ACTIVE';

CREATE TABLE staff_invites (                    -- owner invites staff by phone (12 §2.3) — M4
  id                   uuid PRIMARY KEY,
  restaurant_id        uuid NOT NULL REFERENCES restaurants(id),
  phone_hmac           bytea NOT NULL,                      -- HMAC(pepper, E.164); the phone itself is not stored
  phone_last4          text NOT NULL,
  display_name         text,
  permissions          text[] NOT NULL DEFAULT '{}',
  token_sha256         bytea NOT NULL UNIQUE,               -- single-use invite code sent by SMS
  status               text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','ACCEPTED','REVOKED','EXPIRED')),
  expires_at           timestamptz NOT NULL,                -- created + 72 h (12 §2.3)
  invited_by           uuid NOT NULL REFERENCES users(id),
  accepted_by          uuid REFERENCES users(id),           -- must have the invited phone (HMAC match)
  accepted_at          timestamptz,
  created_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_staff_invites__pending ON staff_invites (restaurant_id, phone_hmac) WHERE status = 'PENDING';
ALTER TABLE restaurant_users ADD CONSTRAINT fk_restaurant_users__invite FOREIGN KEY (staff_invite_id) REFERENCES staff_invites(id);

CREATE TABLE leads (                            -- restaurant "partner with us" form (05 §2) and rider interest — M4
  id                   uuid PRIMARY KEY,
  lead_type            text NOT NULL CHECK (lead_type IN ('RESTAURANT','RIDER')),
  city_id              uuid REFERENCES cities(id),
  locality_id          uuid,                                -- ref: geo.localities
  business_name        text CHECK (char_length(business_name) <= 80),
  contact_name         text NOT NULL CHECK (char_length(contact_name) <= 100),
  phone_e164           text NOT NULL CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),   -- OTP-verified (05 §2)
  cuisine_codes        text[] NOT NULL DEFAULT '{}',
  fssai_state          text CHECK (fssai_state IN ('YES','NO','APPLIED')),
  preferred_call_time  text,
  status               text NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW','CONTACTED','CONVERTED','DISQUALIFIED')),
  assigned_admin_id    uuid REFERENCES users(id),
  converted_restaurant_id uuid,                             -- ref when converted
  converted_rider_id   uuid,
  notes                text,
  consent_notice_version text NOT NULL,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_leads__queue ON leads (city_id, lead_type, status, created_at);
```

```sql
CREATE TABLE restaurant_devices (              -- counter device liveness (ruling 1/11; 05 §8)
  id                  uuid PRIMARY KEY,
  restaurant_id       uuid NOT NULL REFERENCES restaurants(id),
  device_id           uuid NOT NULL,                       -- ref: identity.devices
  user_id             uuid REFERENCES users(id),           -- staff signed in on it
  label               text,                                -- 'Counter tablet'
  is_order_receiver   boolean NOT NULL DEFAULT true,       -- rings for new orders
  session_id          uuid,                                -- ref: identity.sessions (DEVICE_BOUND, R44) while registered
  registered_by       uuid REFERENCES users(id),           -- owner or ops who bound it
  revoked_at          timestamptz,                         -- owner/admin revoked the binding
  last_heartbeat_at   timestamptz NOT NULL DEFAULT now(),  -- heartbeat every 60 s OR SSE presence (R27; 13 §5.1)
  last_sse_connected_at timestamptz,
  push_ok             boolean NOT NULL DEFAULT false,      -- has a live push subscription
  app_version         text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_restaurant_devices__device UNIQUE (restaurant_id, device_id)
) WITH (fillfactor = 70);                                  -- hot heartbeat updates (RV-042)
CREATE INDEX ix_restaurant_devices__liveness ON restaurant_devices (restaurant_id, last_heartbeat_at DESC) WHERE is_order_receiver;
```

Liveness rule (River periodic job every 30 s; T-DEVICE-HB in 13 §5): an outlet that is **open** (accepting orders, within hours, not paused) and has **no order-receiver heartbeat or SSE presence for `restaurant.device_offline_pause_s`** (3 min) is auto-paused with `pause_reason='DEVICE_OFFLINE'` and `paused_until='infinity'`. The owner is sent an SMS. It resumes automatically on the next heartbeat plus an explicit "Resume" tap (05 §8).

**Rule:** `user_roles` is the only table read for authorisation. `staff_invites` holds the invitation; `restaurant_users` holds the staff profile. Activating or removing a member writes both rows in one transaction (catalog calls `identity.Grant/Revoke`, a tx-participating method added to 08 §4.1's list).

```sql
CREATE TABLE restaurant_kyc_documents (
  id                 uuid PRIMARY KEY,
  restaurant_id      uuid NOT NULL REFERENCES restaurants(id),
  doc_type           text NOT NULL CHECK (doc_type IN ('FSSAI_LICENSE','GST_CERTIFICATE','PAN_CARD','BANK_PROOF',
                                                       'SHOP_ESTABLISHMENT','OWNER_ID_PROOF','TRADE_LICENSE','MENU_CARD','OUTLET_PHOTO')),
  file_id            uuid NOT NULL,                     -- ref: platform.file_objects (bucket 'kyc', app-layer encrypted)
  doc_number_enc     bytea,                             -- PAN etc.; FSSAI/GSTIN are public numbers kept on restaurants
  doc_number_last4   text,
  pii_key_id         text,
  valid_until        date,
  status             text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED','SUPERSEDED')),
  reviewed_by        uuid REFERENCES users(id),
  reviewed_at        timestamptz,
  rejection_reason   text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_restaurant_kyc__current ON restaurant_kyc_documents (restaurant_id, doc_type)
  WHERE status IN ('PENDING','APPROVED') AND doc_type <> 'OUTLET_PHOTO';
CREATE INDEX ix_restaurant_kyc__expiry ON restaurant_kyc_documents (valid_until) WHERE status = 'APPROVED';

CREATE TABLE restaurant_operating_hours (
  id              uuid PRIMARY KEY,
  restaurant_id   uuid NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
  day_of_week     smallint NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),   -- ISO 8601: 1 = Monday
  opens_at        time NOT NULL,                                          -- local wall clock (cities.timezone)
  closes_at       time NOT NULL,                                          -- closes_at < opens_at ⇒ crosses midnight
  created_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_hours__nonzero CHECK (opens_at <> closes_at)
);
CREATE INDEX ix_restaurant_hours__restaurant ON restaurant_operating_hours (restaurant_id, day_of_week);
-- The whole week is replaced atomically via PUT (11); slot overlap is validated in the app.

CREATE TABLE restaurant_closures (               -- holidays / one-off closures
  id              uuid PRIMARY KEY,
  restaurant_id   uuid NOT NULL REFERENCES restaurants(id),
  starts_at       timestamptz NOT NULL,
  ends_at         timestamptz NOT NULL,
  reason          text NOT NULL CHECK (reason IN ('HOLIDAY','FESTIVAL','RENOVATION','LICENCE_ISSUE','OTHER')),
  note            text,
  created_by      uuid REFERENCES users(id),
  created_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_closures__range CHECK (ends_at > starts_at),
  CONSTRAINT ex_closures__overlap EXCLUDE USING gist (restaurant_id WITH =, tstzrange(starts_at, ends_at) WITH &&)
);

CREATE TABLE cuisines (
  id          uuid PRIMARY KEY,
  code        text NOT NULL UNIQUE,              -- 'BIRYANI','SOUTH_INDIAN','TIFFINS','INDO_CHINESE'
  name        text NOT NULL,
  name_i18n   jsonb NOT NULL DEFAULT '{}',
  sort_order  int NOT NULL DEFAULT 0,
  is_active   boolean NOT NULL DEFAULT true
);
CREATE TABLE restaurant_cuisines (
  restaurant_id  uuid NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
  cuisine_id     uuid NOT NULL REFERENCES cuisines(id),
  is_primary     boolean NOT NULL DEFAULT false,
  PRIMARY KEY (restaurant_id, cuisine_id)
);
CREATE INDEX ix_restaurant_cuisines__cuisine ON restaurant_cuisines (cuisine_id);

CREATE TABLE menu_categories (
  id              uuid PRIMARY KEY,
  restaurant_id   uuid NOT NULL REFERENCES restaurants(id),
  name            text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
  name_i18n       jsonb NOT NULL DEFAULT '{}',
  sort_order      int NOT NULL DEFAULT 0,
  is_active       boolean NOT NULL DEFAULT true,
  archived_at     timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_menu_categories__id_restaurant UNIQUE (id, restaurant_id)     -- composite-FK target
);
CREATE UNIQUE INDEX ux_menu_categories__name ON menu_categories (restaurant_id, lower(name)) WHERE archived_at IS NULL;

CREATE TABLE menu_items (
  id                  uuid PRIMARY KEY,
  restaurant_id       uuid NOT NULL REFERENCES restaurants(id),
  category_id         uuid NOT NULL,
  name                text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
  name_i18n           jsonb NOT NULL DEFAULT '{}',
  description         text CHECK (char_length(description) <= 300),
  description_i18n    jsonb NOT NULL DEFAULT '{}',
  veg_type            text NOT NULL CHECK (veg_type IN ('VEG','NON_VEG','EGG')),
  price_paise         bigint NOT NULL CHECK (price_paise BETWEEN 0 AND 10000000),   -- base / "from" price
  packaging_paise     bigint NOT NULL DEFAULT 0 CHECK (packaging_paise BETWEEN 0 AND 10000),
  has_variants        boolean NOT NULL DEFAULT false,
  image_file_id       uuid,                          -- ref: platform.file_objects
  is_available        boolean NOT NULL DEFAULT true, -- "Out of stock" toggle
  unavailable_until   timestamptz,                   -- auto back-in-stock (05 §4.3)
  is_recommended      boolean NOT NULL DEFAULT false,
  spice_level         smallint CHECK (spice_level BETWEEN 0 AND 3),
  serves              smallint CHECK (serves BETWEEN 1 AND 20),
  tax_category        text NOT NULL DEFAULT 'RESTAURANT_SERVICE',   -- joins tax_rules.tax_category
  sort_order          int NOT NULL DEFAULT 0,
  archived_at         timestamptz,
  version             int NOT NULL DEFAULT 1,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_menu_items__id_restaurant UNIQUE (id, restaurant_id),
  CONSTRAINT fk_menu_items__category FOREIGN KEY (category_id, restaurant_id)
    REFERENCES menu_categories (id, restaurant_id)                              -- item & category same restaurant
);
CREATE INDEX ix_menu_items__menu ON menu_items (restaurant_id, category_id, sort_order) WHERE archived_at IS NULL;
CREATE INDEX ix_menu_items__name_trgm ON menu_items USING gin (lower(name) gin_trgm_ops) WHERE archived_at IS NULL;
CREATE INDEX ix_menu_items__name_te_trgm ON menu_items USING gin ((name_i18n->>'te') gin_trgm_ops) WHERE archived_at IS NULL;

CREATE TABLE item_variants (                      -- single dimension (size/portion): Half / Full
  id               uuid PRIMARY KEY,
  menu_item_id     uuid NOT NULL REFERENCES menu_items(id),
  name             text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 40),
  name_i18n        jsonb NOT NULL DEFAULT '{}',
  price_paise      bigint NOT NULL CHECK (price_paise BETWEEN 0 AND 10000000),   -- ABSOLUTE price, not a delta
  packaging_paise  bigint CHECK (packaging_paise BETWEEN 0 AND 10000),           -- NULL = inherit item
  is_default       boolean NOT NULL DEFAULT false,
  is_available     boolean NOT NULL DEFAULT true,
  sort_order       int NOT NULL DEFAULT 0,
  archived_at      timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_item_variants__default ON item_variants (menu_item_id) WHERE is_default AND archived_at IS NULL;
CREATE INDEX ix_item_variants__item ON item_variants (menu_item_id) WHERE archived_at IS NULL;

CREATE TABLE addon_groups (                        -- reusable per restaurant ("Choose your bread", "Extras")
  id             uuid PRIMARY KEY,
  restaurant_id  uuid NOT NULL REFERENCES restaurants(id),
  name           text NOT NULL,
  name_i18n      jsonb NOT NULL DEFAULT '{}',
  min_select     smallint NOT NULL DEFAULT 0,
  max_select     smallint NOT NULL DEFAULT 1,
  archived_at    timestamptz,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_addon_groups__id_restaurant UNIQUE (id, restaurant_id),
  CONSTRAINT ck_addon_groups__minmax CHECK (min_select >= 0 AND max_select >= 1 AND min_select <= max_select AND max_select <= 20)
);

CREATE TABLE addons (
  id              uuid PRIMARY KEY,
  addon_group_id  uuid NOT NULL REFERENCES addon_groups(id),
  name            text NOT NULL,
  name_i18n       jsonb NOT NULL DEFAULT '{}',
  price_paise     bigint NOT NULL DEFAULT 0 CHECK (price_paise BETWEEN 0 AND 1000000),
  veg_type        text NOT NULL CHECK (veg_type IN ('VEG','NON_VEG','EGG')),
  is_available    boolean NOT NULL DEFAULT true,
  sort_order      int NOT NULL DEFAULT 0,
  archived_at     timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_addons__group ON addons (addon_group_id) WHERE archived_at IS NULL;

CREATE TABLE menu_item_addon_groups (
  menu_item_id    uuid NOT NULL,
  addon_group_id  uuid NOT NULL,
  restaurant_id   uuid NOT NULL,
  sort_order      int NOT NULL DEFAULT 0,
  PRIMARY KEY (menu_item_id, addon_group_id),
  FOREIGN KEY (menu_item_id, restaurant_id)   REFERENCES menu_items (id, restaurant_id) ON DELETE CASCADE,
  FOREIGN KEY (addon_group_id, restaurant_id) REFERENCES addon_groups (id, restaurant_id)  -- no cross-restaurant groups
);
```

Menu design notes:
- A `NON_VEG` addon on a `VEG` item is allowed, but the cart line becomes `NON_VEG` for display. The order snapshot stores the **effective** veg type.
- `PURE_VEG` restaurants: an app rule rejects `NON_VEG`/`EGG` items and addons. This is not a DB constraint because it would be cross-row.
- Any catalog write in a restaurant's menu bumps `restaurants.menu_version` in the same tx.

---

## 5. DDL — quote & pricing, promotions

### 5.1 Cart decision: device cart + persisted quote (Lead ruling 12)

**Decision: no server-side cart in V1.** The cart lives on the device in both the guest and logged-in state (17). The client sends the full cart to `POST /api/v1/cart/quote` (11 §2.3, §3). The server prices it and persists the result in `quotes`. It returns a **signed `quoteId`**: `q1.<uuid>.<HMAC-SHA256(server key, uuid ‖ user_id ‖ expires_at)>`, truncated to 128 bits and base64url-encoded. The signature lets the API reject forged, foreign or expired IDs before any DB read. The persisted row lets `POST /orders` use **exactly** the lines and amounts the customer saw.

| Option | Pro | Con | Verdict |
|---|---|---|---|
| Server cart (`carts`, `cart_items`) | Cross-device, abandoned-cart analytics | Extra aggregate and writes. The frontend has already chosen a device cart (17). | Deferred (expand-only later) |
| Fully stateless signed quote (no table) | No writes | Order create must re-send the cart and re-price, so a price drift between quote and order needs a diff anyway. A large token. | Rejected |
| **Device cart + persisted quote with signed id** | Price lock, small payloads, 409-with-diff on staleness | One insert per quote (TTL-swept) | **Chosen** |

```sql
CREATE TABLE quotes (                               -- persisted price lock (08 §5.1), TTL 10 min, swept after 24 h
  id                     uuid PRIMARY KEY,
  user_id                uuid NOT NULL REFERENCES users(id),     -- guests get a non-persisted preview quote
  city_id                uuid NOT NULL REFERENCES cities(id),
  restaurant_id          uuid NOT NULL,              -- ref: catalog.restaurants
  menu_version           int NOT NULL,               -- restaurants.menu_version at pricing time
  zone_id                uuid NOT NULL,              -- ref: customer's (drop) zone
  address_id             uuid NOT NULL,              -- ref: users.customer_addresses
  payment_method         text NOT NULL CHECK (payment_method IN ('ONLINE','COD')),
  coupon_code            citext,
  fee_config_id          uuid NOT NULL REFERENCES fee_configs(id),
  input_lines            jsonb NOT NULL,             -- the cart as sent: [{menuItemId, variantId, addonIds[], quantity, note}]
  input_hash             bytea NOT NULL,             -- sha256(canonical input) — lets the client detect "same cart"
  priced_lines           jsonb NOT NULL,             -- same shape as order_items snapshot
  charges                jsonb NOT NULL,             -- same shape as order_charges
  straight_distance_m    int NOT NULL,
  est_road_distance_m    int NOT NULL,
  eta_min_minutes        smallint NOT NULL,
  eta_max_minutes        smallint NOT NULL,
  total_paise            bigint NOT NULL CHECK (total_paise >= 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  issues                 jsonb NOT NULL DEFAULT '[]',  -- [{code:'ITEM_UNAVAILABLE', menuItemId…}] — blocking + advisory
  is_orderable           boolean NOT NULL,
  expires_at             timestamptz NOT NULL,
  consumed_at            timestamptz,
  order_id               uuid,                       -- ref: ordering.orders once consumed
  created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_quotes__user ON quotes (user_id, created_at DESC);
CREATE INDEX ix_quotes__sweep ON quotes (expires_at) WHERE consumed_at IS NULL;
CREATE UNIQUE INDEX ux_quotes__order ON quotes (order_id) WHERE order_id IS NOT NULL;
```

**Staleness at order creation** (13 O-01, 11 §3): the quote is re-validated. Checks: restaurant open and serviceable, items available, `menu_version` unchanged (or, if changed, the re-priced total is equal), coupon still reservable, COD eligibility. Any difference returns `409 QUOTE_CHANGED`, and the problem body carries a **new quote and a line-level diff**. An expired quote returns `409 QUOTE_EXPIRED` with a fresh quote.

### 5.2 Fee configs (versioned per city / zone)

```sql
CREATE TABLE fee_configs (
  id                               uuid PRIMARY KEY,
  city_id                          uuid NOT NULL REFERENCES cities(id),
  zone_id                          uuid,              -- ref: geo.zones; NULL = city default
  version                          int NOT NULL,
  effective_from                   timestamptz NOT NULL,
  effective_to                     timestamptz,       -- NULL = open-ended
  -- customer fees (00 §5 defaults)
  delivery_fee_slabs               jsonb NOT NULL,    -- ROAD metres, [fromM, toM): [{"fromM":0,"toM":2000,"feePaise":2000},
                                                      --  {"fromM":2000,"toM":4000,"feePaise":3000},{"fromM":4000,"toM":6000,"feePaise":4000},
                                                      --  {"fromM":6000,"toM":8000,"feePaise":5000},{"fromM":8000,"toM":10000,"feePaise":6000}] (R18)
  free_delivery_min_order_paise    bigint,            -- NULL = no free-delivery threshold
  platform_fee_paise               bigint NOT NULL DEFAULT 500,
  small_cart_threshold_paise       bigint NOT NULL DEFAULT 14900,
  small_cart_fee_paise             bigint NOT NULL DEFAULT 1500,
  road_factor_milli                int NOT NULL DEFAULT 1300 CHECK (road_factor_milli BETWEEN 1000 AND 3000),
  max_serviceable_radius_m         int NOT NULL DEFAULT 7000 CHECK (max_serviceable_radius_m BETWEEN 500 AND 15000),
                                                      -- STRAIGHT-LINE cap on top of restaurant radius (R18)
  cod_enabled                      boolean NOT NULL DEFAULT true,
  cod_max_order_paise              bigint NOT NULL DEFAULT 100000,   -- ₹1,000 per order (ruling 6)
  cod_first_order_max_paise        bigint NOT NULL DEFAULT 60000,    -- ₹600 on a customer's first order (ruling 6)
  -- bill presentation (ruling 8)
  round_payable_to_rupee           boolean NOT NULL DEFAULT true,    -- adds a ROUND_OFF bill line
  -- rider pay (00 §5 defaults)
  rider_base_pay_paise             bigint NOT NULL DEFAULT 2500,
  rider_base_distance_m            int NOT NULL DEFAULT 2000,
  rider_per_km_paise               bigint NOT NULL DEFAULT 600,
  rider_wait_free_s                int NOT NULL DEFAULT 600,
  rider_wait_per_min_paise         bigint NOT NULL DEFAULT 100,      -- [ASSUMPTION] ₹1/min after 10 min
  rider_cash_limit_paise           bigint NOT NULL DEFAULT 200000,
  rider_cancel_comp_bps            int NOT NULL DEFAULT 5000,        -- % of base pay if cancelled after rider reached restaurant [ASSUMPTION]
  -- ETA model (16 §5)
  eta_params                       jsonb NOT NULL,    -- {"pickupBufferMin":5,"handoverMin":2,"speedKmphByBand":[...]}
  status                           text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','APPROVED','RETIRED')),
  approval_id                      uuid,              -- maker-checker (R31 family 3: fee-config change)
  created_by                       uuid REFERENCES users(id),
  created_at                       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_fee_configs__version UNIQUE NULLS NOT DISTINCT (city_id, zone_id, version),
  CONSTRAINT ck_fee_configs__range CHECK (effective_to IS NULL OR effective_to > effective_from),
  CONSTRAINT ck_fee_configs__slabs CHECK (jsonb_typeof(delivery_fee_slabs) = 'array'
                                          AND jsonb_array_length(delivery_fee_slabs) BETWEEN 1 AND 20),
  CONSTRAINT ex_fee_configs__no_overlap EXCLUDE USING gist (
      city_id WITH =,
      (coalesce(zone_id, '00000000-0000-0000-0000-000000000000'::uuid)) WITH =,
      tstzrange(effective_from, effective_to) WITH &&) WHERE (status = 'APPROVED')
);
```

Resolution: `zone-specific APPROVED row covering :now` (app clock) → else `city default (zone_id NULL)`. The resolved `fee_config_id` is stored on the quote and the order.

**Default values** are owned by 16 §6.5 (R48); the DDL `DEFAULT`s mirror them. **Save-time validation** (app, because it spans JSON elements) guarantees "every serviceable point has exactly one slab" (16 §6.1): slabs contiguous from `fromM = 0` with `toM > fromM`, and `ceil(max_serviceable_radius_m × road_factor_milli / 1000) < last.toM`.

### 5.3 Tax rules [LEGAL]

```sql
CREATE TABLE tax_rules (
  id                     uuid PRIMARY KEY,
  country_code           char(2) NOT NULL DEFAULT 'IN',
  component              text NOT NULL CHECK (component IN ('ITEM_TOTAL','PACKAGING','DELIVERY_FEE',
                                                            'PLATFORM_FEE','SMALL_CART_FEE','COMMISSION')),
  tax_category           text NOT NULL DEFAULT 'DEFAULT',   -- 'RESTAURANT_SERVICE' for food lines
  rate_bps               int NOT NULL CHECK (rate_bps BETWEEN 0 AND 10000),
  split                  text NOT NULL CHECK (split IN ('CGST_SGST','IGST')),   -- intra-state ⇒ CGST+SGST halves
  liable_party           text NOT NULL CHECK (liable_party IN ('PLATFORM_SEC_9_5','PLATFORM','RESTAURANT')),
  price_inclusive        boolean NOT NULL DEFAULT false,   -- ruling 8: food lines exclusive (false); fee lines inclusive (true) [OPEN — CA]
  sac_code               text,
  legal_basis            text NOT NULL,                     -- notification reference, reviewed by CA
  effective_from         timestamptz NOT NULL,
  effective_to           timestamptz,
  created_by             uuid REFERENCES users(id),      -- rows are seeded by migration after CA sign-off (no CRUD UI, C20)
  created_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ex_tax_rules__no_overlap EXCLUDE USING gist (
      country_code WITH =, component WITH =, tax_category WITH =, tstzrange(effective_from, effective_to) WITH &&)
);
```

Seed values are proposals only, **to be confirmed by a chartered accountant before launch [LEGAL]**:

| Component | Rate | Liable | Basis |
|---|---|---|---|
| `ITEM_TOTAL` + `PACKAGING` (category `RESTAURANT_SERVICE`) | 5% (2.5% CGST + 2.5% SGST), no ITC | `PLATFORM_SEC_9_5` | Restaurant services supplied through an e-commerce operator are taxed under CGST §9(5) (since 1 Jan 2022). Verify the current notification. |
| `DELIVERY_FEE` | 18%, **inclusive** in the displayed fee (ruling 8, [OPEN — CA]) | `PLATFORM_SEC_9_5` when the rider is unregistered | Local delivery services through an ECO were notified under §9(5) at 18% from 22 Sep 2025 (Notification 17/2025-CT per [a2ztaxcorp summary](https://a2ztaxcorp.net/gst-alert-local-delivery-services-to-attract-18-tax-from-september-22-says-cbic/) and [PIB doc](https://static.pib.gov.in/WriteReadData/specificdocs/documents/2025/sep/doc2025921642801.pdf), accessed 2026-10-04). **[LEGAL] verify.** |
| `PLATFORM_FEE`, `SMALL_CART_FEE` | 18%, inclusive (ruling 8) | `PLATFORM` | Platform's own service. |
| `COMMISSION` (charged to the restaurant, not on the customer bill) | 18% | `PLATFORM` | Appears on the restaurant's commission invoice and settlement. |

### 5.4 Promotions

```sql
CREATE TABLE coupons (
  id                       uuid PRIMARY KEY,
  city_id                  uuid REFERENCES cities(id),     -- NULL = all cities
  kind                     text NOT NULL DEFAULT 'CAMPAIGN' CHECK (kind IN ('CAMPAIGN','GOODWILL')),  -- ruling 9
  owner_user_id            uuid REFERENCES users(id),      -- GOODWILL: the only user who may redeem it
  source_ticket_id         uuid,                           -- ref: support.support_tickets (GOODWILL)
  source_order_id          uuid,                           -- ref: ordering.orders being compensated
  code                     citext NOT NULL CHECK (code ~ '^[A-Za-z0-9]{4,24}$'),
  title                    text NOT NULL,
  title_i18n               jsonb NOT NULL DEFAULT '{}',
  terms                    text NOT NULL,
  terms_i18n               jsonb NOT NULL DEFAULT '{}',
  discount_type            text NOT NULL CHECK (discount_type IN ('FLAT','PERCENT')),
  applies_to               text NOT NULL DEFAULT 'ITEM_TOTAL' CHECK (applies_to IN ('ITEM_TOTAL','DELIVERY_FEE')),
  flat_paise               bigint CHECK (flat_paise > 0),
  percent_bps              int CHECK (percent_bps BETWEEN 1 AND 10000),
  max_discount_paise       bigint CHECK (max_discount_paise > 0),
  min_order_paise          bigint NOT NULL DEFAULT 0 CHECK (min_order_paise >= 0),   -- on item total
  valid_from               timestamptz NOT NULL,
  valid_until              timestamptz NOT NULL,
  per_user_limit           int NOT NULL DEFAULT 1 CHECK (per_user_limit >= 1),
  global_limit             int CHECK (global_limit >= 1),
  redemption_count         int NOT NULL DEFAULT 0,          -- counts RESERVED + APPLIED
  budget_paise             bigint CHECK (budget_paise > 0),
  budget_used_paise        bigint NOT NULL DEFAULT 0,
  funded_by                text NOT NULL CHECK (funded_by IN ('PLATFORM','RESTAURANT')),   -- SHARED funding is V1.1 (C16)
  new_users_only           boolean NOT NULL DEFAULT false,  -- delivered_order_count = 0
  payment_methods          text[],                           -- NULL = any; e.g. {ONLINE}
  is_public                boolean NOT NULL DEFAULT true,    -- listed in "Offers" vs secret code
  status                   text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','PAUSED','ENDED','ARCHIVED')),
  created_by               uuid REFERENCES users(id),
  version                  int NOT NULL DEFAULT 1,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_coupons__shape CHECK (
       (discount_type = 'FLAT'    AND flat_paise IS NOT NULL AND percent_bps IS NULL)
    OR (discount_type = 'PERCENT' AND percent_bps IS NOT NULL AND flat_paise IS NULL AND max_discount_paise IS NOT NULL)),
  CONSTRAINT ck_coupons__validity CHECK (valid_until > valid_from),
  CONSTRAINT ck_coupons__limits   CHECK (global_limit IS NULL OR redemption_count <= global_limit),
  CONSTRAINT ck_coupons__budget   CHECK (budget_paise IS NULL OR budget_used_paise <= budget_paise),
  CONSTRAINT ck_coupons__goodwill CHECK (kind <> 'GOODWILL' OR (owner_user_id IS NOT NULL AND discount_type = 'FLAT'
                                         AND global_limit = 1 AND per_user_limit = 1 AND is_public = false))
);
CREATE UNIQUE INDEX ux_coupons__code ON coupons (code) WHERE status <> 'ARCHIVED';
CREATE INDEX ix_coupons__live ON coupons (city_id, valid_until) WHERE status = 'ACTIVE';
CREATE INDEX ix_coupons__owner ON coupons (owner_user_id) WHERE kind = 'GOODWILL' AND status = 'ACTIVE';

CREATE TABLE coupon_targets (                       -- empty set = all restaurants/zones in the coupon's city
  coupon_id     uuid NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
  target_type   text NOT NULL CHECK (target_type IN ('ZONE','RESTAURANT')),   -- CUISINE/USER targets are V1.1 (C16)
  target_id     uuid NOT NULL,                      -- ref (logical) to the target's module
  PRIMARY KEY (coupon_id, target_type, target_id)
);
CREATE INDEX ix_coupon_targets__target ON coupon_targets (target_type, target_id);

CREATE TABLE coupon_redemptions (
  id                         uuid PRIMARY KEY,
  coupon_id                  uuid NOT NULL REFERENCES coupons(id),
  user_id                    uuid NOT NULL REFERENCES users(id),
  order_id                   uuid NOT NULL UNIQUE,        -- ref: ordering.orders
  discount_paise             bigint NOT NULL CHECK (discount_paise > 0),
  platform_funded_paise      bigint NOT NULL,
  restaurant_funded_paise    bigint NOT NULL,
  status                     text NOT NULL CHECK (status IN ('RESERVED','APPLIED','RELEASED')),
  reserved_at                timestamptz NOT NULL DEFAULT now(),
  applied_at                 timestamptz,                 -- at DELIVERED
  released_at                timestamptz,                 -- any non-delivered terminal state EXCEPT customer-fault UNDELIVERABLE,
                                                          -- which burns the coupon (stays APPLIED) (RV-048)
  CONSTRAINT ck_coupon_redemptions__split CHECK (platform_funded_paise + restaurant_funded_paise = discount_paise)
);
CREATE INDEX ix_coupon_redemptions__user ON coupon_redemptions (coupon_id, user_id) WHERE status IN ('RESERVED','APPLIED');
```

**Reservation (tx-participating `promotions.Reserve`, 08 §4.1):**
1. Run `SELECT … FROM coupons WHERE id=$1 FOR UPDATE`.
2. Count the user's live redemptions and compare with `per_user_limit`.
3. Run `UPDATE coupons SET redemption_count = redemption_count + 1, budget_used_paise = budget_used_paise + $d …`. The CHECKs then enforce the global limit and budget atomically.
4. Insert `RESERVED`.

Release reverses step 3. **Goodwill coupons (R9)** replace a wallet. They are issued by support (`ADMIN_SUPPORT`; maker-checker above `approvals.goodwill_threshold_paise`, 13 §5.1). For COD-order compensation a coupon is only one of the customer's two choices; the other is a manual UPI refund (R29, §7). Each goodwill coupon is a `FLAT`, single-use, non-public coupon bound to `owner_user_id`, platform-funded and valid 30 days `[ASSUMPTION]`. It is journalled to `EXPENSE_GOODWILL` when redeemed (14 §10.2 has no goodwill-liability account; `[OPEN — Finance]` whether issued-but-unredeemed coupons need an accrual). Restaurant-funded coupons must target only restaurants that signed up for them (app rule plus admin UI). Coupon budgets are total budgets only (per-day budgets and bulk unique codes are V1.1, C16); budget changes are audited, not maker-checker (R31).

---

## 6. DDL — ordering

```sql
CREATE TABLE orders (
  id                            uuid PRIMARY KEY,
  code                          text NOT NULL UNIQUE CHECK (code ~ '^RV-[0-9A-HJKMNP-TV-Z]{6}$'),  -- Crockford base32
  city_id                       uuid NOT NULL REFERENCES cities(id),
  zone_id                       uuid NOT NULL,              -- ref: drop zone at placement
  customer_id                   uuid NOT NULL REFERENCES users(id),
  restaurant_id                 uuid NOT NULL,              -- ref: catalog.restaurants
  quote_id                      uuid NOT NULL,              -- ref: pricing.quotes
  status                        text NOT NULL CHECK (status IN ('PENDING_PAYMENT','PLACED','ACCEPTED','PREPARING',
                                  'READY_FOR_PICKUP','PICKED_UP','DELIVERED','PAYMENT_FAILED','REJECTED','CANCELLED','UNDELIVERABLE')),
  version                       int NOT NULL DEFAULT 1,
  payment_method                text NOT NULL CHECK (payment_method IN ('ONLINE','COD')),
  payment_status                text NOT NULL CHECK (payment_status IN ('PENDING','PAID','COD_DUE','COD_COLLECTED',
                                  'COD_NOT_COLLECTED','PARTIALLY_REFUNDED','REFUNDED','UNPAID')),
  currency                      char(3) NOT NULL DEFAULT 'INR',
  -- price snapshot (all customer-visible; must equal Σ order_charges, enforced by deferred trigger)
  item_total_paise              bigint NOT NULL CHECK (item_total_paise >= 0),
  packaging_paise               bigint NOT NULL DEFAULT 0 CHECK (packaging_paise >= 0),
  delivery_fee_paise            bigint NOT NULL DEFAULT 0 CHECK (delivery_fee_paise >= 0),
  platform_fee_paise            bigint NOT NULL DEFAULT 0 CHECK (platform_fee_paise >= 0),
  small_cart_fee_paise          bigint NOT NULL DEFAULT 0 CHECK (small_cart_fee_paise >= 0),
  discount_paise                bigint NOT NULL DEFAULT 0 CHECK (discount_paise >= 0),
  discount_platform_paise       bigint NOT NULL DEFAULT 0,
  discount_restaurant_paise     bigint NOT NULL DEFAULT 0,
  tax_paise                     bigint NOT NULL DEFAULT 0 CHECK (tax_paise >= 0),        -- EXCLUSIVE taxes added on top (food GST)
  tax_included_paise            bigint NOT NULL DEFAULT 0 CHECK (tax_included_paise >= 0), -- GST already inside fee lines (informational)
  round_off_paise               bigint NOT NULL DEFAULT 0 CHECK (round_off_paise BETWEEN -50 AND 50),  -- ruling 8: payable → whole rupee
  total_paise                   bigint NOT NULL CHECK (total_paise >= 0),
  coupon_id                     uuid,                       -- ref: promotions.coupons
  coupon_code                   citext,
  -- commercial snapshot (not customer-visible)
  fee_config_id                 uuid NOT NULL,              -- ref: pricing.fee_configs
  commission_plan_id            uuid,                       -- ref: ledger.commission_plans
  commission_bps                int NOT NULL CHECK (commission_bps BETWEEN 0 AND 3000),   -- 0–30% (register row 56)
  -- channel (M8: ops-assisted phone ordering, P1)
  placed_via                    text NOT NULL DEFAULT 'CUSTOMER_APP' CHECK (placed_via IN ('CUSTOMER_APP','OPS_ASSISTED')),
  placed_by_admin_id            uuid REFERENCES users(id),                               -- OPS_ASSISTED only
  -- context snapshots
  restaurant_snapshot           jsonb NOT NULL,   -- {name, legalName, addressLine, pinCode, phone, fssaiLicenseNo, gstin, location}
  delivery_address_snapshot     jsonb NOT NULL,   -- full Indian address + contact (§13 redaction after 180 d)
  pickup_location               geography(Point,4326) NOT NULL,
  drop_location                 geography(Point,4326) NOT NULL,
  straight_distance_m           int NOT NULL CHECK (straight_distance_m >= 0),
  est_road_distance_m           int NOT NULL CHECK (est_road_distance_m >= 0),
  customer_note                 text CHECK (char_length(customer_note) <= 200),       -- for the kitchen
  delivery_instructions         text CHECK (char_length(delivery_instructions) <= 200), -- for the rider
  requires_delivery_code        boolean NOT NULL DEFAULT false,  -- R39: ONLINE and total ≥ dispatch.delivery_code_min_payable_paise (13 SM-D13)
  delivery_code_enc             bytea,          -- 4-digit handover code, AES-GCM (stored so the customer app can show it, R39)
  delivery_code_key_id          text,           -- KEK version
  prep_time_min                 smallint CHECK (prep_time_min BETWEEN 5 AND 90),    -- set at accept (register row 58)
  promised_eta_at               timestamptz,    -- shown at placement (upper bound)
  eta_at                        timestamptz,    -- live estimate, updated on milestones
  -- milestone timestamps
  placed_at                     timestamptz,
  accepted_at                   timestamptz,
  preparing_at                  timestamptz,
  ready_at                      timestamptz,
  picked_up_at                  timestamptz,
  delivered_at                  timestamptz,
  closed_at                     timestamptz,    -- any terminal state
  -- terminal reason (CANCELLED / REJECTED / UNDELIVERABLE / PAYMENT_FAILED); 00 §2 cancel_reason + cancelled_by
  cancelled_by_role             text CHECK (cancelled_by_role IN ('CUSTOMER','RESTAURANT','RIDER','ADMIN','SYSTEM')),
  cancelled_by_user_id          uuid REFERENCES users(id),
  cancel_reason_code            text,           -- catalogue in 13 §6.3
  cancel_reason_text            text CHECK (char_length(cancel_reason_text) <= 500),
  fault_party                   text CHECK (fault_party IN ('CUSTOMER','RESTAURANT','RIDER','PLATFORM','NONE')),
  -- settlement snapshot (filled at DELIVERED / terminal; ledger is the authority)
  commission_paise              bigint,
  restaurant_net_paise          bigint,
  rider_pay_paise               bigint,
  client_app                    text,           -- 'customer-web/1.4.2'
  created_at                    timestamptz NOT NULL DEFAULT now(),
  updated_at                    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_orders__total CHECK (total_paise = item_total_paise + packaging_paise + delivery_fee_paise
                                     + platform_fee_paise + small_cart_fee_paise - discount_paise + tax_paise + round_off_paise),
  CONSTRAINT ck_orders__whole_rupee CHECK (round_off_paise = 0 OR total_paise % 100 = 0),
  CONSTRAINT ck_orders__discount_split CHECK (discount_paise = discount_platform_paise + discount_restaurant_paise),
  CONSTRAINT ck_orders__terminal_reason CHECK (
    status NOT IN ('CANCELLED','REJECTED','UNDELIVERABLE','PAYMENT_FAILED')
    OR (cancel_reason_code IS NOT NULL AND cancelled_by_role IS NOT NULL AND closed_at IS NOT NULL)),
  CONSTRAINT ck_orders__cod_never_pending_payment CHECK (NOT (payment_method = 'COD' AND status = 'PENDING_PAYMENT')),
  CONSTRAINT ck_orders__delivery_code CHECK (NOT requires_delivery_code OR (payment_method = 'ONLINE' AND delivery_code_enc IS NOT NULL)),
  CONSTRAINT ck_orders__assisted CHECK ((placed_via = 'OPS_ASSISTED') = (placed_by_admin_id IS NOT NULL))
);
CREATE INDEX ix_orders__customer        ON orders (customer_id, created_at DESC);
CREATE INDEX ix_orders__restaurant      ON orders (restaurant_id, created_at DESC);
CREATE INDEX ix_orders__restaurant_live ON orders (restaurant_id, status, placed_at)
  WHERE status IN ('PLACED','ACCEPTED','PREPARING','READY_FOR_PICKUP','PICKED_UP');
CREATE INDEX ix_orders__city_live       ON orders (city_id, status, created_at)
  WHERE status IN ('PENDING_PAYMENT','PLACED','ACCEPTED','PREPARING','READY_FOR_PICKUP','PICKED_UP');
CREATE INDEX ix_orders__city_created    ON orders (city_id, created_at DESC);          -- admin board, reports
CREATE UNIQUE INDEX ux_orders__quote    ON orders (quote_id);                            -- one order per quote

CREATE TABLE order_items (
  id                   uuid PRIMARY KEY,
  order_id             uuid NOT NULL REFERENCES orders(id),
  line_no              smallint NOT NULL,
  menu_item_id         uuid,                       -- ref (analytics only; snapshot is authoritative)
  variant_id           uuid,
  name                 text NOT NULL,              -- snapshot in the restaurant's canonical language
  name_i18n            jsonb NOT NULL DEFAULT '{}',
  variant_name         text,
  veg_type             text NOT NULL CHECK (veg_type IN ('VEG','NON_VEG','EGG')),   -- effective incl. addons
  addons_snapshot      jsonb NOT NULL DEFAULT '[]',  -- [{"groupName":"Extras","addonId":"…","name":"Raita","pricePaise":3000}]
  unit_price_paise     bigint NOT NULL CHECK (unit_price_paise >= 0),   -- variant/base price + Σ addon prices
  quantity             smallint NOT NULL CHECK (quantity BETWEEN 1 AND 20),      -- register row 57
  packaging_paise      bigint NOT NULL DEFAULT 0 CHECK (packaging_paise >= 0),   -- line total packaging
  line_total_paise     bigint NOT NULL,
  note                 text,
  tax_category         text NOT NULL DEFAULT 'RESTAURANT_SERVICE',
  CONSTRAINT ux_order_items__line UNIQUE (order_id, line_no),
  CONSTRAINT ck_order_items__total CHECK (line_total_paise = unit_price_paise * quantity)
);

CREATE TABLE order_charges (                    -- the customer bill, one row per line (append at creation; refunds don't edit it)
  id                   uuid PRIMARY KEY,
  order_id             uuid NOT NULL REFERENCES orders(id),
  line_no              smallint NOT NULL,
  component            text NOT NULL CHECK (component IN ('ITEM_TOTAL','PACKAGING','DELIVERY_FEE',
                                         'PLATFORM_FEE','SMALL_CART_FEE','DISCOUNT','TAX','ROUND_OFF')),   -- no SURGE_FEE (R30)
  label                text NOT NULL,             -- 'GST on food (CGST 2.5%)'
  label_i18n           jsonb NOT NULL DEFAULT '{}',
  amount_paise         bigint NOT NULL,           -- signed; DISCOUNT ≤ 0
  tax_type             text CHECK (tax_type IN ('CGST','SGST','IGST','CESS')),
  taxed_component      text,                      -- for TAX lines: which component(s) it taxes
  tax_rate_bps         int,
  taxable_base_paise   bigint,
  tax_rule_id          uuid,                      -- ref: pricing.tax_rules
  is_included          boolean NOT NULL DEFAULT false,  -- TAX line already contained in its taxed component (inclusive fees):
                                                        -- shown as "incl. GST ₹x", excluded from the payable sum
  collected_for        text NOT NULL CHECK (collected_for IN ('RESTAURANT','PLATFORM','GOVERNMENT')),
  funded_by            text CHECK (funded_by IN ('PLATFORM','RESTAURANT')),   -- DISCOUNT lines
  CONSTRAINT ux_order_charges__line UNIQUE (order_id, line_no),
  CONSTRAINT ck_order_charges__tax CHECK ((component = 'TAX') = (tax_type IS NOT NULL AND tax_rate_bps IS NOT NULL)),
  CONSTRAINT ck_order_charges__discount CHECK (component <> 'DISCOUNT' OR (amount_paise <= 0 AND funded_by IS NOT NULL)),
  CONSTRAINT ck_order_charges__included CHECK (NOT is_included OR component = 'TAX')
);
CREATE INDEX ix_order_charges__order ON order_charges (order_id);

-- Σ order_charges.amount_paise (excluding informational inclusive-tax lines) must equal orders.total_paise at commit
CREATE FUNCTION assert_order_bill_balanced() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE oid uuid := COALESCE(NEW.order_id, OLD.order_id); s bigint; t bigint;
BEGIN
  SELECT COALESCE(SUM(amount_paise),0) INTO s FROM order_charges WHERE order_id = oid AND NOT is_included;
  SELECT total_paise INTO t FROM orders WHERE id = oid;
  IF s <> t THEN RAISE EXCEPTION 'order % bill (%) <> total (%)', oid, s, t; END IF;
  RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER trg_order_charges_balanced AFTER INSERT OR UPDATE OR DELETE ON order_charges
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION assert_order_bill_balanced();
-- (same function fired from a constraint trigger on orders AFTER UPDATE OF total_paise, using NEW.id)

CREATE TABLE order_status_history (            -- append-only
  id                 uuid PRIMARY KEY,
  order_id           uuid NOT NULL REFERENCES orders(id),
  seq                int NOT NULL,                -- = orders.version after the transition
  from_status        text,                        -- NULL for creation
  to_status          text NOT NULL,
  command            text NOT NULL,               -- 'Accept', 'PaymentCaptured', 'AcceptTimeout' (13 §1.1, §2.1)
  actor_type         text NOT NULL CHECK (actor_type IN ('CUSTOMER','RESTAURANT','RIDER','ADMIN','SYSTEM','PAYMENT_PROVIDER')),
  actor_user_id      uuid REFERENCES users(id),
  actor_role         text,
  reason_code        text,
  reason_text        text,
  command_id         uuid,                        -- idempotency of the command (13 §7.2)
  metadata           jsonb NOT NULL DEFAULT '{}', -- e.g. {"prepTimeMin":20} / {"implicit":true}
  occurred_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_order_status_history__seq UNIQUE (order_id, seq)
);
CREATE UNIQUE INDEX ux_order_status_history__command ON order_status_history (order_id, command_id) WHERE command_id IS NOT NULL;
CREATE INDEX ix_order_status_history__occurred ON order_status_history USING brin (occurred_at);
```

**Snapshot JSON shapes** are defined once as Go structs in `ordering` and exported as JSON Schema (08 §3.3). `delivery_address_snapshot` example:

```json
{"label":"HOME","houseNo":"2-45/3","buildingStreet":"Sri Sai Residency, Lane 4","localityId":"0192…","localityName":"[ASSUMPTION] New Town",
 "areaText":null,"landmark":"Opp. Hanuman temple","city":"Mahabubnagar","state":"Telangana","pinCode":"509001",
 "lat":16.7488,"lng":78.0035,"contactName":"Ravi","contactPhone":"+9198XXXXXX12","instructions":"Call at gate"}
```

---

## 7. DDL — payments

The detailed PA flow, reconciliation and settlement-report tables are in **14**. These are the core tables 13 and 11 depend on.

```sql
CREATE TABLE payments (                         -- one per PA order ("intent"); COD orders get one row with provider='COD'
  id                     uuid PRIMARY KEY,
  order_id               uuid NOT NULL REFERENCES orders(id),   -- money-path FK (R41)
  city_id                uuid NOT NULL REFERENCES cities(id),
  provider               text NOT NULL CHECK (provider IN ('RAZORPAY','CASHFREE','PHONEPE','COD','MANUAL','FAKE')),
  status                 text NOT NULL CHECK (status IN ('CREATED','AUTHORIZED','CAPTURED','FAILED','EXPIRED',
                                                         'COD_PENDING','COD_COLLECTED','COD_NOT_COLLECTED')),
  method                 text CHECK (method IN ('UPI','CARD','NETBANKING','WALLET','EMI','PAYLATER','CASH','OTHER')),
  amount_paise           bigint NOT NULL CHECK (amount_paise > 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  amount_refunded_paise  bigint NOT NULL DEFAULT 0,
  provider_order_id      text,                    -- e.g. Razorpay order_xxx (receipt = our payment id)
  provider_payment_id    text,                    -- the captured attempt
  provider_fee_paise     bigint,                  -- PA MDR, from webhook/settlement
  provider_tax_paise     bigint,
  failure_code           text,
  expires_at             timestamptz,             -- payment pending timeout (13 §5)
  authorized_at          timestamptz,
  captured_at            timestamptz,
  version                int NOT NULL DEFAULT 1,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_payments__refund_bound CHECK (amount_refunded_paise BETWEEN 0 AND amount_paise)
);
CREATE UNIQUE INDEX ux_payments__provider_order   ON payments (provider, provider_order_id)   WHERE provider_order_id IS NOT NULL;
CREATE UNIQUE INDEX ux_payments__provider_payment ON payments (provider, provider_payment_id) WHERE provider_payment_id IS NOT NULL;
CREATE UNIQUE INDEX ux_payments__one_success      ON payments (order_id) WHERE status IN ('CAPTURED','COD_COLLECTED');
CREATE INDEX ix_payments__order   ON payments (order_id);
CREATE INDEX ix_payments__pending ON payments (expires_at) WHERE status IN ('CREATED','AUTHORIZED');

CREATE TABLE payment_attempts (                 -- every PA payment id seen for an intent (UPI retries, card failures)
  id                    uuid PRIMARY KEY,
  payment_id            uuid NOT NULL REFERENCES payments(id),
  provider_payment_id   text NOT NULL,
  method                text,
  status                text NOT NULL CHECK (status IN ('CREATED','AUTHORIZED','CAPTURED','FAILED','REFUNDED')),
  error_code            text,
  error_description     text,
  vpa_masked            text,                     -- 'ra***@okaxis' — never full VPA/card
  card_last4            text,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_payment_attempts__provider UNIQUE (payment_id, provider_payment_id)
);
-- A second CAPTURED attempt on the same intent (rare double payment) ⇒ automatic refund of the extra one (14).

CREATE TABLE payment_events (                   -- raw webhook / poll / client-callback log (append-only)
  id                    uuid PRIMARY KEY,
  provider              text NOT NULL,
  provider_event_id     text NOT NULL,           -- X-Razorpay-Event-Id; for polls: sha256(payload)
  source                text NOT NULL CHECK (source IN ('WEBHOOK','POLL','CLIENT_CALLBACK')),
  event_type            text NOT NULL,           -- 'payment.captured', 'refund.processed', …
  signature_verified    boolean NOT NULL,
  payload               jsonb NOT NULL,          -- REDACTED at ingest: contact, email, VPA and name fields removed (RV-030); kept 8 y
  raw_body_sha256       bytea NOT NULL,
  raw_body_object_key   text,                    -- full raw body in private bucket prefix 'webhooks-raw/' with a 180-day lifecycle rule
                                                 -- (disputes); the key dangles after expiry by design
  provider_order_id     text,
  provider_payment_id   text,
  payment_id            uuid REFERENCES payments(id),
  refund_id             uuid,                    -- same module; FK added after refunds
  received_at           timestamptz NOT NULL DEFAULT now()
);
-- dedupe ONLY verified events, so a forged event cannot "claim" a real event id first
CREATE UNIQUE INDEX ux_payment_events__dedupe ON payment_events (provider, provider_event_id) WHERE signature_verified;
CREATE INDEX ix_payment_events__payment ON payment_events (payment_id, received_at);
-- processing status lives in the River job (args: payment_event_id), keeping this table append-only

CREATE TABLE refunds (
  id                     uuid PRIMARY KEY,
  payment_id             uuid NOT NULL REFERENCES payments(id),
  order_id               uuid NOT NULL REFERENCES orders(id),   -- money-path FK (R41)
  city_id                uuid NOT NULL REFERENCES cities(id),
  channel                text NOT NULL DEFAULT 'PA' CHECK (channel IN ('PA','MANUAL_UPI','MANUAL_BANK')),
                                                 -- MANUAL_*: finance pays from rovo's account (COD compensation by customer choice, R29)
  payee_vpa_enc          bytea,                   -- MANUAL_UPI destination given by the customer
  payee_vpa_masked       text,
  pii_key_id             text,
  utr_reference          text,                    -- MANUAL_*: required once SUCCEEDED
  amount_paise           bigint NOT NULL CHECK (amount_paise > 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  reason_code            text NOT NULL,           -- 'ORDER_REJECTED','ORDER_CANCELLED','LATE_CAPTURE','DUPLICATE_PAYMENT','SUPPORT_GOODWILL'…
  reason_text            text,
  refund_key             text NOT NULL UNIQUE,    -- idempotency: 'order:<id>:full' | 'ticket:<id>:1'
  initiated_by_type      text NOT NULL CHECK (initiated_by_type IN ('SYSTEM','ADMIN')),
  initiated_by_user_id   uuid REFERENCES users(id),
  approval_id            uuid,                    -- maker-checker above threshold
  status                 text NOT NULL CHECK (status IN ('PENDING_APPROVAL','REQUESTED','PROCESSING','SUCCEEDED','FAILED','CANCELLED')),
  provider_refund_id     text,
  speed                  text CHECK (speed IN ('NORMAL','OPTIMUM','INSTANT')),
  failure_reason         text,
  requested_at           timestamptz NOT NULL DEFAULT now(),
  processed_at           timestamptz,
  version                int NOT NULL DEFAULT 1,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_refunds__manual CHECK (channel = 'PA' OR status <> 'SUCCEEDED' OR utr_reference IS NOT NULL),
  CONSTRAINT ck_refunds__upi CHECK (channel <> 'MANUAL_UPI' OR payee_vpa_enc IS NOT NULL)
);
CREATE UNIQUE INDEX ux_refunds__provider ON refunds (provider_refund_id) WHERE provider_refund_id IS NOT NULL;
CREATE INDEX ix_refunds__order ON refunds (order_id);
CREATE INDEX ix_refunds__open  ON refunds (status, requested_at) WHERE status IN ('REQUESTED','PROCESSING','FAILED');
ALTER TABLE payment_events ADD CONSTRAINT fk_payment_events__refund FOREIGN KEY (refund_id) REFERENCES refunds(id);
```

**Refund bound:** creating a refund locks the `payments` row `FOR UPDATE` and increments `amount_refunded_paise` (reservation). The CHECK makes over-refunding impossible. A `FAILED` refund that is abandoned decrements it. **COD orders (R29, register D3):** when a COD customer is owed money (missing/wrong items after paying cash), the customer **chooses** a manual UPI refund or a single-user goodwill coupon, never coupon-only `[LEGAL]`. A manual refund is a `refunds` row against the order's `COD` payment row with `channel = 'MANUAL_UPI'`. Finance pays it and records the UTR (11 `POST /admin/refunds/{id}/mark-paid`). It posts journal `cod_manual_refund.v1` (13 §6.2). Maker-checker applies above `approvals.refund_threshold_paise` (R31 family 1).

### 7.1 PA split settlement and reconciliation (14 §5, §16, §17; R25; M4)

These tables support **both** money-flow models (R25/R35): `transfers` is used only under PA split settlement; `pa_settlements` and `recon_exceptions` are used in both.

```sql
CREATE TABLE transfers (                         -- split-settlement transfer to a restaurant's PA linked account (Route / Easy Split)
  id                     uuid PRIMARY KEY,
  order_id               uuid NOT NULL REFERENCES orders(id),     -- money-path FK (R41)
  payment_id             uuid NOT NULL REFERENCES payments(id),
  restaurant_id          uuid NOT NULL,                            -- ref: catalog.restaurants
  city_id                uuid NOT NULL REFERENCES cities(id),
  provider               text NOT NULL CHECK (provider IN ('RAZORPAY','CASHFREE','FAKE')),
  provider_account_ref   text NOT NULL,                            -- linked-account / vendor id at the PA (not a bank number)
  provider_transfer_id   text,
  amount_paise           bigint NOT NULL CHECK (amount_paise > 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  on_hold                boolean NOT NULL DEFAULT true,            -- released at statement time (14 §17)
  release_at             timestamptz,
  status                 text NOT NULL DEFAULT 'CREATED' CHECK (status IN ('CREATED','PENDING','ON_HOLD','RELEASED',
                                                 'SETTLED','REVERSED','PARTIALLY_REVERSED','FAILED')),
  reversed_paise         bigint NOT NULL DEFAULT 0,
  payout_id              uuid,                                     -- ref: ledger.payouts (statement that released it)
  failure_reason         text,
  version                int NOT NULL DEFAULT 1,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_transfers__reversal CHECK (reversed_paise BETWEEN 0 AND amount_paise)
);
CREATE UNIQUE INDEX ux_transfers__provider ON transfers (provider, provider_transfer_id) WHERE provider_transfer_id IS NOT NULL;
CREATE UNIQUE INDEX ux_transfers__order ON transfers (order_id) WHERE status NOT IN ('FAILED','REVERSED');
CREATE INDEX ix_transfers__held ON transfers (restaurant_id, status) WHERE status IN ('ON_HOLD','PENDING');

CREATE TABLE pa_settlements (                    -- one ingested settlement/recon report or bank statement (idempotent per file)
  id                     uuid PRIMARY KEY,
  city_id                uuid REFERENCES cities(id),
  provider               text NOT NULL,                            -- 'RAZORPAY' | 'CASHFREE' | 'BANK:<bank code>'
  source                 text NOT NULL CHECK (source IN ('PA_REPORT','BANK_STATEMENT')),
  report_date            date NOT NULL,                            -- D-1 for the daily recon; statement date for banks
  provider_settlement_id text,                                     -- PA settlement id / UTR of the bank credit
  file_id                uuid,                                     -- ref: platform.file_objects (original file, private)
  file_sha256            bytea NOT NULL,
  expected_credit_paise  bigint,
  line_count             int NOT NULL DEFAULT 0,
  status                 text NOT NULL DEFAULT 'INGESTED' CHECK (status IN ('INGESTED','MATCHING','MATCHED','EXCEPTIONS')),
  ingested_at            timestamptz NOT NULL DEFAULT now(),
  completed_at           timestamptz,
  CONSTRAINT ux_pa_settlements__file UNIQUE (provider, source, file_sha256)
);
CREATE UNIQUE INDEX ux_pa_settlements__period ON pa_settlements (provider, source, report_date, provider_settlement_id)
  NULLS NOT DISTINCT;                                              -- catch-up completion marker (13 §5.2)

CREATE TABLE pa_settlement_lines (
  id                     uuid PRIMARY KEY,
  settlement_id          uuid NOT NULL REFERENCES pa_settlements(id),
  line_no                int NOT NULL,
  line_type              text NOT NULL CHECK (line_type IN ('PAYMENT','REFUND','TRANSFER','FEE','ADJUSTMENT','CHARGEBACK','BANK_CREDIT')),
  provider_entity_id     text,                                     -- pay_xxx / rfnd_xxx / trf_xxx / bank txn ref
  amount_paise           bigint NOT NULL,                          -- signed as in the report
  fee_paise              bigint,
  tax_on_fee_paise       bigint,
  utr_reference          text,
  occurred_at            timestamptz,
  match_status           text NOT NULL DEFAULT 'UNMATCHED' CHECK (match_status IN ('UNMATCHED','MATCHED','EXCEPTION','IGNORED')),
  payment_attempt_id     uuid REFERENCES payment_attempts(id),
  refund_id              uuid REFERENCES refunds(id),
  transfer_id            uuid REFERENCES transfers(id),
  journal_id             uuid,                                     -- ref: ledger.ledger_journals (J6-style settlement journal)
  raw                    jsonb NOT NULL DEFAULT '{}',              -- PII-minimised row as received
  CONSTRAINT ux_pa_settlement_lines__line UNIQUE (settlement_id, line_no)
);
CREATE INDEX ix_pa_settlement_lines__entity ON pa_settlement_lines (provider_entity_id);
CREATE INDEX ix_pa_settlement_lines__unmatched ON pa_settlement_lines (settlement_id) WHERE match_status = 'UNMATCHED';

CREATE TABLE recon_exceptions (                  -- finance work queue (14 §16.1)
  id                     uuid PRIMARY KEY,
  city_id                uuid REFERENCES cities(id),
  exception_type         text NOT NULL CHECK (exception_type IN ('MISSING_IN_LEDGER','MISSING_AT_PA','AMOUNT_MISMATCH',
                            'FEE_MISMATCH','CHARGEBACK','REFUND_FAILED','TRANSFER_FAILED','BANK_CREDIT_MISMATCH','OTHER')),
  settlement_line_id     uuid REFERENCES pa_settlement_lines(id),
  payment_id             uuid REFERENCES payments(id),
  refund_id              uuid REFERENCES refunds(id),
  transfer_id            uuid REFERENCES transfers(id),
  order_id               uuid,                                     -- ref (denormalised for the finance UI)
  amount_paise           bigint,
  expected_paise         bigint,
  status                 text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','INVESTIGATING','RESOLVED','WRITTEN_OFF')),
  resolution_note        text,
  resolution_journal_id  uuid,                                     -- ref: correcting journal, if any
  assigned_admin_id      uuid REFERENCES users(id),
  raised_at              timestamptz NOT NULL DEFAULT now(),
  resolved_at            timestamptz,
  resolved_by            uuid REFERENCES users(id)
);
CREATE INDEX ix_recon_exceptions__queue ON recon_exceptions (city_id, status, raised_at) WHERE status IN ('OPEN','INVESTIGATING');
```

The open-exception count and the `SUSPENSE` balance are finance KPIs (14 §16.1). A `WRITTEN_OFF` exception needs a ledger adjustment, which is maker-checker above the threshold (R31 family 1).

---

## 8. DDL — dispatch (riders, deliveries)

### 8.1 Riders: cold profile + hot availability (split on purpose)

The logical "rider" entity = `riders` (profile, KYC, vehicle; rarely written) + `rider_availability` (online state, location, cash; written every 30–60 s). Splitting them keeps location updates from churning the profile row and its indexes. It also gives dispatch one small table to lock with `FOR UPDATE SKIP LOCKED` (08 §7.2).

```sql
CREATE TABLE riders (
  id                       uuid PRIMARY KEY,
  user_id                  uuid NOT NULL UNIQUE REFERENCES users(id),
  city_id                  uuid NOT NULL REFERENCES cities(id),
  home_zone_id             uuid,                     -- ref: geo.zones
  status                   text NOT NULL DEFAULT 'APPLIED' CHECK (status IN
                             ('APPLIED','UNDER_REVIEW','CHANGES_REQUESTED','ACTIVE','SUSPENDED','REJECTED','OFFBOARDED')),
  status_reason            text,
  kyc_status               text NOT NULL DEFAULT 'NOT_SUBMITTED' CHECK (kyc_status IN
                             ('NOT_SUBMITTED','PENDING','VERIFIED','REJECTED','EXPIRED')),
  vehicle_type             text NOT NULL CHECK (vehicle_type IN ('BICYCLE','MOTORCYCLE','SCOOTER','EV_SCOOTER','EV_CYCLE')),
  vehicle_reg_no           text CHECK (vehicle_reg_no ~ '^[A-Z]{2}[0-9]{1,2}[A-Z]{0,3}[0-9]{1,4}$'),  -- e.g. TG06AB1234
  dl_number_enc            bytea,
  dl_last4                 text,
  dl_valid_until           date,
  pii_key_id               text,
  emergency_contact_name   text,
  emergency_contact_phone  text CHECK (emergency_contact_phone ~ '^\+[1-9][0-9]{7,14}$'),
  cash_limit_paise         bigint CHECK (cash_limit_paise >= 0),   -- per-rider override (audited, not maker-checker, R31); NULL = fee_configs
  -- gig/platform-worker registration (Code on Social Security 2020; 01 LEG-GIG-001; M5) [LEGAL — final field list per portal spec]
  legal_name               text,                     -- as on the ID document
  date_of_birth            date,                     -- needed by the portal; NOT collected from customers (minimisation)
  gender                   text CHECK (gender IN ('FEMALE','MALE','TRANSGENDER','UNDISCLOSED')),
  residential_state        text,
  residential_district     text,
  ss_portal_id_enc         bytea,                    -- worker id issued by the designated portal (e.g. e-Shram UAN); never Aadhaar
  ss_portal_id_last4       text,
  ss_registration_status   text NOT NULL DEFAULT 'PENDING' CHECK (ss_registration_status IN ('NOT_REQUIRED','PENDING','EXPORTED','REGISTERED','FAILED')),
  ss_last_exported_at      timestamptz,              -- set by the GIG_WORKER_REGISTRATION export (report_exports)
  approved_at              timestamptz,
  approved_by              uuid REFERENCES users(id),
  version                  int NOT NULL DEFAULT 1,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_riders__motor_needs_reg CHECK (vehicle_type IN ('BICYCLE','EV_CYCLE') OR status <> 'ACTIVE' OR vehicle_reg_no IS NOT NULL)
);
CREATE INDEX ix_riders__city_status ON riders (city_id, status);

CREATE TABLE rider_availability (                 -- rider availability state machine (ruling 11; 13 §4.3)
  rider_id                    uuid PRIMARY KEY REFERENCES riders(id),
  city_id                     uuid NOT NULL REFERENCES cities(id),
  state                       text NOT NULL DEFAULT 'OFFLINE' CHECK (state IN ('OFFLINE','AVAILABLE','ON_DELIVERY')),   -- no ON_BREAK (C12)
  state_changed_at            timestamptz NOT NULL DEFAULT now(),
  offline_reason              text CHECK (offline_reason IN ('RIDER','STALE_LOCATION','MISSED_OFFERS','ADMIN_FORCED',
                                                             'SUSPENDED','LOGOUT')),
  online_since                timestamptz,          -- replaces rider_shifts (C12); session history lives in audit/events
  last_location               geography(Point,4326),
  last_location_at            timestamptz,          -- tier 1 if ≤ dispatch.location_fresh_s, tier 2 if ≤ rider.auto_offline_after_s (R34, 13 §5.1)
  last_seen_at                timestamptz,          -- any ping, batched upload or SSE presence; 15 min silence → auto-offline (R34)
  last_location_accuracy_m    int,
  active_delivery_count       smallint NOT NULL DEFAULT 0 CHECK (active_delivery_count BETWEEN 0 AND 1),  -- V1: 1 (P11)
  consecutive_missed_offers   smallint NOT NULL DEFAULT 0,      -- 3 → auto-offline (08 §5.4)
  cash_in_hand_paise          bigint NOT NULL DEFAULT 0,        -- CACHED projection of ledger RIDER_CASH_IN_HAND (§9);
                                                                -- refreshed by ledger event; authoritative check reads ledger_account_balances
  cod_blocked                 boolean NOT NULL DEFAULT false,   -- prefilter: cash_in_hand ≥ limit (zero headroom). Eligibility per order is
                                                                -- cash_in_hand + cod_amount ≤ limit (R6), checked under lock
  updated_at                  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_rider_availability__delivery CHECK ((state = 'ON_DELIVERY') = (active_delivery_count > 0))
) WITH (fillfactor = 70);                         -- room for HOT updates on non-indexed columns
CREATE INDEX ix_rider_availability__dispatch ON rider_availability USING gist (last_location)
  WHERE state = 'AVAILABLE';                       -- KNN candidate search (16 §7)
CREATE INDEX ix_rider_availability__city_state ON rider_availability (city_id, state);

CREATE TABLE rider_kyc_documents (
  id                 uuid PRIMARY KEY,
  rider_id           uuid NOT NULL REFERENCES riders(id),
  doc_type           text NOT NULL CHECK (doc_type IN ('DRIVING_LICENSE','VEHICLE_RC','PAN_CARD','AADHAAR_MASKED','OTHER_OVD',
                                                       'BANK_PROOF','SELFIE','VEHICLE_INSURANCE','POLICE_VERIFICATION')),
  file_id            uuid NOT NULL,                  -- ref: platform.file_objects (KYC bucket, app-encrypted, 12 AUTH-D12)
  doc_number_enc     bytea,                          -- NEVER for AADHAAR_MASKED (no Aadhaar number field exists)
  doc_number_last4   text,
  pii_key_id         text,
  valid_until        date,
  status             text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED','SUPERSEDED')),
  reviewed_by        uuid REFERENCES users(id),
  reviewed_at        timestamptz,
  rejection_reason   text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_rider_kyc__no_aadhaar_number CHECK (doc_type <> 'AADHAAR_MASKED' OR doc_number_enc IS NULL)
);
CREATE UNIQUE INDEX ux_rider_kyc__current ON rider_kyc_documents (rider_id, doc_type) WHERE status IN ('PENDING','APPROVED');
CREATE INDEX ix_rider_kyc__expiry ON rider_kyc_documents (valid_until) WHERE status = 'APPROVED';

CREATE TABLE rider_location_pings (                -- plain table (C8); 30-day retention by batched delete (§13, §14)
  rider_id        uuid NOT NULL,                   -- no FK: high-volume telemetry
  recorded_at     timestamptz NOT NULL,            -- device time (clamped to received_at ± 5 min)
  received_at     timestamptz NOT NULL DEFAULT now(),
  location        geography(Point,4326) NOT NULL,
  accuracy_m      int,
  speed_mps       real,
  heading_deg     smallint,
  delivery_id     uuid,                            -- set while on an active delivery
  PRIMARY KEY (rider_id, recorded_at)
);
CREATE INDEX ix_rider_location_pings__recorded ON rider_location_pings USING brin (recorded_at);   -- retention sweep
CREATE INDEX ix_rider_location_pings__delivery ON rider_location_pings (delivery_id, recorded_at) WHERE delivery_id IS NOT NULL;
```

Ping storage policy (extends 08 §6.3; RV-040): the rider PWA uploads **batched** pings (1–10 points, every `rider.ping_interval_s`, R27). PWAs stop reporting while the rider is in a navigation app, so pings are **not** evidence of a full route. Dispute evidence is the location captured at each milestone tap (`delivery_status_history.location`). Stored: at most 1 point per 60 s during an active delivery and 1 per 5 min while idle. The live position is always upserted into `rider_availability`.

### 8.2 Deliveries and offers

```sql
CREATE TABLE deliveries (
  id                          uuid PRIMARY KEY,
  order_id                    uuid NOT NULL UNIQUE REFERENCES orders(id),   -- money-path FK (R41); one delivery per order in V1
  city_id                     uuid NOT NULL REFERENCES cities(id),
  zone_id                     uuid NOT NULL,                 -- ref: drop zone
  restaurant_id               uuid NOT NULL,                 -- ref (denormalised for rider screens & queries)
  rider_id                    uuid REFERENCES riders(id),
  status                      text NOT NULL DEFAULT 'UNASSIGNED' CHECK (status IN ('UNASSIGNED','OFFERED','ASSIGNED',
                                'AT_RESTAURANT','PICKED_UP','AT_DROP','DELIVERED','FAILED','CANCELLED')),
  version                     int NOT NULL DEFAULT 1,
  pickup_location             geography(Point,4326) NOT NULL,
  drop_location               geography(Point,4326) NOT NULL,
  est_road_distance_m         int NOT NULL,
  dispatch_after              timestamptz NOT NULL,          -- ruling 7: accepted_at + max(0, prep − travel − buffer)
  dispatch_round              smallint NOT NULL DEFAULT 0,
  search_radius_m             int,
  dispatch_exhausted_at       timestamptz,                   -- ops alert raised
  cod_amount_paise            bigint NOT NULL DEFAULT 0 CHECK (cod_amount_paise >= 0),
  cod_collected_paise         bigint CHECK (cod_collected_paise >= 0),
  requires_delivery_code      boolean NOT NULL DEFAULT false,   -- copied from orders (R39)
  delivery_code_attempts      smallint NOT NULL DEFAULT 0 CHECK (delivery_code_attempts <= 5),
  restaurant_skipped_ready    boolean NOT NULL DEFAULT false, -- ruling 4: picked up while order PREPARING
  -- undeliverable flow (ruling 5): rider REQUESTS, support CONFIRMS
  undeliverable_requested_at  timestamptz,
  undeliverable_reason_code   text,                          -- reason_codes (category DELIVERY_FAIL)
  undeliverable_ticket_id     uuid,                          -- ref: support.support_tickets
  call_attempts               smallint NOT NULL DEFAULT 0,   -- counter; each tap is a contact_tap_log row (§8.3)
  -- pay
  waiting_seconds             int,
  rider_pay_paise             bigint,
  rider_pay_breakdown         jsonb,                         -- {"basePaise":2500,"distancePaise":1200,"waitPaise":0} (no surge, R30)
  -- milestones
  assigned_at                 timestamptz,
  at_restaurant_at            timestamptz,
  picked_up_at                timestamptz,
  at_drop_at                  timestamptz,
  delivered_at                timestamptz,
  failed_at                   timestamptz,
  cancelled_at                timestamptz,
  close_reason_code           text,
  created_at                  timestamptz NOT NULL DEFAULT now(),
  updated_at                  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_deliveries__rider_required CHECK (
    status NOT IN ('ASSIGNED','AT_RESTAURANT','PICKED_UP','AT_DROP','DELIVERED','FAILED') OR rider_id IS NOT NULL),
  CONSTRAINT ck_deliveries__cod_collected CHECK (status <> 'DELIVERED' OR cod_amount_paise = 0 OR cod_collected_paise = cod_amount_paise)
);
-- V1: one active delivery per rider (P11). Drop this index when batching arrives.
CREATE UNIQUE INDEX ux_deliveries__rider_active ON deliveries (rider_id)
  WHERE status IN ('ASSIGNED','AT_RESTAURANT','PICKED_UP','AT_DROP');
CREATE INDEX ix_deliveries__dispatch_queue ON deliveries (city_id, dispatch_after) WHERE status IN ('UNASSIGNED','OFFERED');
CREATE INDEX ix_deliveries__rider_history ON deliveries (rider_id, created_at DESC);
CREATE INDEX ix_deliveries__undeliverable ON deliveries (undeliverable_requested_at) WHERE status = 'AT_DROP' AND undeliverable_requested_at IS NOT NULL;

CREATE TABLE delivery_offers (
  id                   uuid PRIMARY KEY,
  delivery_id          uuid NOT NULL REFERENCES deliveries(id),
  rider_id             uuid NOT NULL REFERENCES riders(id),
  round                smallint NOT NULL,
  status               text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','ACCEPTED','DECLINED','EXPIRED','REVOKED')),   -- R16
  close_reason         text CHECK (close_reason IN ('RIDER_ACCEPTED','RIDER_DECLINED','TIMEOUT','REVOKED_MANUAL_ASSIGN',
                                                    'REVOKED_ORDER_CANCELLED','RIDER_WENT_OFFLINE')),
  decline_reason_code  text,                         -- reason_codes (category OFFER_DECLINE)
  offered_at           timestamptz NOT NULL DEFAULT now(),
  expires_at           timestamptz NOT NULL,         -- offered_at + 45 s (P11)
  responded_at         timestamptz,
  rider_location       geography(Point,4326),        -- where the rider was (audit, fairness analysis)
  rider_distance_m     int,
  score                numeric(10,3),
  est_earnings_paise   bigint NOT NULL,              -- shown on the offer card
  tier                 smallint NOT NULL DEFAULT 1 CHECK (tier IN (1, 2)),   -- R34: 1 = fresh location, 2 = stale (push)
  CONSTRAINT ck_delivery_offers__closed CHECK ((status = 'PENDING') = (close_reason IS NULL)),
  CONSTRAINT ck_delivery_offers__reason CHECK (
       status = 'PENDING'
    OR (status = 'ACCEPTED' AND close_reason = 'RIDER_ACCEPTED')
    OR (status = 'DECLINED' AND close_reason = 'RIDER_DECLINED')
    OR (status = 'EXPIRED'  AND close_reason = 'TIMEOUT')
    OR (status = 'REVOKED'  AND close_reason IN ('REVOKED_MANUAL_ASSIGN','REVOKED_ORDER_CANCELLED','RIDER_WENT_OFFLINE')))
);
CREATE UNIQUE INDEX ux_delivery_offers__pending_delivery ON delivery_offers (delivery_id) WHERE status = 'PENDING';
CREATE UNIQUE INDEX ux_delivery_offers__pending_rider    ON delivery_offers (rider_id)    WHERE status = 'PENDING';
CREATE INDEX ix_delivery_offers__delivery ON delivery_offers (delivery_id, offered_at);
CREATE INDEX ix_delivery_offers__rider    ON delivery_offers (rider_id, offered_at DESC);   -- acceptance-rate metrics

CREATE TABLE delivery_status_history (            -- append-only (same shape as order_status_history)
  id               uuid PRIMARY KEY,
  delivery_id      uuid NOT NULL REFERENCES deliveries(id),
  seq              int NOT NULL,
  from_status      text,
  to_status        text NOT NULL,
  command          text NOT NULL,
  actor_type       text NOT NULL CHECK (actor_type IN ('RIDER','ADMIN','SYSTEM','RESTAURANT')),
  actor_user_id    uuid REFERENCES users(id),
  rider_id         uuid REFERENCES riders(id),
  location         geography(Point,4326),           -- rider position reported with the milestone
  geofence_ok      boolean,                         -- within 200 m / 300 m of target (soft check, 13 §2.1 D-07/D-09)
  reason_code      text,
  command_id       uuid,
  client_occurred_at timestamptz,                   -- offline-queued milestones (18)
  metadata         jsonb NOT NULL DEFAULT '{}',
  occurred_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_delivery_status_history__seq UNIQUE (delivery_id, seq)
);
CREATE UNIQUE INDEX ux_delivery_status_history__command ON delivery_status_history (delivery_id, command_id) WHERE command_id IS NOT NULL;
```

### 8.3 Safety and contact logs (M4)

```sql
CREATE TABLE sos_events (                         -- rider SOS (06 §11; 01 RDR-FLOW-013, P1)
  id                 uuid PRIMARY KEY,
  rider_id           uuid NOT NULL REFERENCES riders(id),
  city_id            uuid NOT NULL REFERENCES cities(id),
  delivery_id        uuid REFERENCES deliveries(id),          -- active delivery, if any
  kind               text NOT NULL CHECK (kind IN ('ACCIDENT','UNSAFE','HARASSMENT','MEDICAL','OTHER')),
  location           geography(Point,4326),                   -- fresh fix attempted; else last known
  location_at        timestamptz,
  note               text CHECK (char_length(note) <= 500),
  status             text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','ACKNOWLEDGED','RESOLVED','FALSE_ALARM')),
  acknowledged_by    uuid REFERENCES users(id),
  acknowledged_at    timestamptz,
  resolved_by        uuid REFERENCES users(id),
  resolved_at        timestamptz,
  resolution_note    text,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_sos_events__open ON sos_events (city_id, created_at) WHERE status IN ('OPEN','ACKNOWLEDGED');
CREATE INDEX ix_sos_events__rider ON sos_events (rider_id, created_at DESC);

CREATE TABLE contact_tap_log (                    -- every call/contact tap (01 BR-CONT-001, ADM-ORD-002 "call-tap log")
  id                 uuid PRIMARY KEY,
  delivery_id        uuid NOT NULL REFERENCES deliveries(id),
  order_id           uuid NOT NULL,                           -- ref (for the admin order timeline)
  actor_type         text NOT NULL CHECK (actor_type IN ('RIDER','CUSTOMER','RESTAURANT','ADMIN')),
  actor_user_id      uuid REFERENCES users(id),
  target_type        text NOT NULL CHECK (target_type IN ('CUSTOMER','RIDER','RESTAURANT','SUPPORT')),
  channel            text NOT NULL DEFAULT 'TEL_LINK' CHECK (channel IN ('TEL_LINK','MASKED_CALL')),   -- masked calling V1.1
  location           geography(Point,4326),
  tapped_at          timestamptz NOT NULL,                    -- client time clamped ±5 min (as milestones)
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_contact_tap_log__delivery ON contact_tap_log (delivery_id, tapped_at);
```

`deliveries.call_attempts` is a cached count of `contact_tap_log` rows with `actor_type='RIDER' AND target_type='CUSTOMER'`, maintained in the same transaction. The undeliverable precondition (13 D-11) reads it.

---

## 9. DDL — ledger, commissions, payouts, invoices

Posting templates (which accounts each business event debits and credits) are owned by **14 §10**. This section fixes the structure and invariants.

```sql
CREATE TABLE ledger_accounts (
  id             uuid PRIMARY KEY,
  city_id        uuid NOT NULL REFERENCES cities(id),     -- per-city books (city P&L, GST by state)
  code           text NOT NULL CHECK (code IN (       -- chart of accounts = 14 §10.2 (RV-045, register row 66)
                   'PA_CLEARING','BANK','RIDER_CASH_IN_HAND','CUSTOMER_ADVANCES','REFUNDS_PAYABLE','RESTAURANT_PAYABLE',
                   'RIDER_PAYABLE','GST_OUTPUT_9_5_RESTAURANT','GST_OUTPUT_9_5_DELIVERY','GST_OUTPUT_OWN','GST_INPUT_CREDIT',
                   'TDS_194O_PAYABLE','REVENUE_COMMISSION','REVENUE_DELIVERY_FEE','REVENUE_PLATFORM_FEE','EXPENSE_RIDER_PAY',
                   'EXPENSE_PA_FEES','EXPENSE_PROMOTIONS','EXPENSE_GOODWILL','SUSPENSE')),
  account_type   text NOT NULL CHECK (account_type IN ('ASSET','LIABILITY','REVENUE','EXPENSE','EQUITY')),   -- 14 "Income" = REVENUE
  normal_side    char(1) NOT NULL CHECK (normal_side IN ('D','C')),          -- 14 §10.1
  owner_type     text NOT NULL CHECK (owner_type IN ('PLATFORM','RESTAURANT','RIDER')),
  owner_id       uuid,                                     -- restaurant id / rider id (logical ref); NULL for PLATFORM
  gstin_state    char(2) CHECK (gstin_state ~ '^[0-9]{2}$'),   -- GST_* accounts: state of the GSTIN (GSTR-3B by §9(5) category, 14 §12.1)
  currency       char(3) NOT NULL DEFAULT 'INR',
  is_active      boolean NOT NULL DEFAULT true,
  created_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_ledger_accounts__key UNIQUE NULLS NOT DISTINCT (city_id, code, owner_type, owner_id, gstin_state, currency),
  CONSTRAINT ck_ledger_accounts__gst CHECK ((code LIKE 'GST\_%') = (gstin_state IS NOT NULL)),
  CONSTRAINT ck_ledger_accounts__owner CHECK ((owner_type = 'PLATFORM') = (owner_id IS NULL))
);

CREATE TABLE ledger_journals (                    -- append-only
  id                 uuid PRIMARY KEY,
  city_id            uuid NOT NULL REFERENCES cities(id),
  entry_type         text NOT NULL CHECK (entry_type IN ('PAYMENT_CAPTURED','ORDER_SETTLED','COD_COLLECTED','RIDER_EARNING',
                       'REFUND','PA_FEE','PA_SETTLEMENT','TRANSFER','PAYOUT','COD_DEPOSIT','GOODWILL_REDEEMED',
                       'CANCELLATION_COMPENSATION','ADJUSTMENT','REVERSAL')),
  adjustment_type    text CHECK (adjustment_type IN ('MG_TOPUP','PEAK_BONUS','RECOVERY','CASH_CORRECTION','WRITE_OFF','OTHER')),
                                                 -- M6: MG_TOPUP = pilot rider minimum guarantee (R47); PEAK_BONUS replaces surge (R30)
  source_type        text NOT NULL,              -- 'order','refund','payout','cod_deposit','approval'
  source_id          uuid NOT NULL,
  rule               text NOT NULL,              -- posting rule name/version, e.g. 'order_settled.v1'
  idempotency_key    text NOT NULL UNIQUE,       -- '<source_type>:<source_id>:<rule>' (08 §7.1)
  reverses_journal_id uuid REFERENCES ledger_journals(id),
  approval_id        uuid,                       -- ADJUSTMENT above approvals.adjustment_threshold_paise needs maker-checker (R31 family 1)
  description        text,
  occurred_at        timestamptz NOT NULL,       -- business time from the app clock (R19); payout cut-offs use this
  accounting_date    date NOT NULL,              -- occurred_at in city timezone
  created_by_user_id uuid REFERENCES users(id),
  created_at         timestamptz NOT NULL DEFAULT now(),   -- technical insert time only
  CONSTRAINT ck_ledger_journals__adjustment CHECK ((entry_type = 'ADJUSTMENT') = (adjustment_type IS NOT NULL))
);
CREATE INDEX ix_ledger_journals__source ON ledger_journals (source_type, source_id);
CREATE INDEX ix_ledger_journals__date   ON ledger_journals (city_id, accounting_date);

CREATE TABLE ledger_postings (                    -- append-only
  id             uuid PRIMARY KEY,
  journal_id     uuid NOT NULL REFERENCES ledger_journals(id),
  account_id     uuid NOT NULL REFERENCES ledger_accounts(id),
  amount_paise   bigint NOT NULL CHECK (amount_paise <> 0),   -- + debit, − credit
  currency       char(3) NOT NULL DEFAULT 'INR',
  order_id       uuid REFERENCES orders(id),                   -- money-path FK (R41), denormalised for statements
  memo           text,
  created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_ledger_postings__account ON ledger_postings (account_id, created_at);
CREATE INDEX ix_ledger_postings__journal ON ledger_postings (journal_id);
CREATE INDEX ix_ledger_postings__order   ON ledger_postings (order_id) WHERE order_id IS NOT NULL;

CREATE FUNCTION assert_journal_balanced() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s bigint; n int;
BEGIN
  SELECT COALESCE(SUM(amount_paise),0), COUNT(*) INTO s, n FROM ledger_postings WHERE journal_id = NEW.journal_id;
  IF s <> 0 OR n < 2 THEN RAISE EXCEPTION 'journal % unbalanced (sum %, lines %)', NEW.journal_id, s, n; END IF;
  RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER trg_ledger_postings_balanced AFTER INSERT ON ledger_postings
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION assert_journal_balanced();
-- + forbid_mutation() triggers on ledger_journals and ledger_postings

CREATE TABLE ledger_account_balances (            -- running balance, updated in the posting tx (08 §7.2)
  account_id      uuid PRIMARY KEY REFERENCES ledger_accounts(id),
  balance_paise   bigint NOT NULL DEFAULT 0,      -- Σ postings (debit-positive)
  last_journal_id uuid,
  version         bigint NOT NULL DEFAULT 0,
  updated_at      timestamptz NOT NULL DEFAULT now()
);
```

Only owner-type accounts and gating accounts (`RIDER_CASH_IN_HAND`, `RIDER_PAYABLE`, `RESTAURANT_PAYABLE`) maintain a `ledger_account_balances` row synchronously. Hot platform revenue accounts are computed by `SUM` (indexed) to avoid a single-row hotspot. A nightly job verifies `balance = Σ postings` for every balance row and alerts on drift.

```sql
CREATE TABLE commission_plans (
  id               uuid PRIMARY KEY,
  restaurant_id    uuid NOT NULL,                  -- ref: catalog.restaurants
  city_id          uuid NOT NULL REFERENCES cities(id),
  commission_bps   int NOT NULL CHECK (commission_bps BETWEEN 0 AND 3000),   -- default per 16 §6.5; 0–30% (register row 56)
  basis            text NOT NULL DEFAULT 'ITEM_TOTAL_NET_OF_RESTAURANT_DISCOUNT'
                     CHECK (basis IN ('ITEM_TOTAL_NET_OF_RESTAURANT_DISCOUNT','ITEM_TOTAL_PLUS_PACKAGING_NET')),  -- [OPEN] packaging in basis?
  effective_from   timestamptz NOT NULL,
  effective_to     timestamptz,
  contract_ref     text,
  approval_id      uuid,                           -- maker-checker (R31 family 3)
  created_by       uuid REFERENCES users(id),
  created_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_commission_plans__range CHECK (effective_to IS NULL OR effective_to > effective_from),
  CONSTRAINT ex_commission_plans__no_overlap EXCLUDE USING gist (restaurant_id WITH =, tstzrange(effective_from, effective_to) WITH &&)
);

CREATE TABLE payout_accounts (                    -- bank / UPI destination for restaurants & riders
  id                     uuid PRIMARY KEY,
  owner_type             text NOT NULL CHECK (owner_type IN ('RESTAURANT','RIDER')),
  owner_id               uuid NOT NULL,           -- ref
  method                 text NOT NULL CHECK (method IN ('BANK_ACCOUNT','UPI')),
  account_holder_name    text NOT NULL,
  account_number_enc     bytea,                   -- BANK_ACCOUNT
  account_number_last4   text,
  ifsc                   text CHECK (ifsc ~ '^[A-Z]{4}0[A-Z0-9]{6}$'),
  upi_vpa_enc            bytea,                   -- UPI
  upi_vpa_masked         text,
  pii_key_id             text NOT NULL,
  verification_status    text NOT NULL DEFAULT 'UNVERIFIED' CHECK (verification_status IN ('UNVERIFIED','PENNY_DROP_OK','MANUAL_OK','FAILED')),
  is_primary             boolean NOT NULL DEFAULT false,
  status                 text NOT NULL DEFAULT 'PENDING_APPROVAL' CHECK (status IN ('PENDING_APPROVAL','ACTIVE','RETIRED')),
  approval_id            uuid,                    -- bank/UPI detail change is maker-checker (R31 family 4)
  cooling_off_until      timestamptz,             -- no payout to a new destination for 24 h [ASSUMPTION] (12 §2.5)
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_payout_accounts__shape CHECK (
       (method = 'BANK_ACCOUNT' AND account_number_enc IS NOT NULL AND ifsc IS NOT NULL)
    OR (method = 'UPI' AND upi_vpa_enc IS NOT NULL))
);
CREATE UNIQUE INDEX ux_payout_accounts__primary ON payout_accounts (owner_type, owner_id) WHERE is_primary AND status = 'ACTIVE';

CREATE TABLE payouts (
  id                       uuid PRIMARY KEY,
  city_id                  uuid NOT NULL REFERENCES cities(id),
  batch_id                 uuid NOT NULL,          -- one "payout run" (approval covers the batch)
  payee_type               text NOT NULL CHECK (payee_type IN ('RESTAURANT','RIDER')),
  payee_id                 uuid NOT NULL,          -- ref
  ledger_account_id        uuid NOT NULL REFERENCES ledger_accounts(id),   -- payee's PAYABLE account
  period_start             timestamptz NOT NULL,
  period_end               timestamptz NOT NULL,   -- cut-off: journals with occurred_at < period_end (app clock, R19; register row 41)
  gross_paise              bigint NOT NULL,
  deductions_paise         bigint NOT NULL DEFAULT 0,   -- TDS, COD netting, recoveries
  net_paise                bigint NOT NULL CHECK (net_paise > 0),
  currency                 char(3) NOT NULL DEFAULT 'INR',
  payout_account_id        uuid REFERENCES payout_accounts(id),
  payout_account_snapshot  jsonb,                  -- masked: {"method":"BANK_ACCOUNT","last4":"4321","ifsc":"SBIN0001234"}
  status                   text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','APPROVED','PAID','FAILED','CANCELLED')),
  approval_id              uuid,                   -- batch release is maker-checker (R31 family 2)
  method                   text CHECK (method IN ('NEFT','IMPS','UPI','RTGS','PA_PAYOUT','PA_TRANSFER_RELEASE')),
  utr_reference            text,
  paid_at                  timestamptz,
  paid_by_user_id          uuid REFERENCES users(id),
  journal_id               uuid REFERENCES ledger_journals(id),
  version                  int NOT NULL DEFAULT 1,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_payouts__net CHECK (net_paise = gross_paise - deductions_paise),
  CONSTRAINT ck_payouts__paid CHECK (status <> 'PAID' OR (utr_reference IS NOT NULL AND paid_at IS NOT NULL AND journal_id IS NOT NULL))
);
CREATE UNIQUE INDEX ux_payouts__payee_period ON payouts (payee_type, payee_id, period_end) WHERE status <> 'CANCELLED';
CREATE INDEX ix_payouts__batch ON payouts (batch_id);

CREATE TABLE payout_items (
  id                 uuid PRIMARY KEY,
  payout_id          uuid NOT NULL REFERENCES payouts(id) ON DELETE CASCADE,   -- items deletable only while DRAFT (app rule)
  ledger_posting_id  uuid NOT NULL REFERENCES ledger_postings(id),
  order_id           uuid REFERENCES orders(id),   -- money-path FK (R41)
  item_type          text NOT NULL CHECK (item_type IN ('ORDER_EARNING','COMMISSION','COMMISSION_GST','TDS','REFUND_RECOVERY',
                                                        'COD_NETTING','CANCELLATION_COMPENSATION','ADJUSTMENT','PENALTY','DELIVERY_EARNING',
                                                        'MG_TOPUP','PEAK_BONUS')),
  amount_paise       bigint NOT NULL,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_payout_items__posting ON payout_items (ledger_posting_id);   -- a posting is paid out at most once
CREATE INDEX ix_payout_items__payout ON payout_items (payout_id);

CREATE TABLE cod_deposits (                       -- rider hands cash to platform (UPI to company account / cash at hub)
  id                 uuid PRIMARY KEY,
  rider_id           uuid NOT NULL,               -- ref: dispatch.riders
  city_id            uuid NOT NULL REFERENCES cities(id),
  amount_paise       bigint NOT NULL CHECK (amount_paise > 0),
  currency           char(3) NOT NULL DEFAULT 'INR',
  method             text NOT NULL CHECK (method IN ('UPI_TO_COMPANY','BANK_DEPOSIT','CASH_AT_HUB','PAYOUT_NETTING')),
  reference          text,                        -- UPI ref / deposit slip no.
  proof_file_id      uuid,                        -- ref: platform.file_objects
  status             text NOT NULL DEFAULT 'DECLARED' CHECK (status IN ('DECLARED','VERIFIED','REJECTED')),
  declared_at        timestamptz NOT NULL DEFAULT now(),
  verified_by        uuid REFERENCES users(id),
  verified_at        timestamptz,
  rejection_reason   text,
  journal_id         uuid REFERENCES ledger_journals(id),   -- posted on VERIFIED: Dr BANK / Cr RIDER_CASH_IN_HAND
  created_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_cod_deposits__verified CHECK (status <> 'VERIFIED' OR (verified_by IS NOT NULL AND journal_id IS NOT NULL))
);
CREATE INDEX ix_cod_deposits__queue ON cod_deposits (city_id, status, declared_at) WHERE status = 'DECLARED';
CREATE INDEX ix_cod_deposits__rider ON cod_deposits (rider_id, declared_at DESC);

CREATE TABLE invoice_sequences (                  -- gap-free numbering per GST registration and financial year [LEGAL]
  series          text NOT NULL,                  -- e.g. 'MBNR-F' (food §9(5)), 'MBNR-S' (platform services), 'MBNR-C' (commission)
  financial_year  text NOT NULL,                  -- '2026-27'
  next_value      bigint NOT NULL DEFAULT 1,
  PRIMARY KEY (series, financial_year)
);
CREATE TABLE invoices (                           -- issued documents; format and fields owned by 14
  id               uuid PRIMARY KEY,
  city_id          uuid NOT NULL REFERENCES cities(id),
  series           text NOT NULL,
  financial_year   text NOT NULL,
  number           text NOT NULL,                 -- ≤ 16 chars [LEGAL — GST invoice rules]
  kind             text NOT NULL CHECK (kind IN ('CUSTOMER_FOOD','CUSTOMER_SERVICES','RESTAURANT_COMMISSION','CREDIT_NOTE')),
  order_id         uuid REFERENCES orders(id),    -- money-path FK (R41)
  payee_type       text, payee_id uuid,           -- for commission invoices
  total_paise      bigint NOT NULL,
  tax_paise        bigint NOT NULL,
  document         jsonb NOT NULL,                -- frozen invoice data
  file_id          uuid,                          -- rendered PDF (platform.file_objects)
  issued_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_invoices__number UNIQUE (series, financial_year, number)
);
CREATE INDEX ix_invoices__order ON invoices (order_id);
```

---

## 10. DDL — ratings, support, notifications

```sql
CREATE TABLE ratings (                            -- one per (order, target); restaurant = stars, rider = thumbs (04 §11)
  id                uuid PRIMARY KEY,
  order_id          uuid NOT NULL,                -- ref
  city_id           uuid NOT NULL REFERENCES cities(id),
  rater_user_id     uuid NOT NULL REFERENCES users(id),
  target_type       text NOT NULL CHECK (target_type IN ('RESTAURANT','RIDER')),
  restaurant_id     uuid,                         -- ref
  rider_id          uuid,                         -- ref
  stars             smallint CHECK (stars BETWEEN 1 AND 5),
  thumbs_up         boolean,
  tags              text[] NOT NULL DEFAULT '{}',
  is_excluded       boolean NOT NULL DEFAULT false,      -- removed from aggregates after moderation / dispute
  excluded_reason   text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_ratings__order_target UNIQUE (order_id, target_type),
  CONSTRAINT ck_ratings__target CHECK (
       (target_type = 'RESTAURANT' AND restaurant_id IS NOT NULL AND rider_id IS NULL AND stars IS NOT NULL AND thumbs_up IS NULL)
    OR (target_type = 'RIDER'      AND rider_id IS NOT NULL AND restaurant_id IS NULL AND thumbs_up IS NOT NULL AND stars IS NULL))
);
CREATE INDEX ix_ratings__restaurant ON ratings (restaurant_id, created_at DESC) WHERE target_type = 'RESTAURANT';
CREATE INDEX ix_ratings__rider      ON ratings (rider_id, created_at DESC)      WHERE target_type = 'RIDER';

CREATE TABLE reviews (                            -- public text for restaurant ratings only, ≤ 500 chars (04 §11)
  id                  uuid PRIMARY KEY,
  rating_id           uuid NOT NULL UNIQUE REFERENCES ratings(id),
  restaurant_id       uuid NOT NULL,              -- ref
  body                text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 500),
  language            text,
  -- C14: no moderation queue and no replies in V1. A profanity/PII filter runs at write; admins can hide.
  profanity_flagged   boolean NOT NULL DEFAULT false,   -- filter hit → stored but not shown
  is_hidden           boolean NOT NULL DEFAULT false,
  hidden_by           uuid REFERENCES users(id),
  hidden_at           timestamptz,
  hidden_reason       text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_reviews__restaurant_public ON reviews (restaurant_id, created_at DESC) WHERE NOT is_hidden AND NOT profanity_flagged;

CREATE TABLE rating_aggregates (                  -- read by catalog through the ratings interface (08)
  target_type     text NOT NULL CHECK (target_type IN ('RESTAURANT','RIDER')),
  target_id       uuid NOT NULL,
  rating_count    int NOT NULL DEFAULT 0,
  stars_sum       bigint NOT NULL DEFAULT 0,      -- restaurants
  thumbs_up_count int NOT NULL DEFAULT 0,         -- riders
  avg_30d         numeric(3,2),                   -- recomputed nightly
  updated_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (target_type, target_id)
);

CREATE TABLE support_tickets (
  id                  uuid PRIMARY KEY,
  code                text NOT NULL UNIQUE CHECK (code ~ '^TK-[0-9A-HJKMNP-TV-Z]{6}$'),
  city_id             uuid NOT NULL REFERENCES cities(id),
  requester_user_id   uuid REFERENCES users(id),        -- NULL for system-raised (e.g. undeliverable request is rider-raised)
  requester_app       text NOT NULL CHECK (requester_app IN ('CUSTOMER','RESTAURANT','RIDER','SYSTEM','ADMIN')),
  order_id            uuid,                             -- ref
  restaurant_id       uuid,                             -- ref
  rider_id            uuid,                             -- ref
  category            text NOT NULL CHECK (category IN ('MISSING_ITEMS','WRONG_ITEMS','QUALITY','LATE_DELIVERY','NOT_DELIVERED',
                        'PAYMENT_ISSUE','REFUND_STATUS','RIDER_BEHAVIOUR','RESTAURANT_CANNOT_FULFIL','UNDELIVERABLE_REQUEST',
                        'ACCOUNT','PAYOUT','COD_DEPOSIT','APP_ISSUE','DPDP_GRIEVANCE','OTHER')),
  is_dispute          boolean NOT NULL DEFAULT false,
  priority            text NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW','NORMAL','HIGH','URGENT')),
  status              text NOT NULL DEFAULT 'OPEN' CHECK (status IN
                        ('OPEN','IN_PROGRESS','AWAITING_REQUESTER','AWAITING_APPROVAL','RESOLVED','CLOSED','REOPENED')),
  assigned_admin_id   uuid REFERENCES users(id),
  subject             text NOT NULL,
  fault_party         text CHECK (fault_party IN ('CUSTOMER','RESTAURANT','RIDER','PLATFORM','NONE','UNDETERMINED')),
  resolution_code     text,                             -- reason_codes (category TICKET_RESOLUTION)
  resolution_note     text,
  refund_id           uuid,                             -- ref: payments.refunds
  goodwill_coupon_id  uuid,                             -- ref: promotions.coupons (ruling 9)
  sla_due_at          timestamptz NOT NULL,
  first_response_at   timestamptz,
  resolved_at         timestamptz,
  closed_at           timestamptz,
  version             int NOT NULL DEFAULT 1,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_support_tickets__queue ON support_tickets (city_id, priority, sla_due_at)
  WHERE status NOT IN ('RESOLVED','CLOSED');
CREATE INDEX ix_support_tickets__order ON support_tickets (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX ix_support_tickets__requester ON support_tickets (requester_user_id, created_at DESC);

CREATE TABLE ticket_messages (
  id                    uuid PRIMARY KEY,
  ticket_id             uuid NOT NULL REFERENCES support_tickets(id),
  kind                  text NOT NULL DEFAULT 'MESSAGE' CHECK (kind IN ('MESSAGE','INTERNAL_NOTE','ACTION','STATUS_CHANGE')),
  author_user_id        uuid REFERENCES users(id),
  author_type           text NOT NULL CHECK (author_type IN ('REQUESTER','AGENT','SYSTEM')),
  body                  text NOT NULL CHECK (char_length(body) <= 4000),
  attachment_file_ids   uuid[] NOT NULL DEFAULT '{}',
  action                jsonb,                         -- ACTION: {"type":"REFUND","refundId":"…","amountPaise":12000}
  created_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_ticket_messages__ticket ON ticket_messages (ticket_id, created_at);

CREATE TABLE notification_templates (
  id                uuid PRIMARY KEY,
  key               text NOT NULL,                 -- 'order.accepted.customer'
  channel           text NOT NULL CHECK (channel IN ('IN_APP','WEB_PUSH','SMS','EMAIL','VOICE')),   -- WhatsApp V1.1 (C2); VOICE = R43 P1
  locale            text NOT NULL,
  version           int NOT NULL,
  title             text,
  body              text NOT NULL,
  variables         text[] NOT NULL DEFAULT '{}',
  dlt_template_id   text,                          -- TRAI DLT (SMS) [LEGAL]
  dlt_sender_id     text,
  provider_template_name text,                     -- provider-side template name (voice/IVR, P1)
  status            text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('DRAFT','ACTIVE','RETIRED')),
  created_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_notification_templates__key UNIQUE (key, channel, locale, version)
);

CREATE TABLE notifications (                      -- in-app inbox (one per user per logical message)
  id                uuid PRIMARY KEY,
  user_id           uuid NOT NULL REFERENCES users(id),
  audience          text NOT NULL CHECK (audience IN ('customer','restaurant','rider','admin')),   -- R14
  city_id           uuid REFERENCES cities(id),
  category          text NOT NULL CHECK (category IN ('ORDER','DELIVERY','PAYMENT','PROMO','ACCOUNT','SUPPORT','OPS_ALERT','PAYOUT')),
  template_key      text NOT NULL,
  params            jsonb NOT NULL DEFAULT '{}',
  locale            text NOT NULL,
  title             text NOT NULL,                 -- rendered at creation in the user's locale
  body              text NOT NULL,
  deep_link         text,                          -- app-relative path, never an absolute URL
  priority          text NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('HIGH','NORMAL','LOW')),
  dedupe_key        text,                          -- 'order:<id>:accepted'
  source_event_id   uuid,                          -- domain event id from the River job args (R42)
  read_at           timestamptz,
  expires_at        timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_notifications__dedupe ON notifications (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL;
CREATE INDEX ix_notifications__inbox  ON notifications (user_id, audience, created_at DESC);
CREATE INDEX ix_notifications__unread ON notifications (user_id, audience) WHERE read_at IS NULL;

CREATE TABLE notification_deliveries (
  id                     uuid PRIMARY KEY,
  notification_id        uuid NOT NULL REFERENCES notifications(id),
  channel                text NOT NULL CHECK (channel IN ('SSE','WEB_PUSH','SMS','EMAIL','VOICE')),
  push_subscription_id   uuid,
  to_masked              text,                     -- '+91 98XXXXXX12'
  provider               text,
  provider_message_id    text,
  status                 text NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED','SENT','DELIVERED','FAILED','SKIPPED','EXPIRED')),
  attempts               smallint NOT NULL DEFAULT 0,
  last_error             text,
  cost_micro_inr         bigint,                   -- SMS/voice cost tracking (budget breaker, 12 AUTH-D07)
  queued_at              timestamptz NOT NULL DEFAULT now(),
  sent_at                timestamptz,
  delivered_at           timestamptz,
  created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_notification_deliveries__notification ON notification_deliveries (notification_id);
CREATE UNIQUE INDEX ux_notification_deliveries__provider_msg ON notification_deliveries (provider, provider_message_id)
  WHERE provider_message_id IS NOT NULL;     -- DLR webhook lookup

CREATE TABLE push_subscriptions (
  id                 uuid PRIMARY KEY,
  user_id            uuid NOT NULL REFERENCES users(id),
  device_id          uuid,                       -- ref: identity.devices
  audience           text NOT NULL CHECK (audience IN ('customer','restaurant','rider','admin')),   -- R14
  kind               text NOT NULL DEFAULT 'WEB_PUSH' CHECK (kind IN ('WEB_PUSH','FCM','APNS')),
  endpoint           text NOT NULL,
  p256dh             text,
  auth_secret        text,
  locale             text,
  failure_count      int NOT NULL DEFAULT 0,
  last_success_at    timestamptz,
  revoked_at         timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_push_subscriptions__endpoint ON push_subscriptions (endpoint) WHERE revoked_at IS NULL;
CREATE INDEX ix_push_subscriptions__user ON push_subscriptions (user_id, audience) WHERE revoked_at IS NULL;
```

---

## 11. DDL — admin, platform

```sql
CREATE TABLE approval_requests (                  -- maker-checker, limited to the five R31 action families
  id                 uuid PRIMARY KEY,
  city_id            uuid REFERENCES cities(id),
  action_type        text NOT NULL CHECK (action_type IN (
                        'REFUND','GOODWILL_COUPON','LEDGER_ADJUSTMENT','RIDER_CASH_ADJUSTMENT',   -- (1) money above threshold
                        'PAYOUT_BATCH',                                                         -- (2) payout batch release
                        'COMMISSION_CHANGE','FEE_CONFIG_CHANGE',                                -- (3) commission / fee config
                        'PAYOUT_ACCOUNT_CHANGE',                                                -- (4) payout bank/UPI details
                        'ROLE_GRANT')),                                                         -- (5) admin role grants
  target_type        text,                       -- 'order','payout_batch','restaurant',…
  target_id          uuid,
  payload            jsonb NOT NULL,             -- the exact command to execute
  payload_sha256     bytea NOT NULL,
  amount_paise       bigint,                     -- for threshold rules & dashboards
  maker_id           uuid NOT NULL REFERENCES users(id),
  maker_reason       text NOT NULL CHECK (char_length(maker_reason) >= 15),
  status             text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED','EXECUTED','FAILED')),
  checker_id         uuid REFERENCES users(id),
  checker_reason     text,
  decided_at         timestamptz,
  executed_at        timestamptz,
  execution_error    text,
  expires_at         timestamptz NOT NULL,       -- created_at + approvals.request_ttl_s (13 §5.1)
  -- break-glass (R31): maker self-approves when no checker is reachable; mandatory post-review within 24 h
  is_break_glass     boolean NOT NULL DEFAULT false,
  post_review_due_at timestamptz,
  post_reviewed_by   uuid REFERENCES users(id),
  post_reviewed_at   timestamptz,
  post_review_note   text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_approval_requests__four_eyes CHECK (checker_id IS NULL OR checker_id <> maker_id OR is_break_glass),
  CONSTRAINT ck_approval_requests__break_glass CHECK (NOT is_break_glass OR post_review_due_at IS NOT NULL),
  CONSTRAINT ck_approval_requests__post_review CHECK (post_reviewed_by IS NULL OR post_reviewed_by <> maker_id)
);
CREATE INDEX ix_approval_requests__queue ON approval_requests (city_id, action_type, created_at) WHERE status = 'PENDING';
CREATE INDEX ix_approval_requests__post_review ON approval_requests (post_review_due_at) WHERE is_break_glass AND post_reviewed_at IS NULL;
```

When approval is needed (threshold **values** are 13 §5.1 keys, R48):

| Family (R31) | Action types | Needs approval when |
|---|---|---|
| 1 | `REFUND` | amount > `approvals.refund_threshold_paise` (₹500) |
| 1 | `GOODWILL_COUPON` | value > `approvals.goodwill_threshold_paise` (₹150) |
| 1 | `LEDGER_ADJUSTMENT` (incl. `MG_TOPUP`, `PEAK_BONUS`), `RIDER_CASH_ADJUSTMENT` | amount > `approvals.adjustment_threshold_paise` |
| 2 | `PAYOUT_BATCH` | always |
| 3 | `COMMISSION_CHANGE`, `FEE_CONFIG_CHANGE` | always |
| 4 | `PAYOUT_ACCOUNT_CHANGE` | always |
| 5 | `ROLE_GRANT` | always |

Everything else (role revoke, TOTP reset, rider cash-limit override, coupon budgets, fraud unblock, PII export, DPDP erasure, tax rules) is **audit + step-up where applicable + the next-day review report** (C7). Go-live gate: ≥ 2 named people able to approve money actions (R31). An overdue break-glass post-review raises an ops alert.

```sql
CREATE TABLE reason_codes (                       -- single reason catalogue (ruling 11); referenced by *_reason_code columns
  code                   text PRIMARY KEY,        -- 'RESTAURANT_UNRESPONSIVE'
  category               text NOT NULL CHECK (category IN ('ORDER_CANCEL','ORDER_REJECT','DELIVERY_FAIL','OFFER_DECLINE',
                            'RIDER_RELEASE','REFUND','RESTAURANT_PAUSE','ZONE_PAUSE','TICKET_RESOLUTION','ACCOUNT_BLOCK')),
  allowed_actor_types    text[] NOT NULL,         -- {'CUSTOMER'} / {'ADMIN','SYSTEM'}
  default_fault_party    text CHECK (default_fault_party IN ('CUSTOMER','RESTAURANT','RIDER','PLATFORM','NONE')),
  requires_note          boolean NOT NULL DEFAULT false,
  label                  text NOT NULL,
  label_i18n             jsonb NOT NULL DEFAULT '{}',
  customer_message_i18n  jsonb NOT NULL DEFAULT '{}',   -- friendly text (04 §9.2)
  is_active              boolean NOT NULL DEFAULT true,
  sort_order             int NOT NULL DEFAULT 0
);
-- Seeded from code (13 §6.3 is the source list); app validates (category, actor) on every write.

CREATE TABLE feature_flags (
  key            text PRIMARY KEY,                -- e.g. 'ops_assisted_orders' (M8, P1)
  description    text NOT NULL,
  enabled        boolean NOT NULL DEFAULT false,
  rules          jsonb NOT NULL DEFAULT '{}',     -- {"cityIds":[…],"audiences":["partner"],"percent":10}
  owner          text,
  version        int NOT NULL DEFAULT 1,
  updated_by     uuid REFERENCES users(id),
  updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE app_config (                         -- typed (Go schema per key) runtime configuration
  id             uuid PRIMARY KEY,
  scope_type     text NOT NULL CHECK (scope_type IN ('GLOBAL','CITY','ZONE','RESTAURANT')),
  scope_id       uuid,                            -- NULL for GLOBAL
  key            text NOT NULL,                   -- 'ordering.accept_window_s', 'auth.otp', 'dispatch.offer_ttl_s'
  value          jsonb NOT NULL,
  version        int NOT NULL DEFAULT 1,
  updated_by     uuid REFERENCES users(id),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_app_config__key UNIQUE NULLS NOT DISTINCT (scope_type, scope_id, key),
  CONSTRAINT ck_app_config__scope CHECK ((scope_type = 'GLOBAL') = (scope_id IS NULL))
);

CREATE TABLE report_exports (
  id             uuid PRIMARY KEY,
  city_id        uuid REFERENCES cities(id),
  report_type    text NOT NULL,                   -- 'ORDERS','SETTLEMENT_STATEMENT','GST_SUMMARY','RIDER_EARNINGS','COD_CASH','REFUNDS',
                                                  -- 'GIG_WORKER_REGISTRATION' (M5: portal fields from riders, CSV) [LEGAL]
  params         jsonb NOT NULL,
  status         text NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED','RUNNING','SUCCEEDED','FAILED','EXPIRED')),
  file_id        uuid,                            -- ref: platform.file_objects
  row_count      int,
  contains_pii   boolean NOT NULL DEFAULT false,  -- PII exports need step-up + audit (no maker-checker, R31)
  requested_by   uuid NOT NULL REFERENCES users(id),
  error          text,
  created_at     timestamptz NOT NULL DEFAULT now(),
  completed_at   timestamptz,
  expires_at     timestamptz                      -- file deleted after 7 days
);

CREATE TABLE audit_logs (                         -- append-only (12 §5.7 columns); plain table (C8), no hash chain (C6)
  id               uuid PRIMARY KEY,
  occurred_at      timestamptz NOT NULL,          -- app clock
  actor_type       text NOT NULL CHECK (actor_type IN ('USER','ADMIN','SYSTEM','PROVIDER_WEBHOOK','CLI')),
  actor_id         uuid,
  actor_roles      text[] NOT NULL DEFAULT '{}',
  session_id       uuid,
  request_id       text,
  trace_id         text,
  ip               inet,
  user_agent_hash  bytea,
  city_id          uuid,
  action           text NOT NULL,                 -- 'restaurant.approve', 'order.cancel', 'kyc.view', 'approval.break_glass'
  resource_type    text NOT NULL,
  resource_id      uuid,
  outcome          text NOT NULL CHECK (outcome IN ('SUCCESS','DENIED','ERROR')),
  reason           text,
  approval_id      uuid,
  changes          jsonb,                         -- {"before":{…},"after":{…}} PII redacted to "[redacted]"/last-4
  retention_class  text NOT NULL DEFAULT 'SECURITY_1Y' CHECK (retention_class IN ('SECURITY_1Y','MONEY_8Y','OPS_2Y')),
  created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_audit_logs__resource ON audit_logs (resource_type, resource_id, occurred_at DESC);
CREATE INDEX ix_audit_logs__actor    ON audit_logs (actor_id, occurred_at DESC);
CREATE INDEX ix_audit_logs__action   ON audit_logs (action, occurred_at DESC);
CREATE INDEX ix_audit_logs__retention ON audit_logs (retention_class, occurred_at);   -- retention sweep
-- + trg_audit_logs_immutable (forbid_mutation); rovo_app has INSERT/SELECT only.
```

**Integrity (C6, RV-028):** append-only grants plus the trigger are the V1 control. Tamper evidence (an hourly job that seals new rows in a chain/Merkle hash written to the WORM bucket) is V1.1. High-volume auth telemetry (`auth.otp.requested/verified`) goes to the 180-day log archive (R36, M1), not to `audit_logs`. Only security-relevant auth outcomes (lockouts, step-up, admin login) are audited. The retention sweep deletes expired rows by class; it runs as `rovo_owner` through a `SECURITY DEFINER` function. That is the only sanctioned delete.

**Domain events (R42):** there is **no `outbox_events` table**. The emitting transaction calls River `InsertManyTx` with one job per subscriber; the job args carry the 13 §8 envelope and the `traceparent`. There is no relay, poller or fan-out job.

```sql
CREATE TABLE processed_events (                   -- handler idempotency (08 §4.1 rule 3)
  handler        text NOT NULL,
  event_id       uuid NOT NULL,
  processed_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (handler, event_id)
);
CREATE INDEX ix_processed_events__sweep ON processed_events (processed_at);   -- delete > 30 days

CREATE TABLE idempotency_keys (
  principal_id      text NOT NULL,                -- user id, or 'anon:<ip-hmac>'
  key               text NOT NULL CHECK (char_length(key) BETWEEN 8 AND 128),
  operation_id      text NOT NULL,                -- OpenAPI operationId, e.g. 'createOrder'
  request_hash      bytea NOT NULL,               -- sha256(method|route|canonical body)
  status            text NOT NULL CHECK (status IN ('IN_PROGRESS','COMPLETED')),
  locked_until      timestamptz,                  -- IN_PROGRESS lease (crash recovery)
  response_status   smallint,
  response_headers  jsonb,
  response_body     jsonb,
  resource_type     text,
  resource_id       uuid,
  created_at        timestamptz NOT NULL DEFAULT now(),
  completed_at      timestamptz,
  expires_at        timestamptz NOT NULL,         -- created_at + 24 h
  PRIMARY KEY (principal_id, key)
);
CREATE INDEX ix_idempotency_keys__expiry ON idempotency_keys (expires_at);

CREATE TABLE file_objects (                       -- provider-neutral object storage registry (DB-D13)
  id                 uuid PRIMARY KEY,
  bucket             text NOT NULL,               -- logical bucket name from config: 'media' | 'kyc' | 'exports' | 'invoices' | 'tickets'
  object_key         text NOT NULL,               -- random key, e.g. 'kyc/2026/10/0192…/3f9a…' — never contains PII
  purpose            text NOT NULL CHECK (purpose IN ('MENU_IMAGE','RESTAURANT_LOGO','RESTAURANT_COVER','KYC_DOC',
                                                     'TICKET_ATTACHMENT','COD_DEPOSIT_PROOF','INVOICE_PDF','REPORT_EXPORT',
                                                     'MENU_IMPORT_CSV','SETTLEMENT_FILE')),   -- M7 menu CSV; PA/bank files (§7.1)
  visibility         text NOT NULL CHECK (visibility IN ('PUBLIC_CDN','PRIVATE')),
  content_type       text NOT NULL CHECK (content_type IN ('image/jpeg','image/png','image/webp','application/pdf','text/csv')),
  size_bytes         bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 10485760),
  sha256             bytea,
  width_px           int,
  height_px          int,
  dominant_color     text CHECK (dominant_color ~ '^#[0-9a-f]{6}$'),   -- image placeholder (17)
  variants           jsonb NOT NULL DEFAULT '[]', -- [{"w":320,"key":"…_320.webp"},{"w":800,"key":"…_800.webp"}]
  encryption         text NOT NULL DEFAULT 'PROVIDER_SSE' CHECK (encryption IN ('PROVIDER_SSE','APP_ENVELOPE')),
                                                  -- PROVIDER_SSE = SSE-KMS; V1 uses it for every file incl. KYC (R38).
                                                  -- APP_ENVELOPE kept for a later decision; unused in V1
  dek_wrapped        bytea,                       -- APP_ENVELOPE only
  kek_id             text,
  owner_user_id      uuid REFERENCES users(id),
  status             text NOT NULL DEFAULT 'PENDING_UPLOAD' CHECK (status IN ('PENDING_UPLOAD','QUARANTINE','READY','REJECTED','DELETED')),
  scan_result        text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  deleted_at         timestamptz,
  CONSTRAINT ux_file_objects__key UNIQUE (bucket, object_key),
  CONSTRAINT ck_file_objects__envelope CHECK ((encryption = 'APP_ENVELOPE') = (dek_wrapped IS NOT NULL AND kek_id IS NOT NULL)),
  CONSTRAINT ck_file_objects__kyc_private CHECK (purpose <> 'KYC_DOC'
    OR (visibility = 'PRIVATE' AND content_type IN ('image/jpeg','image/png','image/webp')))   -- R38: images only, server re-encoded
);
CREATE INDEX ix_file_objects__pending ON file_objects (created_at) WHERE status IN ('PENDING_UPLOAD','QUARANTINE');
```

Logical bucket names map to physical buckets per environment through config, for example `media → rovo-prod-media` on S3/GCS and `rovo-media` on local MinIO. Public URLs are `${MEDIA_CDN_BASE_URL}/${object_key}`, built at response time. Private objects are streamed through the API, or served by a ≤ 60 s presigned GET (12 §6.2). No URL is ever stored.

---

## 12. PII classification (DPDP Act 2023 & encryption decisions)

Legend:
- **Protection:** `AC` = plaintext, protected by DB access control and encryption at rest (managed-service storage encryption with KMS); `ENC` = app-layer AES-256-GCM envelope encryption (DEK wrapped by a cloud KMS key, `pii_key_id`); `HASH` = one-way; `MASK` = only a masked form stored; `NS` = not stored.
- **Sensitivity:** H/M/L.
- [LEGAL] DPDP Rules 2025 obligations (notice, consent, retention, breach notification) to be confirmed by counsel.

| Data | Where | Sens. | Protection | Purpose / note |
|---|---|---|---|---|
| Phone number | `users.phone_e164`, `otp_challenges.phone_e164` | M | AC (needed for OTP lookup). Logs carry an HMAC only (12 §5.7). | Login, order contact. Riders see it only during an active delivery (12 §9). |
| Contact phone override | `customer_addresses.contact_phone_e164`, order snapshot | M | AC | Delivery handover |
| Name | `users.full_name`, address/order snapshots | L–M | AC | |
| Email (staff, optional members) | `users.email` | M | AC | |
| Precise home/work location | `customer_addresses.location`, `orders.drop_location`, snapshot | **H** (reveals residence) | AC + redaction after 180 d (§13) | Delivery. Never in analytics exports without coarsening to locality. |
| Address text, landmark | `customer_addresses.*`, order snapshot | H | AC + redaction after 180 d | |
| Rider live location & trail | `rider_availability.last_location`, `rider_location_pings` | **H** | AC, 30-day retention, access by `ADMIN_OPS` in scope only | Dispatch, dispute evidence |
| Date of birth / age | customers: not stored (`users.adult_declared_at` only). Riders: `riders.date_of_birth` | M | AC | Riders only, for gig-worker registration (M5) [LEGAL] |
| Gender, residence state/district, portal worker id | `riders.gender`, `riders.residential_*`, `riders.ss_portal_id_enc` | M–H | AC; portal id ENC + `_last4` | Gig-worker registration export (M5) [LEGAL] |
| Delivery handover code | `orders.delivery_code_enc` | L–M | ENC | R39; shown to the customer only |
| Lead / waitlist contact | `leads.phone_e164`, `waitlist.phone_e164`, `waitlist.location` | M | AC | Sales follow-up / expansion; consent notice recorded |
| Staff invite phone | `staff_invites.phone_hmac`, `_last4` | L | HASH | |
| SOS location and note | `sos_events` | H | AC; ops in scope only | Rider safety |
| Manual refund UPI VPA | `refunds.payee_vpa_enc` | H | ENC + masked | R29 COD compensation |
| Aadhaar number | **never stored** | — | NS (only a masked Aadhaar image, encrypted) | 00 §3, 12 §6.1 |
| PAN | `restaurants.pan_enc`, `*_kyc_documents.doc_number_enc` | H | ENC + `_last4` | TDS / KYC [LEGAL] |
| Driving licence number | `riders.dl_number_enc` | H | ENC + `_last4` | KYC |
| KYC document images | `file_objects` (bucket `kyc`) | **H** | SSE-KMS + private bucket + audited streaming view + ≤ 60 s signed URLs; images only (R38) | 12 AUTH-D12 |
| Bank account number, UPI VPA | `payout_accounts.*_enc` | **H** | ENC + `_last4` / masked | Payouts; full reveal by Finance only, with step-up |
| IFSC | `payout_accounts.ifsc` | L | AC | Public bank code |
| FSSAI no., GSTIN | `restaurants` | L (public business identifiers) | AC | Displayed to customers [LEGAL] |
| Card/UPI payment details | PA only; `payment_attempts.vpa_masked`, `card_last4` | M | MASK | Never full PAN/VPA (PCI scope avoided) |
| TOTP secret | `admin_credentials.totp_secret_enc` | **H** (auth secret) | ENC | |
| Passwords, recovery codes | `admin_credentials.password_hash`, `admin_recovery_codes` | H | HASH (argon2id) | |
| OTP code | `otp_challenges.code_hmac` | H | HASH (HMAC + pepper) | 5 min TTL |
| Refresh tokens | `refresh_tokens.token_sha256` | H | HASH | |
| IP address, user agent | `sessions`, `audit_logs`, `user_consents` | M | AC (UA hashed in audit) | Security, consent proof |
| Device identifiers | `devices.install_id` | M | AC | Fraud, session list |
| Order history & food preferences | `orders`, `order_items` | M (can imply religion/diet) | AC; analytics use aggregates | |
| Ratings, review text | `ratings`, `reviews` | L–M | AC; reviews moderated for embedded phone numbers/names | |
| Support messages & attachments | `ticket_messages`, `file_objects` | M–H (free text) | AC; private bucket | |
| Push endpoints | `push_subscriptions` | L | AC | |
| Webhook payloads | `payment_events.payload` (redacted), raw body in `webhooks-raw/` | M | AC; contact/email/VPA stripped at ingest; raw body 180 days (RV-030) | Disputes |

**Encryption key handling:** KEKs live in the cloud KMS (AWS KMS / Cloud KMS / Key Vault). The app holds only wrapped DEKs. Column DEKs are per table and per key version. Rotation creates a new `pii_key_id` and re-wraps lazily, with a background re-encrypt job. Blind indexes (`*_hmac`) are not needed in V1, because no encrypted column is searched by value. Admins search by `_last4` plus another attribute.

---

## 13. Data retention policy (per table)

Proposed defaults, **all [LEGAL] pending counsel/CA review**. Basis: DPDP purpose limitation; GST record keeping (books retained for at least 72 months from the due date of the annual return, commonly operationalised as 8 years [LEGAL — verify]); Income-tax/Companies Act books (8 years) [LEGAL]; CERT-In logs (180 days rolling, in India); DPDP Rules log retention for breach investigation (≥ 1 year, per 12 §5.7). Enforcement is by the catch-up job `retention.sweep` (hourly trigger, per-table watermark; 13 §5.2), which does batched `DELETE`s of 1–5k rows per transaction by an indexed timestamp. There are no partition drops (C8).

| Table(s) | Retain | Then |
|---|---|---|
| `otp_challenges` | 30 days | hard delete |
| `sessions`, `refresh_tokens` | until expiry/revocation + 90 days | hard delete |
| `idempotency_keys` | 24 h (`expires_at`) | hard delete |
| `quotes` | 24 h | hard delete |
| `processed_events` | 30 days | hard delete |
| `rate_limit_buckets` | `expires_at` | hard delete |
| `staff_invites` | 30 days after `expires_at`/acceptance | hard delete |
| `rider_location_pings` | 30 days | batched delete (BRIN on `recorded_at`) |
| `contact_tap_log` | 1 year [ASSUMPTION] (contact-policy disputes) | delete |
| `sos_events` | 3 years [ASSUMPTION] (safety/insurance claims) [LEGAL] | delete |
| `leads` | 1 year after last update if not converted | delete |
| `waitlist` | until notified + 90 days, or unsubscribe | delete |
| `erasure_requests` | 8 years (proof of compliance; no PII) [LEGAL] | delete |
| `transfers`, `pa_settlements`, `pa_settlement_lines`, `recon_exceptions` | 8 years [LEGAL] | archive |
| `rider_availability.last_location` | current only | overwritten; cleared when going offline > 24 h |
| `customer_addresses` | until the user deletes it / account erasure | hard delete |
| `orders.delivery_address_snapshot`, `orders.drop_location`, order contact phone | 180 days after `closed_at` (dispute window) | **redact**: keep locality, city, PIN and state (place of supply); null the house/landmark/contact; snap the point to the locality centroid |
| `orders`, `order_items`, `order_charges`, `order_status_history` | 8 years [LEGAL] | delete or archive |
| `payments`, `payment_attempts`, `payment_events`, `refunds` | 8 years [LEGAL]; `payment_events` raw body 180 days (object lifecycle) | delete or archive |
| `ledger_*`, `payouts`, `payout_items`, `cod_deposits`, `invoices` | 8 years [LEGAL] | archive |
| `deliveries`, `delivery_offers`, `delivery_status_history` | 3 years [ASSUMPTION] (rider earnings disputes) | delete; earnings live in the ledger |
| KYC documents (`*_kyc_documents` + objects) | partnership + 8 years for approved partners [LEGAL]; **90 days after decision** for rejected/abandoned applications (12 §6.2) | delete object + row |
| `payout_accounts` | partnership + 8 years | delete |
| `support_tickets`, `ticket_messages`, attachments | 3 years after closure [ASSUMPTION]; money-linked tickets 8 years | delete |
| `ratings`, `reviews` | life of the restaurant listing; anonymised on user erasure | |
| `notifications` | 90 days | delete |
| `notification_deliveries` | 180 days (DLT/SMS dispute window) | delete |
| `audit_logs` | `SECURITY_1Y` 1 year, `OPS_2Y` 2 years, `MONEY_8Y` 8 years | batched delete per class (DB-D15) |
| `user_consents` | life of account + 8 years (proof of consent) [LEGAL] | |
| `report_exports` + files | 7 days | delete |
| `users` (erasure request) | anonymise within 30 days [LEGAL]: null phone/email/name, `status='ANONYMIZED'` | rows referenced by financial records are kept (legal-obligation basis) |

---

### 13.1 Erasure map (DPDP, M15, RV-031) [LEGAL]

Applied by `erasure.execute` for an `erasure_requests` row once no hold applies (open order, open payout, rider cash in hand, dispute, legal hold). Each applied row is appended to `erasure_requests.steps_completed`, so a restore can **replay** it (23 §9). **D** = delete, **A** = anonymise in place, **R** = retain (legal basis noted).

| Table / bucket | Action | Detail / basis |
|---|---|---|
| `users` | A | null phone/email/name, `status='ANONYMIZED'`, `anonymized_at`; row kept as an FK target |
| `customer_profiles`, `customer_addresses`, `push_subscriptions`, `devices`, `sessions`, `refresh_tokens`, `user_roles` (member roles) | D | convenience/security data |
| `user_consents` | R | proof of consent (DPDP) for life + 8 y; links to an anonymised user |
| `otp_challenges`, `rate_limit_buckets`, `staff_invites` | D | (also TTL) |
| `orders.delivery_address_snapshot`, `drop_location`, `customer_note`, `delivery_instructions`, `delivery_code_enc` | A | keep locality/city/PIN/state (place of supply); snap the point to the locality centroid |
| `orders`, `order_items`, `order_charges`, `order_status_history` | R | tax/accounting records, 8 y [LEGAL]; the user is already anonymised |
| `payments`, `payment_attempts`, `refunds` (`payee_vpa_enc` → null), `payment_events` (redacted) | R/A | 8 y money records; the manual-refund VPA is nulled once paid |
| `coupons` (goodwill owned), `coupon_redemptions` | R | money records |
| `ratings` | A | keep the score, drop the rater link |
| `reviews` | D | free text may contain PII |
| `support_tickets` | A | requester link nulled; money-linked tickets kept 8 y |
| `ticket_messages` (requester-authored), ticket attachments | D | free text; attachment objects deleted (`tickets` bucket) |
| `notifications`, `notification_deliveries` | D | |
| `waitlist`, `leads` (where phone matches) | D | |
| `audit_logs` | R | security/legal obligation; `changes` already redacts PII (12 §5.7) |
| River job args (in-flight) | — | events carry ids only, never free-text PII (R42 envelope rule) |
| **Riders:** `riders` | A | name/DOB/gender/residence/DL/portal id nulled; row kept for ledger links. **R** while the gig-worker registration obligation applies [LEGAL] |
| `rider_kyc_documents` + `kyc` bucket objects | D / R | rejected: delete after 90 d; approved: partnership + 8 y [LEGAL] |
| `payout_accounts` | R | 8 y money records (encrypted) |
| `rider_location_pings`, `contact_tap_log`, `sos_events` | D / R | pings and taps deleted; SOS kept for its retention (safety claims) [LEGAL] |
| **Restaurant owners/staff:** `restaurant_users`, `restaurant_devices` | D | business entity data in `restaurants` is not personal data, except the owner contact, which is anonymised |
| Buckets `exports` (report files) | D | 7-day TTL already; regenerated reports exclude anonymised users |
| Backups / PITR | R → expire | not edited; restore replays `erasure_requests` before serving traffic (23 §9) |

## 14. Retention jobs and partitioning triggers (C8)

**No table is partitioned in V1** (DB-D11). Every high-volume table has an index usable by `retention.sweep` (a BRIN or B-tree on its timestamp) and is deleted in batches by the catch-up job (13 §5.2). The job records a per-table watermark, so a missed run catches up.

| Table | Expected V1 volume | Partition when | Strategy then |
|---|---|---|---|
| `rider_location_pings` | ~30–170k rows/day | > ~10 M rows, or the sweep can't keep up | RANGE daily on `recorded_at` |
| `audit_logs` | low | > ~10 M rows | RANGE monthly on `occurred_at` |
| `order_status_history`, `delivery_status_history`, `delivery_offers`, `notification_deliveries`, `payment_events`, `ledger_postings` | ~8 rows/order | > ~10 M rows or > 20 GB | RANGE monthly |
| `orders` | — | **never at V1 scale** | Point lookups by id; partitioning would force `created_at` into every unique key (`code`, `quote_id`) |

Rules if and when a table is partitioned: PK/unique constraints must include the partition key; no FKs into partitioned tables; keep a `DEFAULT` partition plus an alert when it is non-empty; create partitions from a catch-up job, never with a plain periodic tick (RV-004).

Conversion recipe (expand/contract): create the new partitioned table `x_p`, then dual-write via trigger, then backfill in batches, then swap names in a short lock, then drop the old table.

## 15. Seed data plan — Mahabubnagar

Seeds are split by purpose. **Reference seeds** are idempotent `INSERT … ON CONFLICT DO NOTHING` goose migrations that run in every environment. **Demo seeds** are a `rovo seed demo` CLI that is refused when `ENV=production`.

### 15.1 Reference seeds (all environments)

| Seed | Content |
|---|---|
| City | `code='MBNR'`, `slug='mahabubnagar'`, `name='Mahabubnagar'`, `name_i18n={"te":"మహబూబ్‌నగర్"}` [ASSUMPTION — Telugu spelling to be verified by a native speaker], `state_name='Telangana'`, `gst_state_code='36'`, `timezone='Asia/Kolkata'`, `centroid=POINT(78.0035 16.7488)` (Wikipedia: 16°44′56″N 78°00′13″E, urban area ~98.6 km², 2011 population 222,573; [Wikipedia](https://en.wikipedia.org/wiki/Mahbubnagar), accessed 2026-10-04), `status='PLANNED'` until launch |
| Zone | **One** zone `MBNR-CORE` from an ops-drawn polygon (16 §8). Until ops draws it, staging/dev use the **test octagon** (16 §10.1). It is tagged `[ASSUMPTION — fixture, not the real service area]` and never seeded to production. |
| Localities | ~15–25 named areas with centroids and aliases, **every one tagged [ASSUMPTION – verify with local ops]** except the census-listed ones. Verified as part of the urban agglomeration (Wikipedia, accessed 2026-10-04): **Boyapalle** (census town), **Yenugonda** (census town). Candidate list to verify on the ground: New Town, Old Town, Padmavathi Colony, Christian Colony/Christianpally, Bandameedipally, Shashab Gutta, Rajendra Nagar, Bhagiratha Colony, Srinivasa Colony, Metugadda, Veerannapet, Ramaiah Bowli, Habeeb Nagar, Laxmi Nagar Colony, Teachers Colony, Pillalamarri Road area, Bus Stand area, Clock Tower area. PIN `509001` verified (Wikipedia). Other PINs (e.g. `509002`) [ASSUMPTION]. Centroids are captured by ops on the map, not guessed in code. |
| Cuisines | BIRYANI, SOUTH_INDIAN, TIFFINS (idli/dosa), NORTH_INDIAN, INDO_CHINESE, ANDHRA_TELANGANA_MEALS, FAST_FOOD, PIZZA, BAKERY, DESSERTS, ICE_CREAM, JUICES_SHAKES, TEA_COFFEE, KEBABS_GRILL, SEAFOOD (en + te labels) |
| Fee config v1 | City default with the values of **16 §6.5** (R48): slabs on road metres `[0,2000)` ₹20 · `[2000,4000)` ₹30 · `[4000,6000)` ₹40 · `[6000,8000)` ₹50 · `[8000,10000)` ₹60 (R18); `max_serviceable_radius_m = 7000` straight-line; platform fee, small-cart fee, road factor, COD caps, rider pay, cash limit and ETA params as listed there. `status='DRAFT'` until Finance approves (maker-checker, R31 family 3). The seed runs the same slab validator as the API (16 §6.1). |
| Tax rules | §5.3 table, `status` gated by CA sign-off [LEGAL] |
| Reason codes | full catalogue from 13 §6.3 |
| Notification templates | en + te per 15; DLT template ids filled after registration [LEGAL] |
| App config: timers & thresholds | Every key in **13 §5.1** with the default listed there (13 owns the values, R48), including `dispatch.delivery_code_enabled = true` and `dispatch.delivery_code_min_payable_paise = 30000`, i.e. **delivery OTP on for prepaid orders ≥ ₹300, off for COD (R39)**, plus the `approvals.*` thresholds and the R34 tiers (`dispatch.location_fresh_s = 180`, `rider.auto_offline_after_s = 900`). |
| App config: operational (owned **here**, R53) | See §15.3. |
| Feature flags | `ops_assisted_orders` (M8, P1) off; `split_settlement` off until the legal opinion (R35); `manual_zone_surge` **absent** (V1.1, R30) |
| Ledger accounts | Platform accounts for MBNR with the 14 §10.2 codes (§9), `normal_side` per 14, and `GST_*` accounts with `gstin_state='36'`. Owner accounts are created lazily on restaurant/rider approval. |
| Admin bootstrap | **Not a seed.** It is created by `rovo admin bootstrap --email` (one-time setup link, 12 §3.6). There are no default passwords anywhere. |

### 15.2 Demo seeds (local/dev/staging only)

The demo set is 8 fictional restaurants ("Demo Biryani House", …) with realistic Telugu/English menus (veg/non-veg/egg, variants Half/Full, addon groups), 4 riders, 5 customers with addresses inside the fixture zone, coupons `WELCOME50` (percent) and `FLAT30`, and one historical delivered order per restaurant. All phones are in a reserved fake range (`+9199999000xx`), which the fake OTP provider accepts with code `000000` in non-prod only. Names and brands are invented. No real businesses are used.


### 15.3 Operational `app_config` defaults owned by this doc (R53)

Payout days, cash ageing and similar operational values are owned here (R53). Docs 01, 07, 13, 14 and 19 reference the **keys**. Timer/threshold keys are in 13 §5.1, and fee/commission values in 16 §6.5.

| Key (scope CITY) | Default | Meaning |
|---|---|---|
| `payouts.rider.schedule` | `{"frequency":"WEEKLY","runDay":"MON","period":"PREV_MON_SUN"}` | Rider payout run every **Monday** for the previous Mon 00:00 – Sun 23:59:59 (city tz) |
| `payouts.restaurant.schedule` | `{"frequency":"WEEKLY","runDay":"TUE","period":"PREV_MON_SUN"}` | Restaurant settlement every **Tuesday** for the previous Mon–Sun |
| `payouts.missed_run_alert_local_time` | `"12:00"` | catch-up alert if the scheduled day's batch is missing by this local time (13 §5.2) |
| `cod.cash_ageing_alert_h` | 24 | rider cash held > 24 h → alert rider + ops |
| `cod.cash_ageing_block_h` | 48 | cash held > 48 h → **no new COD offers** to the rider until a deposit is verified |
| `cod.service_window` | `"SERVICE_HOURS"` | COD offered during the city's service hours (register row 72) |
| `support.phone_e164` | `[OPEN — M9]` | shown in `/config/client` |
| `privacy.erasure_due_days` | 30 [LEGAL] | `erasure_requests.due_at` |
| `retention.*` | §13 values | per-table retention used by `retention.sweep` |

The settlement catch-up jobs (13 §5.2) read `payouts.*.schedule`, so the run day is defined only here.

---

## 16. Migration strategy (goose, zero-downtime)

**Tooling:**
- `goose` SQL migrations live in `migrations/` (timestamp-versioned: `20261004120000_create_orders.sql`).
- River's schema is applied by `rivermigrate` in the same `rovo migrate` job, ordered before app migrations that reference it (no FKs into River tables).
- Migrations run as a **one-off job/task** (ECS run-task / Cloud Run job / K8s Job) with `rovo_owner` credentials **before** the new app version rolls out. App containers never migrate on boot.

**Expand / contract rule:** every release must work with the schema **before and after** its migration (N-1 compatibility), because the old version keeps serving during a rolling deploy.

| Change | Safe recipe |
|---|---|
| Add column | `ADD COLUMN … NULL` or with a constant `DEFAULT` (metadata-only since PG 11). Backfill in batches (River job, 1–5k rows per tx), then add `NOT NULL` via `CHECK (col IS NOT NULL) NOT VALID` → `VALIDATE` → `SET NOT NULL` (PG uses the validated check). |
| Add index | `CREATE INDEX CONCURRENTLY` in its own migration with `-- +goose NO TRANSACTION`. Check `pg_index.indisvalid` afterwards. |
| Add FK / CHECK | `ADD CONSTRAINT … NOT VALID`, then `VALIDATE CONSTRAINT` in a later migration (only `SHARE UPDATE EXCLUSIVE`). |
| Add enum value | Replace the `CHECK` with a wider one (`NOT VALID` + validate). Deploy code that understands the value **before** any code writes it. |
| Rename column/table | Never in place. Add new → dual-write (app) → backfill → switch reads → stop writing old → drop old (≥ 1 release later). |
| Change type | Same as rename (new column). |
| Drop column | Release N stops reading/writing (sqlc queries updated). Release N+1 drops it. |
| Large backfill | River job with checkpointing. Never inside the migration. |
| Partitioning (only past ~10 M rows, §14) | Expand/contract recipe; partitions created by a catch-up job, never by migrations at runtime. |

**Guardrails in every migration session:** `SET lock_timeout = '3s'; SET statement_timeout = '60s';` and retry with backoff on lock timeout. CI runs:
- (1) all migrations from empty on PostGIS images for both PG 17 and 18;
- (2) `squawk` (or an equivalent linter) to flag unsafe DDL `[ASSUMPTION — tool choice to 21]`;
- (3) a down-migration test for the last migration only (down migrations are for dev; production rolls forward);
- (4) `sqlc vet`/compile against the migrated schema.

**Managed-service notes (§4a):** `CREATE EXTENSION` needs the provider's admin role, so it runs once by IaC/bootstrap. Extension versions are pinned in IaC. PostGIS minor upgrades (`ALTER EXTENSION postgis UPDATE`) are scheduled maintenance tasks, tested on staging first.

---

## 17. Multi-city considerations

- **`city_id` on every city-scoped table:** zones, localities, restaurants, riders, orders, payments, refunds, deliveries, quotes, coupons, fee_configs, commission_plans, ledger_accounts and journals, payouts, cod_deposits, support_tickets, report_exports, user_roles (admin scope).
- **Uniqueness is city-scoped where humans pick names:** `(city_id, slug)` for restaurants and localities, `(city_id, code)` for zones. Coupon codes are global, to avoid ambiguity across cities.
- **Time:** all local-time logic (opening hours, accounting dates, daily reports, payout cut-offs) uses `cities.timezone`. India has one zone, but nothing hard-codes `Asia/Kolkata`.
- **Tax:** `cities.gst_state_code` drives place of supply and the CGST/SGST split. A second state needs its own GST registration and invoice series ([LEGAL]); `invoice_sequences.series` is per city.
- **Locales:** `cities.supported_locales` (`{en,te}` for MBNR; a Karnataka city might add `kn`). `*_i18n` JSONB needs no migration.
- **Config:** `fee_configs`, `app_config (scope CITY/ZONE)`, `feature_flags.rules.cityIds` and `tax_rules` are already per city.
- **Ledger:** per-city platform accounts give city P&L and simplify per-state GST.
- **Not needed for multi-city:** sharding, per-city schemas, cross-city orders (a restaurant serves only zones of its own city).
- **Scale path:** a single Postgres comfortably holds several Indian tier-2/3 cities at V1 volumes. Add read replicas for reporting first. City-based partitioning or sharding only beyond ~10⁵ orders/day [ASSUMPTION].

---

## 18. What this schema deliberately does not do (V1)

| Not doing | Why / upgrade path |
|---|---|
| Brand/chain table | DB-D12. Expand-only later. |
| Server-side cart | Ruling 12. Expand-only `carts` later. |
| Wallet / stored value | Ruling 9 (goodwill = coupons). A wallet raises RBI PPI questions [LEGAL]. |
| Multi-dimension variants (size × crust) | Single variant dimension + addon groups cover the local menus. |
| Scheduled / pre-orders | Status model supports it (`ACCEPTED` without `PREPARING`). Add `orders.scheduled_for` later. |
| Batching (2 orders per rider) | `ux_deliveries__rider_active` and `active_delivery_count ≤ 1` are the only blockers. Both are documented and droppable. |
| Postgres RLS | 12 §8. |
| `h3-pg`, `pg_partman`, `pg_cron` | Portability (DB-D01). H3-style hex analytics can be done in Go (`uber/h3-go`) if ever needed. |
| Event sourcing / event table | History tables + audit give the trail; events are River job args (R42). |
| Surge / rain fee | R30, C1: V1.1 at the earliest. Rider peak bonus = ledger `ADJUSTMENT` (`PEAK_BONUS`). |
| Rider shifts / `ON_BREAK` | C12: V2+. |
| Day-one partitioning, hash-chained audit | C8, C6 (§14, DB-D15). |
| Coupon `SHARED` funding, cuisine/user targets, per-day budgets, bulk codes | C16. |
| Review moderation queue and replies | C14 (profanity filter + admin hide only). |
| Separate analytics DB/warehouse | Read replica + `*_report` views (08). |

---

## 19. Open items

- `[OPEN]` PG 17 vs 18 on the chosen managed provider (PostGIS 3.5/3.6 availability). The schema works on both.
- `[OPEN — Finance]` Commission basis: include packaging or not (`commission_plans.basis`).
- `[OPEN — CA]` Inclusive vs exclusive GST on fees (ruling 8). The schema supports both via `tax_rules.price_inclusive` + `order_charges.is_included`.
- `[OPEN — CA]` Whether the discount reduces the GST base for platform-funded vs restaurant-funded coupons.
- `[OPEN]` Rider cancellation compensation % (`fee_configs.rider_cancel_comp_bps`).
- `[OPEN — Legal]` All retention periods in §13, DPDP erasure vs. 8-year tax retention, and labour-code record keeping for riders.
- `[OPEN — Ops]` Real zone polygon and locality list for Mahabubnagar (16 §10).
- ~~Delivery handover code default~~ — resolved by R39 (on for prepaid ≥ ₹300; 13 SM-D13).
- `[OPEN — Legal]` Final gig-worker portal field list and export format (M5); applicability thresholds for a small aggregator.
- `[OPEN — Legal]` Erasure map actions in §13.1 (especially rider records vs gig-worker registration, and SOS retention).
- `[OPEN — Finance]` Whether issued-but-unredeemed goodwill coupons need an accrual account (14 §10.2 has none).

## 20. Sources (accessed 2026-10-04)

- PostgreSQL 18 `uuidv7()` / `uuid_extract_timestamp()`: https://www.postgresql.org/docs/18/functions-uuid.html
- PostGIS `ST_Covers` (geography polygon/point support; index use): https://postgis.net/docs/ST_Covers.html
- River features (InsertTx, unique/periodic/scheduled jobs; Pro-only features such as workflows and durable periodic jobs): https://riverqueue.com/docs
- Mahabubnagar coordinates, area, population, PIN 509001, census towns: https://en.wikipedia.org/wiki/Mahbubnagar
- GST 18% on local delivery through ECO under §9(5) from 22 Sep 2025: https://a2ztaxcorp.net/gst-alert-local-delivery-services-to-attract-18-tax-from-september-22-says-cbic/ and https://static.pib.gov.in/WriteReadData/specificdocs/documents/2025/sep/doc2025921642801.pdf (**secondary summaries; CA to confirm against the notification text**)
- Managed-service extension availability: not re-verified in this document → `[ASSUMPTION]`, owned by DevOps (22/25).
