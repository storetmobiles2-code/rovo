# PostGIS image for local dev and CI

Production uses **Amazon RDS for PostgreSQL 17** with PostGIS enabled by
`CREATE EXTENSION postgis` (doc 22 §6.1), so there is no PostGIS *image* in
production. The image below only exists for local Compose and CI service
containers, which must match the RDS major version (PostgreSQL 17, PostGIS 3.5;
R22).

## The problem: the official image is amd64-only

The upstream `postgis/postgis` image is published for `linux/amd64` only — there
is no `arm64` manifest (verified 2026-10-19 with `docker buildx imagetools
inspect postgis/postgis:17-3.5`). On Apple-silicon / arm64 laptops it runs only
under slow QEMU emulation, and arm64 CI runners cannot use it natively at all.

## Decision: `imresamu/postgis` for multi-arch, official on amd64

| Context | Image | Why |
|---|---|---|
| amd64 (default, CI amd64, most Linux dev) | `postgis/postgis:17-3.5` | The canonical upstream image; already pulled in this environment. |
| arm64 (Apple silicon, arm64 CI runners) | `imresamu/postgis:17-3.5` | A faithful **multi-arch mirror** of the official Dockerfiles (amd64 + arm64/v8), same PostgreSQL 17.6 / PostGIS 3.5.3, same entrypoint and env vars (`POSTGIS_VERSION`, `PG_MAJOR`). Verified 2026-10-19. |

`imresamu/postgis` is the image the upstream PostGIS maintainers point to for
arm64 (the author contributes the multi-arch tooling to the official repo); it
tracks the same Debian/PGDG sources. This avoids maintaining our own Dockerfile
for a dev/CI-only dependency.

### How it is selected

- **Compose:** both `compose.yaml` and `compose.prod.yaml` read
  `ROVO_POSTGRES_IMAGE` (default `postgis/postgis:17-3.5`). The `Makefile`
  detects `uname -m` and exports `imresamu/postgis:17-3.5` on arm64.
- **Scripts:** `scripts/lib.sh` `default_postgres_image()` does the same for
  throwaway containers in `backup`/`restore`/`verify-restore` (override with
  `ROVO_POSTGRES_IMAGE_ARM64`).
- **CI:** the `go` job's service container pins the official amd64 image because
  GitHub-hosted `ubuntu-24.04` runners are amd64. If an `ubuntu-24.04-arm`
  runner is used for integration tests, the workflow sets
  `ROVO_POSTGRES_IMAGE=imresamu/postgis:17-3.5`.

## Alternative considered: build our own

Doc 21 §5.3 and doc 22 §10 originally proposed building a `rovo-postgres:17-3.5`
image from `postgres:17-bookworm` + PGDG `postgresql-17-postgis-3` on native
runners and publishing it to GHCR. `postgres:17-bookworm` **is** multi-arch, and
PGDG ships `postgresql-17-postgis-3` for both arches, so this works
(`apt-cache madison postgresql-17-postgis-3` on bookworm shows PostGIS 3.6.x and
3.5.x). It is kept as a fallback: if `imresamu/postgis` ever lags the required
version or is unavailable, add `postgres-postgis.Dockerfile` here and the weekly
`postgres-image.yml` workflow (doc 21 §5.3). For V1 the maintained mirror is
less work and carries the identical upstream contents, so we do not run our own
build.
