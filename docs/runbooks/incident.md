# Runbook: Incident response

What to do when an alert fires or a user reports the platform is broken. Alert
definitions: `deploy/observability/prometheus/rules/alerts.yml` (ids match
`docs/planning/24-observability-strategy.md` §7). Each alert's `runbook_url`
points at a section below.

Commands assume the production-like Compose stack (ADR-P2-003). `dc` below means:

```bash
dc() { docker compose -f deploy/compose/compose.prod.yaml --env-file .env "$@"; }
```

## Severity and response

| Severity | Meaning | Response |
|---|---|---|
| **P1** | Orders, payments or the whole API are failing now | Page on-call; acknowledge within 5 min; work until mitigated |
| **P2** | Degraded, or will become P1 if ignored | On-call (ops for business alerts) within 30 min in service hours |
| **P3** | Trend / ticket | Next business day |

## First five minutes (every incident)

1. **Acknowledge** the page and say so in the incident channel. One person is
   incident lead; everyone else takes direction.
2. **Is it up?** `make smoke` (edge → API on every host) and `dc ps`.
3. **What changed?** Last deploy (`git log -1`, image tag in `dc images`), last
   migration, last config change. If a release shipped in the last hour and the
   symptom started after it, **roll back first, debug second**:
   [rollback.md](rollback.md).
4. **Look:** Grafana "rovo — golden flow overview" (5xx ratio, p95, queue
   latency, orders by state), then `dc logs --since 15m api worker | grep -i error`.
   Every log line carries `trace_id` and `request_id`; ask the reporter for the
   request id shown on the error screen.
5. **Mitigate before root-causing.** Restart, roll back, pause a restaurant or
   zone from admin, or switch the city to COD-only — whatever stops the harm.

Never: edit ledger rows by hand (the ledger is append-only; post a reversing
entry through the admin tooling), run `migrate down` on a database with real
data, or paste secrets, phone numbers or KYC data into the incident channel.

## Closing an incident

- Confirm the alert has resolved and `make smoke` passes.
- Reconcile anything money-related (stuck payments, orders cancelled by ops).
- Write the incident note within two working days: timeline, impact (orders
  affected, rupees), detection, root cause, what we change. Blameless.

---

## api-down

`up{job="rovo-api"} == 0` — the API's internal listener is not scrapeable.

1. `dc ps api` and `dc logs --tail=100 api`. A crash loop at start is almost
   always config (missing or malformed key, `ROVO_APP_HOSTS`) or the database.
2. `dc exec postgis pg_isready -U rovo -d rovo`. If Postgres is down, fix that
   first (`dc logs postgis`; disk full is the usual cause: `df -h`).
3. `dc up -d --wait api`. If the new release will not start, roll back.
4. Confirm with `make smoke`.

## a10-worker-down

`up{job="rovo-worker"} == 0`. Orders still get placed but nothing asynchronous
happens: no dispatch offers, no notifications, no accept-deadline timeouts.

1. `dc ps worker`, `dc logs --tail=100 worker`.
2. `dc up -d --wait worker`. Jobs are durable in Postgres (River); they resume
   when the worker returns. Expect a burst of A8 queue latency while it drains.
3. If it was OOM-killed (`dc inspect` shows `OOMKilled`), raise the memory limit
   in `compose.prod.yaml` and find the job that grew.

## a12-5xx-spike

5xx ratio above threshold for 5 minutes.

1. Which route? Grafana 5xx-by-route panel, or
   `dc logs --since 10m api | grep '"status":5'`.
2. One route → a bug in that handler or its dependency; if it shipped recently,
   roll back. All routes → database (pool exhausted, slow queries:
   `select pid, now()-query_start, state, left(query,80) from pg_stat_activity where state <> 'idle' order by 2 desc;`)
   or the API is restarting.
3. Statement timeouts in the logs with one long-running query holding locks:
   `select pg_cancel_backend(<pid>);`.

## a11-slo-burn

Availability error budget burning fast (1 h/5 min), slow (6 h/30 min), the
budget ticket, or p95 latency above SLO.

