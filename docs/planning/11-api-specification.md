# 11 — API Specification (REST, OpenAPI 3.1)

| | |
|---|---|
| **Purpose** | The HTTP contract for all rovo clients: conventions (paths, casing, pagination, errors, idempotency, concurrency, rate limits, i18n, money, time, auth, versioning), the full endpoint catalogue by audience, real-time SSE and push, webhooks, and OpenAPI 3.1 excerpts for the six most critical operations. |
| **Owner** | Backend Architect |
| **Status** | Draft v1.1 — reconciled with review (31) and rulings R1–R48, 2026-10-04 |
| **Depends on** | 00 baseline (P2 spec-first, P3 SSE, P9 auth, P10 payments); 08 system architecture (idempotency, CAS, SSE via `NOTIFY`); 10 database schema; 12 auth/RBAC (hosts, cookies, CSRF, permissions, maker-checker); 13 state machines (commands, errors, events); 14 payment architecture (PA specifics); 15 notifications; 16 geography (serviceability, fees, ETA); 17/18 frontend & PWA (client expectations) |
| **Consumed by** | Go server codegen (`oapi-codegen` strict server [ASSUMPTION — 26 decides]), TS client (`openapi-typescript` + `openapi-fetch`, 17), 20 testing (contract tests), 19 threat model |

**Changes in v1.1**
- **R14 / R27 / RV-001 / RV-025 (register rows 31–33):** four app hosts (`app.`, `restaurant.`, `rider.`, `admin.`), each serving `/api/v1/*` **same-origin** through the CDN, with no CORS. The audience is derived from `Host` plus the CDN origin-verify secret; a client-sent `X-Rovo-Audience` is ignored. `api.` is reserved for future native bearer clients and carries only PA/SMS webhooks in V1.
- **R30 / C1 / C15 / C20:** removed surge endpoints and fields (`SURGE_FEE`, `SURGE_CHANGED`, `zone.surge`, rider `surge` earnings), zone import/export, `geo/live`, broadcasts, and tax-rule writes. **C2:** OTP over SMS only. **C12:** no `ON_BREAK`. **C13:** partner analytics today/this week only. **C16:** coupon `SHARED` funding and cuisine/user targets removed.
- **R31 / C7:** ⚖ (maker-checker) kept only on the five action families. Removed it from user block/unblock, TOTP reset, role revoke, cash-limit override, coupon budgets, PII reports and erasure.
- **R16:** offer status `REVOKED` (`DeliveryOfferStatus`). **R29:** COD compensation choice + manual refund `mark-paid` with UTR. **R39:** `Order.deliveryCode`. **R40:** no restaurant self-cancel after accept (issues endpoint only). **R44 / M10:** device registration endpoints for device-bound sessions.
- **M4 / M5 / M7 / M8 and others:** new endpoints for waitlist, restaurant leads, staff invites, SOS, contact taps, recon exceptions / PA settlements / bank-statement import, erasure requests, ledger adjustments (`MG_TOPUP`, `PEAK_BONUS`), gig-worker export, ops bulk menu CSV import (P0 for Gate B), ops-assisted phone orders (P1), and the order `ack` endpoint (row 64).
- **Row 29 / RV-011 / R52:** SSE keeps one rule. The stream outlives the access token while the session is valid, the server closes at 30 min, `revoked` is sent on revocation, and the heartbeat is 20 s. **R27:** batched rider pings (60 s); restaurant heartbeat 60 s, with SSE presence counting.
- **R21:** rate-limit counters in Postgres (`rate_limit_buckets`). **Rows 57/58:** quantity 1–20, prep time 5–90. **R48:** timer/threshold values are cited by key (13 §5.1), fee values by 16 §6.5.
- §2.9 operation counts updated.

The **single source of truth** is `api/openapi.yaml` in Phase 2 (spec-first, P2). This document fixes its conventions and content. Paths below omit the `/api/v1` prefix in tables unless shown.

---

## 0. Decision summary

| ID | Decision |
|---|---|
| API-D01 | Base path **`/api/v1`** on every host (R15, R27). The four app hosts (`app.`, `restaurant.`, `rider.`, `admin.`; R14) route `/api/*` **same-origin through CloudFront** to the API, with cookie auth (12 AUTH-D03) and **no CORS**. `api.<domain>` is reserved for future native bearer clients; in V1 it serves only `/api/v1/webhooks/*` (R27). One router, one spec. |
| API-D02 | **JSON bodies and query params in `camelCase`**. Enum values in `UPPER_SNAKE` (identical to DB). Path segments in `kebab-case`. |
| API-D03 | **Resources + command sub-resources.** CRUD uses REST verbs. State-machine transitions are explicit `POST …/{id}/{command}` endpoints (`/accept`, `/cancel`, `/picked-up`). There is no generic "PATCH status". Each command maps 1:1 to a transition ID in 13. |
| API-D04 | **Money** is always `{"amountPaise": 24900, "currency": "INR"}`. Integers only. Clients format with `Intl` (`en-IN`/`te-IN`). |
| API-D05 | **Errors:** RFC 9457 `application/problem+json` with a stable `code`, `fieldErrors[]` (the shape 17 expects) and `traceId`. |
| API-D06 | **Pagination:** opaque cursor (`?cursor=&limit=`), response `{items, nextCursor}`. No offset pagination on unbounded lists. |
| API-D07 | **Idempotency-Key** is required on every POST that creates or transitions something (orders, payments, refunds, all state commands, offers, milestones, payouts, adjustments, imports). Optional on other POSTs. |
| API-D08 | **Optimistic concurrency:** `ETag: "v<version>"` on aggregate reads. `If-Match` is **required** on PUT/PATCH of mutable aggregates (menu items, restaurant profile, zones, fee configs, coupons) and **optional** on state commands (13 §7.1). |
| API-D09 | **Cart is client-side; `POST /cart/quote` is stateless for the client** and returns a signed short-TTL `quoteId`. `POST /orders` takes `quoteId` + `Idempotency-Key`. A stale quote returns `409` with a new quote and diff (Lead ruling 12; 10 §5.1). |
| API-D10 | **Real-time:** one multiplexed SSE stream `GET /stream?topics=` per tab. Thin events. No replay buffer. Heartbeat every **20 s** (R10, R52). The stream continues past access-token expiry while the server-side session is valid; the server closes it at 30 min for rebalancing, and immediately with `event: revoked` on session revocation (RV-011). Clients refetch snapshots on (re)connect (R10; 08 §6; 12 §4.7). |
| API-D11 | **Maker-checker actions** (⚖) return `202 Accepted` + an `ApprovalRequest` when approval is needed, and `200/201` when below threshold. Only the five R31 action families use ⚖ (10 §11). |
| API-D13 | **Parameter values are cited by key (R48):** timers/thresholds → 13 §5.1, fee/commission defaults → 16 §6.5, operational keys → 10 §15.3. |
| API-D12 | Public catalog endpoints live under **`/public/*`**: cookie-less, `Cache-Control: public`, CDN-cacheable, usable by the service worker (17). |

---

## 1. Conventions

### 1.1 Hosts, audiences, prefixes

| Host | Audience | Auth | Paths served (all under `/api/v1`) |
|---|---|---|---|
| `app.<domain>` | `customer` | cookies | `/public/*`, `/config/*`, `/auth/*`, `/me/*`, `/cart/*`, `/orders/*`, `/payments/*`, `/support/*`, `/stream`, `/uploads` |
| `restaurant.<domain>` | `restaurant` | cookies | public, auth, `/me/*`, `/partner/*`, `/support/*`, `/stream`, `/uploads`, `/kyc/*` |
| `rider.<domain>` | `rider` | cookies | public, auth, `/me/*`, `/rider/*`, `/support/*`, `/stream`, `/uploads`, `/kyc/*` |
| `admin.<domain>` | `admin` | cookies (`SameSite=Strict`) | `/auth/admin/*`, `/me/*`, `/admin/*`, `/stream` |
| `api.<domain>` | V1: webhooks only; later `*_native` | signature (webhooks); bearer later; **no cookies, no CORS** | `/webhooks/*` |

Each app host has a CloudFront behaviour for `/api/*` (caching off, cookies forwarded) to the ALB. The ALB accepts only the CloudFront origin-facing prefix list **and** a secret origin-verify header. The server derives the audience from **`Host`** and requires that secret. A request carrying `X-Rovo-Audience` without the secret is rejected, and the header is otherwise ignored (RV-001, RV-025). A token or session whose audience doesn't match the host is rejected (`401 TOKEN_AUDIENCE_MISMATCH`). `/admin/*` on a non-admin host, `/partner/*` off the restaurant host and `/rider/*` off the rider host return 404. Because `RIDER` and `RESTAURANT_*` roles are mutually exclusive (R26), no user needs both partner hosts.

Health endpoints `GET /healthz` (liveness) and `GET /readyz` (DB, River, LISTEN) are outside `/api` and not exposed publicly (internal LB only).

### 1.2 Resource naming

- Plural nouns: `/orders`, `/menu-items`, `/delivery-offers`. IDs are UUIDv7 strings. The human order code (`RV-7K3P9Q`) is a field, accepted by admin search only.
- Nesting is at most 2 levels and only where ownership is real: `/partner/restaurants/{restaurantId}/menu-items/{itemId}`.
- Commands: `POST /orders/{orderId}/cancel`, `POST /partner/restaurants/{rid}/orders/{orderId}/accept`, `POST /rider/deliveries/{deliveryId}/picked-up`.
- Singletons for "my" resources: `/me`, `/rider/me`.
- `operationId` = lowerCamel verb + noun (`createOrder`, `acceptRestaurantOrder`). The TS client and Go handlers are named from it.
- Every operation declares `x-rovo-permission` (12 §5.2, e.g. `order.accept`) and `x-rovo-audiences`. Middleware enforces both (12 AUTH-D09).

### 1.3 JSON

- `camelCase` keys; `UPPER_SNAKE` enum values; booleans `isX`/`hasX` only where the noun is ambiguous.
- `null` is explicit for nullable fields, and fields are never omitted in responses (stable shapes for TS). Requests reject unknown properties (`additionalProperties: false` → `400 VALIDATION_FAILED`). Clients must ignore unknown response properties (forward compatibility).
- Translations: `name` (canonical) + `nameI18n: {"te": "…"}`. A localized convenience field `displayName` is resolved from `Accept-Language`, falling back to `name`. **This replaces 17's `name_te` expectation:** clients read `nameI18n.te` or `displayName` (challenge in the report).
- Images: `{"id", "url", "width", "height", "dominantColor", "variants": [{"w": 320, "url": "…"}]}`, where URLs are built at response time from the CDN base + object key (10 DB-D13).
- Geo: `{"lat": 16.7488, "lng": 78.0035}` (never GeoJSON in customer APIs; admin zone APIs use GeoJSON).
- Phone numbers are E.164 strings. Masked numbers are shown where policy requires (`"+91 98XXXXXX12"`).

### 1.4 Money, numbers, rounding

```json
{"amountPaise": 38100, "currency": "INR"}
```

- Never floats, never rupee strings. Negative amounts appear only in bill lines (discount, negative round-off).
- Rates are basis points (`"rateBps": 500` = 5%).
- **Bill lines** (`BillLine`): `component` (`ITEM_TOTAL | PACKAGING | DELIVERY_FEE | PLATFORM_FEE | SMALL_CART_FEE | DISCOUNT | TAX | ROUND_OFF`), `label`, `amount`, `isIncluded` (for "incl. GST" informational lines), and `tax` details. **Payable = Σ lines where `isIncluded=false`**. It is rounded to a whole rupee via an explicit `ROUND_OFF` line (ruling 8).

### 1.5 Pagination, filtering, sorting

- Request: `?limit=20&cursor=<opaque>` (limit 1–100, default 20).
- Response: `{"items": [...], "nextCursor": "eyJ…" | null}`.
- The cursor is base64url(JSON `{sortKey, id}`) + HMAC, so clients cannot forge it. An invalid cursor returns `400 INVALID_CURSOR`.
- Filters are explicit, allow-listed query params: `?status=PLACED,ACCEPTED` (comma = OR), `?createdFrom=2026-10-01T00:00:00Z&createdTo=…`, `?zoneId=`, `?q=` (search).
- Sorting: `?sort=-createdAt` (allow-listed per endpoint; `-` = descending). Default sort is documented per endpoint.
- No total counts on large lists. Dashboards use dedicated `/…/summary` endpoints.

### 1.6 Time

- Timestamps are RFC 3339 in UTC with `Z` and millisecond precision (`2026-10-04T13:05:12.345Z`). Clients render in the city timezone (`cities.timezone`, exposed in `/config/client`).
- Local wall-clock values (opening hours) are `"HH:MM"` plus `dayOfWeek` (ISO 1–7), always interpreted in the restaurant's city timezone.
- Dates are `YYYY-MM-DD`. Durations are integer seconds or minutes with the unit in the name (`prepTimeMinutes`, `waitingSeconds`).

### 1.7 Idempotency

- Header `Idempotency-Key: <UUIDv7>`, 8–128 chars. **Required** (missing → `400 IDEMPOTENCY_KEY_REQUIRED`) on:
  - `POST /orders`, `/orders/{id}/payments`, `/orders/{id}/cancel`, `/payments/confirm`;
  - all `/partner/.../orders/{id}/*` commands;
  - `/rider/offers/{id}/accept|decline`, all `/rider/deliveries/{id}/*` milestones, `/rider/cash-deposits`;
  - admin refunds, goodwill, cancellations, assignments, payout runs and mark-paid, COD verify, approvals, ledger and cash adjustments.
- Semantics (08 §7.1): same principal + key + same body → **replay** the stored status, headers and body (`Idempotent-Replayed: true`). Same key + different body → `422 IDEMPOTENCY_KEY_REUSED`. In-flight → `409 REQUEST_IN_PROGRESS` + `Retry-After: 1`. Keys expire after 24 h.
- The client generates one key **per user intent** and reuses it across retries (17). Rider milestones queued offline use the key as `clientEventId` (13 §7.2).

### 1.8 Errors (RFC 9457)

