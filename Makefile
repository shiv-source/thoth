# =============================================================================
# Thoth — build, development, and release automation.
# Run `make help` for the self-documenting target list.
# =============================================================================

SHELL       := /bin/bash
BIN         := bin/thoth
DIST_DIR    := dist
EMBED_DIST  := internal/webui/dist
PREFIX      ?= /usr/local/bin
MODULE      := github.com/shiv-source/thoth
VERSION     ?= dev
LDFLAGS     := -s -w -X $(MODULE)/internal/cli.version=$(VERSION)
GO_BUILD    := go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)"

.DEFAULT_GOAL := help

# The web target swaps the embed (rm -rf + cp) while go test/build read it —
# never parallelize.
.NOTPARALLEL:

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

help: ## list available targets with descriptions
.PHONY: help
help:
	@awk -F':.*## ' '/^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# -----------------------------------------------------------------------------
# Setup — the embedded frontend
# -----------------------------------------------------------------------------

web: ## build the frontend and sync it into the Go embed
.PHONY: web
web:
	pnpm install --frozen-lockfile
	pnpm build
	rm -rf $(EMBED_DIST)
	cp -r web/dist $(EMBED_DIST)

web-sync: ## ensure the embed exists without reinstalling (dev fast path)
.PHONY: web-sync
web-sync:
	@test -f $(EMBED_DIST)/index.html || $(MAKE) --no-print-directory web

# -----------------------------------------------------------------------------
# Documentation site — Docusaurus (docs-site/), consumes docs/ directly
# -----------------------------------------------------------------------------

docs-install: ## install docs-site dependencies (workspace root lockfile)
.PHONY: docs-install
docs-install:
	pnpm install --frozen-lockfile

docs-dev: ## run the Docusaurus dev server with hot reload
.PHONY: docs-dev
docs-dev:
	pnpm docs:dev

docs-build: ## build the static docs site into docs-site/build/
.PHONY: docs-build
docs-build:
	pnpm docs:build

# -----------------------------------------------------------------------------
# Development — live servers with hot reload
# -----------------------------------------------------------------------------

dev: web-sync ## run Vite (HMR) and the Go server (air hot-reload) on :8334; Ctrl+C stops both
	@command -v air >/dev/null 2>&1 || { echo "air not found — install it: go install github.com/air-verse/air@latest" >&2; exit 1; }
	@trap 'kill 0' EXIT; \
	( THOTH_PORT=8334 pnpm dev ) & vite=$$!; \
	air & server=$$!; \
	while kill -0 $$vite 2>/dev/null && kill -0 $$server 2>/dev/null; do sleep 1; done; \
	echo "dev: one of the dev processes exited — shutting down" >&2
.PHONY: dev

dev-web: ## Vite dev server only (proxies /api and /ws to 127.0.0.1:8333)
.PHONY: dev-web
dev-web:
	pnpm dev

dev-server: web ## Go server only, with the embedded frontend
.PHONY: dev-server
dev-server:
	go run ./cmd/thoth serve

# -----------------------------------------------------------------------------
# Build & release
# -----------------------------------------------------------------------------

build: web ## compile the release binary (VERSION=v1.2.3 to stamp it)
	go mod download
	$(GO_BUILD) -o $(BIN) ./cmd/thoth
.PHONY: build

install-bin: build ## build and copy the binary to $(PREFIX) (default /usr/local/bin)
	install -d -m 0755 $(PREFIX)
	install -m 0755 $(BIN) $(PREFIX)/thoth
.PHONY: install-bin

release: guard-release web test ## cross-compile all five targets into dist/ (VERSION=vX.Y.Z required)
	@mkdir -p $(DIST_DIR)
	@for target in \
	  "darwin amd64" "darwin arm64" "linux amd64" "linux arm64" "windows amd64"; do \
	    set -- $$target; \
	    out=$(DIST_DIR)/thoth-$$1-$$2$$( [ $$1 = windows ] && echo .exe ); \
	    echo "  building $$out"; \
	    GOOS=$$1 GOARCH=$$2 $(GO_BUILD) -o $$out ./cmd/thoth; \
	done
.PHONY: release

# First prereq so the guard fires before web/test run
.PHONY: guard-release
guard-release:
	@[ "$(VERSION)" != "dev" ] && [ -n "$(VERSION)" ] || { echo "FAIL: release needs a real version — run: make release VERSION=v1.2.3" >&2; exit 1; }

# -----------------------------------------------------------------------------
# Quality gates — what CI enforces (check runs them in this order)
# -----------------------------------------------------------------------------

fmt: web-sync ## format and autofix the Go tree (gofmt/goimports + autofixable linters)
.PHONY: fmt
fmt:
	golangci-lint run --fix

lint: web-sync ## backend and frontend static analysis
.PHONY: lint
lint:
	golangci-lint run
	pnpm lint
	pnpm typecheck

test: web-sync ## unit tests (fail-fast)
.PHONY: test
test:
	go test -failfast ./...

race: web-sync ## tests under the race detector (fail-fast)
.PHONY: race
race:
	go test -race -failfast ./...

# COVER_PKGS is the coverage gate scope: the whole Go backend — the
# native-agent epic (the agent module plus the internal packages it drives)
# and every other internal package plus cmd/. A change to any Go code must
# not drag the shared 90% floor below the gate.
COVER_PKGS := ./agent/... ./internal/... ./cmd/...

cover: web-sync ## coverage report with the CI gate (>= 90% on the Go backend)
.PHONY: cover
cover:
	go test -failfast -coverprofile=coverage.out $(COVER_PKGS)
	@go tool cover -func=coverage.out | awk -F'\t' '/^total:/ { printf "total: %s\n", $$NF; gsub(/%/,"",$$NF); total=$$NF } END { if (total=="") { print "FAIL: no total coverage line — coverage.out missing or corrupt"; exit 1 } if (total+0 < 90) { print "FAIL: coverage below 90% floor"; exit 1 } }'

web-test: ## frontend unit tests
.PHONY: web-test
web-test:
	pnpm test

tools-test: ## .github/actions/issue-labels JS suite + scripts/ smoke tests
.PHONY: tools-test
tools-test:
	node --test .github/actions/issue-labels/test/*.test.mjs
	./scripts/git-worktree_test.sh
	./scripts/pr_test.sh

check: fmt lint race cover web-test tools-test build ## everything CI runs, locally
.PHONY: check

# -----------------------------------------------------------------------------
# Run
# -----------------------------------------------------------------------------

run: build ## build and serve
	./$(BIN) serve
.PHONY: run

# Fast repeated starts: rebuilds only the Go binary and reuses the existing
# embed. Run `make web` after frontend edits to refresh the embedded UI.
run-fast: web-sync ## rebuild Go only and serve (reuse existing embed — run `make web` after frontend edits)
	$(GO_BUILD) -o $(BIN) ./cmd/thoth
	./$(BIN) serve
.PHONY: run-fast

# -----------------------------------------------------------------------------
# Maintenance
# -----------------------------------------------------------------------------

doctor: web-sync ## diagnose the local Thoth setup
.PHONY: doctor
doctor:
	go run ./cmd/thoth doctor

init: web-sync ## scaffold the default wiki
.PHONY: init
init:
	go run ./cmd/thoth init

clean: ## remove all build output
.PHONY: clean
clean:
	rm -rf $(BIN) $(DIST_DIR) $(EMBED_DIST) web/dist coverage.out
