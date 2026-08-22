#!/usr/bin/env bash
# git-worktree_test.sh — smoke tests for scripts/git-worktree.sh.
# Builds a throwaway bare-clone container in a temp dir and exercises the
# new/list/rm commands, their guards, and running from inside a worktree.
# Run: ./scripts/git-worktree_test.sh
set -euo pipefail

SCRIPT="$(cd "$(dirname "$0")" && pwd)/git-worktree.sh"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

check() {
  # check <name> <cmd...> — asserts the command exits 0
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then
    pass=$((pass + 1)); echo "ok   - $name"
  else
    fail=$((fail + 1)); echo "FAIL - $name"
  fi
}

check_fails() {
  # check_fails <name> <cmd...> — asserts the command exits non-zero
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then
    fail=$((fail + 1)); echo "FAIL - $name (expected non-zero)"
  else
    pass=$((pass + 1)); echo "ok   - $name"
  fi
}

# Seed a bare clone with one main commit, plus the container .git gitfile.
mkdir -p "$WORK/src"
git -C "$WORK/src" init -q
git -C "$WORK/src" commit -q --allow-empty -m seed
git -C "$WORK/src" branch -M main
git clone -q --bare "$WORK/src" "$WORK/wt/.bare"
cd "$WORK/wt"
echo "gitdir: ./.bare" > .git
git config remote.origin.fetch "+refs/heads/*:refs/remotes/origin/*"
git fetch -q origin
git remote set-head origin -a >/dev/null 2>&1 || true

# opencode.json lives in the main worktree, not the container root — a new
# worktree should inherit it (the copy_config fallback).
git -C "$WORK/wt" worktree add -q main >/dev/null 2>&1
echo '{"mcp":{"x":{"type":"local","command":["true"]}}}' > "$WORK/wt/main/opencode.json"

check "list runs on a fresh container" "$SCRIPT" list
check "new creates a flat-hyphen worktree" "$SCRIPT" new feat/test/demo
[ -d "$WORK/wt/feat-test-demo" ] && pass=$((pass + 1)) && echo "ok   - worktree dir feat-test-demo exists" || { fail=$((fail + 1)); echo "FAIL - worktree dir feat-test-demo exists"; }
check "new copies opencode.json from the main worktree" bash -c "test -f '$WORK/wt/feat-test-demo/opencode.json'"
check "new works from inside a worktree" bash -c "cd '$WORK/wt/feat-test-demo' && '$SCRIPT' new docs/test/guide"
check_fails "new rejects a non-conventional branch" "$SCRIPT" new BAD/x/y
check_fails "new rejects a bad slug" "$SCRIPT" new feat/test/Weird_Name
check "rm by dir name" "$SCRIPT" rm feat-test-demo
check "rm by branch name" "$SCRIPT" rm docs/test/guide
"$SCRIPT" new fix/test/bug >/dev/null 2>&1
echo dirty > "$WORK/wt/fix-test-bug/scratch.txt"
check_fails "rm refuses a dirty worktree without --force" "$SCRIPT" rm fix-test-bug
check "rm --force removes a dirty worktree" "$SCRIPT" rm fix-test-bug --force
check_fails "rm rejects a nonexistent target" "$SCRIPT" rm nope
check_fails "rm never removes the bare repo" "$SCRIPT" rm .bare
check_fails "rm never removes the container root" "$SCRIPT" rm .
check_fails "list fails outside the container" bash -c "cd '$WORK' && '$SCRIPT' list"

echo
echo "git-worktree_test.sh: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