```http
HTTP/1.1 409 Conflict
Content-Type: application/problem+json
Content-Language: en

{
  "type": "/problems/quote-changed",
  "title": "Your order total changed",
  "status": 409,
  "detail": "Prices or availability changed since you reviewed the bill.",
  "instance": "/api/v1/orders",
  "code": "QUOTE_CHANGED",
  "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
  "retryable": false,
  "fieldErrors": [],
  "quote": { "...": "new Quote object" },
  "diff": [{"menuItemId": "0192…", "change": "PRICE_CHANGED", "from": {"amountPaise": 18000, "currency": "INR"}, "to": {"amountPaise": 19000, "currency": "INR"}}]
}
```

- `type` is a relative URI resolved against the API origin. Each code has a human page in the docs site later.
- `code` is **stable** and the only thing clients branch on. `title`/`detail` are localized per `Accept-Language` for display, but clients should prefer their own translation of `code` (17).
- `fieldErrors[]`: `{ "field": "lines[0].quantity", "code": "OUT_OF_RANGE", "detail": "1–20" }`.
- Extension members are allowed per code (`quote`, `diff`, `currentStatus`, `currentVersion`, `approvalRequest`, `retryAfterSeconds`).
- 404 is used for resources outside the caller's scope (12 AUTH-D09). 403 is used only when the caller can see the resource but may not act on it (e.g. a staff member without `MENU_EDIT`).
- 5xx never leak internals. `traceId` links to logs and traces.

**Error code catalogue (V1; stable):**

| HTTP | Codes |
|---|---|
| 400 | `VALIDATION_FAILED`, `INVALID_CURSOR`, `IDEMPOTENCY_KEY_REQUIRED`, `UNSUPPORTED_LOCALE`, `INVALID_LOCATION` |
| 401 | `UNAUTHENTICATED`, `TOKEN_EXPIRED`, `TOKEN_AUDIENCE_MISMATCH`, `SESSION_REVOKED`, `MFA_REQUIRED`, `STEP_UP_REQUIRED`, `WEBHOOK_SIGNATURE_INVALID` |
| 403 | `FORBIDDEN`, `ACCOUNT_BLOCKED`, `CAPTCHA_FAILED`, `CSRF_REJECTED`, `MAKER_CANNOT_APPROVE` |
| 404 | `NOT_FOUND` |
| 409 | `CONFLICT`, `REQUEST_IN_PROGRESS`, `ORDER_STATE_CONFLICT`, `ORDER_INVALID_TRANSITION`, `ORDER_NOT_CANCELLABLE`, `CANCEL_GRACE_EXPIRED`, `ACCEPT_WINDOW_CLOSED`, `QUOTE_CHANGED`, `QUOTE_EXPIRED`, `QUOTE_ALREADY_USED`, `PAYMENT_ALREADY_COMPLETED`, `OFFER_EXPIRED`, `OFFER_NOT_PENDING`, `DELIVERY_INVALID_TRANSITION`, `RIDER_HAS_ACTIVE_DELIVERY`, `UNDELIVERABLE_REQUEST_OPEN`, `REFRESH_RACE`, `ZONE_OVERLAP`, `DUPLICATE` |
| 412 | `VERSION_MISMATCH` (`If-Match` failed; body has `currentVersion`) |
| 422 | `IMPORT_INVALID` (row-level `fieldErrors`, M7), `IDEMPOTENCY_KEY_REUSED`, `OTP_INVALID`, `OTP_EXPIRED`, `OTP_ATTEMPTS_EXCEEDED`, `OUTSIDE_SERVICE_AREA`, `CITY_NOT_LIVE`, `ZONE_PAUSED`, `RESTAURANT_ZONE_PAUSED`, `TOO_FAR`, `RESTAURANT_CLOSED`, `RESTAURANT_PAUSED`, `ITEM_UNAVAILABLE`, `ADDON_SELECTION_INVALID`, `MIN_ORDER_NOT_MET`, `CART_EMPTY`, `CART_TOO_LARGE`, `COUPON_INVALID`, `COUPON_EXPIRED`, `COUPON_NOT_APPLICABLE`, `COUPON_MIN_ORDER_NOT_MET`, `COUPON_USAGE_EXCEEDED`, `COUPON_BUDGET_EXHAUSTED`, `COD_NOT_AVAILABLE`, `COD_LIMIT_EXCEEDED`, `COD_DISABLED_FOR_CUSTOMER`, `RIDER_NOT_ELIGIBLE`, `RIDER_CASH_LIMIT_REACHED`, `RIDER_NOT_ONLINE`, `DELIVERY_CODE_INVALID`, `COD_AMOUNT_MISMATCH`, `UNDELIVERABLE_PRECONDITION`, `REFUND_EXCEEDS_CAPTURED`, `ZONE_GEOMETRY_INVALID`, `REASON_CODE_INVALID`, `KYC_INCOMPLETE`, `FEATURE_DISABLED` |
| 426 | `CLIENT_UPGRADE_REQUIRED` (when `X-Rovo-App-Version` < minimum in `/config/client`) |
| 429 | `RATE_LIMITED`, `OTP_RESEND_TOO_SOON`, `SMS_BUDGET_EXCEEDED`, `TOO_MANY_STREAMS` |
| 502/503/504 | `PAYMENT_PROVIDER_UNAVAILABLE`, `SERVICE_UNAVAILABLE`, `UPSTREAM_TIMEOUT` (`retryable: true`) |
| 500 | `INTERNAL` |

### 1.9 Concurrency (ETag / If-Match)

- Reads of versioned aggregates return `ETag: "v12"` (weak semantics are not needed; the value is the row `version`). Bodies also include `"version": 12`.
- `PUT`/`PATCH` on menu items, categories, addon groups, restaurant profile, hours, zones, fee configs, coupons, payout accounts **require** `If-Match: "v12"`. Missing → `428 Precondition Required` (`IF_MATCH_REQUIRED`); stale → `412 VERSION_MISMATCH`.
- State commands accept an optional `If-Match`. Without it, the server applies 13 §7.1 (CAS on the current version, re-validated from the new state).
- `GET /public/restaurants/{id}/menu` returns `ETag: "m<menu_version>"` and supports `If-None-Match` → `304`.

### 1.10 Rate limiting

- Edge (WAF) limits plus app limits (12 §2, 19). App counters for OTP, auth and money endpoints live in Postgres `rate_limit_buckets`, shared across replicas (R21). Only soft general limits may use per-replica memory. Responses carry the IETF draft fields `RateLimit-Policy: "default";q=120;w=60` and `RateLimit: "default";r=37;t=21` (draft-ietf-httpapi-ratelimit-headers, still a draft [ASSUMPTION — track the final RFC]). A 429 carries `Retry-After` (seconds).
- Default budgets [ASSUMPTION — tuned in 19/24]:

| Scope | Budget |
|---|---|
| public reads | 120 / min / IP |
| authenticated | 300 / min / user |
| OTP request | per 12 §2.2 (per phone, per IP, global SMS budget); values are owned there |
| location pings | 2 / min / rider (batched uploads of 1–10 points every `rider.ping_interval_s`, R27) |
| restaurant heartbeat | 2 / min / device (every `restaurant.heartbeat_interval_s` = 60 s; SSE presence counts, R27) |
| quote | 30 / min / user |
| `POST /orders` | 10 / min / user |

### 1.11 Localization

- `Accept-Language: te-IN, en;q=0.8` selects localized `title`/`detail` in problems, `displayName` fields and server-rendered texts (notifications, invoices). Supported values come from `cities.supported_locales`. Unsupported values fall back to `en` (never an error). The response has `Content-Language`.
- The user's stored preference (`users.preferredLocale`, `PATCH /me`) drives push/SMS language. `Accept-Language` drives the HTTP response.
- Public cacheable endpoints `Vary: Accept-Language`, or better, return all translations (`nameI18n`) so the CDN cache isn't fragmented. **Decision: public catalog returns `name` + `nameI18n` and does not vary.**

### 1.12 Auth (detail in 12)

| Client | Credential | Notes |
|---|---|---|
| Web PWA (customer, restaurant, rider, admin) | `__Host-rovo_at` access cookie; `__Secure-rovo_rt` refresh cookie (path `/api/v1/auth`), host-only per app host | Unsafe methods require `X-Rovo-Client: customer-web|restaurant-web|rider-web|admin-web` + same-origin Fetch-Metadata (12 AUTH-D06) |
| Registered restaurant order-receiver device | as web, but the session is **device-bound** (R44): sliding 30-day idle, 90-day absolute, revocable by owner/admin (`POST /partner/restaurants/{rid}/devices/{deviceId}/register`) | 10 §3.3 `sessions.binding` |
| Rider PWA | as web; 30-day sliding idle (R44) | |
| Future native | `Authorization: Bearer <jwt>` on `api.` host (not enabled in V1); refresh token in the body of `/auth/refresh` | cookies ignored on `api.` |
| PA webhooks | HMAC signature header | no session |

All clients send `X-Rovo-App-Version: customer/1.4.2` (telemetry + `426` gate).

### 1.13 Versioning and deprecation

- The major version is in the path (`/api/v1`). **Additive changes** are non-breaking and ship any time: new endpoints, new optional request fields, new response fields, **new enum values** (clients must handle unknown enum values with a generic fallback; generated TS types use open unions).
- **Breaking changes** require `/api/v2` for the affected resources. The old version runs **≥ 6 months**, or ≥ 2 PWA release cycles after usage drops below 1%. Deprecated operations send `Deprecation: @<unix-ts>` (RFC 9745 [ASSUMPTION — verify number]) and `Sunset: <HTTP-date>` (RFC 8594) plus `Link: <docs>; rel="deprecation"`.
- The PWA self-updates. `/config/client.minVersions` forces a reload or upgrade for critical fixes.
- The spec is linted in CI (Spectral ruleset: operationId, `x-rovo-permission`, examples, problem responses) and **oasdiff** blocks breaking changes on `v1` [ASSUMPTION — 21 owns tooling].

### 1.14 Caching

| Endpoint class | Header |
|---|---|
| `/public/*` lists | `Cache-Control: public, max-age=30, stale-while-revalidate=60` |
| `/public/restaurants/{id}/menu` | `public, max-age=30` + `ETag` (menu_version) |
| `/config/client` | `public, max-age=300` |
| Everything authenticated | `Cache-Control: private, no-store` (default) |
| KYC/document views | `no-store` + `Content-Disposition: inline` (12 §6.2) |

---

## 2. Endpoint catalogue

Legend:
- Roles: `PUB` (anonymous), `CUST`, `OWN` (RESTAURANT_OWNER), `STAFF` (RESTAURANT_STAFF), `RIDER`, `SUPER`, `OPS`, `SUP` (support), `FIN`. `ADMIN*` = any admin role in city scope.
- `🔑` = Idempotency-Key required. `🔒` = If-Match required. `⚖` = may return 202 + approval (maker-checker). `⬆` = step-up required (12).
- The permission strings in `x-rovo-permission` follow 12 §5.2.

### 2.1 Public catalog and config (`/public`, `/config`)

| Method & path | Roles | Request → Response | Notable errors |
|---|---|---|---|
| `GET /config/client` | PUB | → `{cities[], minVersions, supportPhone, features{}, tileStyleUrl, timezoneByCity, legalLinks}` | — |
| `GET /public/cities` | PUB | → `[{id, code, name, nameI18n, status, centroid}]` | — |
| `GET /public/serviceability?lat&lng` | PUB | → `{status: SERVICEABLE|OUTSIDE_SERVICE_AREA|CITY_NOT_LIVE|ZONE_PAUSED, cityId, zoneId, message, localityGuess{id,name}}` (16 §3) | `INVALID_LOCATION` |
| `GET /public/localities?cityId&q` | PUB | → `[{id, name, nameI18n, pinCodes, centroid}]` (trigram + aliases) | — |
| `GET /public/cuisines?cityId` | PUB | → `[{code, name, nameI18n}]` | — |
| `GET /public/restaurants?lat&lng&cuisine&veg&minRating&maxEtaMin&hasOffer&sort&cursor&limit` | PUB | → `{items: [RestaurantCard{id, name, nameI18n, image, cuisines, dietType, rating{avg,count}, costForTwo, eta{minMinutes,maxMinutes}, deliveryFee, distanceM, isOpen, opensAt, availability: OPEN|CLOSED|PAUSED|TOO_FAR, offerBadge}], nextCursor, zone{paused, message}}` (no surge, R30) | `OUTSIDE_SERVICE_AREA` (422 with a body), `INVALID_LOCATION` |
| `GET /public/restaurants/{restaurantId}?lat&lng` | PUB | → `RestaurantDetail` + `serviceability{status, deliveryFee, eta}` for the point, `fssaiLicenseNo` (shown on the menu [LEGAL]), hours, address | `NOT_FOUND` |
| `GET /public/restaurants/{restaurantId}/menu` | PUB | → `{menuVersion, categories[{id, name, nameI18n, items[MenuItem{id, name, nameI18n, description, vegType, price, packaging, image, isAvailable, unavailableUntil, isRecommended, variants[], addonGroups[{id, name, minSelect, maxSelect, addons[]}]}]}]}`; `ETag` | `NOT_FOUND`; `304` |
| `GET /public/search?q&lat&lng&cursor` | PUB | → `{restaurants[RestaurantCard], dishes[{item, restaurant}]}` (serviceable only) | — |
| `GET /public/offers?lat&lng` | PUB | → public coupons applicable in the zone `[{code, title, terms, discountType, minOrder, maxDiscount}]` | — |
| `POST /public/waitlist` 🔑 | PUB/CUST (app host) | `{location{lat,lng}, phone?, pinCode?, localityId?, noticeVersion, captchaToken}` → 201 (shown when `OUTSIDE_SERVICE_AREA`; M4 `waitlist`) | `RATE_LIMITED`, `CAPTCHA_FAILED` |
| `POST /public/restaurant-leads` 🔑 | PUB (restaurant host) | `{businessName, contactName, phone, otpChallengeId, otpCode, localityId?, cuisineCodes[], fssaiState: YES|NO|APPLIED, preferredCallTime?, noticeVersion}` → 201 (05 §2; M4 `leads`) | `OTP_INVALID`, `RATE_LIMITED` |

