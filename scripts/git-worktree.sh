#!/usr/bin/env bash
# git-worktree.sh — worktree helper for the bare-clone layout.
#
# The "pro setup" keeps the git history in a hidden .bare clone and creates
# one working directory per branch (git-workflow skill). This script wraps
# `git worktree` so the two conventions stay consistent:
#   - worktrees live as siblings of the container root (e.g. feat-api-x)
#   - a branch <type>/<scope>/<slug> maps to a flat-hyphen dir <type>-<scope>-<slug>
#
# Usage:
#   git-worktree.sh new <type>/<scope>/<slug> [--base <ref>]   create a branch + worktree
#   git-worktree.sh list                                       list worktrees (git worktree list)
#   git-worktree.sh rm <dir-or-branch> [--force]               remove a worktree and delete its branch
#   git-worktree.sh help
#
# Run from anywhere inside the container (the root that holds .bare + .git,
# or any worktree under it). New worktrees inherit the container's
# opencode.json (MCP config) when one exists.
set -euo pipefail

# find_container lives in the shared lib (pr.sh uses it too).
# shellcheck source=./lib-worktree.sh
source "$(dirname "$0")/lib-worktree.sh"

# codegraph_init lives in the shared lib (pr.sh uses it too).
# shellcheck source=./lib-codegraph.sh
source "$(dirname "$0")/lib-codegraph.sh"

# Conventional-commit prefixes — mirrors git-workflow skill workflow 1.
VALID_TYPES="feat fix perf ci docs refactor test chore"

die() {
  echo "git-worktree: $*" >&2
  exit 1
}

# parse_branch validates <type>/<scope>/<slug> and fills BRANCH_TYPE/SCOPE/SLUG.
parse_branch() {
  local branch="$1"
  case "$branch" in
    */*/*) BRANCH_TYPE="${branch%%/*}"; BRANCH_SCOPE="${branch#*/}"; BRANCH_SLUG="${BRANCH_SCOPE#*/}"; BRANCH_SCOPE="${BRANCH_SCOPE%%/*}" ;;
    *) die "'$branch' does not match <type>/<scope>/<slug> — see the git-workflow skill workflow 1" ;;
  esac
  case " $VALID_TYPES " in
    *" $BRANCH_TYPE "*) : ;;
    *) die "type '$BRANCH_TYPE' is not one of: $VALID_TYPES" ;;
  esac
  printf '%s' "$BRANCH_SCOPE" | grep -qE '^[a-z0-9]+(-[a-z0-9]+)*$' \
    || die "scope '$BRANCH_SCOPE' must be lowercase kebab-case"
  printf '%s' "$BRANCH_SLUG" | grep -qE '^[a-z0-9]+(-[a-z0-9]+)*$' \
    || die "slug '$BRANCH_SLUG' must be lowercase kebab-case"
}

# flat_dir prints the worktree dir for a branch: <type>-<scope>-<slug>.
flat_dir() {
  echo "${BRANCH_TYPE}-${BRANCH_SCOPE}-${BRANCH_SLUG}"
}

# copy_config copies opencode.json (MCP config) into a fresh worktree so the
# MCP servers (playwright, antd, …) work there without a manual copy. The
# config is per-worktree: the container root holds none, so fall back to the
# main worktree's copy.
copy_config() {
  local root="$1" dir="$2" src
  if [ -f "$root/opencode.json" ]; then
    src="$root/opencode.json"
  elif [ -f "$root/main/opencode.json" ]; then
    src="$root/main/opencode.json"
  else
    return 0
  fi
  cp "$src" "$dir/opencode.json"
  echo "git-worktree: copied opencode.json into $dir"
}

# codegraph_init (lib-codegraph.sh) runs `codegraph init` in the fresh
# worktree so the CodeGraph MCP is active there — indexing is per-worktree
# (the container root and main worktree are not indexed by default).
# Best-effort: a missing or failing codegraph must never block creating the
# worktree.

