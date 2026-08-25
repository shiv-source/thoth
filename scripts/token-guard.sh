#!/usr/bin/env bash
# token-guard.sh — read-guard for Claude Code hooks.
#
# Enforces the code-rules skill token-efficiency rule "Don't re-read what you just
# wrote": PostToolUse(Read) records every file read per session (record);
# PreToolUse(Read) warns once per file when a file is about to be re-read
# (check). Warnings never block — they nudge.
#
# Hook input JSON arrives on stdin (session_id + tool_input.file_path).
# Wire-up (local settings, see .claude/settings.json):
#   PreToolUse  Read → "$CLAUDE_PROJECT_DIR/scripts/token-guard.sh" check
#   PostToolUse Read → "$CLAUDE_PROJECT_DIR/scripts/token-guard.sh" record
set -u

# No hook JSON on stdin (manual invocation) — nothing to do. Without this,
# `cat` would block forever waiting for input outside the hook context.
if [ -t 0 ]; then
  exit 0
fi

input="$(cat)"
session_id="$(printf '%s' "$input" | sed -n 's/.*"session_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
path="$(printf '%s' "$input" | sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"

# session_id is interpolated into a file path — strip anything that could
# escape the state dir (path separators, dots, quotes, backslashes).
session_id="$(printf '%s' "$session_id" | tr -cd '[:alnum:]_-')"

[ -n "$session_id" ] || session_id="unknown"
[ -n "$path" ] || exit 0

state_dir="${TMPDIR:-/tmp}/claude-token-guard"
state_file="$state_dir/reads-$session_id.log"
mkdir -p "$state_dir"

mode="${1:-}"

case "$mode" in
  record)
    printf '%s\n' "$path" >> "$state_file"
    ;;
  check)
    if grep -qxF -- "$path" "$state_file" 2>/dev/null \
      && ! grep -qxF -- "warned:$path" "$state_file" 2>/dev/null; then
      echo "token-guard: about to re-read $path — the code-rules skill rule says don't re-read what you just wrote. Only continue if the earlier content was compacted away." >&2
      printf 'warned:%s\n' "$path" >> "$state_file"
    fi
    ;;
esac
exit 0