### 2.2 Auth, session, me (`/auth`, `/me`), aligned with 12 §7

| Method & path | Roles | Request → Response | Notable errors |
|---|---|---|---|
| `POST /auth/otp/request` | PUB (customer, restaurant, rider hosts) | `{phone, channel: SMS, captchaToken?}` (SMS only, C2; captcha only at risk per 12) → **202** `{challengeId, resendAfterSeconds, expiresAt}`. The response is the same whether or not the phone exists. | `CAPTCHA_FAILED`, `OTP_RESEND_TOO_SOON`, `RATE_LIMITED`, `SMS_BUDGET_EXCEEDED`, `VALIDATION_FAILED` (non-Indian mobile) |
| `POST /auth/otp/verify` | PUB | `{challengeId, code, deviceInstallId}` → `{user{id, fullName, roles}, isNewUser, accessExpiresAt, needsAccountConfirmation}`; sets cookies (bearer variant returns tokens) | `OTP_INVALID`, `OTP_EXPIRED`, `OTP_ATTEMPTS_EXCEEDED`, `ACCOUNT_BLOCKED` |
| `POST /auth/account-confirmation` | CUST (fresh session) | `{decision: SAME_PERSON|NEW_NUMBER}` → recycled-number flow (12 §1.4) | — |
| `POST /auth/refresh` | any session | (cookie) → `{accessExpiresAt}`; rotates the refresh token | `SESSION_REVOKED`, `REFRESH_RACE` (409) |
| `POST /auth/logout` | any | → 204; revokes the family | — |
| `POST /auth/context` | OWN/STAFF (restaurant host) | `{context: "restaurant:<id>"}` → new access token (multi-outlet owners; rider ⟂ restaurant per R26) | `FORBIDDEN` |
| `POST /auth/step-up` | any | `{method: OTP|TOTP, code}` → `{stepUpUntil}` | `OTP_INVALID` |
| `POST /auth/admin/login` | PUB (admin host) | `{email, password}` → `{mfaToken, mfaMethods: ["TOTP","RECOVERY_CODE"]}` | 401 generic |
| `POST /auth/admin/mfa` | PUB + mfaToken | `{mfaToken, totp | recoveryCode}` → session cookies | `MFA_INVALID` (401) |
| `POST /auth/admin/setup` | PUB + setup token | `{setupToken, password, totpCode}` → session | — |
| `POST /auth/admin/password-reset/request`, `/complete` | PUB | generic responses | — |
| `GET /me` | any | → `{id, kind, fullName, phoneMasked, email, preferredLocale, roles[{role, cityId, restaurantId}], activeContext, consents{}, accessExpiresAt}` | — |
| `PATCH /me` | any | `{fullName?, preferredLocale?}` → Me | — |
| `GET /me/sessions`, `DELETE /me/sessions/{sid}`, `DELETE /me/sessions?others=true` | any | device list (incl. `binding: STANDARD|DEVICE_BOUND`) / revoke | — |
| `POST /me/phone-change/start`, `/confirm` | CUST, OWN, STAFF, RIDER | step-up + double OTP (12 §2.4) | — |
| `GET /me/consents`, `POST /me/consents` | any | `{purpose, granted, noticeVersion}` (append-only) | — |
| `POST /me/data-export` | any | → 202 `{reportExportId}` (DPDP access request) | `RATE_LIMITED` |
| `POST /me/erasure-request` | CUST, RIDER, OWN | `{reason?}` → 202 `{erasureRequestId, status, dueAt}` (DPDP; `erasure_requests`; executed by the erasure job after hold checks, 10 §13.1; audited, no maker-checker per R31) | `CONFLICT` (request already open) |
| `GET /me/notifications?cursor&unreadOnly` | any | inbox | — |
| `POST /me/notifications/read` | any | `{ids[] | all: true}` → 204 | — |
| `POST /me/push-subscriptions` | any | `{endpoint, keys{p256dh, auth}, deviceInstallId, locale}` → 201 `{id}` (upsert by endpoint) | `VALIDATION_FAILED` |
| `DELETE /me/push-subscriptions/{id}` | any | → 204 | — |

### 2.3 Customer

| Method & path | Roles | Request → Response | Notable errors |
|---|---|---|---|
| `GET /me/addresses` | CUST | → `[Address]` | — |
| `POST /me/addresses` | CUST | `{label, labelCustom?, houseNo, buildingStreet?, localityId?, areaText?, landmark (required), pinCode, location{lat,lng}, locationAccuracyM?, contactName?, contactPhone?, deliveryInstructions?, isDefault?}` → 201 Address + `serviceability` | `VALIDATION_FAILED` (landmark/pin required, ruling 13), `INVALID_LOCATION` |
| `PATCH /me/addresses/{addressId}` | CUST | partial → Address | — |
| `DELETE /me/addresses/{addressId}` | CUST | → 204 (hard delete) | — |
| **`POST /cart/quote`** | CUST (guests get a preview: `addressId` omitted and `location` given; `quoteId` null, not orderable) | `{restaurantId, lines[{menuItemId, variantId?, addonIds[], quantity, note?}], addressId, paymentMethod: ONLINE|COD, couponCode?}` → `Quote{quoteId, expiresAt, isOrderable, lines[], bill{lines[], payable}, eta, coupon{code, status, message}, paymentOptions[{method, available, reasonCode}], issues[]}` (§3.1) | `RESTAURANT_CLOSED`, `TOO_FAR`, `ZONE_PAUSED`, `CART_EMPTY` (422). Item-level problems are returned in `issues[]` with 200, so the cart UI can show them inline. |
| `GET /me/offers?restaurantId` | CUST | → coupons the user can apply (public + goodwill owned), with eligibility hints | — |
| **`POST /orders`** 🔑 | CUST | `{quoteId, customerNote?, deliveryInstructions?}` → **201** `{order: Order, payment: PaymentSession | null}` (§3.2) | `QUOTE_CHANGED` / `QUOTE_EXPIRED` (409 + new quote + diff), `QUOTE_ALREADY_USED`, `COD_*`, `COUPON_*`, `RESTAURANT_*`, `PAYMENT_PROVIDER_UNAVAILABLE` (order still created, `payment: null`, `paymentRetryable: true`) |
| `POST /orders/{orderId}/payments` 🔑 | CUST | → 201 `PaymentSession` (retry/initiate checkout for a `PENDING_PAYMENT` order) | `PAYMENT_ALREADY_COMPLETED`, `ORDER_INVALID_TRANSITION` |
| `POST /payments/confirm` 🔑 | CUST | `{paymentId, providerOrderId, providerPaymentId, signature}` → `{paymentStatus, orderStatus}` (fast path; the webhook is authoritative; 08 §5.1) | `WEBHOOK_SIGNATURE_INVALID` (401 → treated as "pending") |
| `GET|POST /payments/return/{provider}` | PUB (unauthenticated, non-trusting) | PA redirect target → 303 to `/orders/{id}` in the app (12 §4.4) | — |
| `GET /orders?status=ACTIVE|PAST&cursor` | CUST | → `{items[OrderSummary], nextCursor}`; active pinned first | — |
| `GET /orders/{orderId}` | CUST | → `Order` (items, bill, status timeline, delivery milestone + rider first name, `cancellable{allowed, freeUntil, reasonCodes[]}`, `deliveryCode` (R39), refunds[], ratings, invoiceAvailable); `ETag` | `NOT_FOUND` |
| `POST /orders/{orderId}/cancel` 🔑 | CUST | `{reasonCode}` → Order (O-05/O-09/O-11/O-13 within grace) | `CANCEL_GRACE_EXPIRED`, `ORDER_NOT_CANCELLABLE` |
| `POST /orders/{orderId}/reorder` | CUST | → `{restaurantId, lines[], issues[]}` (for the device cart; never places an order) | — |
| `POST /orders/{orderId}/ratings` 🔑 | CUST | `{restaurant?{stars, tags[], review?}, rider?{thumbsUp, tags[]}}` → 201. Window `ratings.window_days`; once per target. Review text passes a profanity/PII filter and is shown immediately unless flagged; no moderation queue or replies (C14). | `CONFLICT` (already rated), `ORDER_INVALID_TRANSITION` (not delivered) |
| `GET /orders/{orderId}/invoices` | CUST | → `[{id, kind, number, issuedAt, fileUrl}]` | — |
| `POST /support/tickets` 🔑 | CUST, OWN, STAFF, RIDER | `{orderId?, category, subject, message, attachmentFileIds[]}` → 201 Ticket | `VALIDATION_FAILED` |
| `GET /support/tickets`, `GET /support/tickets/{id}` | requester | list/detail with messages (no internal notes) | — |
| `POST /support/tickets/{id}/messages` | requester | `{body, attachmentFileIds[]}` → 201 | — |
| `POST /support/tickets/{id}/compensation-choice` 🔑 | CUST (COD order, when the ticket offers compensation) | `{choice: UPI_REFUND|COUPON, upiVpa?}` → `{refundId|couponId, status}`. **R29:** a manual UPI refund (finance pays and records the UTR) or a single-user coupon; never coupon-only. | `VALIDATION_FAILED` (VPA), `CONFLICT` (already chosen) |
| `POST /uploads` | CUST, OWN, STAFF, RIDER | `{purpose: MENU_IMAGE|TICKET_ATTACHMENT|…, contentType, sizeBytes, sha256}` → `{fileId, uploadUrl, uploadHeaders, expiresAt}` (presigned PUT, 5 min) | `VALIDATION_FAILED` (type/size) |
| `POST /uploads/{fileId}/complete` | uploader | → `File{id, status, url?}` (validates type/size, re-encodes images; no ClamAV in V1, R38) | — |

### 2.4 Restaurant partner (`/partner/…`, partner host)

All `/partner/restaurants/{restaurantId}/*` operations require a restaurant role on `restaurantId` (12 §5.3). They are served on the `restaurant.` host only. Order-, finance- and menu-edit operations also check staff `permissions[]`.

