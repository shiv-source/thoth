#!/usr/bin/env bash
# graph-check.sh — staleness guard for the committed graphify graph.
#
# Exits 0 when graphify-out/graph.json is fresh (or missing — nothing to
# guard), 1 when the committed graph lags committed code or when uncommitted
# code changes exist. Callers (pr.sh, humans) should then run
# `graphify update .` and commit the refreshed graph — the committed graph
# must match the committed tree (CLAUDE.md § graphify, git-workflow skill
# workflow 2 step 5).
#
# Scope: code only (Go/TS/TSX — graphify's AST inputs; CLAUDE.md: "after
# modifying code"). Docs/.sh changes don't affect graph.json.
# Commit-based, not mtime-based: a `git switch` rewrites mtimes without
# changing content, which would false-positive a mtime check.
set -u

cd "$(dirname "$0")/.."

graph="graphify-out/graph.json"

# 0. Location: graphify initializes a new graph at any cwd, so a run from a
# subdirectory silently creates a nested graphify-out/ there (and queries
# from that dir then read the stub). There is exactly one graph: the root's.
nested="$(find . -type d -name graphify-out \
  -not -path './graphify-out' -not -path './graphify-out/*' \
  -not -path './node_modules/*' 2>/dev/null)"
if [ -n "$nested" ]; then
  echo "graph-check: nested graphify-out found — run graphify from the repo root and delete it:" >&2
  printf '%s\n' "$nested" | sed 's/^/  /' >&2
  exit 1
fi

if [ ! -f "$graph" ]; then
  echo "graph-check: $graph missing — nothing to guard" >&2
  exit 0
fi

# Code files only; generated dirs (bin/, web/dist/, internal/webui/dist/,
# node_modules/) are gitignored and never appear.
code=( '*.go' '*.ts' '*.tsx' )

# 1. Uncommitted changes to tracked code — the graph cannot reflect them.
dirty="$(git status --porcelain -- "${code[@]}" || true)"
if [ -n "$dirty" ]; then
  echo "graph-check: stale graph — uncommitted code changes not in the graph:" >&2
  printf '%s\n' "$dirty" | head -20 | sed 's/^/  /' >&2
  echo "Run: graphify update .  (then commit the refreshed graph)" >&2
  exit 1
fi

# 2. Code committed since the graph last changed — the graph lags the tree.
graph_commit="$(git log -1 --format=%H -- "$graph")"
if [ -n "$graph_commit" ]; then
  lagged="$(git log --format='%h %s' "$graph_commit..HEAD" -- "${code[@]}" || true)"
  if [ -n "$lagged" ]; then
    echo "graph-check: stale graph — code committed since the last graph refresh:" >&2
    printf '%s\n' "$lagged" | head -20 | sed 's/^/  /' >&2
    echo "Run: graphify update .  (then commit the refreshed graph)" >&2
    exit 1
  fi
fi

exit 0
