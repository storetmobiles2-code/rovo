# 24 — Observability Strategy

| | |
|---|---|
| **Purpose** | Define telemetry for rovo: OpenTelemetry instrumentation (Go + frontend), the collector, logs (with PII redaction), metrics (RED/USE/business KPIs), traces and sampling, dashboards, SLOs and alerts, alert routing and on-call for a tiny team, uptime/synthetics, RUM, the audit-log boundary, and the data-volume budget vs cost. Also picks the production telemetry backend with a cost comparison. |
| **Owner** | DevOps Architect |
| **Status** | Draft v1.1 — reconciled with review (31) and rulings R1–R48, 2026-10-04 |
| **Depends on** | `00-planning-baseline.md` P3/P4/P11/P15; `22-deployment-architecture.md` (otel-gateway, accounts); `20-testing-strategy.md` §12.1 (load model, R45); `25-free-hosting-comparison.md` §14.6 (prices), §15 (cost, R46); `13-order-state-machine.md` (statuses); `14-payment-architecture.md` (webhooks); `12-auth-rbac.md` / `19` (PII classes, audit); `20-testing-strategy.md` (synthetics) |
| **Feeds** | `21-cicd-strategy.md` (deploy verification queries), `23-backup-disaster-recovery.md` (backup alerts), `29-production-readiness-checklist.md` |

**Changes in v1.1**
- **R36 / D10:** Grafana Cloud for metrics, traces, logs **and Faro** (frontend errors + RUM). **No Sentry in V1.** UptimeRobot (free, card-free) for external uptime checks.
- **CERT-In archive (M1):** CloudWatch Logs / S3 in `ap-south-1`, 180 days for all logs and ≥ 1 year (400 d) for security events (§1.2). The 3-day "safety copy" is gone. Cost is in `25` §15.1.
- **No CORS (R27/R14):** frontends call same-origin `/api/v1/*` on their own host, so trace headers need no CORS allowance. `tracePropagationTargets` is same-origin. Span names use `/api/v1` (R15).
- **No outbox table (R22/R42):** the outbox metrics and relay span are replaced by **River queue latency** and trace context carried in River job metadata. Alert A8 has been redefined accordingly.
- Added the **NOTIFY queue-usage alert and LISTEN watchdog (M12)**, the **missed-settlement / periodic catch-up alert (M11)** and the CERT-In archive delivery alert.
- `closed-pilot` exports OTLP **directly** from the SDKs (no Alloy gateway, R32). DB metrics come from a worker `db-stats` job plus CloudWatch.
- Accept-timeout values are referenced by doc 13 keys, not restated (R48). Volumes follow R45/doc 20. On-call aligned with RV-067.

---

## 1. Decision: telemetry backends per environment

| Env | Logs / metrics / traces | Errors | Uptime |
|---|---|---|---|
| local | **`grafana/otel-lgtm`** container (Loki, Grafana, Tempo, Prometheus/Mimir), dashboards provisioned from `deploy/observability/`. Works offline | console; Faro optional against local Alloy | — |
| ci | none (test output); E2E traces dumped as artifacts on failure | — | — |
| demo | the local stack (R24); no separate telemetry backend | — | — |
| staging | Grafana Cloud stack `rovo-nonprod` | **Faro** | Grafana synthetic golden flow (§9) |
| **production** | **Grafana Cloud stack `rovo-prod`, region AWS `ap-south-1` (Mumbai)** [S73] via OTLP: Free in `closed-pilot`, Pro in `public-launch` (`25` §15.1) | **Grafana Faro** (frontend errors + RUM) + backend error logs/spans | **UptimeRobot** (external, independent, free) + Grafana Synthetic Monitoring |
| production (compliance + last resort) | **CloudWatch**: AWS-vended metrics (ALB, ECS, RDS, CloudFront) + ~15 alarms → SNS; **CERT-In log archive** (§1.2) | — | — |

Grafana Cloud Free and UptimeRobot Free need no credit card. They are optional for developers and never required for the golden flow (RV-024).

### 1.1 Cost comparison for production (pilot volumes from `25` §10)