| Method & path | Roles | Request → Response | Notable errors |
|---|---|---|---|
| `POST /partner/restaurant-applications` | any member (becomes OWN on approval) | `{name, phone, address, location, pinCode, localityId?, cuisineCodes[], dietType}` → 201 `{restaurantId, status: DRAFT}` | `OUTSIDE_SERVICE_AREA` (warning field, not a block) |
| `GET/PATCH /partner/restaurant-applications/{restaurantId}` 🔒 | applicant OWN | save-and-resume steps (05 §2) | `VERSION_MISMATCH` |
| `POST /kyc/uploads` | OWN, RIDER | `{docType, contentType, sizeBytes}` → presigned PUT to the private KYC bucket (12 §6.2) | — |
| `POST /partner/restaurant-applications/{restaurantId}/documents` | OWN | `{docType, fileId, docNumber?, validUntil?}` → 201 | `VALIDATION_FAILED` |
| `POST /partner/restaurant-applications/{restaurantId}/submit` 🔑 | OWN | `{agreementVersion, otpChallengeId, otpCode}` → `{status: SUBMITTED}` | `KYC_INCOMPLETE` |
| `GET /partner/restaurants` | OWN, STAFF | → my outlets + role per outlet | — |
| `GET /partner/restaurants/{rid}` / `PATCH` 🔒 | OWN (STAFF read) | profile (name, description, images, minOrder, packaging mode, radius ≤ admin cap, avgPrepTime) | `VERSION_MISMATCH` |
| `GET /partner/restaurants/{rid}/operating-hours` / `PUT` 🔒 | GET: OWN, STAFF · PUT: OWN only (R57) | `{week: [{dayOfWeek, slots[{opensAt, closesAt}]}]}` (atomic replace) | `VALIDATION_FAILED` (overlap) |
| `POST /partner/restaurants/{rid}/closures` / `DELETE …/closures/{id}` | OWN | `{startsAt, endsAt, reason, note}` | `CONFLICT` (overlap) |
| `POST /partner/restaurants/{rid}/open` | OWN, STAFF | "Start taking orders" (`accepting_orders=true`) | `KYC_INCOMPLETE`, `FORBIDDEN` (suspended) |
| `POST /partner/restaurants/{rid}/close` | OWN, STAFF | stop for the day | — |
| `POST /partner/restaurants/{rid}/pause` | OWN, STAFF | `{durationMinutes: 15|30|60|null(=until resume), reasonCode}` → `{pausedUntil}` | — |
| `POST /partner/restaurants/{rid}/resume` | OWN, STAFF | → `{pausedUntil: null}` (also clears auto-pauses, ruling 1) | — |
| `POST /partner/restaurants/{rid}/devices/heartbeat` | OWN, STAFF | `{deviceInstallId, isOrderReceiver, pushOk, appVersion}` → `{serverTime, outletState}` every `restaurant.heartbeat_interval_s` (60 s); an open SSE stream also counts as presence (R1, R27) | — |
| `GET /partner/restaurants/{rid}/devices` | OWN | → `[{deviceId, label, isOrderReceiver, binding, lastHeartbeatAt, registeredBy}]` | — |
| `POST /partner/restaurants/{rid}/devices/{deviceId}/register` 🔑 | OWN (on the device, fresh step-up) | `{label, isOrderReceiver: true}` → rebinds this session as **device-bound** (R44: sliding 30 d idle, 90 d absolute) | `STEP_UP_REQUIRED` ⬆ |
| `POST /partner/restaurants/{rid}/devices/{deviceId}/revoke` 🔑 | OWN | → 204; revokes the bound session (`DEVICE_REVOKED`) | — |
| `GET /partner/restaurants/{rid}/menu` | OWN, STAFF | full menu incl. archived toggle and unavailable items | — |
| `POST /partner/restaurants/{rid}/menu-categories` / `PATCH /{id}` 🔒 / `DELETE /{id}` (archive) | OWN, STAFF+`MENU_EDIT` | `{name, nameI18n, sortOrder, isActive}` | `CONFLICT` (name) |
| `POST /partner/restaurants/{rid}/menu-items` / `PATCH /{itemId}` 🔒 / `DELETE /{itemId}` (archive) | OWN, STAFF+`MENU_EDIT` | `{categoryId, name, nameI18n, description, descriptionI18n, vegType, price, packaging, imageFileId, isRecommended, spiceLevel, serves, variants[], addonGroupIds[]}` | `VALIDATION_FAILED` (pure-veg rule), `VERSION_MISMATCH` |
| `POST /partner/restaurants/{rid}/menu-items/{itemId}/availability` | OWN, STAFF | `{isAvailable, until?: ISO | "END_OF_DAY" | null}` (no If-Match: a toggle, last write wins) | — |
| `POST /partner/restaurants/{rid}/addon-groups` / `PATCH` 🔒 / `DELETE` | OWN, STAFF+`MENU_EDIT` | `{name, minSelect, maxSelect, addons[{name, price, vegType, isAvailable}]}` | `VALIDATION_FAILED` (min ≤ max) |
| `PUT /partner/restaurants/{rid}/menu/sort-order` 🔒 | OWN, STAFF+`MENU_EDIT` | `{categories[{id, items[id]}]}` | — |
| `GET /partner/restaurants/{rid}/orders?view=NEW|PREPARING|READY|OUT|HISTORY&date&cursor` | OWN, STAFF | → `{items[PartnerOrder]}` (customer first name only; no phone) | — |
| `GET /partner/restaurants/{rid}/orders/{orderId}` | OWN, STAFF | → PartnerOrder (items, notes, rider status, timeline, earnings breakdown) | `NOT_FOUND` |
| `POST …/orders/{orderId}/ack` | OWN, STAFF | → 204. "Seen" acknowledgement that stops the repeat ring on this device (15 §7; register row 64). It does not change order state. | — |
| **`POST …/orders/{orderId}/accept`** 🔑 | OWN, STAFF | `{prepTimeMinutes}` → PartnerOrder (O-06) (§3.4) | `ACCEPT_WINDOW_CLOSED`, `ORDER_INVALID_TRANSITION` |
| `POST …/orders/{orderId}/reject` 🔑 | OWN, STAFF | `{reasonCode, outOfStockItemIds[]?, outOfStockUntil?, pauseMinutes?}` → PartnerOrder (O-07) | `REASON_CODE_INVALID` |
| `POST …/orders/{orderId}/start-preparing` 🔑 | OWN, STAFF | → PartnerOrder (O-10) | — |
| `POST …/orders/{orderId}/ready` 🔑 | OWN, STAFF | → PartnerOrder (O-12) | — |
| `POST …/orders/{orderId}/prep-extension` 🔑 | OWN, STAFF | `{extraMinutes: 5|10}` → updates ETA (max 2 extensions) | `CONFLICT` |
| `POST …/orders/{orderId}/issues` 🔑 | OWN, STAFF | `{type: CANNOT_FULFIL|ITEM_PROBLEM, itemIds[], note}` → 201 Ticket (urgent; ops decides and cancels with fault, 05 §4.5). **There is no restaurant cancel endpoint after accept (R40).** | — |
| `GET /partner/restaurants/{rid}/analytics/summary?period=TODAY|THIS_WEEK` | OWN | → `{orders, gross, netEarnings, avgRating, acceptRate, missedOrders, avgAcceptSeconds, prepOnTimeRate, topItems[]}` | — |
| `GET /partner/restaurants/{rid}/statements?from&to` | OWN (+STAFF with `FINANCE_VIEW`) | → per-order lines (gross, commission, GST, TDS, net) | — |
| `GET /partner/restaurants/{rid}/payouts`, `/payouts/{payoutId}` | OWN | payouts with UTR and items | — |
| `GET/POST /partner/restaurants/{rid}/payout-accounts` | OWN | add a bank/UPI destination ⚖ (maker-checker + cooling-off) | `STEP_UP_REQUIRED` ⬆ |
| `GET/POST/DELETE /partner/restaurants/{rid}/staff` | OWN | `POST` creates a `staff_invites` row (single-use code by SMS, 72 h) `{phone, displayName, permissions[]}`; `DELETE` removes a member | `CONFLICT` (rider role exclusion, 12 AUTH-D02) |
| `POST /partner/staff-invites/accept` 🔑 | any member (fresh OTP session on the restaurant host) | `{inviteCode}` → `{restaurantId, role: RESTAURANT_STAFF}`; the session phone must match the invite | `NOT_FOUND` (expired/used), `CONFLICT` (rider role) |
| `GET /partner/restaurants/{rid}/reviews?cursor` | OWN, STAFF | visible reviews + ratings (no reply in V1, C14) | — |

### 2.5 Rider (`/rider/…`, `rider.` host)

| Method & path | Roles | Request → Response | Notable errors |
|---|---|---|---|
| `POST /rider/applications` | any member | `{cityId, vehicleType, vehicleRegNo?, emergencyContact, adultDeclaration: true, gigRegistration{legalName, dateOfBirth, gender, residentialState, residentialDistrict, portalWorkerId?}}` → 201 `{riderId, status: APPLIED}`. Gig-worker fields per M5 [LEGAL — final list per portal spec]; never Aadhaar. | `CONFLICT` (restaurant role exclusion) |
| `POST /rider/applications/documents` | applicant | `{docType, fileId, docNumber?, validUntil?}` (AADHAAR_MASKED: no number) | `VALIDATION_FAILED` |
| `POST /rider/applications/submit` 🔑 | applicant | → `{status: UNDER_REVIEW}` | `KYC_INCOMPLETE` |
| `GET /rider/me` | RIDER | → profile, status, availability state, cash `{inHand, limit, codBlocked, oldestCashAgeHours}`, today's earnings, rating (thumbs %) | — |
| `PUT /rider/availability` 🔑 | RIDER | `{state: AVAILABLE|OFFLINE, location{lat,lng,accuracyM}}` → availability (13 §4.3; no `ON_BREAK`, C12) | `RIDER_NOT_ELIGIBLE`, `RIDER_HAS_ACTIVE_DELIVERY`, `KYC_INCOMPLETE` |
| `POST /rider/location` | RIDER | `{points[{lat, lng, accuracyM, speedMps?, headingDeg?, recordedAt}] (1–10)}` → 204. Batched upload every `rider.ping_interval_s` (60 s) while online (P12, R27); also refreshes `last_seen_at` for the 15-min auto-offline (R34). | `RATE_LIMITED` |
| `POST /rider/sos` 🔑 | RIDER | `{kind: ACCIDENT|UNSAFE|HARASSMENT|MEDICAL|OTHER, location?, deliveryId?, note?}` → 201 `{sosId}` → ops `ops.alert` SOS (06 §11; `sos_events`; P1) | — |
| `GET /rider/offers?status=PENDING` | RIDER | → current pending offer (polling fallback for SSE) | — |
| **`POST /rider/offers/{offerId}/accept`** 🔑 | RIDER | → `RiderDelivery` (D-03) (§3.5) | `OFFER_EXPIRED`, `OFFER_NOT_PENDING`, `RIDER_CASH_LIMIT_REACHED`, `RIDER_NOT_ONLINE` |
| `POST /rider/offers/{offerId}/decline` 🔑 | RIDER | `{reasonCode}` → 204 (D-04) | `OFFER_NOT_PENDING` |
| `GET /rider/deliveries/active` | RIDER | → RiderDelivery or `null` (pickup name, address, phone, items count, COD amount, drop address + landmark + customer first name + masked/relay phone [OPEN 12 §9], instructions, milestones) | — |
| `GET /rider/deliveries/{deliveryId}` | RIDER (own) | detail/history item | `NOT_FOUND` |
| **`POST /rider/deliveries/{deliveryId}/arrived-at-restaurant`** 🔑 | RIDER (own); OPS on behalf via the admin path | `DeliveryMilestoneRequest` → RiderDelivery (D-07) (§3.6) | `DELIVERY_INVALID_TRANSITION` |
| **`POST /rider/deliveries/{deliveryId}/picked-up`** 🔑 | RIDER | `DeliveryMilestoneRequest{pickupPin?}` → RiderDelivery (D-08/D-08b) | `DELIVERY_INVALID_TRANSITION` (order not PREPARING/READY) |
| **`POST /rider/deliveries/{deliveryId}/arrived-at-drop`** 🔑 | RIDER | `DeliveryMilestoneRequest` (D-09) | — |
| **`POST /rider/deliveries/{deliveryId}/delivered`** 🔑 | RIDER | `DeliveryMilestoneRequest{codCollected?: Money, deliveryCode?}` (D-10/D-10b) | `COD_AMOUNT_MISMATCH`, `DELIVERY_CODE_INVALID`, `UNDELIVERABLE_REQUEST_OPEN` |
| `POST /rider/deliveries/{deliveryId}/call-attempts` 🔑 | RIDER | `{at, location?}` → `{callAttempts}`. Writes a `contact_tap_log` row; counts toward the undeliverable precondition. | — |
| `POST /rider/deliveries/{deliveryId}/undeliverable-requests` 🔑 | RIDER | `DeliveryMilestoneRequest{reasonCode, note?}` → `{ticketId, status: AWAITING_SUPPORT}` (D-11, ruling 5) | `UNDELIVERABLE_PRECONDITION` (wait < 10 min / calls < 2) |
| `POST /rider/deliveries/{deliveryId}/release` 🔑 | RIDER | `{reasonCode}` → 204 (D-13; before pickup) | `DELIVERY_INVALID_TRANSITION` |
| `GET /rider/history?cursor` | RIDER | completed deliveries with pay | — |
| `GET /rider/earnings?from&to` | RIDER | → `{total, deliveries, base, distance, waiting, cancellations[], adjustments[{type: MG_TOPUP|PEAK_BONUS|…, amount}], byDay[]}` | — |
| `GET /rider/cash` | RIDER | → `{cashInHand, limit, codBlocked, pendingDeposits[]}` | — |
| `POST /rider/cash-deposits` 🔑 | RIDER | `{amount, method: UPI_TO_COMPANY|BANK_DEPOSIT|CASH_AT_HUB, reference, proofFileId?}` → 201 `CodDeposit{status: DECLARED}` | `VALIDATION_FAILED` (amount > cash in hand) |
| `GET /rider/payouts`, `GET/POST /rider/payout-accounts` | RIDER | as the restaurant equivalents ⚖ ⬆ (payout schedule per `payouts.rider.schedule`, 10 §15.3) | — |

### 2.6 Admin (`/admin/…`, admin host; every list is filtered by the caller's city scope)

