# Backend guide

How the rovo backend is laid out, how to run it, and how to add a feature to a module. The
binding rules are in `00-implementation-charter.md`; this guide is the practical companion.

## 1. Run it

Prerequisites: Go (the toolchain downloads itself with `GOTOOLCHAIN=auto`), Docker, and a PostGIS
17 database.

```bash
docker run -d --name rovo-pg-test -e POSTGRES_PASSWORD=postgres -p 55432:5432 postgis/postgis:17-3.5
export GOTOOLCHAIN=auto
export ROVO_DATABASE_URL='postgres://postgres:postgres@localhost:55432/postgres?sslmode=disable'

cd backend
go run ./cmd/rovo migrate up      # River schema, then the goose migrations
go run ./cmd/rovo seed            # Mahabubnagar demo data (section 6)
go run ./cmd/rovo all             # api on :8080, health on :9090, worker in-process
curl -s localhost:9090/healthz    # 200
curl -s localhost:9090/readyz     # 200 once the database answers
```

| Command | What it does |
|---|---|
| `rovo api` | Public listener (`ROVO_HTTP_ADDR`, default `:8080`) and internal listener (`ROVO_INTERNAL_ADDR`, default `:9090`) with `/healthz` and `/readyz`. |
| `rovo worker` | River workers and periodic jobs, plus the internal listener. |
| `rovo all` | Both in one process. Local development only. |
| `rovo migrate [up\|down\|status]` | Apply all, roll back one step, or list migrations. |
| `rovo seed` | Load the demo dataset. Idempotent. Refused when `ROVO_ENV=prod`. |
| `rovo version` | Print the build version. |

Configuration is `ROVO_*` environment variables, parsed and validated in
`internal/platform/config`. In `dev` and `test`, missing signing and encryption keys are generated
at start-up and a warning is logged; tokens and encrypted fields from one run are then unreadable
in the next. Set `ROVO_JWT_SIGNING_KEY`, `ROVO_QUOTE_SIGNING_KEY` and `ROVO_FIELD_ENCRYPTION_KEY`
in `.env` for anything longer-lived than a single run. Outside dev/test the process refuses to
start without them.

## 2. Layout

```
backend/
  cmd/rovo/            entrypoint: subcommands, listeners, graceful shutdown
  migrations/          goose SQL, embedded in the binary
  internal/
    app/               composition root: builds Deps, constructs and registers the 15 modules
    platform/          shared infrastructure (no domain logic)
    modules/<name>/    one package per domain module
    adapters/          provider implementations (blob fake today; S3, SMS, PA later)
    seed/              demo dataset and its loader
openapi/openapi.yaml   the HTTP contract (ADR-P2-001)
```

Platform packages: `app` (the `Deps` container and `Module` type), `apperr`, `auth` (Ed25519 JWT,
refresh tokens, argon2id, TOTP, field encryption), `blob`, `clock`, `config`, `db`, `events`,
`health`, `httpx`, `idgen`, `migrate`, `money`, `obs`, `queue` (River), `ratelimit`, `realtime`
(SSE over LISTEN/NOTIFY), `testhooks`, `testutil`.

## 3. Modules

Fifteen modules, constructed in this order by `internal/app`:

| Module | Owns | Engineer |
|---|---|---|
| `identity` | OTP, sessions, tokens, admin login, roles | A |
| `geo` | cities, localities, zones, serviceability | A |
| `users` | profiles, addresses, consents, erasure | A |
| `catalog` | restaurants, onboarding, menus, discovery | A |
| `pricing` | fee configs, tax rules, quotes | B |
| `promotions` | coupons and redemptions | B |
| `ordering` | order placement and state machine | B |
| `payments` | payment sessions, webhooks, refunds | B |
| `dispatch` | riders, offers, deliveries | C |
| `ledger` | journals, settlement, payouts, COD deposits | C |
| `ratings` | ratings, reviews, aggregates | C |
| `notifications` | inbox, SSE fan-out, SMS, push | C |
| `support` | tickets, undeliverable approvals | C |
| `adminaudit` | audit log, approvals, flags, app_config, reason codes | A |
| `reporting` | summaries, dashboard, CSV exports | C |