| Signal (pilot volume) | AWS-native (Mumbai, verified unit prices) | Grafana Cloud |
|---|---|---|
| Logs ≈ 2–15 GB/month (closed pilot → month 3; §11) | CloudWatch Logs $0.67/GB ingest (Standard) [C11] | Included in Free (50 GB/mo, 14 d) [S71] |
| Metrics ≈ 5,000 active series | CloudWatch custom metrics $0.30/metric-month → **≈ $1,500** [C11] | Included in Free (10k series) [S71] |
| Traces ≈ 25 GB/month (sampled) | X-Ray **UNVERIFIED** | Included in Free (50 GB) [S71] |
| Alerting | $0.10/alarm-month [C11] | Included; Telegram/Discord/email contact points [S74] |
| Users | IAM | Free: 3 users; **Pro $19/mo** platform fee + usage [S71] |
| **Pilot total** | **≈ $1,500+/month** with OTel metrics | **$0 (Free) / $19 (Pro)** |

**Decision (R36):** Grafana Cloud for application telemetry, including Faro. **Free in `closed-pilot`; Pro in `public-launch`** ($19/mo, for more than 3 users and IRM headroom) [S71]. No Sentry in V1; re-evaluate after the pilot only if Faro's error triage is inadequate. CloudWatch only for vended metrics and last-resort alarms. Telemetry leaves AWS only through OTLP, so the backend stays swappable (P15). If Grafana pricing/terms change, the fallback is self-hosted LGTM on ECS + S3 (more ops) or New Relic (100 GB free [S81]).
Data residency: the Grafana stack in `ap-south-1` keeps telemetry in India. Logs are PII-redacted at source (§3.3) anyway. Grafana Labs is recorded as a processor in the DPA register (RV-032) [LEGAL].

### 1.2 CERT-In log archive (M1, R36) — compliance copy, separate from Grafana

| Log class | Path | Retention (India, `ap-south-1`) |
|---|---|---|
| App JSON logs (`api`, `worker`, `migrate`) | stdout → `awslogs` → CloudWatch Logs `/rovo/prod/app`, **Infrequent Access** class | **180 days** |
| App security events (auth/OTP abuse, admin sign-in/TOTP, role grants, maker-checker decisions, KYC views, refused outbound hosts, origin-secret failures) | `slog` with `log.stream=security` → `/rovo/prod/security` | **400 days** |
| RDS PostgreSQL logs (incl. connections/auth) | RDS log export → CloudWatch Logs | 180 days (auth subset 400 days) |
| ALB + CloudFront access logs | S3 `rovo-prod-logs` | 180 days |
| CloudTrail (org), AWS Config, WAF logs, VPC flow logs (all), GuardDuty findings | S3 `rovo-audit-security-logs`, Object Lock | **400 days** |

- Clock source: AWS time sync (NTP) for all tasks; CERT-In requires synchronised clocks.
- Retrieval: CloudWatch Logs Insights (works on the IA class) and Athena over S3 for incident response and CERT-In requests (6-hour reporting, doc 19 §8.10).
- **Never rely on Grafana retention for compliance.** The archive is a Gate A item (`29`). Delivery is monitored by alert A24. Cost: `25` §15.1.

---

## 2. Instrumentation

### 2.1 Backend (Go)

| Area | Library / approach | Output |
|---|---|---|
| SDK | `go.opentelemetry.io/otel` SDK; resource attrs `service.name=rovo-api|rovo-worker`, `service.version`, `deployment.environment`, `cloud.region`, `aws.ecs.task.arn` | `closed-pilot`: OTLP/HTTPS **directly** to the Grafana Cloud OTLP endpoint (batch processor, bounded queue, drop on overflow). `public-launch`: the same, or `otel.rovo.internal:4317` if the optional gateway is deployed (§2.3) |
| HTTP server | `otelhttp` middleware; **span name = route template** (`GET /api/v1/orders/{id}`, R15), never raw path | Spans + `http.server.request.duration` histogram |
| HTTP clients (PA, SMS, Web Push) | `otelhttp.NewTransport` | Client spans, dependency latency |
| PostgreSQL | **pgx tracer** (`otelpgx`) attached to `pgxpool`; SQL text recorded **without parameters**; sqlc query name as span name | DB spans; pool stats metrics (acquired, idle, wait duration) |
| River jobs | River middleware/hooks: span per job (`river.job <kind>`), attrs `job.kind`, `attempt`, `queue`; **trace context carried in job metadata** at insert, so the job span **links** to the enqueuing request | Job spans + metrics (§4.3) |
| Events (R22/R42) | No outbox table and no relay. The publisher inserts one River job per subscriber in the business transaction (`InsertManyTx`), and each job's metadata carries `traceparent`. The subscriber job span **links** to the publishing request | End-to-end causality across async hops |
| SSE | One short span per connect/disconnect; **span events** (not child spans) per pushed message; gauge of open connections | Avoids hour-long spans |
| Logs | `slog` JSON handler wrapped to add `trace_id`, `span_id` from context; `ReplaceAttr` redaction (§3.3) | stdout → `awslogs` → CloudWatch Logs (**CERT-In archive**, §1.2) **and** OTLP logs exporter → Grafana Cloud |
| Runtime | `go.opentelemetry.io/contrib/instrumentation/runtime` | GC, goroutines, heap |
| Health | `/livez` (process), `/readyz` (DB ping, migrations at expected version, River client running for worker) — **excluded from tracing/logging** | — |