| Area | Method & path | Roles | Notes / errors |
|---|---|---|---|
| Dashboard | `GET /admin/dashboard?cityId&date` | ADMIN* | `{ordersByStatus, gmv, aov, cancelRate, avgDeliveryMin, onlineRiders, unassignedDeliveries, pausedRestaurants, pausedZones, alerts[]}` |
| Live board | `GET /admin/orders?status&zoneId&restaurantId&late=true&q&cursor` | OPS, SUP | `q` matches order code / phone (audited). Exceptions first. |
| | `GET /admin/orders/{orderId}` | OPS, SUP, FIN | full: history, payments, refunds, delivery + offers, ledger summary, tickets, audit |
| Interventions | `POST /admin/orders/{orderId}/cancel` 🔑 ⚖ | OPS, SUP | `{reasonCode, faultParty, refundPolicy: FULL|PARTIAL|NONE, refundAmount?, restaurantCompensation: bool, note}` (13 §6) → Order (+ ApprovalRequest if the refund > `approvals.refund_threshold_paise`). Also the path for restaurant `CANNOT_FULFIL` issues (R40). |
| | `POST /admin/orders/{orderId}/accept-on-behalf` 🔑 | OPS | `{prepTimeMinutes, reason}` (O-06, ruling 1) |
| | `POST /admin/orders/{orderId}/ready-on-behalf` 🔑 | OPS | O-12 |
| | `POST /admin/deliveries/{deliveryId}/assign` 🔑 | OPS | `{riderId, reason, overrideCashLimit?}` → Delivery (D-06); a pending offer becomes `REVOKED`; the override is audited, not ⚖ (R31) |
| | `POST /admin/deliveries/{deliveryId}/unassign` 🔑 | OPS | `{reason}` (D-13) |
| | `POST /admin/deliveries/{deliveryId}/milestones/{milestone}` 🔑 | OPS | on-behalf rider milestone (`arrived-at-restaurant|picked-up|arrived-at-drop|delivered`) with reason (13 §1.2) |
| | `POST /admin/deliveries/{deliveryId}/undeliverable/confirm` 🔑 | SUP, OPS | `{faultParty, note}` (D-12 → O-18) |
| | `POST /admin/deliveries/{deliveryId}/undeliverable/reject` 🔑 | SUP, OPS | `{note}` (D-11r) |
| Refunds & goodwill | `POST /admin/orders/{orderId}/refunds` 🔑 ⚖ | SUP, FIN | `{amount, reasonCode, note}` → Refund or 202 Approval | `REFUND_EXCEEDS_CAPTURED` |
| | `GET /admin/refunds?status&cursor` | FIN, SUP | |
| | `POST /admin/goodwill-coupons` 🔑 ⚖ | SUP | `{userId, amount, ticketId?, orderId?, validDays}` → Coupon (R9; approval > `approvals.goodwill_threshold_paise`) |
| | `POST /admin/refunds/{refundId}/mark-paid` 🔑 | FIN | `{utrReference, paidAt}` for `MANUAL_UPI`/`MANUAL_BANK` refunds (COD compensation, R29) → Refund `SUCCEEDED` + journal | `VALIDATION_FAILED` |
| Users | `GET /admin/users?q&role&cursor`, `GET /admin/users/{id}` | SUP, OPS, SUPER | masked PII; reveal via `POST /admin/users/{id}/reveal` ⬆ (audited) |
| | `POST /admin/users/{id}/block` / `unblock` 🔑 | SUP, OPS | reason required; audited + next-day review (no ⚖, R31) |
| | `POST /admin/users/{id}/sessions/revoke` | SUP | |
| | `POST /admin/users/{id}/cod-status` | SUP, OPS | `{status: ENABLED|DISABLED, reason}` |
| Admin staff | `POST /admin/admins` ⚖, `POST /admin/admins/{id}/roles` ⚖ (R31 family 5), `DELETE /admin/admins/{id}/roles/{roleId}` ⬆, `POST /admin/admins/{id}/totp-reset` ⬆ | SUPER | 12 §3.3; revoke and TOTP reset are step-up + audit |
| Restaurants | `GET /admin/restaurants?status&q&cursor`, `GET /admin/restaurants/{id}` | OPS | |
| | `POST /admin/restaurants/{id}/approve` 🔑 | OPS | requires an approved FSSAI doc + zone + commission plan |
| | `POST /admin/restaurants/{id}/request-changes`, `/reject`, `/suspend`, `/reinstate`, `/offboard` 🔑 | OPS | reason required |
| | `PATCH /admin/restaurants/{id}` 🔒 | OPS | admin-only fields (radius cap, zone override) |
| | `POST /admin/restaurants/{id}/pause` / `resume` | OPS | |
| | `POST /admin/restaurants/{id}/devices/{deviceId}/revoke` 🔑 | OPS | revoke a device-bound session (R44) |
| | `POST /admin/restaurants/{id}/menu-imports?dryRun=true|false` 🔑 | OPS | **M7 (P0 for Gate B).** `{fileId}` (CSV uploaded via `/uploads`, purpose `MENU_IMPORT_CSV`; columns: category, name, nameTe, vegType, price, packaging, variant, addonGroup…). The dry run returns `{rowsOk, rowErrors[]}`. A real run validates again and applies everything in one transaction, then bumps `menu_version`. Synchronous; ≤ 1,000 rows. | `IMPORT_INVALID` (row errors) |
| | `GET /admin/kyc/{docId}/view` ⬆ | OPS, SUPER, FIN (bank proof) | streamed, `no-store`, audited (12 §6.2) |
| | `POST /admin/kyc/{docId}/decision` 🔑 | OPS | `{decision: APPROVED|REJECTED, reason?}` |
| Commissions | `GET /admin/restaurants/{id}/commission-plans`, `POST …` 🔑 ⚖ | OPS (maker), FIN/SUPER (checker) | `{commissionBps, basis, effectiveFrom, contractRef}`; overlap → `CONFLICT` |
| Riders | `GET /admin/riders?status&state&q`, `GET /admin/riders/{id}` | OPS | |
| | `POST /admin/riders/{id}/approve`, `/reject`, `/suspend`, `/reinstate`, `/force-offline` 🔑 | OPS | |
| | `POST /admin/riders/{id}/cash-limit` 🔑 | OPS | per-rider override; audited (no ⚖, R31) |
| | `POST /admin/riders/{id}/cash-adjustments` 🔑 ⚖ | FIN | cash correction (`CASH_CORRECTION`); ⚖ above `approvals.adjustment_threshold_paise` |
| | `GET /admin/sos-events?status=OPEN`, `POST /admin/sos-events/{id}/acknowledge` / `resolve` 🔑 | OPS | SOS board (06 §11, 07 §4; P1) |
| Ledger | `POST /admin/ledger-adjustments` 🔑 ⚖ | FIN (maker); OPS may propose `PEAK_BONUS` | `{payeeType: RIDER|RESTAURANT, payeeId, adjustmentType: MG_TOPUP|PEAK_BONUS|RECOVERY|WRITE_OFF|OTHER, amount, periodRef?, reason}` → Journal or 202 Approval (R30 peak bonus, R47/M6 minimum guarantee; ⚖ above threshold) |
| COD | `GET /admin/cod-deposits?status=DECLARED` | FIN, OPS | |
| | `POST /admin/cod-deposits/{id}/verify` / `reject` 🔑 | FIN | posts the journal; may lift `cod_blocked` |
| Coupons | `GET/POST /admin/coupons` 🔑, `PATCH /admin/coupons/{id}` 🔒, `POST /{id}/activate|pause|end` | OPS | `fundedBy: PLATFORM|RESTAURANT`; targets `ZONE|RESTAURANT` only (C16). Budgets audited, no ⚖ (R31). |
| Geography | `GET /admin/cities`, `POST /admin/cities` (SUPER), `PATCH /admin/cities/{id}` 🔒 | SUPER | `status: LIVE` gate |
| | `GET /admin/zones?cityId`, `POST /admin/zones` (GeoJSON geometry), `PUT /admin/zones/{id}` 🔒, `POST /admin/zones/{id}/activate|deactivate|pause|resume` | OPS | 16 §8 (`ZONE_GEOMETRY_INVALID`, `ZONE_OVERLAP`; save-time outlet check) |
| | `GET/POST/PATCH /admin/localities` | OPS | |
| Pricing | `GET /admin/fee-configs?cityId&zoneId`, `POST /admin/fee-configs` 🔑 ⚖ (new version; slab validator 16 §6.1), `POST /admin/fee-configs/{id}/retire` | OPS (maker), FIN (checker) | `VALIDATION_FAILED` (slab gap/overlap, radius × factor ≥ last slab) |
| | `GET /admin/tax-rules` | FIN, SUPER | read-only; rows are seeded by migration after CA sign-off (C20) [LEGAL] |
| Growth | `GET /admin/restaurant-leads?status&cursor`, `PATCH /admin/restaurant-leads/{id}` 🔒 | OPS | lead pipeline (`NEW→CONTACTED→CONVERTED|DISQUALIFIED`) |
| | `GET /admin/waitlist/summary?cityId` | OPS | counts by locality/PIN (no PII) for coverage decisions |
| Assisted orders (**P1**, M8; flag `ops_assisted_orders`) | `POST /admin/assisted-orders/quote` | OPS | `{customerPhone, customerName, address{…, landmark, location}, restaurantId, lines[]}` → Quote (as `/cart/quote`, `paymentMethod: COD`) |
| | `POST /admin/assisted-orders` 🔑 | OPS | `{quoteId, customerConsentNoticeVersion}` → Order (`placed_via=OPS_ASSISTED`, COD only; finds or creates the customer by phone; confirmation SMS) | `COD_*`, `QUOTE_*` |
| Support | `GET /admin/support/tickets?status&priority&category&assignee&cursor` | SUP, OPS | SLA ordering |
| | `GET /admin/support/tickets/{id}`, `PATCH` 🔒 (assign, priority, status), `POST /{id}/messages` (incl. internal notes), `POST /{id}/resolve` 🔑 (`{resolutionCode, faultParty, refund?, goodwill?}`) | SUP | |
| Payouts | `POST /admin/payout-runs` 🔑 ⚖ | FIN | `{cityId, payeeType, periodEnd}` → DRAFT batch `{batchId, payouts[], totals}`; approval releases it (R31 family 2). Normally created by the catch-up settlement job on the `payouts.*.schedule` day (10 §15.3); this endpoint re-runs or backfills. |
| | `GET /admin/payouts?batchId&status`, `GET /admin/payouts/{id}` | FIN | |
| | `POST /admin/payouts/{id}/mark-paid` 🔑 | FIN | `{utrReference, paidAt, method}` → journal posted |
| | `POST /admin/payouts/{id}/mark-failed` / `cancel` 🔑 | FIN | |
| Reconciliation | `GET /admin/pa-settlements?from&to`, `GET /admin/pa-settlements/{id}` (lines + match status) | FIN | 14 §16.1 |
| | `POST /admin/bank-statement-imports` 🔑 | FIN | `{fileId}` (CSV) → `pa_settlements(source=BANK_STATEMENT)`; matches UTRs to COD deposits, payouts and PA credits |
| | `GET /admin/recon-exceptions?status&type`, `PATCH /admin/recon-exceptions/{id}` 🔒 (assign/status/note) | FIN | write-off goes through `POST /admin/ledger-adjustments` ⚖ |
| Privacy | `GET /admin/erasure-requests?status`, `POST /admin/erasure-requests/{id}/hold` / `release` 🔑 | SUP, SUPER | DPDP queue (10 §13.1); audited, no ⚖ |
| Approvals | `GET /admin/approvals?status=PENDING&actionType`, `GET /admin/approvals/{id}` | role-dependent checker | five families only (R31) |
| | `POST /admin/approvals/{id}/approve` / `reject` 🔑 ⬆ | checker ≠ maker | `MAKER_CANNOT_APPROVE` |
| | `POST /admin/approvals/{id}/break-glass` 🔑 ⬆ | maker (when no checker is reachable) | self-approve with a mandatory reason; post-review due in 24 h (R31) |
| | `POST /admin/approvals/{id}/post-review` 🔑 | another approver | closes the break-glass review |
| Reports | `POST /admin/reports` 🔑 ⬆ (PII types) | OPS, FIN | `{type: ORDERS|SETTLEMENT_STATEMENT|GST_SUMMARY|RIDER_EARNINGS|COD_CASH|REFUNDS|GIG_WORKER_REGISTRATION, cityId, from, to}` → 202 `{reportExportId}`. PII reports need step-up + audit (no ⚖, R31). `GIG_WORKER_REGISTRATION` = M5 portal export [LEGAL]. |
| | `GET /admin/reports/{id}` | requester | `{status, downloadUrl (≤ 60 s presigned), rowCount}` |
| Audit | `GET /admin/audit-logs?resourceType&resourceId&actorId&action&from&to&cursor` | SUPER, (OPS own city read) | export ⬆ |
| Config | `GET/PUT /admin/feature-flags/{key}` 🔒, `GET/PUT /admin/app-config/{scopeType}/{scopeId}/{key}` 🔒, `GET /admin/reason-codes` | SUPER (flags/config), ADMIN* (read) | keys per 13 §5.1 / 10 §15.3 |

**V1.1 (removed from the V1 catalogue):** `PUT|DELETE /admin/zones/{id}/surge` (R30), `POST /admin/zones/import`, `GET /admin/zones/export` (C15), `GET /admin/geo/live` (C20), `POST /admin/tax-rules` (C20), `POST /admin/broadcasts` (C20).

### 2.7 Webhooks (`api.` host only)

| Method & path | Caller | Behaviour |
|---|---|---|
| **`POST /webhooks/payments/{provider}`** (`razorpay` / `cashfree` per R25) | PA | Verify the signature over the **raw body** before parsing. Persist to `payment_events`. Enqueue `payments.process_event` (River) in the same tx. Respond **200 within ~1 s**. Unknown event types: 200 + stored + ignored. Bad signature: 401, stored with `signature_verified=false`, not processed, rate-alerted (§3.3). |
| `POST /webhooks/notifications/{provider}` | SMS (and P1 voice) provider | Delivery receipts (DLR) → `notification_deliveries`; signature or shared-secret verified per provider (15) |

### 2.8 Real-time (SSE)

| Method & path | Roles | Notes |
|---|---|---|
| `GET /stream?topics=order:{id},inbox:{restaurantId},rider:{riderId},ops:{cityId}` | any session | One stream per tab. Topics default to the principal's entitlements, and `?topics=` narrows them. Authorised per topic at subscribe **and** at routing time (12 §4.7). Detail in §4. |

### 2.9 Counts

A mechanical count of §2.1–§2.8 (each HTTP verb and command variant counted once) gives **≈ 269 operations** in v1.1, against **≈ 242** for v1 by the same method (v1 rounded this to "~200"). By area: public ≈ 12, auth/me ≈ 27, customer ≈ 23, restaurant ≈ 57, rider ≈ 27, admin ≈ 119, webhooks 3, SSE 1. The v1.1 changes are −13 cut (surge ×2, zone import/export ×2, `geo/live`, tax-rule write, broadcasts, `ON_BREAK` and maker-checker variants) and ≈ +40 added (waitlist, leads, staff invites, device registration, ack, SOS, compensation choice, manual-refund mark-paid, ledger adjustments, menu import, assisted orders, recon/PA settlements/bank import, erasure queue, break-glass). The P1 items (assisted orders, SOS) are flagged. RV-008's suggested freeze at ≈ 110 golden-flow + ops-critical operations is **not** applied here (it is not in the accepted cut list); 27 sequences the build. Phase 2 implements in the order of the golden flow (27): auth → public catalog → quote → order → payment webhook → restaurant commands → dispatch/rider → settlement → rating. Admin endpoints are built as the flow needs them.

---

## 3. OpenAPI 3.1 excerpts: the six critical operations

These excerpts are design artifacts. Phase 2 moves them verbatim into `api/openapi.yaml` and completes the remaining operations to the same standard. Shared components come first, then each path.