Every module follows one shape. `module.go` is the public contract and is the only file other
modules may depend on:

```go
// The service other modules call, resolved through deps.Service(app.Service<Name>).
type Service interface { ... }

// DTOs, status enums, event-type constants ("<module>.<event>.v1"),
// and River job-arg types with a Kind() of "<module>.<job>".

func New(deps *app.Deps) *app.Module {
    s := &service{deps: deps}
    return &app.Module{Name: app.Service<Name>, Service: Service(s),
        Routes: s.registerRoutes, Jobs: s.registerJobs}
}
```

`payments.New` and `notifications.New` also take their provider ports (the fake payment
aggregator; fake SMS and push senders).

All service methods currently return `apperr.NotImplemented(...)` and the route and job
registrars are empty. The binary starts, registers all fifteen modules and serves health checks;
feature work fills the modules in.

### Rules

- A module never imports another module's package internals. Call the other module's `Service`:

  ```go
  geoSvc := deps.Service(app.ServiceGeo).(geo.Service)
  ```

  Resolve services inside handlers or methods, not in `New`: all modules are constructed before
  the registry is complete.
- Do not enqueue jobs or publish events from `New`. `deps.Queue` and `deps.Events` are set after
  every module is constructed and every job registrar has run.
- Side effects across modules go through events (`deps.Events.Publish` in the same transaction as
  the state change), handled by River jobs.
- Use `deps.Clock.Now()`, never `time.Now()`, and pass time into SQL as a parameter (R19).
- Money is integer paise (`platform/money`).

## 4. Adding an endpoint

1. Add or adjust the operation in `openapi/openapi.yaml` and run the lint (section 7).
2. Put SQL in the module's `store.go` (parameterised; ADR-P2-002).
3. Implement the service method in `service.go`, replacing the `NotImplemented` stub.
4. Add the handler in `http.go` and register it in `registerRoutes`:

   ```go
   func (s *service) registerRoutes(mux *httpx.Mux, _ *app.Deps) {
       mux.Handle("GET /api/v1/public/cities", s.listCities)
       admin := mux.Group( /* auth middleware */ )
       admin.Handle("POST /api/v1/admin/zones", httpx.RequireAdmin(s.createZone))
   }
   ```

   A handler decodes with `httpx.DecodeJSON` (size limit, unknown fields rejected), validates,
   authorises by role and ownership, calls the service, and answers with `httpx.JSON` or
   `httpx.Error` (RFC 9457 problem JSON with a stable `code`).
5. Test it: unit tests for pure logic, an integration test against a real database with
   `testutil.NewDB(t)`.

Two patterns registered on the same mux panic at start-up. `internal/app` has a test that wires
every module, so a clash fails `go test` rather than a deploy.

## 5. Adding a job

Define the args type in `module.go` (`Kind()` returns `"<module>.<job>"`), implement a worker
embedding `queue.WorkerDefaults[Args]`, and register it in `registerJobs` with
`queue.AddWorker(reg, worker)`. Periodic jobs use `reg.AddPeriodic`. Enqueue inside the
transaction that makes the job necessary: `deps.Queue.EnqueueTx(ctx, tx, args, opts)`.

The platform registers one job of its own, `platform.ratelimit_sweep`, every ten minutes.

## 6. Seed data

`rovo seed` loads a fictional Mahabubnagar dataset from `internal/seed`:

