<!-- SPDX-License-Identifier: Apache-2.0 -->

# rovo

**rovo** is an open-source food-delivery platform built for **Mahabubnagar,
Telangana**. It covers the full golden flow — customer orders from a nearby
restaurant, a delivery partner picks up and delivers, payment is settled, and
both sides rate each other — across four progressive web apps (customer,
restaurant, rider, admin), a Go backend, and an OpenAPI contract that both sides
generate from.

It is designed to run **entirely offline with fakes** for development and demos:
no payment gateway, SMS provider or cloud account is required. The same container
images target AWS (ECS/Fargate) in production via OpenTofu, but the Phase-2
deployment target is the production-like Docker Compose stack (see
`docs/implementation/00-implementation-charter.md`, ADR-P2-003).

> Status: **pre-1.0, foundation stage.** `v1.0.0` is the Mahabubnagar go-live.

## Highlights

- **Golden flow end-to-end:** discovery → cart/checkout → order state machine →
  dispatch & delivery → payments/settlement → ratings.
- **Four PWAs** (`customer`, `restaurant`, `rider`, `admin`), mobile-first,
  bilingual **English + Telugu**, WCAG 2.2 AA.
- **One Go binary** — `api`, `worker`, `all`, `migrate`, `seed`, `version` — on
  stdlib `net/http`, `pgx` + hand-written SQL, goose migrations, River jobs.
- **Contract-first:** OpenAPI 3.1 is the source of truth; responses are validated
  against it and TypeScript types are generated from it.
- **Ops-ready:** Prometheus metrics, Grafana dashboards and alert rules, backup/
  restore with verification, and validated OpenTofu IaC.

## Requirements

- **Docker** with the daemon running
- **Go** (version pinned in `backend/go.mod`)
- **Node 22** and **pnpm** (`corepack enable`)
- **make**

## Quick start (local dev)

```bash
make bootstrap     # check prerequisites, write .env with dev-only keys, prepare pnpm/tools
make dev           # deps + migrate + seed + api + worker + the 4 Vite apps
```

| App | URL |
|---|---|
| customer | <http://localhost:5173> |
| restaurant | <http://localhost:5174> |
| rider | <http://localhost:5175> |
| admin | <http://localhost:5176> |

Supporting UIs: MinIO console <http://localhost:9001>, Mailpit <http://localhost:8025>.

`make bootstrap` copies `.env.example` to `.env` (git-ignored) and fills in fresh
**dev-only** secrets with `make dev-keys`. Never commit `.env`.

## Production-like stack

One Caddy edge serves the four app hosts and routes `/api/*` same-origin, using
the same images that would run in the cloud:

```bash
make prod-up       # build + start + wait healthy (TLS=off | internal | acme)
make prod-seed     # load the Mahabubnagar demo dataset
make smoke         # smoke-test the running stack (see below)
make prod-obs-up   # optional: Prometheus + Grafana next to the stack
make prod-down     # stop (DOWN_ARGS=-v to wipe volumes)

make smoke-up      # one shot: bring the stack up, then smoke-test it
```

| Host | Serves |
|---|---|
| <http://app.rovo.localhost> | customer app |
| <http://restaurant.rovo.localhost> | restaurant app |
| <http://rider.rovo.localhost> | rider app |
| <http://admin.rovo.localhost> | admin app |
| `api.rovo.localhost` | provider webhooks only (`POST /api/v1/webhooks/*`) |

Most browsers resolve `*.localhost` to `127.0.0.1` on their own; otherwise add
the five hosts to `/etc/hosts`. Mailpit (OTP and e-mail capture) is at
<http://localhost:8025>.

`make smoke` (`scripts/smoke.sh`) needs no hosts entries. It checks, on every
host, that `GET /healthz` is answered by the API through the edge; on each app
host, that the SPA and its fallback are served, the security headers are
present, a same-origin `/api/v1` request reaches the API, and **no CORS grant**
is given to a foreign origin; and that the `api.` host refuses everything except
webhooks.

## Observability

```bash
make obs-up        # local dev: Prometheus + Grafana scraping api/worker on the host
make obs-down
make prod-obs-up   # production-like stack: scrape the api/worker containers
# Grafana    http://localhost:3000
# Prometheus http://localhost:9091
```

Prometheus scrapes the api and worker internal port (`:9090`, `/metrics`), Grafana
auto-provisions the **rovo overview** dashboard, and the alert rules in
`deploy/observability/prometheus/rules/` match the SLOs and alerts in
`docs/planning/24-observability-strategy.md`.

## Tests

```bash
make check             # lint + unit + integration + contract (what CI gates on)
make test              # Go unit (-race) + vitest
make test-integration  # Go tests against a throwaway PostGIS
make test-e2e          # Playwright golden flow (when e2e/ exists)
make lint              # go, web, openapi, shell, docker, actions
make vuln              # govulncheck + pnpm audit + trivy
```

## Repository map

| Path | What |
|---|---|
| `backend/` | Go module: one binary (`api`/`worker`/`all`/`migrate`/`seed`/`version`) |
| `openapi/` | OpenAPI 3.1 contract — single source of truth for the HTTP API |
| `web/` | pnpm workspace: 4 PWAs + shared packages |
| `deploy/compose/` | local dev (`compose.yaml`) and production-like (`compose.prod.yaml`) stacks |
| `deploy/docker/` | Dockerfiles (backend distroless, web/Caddy edge) |
| `deploy/caddy/` | edge config for the production-like stack |
| `deploy/terraform/` | OpenTofu IaC (modules + envs); validate-only this phase |
| `deploy/observability/` | Prometheus rules + Grafana dashboards/provisioning |
| `scripts/` | shell helpers behind `make` (bootstrap, backup/restore, smoke, tf-validate) |
| `docs/planning/` | the planning documents (architecture, APIs, deployment, DR, …) |
| `docs/runbooks/` | deploy, rollback, backup-restore, incident |
| `.github/` | CI/CD, issue/PR templates, CODEOWNERS, dependabot |
| `Makefile` | single entry point for every developer and CI task (`make help`) |

## Documentation

- Architecture, APIs, deployment, DR and observability: `docs/planning/`
- Phase-2 implementation charter: `docs/implementation/00-implementation-charter.md`
- Operational runbooks: `docs/runbooks/`
- Brand assets and usage: `docs/brand/README.md`

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Please also read the
[Code of Conduct](CODE_OF_CONDUCT.md). Report vulnerabilities privately per
[SECURITY.md](SECURITY.md).

## License & trademark

The code is licensed under **Apache-2.0** (see [LICENSE](LICENSE)). The Apache
license does **not** grant rights to the **rovo name or logos** — the brand
assets in `docs/brand/` are trademarks of the rovo project. Forks must remove or
replace them and use their own name and logo. See [NOTICE](NOTICE) for the full
trademark statement.
