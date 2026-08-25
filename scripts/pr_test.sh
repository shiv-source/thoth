#!/usr/bin/env bash
# pr_test.sh — smoke test for scripts/pr.sh's main-sync step.
# Builds a throwaway clone, runs pr.sh from a feature branch with gh + a git
# call-logger in PATH, and asserts the sync step uses `git switch main` +
# `git pull` (never `git fetch origin`).
# Run: ./scripts/pr_test.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REAL_GIT="$(command -v git)"

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

# git call-logger: records sync-relevant invocations, passes everything else
# through to the real git.
mkdir -p "$WORK/bin"
cat > "$WORK/bin/git" <<SHIM
#!/usr/bin/env bash
LOG="\${GIT_SHIM_LOG:?}"
case "\$1" in
  switch) [ "\${2:-}" = "main" ] && echo "switch main" >> "\$LOG" ;;
  fetch) [ "\${2:-}" = "origin" ] && echo "fetch origin" >> "\$LOG" ;;
  pull) echo "pull" >> "\$LOG" ;;
esac
exec "$REAL_GIT" "\$@"
SHIM

# gh stub: auth + PR-create succeed; PR-view reports "no PR" unless it asks
# for a url, in which case it returns one (push_and_open's create path).
cat > "$WORK/bin/gh" <<SHIM
#!/usr/bin/env bash
case "\$1 \$2" in
  "api user") exit 0 ;;
  "pr view")
    for a in "\$@"; do
      [ "\$a" = "url" ] && { echo "https://github.com/shiv-source/thoth/pull/1"; exit 0; }
    done
    exit 1
    ;;
  "pr create") exit 0 ;;
esac
exit 0
SHIM
chmod +x "$WORK/bin/git" "$WORK/bin/gh"

# Seed a src repo with the scripts + labels + PR template committed.
mkdir -p "$WORK/src"
git -C "$WORK/src" init -q
mkdir -p "$WORK/src/scripts" "$WORK/src/.claude/skills/git-workflow/references" "$WORK/src/.github"
cp "$ROOT/scripts/lib-codegraph.sh" "$ROOT/scripts/pr.sh" "$WORK/src/scripts/"
cp "$ROOT/.claude/skills/git-workflow/references/labels.md" "$WORK/src/.claude/skills/git-workflow/references/labels.md"
cp "$ROOT/.github/pull_request_template.md" "$WORK/src/.github/pull_request_template.md"
git -C "$WORK/src" add -A
git -C "$WORK/src" commit -q -m seed
git -C "$WORK/src" branch -M main

# Normal clone: sync switches to main and pulls.
git clone -q "$WORK/src" "$WORK/normal"
git -C "$WORK/normal" checkout -q -b feat/test/guide
check "normal layout syncs via switch main + pull" bash -c "
  cd '$WORK/normal' &&
  GIT_SHIM_LOG='$WORK/calls-normal.log' PATH='$WORK/bin:$PATH' '$WORK/normal/scripts/pr.sh' --no-check &&
  grep -qx 'switch main' '$WORK/calls-normal.log' &&
  grep -qx 'pull' '$WORK/calls-normal.log'
"

echo
echo "pr_test.sh: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