- **Fast burn:** treat as [a12-5xx-spike](#a12-5xx-spike).
- **Slow burn / latency:** find the top routes by error and by p95, check
  `pg_stat_statements` for the slowest statements, and check for a missing index
  after a recent migration. Roll back if it started with a release.
- **Budget ticket (P3):** open a ticket; freeze risky releases until the 30-day
  budget recovers.

## a1-order-past-accept-deadline

Orders sit in `placed` beyond the restaurant accept deadline and were not
auto-cancelled. The timeout job is not running.

1. Check the worker ([a10-worker-down](#a10-worker-down)) and queue latency
   ([a8-river-queue-latency](#a8-river-queue-latency)).
2. Ops: call the affected restaurants from the admin order list; cancel with
   refund from admin any order the restaurant cannot fulfil.
3. Once the worker is healthy the deadline jobs run and catch up.

## a2-pending-accept-backlog

Many orders waiting for restaurant acceptance (the timeouts are working, the
restaurants are not responding).

1. One restaurant or many? One → call them; pause the restaurant from admin if
   unreachable. Many → suspect notifications ([a22-notify-queue](#a22-notify-queue))
   or SSE ([a18-no-sse](#a18-no-sse)): restaurants are not hearing the order alert.
2. Tell support so customers get a consistent answer.

## a3-no-riders-online

No rider online in a zone during service hours.

1. Ops: call the rider roster for that zone.
2. If nobody can come online within 15 minutes, pause delivery for the zone from
   admin so customers see "not delivering right now" instead of stuck orders.
3. If riders say they *are* online, the location heartbeat is failing: check the
   rider app release and `POST /api/v1/rider/location` errors in the API logs.

## a4-unassigned-delivery

A ready or nearly ready order has had no rider for too long.

1. Admin → dispatch board: manually assign the nearest online rider.
2. No rider available → [a3-no-riders-online](#a3-no-riders-online); phone the
   customer; cancel with full refund if it cannot be delivered.
3. Riders are online but receive no offers → dispatch jobs are stuck:
   [a9-river-queue-stuck](#a9-river-queue-stuck).

## a5-offer-accept-rate

Riders are rejecting or ignoring most offers (P3 trend).

Review offer distance, payout per delivery and offer timeout for the zone with
ops. Check that offer push notifications are arriving
([a22-notify-queue](#a22-notify-queue)). No immediate engineering action unless
it coincides with a release.

## a6-payment-webhooks

Payment-provider webhooks are failing signature verification or returning 5xx.

1. `dc logs --since 15m api | grep webhooks`. Signature failures on every call →
   the webhook secret was rotated on one side only; fix the secret and restart
   the API. 5xx → handler bug or database; roll back if recent.
2. Webhooks arrive only on `api.<domain>` (`POST /api/v1/webhooks/*`). Check the
   edge is forwarding them: `dc logs --since 15m caddy | grep webhooks`.
3. Nothing is lost: the provider retries, and the reconciliation job polls
   payment status for orders stuck in `payment_pending`. After the fix, confirm
   the stuck count drains.
4. If it will take more than 15 minutes, switch the city to **COD-only** in
   admin.

## a7-payment-success-drop

Online payment success rate has dropped.

1. Provider status page first — most drops are upstream. If the provider is
   degraded, switch to COD-only and post a banner.
2. If the provider is healthy, check for a client release that broke checkout
   and for [a6-payment-webhooks](#a6-payment-webhooks) (payments succeed but we
   never learn about it).
3. Afterwards, run reconciliation and refund any captured payment whose order
   was cancelled.

## a8-river-queue-latency

Jobs wait too long before starting.

1. Is the worker up and not restarting? ([a10-worker-down](#a10-worker-down))
2. Which queue and kind:
   `select queue, kind, state, count(*) from river_job where state in ('available','running','retryable') group by 1,2,3 order by 4 desc;`
3. A flood of one kind → find the producer (a retry storm from a failing
   provider is typical; fix or disable the provider, the jobs back off).
4. Genuine load → run a second worker: `dc up -d --scale worker=2 worker`.

## a9-river-queue-stuck

Jobs are available but none complete.

1. `dc logs --tail=200 worker` for a panic or a job blocking on an external call.
2. Long-running jobs:
   `select id, kind, attempted_at from river_job where state='running' order by attempted_at limit 20;`
3. Restart the worker (`dc restart worker`; it has a 120 s grace period to
   finish in-flight jobs). River rescues orphaned jobs automatically.
4. If one job kind poisons the queue, cancel those jobs from admin tooling and
   open a bug; do not delete rows from `river_job` by hand.

## a22-notify-queue

The notification queue is filling (fast = P1, slow = P2). Restaurants and riders
may be missing order alerts.

1. Provider outage (SMS / web push)? Check send errors in the worker logs. With
   a provider down, jobs retry with backoff; in-app SSE alerts still work.
2. Worker under-provisioned → scale as in
   [a8-river-queue-latency](#a8-river-queue-latency).
3. While it lasts, ops should phone restaurants for new orders
   ([a2-pending-accept-backlog](#a2-pending-accept-backlog)).

## a23-listen-watchdog

The Postgres `LISTEN` connection that feeds realtime fan-out (SSE) is not
receiving its heartbeat notification. Live order updates are stale; clients fall
back to polling.

1. `dc logs --since 15m api | grep -i listen`. The listener reconnects by
   itself; if it has not, `dc restart api` (SSE clients reconnect and replay
   from `Last-Event-ID`).
2. If it recurs, check Postgres connection limits
   (`select count(*) from pg_stat_activity;` against `max_connections`).

## a24-periodic-job-missed

A scheduled job has not completed in its window. `A24_SettlementMissed` (P1,
finance) means the daily settlement did not run.

1. Worker health first ([a10-worker-down](#a10-worker-down)); periodic jobs are
   enqueued only by a running worker that holds leadership.
2. Find the last run:
   `select kind, state, finalized_at, errors from river_job where kind = '<kind>' order by id desc limit 5;`
3. Fix the failure cause and re-run the job from admin tooling. Settlement is
   idempotent per (restaurant or rider, period): re-running cannot double pay.
4. For settlement, tell finance the new expected payout time before the bank
   cut-off.

## a18-no-sse

No open SSE connections during service hours, while orders are flowing.

1. Through the edge: `curl -N http://app.<domain>/api/v1/...stream` should hold
   open and emit heartbeats every 20 s. If it closes or buffers, the proxy is
   the problem — the Caddyfile must keep `flush_interval -1` and long read
   timeouts on `/api/*`.
2. If the edge is fine, see [a23-listen-watchdog](#a23-listen-watchdog).
3. Customers are not blocked (clients poll as a fallback), but restaurants' new
   order chime depends on this — warn ops.

## Backup alerts

`A16_BackupMissing` → [backup-restore.md](backup-restore.md).
