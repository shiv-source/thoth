#!/usr/bin/env bash
# graph-check.sh — staleness guard for the committed graphify graph.
#
# Exits 0 when graphify-out/graph.json is fresh (or missing — nothing to
# guard), 1 when any tracked source file is newer than the graph.
# Callers (pr.sh, humans) should then run `graphify update .` and commit
# the refreshed graph — the committed graph must match the committed tree
# (CLAUDE.md § graphify, git-workflow skill workflow 2 step 5).
#
# Caveat: mtimes, not content. A fresh checkout or `git switch` rewrites
# mtimes, so the guard can look stale right after a switch — the remedy is
# the same one command.
set -u

cd "$(dirname "$0")/.."

graph="graphify-out/graph.json"

if [ ! -f "$graph" ]; then
  echo "graph-check: graphify-out/graph.json missing — nothing to guard" >&2
  exit 0
fi

# Tracked sources the graph covers, minus generated dirs (CLAUDE.md § Repo
# rules: bin/, web/dist/, internal/webui/dist/, node_modules/), the graph
# itself, .git/ (churns on every git op), and docs/specs/ (untracked design
# docs by convention — git-workflow skill workflow 5).
stale="$(find . -type f \
  \( -name '*.go' -o -name '*.ts' -o -name '*.tsx' -o -name '*.md' \
     -o -name '*.yaml' -o -name '*.yml' \) \
  -not -path './.git/*' \
  -not -path './bin/*' \
  -not -path './web/dist/*' \
  -not -path './internal/webui/dist/*' \
  -not -path './node_modules/*' \
  -not -path './graphify-out/*' \
  -not -path './docs/specs/*' \
  -newer "$graph" \
  -print | head -20)"

if [ -n "$stale" ]; then
  echo "graph-check: stale graph — newer than $graph:" >&2
  printf '%s\n' "$stale" | sed 's/^/  /' >&2
  [ "$(printf '%s\n' "$stale" | wc -l | tr -d ' ')" -ge 20 ] \
    && echo "  … and more" >&2
  echo "Run: graphify update .  (then commit the refreshed graph)" >&2
  exit 1
fi

exit 0