```yaml
openapi: 3.1.0
info:
  title: rovo API
  version: 1.0.0-draft
  license: { name: Apache-2.0, identifier: Apache-2.0 }
servers:
  - url: https://app.rovo.in/api/v1          # [ASSUMPTION] domain; one server entry per host (R14, R27)
  - url: https://restaurant.rovo.in/api/v1
  - url: https://rider.rovo.in/api/v1
  - url: https://admin.rovo.in/api/v1
  - url: https://api.rovo.in/api/v1          # V1: webhooks only; future native bearer clients
security:
  - cookieAuth: []
  - bearerAuth: []

components:
  securitySchemes:
    cookieAuth: { type: apiKey, in: cookie, name: __Host-rovo_at }
    bearerAuth: { type: http, scheme: bearer, bearerFormat: JWT }

  parameters:
    IdempotencyKey:
      name: Idempotency-Key
      in: header
      required: true
      description: One UUIDv7 per user intent; reused on retries (§1.7).
      schema: { type: string, minLength: 8, maxLength: 128 }
    IfMatch:
      name: If-Match
      in: header
      required: false
      schema: { type: string, pattern: '^"v[0-9]+"$' }
    AcceptLanguage:
      name: Accept-Language
      in: header
      required: false
      schema: { type: string, example: "te-IN, en;q=0.8" }
    RestaurantId: { name: restaurantId, in: path, required: true, schema: { $ref: '#/components/schemas/Uuid' } }
    OrderId:      { name: orderId,      in: path, required: true, schema: { $ref: '#/components/schemas/Uuid' } }
    OfferId:      { name: offerId,      in: path, required: true, schema: { $ref: '#/components/schemas/Uuid' } }
    DeliveryId:   { name: deliveryId,   in: path, required: true, schema: { $ref: '#/components/schemas/Uuid' } }

  headers:
    ETag: { schema: { type: string, example: '"v3"' } }
    RateLimit: { schema: { type: string, example: '"default";r=37;t=21' } }
    IdempotentReplayed: { schema: { type: boolean } }

  responses:
    Problem400: { description: "Validation failed, content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }" }
    Problem401: { description: "Unauthenticated, content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }" }
    Problem404: { description: "Not found or not in scope, content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }" }
    Problem409: { description: "State conflict, content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }" }
    Problem422: { description: "Business rule violated, content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }" }
    Problem429: { description: "Rate limited, headers: { Retry-After: { schema: { type: integer } } }, content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }" }

  schemas:
    Uuid: { type: string, format: uuid }
    Timestamp: { type: string, format: date-time, example: "2026-10-04T13:05:12.345Z" }
    Currency: { type: string, enum: [INR] }
    Money:
      type: object
      additionalProperties: false
      required: [amountPaise, currency]
      properties:
        amountPaise: { type: integer, format: int64, examples: [38100] }
        currency: { $ref: '#/components/schemas/Currency' }
    LatLng:
      type: object
      additionalProperties: false
      required: [lat, lng]
      properties:
        lat: { type: number, minimum: -90, maximum: 90 }
        lng: { type: number, minimum: -180, maximum: 180 }
    I18nText:
      type: object
      description: Translations keyed by BCP-47 language (e.g. "te"). Canonical text is in the sibling field.
      additionalProperties: { type: string }
    FieldError:
      type: object
      required: [field, code]
      properties:
        field: { type: string, examples: ["lines[0].quantity"] }
        code: { type: string, examples: [OUT_OF_RANGE] }
        detail: { type: string }
    Problem:
      type: object
      description: RFC 9457 problem details with rovo extensions.
      required: [type, title, status, code, traceId]
      properties:
        type: { type: string, format: uri-reference, examples: ["/problems/quote-changed"] }
        title: { type: string }
        status: { type: integer }
        detail: { type: string }
        instance: { type: string }
        code: { type: string, description: "Stable machine-readable error code (§1.8)" }
        traceId: { type: string }
        retryable: { type: boolean }
        fieldErrors: { type: array, items: { $ref: '#/components/schemas/FieldError' } }
        currentStatus: { type: string }
        currentVersion: { type: integer }
        quote: { $ref: '#/components/schemas/Quote' }
        diff: { type: array, items: { $ref: '#/components/schemas/QuoteDiff' } }
      additionalProperties: true

    PaymentMethod: { type: string, enum: [ONLINE, COD] }
    OrderStatus:
      type: string
      enum: [PENDING_PAYMENT, PLACED, ACCEPTED, PREPARING, READY_FOR_PICKUP, PICKED_UP, DELIVERED,
             PAYMENT_FAILED, REJECTED, CANCELLED, UNDELIVERABLE]
    DeliveryStatus:
      type: string
      enum: [UNASSIGNED, OFFERED, ASSIGNED, AT_RESTAURANT, PICKED_UP, AT_DROP, DELIVERED, FAILED, CANCELLED]
    DeliveryOfferStatus:
      type: string
      enum: [PENDING, ACCEPTED, DECLINED, EXPIRED, REVOKED]     # REVOKED = withdrawn by system/admin (R16)
    VegType: { type: string, enum: [VEG, NON_VEG, EGG] }
    BillComponent:
      type: string
      enum: [ITEM_TOTAL, PACKAGING, DELIVERY_FEE, PLATFORM_FEE, SMALL_CART_FEE, DISCOUNT, TAX, ROUND_OFF]   # no SURGE_FEE (R30)

    # ---------- Quote ----------
    QuoteLineInput:
      type: object
      additionalProperties: false
      required: [menuItemId, quantity]
      properties:
        menuItemId: { $ref: '#/components/schemas/Uuid' }
        variantId: { oneOf: [ { $ref: '#/components/schemas/Uuid' }, { type: 'null' } ] }
        addonIds: { type: array, items: { $ref: '#/components/schemas/Uuid' }, maxItems: 30, default: [] }
        quantity: { type: integer, minimum: 1, maximum: 20 }   # register row 57
        note: { type: [string, 'null'], maxLength: 140 }
    QuoteRequest:
      type: object
      additionalProperties: false
      required: [restaurantId, lines, paymentMethod]
      properties:
        restaurantId: { $ref: '#/components/schemas/Uuid' }
        lines: { type: array, minItems: 1, maxItems: 50, items: { $ref: '#/components/schemas/QuoteLineInput' } }
        addressId: { $ref: '#/components/schemas/Uuid', description: "Required for an orderable quote (logged-in)" }
        location: { $ref: '#/components/schemas/LatLng', description: "Guest preview only (no addressId)" }
        paymentMethod: { $ref: '#/components/schemas/PaymentMethod' }
        couponCode: { type: [string, 'null'], pattern: '^[A-Za-z0-9]{4,24}$' }
    QuoteLine:
      type: object
      required: [lineNo, menuItemId, name, vegType, quantity, unitPrice, lineTotal, available]
      properties:
        lineNo: { type: integer }
        menuItemId: { $ref: '#/components/schemas/Uuid' }
        variantId: { oneOf: [ { $ref: '#/components/schemas/Uuid' }, { type: 'null' } ] }
        name: { type: string }
        nameI18n: { $ref: '#/components/schemas/I18nText' }
        variantName: { type: [string, 'null'] }
        addons: { type: array, items: { type: object, required: [addonId, name, price], properties: { addonId: { $ref: '#/components/schemas/Uuid' }, name: { type: string }, price: { $ref: '#/components/schemas/Money' } } } }
        vegType: { $ref: '#/components/schemas/VegType' }
        quantity: { type: integer }
        unitPrice: { $ref: '#/components/schemas/Money' }
        lineTotal: { $ref: '#/components/schemas/Money' }
        available: { type: boolean }
    BillLine:
      type: object
      required: [component, label, amount, isIncluded]
      properties:
        component: { $ref: '#/components/schemas/BillComponent' }
        label: { type: string, examples: ["GST on food (CGST 2.5%)"] }
        amount: { $ref: '#/components/schemas/Money', description: "Negative for DISCOUNT / negative ROUND_OFF" }
        isIncluded: { type: boolean, description: "true = informational 'incl. GST' line, not added to payable" }
        tax:
          type: [object, 'null']
          properties:
            type: { type: string, enum: [CGST, SGST, IGST, CESS] }
            rateBps: { type: integer }
            taxedComponent: { $ref: '#/components/schemas/BillComponent' }
        fundedBy: { type: [string, 'null'], enum: [PLATFORM, RESTAURANT, null] }
    Bill:
      type: object
      required: [lines, payable]
      properties:
        lines: { type: array, items: { $ref: '#/components/schemas/BillLine' } }
        payable: { $ref: '#/components/schemas/Money' }
        savings: { $ref: '#/components/schemas/Money' }
    QuoteIssue:
      type: object
      required: [code, blocking]
      properties:
        code: { type: string, examples: [ITEM_UNAVAILABLE, PRICE_CHANGED, MIN_ORDER_NOT_MET, COUPON_NOT_APPLICABLE, SMALL_CART_FEE_APPLIES] }
        blocking: { type: boolean }
        lineNo: { type: [integer, 'null'] }
        message: { type: string }
        params: { type: object, additionalProperties: true, examples: [{ "addMorePaise": 1200 }] }
    PaymentOption:
      type: object
      required: [method, available]
      properties:
        method: { $ref: '#/components/schemas/PaymentMethod' }
        available: { type: boolean }
        reasonCode: { type: [string, 'null'], examples: [COD_LIMIT_EXCEEDED, COD_DISABLED_FOR_CUSTOMER] }
    Quote:
      type: object
      required: [quoteId, expiresAt, isOrderable, restaurantId, lines, bill, eta, paymentOptions, issues]
      properties:
        quoteId: { type: [string, 'null'], description: "Signed opaque id 'q1.<uuid>.<mac>'; null for guest previews", examples: ["q1.0192f3a17c2e7b4d9a105e7c1d2f3a4b.Zk9xY2V3cXJ0eXVp"] }
        expiresAt: { $ref: '#/components/schemas/Timestamp' }
        isOrderable: { type: boolean }
        restaurantId: { $ref: '#/components/schemas/Uuid' }
        menuVersion: { type: integer }
        lines: { type: array, items: { $ref: '#/components/schemas/QuoteLine' } }
        bill: { $ref: '#/components/schemas/Bill' }
        distanceM: { type: integer, description: "Estimated road distance used for the fee slab (16 §4; serviceability uses straight-line, R18)" }
        eta: { type: object, required: [minMinutes, maxMinutes], properties: { minMinutes: { type: integer }, maxMinutes: { type: integer } } }
        coupon:
          type: [object, 'null']
          properties:
            code: { type: string }
            status: { type: string, enum: [APPLIED, NOT_APPLICABLE, INVALID, EXPIRED, USAGE_EXCEEDED] }
            discount: { $ref: '#/components/schemas/Money' }
            message: { type: string }
        paymentOptions: { type: array, items: { $ref: '#/components/schemas/PaymentOption' } }
        issues: { type: array, items: { $ref: '#/components/schemas/QuoteIssue' } }
    QuoteDiff:
      type: object
      required: [change]
      properties:
        lineNo: { type: [integer, 'null'] }
        menuItemId: { oneOf: [ { $ref: '#/components/schemas/Uuid' }, { type: 'null' } ] }
        change: { type: string, enum: [PRICE_CHANGED, ITEM_UNAVAILABLE, FEE_CHANGED, COUPON_CHANGED, TOTAL_CHANGED] }
        from: { $ref: '#/components/schemas/Money' }
        to: { $ref: '#/components/schemas/Money' }

    # ---------- Orders ----------
    CreateOrderRequest:
      type: object
      additionalProperties: false
      required: [quoteId]
      properties:
        quoteId: { type: string }
        customerNote: { type: [string, 'null'], maxLength: 200, description: "For the kitchen" }
        deliveryInstructions: { type: [string, 'null'], maxLength: 200, description: "For the rider" }
    OrderSummary:
      type: object
      required: [id, code, status, restaurant, total, paymentMethod, createdAt]
      properties:
        id: { $ref: '#/components/schemas/Uuid' }
        code: { type: string, pattern: '^RV-[0-9A-HJKMNP-TV-Z]{6}$', examples: [RV-7K3P9Q] }
        status: { $ref: '#/components/schemas/OrderStatus' }
        restaurant: { type: object, properties: { id: { $ref: '#/components/schemas/Uuid' }, name: { type: string }, nameI18n: { $ref: '#/components/schemas/I18nText' } } }
        total: { $ref: '#/components/schemas/Money' }
        paymentMethod: { $ref: '#/components/schemas/PaymentMethod' }
        createdAt: { $ref: '#/components/schemas/Timestamp' }
    Order:
      allOf:
        - $ref: '#/components/schemas/OrderSummary'
        - type: object
          required: [version, paymentStatus, items, bill, timeline, cancellable]
          properties:
            version: { type: integer }
            paymentStatus: { type: string, enum: [PENDING, PAID, COD_DUE, COD_COLLECTED, COD_NOT_COLLECTED, PARTIALLY_REFUNDED, REFUNDED, UNPAID] }
            items: { type: array, items: { $ref: '#/components/schemas/QuoteLine' } }
            bill: { $ref: '#/components/schemas/Bill' }
            deliveryAddress: { type: object, additionalProperties: true }
            etaAt: { oneOf: [ { $ref: '#/components/schemas/Timestamp' }, { type: 'null' } ] }
            delivery:
              type: [object, 'null']
              properties:
                status: { $ref: '#/components/schemas/DeliveryStatus' }
                riderFirstName: { type: [string, 'null'] }
            timeline:
              type: array
              items: { type: object, required: [status, at], properties: { status: { $ref: '#/components/schemas/OrderStatus' }, at: { $ref: '#/components/schemas/Timestamp' } } }
            cancellable:
              type: object
              required: [allowed]
              properties:
                allowed: { type: boolean }
                freeUntil: { oneOf: [ { $ref: '#/components/schemas/Timestamp' }, { type: 'null' } ] }
                reasonCodes: { type: array, items: { type: string } }
            cancelReasonCode: { type: [string, 'null'] }
            deliveryCode:
              type: [string, 'null']
              pattern: '^[0-9]{4}$'
              description: "Handover code to read to the rider (R39). Set for prepaid orders with payable ≥ dispatch.delivery_code_min_payable_paise; null for COD. Returned to the ordering customer only."
            placedVia: { type: string, enum: [CUSTOMER_APP, OPS_ASSISTED] }
    PaymentSession:
      type: object
      description: Data the client needs to open the PA checkout (Razorpay Checkout shape; 14 owns details).
      required: [paymentId, provider, providerOrderId, amount, expiresAt, checkout]
      properties:
        paymentId: { $ref: '#/components/schemas/Uuid' }
        provider: { type: string, enum: [RAZORPAY, CASHFREE, PHONEPE, FAKE] }
        providerOrderId: { type: string, examples: [order_PZx1abcDEF] }
        amount: { $ref: '#/components/schemas/Money' }
        expiresAt: { $ref: '#/components/schemas/Timestamp' }
        checkout:
          type: object
          additionalProperties: true
          description: Provider-specific public params (e.g. keyId, name, prefill.contact, theme). Never secrets.
    CreateOrderResponse:
      type: object
      required: [order, payment]
      properties:
        order: { $ref: '#/components/schemas/Order' }
        payment: { oneOf: [ { $ref: '#/components/schemas/PaymentSession' }, { type: 'null' } ] }
        paymentRetryable: { type: boolean, description: "true when the PA was unreachable; call POST /orders/{id}/payments" }

    # ---------- Restaurant ----------
    AcceptOrderRequest:
      type: object
      additionalProperties: false
      required: [prepTimeMinutes]
      properties:
        prepTimeMinutes: { type: integer, minimum: 5, maximum: 90, examples: [20], description: "05 §4.3 chips (5,10,15,20,25,30,45,60,90); server accepts 5..90 (ordering.prep_time_min_range, 13 §5.1)" }
    PartnerOrder:
      type: object
      required: [id, code, status, version, placedAt, acceptBy, items, customerFirstName, earnings]
      properties:
        id: { $ref: '#/components/schemas/Uuid' }
        code: { type: string }
        status: { $ref: '#/components/schemas/OrderStatus' }
        version: { type: integer }
        placedAt: { $ref: '#/components/schemas/Timestamp' }
        acceptBy: { $ref: '#/components/schemas/Timestamp', description: "placedAt + 180 s (ruling 1)" }
        prepTimeMinutes: { type: [integer, 'null'] }
        readyBy: { oneOf: [ { $ref: '#/components/schemas/Timestamp' }, { type: 'null' } ] }
        items: { type: array, items: { $ref: '#/components/schemas/QuoteLine' } }
        customerFirstName: { type: string }
        customerNote: { type: [string, 'null'] }
        paymentMethod: { $ref: '#/components/schemas/PaymentMethod' }
        rider:
          type: [object, 'null']
          properties:
            firstName: { type: string }
            deliveryStatus: { $ref: '#/components/schemas/DeliveryStatus' }
            etaToRestaurantMinutes: { type: [integer, 'null'] }
        earnings:
          type: object
          description: What the restaurant earns from this order (05 §4.6)
          properties:
            foodValue: { $ref: '#/components/schemas/Money' }
            commission: { $ref: '#/components/schemas/Money' }
            commissionGst: { $ref: '#/components/schemas/Money' }
            tds: { $ref: '#/components/schemas/Money' }
            net: { $ref: '#/components/schemas/Money' }

    # ---------- Rider ----------
    RiderDelivery:
      type: object
      required: [id, orderId, orderCode, status, version, pickup, drop, cod, milestones]
      properties:
        id: { $ref: '#/components/schemas/Uuid' }
        orderId: { $ref: '#/components/schemas/Uuid' }
        orderCode: { type: string }
        status: { $ref: '#/components/schemas/DeliveryStatus' }
        version: { type: integer }
        pickup:
          type: object
          properties:
            restaurantName: { type: string }
            address: { type: string }
            landmark: { type: [string, 'null'] }
            location: { $ref: '#/components/schemas/LatLng' }
            phone: { type: string }
            orderStatus: { $ref: '#/components/schemas/OrderStatus' }
            itemCount: { type: integer }
        drop:
          type: object
          description: Customer details are revealed only after assignment and only while active (12 §9).
          properties:
            customerFirstName: { type: string }
            houseNo: { type: string }
            buildingStreet: { type: [string, 'null'] }
            landmark: { type: string }
            locality: { type: [string, 'null'] }
            pinCode: { type: string }
            location: { $ref: '#/components/schemas/LatLng' }
            contactPhoneMasked: { type: string }
            instructions: { type: [string, 'null'] }
        cod:
          type: object
          required: [required]
          properties:
            required: { type: boolean }
            amount: { $ref: '#/components/schemas/Money' }
        requiresDeliveryCode: { type: boolean }
        estimatedEarnings: { $ref: '#/components/schemas/Money' }
        distanceM: { type: integer }
        milestones:
          type: array
          items: { type: object, properties: { status: { $ref: '#/components/schemas/DeliveryStatus' }, at: { $ref: '#/components/schemas/Timestamp' } } }
        undeliverableRequest:
          type: [object, 'null']
          properties: { ticketId: { $ref: '#/components/schemas/Uuid' }, reasonCode: { type: string }, requestedAt: { $ref: '#/components/schemas/Timestamp' } }
    DeliveryMilestoneRequest:
      type: object
      additionalProperties: false
      required: [clientEventId, occurredAt, location]
      properties:
        clientEventId: { $ref: '#/components/schemas/Uuid', description: "Equals Idempotency-Key; lets offline-queued milestones dedupe (13 §7.2)" }
        occurredAt: { $ref: '#/components/schemas/Timestamp', description: "Device time; server clamps to ±5 min of receipt" }
        location:
          allOf:
            - $ref: '#/components/schemas/LatLng'
            - type: object
              properties: { accuracyM: { type: integer, minimum: 0 } }
        expectedVersion: { type: integer, description: "Optional CAS guard (alternative to If-Match)" }
    DeliveredRequest:
      allOf:
        - $ref: '#/components/schemas/DeliveryMilestoneRequest'
        - type: object
          properties:
            codCollected: { $ref: '#/components/schemas/Money', description: "Required when cod.required; must equal cod.amount" }
            deliveryCode: { type: string, pattern: '^[0-9]{4}$' }
    PickedUpRequest:
      allOf:
        - $ref: '#/components/schemas/DeliveryMilestoneRequest'
        - type: object
          properties:
            pickupPin: { type: string, pattern: '^[0-9]{4}$', description: "Only when pickup_pin_required (05)" }

paths:
  # 1 ─────────────────────────────────────────────────────────── QUOTE
  /cart/quote:
    post:
      operationId: createQuote
      tags: [Checkout]
      summary: Price a device cart for an address; returns a signed short-lived quoteId
      x-rovo-permission: cart.quote
      x-rovo-audiences: [customer, customer_native]
      security: [ { cookieAuth: [] }, { bearerAuth: [] }, {} ]   # anonymous → guest preview (quoteId null)
      parameters: [ { $ref: '#/components/parameters/AcceptLanguage' } ]
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/QuoteRequest' }
            example:
              restaurantId: 0192e8b2-5d1a-7c33-8f10-3a2b1c0d9e8f
              addressId: 0192e8b2-6a00-7aaa-9bbb-1c2d3e4f5a6b
              paymentMethod: ONLINE
              couponCode: WELCOME50
              lines:
                - { menuItemId: 0192e8b2-7001-7000-8000-000000000001, variantId: 0192e8b2-7001-7000-8000-0000000000f1, addonIds: [], quantity: 2 }
                - { menuItemId: 0192e8b2-7001-7000-8000-000000000002, addonIds: [0192e8b2-7001-7000-8000-0000000000a1], quantity: 1 }
      responses:
        '200':
          description: Quote (may contain non-blocking or blocking issues; isOrderable tells the client whether to enable Pay)
          headers: { RateLimit: { $ref: '#/components/headers/RateLimit' } }
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Quote' }
              example:
                quoteId: q1.0192f3a17c2e7b4d9a105e7c1d2f3a4b.Zk9xY2V3cXJ0eXVp
                expiresAt: '2026-10-04T13:15:00.000Z'
                isOrderable: true
                restaurantId: 0192e8b2-5d1a-7c33-8f10-3a2b1c0d9e8f
                menuVersion: 41
                distanceM: 2800
                eta: { minMinutes: 35, maxMinutes: 45 }
                lines: []            # priced lines omitted for brevity
                bill:
                  lines:
                    - { component: ITEM_TOTAL,   label: Item total,         amount: { amountPaise: 36000, currency: INR }, isIncluded: false }
                    - { component: PACKAGING,    label: Packaging,          amount: { amountPaise: 2000,  currency: INR }, isIncluded: false }
                    - { component: DELIVERY_FEE, label: Delivery fee (2.8 km), amount: { amountPaise: 3000, currency: INR }, isIncluded: false }
                    - { component: TAX, label: "incl. CGST 9% on delivery", amount: { amountPaise: 229, currency: INR }, isIncluded: true, tax: { type: CGST, rateBps: 900, taxedComponent: DELIVERY_FEE } }
                    - { component: TAX, label: "incl. SGST 9% on delivery", amount: { amountPaise: 229, currency: INR }, isIncluded: true, tax: { type: SGST, rateBps: 900, taxedComponent: DELIVERY_FEE } }
                    - { component: PLATFORM_FEE, label: Platform fee,       amount: { amountPaise: 500,   currency: INR }, isIncluded: false }
                    - { component: DISCOUNT,     label: WELCOME50,          amount: { amountPaise: -5000, currency: INR }, isIncluded: false, fundedBy: PLATFORM }
                    - { component: TAX, label: CGST 2.5% on food,  amount: { amountPaise: 825, currency: INR }, isIncluded: false, tax: { type: CGST, rateBps: 250, taxedComponent: ITEM_TOTAL } }
                    - { component: TAX, label: SGST 2.5% on food,  amount: { amountPaise: 825, currency: INR }, isIncluded: false, tax: { type: SGST, rateBps: 250, taxedComponent: ITEM_TOTAL } }
                    - { component: ROUND_OFF,    label: Round off,          amount: { amountPaise: -50,   currency: INR }, isIncluded: false }
                  payable: { amountPaise: 38100, currency: INR }
                  savings: { amountPaise: 5000, currency: INR }
                coupon: { code: WELCOME50, status: APPLIED, discount: { amountPaise: 5000, currency: INR }, message: "₹50 off applied" }
                paymentOptions:
                  - { method: ONLINE, available: true, reasonCode: null }
                  - { method: COD, available: true, reasonCode: null }
                issues: []
        '400': { $ref: '#/components/responses/Problem400' }
        '404': { $ref: '#/components/responses/Problem404' }
        '422': { $ref: '#/components/responses/Problem422' }   # OUTSIDE_SERVICE_AREA, ZONE_PAUSED, TOO_FAR, RESTAURANT_CLOSED, CART_EMPTY
        '429': { $ref: '#/components/responses/Problem429' }
```