| What | Detail |
|---|---|
| City | `MBNR`, Mahabubnagar, Telangana, GST state 36, `Asia/Kolkata`, centroid 16.7488 N 78.0035 E, status `LIVE` |
| Zone | `MBNR-CORE`: the fixture octagon of about 5 km radius from doc 16 §10.1 |
| Localities | 12, all with PIN 509001 |
| Fee config | v1, `APPROVED`. R18 slabs on road distance: ₹20 / 30 / 40 / 50 / 60 for 0–2 / 2–4 / 4–6 / 6–8 / 8–10 km. `platform_fee_paise = 0` (R59). COD caps ₹1,000 and ₹600 for a first order (R6). |
| app_config | Eight CITY-scoped operational keys (payout schedules, cash ageing, support phone, service hours). The GLOBAL timer and threshold defaults and the reason-code catalogue are reference data in migration `00011`; the seed adds Telugu labels to thirteen customer- and partner-facing reason codes. |
| Ledger | The 18 platform accounts of the doc 14 chart for the city |
| Admin | `admin@rovo.local` with a global `ADMIN_SUPER` role |
| Restaurants | 4, `ACTIVE` and accepting orders, each with an owner, hours, cuisines, a 15% commission plan and an English/Telugu menu (45 items, 26 variants, 7 add-on groups; veg, non-veg and egg) |
| Riders | 4, `ACTIVE`, offline |
| Coupons | `WELCOME50` (50% up to ₹75, first order, minimum ₹149) and `FLAT30` (₹30 off, minimum ₹199) |

Admin credentials:

- Password: `ROVO_ADMIN_BOOTSTRAP_PASSWORD` if set (re-running with it set also resets the
  password). Otherwise a random password is generated when the admin is first created, printed
  once to stdout and flagged must-change.
- TOTP: in `dev` and `test` a secret is enrolled and printed. Elsewhere it is left unenrolled for
  the first login. The secret is encrypted with `ROVO_FIELD_ENCRYPTION_KEY`, bound to the row
  with the additional data `seed.TOTPAAD(userID)`; the identity module must decrypt with the same
  value. If the key is unset the seed says so, because the api process will generate a different
  key and will not be able to read the secret.

Things to know:

- Row IDs are derived from natural keys (`seed.ID("restaurant", "demo-biryani-house")`), so tests
  and fixtures can address seeded rows directly.
- Inserts are `ON CONFLICT DO NOTHING`. A re-run adds missing rows and never overwrites edits. To
  change seeded values in an existing database, reset the database.
- The seed fails if a seeded restaurant or locality falls outside the zone polygon.

Assumptions in the data, to be replaced before any real launch:

- The zone polygon is a synthetic octagon, not the real service area.
- Locality centroids are approximate placements inside that polygon, not surveyed points. Only
  the locality names come from the planning candidate list.
- Telugu text (city, localities, menus, reason labels) needs review by a native speaker.
- Restaurant FSSAI numbers are synthetic placeholders. Restaurants, people and phone numbers
  (`+9199999000xx`) are invented.
- The fee config is seeded `APPROVED` and the city `LIVE` so the demo works immediately. Doc 10
  §15.1 has both start as draft/planned and go through maker-checker.
- Doc 10 §15.1 says the first admin comes from a one-time setup link, with no default password
  anywhere. The seed creates one for dev and demo use only, which is one reason it is refused in
  production. The production bootstrap command is not built yet.

Not seeded: demo customers with addresses and historical delivered orders (doc 10 §15.2), and
notification templates.

## 7. Checks

```bash
cd backend
export GOTOOLCHAIN=auto
export ROVO_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:55432/postgres?sslmode=disable'
go vet ./...
go test -race ./...     # integration tests are skipped when ROVO_TEST_DATABASE_URL is unset

npx -y @redocly/cli@latest lint openapi/openapi.yaml --config openapi/redocly.yaml   # from the repo root
```

Integration tests create a throwaway database per test, cloned from a migrated template, so the
database user needs `CREATE DATABASE`.

## 8. Not built yet

- Module logic, handlers and jobs (all fifteen are contract-only skeletons).
- The auth, audience and rate-limit middleware exist in `platform/httpx` but are not yet applied
  to the public mux; `identity` wires them when it lands.
- `GET /api/v1/stream`: the realtime bus runs, the HTTP endpoint is not mounted.
- Response validation against the OpenAPI document in Go tests (ADR-P2-001). The validator
  dependency is not in `go.mod` yet.
- The S3/MinIO blob adapter (an in-memory fake is wired), real SMS and payment adapters.
- The `/_test/*` hooks listener is compiled only with the `testhooks` build tag and is not
  started by `cmd/rovo` yet.
