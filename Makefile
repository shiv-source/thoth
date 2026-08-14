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
	( cd web && $(PNPM) dev ) & \
	$(GO) run ./cmd/thoth serve
.PHONY: dev

dev-web: ## Vite dev server only (proxies /api and /ws to 127.0.0.1:8333)
.PHONY: dev-web
dev-web:
	cd web && $(PNPM) dev

dev-server: web ## Go server only, with the embedded frontend
	$(GO) run ./cmd/thoth serve
.PHONY: dev-server

# -----------------------------------------------------------------------------
# Build & release
# -----------------------------------------------------------------------------

web: ## build the frontend and sync it into the Go embed
.PHONY: web
web:
	cd web && $(PNPM) install --frozen-lockfile && $(PNPM) run build
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

fmt: ## format and vet the Go tree
.PHONY: fmt
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './web/node_modules/*')
	$(GO) vet ./...

test: ## unit tests
.PHONY: test
test:
	$(GO) test ./...

race: ## tests under the race detector
.PHONY: race
race:
	$(GO) test -race ./...

cover: ## coverage report with the CI gate (>= 80%)
.PHONY: cover
cover:
	$(GO) test -coverprofile=coverage.out ./internal/... ./cmd/...
	$(GO) tool cover -func=coverage.out | tail -1
	@$(GO) tool cover -func=coverage.out | awk -F'\t' '/^total:/ { gsub(/%/,"",$$NF); if ($$NF+0 < 80) { print "FAIL: coverage below 80% floor"; exit 1 } }'

lint: ## backend and frontend static analysis
.PHONY: lint
lint:
	golangci-lint run
	cd web && $(PNPM) run lint && $(PNPM) exec tsc --noEmit

check: fmt lint race cover build ## everything CI runs, locally
.PHONY: check

# -----------------------------------------------------------------------------
# Operations
# -----------------------------------------------------------------------------

run: build ## build and serve
	./$(BIN) serve
.PHONY: run

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