> Example arithmetic: 36,000 + 2,000 + 3,000 + 500 − 5,000 + 825 + 825 = 38,150 → ROUND_OFF −50 → **38,100** payable. The GST inside the delivery fee (₹30 × 18/118 ≈ ₹4.58, split CGST/SGST) is informational (`isIncluded`) under ruling 8. Whether the food GST base should be net of the platform-funded discount is `[OPEN — CA]`; the example taxes 33,000 (36,000 − 5,000 + 2,000 packaging). `tax.type` and rates are data from `tax_rules`, not hard-coded.

```yaml
  # 2 ─────────────────────────────────────────────────────────── CREATE ORDER
  /orders:
    post:
      operationId: createOrder
      tags: [Orders]
      summary: Place an order from a quote (ONLINE → PENDING_PAYMENT + checkout session; COD → PLACED)
      description: |
        Re-validates the quote (13 O-01/O-02). Idempotent per Idempotency-Key. If the PA is unreachable
        after the order is committed, returns 201 with payment=null and paymentRetryable=true.
      x-rovo-permission: order.create
      x-rovo-audiences: [customer, customer_native]
      x-rovo-state-transition: [O-01, O-02]
      parameters:
        - $ref: '#/components/parameters/IdempotencyKey'
        - $ref: '#/components/parameters/AcceptLanguage'
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/CreateOrderRequest' }
            example: { quoteId: q1.0192f3a17c2e7b4d9a105e7c1d2f3a4b.Zk9xY2V3cXJ0eXVp, customerNote: "Less spicy please", deliveryInstructions: "Call at the gate" }
      responses:
        '201':
          description: Order created
          headers:
            Location: { schema: { type: string }, description: "/api/v1/orders/{id}" }
            ETag: { $ref: '#/components/headers/ETag' }
            Idempotent-Replayed: { $ref: '#/components/headers/IdempotentReplayed' }
          content:
            application/json:
              schema: { $ref: '#/components/schemas/CreateOrderResponse' }
        '400': { $ref: '#/components/responses/Problem400' }      # IDEMPOTENCY_KEY_REQUIRED, VALIDATION_FAILED
        '401': { $ref: '#/components/responses/Problem401' }
        '409':
          description: QUOTE_CHANGED | QUOTE_EXPIRED (problem includes new `quote` and `diff`) | QUOTE_ALREADY_USED | REQUEST_IN_PROGRESS
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }
        '422':
          description: COD_NOT_AVAILABLE | COD_LIMIT_EXCEEDED | COD_DISABLED_FOR_CUSTOMER | COUPON_* | RESTAURANT_CLOSED | RESTAURANT_PAUSED | ZONE_PAUSED | ITEM_UNAVAILABLE | IDEMPOTENCY_KEY_REUSED
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }
        '429': { $ref: '#/components/responses/Problem429' }

  # 3 ─────────────────────────────────────────────────────────── PAYMENT WEBHOOK
  /webhooks/payments/{provider}:
    post:
      operationId: receivePaymentWebhook
      tags: [Webhooks]
      summary: PA event receiver (api host only); Razorpay shown, Cashfree adapter equivalent (R25)
      description: |
        1. Read the raw body (≤ 256 KB) BEFORE JSON parsing; verify X-Razorpay-Signature =
           hex(HMAC-SHA256(webhook_secret, raw_body)) in constant time [ASSUMPTION — header names/algorithm
           per Razorpay docs; 14 verifies]. Support two active secrets during rotation.
        2. Insert payment_events (payload PII-redacted at ingest, raw body to the 180-day object prefix;
           provider_event_id = X-Razorpay-Event-Id; unique only when verified) and
           InsertTx River job payments.process_event in the same tx; duplicates → 200 without re-enqueue.
        3. Respond 200 fast (< 1 s). Business processing (payment CAS, O-03, refunds, late-capture refund)
           happens in the job. Events handled: payment.authorized, payment.captured, payment.failed,
           order.paid, refund.processed, refund.failed (others stored and ignored).
      security: []                 # authenticated by signature, not session
      x-rovo-permission: webhook.payments
      x-rovo-audiences: [webhook]
      parameters:
        - { name: provider, in: path, required: true, schema: { type: string, enum: [razorpay, cashfree, fake] } }
        - { name: X-Razorpay-Signature, in: header, required: false, description: "Required when provider=razorpay", schema: { type: string } }
        - { name: X-Razorpay-Event-Id, in: header, required: false, schema: { type: string } }
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [event, payload]
              additionalProperties: true
              properties:
                entity: { type: string, examples: [event] }
                account_id: { type: string }
                event: { type: string, examples: [payment.captured] }
                contains: { type: array, items: { type: string } }
                payload: { type: object, additionalProperties: true }
                created_at: { type: integer }
      responses:
        '200': { description: "Accepted (stored; processing async), content: { application/json: { schema: { type: object, properties: { received: { type: boolean } } } } }" }
        '400': { description: "Malformed body / too large" }
        '401': { description: "WEBHOOK_SIGNATURE_INVALID (stored with signature_verified=false, alerted)" }
        '503': { description: "DB unavailable — provider retries" }

  # 4 ─────────────────────────────────────────────────────────── RESTAURANT ACCEPT
  /partner/restaurants/{restaurantId}/orders/{orderId}/accept:
    post:
      operationId: acceptRestaurantOrder
      tags: [Partner Orders]
      summary: Accept a PLACED order with a prep time (13 O-06)
      x-rovo-permission: order.accept
      x-rovo-audiences: [partner, partner_native]
      x-rovo-state-transition: [O-06]
      parameters:
        - $ref: '#/components/parameters/RestaurantId'
        - $ref: '#/components/parameters/OrderId'
        - $ref: '#/components/parameters/IdempotencyKey'
        - $ref: '#/components/parameters/IfMatch'
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/AcceptOrderRequest' }
            example: { prepTimeMinutes: 20 }
      responses:
        '200':
          description: Accepted (also returned on an idempotent replay or if this same command was already applied)
          headers: { ETag: { $ref: '#/components/headers/ETag' } }
          content: { application/json: { schema: { $ref: '#/components/schemas/PartnerOrder' } } }
        '404': { $ref: '#/components/responses/Problem404' }   # not this restaurant's order
        '409':
          description: ACCEPT_WINDOW_CLOSED (auto-cancelled at 180 s) | ORDER_INVALID_TRANSITION (currentStatus in body) | ORDER_STATE_CONFLICT
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }
        '412': { description: "VERSION_MISMATCH, content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }" }
        '422': { $ref: '#/components/responses/Problem422' }   # prepTimeMinutes out of range

  # 5 ─────────────────────────────────────────────────────────── RIDER ACCEPT OFFER
  /rider/offers/{offerId}/accept:
    post:
      operationId: acceptDeliveryOffer
      tags: [Rider]
      summary: Accept a pending delivery offer within its 45 s window (13 D-03)
      description: |
        Locks delivery → rider_availability → offer (13 §7.1). Re-checks offer PENDING and not expired
        (2 s grace), rider AVAILABLE, COD headroom (cash_in_hand + cod ≤ limit). Same rider repeating → 200.
      x-rovo-permission: delivery_offer.respond
      x-rovo-audiences: [partner, partner_native]
      x-rovo-state-transition: [D-03]
      parameters:
        - $ref: '#/components/parameters/OfferId'
        - $ref: '#/components/parameters/IdempotencyKey'
      responses:
        '200':
          description: Assigned; full delivery details now visible to the rider
          content: { application/json: { schema: { $ref: '#/components/schemas/RiderDelivery' } } }
        '404': { $ref: '#/components/responses/Problem404' }   # not this rider's offer
        '409':
          description: OFFER_EXPIRED | OFFER_NOT_PENDING (offer DECLINED or REVOKED, e.g. manual assign / order cancelled; currentStatus in body)
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }
        '422':
          description: RIDER_CASH_LIMIT_REACHED | RIDER_NOT_ONLINE | RIDER_NOT_ELIGIBLE
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }

  # 6 ─────────────────────────────────────────────────────────── DELIVERY STATUS UPDATES
  /rider/deliveries/{deliveryId}/picked-up:
    post:
      operationId: markDeliveryPickedUp
      tags: [Rider]
      summary: Rider collected the food (13 D-08/D-08b; order O-14/O-15 via event)
      x-rovo-permission: delivery.progress
      x-rovo-audiences: [partner, partner_native]
      x-rovo-state-transition: [D-08, D-08b]
      parameters:
        - $ref: '#/components/parameters/DeliveryId'
        - $ref: '#/components/parameters/IdempotencyKey'
        - $ref: '#/components/parameters/IfMatch'
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/PickedUpRequest' }
            example: { clientEventId: 0192f3b0-0000-7000-8000-00000000c0de, occurredAt: '2026-10-04T13:31:02.000Z', location: { lat: 16.7489, lng: 78.0036, accuracyM: 18 } }
      responses:
        '200': { description: "Updated, content: { application/json: { schema: { $ref: '#/components/schemas/RiderDelivery' } } }" }
        '404': { $ref: '#/components/responses/Problem404' }
        '409':
          description: DELIVERY_INVALID_TRANSITION (e.g. order still ACCEPTED, or delivery cancelled) | ORDER_STATE_CONFLICT
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }
        '422': { $ref: '#/components/responses/Problem422' }   # pickup PIN invalid
  /rider/deliveries/{deliveryId}/delivered:
    post:
      operationId: markDeliveryDelivered
      tags: [Rider]
      summary: Handover complete (13 D-10/D-10b; order O-17 via event; ledger, rider pay, cash limit)
      x-rovo-permission: delivery.progress
      x-rovo-audiences: [partner, partner_native]
      x-rovo-state-transition: [D-10, D-10b]
      parameters:
        - $ref: '#/components/parameters/DeliveryId'
        - $ref: '#/components/parameters/IdempotencyKey'
        - $ref: '#/components/parameters/IfMatch'
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/DeliveredRequest' }
            example:
              clientEventId: 0192f3b0-0000-7000-8000-00000000d0de
              occurredAt: '2026-10-04T13:49:40.000Z'
              location: { lat: 16.7752, lng: 78.0349, accuracyM: 22 }
              codCollected: { amountPaise: 38100, currency: INR }
      responses:
        '200': { description: "Delivered, content: { application/json: { schema: { $ref: '#/components/schemas/RiderDelivery' } } }" }
        '404': { $ref: '#/components/responses/Problem404' }
        '409':
          description: DELIVERY_INVALID_TRANSITION | UNDELIVERABLE_REQUEST_OPEN
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }
        '422':
          description: COD_AMOUNT_MISMATCH | DELIVERY_CODE_INVALID (attempts remaining in body)
          content: { application/problem+json: { schema: { $ref: '#/components/schemas/Problem' } } }
  # arrived-at-restaurant, arrived-at-drop and undeliverable-requests follow the same pattern
  # with DeliveryMilestoneRequest (+ reasonCode/note for undeliverable-requests).
```

