# Contributing to rovo

Thanks for your interest in rovo, an open-source food-delivery platform for
Mahabubnagar, Telangana. This guide covers how to set up, the repository layout,
the rules that keep the codebase coherent, and how to get a change merged.

By contributing you agree to the [Code of Conduct](CODE_OF_CONDUCT.md) and to
license your work under **Apache-2.0** with a DCO sign-off (see
[Licensing & DCO](#licensing--dco)).

## Quick start

You need: Docker (with the daemon running), Go (version in `backend/go.mod`),
Node 22 + `pnpm` (via `corepack enable`), and `make`.

```bash
make bootstrap     # check prerequisites, write .env with dev-only keys, set up pnpm/tools
make dev           # deps (postgis/minio/mailpit) + migrate + seed + api + worker + 4 Vite apps
```

Then open the customer app at <http://localhost:5173> (restaurant :5174, rider
:5175, admin :5176). Everything runs **offline with fakes** — no payment, SMS or
cloud account is needed.

To exercise the production-like stack (one Caddy edge, four app hosts, same
images that would run in the cloud):

```bash
make prod-up       # build images, start, wait healthy (TLS=off by default)
make prod-seed     # load the Mahabubnagar demo dataset
make smoke         # edge smoke test
make prod-down
```

Run `make help` to list every target.

## Repository layout

A condensed map (full detail in `docs/planning/26-repository-structure.md`):

| Path | What |
|---|---|
| `backend/` | Go module: one binary with `api`, `worker`, `all`, `migrate`, `seed`, `version` subcommands |
| `openapi/` | OpenAPI 3.1 contract — the single source of truth for the HTTP API |
| `web/` | pnpm workspace: `customer`, `restaurant`, `rider`, `admin` PWAs + shared packages |
| `deploy/compose/` | local dev and production-like Docker Compose stacks |
| `deploy/docker/` | Dockerfiles (backend distroless, web/Caddy edge) |
| `deploy/caddy/` | edge config for the production-like stack |
| `deploy/terraform/` | OpenTofu IaC (modules + envs); **validate-only** this phase |
| `deploy/observability/` | Prometheus rules, Grafana dashboards + provisioning |
| `scripts/` | shell helpers called by `make` (bootstrap, backup/restore, smoke, tf-validate) |
| `docs/runbooks/` | operational runbooks (deploy, rollback, backup-restore, incident) |
| `.github/` | CI/CD workflows, issue/PR templates, CODEOWNERS, dependabot |

## Module boundary rules (backend)

- A module lives in `backend/internal/modules/<module>/` and exposes only its
  public API (`module.go`); `store.go`, `service.go`, `http.go`, `jobs.go` are
  internal.
- **No module imports another module's internals.** Cross-module calls go
  through the exported service interface; cross-module events go through River
  jobs. Shared code lives in `backend/internal/platform/*`.
- Table ownership is declared in `table_ownership.yaml`; migrations are global
  and ordered (goose), named `<ts>_<module>_<desc>.sql`.

## Coding standards

- **Go:** `gofmt`, `go vet`, `golangci-lint` clean. stdlib `net/http` ServeMux,
  `log/slog` JSON. Parameterised SQL only (no string concatenation of input).
  Inject `platform/clock.Clock`; never call `time.Now()` in a module or use SQL
  `now()` for business deadlines.
- **TypeScript:** strict mode, ESLint clean, no `any` escape hatches. User-facing
  strings go through i18next in **both `en` and `te`**. UI meets WCAG 2.2 AA.
- **SQL / migrations:** expand/contract only — a release adds (nullable/defaulted
  columns, new tables, `CREATE INDEX CONCURRENTLY`); destructive changes ship at
  least one release later, after no code reads the old shape. This keeps rollback
  safe.
- **OpenAPI:** the spec is the contract. Change `openapi/openapi.yaml` first; Go
  responses are validated against it in tests and TS types are generated from it.
- **SPDX headers:** source files carry `SPDX-License-Identifier: Apache-2.0`.
- Every handler: size-limited decode with `DisallowUnknownFields`, validate,
  authorise (role + ownership), map errors to RFC 9457 `problem+json`, log with
  the request id. No feature is "done" without tests, error handling, authz,
  logging and docs.

## Tests

`make check` is what CI gates on:

```bash
make check             # lint + unit + integration + contract tests
# or piecemeal:
make lint              # go, web, openapi, shell, docker, actions
make test              # go unit (-race) + vitest
make test-integration  # go tests against a throwaway PostGIS
```

Add or update tests with every change. Integration tests run against a real
PostGIS; contract tests validate responses against the OpenAPI spec.

## Commits and pull requests

- **Conventional Commits** for the PR title, e.g.
  `feat(dispatch): add offer cascade timeout`. Allowed types: `feat, fix, perf,
  refactor, docs, test, build, ci, chore, revert`. Scopes are module or app
  names. A `!` or `BREAKING CHANGE:` footer marks a breaking change.
- Keep PRs **small and focused**; feature branches live 2–3 days. `main` is
  always releasable; land unfinished work behind a feature flag.
- Fill in the PR template checklist. Protected paths (`payments`, `ledger`,
  `migrations`, `auth`, `deploy/terraform/envs/prod`) need two Code Owner
  approvals.
- CI must be green. Third-party GitHub Actions are pinned by commit SHA;
  Dependabot moves the pins.

## Licensing & DCO

- Contributions are accepted under **Apache-2.0** (inbound = outbound). There is
  no CLA.
- Sign off every commit with the **Developer Certificate of Origin**:

  ```bash
  git commit -s -m "feat(catalog): ..."
  ```

  which adds a `Signed-off-by: Your Name <you@example.com>` line certifying you
  wrote the change or have the right to submit it under Apache-2.0.
- Do not add dependencies under AGPL, SSPL or BSL/BUSL. Allowed: Apache-2.0,
  MIT, BSD-2/3, ISC, MPL-2.0, and OFL/Unicode for fonts.

## Trademark note

The Apache-2.0 license covers the **code**. It does **not** grant any rights to
the **rovo name or logos**. The brand assets in `docs/brand/` (the "rovo"
wordmark and the plate/wheel app-icon marks) are trademarks of the rovo project
and are **not** licensed under Apache-2.0.

You are free to fork and redistribute the code, but if you run or redistribute a
modified version you must **remove or replace the rovo brand assets** and use
your own name and logo, so that your fork is not presented as endorsed by or
affiliated with the rovo project. Factual references ("a fork of rovo") are fine.
See [`NOTICE`](NOTICE) for the full trademark statement.

## Security

Never file a public issue for a vulnerability. Follow [`SECURITY.md`](SECURITY.md)
for private disclosure.
