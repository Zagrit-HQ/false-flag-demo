.PHONY: help test test-all test-all-reseed test-go test-js lint lint-go lint-js typecheck build build-js \
	generate generate-go generate-js generate-check contract-test smoke api-dev \
	dashboard-dev hurl build-go vet bake-print up down logs ps restart nuke \
	kind-smoke conformance dashboard-e2e seed mcp-smoke

COMPOSE := docker compose

GO_TOOL := go tool

help:
	@echo "Common targets:"
	@echo "  make test           Run Go and TypeScript unit tests"
	@echo "  make test-all       Run unit + integration + e2e (requires \`make up && make seed\`)"
	@echo "  make lint           Run Go and TypeScript linters"
	@echo "  make typecheck      Run TypeScript typechecking"
	@echo "  make build          Build Go binaries and TypeScript packages"
	@echo "  make generate       Run all code generation"
	@echo "  make smoke          Run API and demo smoke checks"
	@echo "  make mcp-smoke      Run MCP tool-call smoke against the running compose stack"
	@echo "  make kind-smoke     End-to-end operator smoke (requires kind + running API)"
	@echo "  make conformance    Run SDK conformance suite (Go + TS, shared corpus)"
	@echo "  make dashboard-e2e  Run Playwright happy-path against running compose stack"
	@echo "  make seed           Seed the local Postgres with the demo dataset"
	@echo "  make api-dev        Run the API during implementation"
	@echo "  make dashboard-dev  Run the Remix dashboard"
	@echo "  make bake-print     Validate docker-bake.hcl"
	@echo "  make up             Build and start the full stack via docker compose"
	@echo "  make down           Stop the stack (keeps volumes)"
	@echo "  make nuke           Stop the stack and delete volumes"
	@echo "  make logs           Tail logs from the running stack"
	@echo "  make ps             Show docker compose service status"

test: test-go test-js

# test-all is the full ladder: unit tests, real-Postgres integration
# tests (contract, store), the cross-runtime conformance corpus, the
# Hurl HTTP e2e suite (smoke + mcp-smoke), and the Playwright
# dashboard happy-path. Each step in turn requires more from the
# environment:
#
#   * test-go / test-js          — hermetic
#   * contract-test / store IT   — needs Postgres at FALSEFLAG_TEST_DATABASE_URL
#   * smoke / mcp-smoke / e2e    — needs `make up && make seed`
#
# Targets are intentionally sequential: each layer asserts state the
# next layer depends on, and the goal of this aggregate is total
# walltime visibility (CI sharding demo), not throughput.
test-all: test-go test-js conformance contract-test smoke mcp-smoke test-all-reseed dashboard-e2e

# test-all-reseed re-seeds the demo dataset after smoke.sh truncates
# everything, so dashboard-e2e finds projects + flags to navigate.
test-all-reseed:
	go run ./cmd/falseflag-seed

test-go:
	$(GO_TOOL) gotestsum --format pkgname -- ./...

test-js:
	cd js && pnpm test

lint: lint-go lint-js

lint-go:
	$(GO_TOOL) golangci-lint run ./...

lint-js:
	cd js && pnpm lint

vet:
	go vet ./...

typecheck:
	cd js && pnpm typecheck

build: build-go build-js

build-go:
	go build ./cmd/...

build-js:
	cd js && pnpm build

generate: generate-go generate-js

generate-go:
	$(GO_TOOL) buf generate
	$(GO_TOOL) sqlc generate
	$(GO_TOOL) controller-gen object paths=./operator/api/...
	$(GO_TOOL) controller-gen crd paths=./operator/api/... output:crd:dir=deploy/crds
	cd api/openapi && $(GO_TOOL) oapi-codegen -config cfg.yaml openapi.yaml

generate-js:
	cd js && pnpm -r generate

# generate-check regenerates everything and fails when the working
# tree is dirty afterwards — i.e. when generated artifacts have
# drifted from their sources. CI gate; locally use it before pushing.
generate-check: generate
	git diff --exit-code

# contract-test runs the REST↔Connect parity test under a live
# Postgres. FALSEFLAG_TEST_DATABASE_URL is required.
contract-test:
	@if [ -z "$$FALSEFLAG_TEST_DATABASE_URL" ]; then \
		echo "set FALSEFLAG_TEST_DATABASE_URL"; exit 2; \
	fi
	$(GO_TOOL) gotestsum --format pkgname -- ./internal/server/... -run TestRESTConnectParity

smoke:
	./scripts/smoke.sh

# smoke-sqlite runs the same Hurl suite against the SQLite-backed
# stack (compose.sqlite.yaml). Wipes the SQLite volume between runs.
smoke-sqlite:
	FALSEFLAG_BACKEND=sqlite ./scripts/smoke.sh

# mcp-smoke exercises the MCP server tool surface end-to-end through
# Hurl: initialize handshake → tools/list → tools/call list_projects
# → tools/call validate_config. Requires `make up && make seed`.
mcp-smoke:
	./scripts/mcp-smoke.sh

# kind-smoke spins up a kind cluster, installs the operator, applies
# the sample manifests, asserts the upstream API receives them, then
# tears down. Best-effort: requires kind + docker + a running API.
kind-smoke:
	./scripts/kind-smoke.sh

# conformance runs the shared 25-fixture corpus against the Go SDK
# and the TypeScript SDK. Both runtimes must return byte-identical
# Decision JSON for every fixture.
conformance:
	$(GO_TOOL) gotestsum --format pkgname -- ./internal/sdkgo/... -run TestConformance
	cd js && pnpm --filter @falseflag/sdk test -- conformance

# dashboard-e2e runs the Playwright happy-path against a dashboard
# pointing at a running compose stack (make up && make seed). Headless
# Chromium auto-downloads on first run.
dashboard-e2e:
	cd js && pnpm --filter @falseflag/dashboard test:e2e

# seed populates the compose Postgres with the demo dataset (3 projects,
# 25 flags, environments, segments, audit events). Idempotent.
seed:
	go run ./cmd/falseflag-seed

# `make hurl` assumes an API process is already running on :8080.
hurl:
	hurl --test tests/hurl/*.hurl

api-dev:
	go run ./cmd/falseflag-api

dashboard-dev:
	cd js && pnpm --filter @falseflag/dashboard dev

bake-print:
	cd infra && docker buildx bake --print

bake:
	cd infra && docker buildx bake

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

nuke:
	$(COMPOSE) down -v

restart:
	$(COMPOSE) restart

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps
