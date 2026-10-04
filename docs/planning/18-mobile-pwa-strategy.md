# 18 — Mobile & PWA Strategy

| | |
|---|---|
| **Purpose** | Define how rovo's four web apps behave as installable PWAs on the devices our users actually have. Covers manifests, install, service-worker caching, offline rules, updates, Web Push, restaurant order-alert reliability, rider constraints, payments, calls and navigation hand-offs, and the path to Google Play (TWA) and later to native apps. Includes the device/browser support matrix and the low-end device test plan. |
| **Owner** | Frontend Architect |
| **Status** | Draft v1 (2026-10-04) |
| **Depends on** | `00-planning-baseline.md` (P3, P7, P9, P12, §4a), `17-frontend-architecture.md` (apps, origins, SSE client, auth), `05-restaurant-workflow.md` / `06-delivery-workflow.md` (alert and rider flows), `08-system-architecture.md` (SSE + Web Push fan-out, escalation), `12-auth-rbac.md` (cookies, bearer for native), `13-order-state-machine.md` (which transitions may be queued), `14-payment-architecture.md` (PA checkout, return URLs), `15-notification-architecture.md` (VAPID, push payloads, SMS/voice escalation), `20-testing-strategy.md` (device lab) |

Tags: `[ASSUMPTION]`, `[OPEN]`, `[LEGAL]`. All web sources were accessed **2026-10-04**.

---

## 0. Key constraints that shape product scope