### 2.2 Frontend (4 SPAs/PWAs: customer, restaurant, rider, admin — R14)

- **Grafana Faro Web SDK** → Grafana Cloud Frontend Observability (R36; no Sentry). It captures JS errors (with source maps uploaded in CI and not served publicly), Web Vitals and release tags. The Faro OTel web tracing instrumentation adds W3C `traceparent` to **same-origin** `/api/v1/*` calls only (`propagateTraceHeaderCorsUrls` not needed). A browser action then links to the backend trace. **No CORS is involved**, because every app calls the API on its own host through CloudFront (R27).
- **Sampling:** session sampling by phase (§11). Faro Free covers 50k sessions/month [S71]. Use 100% in `closed-pilot` and ≈ 50% from month 3 (≈ 75k sessions/month).
- **PII:** a `beforeSend` hook scrubs phone, address, lat/lng and IDs in URLs (order IDs replaced by the route template). No session replay on any app [LEGAL].
- **Offline/PWA:** errors captured offline are buffered by the transport and sent on reconnect (bounded).

### 2.3 Collector: none in `closed-pilot`; optional Alloy gateway in `public-launch`

- **`closed-pilot` (R32):** there is no gateway task. The Go SDK exports OTLP/HTTPS directly to Grafana Cloud with batching. PII redaction happens in code (§3.3). Traces are sampled at **100%** in the SDK, which fits the Free allowance at pilot volume (§11). If the exporter fails, the bounded queue **drops telemetry and never blocks requests**.
- **DB metrics without a gateway:** a worker periodic job `ops.db_stats` (every 60 s, leader-only, catch-up irrelevant) reads `pg_notification_queue_usage()`, River queue stats, long-running transactions, dead tuples and connection counts, and emits them as OTel gauges. RDS CPU, credits, storage and IOPS come from the Grafana CloudWatch data source. Top queries come from Performance Insights / `pg_stat_statements` on demand.
- **`public-launch`:** still direct by default. Add a 1-task **Grafana Alloy** gateway (0.25 vCPU / 0.5 GB, cost in `25` §15.1) only when trace volume exceeds ≈ 70% of the plan allowance. It provides tail sampling (§5) and a second redaction layer. Locally, the same Alloy config can run against the LGTM container.

---

## 3. Logs

### 3.1 Schema (JSON, one line per event)

`ts`, `level`, `msg`, `service`, `version`, `env`, `trace_id`, `span_id`, `module` (orders, dispatch, payments…), `event` (stable snake_case name, e.g. `order_placed`, `offer_expired`), `actor_role`, `actor_id_hash`, `order_code` (RV-xxxx is allowed; not PII), `http.route`, `http.status`, `duration_ms`, `err` (message + type; stack only at error level).

### 3.2 Levels

| Level | Use |
|---|---|
| `ERROR` | Request/job failed and needs attention (always alerts via rate) |
| `WARN` | Degraded but handled (retry scheduled, fallback used, offer expired with no rider) |
| `INFO` | State transitions (order/delivery status changes), deploy/start/stop, admin actions summary |
| `DEBUG` | Off in prod; enabled per module for 15 min via feature flag |

### 3.3 PII redaction rules (enforced in code + collector) [LEGAL: DPDP]

