# rovo — single entry point for developers and CI (doc 26 §6).
# `make help` lists every target. Targets delegate to scripts/ and tool CLIs.
# SPDX-License-Identifier: Apache-2.0

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help
.ONESHELL:

# --- paths -------------------------------------------------------------------
ROOT        := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
BACKEND_DIR := $(ROOT)/backend
WEB_DIR     := $(ROOT)/web
COMPOSE_DIR := $(ROOT)/deploy/compose
COMPOSE      := docker compose -f $(COMPOSE_DIR)/compose.yaml
COMPOSE_PROD := docker compose -f $(COMPOSE_DIR)/compose.prod.yaml
TF_DIR      := $(ROOT)/deploy/terraform
APPS        := customer restaurant rider admin

# --- versioning / images -----------------------------------------------------
VERSION    ?= $(shell git -C $(ROOT) describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git -C $(ROOT) rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
REGISTRY   ?= ghcr.io/storetmobiles2-code/rovo
IMAGE      ?= rovo:local
WEB_IMAGE  ?= rovo-web:local
PLATFORMS  ?= linux/amd64,linux/arm64

# PostGIS: official image is amd64-only; use the multi-arch mirror on arm64
# (deploy/docker/POSTGIS.md).
ARCH := $(shell uname -m)
ifneq (,$(filter arm64 aarch64,$(ARCH)))
export ROVO_POSTGRES_IMAGE ?= imresamu/postgis:17-3.5
endif

# Pinned tool image versions (match .github/workflows/ci.yml).
SHELLCHECK_IMG := koalaman/shellcheck:v0.11.0
HADOLINT_IMG   := hadolint/hadolint:v2.15.1
ACTIONLINT_IMG := rhysd/actionlint:1.7.12
TOFU_IMG       := ghcr.io/opentofu/opentofu:1.13.1
TFLINT_IMG     := ghcr.io/terraform-linters/tflint:v0.59.1
CHECKOV_IMG    := bridgecrew/checkov:3.2.483
TRIVY_IMG      := aquasecurity/trivy:0.75.0
REDOCLY        := pnpm --dir $(WEB_DIR) dlx @redocly/cli@2.2.1

# Run a target that needs .env, loading it first.
define with_env
set -a; [ -f $(ROOT)/.env ] && . $(ROOT)/.env; set +a;
endef

## --- help --------------------------------------------------------------------
.PHONY: help
help: ## List available targets
	@echo "rovo make targets:"
	@grep -hE '^[a-zA-Z0-9_.-]+:.*?## ' $(MAKEFILE_LIST) \
	  | sort | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- setup / config ----------------------------------------------------------
.PHONY: bootstrap dev-keys
bootstrap: ## Check prerequisites and prepare the machine (.env, pnpm, tools)
	@$(ROOT)/scripts/bootstrap.sh

dev-keys: ## Generate dev-only secrets into .env (FORCE=1 to rotate)
	@$(ROOT)/scripts/dev-keys.sh $(ROOT)/.env

# --- local dev stack ---------------------------------------------------------
.PHONY: up up-cache down reset obs-up obs-down ps logs
up: ## Start local deps (postgis, minio+init, mailpit)
	@$(COMPOSE) up -d --wait postgis minio minio-init mailpit

up-cache: ## Start local deps plus Valkey (profile cache)
	@$(COMPOSE) --profile cache up -d --wait

obs-up: ## Start local Prometheus + Grafana (scrape host api/worker)
	@$(COMPOSE) --profile observability up -d
	@echo "Grafana: http://localhost:3000 (anonymous viewer)   Prometheus: http://localhost:9091"

obs-down: ## Stop the local observability profile
	@$(COMPOSE) --profile observability down

down: ## Stop the local stack (keep volumes)
	@$(COMPOSE) --profile cache --profile observability down

reset: ## Stop the local stack and DELETE its volumes (fresh DB/buckets)
	@$(COMPOSE) --profile cache --profile observability down -v
	@echo "local volumes removed"

ps: ## Show local stack status
	@$(COMPOSE) ps

logs: ## Tail local stack logs
	@$(COMPOSE) logs -f --tail=100

# --- database ----------------------------------------------------------------
.PHONY: migrate migrate-new migrate-down seed
migrate: up ## Apply migrations to the local DB (rovo migrate up)
	@$(with_env) go -C $(BACKEND_DIR) run ./cmd/rovo migrate up

migrate-down: ## Roll the local DB back one migration (local only)
	@$(with_env) go -C $(BACKEND_DIR) run ./cmd/rovo migrate down

migrate-new: ## Create a timestamped migration: make migrate-new name=<module>_<desc>
	@test -n "$(name)" || { echo "usage: make migrate-new name=<module>_<desc>" >&2; exit 2; }
	@f="$(BACKEND_DIR)/migrations/$$(date -u +%Y%m%d%H%M%S)_$(name).sql"; \
	printf -- '-- +goose Up\n\n-- +goose Down\n' > "$$f"; \
	echo "created $$f"

seed: ## Load the Mahabubnagar demo dataset
	@$(with_env) go -C $(BACKEND_DIR) run ./cmd/rovo seed

# --- run ---------------------------------------------------------------------
.PHONY: run-api run-worker run-all dev
run-api: up ## Run the API on :8080 (internal :9090)
	@$(with_env) go -C $(BACKEND_DIR) run ./cmd/rovo api

run-worker: up ## Run the worker (internal :9092 for scraping)
	@$(with_env) ROVO_INTERNAL_ADDR=:9092 go -C $(BACKEND_DIR) run ./cmd/rovo worker

run-all: up ## Run api+worker in one process (rovo all)
	@$(with_env) go -C $(BACKEND_DIR) run ./cmd/rovo all

dev: ## Full stack: deps + migrate + seed + api + worker + 4 Vite apps
	@$(ROOT)/scripts/dev.sh

# --- codegen -----------------------------------------------------------------
.PHONY: generate
generate: ## Regenerate code from the OpenAPI spec (TS types, etc.)
	@if [ -f $(WEB_DIR)/package.json ]; then pnpm --dir $(WEB_DIR) run gen:api; else echo "web/ not present; skipping"; fi

# --- lint --------------------------------------------------------------------
.PHONY: lint lint-go lint-web lint-openapi lint-shell lint-docker lint-actions
lint: lint-go lint-web lint-openapi lint-shell lint-docker lint-actions ## Run every linter

lint-go: ## gofmt + go vet + golangci-lint
	@test -f $(BACKEND_DIR)/go.mod || { echo "backend/ not present; skipping"; exit 0; }
	@out=$$(gofmt -l $(BACKEND_DIR) | grep -v '/vendor/' || true); \
	 if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	@go -C $(BACKEND_DIR) vet ./...
	@if command -v golangci-lint >/dev/null; then (cd $(BACKEND_DIR) && golangci-lint run); \
	 else echo "golangci-lint not installed; run 'go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest'"; fi

lint-web: ## pnpm typecheck + eslint
	@test -f $(WEB_DIR)/package.json || { echo "web/ not present; skipping"; exit 0; }
	@pnpm --dir $(WEB_DIR) typecheck
	@pnpm --dir $(WEB_DIR) lint

lint-openapi: ## Redocly lint of the OpenAPI contract
	@test -f $(ROOT)/openapi/openapi.yaml || { echo "openapi/ not present; skipping"; exit 0; }
	@$(REDOCLY) lint $(ROOT)/openapi/openapi.yaml

lint-shell: ## shellcheck every script
	@docker run --rm -v $(ROOT):/mnt -w /mnt $(SHELLCHECK_IMG) -x scripts/*.sh deploy/compose/*.sh

lint-docker: ## hadolint the Dockerfiles
	@for f in $(ROOT)/deploy/docker/*.Dockerfile; do \
	  echo "== $$f"; docker run --rm -i $(HADOLINT_IMG) hadolint --failure-threshold warning - < "$$f"; done

lint-actions: ## actionlint the GitHub workflows
	@docker run --rm -v $(ROOT):/repo -w /repo $(ACTIONLINT_IMG) -color

# --- test --------------------------------------------------------------------
.PHONY: test test-integration test-e2e check
test: ## Go unit tests + vitest
	@test -f $(BACKEND_DIR)/go.mod && go -C $(BACKEND_DIR) test -race -shuffle=on ./... || echo "backend/ not present; skipping go tests"
	@if [ -f $(WEB_DIR)/package.json ]; then pnpm --dir $(WEB_DIR) test; else echo "web/ not present; skipping vitest"; fi

test-integration: up ## Go integration tests against a throwaway PostGIS
	@$(ROOT)/scripts/test-integration.sh

test-e2e: ## Playwright golden flow against compose.prod (e2e/)
	@test -d $(ROOT)/e2e || { echo "e2e/ not present yet (QA engineer owns it)"; exit 0; }
	@$(with_env) pnpm --dir $(ROOT)/e2e exec playwright test

check: lint test test-integration ## lint + unit + integration (what CI gates on)

# --- build / images ----------------------------------------------------------
.PHONY: build image image-web image-push
build: ## Build the backend binary and web bundles locally
	@test -f $(BACKEND_DIR)/go.mod && CGO_ENABLED=0 go -C $(BACKEND_DIR) build -trimpath \
	  -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)" \
	  -o $(BACKEND_DIR)/bin/rovo ./cmd/rovo && echo "built backend/bin/rovo" || echo "backend/ not present; skipping"
	@if [ -f $(WEB_DIR)/package.json ]; then for a in $(APPS); do pnpm --dir $(WEB_DIR) --filter "./apps/$$a" build; done; fi

image: ## Build the backend image locally (IMAGE=$(IMAGE))
	@docker buildx build --load -f $(ROOT)/deploy/docker/backend.Dockerfile \
	  --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) \
	  -t $(IMAGE) $(ROOT)

image-web: ## Build the edge (Caddy + 4 SPAs) image locally (WEB_IMAGE=$(WEB_IMAGE))
	@docker buildx build --load -f $(ROOT)/deploy/docker/web.Dockerfile \
	  --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t $(WEB_IMAGE) $(ROOT)

image-push: ## Build + push multi-arch images to the registry (CI)
	@docker buildx build --push --platform $(PLATFORMS) -f $(ROOT)/deploy/docker/backend.Dockerfile \
	  --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) \
	  -t $(REGISTRY):$(VERSION) -t $(REGISTRY):sha-$(COMMIT) $(ROOT)

# --- production-like stack ---------------------------------------------------
# TLS=off (default, plain HTTP) | internal (Caddy local CA) | acme
.PHONY: prod-up prod-down prod-seed prod-logs smoke
prod-up: ## Build + start the production-like stack (ADR-P2-003). TLS=off|internal
	@test -f $(ROOT)/.env || $(ROOT)/scripts/dev-keys.sh $(ROOT)/.env
	@tls="$${TLS:-off}"; scheme=http; [ "$$tls" != off ] && scheme=https; \
	  ROVO_CADDY_TLS=$$tls ROVO_SITE_SCHEME=$$scheme VERSION=$(VERSION) COMMIT=$(COMMIT) \
	  $(COMPOSE_PROD) --env-file $(ROOT)/.env up -d --build --wait
	@echo "edge up (TLS=$${TLS:-off}); add *.$${ROVO_DOMAIN:-rovo.localhost} to /etc/hosts -> 127.0.0.1"

prod-down: ## Stop the production-like stack (DOWN_ARGS=-v to wipe volumes)
	@$(COMPOSE_PROD) --env-file $(ROOT)/.env down $(DOWN_ARGS)

prod-seed: ## Load demo data into the production-like stack
	@$(COMPOSE_PROD) --env-file $(ROOT)/.env run --rm --no-deps migrate seed || \
	 $(COMPOSE_PROD) --env-file $(ROOT)/.env exec -T api /usr/local/bin/rovo seed

prod-logs: ## Tail the production-like stack logs
	@$(COMPOSE_PROD) --env-file $(ROOT)/.env logs -f --tail=100

smoke: ## Run the edge smoke test (BASE=<domain>, default $$ROVO_DOMAIN)
	@$(with_env) $(ROOT)/scripts/smoke.sh $(BASE)

# --- backup / restore --------------------------------------------------------
.PHONY: backup restore verify-restore
backup: ## Logical backup of the local DB (BACKUP_DIR=./backups)
	@$(ROOT)/scripts/backup.sh --stack dev

restore: ## Restore a dump: make restore DUMP=backups/rovo-<ts>.dump.gz [CLEAN=1]
	@test -n "$(DUMP)" || { echo "usage: make restore DUMP=<file.dump.gz> [CLEAN=1]" >&2; exit 2; }
	@$(ROOT)/scripts/restore.sh $(DUMP) --stack dev $(if $(CLEAN),--clean,) --yes

verify-restore: ## Restore the latest backup into a throwaway DB and check invariants
	@$(ROOT)/scripts/verify-restore.sh

# --- IaC ---------------------------------------------------------------------
.PHONY: tf-validate tf-lint
tf-validate: ## OpenTofu fmt-check + validate (never apply)
	@$(ROOT)/scripts/tf-validate.sh

tf-lint: ## tflint + checkov on the IaC
	@docker run --rm -v $(TF_DIR):/data -w /data $(TFLINT_IMG) --recursive || true
	@docker run --rm -v $(TF_DIR):/tf $(CHECKOV_IMG) -d /tf --quiet --compact || true

# --- security ----------------------------------------------------------------
.PHONY: vuln
vuln: ## govulncheck + pnpm audit + trivy fs (osv-scanner if installed)
	@if command -v govulncheck >/dev/null; then (cd $(BACKEND_DIR) && govulncheck ./...); \
	 else echo "govulncheck not installed; run 'go install golang.org/x/vuln/cmd/govulncheck@latest'"; fi
	@if [ -f $(WEB_DIR)/pnpm-lock.yaml ]; then pnpm --dir $(WEB_DIR) audit --prod || true; fi
	@if command -v osv-scanner >/dev/null; then osv-scanner --lockfile=$(BACKEND_DIR)/go.sum || true; \
	 else docker run --rm -v $(ROOT):/src $(TRIVY_IMG) fs --scanners vuln,misconfig,secret --exit-code 0 /src; fi

.PHONY: graph graph-query
graph: ## Build/update the code knowledge graph (graphify)
	graphify update . --force
graph-query: ## Query the graph, e.g. make graph-query Q="what connects auth to the db?"
	graphify query "$(Q)"
