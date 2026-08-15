# =============================================================================
# Thoth — build, development, and release automation.
# Run `make help` for the self-documenting target list.
# =============================================================================

SHELL       := /bin/bash
GO          ?= go
PNPM        ?= pnpm
BIN         := bin/thoth
DIST_DIR    := dist
EMBED_DIST  := internal/webui/dist
PREFIX      ?= /usr/local/bin
MODULE      := github.com/shiv-source/thoth
VERSION     ?= dev
LDFLAGS     := -s -w -X $(MODULE)/internal/cli.version=$(VERSION)
GO_BUILD    := $(GO) build -ldflags "$(LDFLAGS)"

.DEFAULT_GOAL := help

# -----------------------------------------------------------------------------
# Utility
# -----------------------------------------------------------------------------

help: ## list available targets with descriptions
.PHONY: help
help:
	@awk -F':.*## ' '/^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# -----------------------------------------------------------------------------
# Setup & install (one command = everything: frontend deps → embed → binary)
# -----------------------------------------------------------------------------

install: web ## install frontend deps, build the UI, install the binary into $(GOBIN)
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/thoth
.PHONY: install

install-bin: build ## build and copy the binary to $(PREFIX) (default /usr/local/bin)
	install -m 0755 $(BIN) $(PREFIX)/thoth
.PHONY: install-bin

# -----------------------------------------------------------------------------
# Development — live servers with hot reload
# -----------------------------------------------------------------------------

dev: web-sync ## run Vite (HMR) and the Go server together; Ctrl+C stops both
	@trap 'kill 0' EXIT; \
	( $(PNPM) dev ) & \
	$(GO) run ./cmd/thoth serve
.PHONY: dev

dev-web: ## Vite dev server only (proxies /api and /ws to 127.0.0.1:8333)
.PHONY: dev-web
dev-web:
	$(PNPM) dev

.PHONY: dev-server
dev-server: web ## Go server only, with the embedded frontend
	$(GO) run ./cmd/thoth serve

# -----------------------------------------------------------------------------
# Build & release
# -----------------------------------------------------------------------------

web: ## build the frontend and sync it into the Go embed
.PHONY: web
web:
	$(PNPM) install --frozen-lockfile
	$(PNPM) build
	rm -rf $(EMBED_DIST)
	cp -r web/dist $(EMBED_DIST)

web-sync: ## ensure the embed exists without reinstalling (dev fast path)
.PHONY: web-sync
web-sync:
	@test -f $(EMBED_DIST)/index.html || $(MAKE) --no-print-directory web

build: web ## compile the release binary (VERSION=v1.2.3 to stamp it)
	$(GO_BUILD) -o $(BIN) ./cmd/thoth
.PHONY: build

release: web test ## cross-compile all five targets into dist/ (stamped with VERSION)
	@mkdir -p $(DIST_DIR)
	@for target in \
	  "darwin amd64" "darwin arm64" "linux amd64" "linux arm64" "windows amd64"; do \
	    set -- $$target; \
	    out=$(DIST_DIR)/thoth-$$1-$$2$$( [ $$1 = windows ] && echo .exe ); \
	    echo "  building $$out"; \
	    GOOS=$$1 GOARCH=$$2 $(GO_BUILD) -o $$out ./cmd/thoth; \
	done
.PHONY: release

# -----------------------------------------------------------------------------
# Quality gates — what CI enforces
# -----------------------------------------------------------------------------

fmt: ## format and autofix the Go tree (gofmt/goimports + autofixable linters)
.PHONY: fmt
fmt:
	golangci-lint run --fix

test: ## unit tests (fail-fast)
.PHONY: test
test:
	$(GO) test -failfast ./...

race: ## tests under the race detector (fail-fast)
.PHONY: race
race:
	$(GO) test -race -failfast ./...

cover: ## coverage report with the CI gate (>= 80%)
.PHONY: cover
cover:
	$(GO) test -failfast -coverprofile=coverage.out ./internal/... ./cmd/...
	$(GO) tool cover -func=coverage.out | tail -1
	@$(GO) tool cover -func=coverage.out | awk -F'\t' '/^total:/ { gsub(/%/,"",$$NF); if ($$NF+0 < 80) { print "FAIL: coverage below 80% floor"; exit 1 } }'

lint: ## backend and frontend static analysis
.PHONY: lint
lint:
	golangci-lint run
	$(PNPM) lint
	$(PNPM) typecheck

web-test: ## frontend unit tests
.PHONY: web-test
web-test:
	$(PNPM) test

check: fmt lint race cover web-test build ## everything CI runs, locally
.PHONY: check

# -----------------------------------------------------------------------------
# Operations
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

doctor: ## diagnose the local Thoth setup
.PHONY: doctor
doctor:
	$(GO) run ./cmd/thoth doctor

init: ## scaffold the default wiki
.PHONY: init
init:
	$(GO) run ./cmd/thoth init

clean: ## remove all build output
.PHONY: clean
clean:
	rm -rf $(BIN) $(DIST_DIR) $(EMBED_DIST) web/dist coverage.out