---

## 4. Real-time: Server-Sent Events

### 4.1 Endpoint and lifecycle

`GET /api/v1/stream?topics=order:0192f…,inbox:0192e…` with `Accept: text/event-stream`.

| Aspect | Rule |
|---|---|
| Auth | Session cookie (web) or bearer (native). **No tokens in URLs.** |
| Topics | Defaults by principal: customer → `user:{id}` + their active `order:{id}`s; restaurant host → `inbox:{restaurantId}` (requires `ctx` match); rider host → `rider:{riderId}` (offers, delivery); admin → `ops:{cityId}` for the cities in scope. `?topics=` may only narrow. Unauthorised topic → `403 FORBIDDEN` before the stream opens. |
| Frames | `id: <entityType>:<entityId>:<version>`, `event: <name>`, `data: <JSON, ≤ 4 KB>`. First frame `retry: 3000`. |
| Heartbeat | `: ping` comment every **20 s** (R10, R52; under the ≥ 120 s LB/CDN idle timeout). For restaurant devices, an open stream counts as device presence (R27). |
| Lifetime | The stream **continues past access-token expiry** while the server-side session is valid (the session is re-checked every 5 min [ASSUMPTION] and at routing). The server closes it after **30 min** with `event: reconnect` for rebalancing (jittered). Session revocation closes immediately with `event: revoked`. One rule everywhere (RV-011, register row 29). |
| Reconnect | Client backoff 1 s → 2 s → 5 s → 10 s with 2–10 s jitter on mass reconnects (RV-038). `Last-Event-ID` is accepted but used **only to drop stale duplicates**. **No server replay buffer.** On every (re)connect the server first sends `event: ready {topics, serverTime}` and the client **refetches snapshots** over REST (TanStack Query invalidation) (ruling 10). |
| Degraded | If the server's LISTEN connection is down → `event: degraded`. Clients poll (`GET /orders/{id}` every 10 s; restaurant inbox every 5 s; rider offers every 5 s) until `event: ready`. |
| Limits | 3 streams per session, 5 per user, 50 per IP. Beyond that, `429 TOO_MANY_STREAMS`. |
| Compression/buffering | No gzip on the stream. `X-Accel-Buffering: no`, `Cache-Control: no-store`. HTTP/2. |

### 4.2 Event types and payloads (thin; see 13 §8 for the domain events behind them)

| `event:` | Topic | `data` |
|---|---|---|
| `ready` | all | `{"topics": ["order:…"], "serverTime": "…"}` |
| `order.status` | `order:{id}`, `inbox:{rid}` | `{"orderId","status","version","at","etaAt","cancelReasonCode"}` |
| `order.eta` | `order:{id}` | `{"orderId","etaAt","version"}` |
| `delivery.milestone` | `order:{id}` | `{"orderId","deliveryStatus","riderFirstName","at"}` |
| `payment.status` | `order:{id}` | `{"orderId","paymentStatus","refund": {"id","status","amount"}|null}` |
| `order.placed` | `inbox:{rid}` | `{"orderId","code","placedAt","acceptBy","itemCount","total"}`. Triggers the ring (05 §4.3). |
| `order.cancelled` | `inbox:{rid}` | `{"orderId","code","reasonCode"}`. Triggers the "do not prepare" alert. |
| `order.rider_assigned` / `order.rider_arrived` | `inbox:{rid}` | `{"orderId","riderFirstName","at"}` |
| `restaurant.status` | `inbox:{rid}` | `{"restaurantId","acceptingOrders","pausedUntil","pauseReason"}` |
| `offer.new` | `rider:{id}` | `{"offerId","expiresAt","pickup":{"name","locality","distanceM"},"drop":{"locality","distanceM"},"estimatedEarnings","cod"}` |
| `offer.revoked` | `rider:{id}` | `{"offerId","status","reason"}` (`status`: `EXPIRED|REVOKED`, R16) |
| `delivery.updated` | `rider:{id}` | `{"deliveryId","status","orderStatus","version"}` (e.g. food ready, order cancelled) |
| `account.cash_limit` | `rider:{id}` | `{"cashInHand","limit","codBlocked"}` |
| `inbox.new` | `user:{id}` | `{"notificationId","category","unreadCount"}` |
| `ops.alert` | `ops:{cityId}` | `{"alertType": "ACCEPT_SLA|DISPATCH_EXHAUSTED|READY_NOT_PICKED|DELIVERY_LATE|UNDELIVERABLE_REQUEST|STATE_DRIFT|DEVICE_OFFLINE|STUCK_ORDER|SOS|MISSED_SETTLEMENT|BREAK_GLASS_REVIEW_DUE","orderId","restaurantId","riderId","severity","at"}` |
| `ops.order_updated` | `ops:{cityId}` | `{"orderId","status","deliveryStatus","version","flags":[]}` |
| `reconnect` / `revoked` / `degraded` | all | `{}` |

Example frame:

```
id: order:0192f3a1-7c2e-7b4d-9a10-5e7c1d2f3a4b:4
event: order.status
data: {"orderId":"0192f3a1-7c2e-7b4d-9a10-5e7c1d2f3a4b","status":"PREPARING","version":4,"at":"2026-10-04T13:06:12.000Z","etaAt":"2026-10-04T13:45:00.000Z","cancelReasonCode":null}

```

### 4.3 Web Push (background)

- `POST /me/push-subscriptions` registers a W3C Push subscription (VAPID; public key in `/config/client`).
- Payloads are **encrypted and minimal**: `{"t": "order.status", "id": "<orderId>", "title", "body", "deepLink"}`. Never addresses or phone numbers.
- Push is sent **in addition to** SSE for: new order (restaurant; repeated every `ordering.accept_ring_interval_s` until acknowledged via `…/ack` or accepted, R1), new offer (rider; `Urgency: high`, TTL = offer TTL; the only channel that reaches tier-2 riders, R34), order accepted/picked up/delivered (customer), ops alerts (admin). Rules live in 15.

---

## 5. Not doing in V1

- **GraphQL, gRPC, WebSockets.** REST + SSE meet V1 needs (P2, P3).
- **HATEOAS links**, except `Location` on creates.
- **Bulk/batch endpoints**, except menu sort order and location pings.
- **Public partner API / API keys for third parties** (POS integrations). Later, on `api.` with OAuth client credentials.
- **Server-side cart** (ruling 12).
- **Surge endpoints and fields** (R30), **zone GeoJSON import/export** (C15), **`geo/live`, broadcasts, tax-rule CRUD** (C20), **WhatsApp OTP** (C2), **review replies/moderation queue** (C14). All V1.1+.
- **CORS on any host** (R27): the API is same-origin on every app host.
- **SSE replay** (ruling 10).
- **Field selection / sparse fieldsets.** Payloads are small.
- **ETags on list endpoints.**

## 6. Open items

- `[OPEN — 14]` Exact Razorpay webhook headers/events and signature scheme. Client-side `payments/confirm` fields. Callback behaviour (12 §4.4).
- `[OPEN — CA]` Inclusive fee GST presentation (ruling 8); GST base after platform-funded discounts.
- `[OPEN — 12]` Rider access to the customer phone (masked relay vs. direct) — the `contactPhoneMasked` field is a placeholder.
- `[OPEN — 21]` Spec linting (Spectral) and breaking-change detection (oasdiff) tooling.
- ~~`[OPEN — 17]` `nameI18n` instead of `name_te`~~ — resolved by R17.
- `[OPEN — Legal]` Gig-worker registration field list (M5) and the COD manual-refund flow wording (R29).
- `[ASSUMPTION]` Rate-limit budgets in §1.10.

## 7. Sources (accessed 2026-10-04)

- RFC 9457 Problem Details for HTTP APIs (IETF, 2023): https://www.rfc-editor.org/rfc/rfc9457
- RFC 8594 `Sunset` header; RFC 9745 `Deprecation` header (2025) — standard references, not re-fetched [ASSUMPTION — verify RFC 9745 number].
- IETF draft "RateLimit header fields for HTTP" (draft-ietf-httpapi-ratelimit-headers) — still a draft; header syntax may change.
- HTML Living Standard, Server-sent events (`Last-Event-ID`, `retry`): https://html.spec.whatwg.org/multipage/server-sent-events.html
- OpenAPI 3.1.0 (JSON Schema 2020-12 alignment, `type: [x, 'null']`): https://spec.openapis.org/oas/v3.1.0