| Data | Rule |
|---|---|
| Phone | Mask to `+91******1234` |
| OTP, passwords, TOTP secrets, JWTs, refresh tokens, PA secrets, webhook signatures | **Never logged** (deny-list keys: `otp`, `password`, `token`, `authorization`, `secret`, `signature`, `cookie`) → `[REDACTED]` |
| Names, email | Not logged; use `actor_id_hash` (HMAC-SHA256 with a log pepper) |
| Address | Locality ID only; no free text |
| Lat/lng | Rounded to 3 decimals (≈ 100 m) in logs; precise values only in DB |
| UPI VPA / bank account | Masked (`ab***@okxyz`, last 4 digits) |
| KYC | Document type + object key hash only; never contents or numbers. **Aadhaar numbers must never appear anywhere** (baseline §3) |
| Request/response bodies | Not logged in prod |

CI test: a unit test feeds known PII through the logger and asserts redaction, and the Alloy config has a regex scrub for E.164 and JWT patterns as a backstop.

### 3.4 Volume control

- Drop `/livez` and `/readyz` access logs and SSE heartbeat logs.
- Access logs: 100% at pilot (≈ 21 GB/month fits). At 10×, sample successful `GET` access logs at 20% (errors/4xx/5xx and all writes stay 100%).
- The CloudWatch copy is the **CERT-In archive** (§1.2): 180 days, IA class, security stream 400 days. Volume control applies only to the OTLP/Grafana path. The archive keeps every non-health log line.

### 3.5 Audit log ≠ observability

The **audit log** (admin actions, money movements, KYC access, role changes, order overrides) is a **business record**:
- An append-only DB table owned by the Security/Backend design (`12`).
- Kept per legal retention and included in backups (`23`).
- Queryable in the admin app.

Observability logs in Grafana are **operational, sampled, short-retention (14 days on Grafana Free; 30-day+ retention on Pro is UNVERIFIED) and PII-redacted**. Never rely on Loki for audit/compliance evidence. The compliance log copy is the CERT-In archive (§1.2), and business audit records live in the DB. Never write audit records only to logs.

---

## 4. Metrics

### 4.1 RED (HTTP, per route template)

| Metric | Type | Labels (bounded) |
|---|---|---|
| `http.server.request.duration` | histogram (buckets 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000 ms) | `http.route` (≤ 60 templated routes), `http.request.method`, `status_class` (2xx/3xx/4xx/5xx) |
| `http.server.active_requests` | up-down counter | `http.route` group |
| `rovo_sse_connections` | gauge | `channel` (customer_order, partner_inbox, rider_offers) |
| `rovo_sse_messages_sent_total` | counter | `channel` |

### 4.2 USE (resources)

- **ECS tasks:** CPU/memory utilisation from CloudWatch vended metrics (free basic metrics [ASSUMPTION]) via the Grafana CloudWatch data source (pull on view; GetMetricData API cost **UNVERIFIED**, small). Go runtime metrics come from OTel.
- **RDS:** CPU, CPU credit balance, freeable memory, free storage, read/write IOPS & latency, DB connections, replica lag (when a replica exists), via the CloudWatch data source. `pg_stat_statements`, locks, dead tuples and long transactions come from the Alloy postgres integration.
- **ALB:** request count, target 5xx, target response time, active connection count.

### 4.3 Async / pipeline

