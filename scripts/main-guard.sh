#!/usr/bin/env bash
# main-guard.sh — block commits directly on main.
# Wired into .husky/pre-commit. Enforces code-rules skill § Repo rules ("never
# commit to it directly") — changes live on <type>/<scope>/<slug> branches
# and land via reviewed PRs that a human squash-merges (git-workflow skill
# workflows 1 and 6). Git runs pre-commit only for direct commits, not
# merges — that is fine here because main is never merged into locally.
set -u

cd "$(dirname "$0")/.."

branch="$(git branch --show-current)"

if [ "$branch" = "main" ]; then
  echo "main-guard: refusing commit on main — changes live on branches (code-rules skill § Repo rules): git switch -c <type>/<scope>/<slug>" >&2
  exit 1
fi

exit 0
