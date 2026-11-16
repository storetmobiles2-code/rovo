# ADR-P2-002: pgx with hand-written parameterised SQL instead of sqlc

| | |
|---|---|
| **Status** | Accepted (Phase 2) |
| **Date** | 2026-11-16 |
| **Owner** | Architect / Database Engineer |
| **Relates to** | Charter §2 and §6, doc 10 (database schema) |

## Context

The planning documents chose `sqlc` to generate typed query code. In Phase 2 the schema is still
moving (twelve migrations so far, more with each milestone) and three backend engineers write
queries in parallel. Generated code would have to be regenerated and re-merged on every schema or
query change, and sqlc's handling of PostGIS types, `jsonb` and partial-index `ON CONFLICT`
targets needs overrides that are easy to get wrong.

## Decision

- Database access uses `jackc/pgx/v5` (`pgxpool`) directly.
- Each module keeps its SQL in its own `store.go`. No module reads or writes another module's
  tables except through that module's exported service. `reporting` is the one read-only
  exception.
- Every statement is parameterised (`$1`, `$2`, …). Building SQL by concatenating request input
  is forbidden. Dynamic parts (sort column, filter set) come from fixed allow-lists in code.
- Transactions go through `platform/db.WithTx` / `InTx`. Jobs and events are enqueued in the same
  transaction as the state change (`queue.EnqueueTx`, `events.Publish`).
- Business time comes from the injected clock and is passed as a parameter. SQL `now()` is only
  for technical `created_at` / `updated_at` defaults (R19).
- Integrity that must hold regardless of the caller lives in the database: CHECK and exclusion
  constraints, the deferred journal-balance trigger, append-only triggers, and least-privilege
  grants for the runtime role.

## Consequences

- No generation step and no generated diff noise in reviews.
- Query/column mismatches surface at run time, not compile time. Every store method therefore
  needs an integration test against real PostGIS (`testutil.NewDB` gives each test a migrated
  database cloned from a template).
- Scanning rows is manual. Keep store methods small and return module DTOs.
- The security review covers all SQL for parameterisation.

## Revisit when

The schema stabilises (after M5). Module stores can be moved to sqlc one at a time without
changing the service interfaces, because stores are private to their module.