| Metric | Labels |
|---|---|
| `rovo_river_jobs` (gauge by state: available, running, retryable, scheduled) | `kind`, `state` |
| `rovo_river_job_duration` (histogram), `rovo_river_job_failures_total` | `kind` |
| `rovo_river_queue_latency_seconds` (age of the oldest `available` job whose `scheduled_at` ≤ now; replaces outbox lag, R22/R42) | `queue` |
| `rovo_pg_notification_queue_usage` (ratio 0–1 from `pg_notification_queue_usage()`, M12) | — |
| `rovo_listen_watchdog_ok` (1 if the replica's self-NOTIFY ping, sent every 30 s, returned within 5 s), `rovo_listen_reconnects_total` (M12) | `replica` |
| `rovo_periodic_job_last_success_timestamp` (catch-up jobs, M11: settlement, PA recon, retention sweeps) | `job` |
| `rovo_settlement_last_completed_period_end` (M11) | `kind` (restaurant, rider) |
| `rovo_webhook_lag_seconds` (provider event time → processed) | `provider` |
| `rovo_webhook_failures_total` | `provider`, `reason` (signature, parse, handler) |

### 4.4 Business KPIs (emitted on state transitions; city label for multi-city readiness)

| KPI | Metric | Type |
|---|---|---|
| Orders placed / min | `rovo_orders_placed_total{city,payment_method}` | counter |
| Order status transitions | `rovo_order_transitions_total{city,from,to}` (bounded by state machine) | counter |
| Restaurant acceptance time | `rovo_order_accept_seconds{city}` | histogram |
| Orders awaiting acceptance | `rovo_orders_pending_accept{city}` + oldest age `rovo_orders_pending_accept_oldest_seconds` | gauge |
| Unassigned deliveries | `rovo_deliveries_unassigned{city,zone}` + oldest age | gauge |
| Offer outcomes | `rovo_delivery_offers_total{city,outcome=accepted|declined|expired}` | counter → accept rate |
| Time to assign | `rovo_delivery_assign_seconds{city}` | histogram |
| Riders online / busy | `rovo_riders_online{city,zone}`, `rovo_riders_busy{city}` | gauge |
| Payment success rate | `rovo_payments_total{provider,result=success|failed|pending_timeout}` | counter |
| Delivery time (placed → delivered) | `rovo_order_fulfilment_seconds{city}` | histogram |
| Cancellations | `rovo_orders_cancelled_total{city,cancelled_by,reason_group}` | counter |
| COD cash-limit blocks | `rovo_rider_cod_blocked_total{city}` | counter |

Zones are bounded (dozens). No restaurant/rider/order IDs as labels; use exemplars (trace IDs) and logs for drill-down.

### 4.5 Cardinality budget (Grafana Free: 10k active series [S71])

| Source | Budget (series) |
|---|---|
| HTTP RED (60 routes × 2 methods avg × 4 status classes × 12 histogram series, many combinations absent) | ≈ 2,000 |
| SSE + async + webhooks | ≈ 300 |
| Business KPIs | ≈ 500 |
| Go runtime + pgx pool (2 services × ~4 tasks) | ≈ 800 |
| Postgres integration (filtered) | ≈ 800 |
| CloudWatch (queried, not ingested) | 0 |
| Alloy self + synthetic checks | ≈ 400 |
| **Total** | **≈ 4,800** (≤ 50% of free) |

Guardrails: Alloy `relabel` drops unknown labels; CI lint for metric definitions (label allow-list); a Grafana "cardinality" dashboard alerts at 7k series.

---

## 5. Traces & sampling

- **Propagation:** W3C `traceparent` from browser (Sentry SDK) → ALB → api → River job metadata / outbox rows → worker → outbound HTTP (PA, SMS).
- **Sampling** (in `otel-gateway`, tail-based; the SDK is parent-based and exports 100% to the gateway):

  | Policy | Keep |
  |---|---|
  | Any span with error status | 100% |
  | Root duration > 1 s (api) / > 10 s (jobs) | 100% |
  | `payments.*`, `webhook.*`, `dispatch.*`, `admin.*` routes/jobs | 100% |
  | Everything else | **10%** (pilot), 5% (10×) |

- Budget at pilot: ≈ 1.3M requests/day × ~6 spans × ~1 KB ≈ 7.8 GB/day raw. After sampling ≈ **0.8 GB/day ≈ 25 GB/month**, inside the 50 GB Free allowance [S71]. At 10×, tighten to 5% plus Pro usage.
- Exemplars link latency histograms to traces.

---

## 6. Dashboards (as code: `deploy/observability/dashboards/*.json`, provisioned to local LGTM and Grafana Cloud)

1. **Golden flow overview** (single pane for ops): orders/min, pending accept (count + oldest), unassigned deliveries, riders online by zone, offer accept rate, payment success, p95 API latency, 5xx rate, SSE connections.
2. **API RED** by route group; top slow routes; error log panel (Loki) with trace links.
3. **Dispatch**: offers sent/accepted/expired, time to assign histogram, cascade depth, riders online/busy heatmap by zone/hour.
4. **Payments & webhooks**: success/failure by method, webhook lag, signature failures, reconciliation status.
5. **Async**: River queue depth by kind/state, queue latency per queue, job latency/failures, NOTIFY queue usage, LISTEN watchdog/reconnects, periodic-job last success.
6. **Database**: RDS CPU/credits/memory/storage/IOPS, connections, `pg_stat_statements` top queries, locks, long transactions, autovacuum.
7. **Infrastructure**: ECS task CPU/memory/restarts, ALB, deploy annotations.
8. **Frontend (RUM, Faro)**: Web Vitals (LCP, INP, CLS) by app/device class, JS errors with stack traces, PWA install/offline usage.
9. **SLOs**: error budget burn per SLO.
10. **Cost & cardinality**: Grafana usage (series, GB, Faro sessions), AWS cost (CloudWatch billing metric in us-east-1 [ASSUMPTION]), **CloudFront requests per month vs the 10M flat-rate allowance** (`25` §15.7, M13).

---

## 7. SLOs (pilot; reviewed after 4 weeks of data)

| SLO | SLI | Target (28-day) |
|---|---|---|
| API availability | Non-5xx ratio of `api` requests excluding health | **99.5%** |
| API latency | p95 of core routes (menu, cart, place order, order status) < 400 ms | 99% of 5-min windows |
| Order placement success | `PLACED` ÷ (place-order attempts with valid cart, excl. payment declines) | **99%** |
| Restaurant acceptance flow | Orders accepted or explicitly rejected/auto-cancelled within 3 min (P11 timeout honoured) | 99% (system honours the timer) |
| Dispatch | Deliveries assigned within 5 min of `READY_FOR_PICKUP`/assignment trigger, **when riders are online in the zone** | 95% |
| Real-time | SSE event delivered ≤ 5 s after state change (measured by synthetic) | 99% |
| Webhooks | PA webhook processed ≤ 60 s after receipt | 99.5% |

**Burn-rate alerts** (multi-window, multi-burn [Google SRE workbook pattern]):
- **Page:** 2% of budget in 1 h (burn 14.4×) AND in 5 min.
- **Page:** 5% in 6 h (6×) AND in 30 min.
- **Ticket:** 10% in 3 d (1×).

---

## 8. Alerts

| # | Alert | Condition (indicative) | Severity | Route |
|---|---|---|---|---|
| A1 | **Order unaccepted > 3 min** | `rovo_orders_pending_accept_oldest_seconds > 180` for 1 min (the system should have auto-cancelled/escalated; indicates worker/timer failure) | **P1** | Telegram on-call + ops group |
| A2 | Pending-accept backlog | `rovo_orders_pending_accept > 5` for 5 min (operating hours) | P2 | Ops group (call the restaurant) |
| A3 | **No riders online in zone** | `rovo_riders_online{zone} == 0` for 10 min within zone operating hours | P2 | Ops group |
| A4 | Unassigned deliveries aging | oldest unassigned > 10 min | P1 | On-call + ops |
| A5 | Offer accept rate collapse | accept rate < 30% over 30 min with ≥ 10 offers | P3 | Ops (email) |
| A6 | **Payment webhooks failing** | `rate(rovo_webhook_failures_total[10m]) > 0.1/s` OR `rovo_webhook_lag_seconds > 300` OR no webhooks for 30 min while online payments > 5 in that window | **P1** | On-call |
| A7 | Payment success rate drop | success < 85% over 15 min with ≥ 20 attempts | P1 | On-call (consider COD-only flag) |
| A8 | **Outbox lag > 1 min** | `rovo_outbox_lag_seconds > 60` for 2 min | P1 | On-call |
| A9 | River queue stuck | `available` jobs > 500 for 5 min OR job failure rate > 10% | P2 | On-call |
| A10 | Worker down | `up{service="rovo-worker"} == 0` for 2 min OR CloudWatch ECS `RunningTaskCount < 1` | **P1** | On-call + **CloudWatch last-resort** |
| A11 | SLO burn (availability/latency/placement) | §7 | P1/P2 | On-call |
| A12 | 5xx spike | 5xx ratio > 5% for 5 min | P1 | On-call |
| A13 | **DB storage > 80%** | RDS `FreeStorageSpace` < 20% | P2 | On-call + CloudWatch alarm |
| A14 | DB CPU credits low / CPU high | credit balance < 20% OR CPU > 80% for 15 min | P2 | On-call |
| A15 | DB connections > 80% of max | — | P2 | On-call |
| A16 | **Backup failed / missing** | Dump job metric absent > 26 h OR job exit ≠ 0 OR dump size < 70% of 7-day median; RDS cross-Region replication stopped (EventBridge RDS event) | P2 | On-call + email |
| A17 | Certificate/DNS | ACM renewal failure event; UptimeRobot TLS expiry < 14 d | P2 | Email |
| A18 | SSE connections drop to 0 during operating hours | `rovo_sse_connections == 0` for 5 min, 11:00–23:00 IST | P2 | On-call |
| A19 | Security signals | KMS `ScheduleKeyDeletion`, root login, IAM policy change in prod, GuardDuty high (EventBridge → SNS) | P1 | Founders + on-call |
| A20 | Cost | AWS Budget 80/100%, Cost Anomaly Detection, Grafana series > 7k | P3 | Email |
| A21 | Synthetic golden flow failed (staging) / external uptime down (prod) | 2 consecutive failures | P2 (staging) / P1 (prod) | On-call |

Every alert has a **runbook link** (`docs/ops/runbooks/<alert>.md`, Phase 2) and a dashboard link.

---

## 9. Uptime & synthetic checks

| Check | Tool | Frequency | Target |
|---|---|---|---|
| `https://api.rovo.example/readyz`, `app.`, `partner.`, `admin.` home, `cdn.` sample image, TLS expiry | **UptimeRobot Free** (50 monitors, 5-min interval; ToS allows commercial use [S76, S77]) | 5 min | Independent of AWS + Grafana |
| API multi-step (login with test account → menu → cart quote) | **Grafana Synthetic Monitoring** (100k API checks/month free [S71]) | 1 min (≈ 43k/month per check) | prod |
| **Synthetic golden-flow order** (fake/sandbox payment, test restaurant + test rider bot in a hidden "QA zone") | Scheduled job (k6 or Go `rovo synth` in an ECS scheduled task) | Staging: every 15 min, 08:00–23:30 IST; **Prod: every 30 min in a hidden test city/zone with COD and auto-cancel** [OPEN: business approval; excluded from KPIs via `is_synthetic` flag] | Emits `rovo_synthetic_success`, SSE latency |
| Public status page | UptimeRobot status page (1 free) [S76] | — | Partners/riders |

---

## 10. Alert routing & on-call (tiny team)

- **Channels** (Grafana contact points [S74]):
  - **Telegram** bot → `rovo-oncall` group (P1/P2) and `rovo-ops` group (business alerts A2/A3/A5 to city ops staff).
  - **Email** (P3, digests).
  - **Discord** optional for the open-source community status channel (no PII).
- **CloudWatch last resort** (A10, A13, A16, A19): SNS → email to on-call + founders. Fires even if Grafana Cloud is down.
- **On-call:** Grafana IRM (Free: 3 IRM users [S71]) weekly rotation, primary + secondary.
  - **Operating hours (10:30–23:30 IST):** ack P1 within 5 min.
  - **Overnight:** only infra P1 (DB down, worker down, security). Business alerts are suppressed outside zone hours.
  - Phone-call escalation for unacked P1 after 10 min [OPEN: Grafana IRM phone/SMS on Free **UNVERIFIED**; fallback is a second Telegram mention + UptimeRobot voice credits].
- **Ops vs engineering split:** business alerts (no riders, restaurant not accepting) go to the **ops desk** with clear actions. Engineering gets system alerts.
- **Post-incident:** a blameless review for every P1 (template in Phase 2), with action items in the backlog.

---

## 11. Data-volume budget (production)

| Signal | Pilot (high scenario) | Grafana Free limit [S71] | 10× | Plan at 10× |
|---|---|---|---|---|
| Logs | ≈ 21 GB/mo | 50 GB | ≈ 120 GB (with 2xx sampling) | Pro usage (price **UNVERIFIED**, est. `25` §15.2) |
| Metrics | ≈ 4,800 series | 10k | ≈ 7–9k (more tasks, zones) | Pro or tighter relabel |
| Traces | ≈ 25 GB/mo (10% + tail) | 50 GB | ≈ 120 GB at 5% | Pro usage |
| Frontend sessions | 10% sample ≈ 60k/mo | 50k | — | 5% or Pro |
| Sentry errors | backend ≪ 1k; frontend unknown | 5k (Developer) | — | **Team plan** from launch ($26 [S75]) for > 1 user |
| CloudWatch Logs (safety copy) | ≈ 21 GB × $0.67 ≈ $14 [C11] | n/a | ≈ $80 | Reduce to errors-only safety copy at 10× |

Grafana Free retention is 14 days [S71]. Longer retention needs for investigations rely on the audit log (DB) and S3-archived ALB logs (90 d).

---

## 12. What we are not doing in V1

- No APM agents with proprietary protocols. OTel only.
- No log-based billing analytics or BI in the observability stack (that is the analytics roadmap, deferred).
- No session replay on customer/rider apps (privacy + cost).
- No self-hosted Prometheus/Loki in production (ops cost). It remains the fallback if Grafana Cloud terms change.