cmd_new() {
  local branch="$1" base="origin/main"
  while [ $# -gt 1 ]; do
    case "$2" in
      --base) [ $# -ge 3 ] || die "--base needs a value"; base="$3"; shift 2 ;;
      *) die "unknown argument '$2' — usage: git-worktree.sh new <type>/<scope>/<slug> [--base <ref>]" ;;
    esac
  done
  parse_branch "$branch"
  local dir root
  dir="$(flat_dir)"
  root="$(find_container)"
  [ -e "$root/$dir" ] && die "worktree dir '$dir' already exists"
  git -C "$root" worktree add -b "$branch" "$dir" "$base"
  echo "git-worktree: created $root/$dir (branch $branch, base $base)"
  copy_config "$root" "$root/$dir"
  codegraph_init "$root/$dir"
}

cmd_rm() {
  local target="$1" force=0
  shift
  for arg in "$@"; do
    case "$arg" in
      --force) force=1 ;;
      *) die "unknown argument '$arg' — usage: git-worktree.sh rm <dir-or-branch> [--force]" ;;
    esac
  done
  local root
  root="$(find_container)"
  local dir branch
  # Accept a worktree dir (flat or nested) or a branch name. Compare paths
  # canonically: /tmp on macOS is a symlink to /private/tmp, and `git worktree
  # list --porcelain` emits canonical paths. A porcelain block is
  #   worktree <path>        ← remember the path for this block
  #   HEAD <sha>
  #   branch refs/heads/<n>  ← the branch for this block
  if [ -d "$root/$target" ]; then
    dir="$(cd "$root/$target" && pwd -P)"
    branch="$(git -C "$root" worktree list --porcelain | awk -v d="$dir" '
      $1=="worktree" { wt=$2; next }
      $1=="branch" && wt==d { sub(/^branch refs\/heads\//, ""); print; exit }
    ')"
    [ -n "$branch" ] || die "no worktree at $dir — run 'git-worktree.sh list'"
  elif git -C "$root" show-ref --verify --quiet "refs/heads/$target"; then
    branch="$target"
    dir="$(git -C "$root" worktree list --porcelain | awk -v b="refs/heads/$target" '
      $1=="worktree" { wt=$2; next }
      $1=="branch" && $2==b { print wt; exit }
    ')"
    [ -n "$dir" ] || die "branch '$target' is not checked out in any worktree"
  else
    die "'$target' is neither a worktree dir nor a branch"
  fi
  if [ "$force" = 1 ]; then
    git -C "$root" worktree remove --force "$dir"
    git -C "$root" branch -D "$branch"
  else
    # A freshly created worktree carries the copied opencode.json and (when
    # CodeGraph auto-indexed it) a .codegraph/ dir as untracked files; those
    # are our own bookkeeping, not a reason to refuse removal. Remove with
    # --force only when no other dirty files exist.
    local other_dirty
    other_dirty="$(git -C "$dir" status --porcelain | grep -v '^?? opencode.json$' | grep -v '^?? .codegraph/' || true)"
    if [ -z "$other_dirty" ]; then
      git -C "$root" worktree remove --force "$dir"
    else
      git -C "$root" worktree remove "$dir"
    fi
    git -C "$root" branch -d "$branch"
  fi
  echo "git-worktree: removed $dir and deleted branch $branch"
}

cmd_list() {
  local root
  root="$(find_container)"
  git -C "$root" worktree list
}

usage() {
  sed -n '2,/^#   git-worktree.sh help$/p' "$0" | sed 's/^# \{0,1\}//'
}

# find_container must succeed for every subcommand except help.
if [ "${1:-}" = "help" ] || [ $# -lt 1 ]; then
  usage
  exit 0
fi

if ! find_container >/dev/null; then
  die "not inside a bare-clone container (no '.bare' + '.git' gitfile found) — run this from the worktree root"
fi

case "$1" in
  new) shift; [ $# -ge 1 ] || die "usage: git-worktree.sh new <type>/<scope>/<slug> [--base <ref>]"; cmd_new "$@" ;;
  list) cmd_list ;;
  rm) shift; [ $# -ge 1 ] || die "usage: git-worktree.sh rm <dir-or-branch> [--force]"; cmd_rm "$@" ;;
  *) usage; die "unknown command '$1'" ;;
esac
