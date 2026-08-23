#!/usr/bin/env bash
# lib-codegraph.sh — shared CodeGraph helpers for the repo scripts. Source-only;
# nothing here runs on load. Consumers: git-worktree.sh, pr.sh.
#
# Both helpers are best-effort by design: a missing or failing codegraph must
# never block a git operation — the worktree/PR flows carry on without an
# updated index (the same stance git-worktree.sh took when it inlined init).

# codegraph_available reports whether the codegraph CLI is on PATH.
codegraph_available() {
  command -v codegraph >/dev/null 2>&1
}

# codegraph_init builds a fresh index in dir (mirrors `codegraph init`).
# Prints a skip note and returns 0 when codegraph is missing or the init
# fails, so creating a worktree is never blocked by indexing.
codegraph_init() {
  local dir="$1"
  if ! codegraph_available; then
    echo "codegraph: not on PATH — $dir not indexed (run 'codegraph init $dir' later)" >&2
    return 0
  fi
  echo "codegraph: indexing $dir"
  codegraph init "$dir" || {
    echo "codegraph: init failed for $dir — continuing without an index" >&2
  }
}

# codegraph_sync refreshes an existing index in dir so it reflects the current
# tree (e.g. the branch's post-make-check state, before a push). No-op unless
# the index db exists and codegraph is installed; failures never block the
# caller.
codegraph_sync() {
  local dir="${1:-.}" db
  db="$dir/.codegraph/codegraph.db"
  if [ ! -e "$db" ]; then
    echo "codegraph: no $db — skipping sync"
    return 0
  fi
  if ! codegraph_available; then
    echo "codegraph: not on PATH — skipping sync"
    return 0
  fi
  echo "codegraph: syncing index ($db)"
  codegraph sync "$dir" || {
    echo "codegraph: sync failed for $dir — continuing without it" >&2
  }
}
