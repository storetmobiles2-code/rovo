# ADR-P2-001: OpenAPI is the contract; no Go server code generation in the first slice

| | |
|---|---|
| **Status** | Accepted (Phase 2) |
| **Date** | 2026-11-16 |
| **Owner** | Architect / Database Engineer |
| **Relates to** | Charter §2 and §6, doc 11 (API specification) |

## Context

Doc 11 defines about 220 operations across four app hosts and the server-to-server host. Several
engineers implement modules in parallel. The contract has to stay the single source of truth for
the Go handlers and for the four web apps, and it must not drift from what the server actually
returns.

The planning documents assumed a generated strict server (oapi-codegen). Generation adds a build
step every engineer must run, produces one large interface that all fifteen modules would have to
satisfy at once, and fits poorly with stdlib `net/http` ServeMux patterns and the module-owned
`Register(mux, deps)` convention.

## Decision

- `openapi/openapi.yaml` (OpenAPI 3.1) is the contract. A change to an endpoint starts with a
  change to this file.
- Go handlers are hand-written on the stdlib ServeMux. Each module registers its own routes.
- Drift is prevented by tests, not by generation: handler tests validate real responses against
  the spec (`getkin/kin-openapi`), and the spec is linted in CI with Redocly. The response
  validator is not wired yet (the dependency is not in `go.mod`); it lands with the first module
  that serves a real endpoint, and no handler merges without it after that.
- TypeScript types are generated from the spec with `openapi-typescript` into
  `web/packages/api-client`, so the web apps cannot compile against a shape the spec lacks.

## Consequences

- No generation step in the Go build; modules can land independently.
- A handler can be written before its contract test exists. Review must reject a handler PR that
  has no response-validation test.
- Request validation is hand-written (`httpx.DecodeJSON` with unknown-field rejection plus
  explicit checks), so request schemas in the spec are enforced only as far as the handler tests
  cover them.
- Admin read endpoints whose payloads are still open are typed as
  `additionalProperties: true` in the spec. Tightening them is part of implementing each endpoint.

## Lint policy

`npx @redocly/cli lint openapi/openapi.yaml` must exit 0. With the project configuration
(`--config openapi/redocly.yaml`) it reports no warnings; six findings are suppressed on purpose
in `openapi/.redocly.lint-ignore.yaml`:

| Finding | Why it is accepted |
|---|---|
| `/healthz`, `/readyz` have no 4XX | Probes have no client-error case. |
| `/payments/return/{provider}` has no 2XX | It always answers 303 to the order page. |
| `Page`, `DeliveryOfferStatus` unused | Shared shapes kept for the endpoints that will reference them. |
| `bearerAuth` unused | Reserved for native clients on the api host (doc 11 §1.12). |

## Revisit when

The schema and endpoint set are stable after M5. At that point generating request/response types
for Go (types only, not the server interface) is worth re-evaluating.
