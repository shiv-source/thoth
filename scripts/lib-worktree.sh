#!/usr/bin/env bash
# lib-worktree.sh — shared helpers for the bare-clone layout. Source-only;
# nothing here runs on load. Consumers: git-worktree.sh, pr.sh.
#
# The "pro setup" keeps git history in a hidden .bare clone at a container
# root that also holds a `.git` gitfile ("gitdir: ./.bare"); one working
# directory per branch lives as a sibling of that root.

# find_container walks up from the current dir to the bare-clone container
# root: the directory that holds both the .bare clone and the .git gitfile
# ("gitdir: ./.bare"). Prints the path or exits 1.
find_container() {
  local dir
  dir="$(pwd)"
  while [ -n "$dir" ]; do
    if [ -d "$dir/.bare" ] && [ -f "$dir/.git" ] && grep -q '^gitdir: \./\.bare' "$dir/.git" 2>/dev/null; then
      echo "$dir"
      return 0
    fi
    [ "$dir" = "/" ] && break
    dir="$(dirname "$dir")"
  done
  return 1
}
