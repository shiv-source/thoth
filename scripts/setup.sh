#!/usr/bin/env bash
# setup.sh — one-command developer setup for Thoth.
# Run from anywhere: ./scripts/setup.sh (cds to the repo root itself).
# Toolchain versions: go.mod (Go) and CLAUDE.md (Node via nvm, pnpm) are
# authoritative — this script checks presence and warns on major mismatch.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "== 1/4 Toolchain =="
if ! command -v go >/dev/null 2>&1; then
  echo "go not found — install it (see go.mod for the version)" >&2
  exit 1
fi
echo "go:      $(go version)"

if ! command -v node >/dev/null 2>&1; then
  echo "node not found — expected via nvm (CLAUDE.md pins the version)" >&2
  exit 1
fi
echo "node:    $(node --version)"

node_major="$(node --version | sed 's/^v//' | cut -d. -f1)"
[ "$node_major" = "24" ] || echo "warning: node major $node_major — CLAUDE.md pins 24" >&2

if ! command -v pnpm >/dev/null 2>&1; then
  echo "pnpm not found — enable corepack or install it (CLAUDE.md pins the version)" >&2
  exit 1
fi
echo "pnpm:    $(pnpm --version)"
pnpm_major="$(pnpm --version | cut -d. -f1)"
[ "$pnpm_major" = "11" ] || echo "warning: pnpm major $pnpm_major — CLAUDE.md pins 11" >&2

if ! command -v git >/dev/null 2>&1; then
  echo "git not found — required" >&2
  exit 1
fi
echo "git:     $(git --version)"

if ! command -v gh >/dev/null 2>&1; then
  echo "warning: gh CLI not found — required for the PR workflow (git-workflow skill)" >&2
else
  echo "gh:      $(gh --version | head -1)"
  gh api user >/dev/null 2>&1 || echo "warning: gh API access failing — run gh auth login" >&2
fi

if ! command -v air >/dev/null 2>&1; then
  echo "warning: air not found — make dev hot-reloads Go with it (go install github.com/air-verse/air@latest)" >&2
else
  echo "air:     installed ($(command -v air))"
fi

echo "== 2/4 Dependencies =="
pnpm install --frozen-lockfile   # also runs husky's prepare hook
go mod download

echo "== 3/4 Frontend embed =="
make web                          # required before go build/test

echo "== 4/4 Doctor =="
make doctor

echo "Setup complete. Next: make check (all gates) or make dev (Vite HMR + server)."