1. **A PWA cannot track location in the background.** Rider location is sent only while the rider app is open and visible. That is enough for dispatch (P12). It is **not** enough for a live map for customers, so customers see **status milestones only** in V1.
2. **Restaurant order alerts are only fully reliable while the restaurant app is open and in the foreground** on a plugged-in device. Web Push is a backup channel, and server-side SMS/voice escalation is mandatory. **Restaurant onboarding must include device setup** (§6).
3. **Web notifications cannot play a custom looping sound.** A loud repeating alert is possible only in the open app.
4. **No offline ordering or payment.** Customers can browse cached menus read-only. Riders may queue a small set of forward-only status updates (§4.3).
5. **iOS:** Web Push works only after the user adds the app to the Home Screen (iOS 16.4+). Wake Lock in installed iOS web apps works only from iOS 18.4. iOS is a minority platform in Mahabubnagar [ASSUMPTION; Safari ≈ 4.3% of Indian mobile browser share ([Statcounter, Jul 2026](https://gs.statcounter.com/browser-market-share/mobile/india))]. **Partner apps are Android-first; iOS is best effort.**
6. **Play Store presence for partner apps** is cheap via a Trusted Web Activity (TWA). It does not add background capabilities. **Native apps are the V2 answer** for live GPS and alarm-grade alerts (§10).

---

## 1. Per-app PWA profile

| | customer | restaurant | rider | admin |
|---|---|---|---|---|
| Installable | Yes (optional, prompted after value) | **Yes (required in onboarding)** | **Yes (required in onboarding)** | No (plain SPA) |
| Service worker | Yes | Yes | Yes | **No** (no offline need; avoids caching sensitive data; simpler updates) |
| `display` | `standalone` | `standalone` | `standalone` | — |
| `orientation` | `portrait` | `any` (tablets in landscape) | `portrait` | — |
| Web Push | Order milestones (opt-in after first order) | **New orders (critical)** | **New offers (critical)**, cancellations | — (admins use desktop and the live board; email/SMS for ops alerts) |
| Wake Lock | No | **Yes**, while on shift | **Yes**, while online / on a delivery | — |
| Geolocation | Pin-drop only (on tap) | No | **Yes**, foreground only | — |
| TWA / Play | No (V1) | Yes (phase 1b) | Yes (phase 1b) | No |

### 1.1 Manifest (example: rider)

```json
{
  "id": "/?app=rider",
  "name": "rovo Delivery Partner",
  "short_name": "rovo Rider",
  "description": "Accept and deliver rovo orders",
  "start_url": "/?source=pwa",
  "scope": "/",
  "display": "standalone",
  "orientation": "portrait",
  "lang": "en-IN",
  "dir": "ltr",
  "theme_color": "#0F766E",
  "background_color": "#FFFFFF",
  "categories": ["food", "business"],
  "icons": [
    { "src": "/icons/192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icons/512.png", "sizes": "512x512", "type": "image/png" },
    { "src": "/icons/maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ],
  "shortcuts": [{ "name": "Go online", "url": "/?action=online" }, { "name": "Earnings", "url": "/earnings" }],
  "screenshots": [{ "src": "/screens/home.png", "sizes": "1080x1920", "type": "image/png", "form_factor": "narrow" }],
  "prefer_related_applications": false
}
```

- Each app has its **own origin** (doc 17 §15.1), so each has its own manifest `id`, icon, push subscription and storage. Icons and names make the apps easy to tell apart on a shared phone (e.g. "rovo" orange, "rovo Partner" blue, "rovo Rider" teal [ASSUMPTION – brand TBD]).
- Manifests are not localisable per user. `name` stays brand-led ("rovo …"), which works in both languages. In-app text follows the user's language.
- `screenshots` enable Chrome's richer install sheet on Android.

---

## 2. Install prompts

| Platform | Mechanism | rovo UX |
|---|---|---|
| Android Chrome / Samsung Internet / Edge | `beforeinstallprompt` → we call `preventDefault()` and keep the event, then call `prompt()` from our own button. Chrome installs a **WebAPK**: it appears in the app drawer and has its own Android notification settings. [ASSUMPTION: WebAPK minting requires Google Play Services; on devices without it, a shortcut is created instead.] | **Customer:** never on first visit. Shown as a dismissible card (a) on the order-tracking page after the first order, or (b) on the 3rd visit. After 2 dismissals it is hidden for 30 days. **Restaurant / rider:** a mandatory onboarding step "Install rovo Partner", with a fallback instruction (⋮ → *Install app* / *Add to Home screen*) if the event did not fire. Completion is detected by `display-mode: standalone`, which the server records as a device-readiness check. |
| iOS Safari 16.4+ | No install API. Users must do Share → *Add to Home Screen*. | An instruction sheet with an illustration, shown **only in iOS Safari** (not in in-app browsers; there it says "Open in Safari") and only when push or wake lock is needed. For customers it is optional. For partner apps it is required on iOS. |
| In-app browsers (WhatsApp, Instagram, Facebook) | Install and push are usually unavailable [ASSUMPTION]. | Detect by user-agent heuristics and show "Open in Chrome" for partner apps. Customers can still browse and order in the in-app browser. Payment intents generally work [ASSUMPTION – test in §12]. |
| Opera Mini (extreme/data-saving mode) | Proxy rendering with limited JS [ASSUMPTION]. | `<noscript>` and capability check → "Please open in Chrome" page, which is static, < 10 KB, and bilingual. |

---

## 3. Service worker caching strategy (customer, restaurant, rider)

Built with **vite-plugin-pwa in `injectManifest` mode**: we write our own `sw.ts` with Workbox modules, because we need push and notification-click handlers plus custom routes.

| Resource | Strategy | Details |
|---|---|---|
| App shell: `index.html`, hashed JS/CSS of the shell and main routes, icons, Latin UI SVGs | **Precache** (Workbox manifest, revisioned) | Customer precache budget ≤ 400 KB gzip in total. Admin-like heavy chunks are not precached. |
| Telugu font WOFF2 | **Runtime CacheFirst** (`fonts`, max 4 entries, 1 year) | Downloaded only when needed (doc 17 §9). Telugu-first users get it cached after the first view. |
| Lazy route chunks (e.g. MapLibre, address editor) | **Runtime CacheFirst** (`chunks`, max 30, 30 days) | **Not** precached, so we do not spend ~200+ KB of user data on a map they may never open. Hashed names make CacheFirst safe. |
| SPA navigations (`mode: navigate`) | **NavigationRoute → precached `index.html`** | Instant repeat loads, and offline-capable. Denylist: `/api/`, `/media/`, `/.well-known/`. |
| Prerendered public pages (customer `/{city}`, `/{city}/r/{slug}`) | **NetworkFirst, 3 s timeout** → cache → shell | Crawlers get fresh HTML. Users offline get the last copy. |
| Public catalog API: `GET /api/v1/public/**` (restaurant list, menus, localities) | **StaleWhileRevalidate** (`catalog`, max 60 entries, maxAge 24 h) | Enables read-only offline browsing. The UI marks data older than 5 min as "may be outdated" while offline. **Prices are always re-quoted** at checkout (doc 17 §4.3). |
| Menu/restaurant images `/media/**` | **CacheFirst** (`images`, max 200 entries, 7 days, `purgeOnQuotaError`) | Immutable, content-hashed URLs. |
| Map tiles `/tiles/**` (PMTiles range requests) | **No SW caching** [ASSUMPTION: Workbox handles HTTP Range responses poorly; rely on the HTTP cache] | Map is used rarely. |
| Authenticated API (`/api/v1/**` other than `public`) | **NetworkOnly** (not cached by the SW) | Contains PII and per-user state. TanStack Query holds it in memory only. |
| **Never cached in any way** | `/api/v1/auth/**`, `/api/v1/payments/**`, `/api/v1/orders` POST, `/api/v1/stream` (SSE), `/api/v1/me`, any non-GET, any response with `Cache-Control: no-store`, and third-party PA domains (Razorpay script, iframe, intents) | Enforced by explicit Workbox routes **and** a unit test over the SW route table. |

**Storage hygiene.**
- Call `navigator.storage.persist()` only for restaurant and rider apps (lower eviction risk).
- Clean old caches by name version on SW activate.
- On logout: delete the `catalog` and `images` caches on partner apps (shared devices), clear IndexedDB stores, and clear Zustand stores (doc 17 §6).

---

## 4. Offline behaviour

### 4.1 Customer

| State | Behaviour |
|---|---|
| Offline on launch | The shell loads from the SW. Banner: "You're offline – showing saved restaurants". Cached list and menus are browsable read-only. Images show if cached, otherwise a dominant-colour placeholder. |
| Cart | Adding to the cart works offline, because the cart is client-side (doc 17 §4.3). **Checkout is disabled** with "Connect to the internet to place your order". |
| Order tracking | Shows the last known status with "Last updated 14:02". The SSE client reconnects on `online` and refetches. |
| **No offline ordering** | Availability, prices, serviceability and payment all need the server. Queuing an order to send later would create orders the customer did not intend at that later time. |

### 4.2 Restaurant

- Offline means the restaurant **may be missing orders**. The app shows a full-width red banner, plays a short distinct "connection lost" tone every 60 s (if audio is unlocked), and keeps retrying. The health indicator turns red.
- **Actions are not queued** (accept, reject, mark ready, stock toggles). They are time-sensitive and depend on state that may have changed (the customer cancelled, or the system auto-rejected). Buttons are disabled while offline, and a failed request shows "Not sent – Retry".
- **Server side** (requirement on docs 08/15): the presence of the restaurant's SSE stream is tracked. If an outlet has **no live stream for > 3 min during opening hours** [ASSUMPTION], ops are alerted. After > 10 min, the outlet is **auto-paused** (stops receiving new orders) with an SMS to the owner. A restaurant cannot silently "accept orders" while unreachable.

### 4.3 Rider: queued status updates (decision)

Riders lose signal in lifts, basements and dead zones near drop points. Blocking every step on connectivity would stall deliveries. We therefore allow a **narrow, idempotent outbox**:

| Action | Queue offline? | Reason |
|---|---|---|
| `AT_RESTAURANT`, `PICKED_UP`, `AT_DROP` | **Yes** | These are forward-only transitions in the delivery state machine. Each carries `Idempotency-Key`, `occurred_at` (device time), `last_known_location` and `expected_from_status`. |
| `DELIVERED` (prepaid, no proof requirement) | **Yes** | Same as above. |
| `DELIVERED` with **COD cash collected** | **Yes, with care** | The cash amount is recorded with it. The rider's cash-in-hand counter is shown as "pending sync". The server ledgers it on arrival. |
| `DELIVERED` requiring a **delivery OTP / server check** [OPEN – doc 06/13 decide whether a delivery OTP exists] | **No** | Needs the server. Show "Waiting for network to verify code". |
| Accept / decline offer | **No** | 45 s expiry. An offline accept is meaningless. |
| Go online / offline | **No** (go-offline is applied locally and retried) | Going online requires a fresh location and server-side eligibility (cash limit). |
| Location pings | **Not queued**; only the latest is sent on reconnect | Stale positions are useless for dispatch. |
| `FAILED` / undeliverable with reason | **No** | Triggers customer contact and refund flows. Needs a live confirmation dialog. |

Rules:
- The outbox lives in **IndexedDB** with strict FIFO per delivery. Each item has `{key, deliveryId, transition, occurred_at, location, attempts}`.
- **Flush triggers:** the `online` event, `visibilitychange`, app start, a 15 s timer while non-empty, and the **Background Sync API** where available (Chrome/Samsung; **not on iOS Safari** ([caniuse](https://caniuse.com/background-sync))).
- **Server contract** (requirement on docs 11/13):
  - Idempotent by key.
  - Accept if `expected_from_status` matches, or if the transition was already applied (returns 200 with the current state).
  - Reject with `409` if the delivery was cancelled or reassigned. The client then drops the item and shows a blocking notice.
  - Clamp `occurred_at` to `[assigned_at, server_now]` and store both device and server times for audit.
- While the outbox is non-empty, the rider **cannot accept new offers** ("Syncing 2 updates…"), and the UI shows a sync badge. This keeps dispatch consistent: the server never offers a job to a rider whose last delivery it thinks is still in progress.
- Customer-facing milestones may arrive late. That is acceptable, and it is better than a stalled rider.

---

## 5. Update flow

- `registerType: 'prompt'`. The SW checks for updates on start, on `visibilitychange → visible`, and every 60 min.
- **When a new SW is waiting:**

  | App | Behaviour |
  |---|---|
  | Customer | Non-blocking toast "New version available · Reload". **Never auto-reload during checkout, payment handoff, or the address editor.** |
  | Restaurant | Auto-apply at a **safe point**: no unacknowledged order alert, no open dialog, idle ≥ 30 s. Otherwise the toast persists. Kitchen staff often never tap toasts, so auto-apply is needed. |
  | Rider | Auto-apply only when **offline (not on shift) or idle on the home screen with no active delivery and an empty outbox**. Never mid-delivery. |

- **Forced update:** if `min_supported_version` from `/api/v1/config/client` (doc 17 §13.4) exceeds the running version, show a blocking "Update required" screen whose button calls `skipWaiting` and reloads. Used for breaking API changes. **API changes must stay backward compatible for ≥ 2 frontend releases** (requirement on doc 11).
- **Kill switch for a broken SW:** the deploy pipeline can publish a `sw.js` that calls `self.registration.unregister()` and clears caches (a "self-destroying SW"). The error boundary's "Reload app" button escalates to this on repeated crashes.
- Old hashed assets are kept for 2 releases (doc 17 §15.4), so a page on an old SW can still lazy-load chunks.

---

## 6. Web Push

### 6.1 Mechanics

- **VAPID** key pair. The private key is in the cloud secrets manager, used by the worker's push sender (Go, RFC 8291 encryption, e.g. `webpush-go` [ASSUMPTION]). The public key is in runtime config (doc 17 §15.2).
- **APIs (requirement on docs 11/15):**
  - `POST /api/v1/push-subscriptions` with body `{ endpoint, keys: { p256dh, auth }, app: "customer"|"restaurant"|"rider", locale, ua_hint, device_label? }`. Idempotent on `endpoint`.
  - `DELETE /api/v1/push-subscriptions/{id}`, or by endpoint on logout.
  - `POST /api/v1/push-subscriptions/test` sends a test notification to this device (used by the alert self-test).
  - The server deletes subscriptions on push-service `404`/`410`. It sets `TTL` and `Urgency`:

    | Message | TTL | Urgency |
    |---|---|---|
    | Rider offers | 45 s | `high` |
    | Restaurant new order | 300 s | `high` |
    | Customer milestones | 1 h | `normal` |

- **Payload** (≤ 3 KB): `{ type, title, body, url, tag, entity_id, version }`, localised by the server using the subscription's `locale`.
  - SW `push` handler: always calls `showNotification` (Chrome requires `userVisibleOnly`). It sets `tag` for coalescing (e.g. `order-{id}`), `renotify: true` for restaurant/rider, `requireInteraction: true` for restaurant new orders and rider offers, and `vibrate` where supported.
  - If a **focused client** exists, the SW also `postMessage`s it, and the app shows its in-app alert instead. The OS notification is shown anyway, to satisfy the platform rule; it is quietly closed when the app acknowledges.
  - `notificationclick` focuses an existing window or opens `url`. Actions (e.g. "Accept") are **not** offered in notifications in V1. Acceptance needs the full order view, and action-button support varies.

### 6.2 Permission UX timing

| App | When we ask |
|---|---|
| Customer | **After placing the first order**, on the tracking page: a soft pre-prompt "Get updates when your food is on the way?" → native prompt only on "Yes". Never on first load. Chrome may show its quiet permission UI for sites with low acceptance, which is another reason to ask only at a moment of clear value. |
| Restaurant / rider | During onboarding: a mandatory checklist step with an explanation, then the native prompt, then a test notification. If denied, a "How to enable" guide (Chrome site settings → Notifications) is shown, and the device-readiness check stays red. |
| iOS | Only in the Home Screen web app, and only from a button tap: "a web app that has been added to the Home Screen can request permission to receive push notifications" in response to direct user interaction ([WebKit, 2023-02-16](https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/)). |

### 6.3 Platform behaviour

- **Android Chrome:** push is delivered via FCM even when the browser is closed. Delivery can still be **delayed by Doze and OEM battery managers**; brands common in India (Xiaomi/Redmi, Vivo, Oppo, Realme) are aggressive here [ASSUMPTION – widely reported]. Since 2025, Chrome on Android also enforces stricter Web Push spam handling ([Netcore changelog, 2025-08-04](https://updates.netcorecloud.com/changelog/google-chromes-crackdown-on-web-push-notifications-android)).
- Chrome's **automatic revocation** of notification permission for low-engagement sites **does not apply to installed web apps** ([9to5Google](https://9to5google.com/?p=691969)). This is another reason partner apps must be installed.
- **Custom notification sounds are not available to web notifications** [ASSUMPTION: the `sound` option is not implemented in Chromium]. The restaurant's long alert therefore lives in the open app (§7). In a WebAPK or TWA, the user can raise the notification channel's importance and sound in Android settings, and the onboarding guide shows how.

---

## 7. Restaurant order-alert reliability (layered design)

```mermaid
flowchart TD
  P[Order PLACED] --> S{Restaurant SSE live?}
  S -- yes --> A[In-app alert loop:<br/>loud audio + flashing banner + vibrate<br/>repeat until Accept/Reject]
  S -- no --> W[Web Push high urgency<br/>requireInteraction, renotify]
  A --> K{Acknowledged ≤ 60 s?}
  W --> K
  K -- yes --> Done[Accepted / Rejected]
  K -- no --> R1[Repeat push at 60 s, 120 s]
  R1 --> K2{Ack ≤ 2 min?}
  K2 -- no --> SMS[SMS to outlet phone<br/>DLT template]
  SMS --> K3{Ack ≤ 3–4 min?}
  K3 -- no --> CALL[Automated voice call (IVR)<br/>+ admin live-board alert for manual call]
  CALL --> K4{Ack ≤ SLA (e.g. 5–7 min)?}
  K4 -- no --> AUTO[Auto-reject/cancel per doc 13;<br/>customer refunded/notified; outlet auto-paused]
```

Timings are illustrative. Docs 05/13/15 own the policy [OPEN].

### 7.1 In-app alert (primary path)

- **"Start shift" gesture.** At the start of each session (and after any reload) staff tap **Start shift**. This single gesture:
  1. Unlocks audio by creating/resuming an `AudioContext` and playing a silent buffer.
  2. Requests the **Screen Wake Lock**.
  3. Checks notification permission.
  4. Connects SSE.

  Chrome's autoplay policy allows sound after user interaction with the site, and also when the user "has added the site to their home screen on mobile" ([Chrome autoplay policy](https://developer.chrome.com/blog/autoplay)). We still require the tap so behaviour does not depend on install state or browser quirks.
- **Alert loop:** a 2–3 s attention sound (bundled, ~30 KB, precached) every 4 s via Web Audio, with a full-screen red overlay (order count, items summary, Accept/Reject), `navigator.vibrate` pattern, and document title flashing. It stops **only** on acknowledgement (accept/reject by any staff device, propagated by SSE `order.*` events) or when the server cancels. Volume is advisory: the app cannot raise system volume, so the settings page includes a volume check.
- **Wake Lock:**
  - Acquired on Start shift.
  - Re-acquired on `visibilitychange → visible` (the browser releases it when hidden).
  - Status shown in the header.
  - Support: Chrome Android, Samsung Internet 14+, Safari iOS 16.4+ ([caniuse](https://caniuse.com/wake-lock)). In **iOS Home Screen web apps** it only works from **iOS 18.4** ([WebKit bug 254545](https://bugs.webkit.org/show_bug.cgi?id=254545)).
  - Fallback where unsupported: instruct the user to set "Screen timeout: never while charging" or use the Android developer option "Stay awake".
- **Keep the tab foregrounded.** Background tabs and apps are throttled or frozen by the browser and OS [ASSUMPTION: Chrome intensive timer throttling and Android process freezing]. The device should be **dedicated, plugged in, with the rovo Partner app on screen**. The restaurant workflow (doc 05) makes this an onboarding agreement.

### 7.2 Device setup checklist (onboarding + `/settings/alerts` self-test)

| Check | How verified |
|---|---|
| App installed (standalone) | `matchMedia('(display-mode: standalone)')` |
| Notifications allowed + test push received | `/push-subscriptions/test` round trip; the user taps the notification |
| Sound unlocked + audible | User confirms they heard the test tone |
| Wake lock active | API state |
| SSE connected, heartbeat < 30 s old | Health store |
| Charger connected | Battery Status API (`navigator.getBattery`) where available (Chrome Android) [ASSUMPTION]; otherwise manual confirmation |
| Battery optimisation off for Chrome / rovo Partner | Manual, with brand-specific illustrated steps (Xiaomi, Samsung, Vivo, Oppo, Realme), referencing dontkillmyapp.com-style guidance [ASSUMPTION] |
| Do Not Disturb off, media volume ≥ 70% | Manual confirmation |
| Outlet phone number verified for SMS/call escalation | Server |

The server stores a **device-readiness score** per outlet, which admins see during activation. [OPEN – doc 05: an outlet cannot go live unless all mandatory checks pass.]

### 7.3 Multiple devices

Several staff devices may be logged into the same outlet. All of them alert. The first acknowledgement stops the alert everywhere via SSE, and the server makes accept/reject idempotent with `If-Match` versioning (doc 08).

---

## 8. Rider constraints

| Constraint | Consequence / design |
|---|---|
| **Geolocation only while the page is visible.** `watchPosition` stops delivering, or is throttled, when the app is hidden or the screen is off [ASSUMPTION – consistent with Chrome/Android behaviour; verify in §12]. | Location pings (every 30–60 s, P12) are sent only in the foreground. While online, the rider app keeps the **wake lock** and asks the rider to keep the app on screen (phone mount + charger recommended). |
| Stale location | The server treats a location older than N min (e.g. 3 [ASSUMPTION]) as stale. A stale rider is deprioritised or excluded from offers (doc 08 dispatch). On becoming visible again, the app sends an immediate fix. |
| Navigation | Navigating opens Google Maps (an external app), so rovo goes to the background. **Our pings stop while Maps is in front.** We accept this in V1: status milestones (picked up, at drop) are manual taps, and dispatch only needs location between deliveries. |
| Battery | High-accuracy GPS + screen on drains the battery. We use `enableHighAccuracy: true` only on the "go online" fix and the at-restaurant/at-drop taps, and `false` with `maximumAge: 30000` for periodic pings. The app warns below 20% battery. |
| Offers while backgrounded | Web Push with `Urgency: high`, `TTL: 45 s`. Tapping opens `/offers/{id}`. If the offer has expired, the app says so. |
| Permission denied | The rider cannot go online. An instructions screen explains how to enable location. |
| **Why live GPS tracking is deferred** | A web app has no background location, no foreground service, and no guaranteed delivery while another app (Maps) is in front. A customer-facing live map would freeze whenever the rider navigates, which is exactly when customers look. Live tracking requires a native app (§10). |

---

## 9. Payments, calls and navigation hand-offs

### 9.1 Payments (UPI intent from mobile web)

- **Checkout:** the PA's standard web checkout (Razorpay lean, doc 14), loaded lazily on `/checkout/pay/{orderId}`. The PA supports **UPI Intent on mobile web (Android and iOS)**. It lists UPI apps (GPay > PhonePe > Paytm > BHIM, plus "other apps") and hands off to the chosen app ([Razorpay UPI Intent](https://razorpay.com/docs/payments/payment-methods/upi/upi-intent)). We do **not** build our own `upi://pay` links. That keeps payment collection inside the licensed PA (P10) and gets PA-side reconciliation.
- **Return handling** (low-RAM phones often kill the browser tab while the UPI app is in front):
  1. Before handing off, the order exists in `PENDING_PAYMENT` and the `orderId` is in the URL. Nothing important lives only in memory.
  2. On return, or on any reload of `/checkout/pay/{orderId}` or `/orders/{orderId}`, the page fetches `GET /api/v1/orders/{id}/payment-status` and subscribes to SSE. It shows "Confirming your payment…", polling every 3 s for up to 2 min, then every 10 s.
  3. **The truth comes from the PA webhook plus a server-side fetch** (docs 12/14). The client-side success callback only navigates; it never marks an order paid.
  4. The PA's redirect/`callback_url` mode (a cross-site POST) lands on an unauthenticated, idempotent return endpoint that redirects to the order page (doc 12 §4.4).
  5. The `PENDING_PAYMENT` expiry (e.g. 15 min, doc 13) shows "Payment not completed – Retry or choose Cash on Delivery".
- The SW never caches payment routes or PA domains (§3). CSP allows the PA domains on the customer app only (doc 17 §15.6).
- **TWA note (future):** UPI intents launched from a TWA open the UPI app the same way as in Chrome [ASSUMPTION – verify in the Phase 2 spike]. The customer app is not a TWA in V1 anyway.

### 9.2 Phone calls

- `tel:` links (`<a href="tel:+91XXXXXXXXXX">`) to call the restaurant, rider or customer, as large 48 px buttons.
- Number masking or call bridging (so riders and customers do not see each other's real numbers) is [OPEN – docs 12/15; a privacy improvement and a paid line item]. The frontend shows whatever number the API returns: real or masked.

### 9.3 Navigation deep links

- The rider app opens Google Maps with a universal URL:
  `https://www.google.com/maps/dir/?api=1&destination={lat},{lng}&travelmode=two-wheeler&dir_action=navigate`.
  - Maps URLs need no API key.
  - `two-wheeler` is a documented `travelmode` that "works only if you're physically located in the country that supports two-wheelers"; India does [ASSUMPTION – documented list not fetched].
  - `dir_action=navigate` starts turn-by-turn ([Google Maps URLs](https://developers.google.com/maps/documentation/urls/get-started)).
- A fallback link to a plain map pin (`/maps/search/?api=1&query={lat},{lng}`) is offered. The address text and landmark are always shown on our screen, because Indian addresses depend on landmarks.

---

## 10. Native path

### 10.1 Step 1b: Trusted Web Activity (Bubblewrap) for restaurant and rider apps

| Aspect | Plan |
|---|---|
| What | Wrap the **same PWA** in a TWA Android package with **Bubblewrap** (or PWABuilder, which uses it). Full-screen Chrome, no URL bar, verified by **Digital Asset Links** (`/.well-known/assetlinks.json` with the Play app-signing SHA-256) ([web.dev](https://web.dev/articles/using-a-pwa-in-your-android-app), [Android TWA guide](https://developer.android.com/develop/ui/views/layout/webapps/guide-trusted-web-activities-version2)). |
| Gains | Play Store listing and trust ("download from Play" is how partners expect apps). App-attributed notifications via **notification delegation**: web push is shown by the TWA app and governed by *its* Android notification permission ([TrustedWebActivityService](https://developer.android.com/reference/androidx/browser/trusted/TrustedWebActivityService)). Users can set channel importance and sound. Updates ship with the web deploy, with no Play re-release for content. |
| Does **not** gain | Background location, foreground services, custom alarm sounds or full-screen intents. The engine is still Chrome with the same lifecycle. |
| Cost | Google Play developer registration: **one-time US$25** [ASSUMPTION – secondary sources]. Use an **organisation account** (needs a D-U-N-S number, free, can take days to weeks). New **personal** accounts created after 2023-11-13 must run a closed test with **12 testers for 14 days** before production access; organisation accounts are exempt ([Play Console help](https://support.google.com/googleplay/android-developer/answer/14151465)). |
| Effort | ~2–3 days per app (signing, assetlinks, store listing, privacy policy, Data safety form [LEGAL]). The listing must disclose location use for the rider app. |
| When | After the PWA pilot is stable (milestone owned by doc 28). The customer app stays web-only in V1. |

### 10.2 Criteria for going native (V2 triggers)

Go native for **rider** (and possibly restaurant) when **any** of these hold:
1. Product commits to **customer live tracking**. This requires background location with an Android foreground service and the Play "background location" declaration with prominent disclosure [LEGAL/Play policy].
2. Restaurant missed-order rate due to device/browser issues stays above the target (e.g. > 2% of orders reaching the SMS escalation stage [ASSUMPTION]) after the setup programme.
3. Rider offer acceptance is hurt by delayed push (median offer-to-seen latency > 10 s [ASSUMPTION]).
4. A need for hardware or OS integrations: Bluetooth KOT printers, NFC, background uploads, call masking via SDK.
5. More than 1 city or more than ~100 active riders, where per-rider productivity gains pay for the native team [ASSUMPTION].

Native capabilities that matter:
- **FCM high-priority data messages** with our own notification channel and **custom sound**.
- A **foreground service** for an "on shift" restaurant or "online" rider.
- Background location.
- Note on full-screen intents: on Android 14+, `USE_FULL_SCREEN_INTENT` is granted by default **only to calling/alarm apps**; others must ask the user and complete a Play declaration ([Play Console help](https://support.google.com/googleplay/android-developer/answer/13392821), [AOSP FSI limits](https://source.android.com/docs/core/permissions/fsi-limits)). Do not plan the restaurant alert around full-screen intents. Plan it around a high-importance channel plus a foreground service.

### 10.3 React Native (Expo) vs Flutter

| Criterion | React Native + Expo | Flutter |
|---|---|---|
| Code reuse with web | **High.** It reuses `packages/api-client` (openapi-fetch, bearer adapter), `domain`, `utils`, i18n catalogs and zod schemas directly. | Low. Dart client generated from the same OpenAPI (`openapi-generator` `dart-dio`), but domain logic and i18n must be ported. |
| Team skills | Same TypeScript/React team | New language and framework |
| Background location / foreground service | `expo-location` + `expo-task-manager` (background), or community libraries; config plugins | `geolocator` / `flutter_background_service`; mature |
| Push | `expo-notifications` or `@react-native-firebase/messaging` (FCM high priority, channels, custom sounds) | `firebase_messaging` (first-party) |
| Low-end Android performance | Good with the New Architecture and Hermes [ASSUMPTION]. Needs discipline on list rendering. | Very good, consistent rendering |
| Build/distribution | EAS Build (free tier limited) or local Gradle builds in CI | Standard Gradle |
| **Recommendation** | **Choose React Native + Expo** for rider (then restaurant), mainly for TS package reuse and a single team. | Re-evaluate only if the team composition changes. |

### 10.4 How V1 prepares for native

- **One OpenAPI contract** (P2) generates the TS client (web and RN) and, if ever needed, Dart. Native-only endpoints are not allowed; the same REST is used.
- **Auth:** the API already supports **bearer tokens on `api.<domain>`** (refresh in the body, stored in Android Keystore / iOS Keychain), with the same rotation and reuse detection (doc 12). `packages/api-client` has a `bearer` transport adapter from day one.
- **DOM-free packages** (`utils`, `domain`, `api-client` core, i18n catalogs) are linted with an `no-restricted-globals: window, document` rule.
- **Push abstraction on the server:** `push_subscriptions` gets a `kind` column (`webpush` | `fcm` | `apns`) now, so FCM tokens fit later without schema changes [requirement on doc 10/15].
- **Deep-link paths are stable** (`/orders/{id}`, `/offers/{id}`, `/deliveries/{id}`) and are reused as Android App Links.
- **Idempotent, offline-tolerant rider transitions** (§4.3) are the same contract a native app needs.
- **SSE works in RN** via a fetch/XHR-based client, so no WebSocket migration is needed.

---

## 11. Device and browser support matrix

| Tier | Platform / browser | Apps | Notes |
|---|---|---|---|
| **Supported (tested every release)** | Android 10+ with **Chrome ≥ 120** (evergreen) | all mobile apps | Chrome 139+ requires Android 10; Chrome 138 was the last for Android 8/9 ([Business Standard](https://www.business-standard.com/technology/tech-news/google-chrome-drop-old-android-devices-support-check-compatibility-125062700607_1.html)). Chrome ≈ 88% of Indian mobile browsing ([Statcounter, Jul 2026](https://gs.statcounter.com/browser-market-share/mobile/india)). |
| Supported | Samsung Internet ≥ 23 (Android) | all mobile apps | Push, wake lock (14+) and Background Sync supported ([caniuse wake-lock](https://caniuse.com/wake-lock), [background-sync](https://caniuse.com/background-sync)). Small share (≈ 0.6%), so a smoke test per release. |
| Supported (customer) / best effort (partner) | **iOS/iPadOS Safari ≥ 16.4** | customer; partner best effort | Push only when installed to the Home Screen. Wake lock in installed apps needs iOS ≥ 18.4. No Background Sync. |
| Best effort | Android 8/9 on Chrome 109–138 (no more updates) | customer, rider | Works if features are present (our build target is ES2020). Not in the release test cycle. A banner suggests updating if a crash occurs. |
| Best effort | Opera (Chromium) and Edge on Android; in-app browsers (WhatsApp/Instagram/Facebook) | customer | Opera ≈ 4.7% share. Install/push may be unavailable in in-app browsers. |
| Unsupported (graceful message) | Opera Mini extreme mode, UC Browser legacy, KaiOS, Android ≤ 7 | — | Static "Open in Chrome" page (§2). |
| **Admin** | Latest 2 versions of desktop Chrome, Edge, Firefox; Safari best effort; min viewport 1280 px | admin | Not supported on phones beyond read-only views [ASSUMPTION]. |

Capability detection, not UA sniffing, gates features (`'wakeLock' in navigator`, `'PushManager' in window`, `'serviceWorker' in navigator`). The partner onboarding checklist turns missing capabilities into actionable instructions.

---

## 12. Low-end device test plan

**Device lab** (bought, not emulated; about ₹40–60k in total [ASSUMPTION]):

| # | Class | Example profile | Role |
|---|---|---|---|
| D1 | Entry Android Go, 2–3 GB RAM | Android 13/14 Go edition, MediaTek Helio-class | Customer worst case |
| D2 | Budget Xiaomi/Redmi, 4 GB, MIUI/HyperOS | Aggressive battery manager | Rider + push/doze |
| D3 | Budget Vivo/Oppo/Realme, 4 GB | ColorOS/Funtouch battery manager | Rider + restaurant alerts |
| D4 | Samsung Galaxy A/M series, 4 GB, Samsung Internet + Chrome | One UI | Customer + Samsung Internet |
| D5 | Cheap 8–10" Android tablet, 3 GB | Kitchen device | Restaurant (landscape, long shifts) |
| D6 | Moto G-class mid-range | Lighthouse reference device | Performance baseline |
| D7 | Older iPhone on iOS 16.4–17, and one on iOS ≥ 18.4 | Safari | Customer + iOS install/push checks |
| D8 | An Android 9 device | Chrome 138 | Best-effort smoke test |

**Network conditions:**
- Chrome DevTools "Slow 4G" and "3G" presets.
- A real Jio and a real Airtel SIM in Mahabubnagar (field test).
- Flapping connectivity (airplane-mode toggles every 20–60 s).
- A captive "connected but no internet" Wi-Fi.

**Scenarios (each with pass criteria):**

| # | Scenario | Pass criteria |
|---|---|---|
| T1 | First visit from a WhatsApp-shared restaurant link on D1 / Slow 4G | LCP < 2.5 s (prerendered page), interactive menu < 5 s, total transfer < 500 KB |
| T2 | Repeat visit, offline | Shell < 1 s; cached menu browsable; checkout disabled with message |
| T3 | Full golden flow on D1 with UPI (PA test mode) where the OS **kills the tab** while the UPI app is open | Order page recovers and shows the correct payment state within 10 s of return |
| T4 | Restaurant D5: 8-hour shift, screen on, plugged in, 30 test orders at random intervals | 100% alerted in-app within 5 s; no tab discard; wake lock held; memory stable (no growth > 50 MB) |
| T5 | Restaurant D2/D3 with the app **backgrounded / screen off** for 15 min, then an order | Push received (record latency); escalation SMS fires on schedule when not acknowledged |
| T6 | Rider D2/D3: online for 2 h, 10 deliveries with Google Maps hand-offs | Location fresh when the app is foregrounded; offers received via push when backgrounded (latency recorded); battery drain recorded (target ≤ 20%/h [ASSUMPTION]) |
| T7 | Rider offline transitions: `PICKED_UP` and `DELIVERED` (COD) taken in airplane mode, reconnect after 5 min | Outbox flushes in order; server accepts; no duplicates; new offers blocked until synced |
| T8 | Rider: a delivery is cancelled by admin while the rider is offline with a queued `PICKED_UP` | `409` handled; blocking notice; outbox cleared |
| T9 | SW update during an active delivery or an unacknowledged order alert | No reload until a safe point |
| T10 | Telugu UI on D1 with TalkBack (Google TTS Telugu voice) | Key flows operable; labels read in Telugu; veg/non-veg announced |
| T11 | Data saver on / `saveData` | Small image variants only; no hero images |
| T12 | Low storage (< 500 MB free) | App works; caches purge on quota errors; no crash loops |

**Automation:**
- Lighthouse CI (mobile) and `size-limit` on every PR (doc 17 §12).
- Playwright E2E on emulated Pixel/Galaxy viewports with CPU throttling.
- WebPageTest (or equivalent) from an India location on staging nightly [ASSUMPTION – availability of a free India test agent].
- Field RUM (web-vitals) segmented by `deviceMemory` ≤ 2 vs > 2.
- Manual device passes before each release (T1–T12 subset) and the full set before pilot launch. Results recorded in the release checklist (doc 29).

---

## 13. Not doing in V1

- No offline ordering, no queued restaurant actions, no offline payment.
- No customer live map, no background rider tracking.
- No TWA or Play listing for the customer app. No iOS App Store presence.
- No native apps (React Native) until the §10.2 triggers fire.
- No notification action buttons. No custom notification sounds (not available on the web).

## 14. Requirements on other docs

| To | Requirement |
|---|---|
| 08 / 15 Notifications | VAPID keys in the secrets manager. Push sender with TTL/Urgency per type. 404/410 cleanup. Localised payloads. Restaurant presence tracking from SSE. Escalation ladder (repeat push → SMS → voice/IVR → ops → auto-reject) with configurable timers. Auto-pause of an unreachable outlet. `push_subscriptions.kind` for future FCM/APNs. |
| 11 API | Push-subscription endpoints (§6.1). `payment-status` endpoint. Rider transition endpoint accepting `Idempotency-Key`, `occurred_at`, `expected_from_status`, `location`. `/config/client` with `min_supported_version`. Backward compatibility across ≥ 2 frontend releases. |
| 13 State machine | Mark which delivery transitions may arrive late or out of band (§4.3) and the clamping rule for `occurred_at`. Decide whether a delivery OTP exists. |
| 14 Payments | PA checkout with UPI intent on mobile web. Return URL and status polling contract. Webhook-driven truth. |
| 05 / 06 UX | "Start shift" and device-readiness onboarding for restaurants. Install and permissions checklist for riders. "Syncing" state in the rider flow. |
| 22 DevOps | Serve `/.well-known/assetlinks.json` (restaurant, rider). `sw.js` and manifests `no-cache`. Ability to ship a self-destroying SW. |
| 28 Milestones | TWA packaging after the PWA pilot. Play organisation account (D-U-N-S) started early. |

## Sources (accessed 2026-10-04)

- WebKit, "Web Push for Web Apps on iOS and iPadOS" (2023-02-16): https://webkit.org/blog/13878/web-push-for-web-apps-on-ios-and-ipados/
- caniuse, Screen Wake Lock API: https://caniuse.com/wake-lock
- WebKit bug 254545 (Wake Lock in Home Screen web apps; fixed in iOS 18.4 per secondary reports): https://bugs.webkit.org/show_bug.cgi?id=254545
- caniuse, Background Sync API: https://caniuse.com/background-sync
- Chrome autoplay policy: https://developer.chrome.com/blog/autoplay
- Chrome auto-revocation of notification permissions (installed web apps exempt), 9to5Google: https://9to5google.com/?p=691969
- Chrome Android Web Push spam enforcement (Netcore changelog, 2025-08-04): https://updates.netcorecloud.com/changelog/google-chromes-crackdown-on-web-push-notifications-android
- Razorpay UPI Intent: https://razorpay.com/docs/payments/payment-methods/upi/upi-intent
- Google Maps URLs (travelmode incl. two-wheeler, dir_action=navigate, no API key): https://developers.google.com/maps/documentation/urls/get-started
- Using a PWA in your Android app (TWA, Bubblewrap): https://web.dev/articles/using-a-pwa-in-your-android-app
- Android TWA quick-start guide: https://developer.android.com/develop/ui/views/layout/webapps/guide-trusted-web-activities-version2
- TrustedWebActivityService (notification delegation): https://developer.android.com/reference/androidx/browser/trusted/TrustedWebActivityService
- Play Console, app testing requirements for new personal developer accounts: https://support.google.com/googleplay/android-developer/answer/14151465
- Play Console, foreground service and full-screen intent requirements: https://support.google.com/googleplay/android-developer/answer/13392821 ; AOSP full-screen intent limits: https://source.android.com/docs/core/permissions/fsi-limits
- Statcounter, mobile browser market share India: https://gs.statcounter.com/browser-market-share/mobile/india
- Chrome drops Android 8/9 support from Chrome 139 (Business Standard): https://www.business-standard.com/technology/tech-news/google-chrome-drop-old-android-devices-support-check-compatibility-125062700607_1.html
