#!/usr/bin/env bash
# pr.sh — one command for the whole Thoth contribute flow.
# Run from anywhere: ./scripts/pr.sh [--no-check] [--title <title>] [--area <label>]…
#
# Automates git-workflow skill workflows 1/3/4 from an existing branch:
# preflight → sync with main → branch-name check →
# label derivation (parsed from references/labels.md) → make check → push →
# gh pr create with the template. The steps in the skill remain
# authoritative when run by hand.
#
# A session never merges — the human merges (git-workflow skill workflow 6).
# Limitation: the branch itself is not rebased onto main, only main is
# synced (workflow 1) — merge conflicts surface at CI, the human resolves.
set -euo pipefail

cd "$(dirname "$0")/.."

# codegraph_sync (the pre-push index refresh) lives in the shared lib too.
# shellcheck source=./lib-codegraph.sh
source "$(dirname "$0")/lib-codegraph.sh"

LABELS_MD=".claude/skills/git-workflow/references/labels.md"

TYPE_LABELS=""
AREA_LABELS=""
NO_CHECK=0
TITLE_OVERRIDE=""
AREA_EXTRA=""

BRANCH_TYPE=""
BRANCH_SCOPE=""
BRANCH_SLUG=""
TITLE=""
LABELS=""

die() {
  echo "pr: $*" >&2
  exit 1
}

# The 8 conventional-commit prefixes — git-workflow skill workflow 1 step 2;
# keep in sync with the skill when the prefix list changes.
VALID_TYPES="feat fix perf ci docs refactor test chore"

load_label_sets() {
  # Parses the | label | rows under ## Types / ## Areas — labels.md is the
  # single source of truth for the three-tier label set.
  local awk_types='{ if ($0 ~ /^## /) sec=$2; else if ($0 ~ /^\| /) { lbl=$2; if (sec=="Types" && lbl ~ /^[a-z][a-z-]*$/) print lbl } }'
  local awk_areas='{ if ($0 ~ /^## /) sec=$2; else if ($0 ~ /^\| /) { lbl=$2; if (sec=="Areas" && lbl ~ /^[a-z][a-z-]*$/) print lbl } }'
  TYPE_LABELS="$(awk "$awk_types" "$LABELS_MD")"
  AREA_LABELS="$(awk "$awk_areas" "$LABELS_MD")"
  [ -n "$TYPE_LABELS" ] && [ -n "$AREA_LABELS" ] \
    || die "could not parse labels from $LABELS_MD — is the file in shape?"
}

label_known() {
  # $1 = label, $2 = set (whitespace-separated, one label per line)
  printf '%s\n' $2 | grep -qxF "$1"
}

# Warn on untracked files (docs/specs/ is untracked by convention); die on
# anything else, including make check's autofixes when called post-check.
check_worktree() {
  local mode="$1" status untracked

  status="$(git status --porcelain)"

  if printf '%s\n' "$status" | grep -qE '^[MADRC]'; then
    die "staged changes present — commit them first, then re-run pr.sh"
  fi

  local dirty
  dirty="$(printf '%s\n' "$status" | grep -E '^.[MD]' | cut -c4- || true)"
  if [ -n "$dirty" ]; then
    if [ "$mode" = post-check ]; then
      die "make check autofixed files — commit them, then re-run pr.sh: $dirty"
    fi
    die "unstaged changes present — commit or stash them first: $dirty"
  fi

  untracked="$(printf '%s\n' "$status" | grep '^??' | cut -c4- || true)"
  if [ -n "$untracked" ]; then
    echo "pr: warning — untracked files (kept as-is):" >&2
    printf '%s\n' "$untracked" | sed 's/^/  /' >&2
  fi
}

preflight() {
  # local checks first — the most common mistakes get the most direct message
  local branch
  branch="$(git branch --show-current)"
  [ -n "$branch" ] || die "detached HEAD — create a branch first (git-workflow skill workflow 1)"
  [ "$branch" != "main" ] \
    || die "on main — changes live on branches: git switch -c <type>/<scope>/<slug> (git-workflow skill workflow 1)"

  check_worktree pre

  command -v gh >/dev/null 2>&1 || die "gh not found — install it (setup.sh checks it too)"
  # gh api user, not gh auth status — the latter exits 1 when any stored
  # keyring account is invalid, even with a working active account
  # (setup.sh § 1/4 checks API access the same way)
  gh api user >/dev/null 2>&1 || die "gh API access failing — run gh auth login"
}

sync_main() {
  if ! git switch main; then
    die "switching to main failed — resolve or stash the differing files first"
  fi
  git pull --ff-only || die "main is not fast-forwardable — resolve manually"
  git switch - || die "could not switch back to the branch"
}

parse_branch() {
  local branch
  branch="$(git branch --show-current)"

  case "$branch" in
    */*/*)
      BRANCH_TYPE="${branch%%/*}"
      local rest
      rest="${branch#*/}"
      BRANCH_SCOPE="${rest%%/*}"
      BRANCH_SLUG="${rest#*/}"
      ;;
    */*)
      BRANCH_TYPE="${branch%%/*}"
      BRANCH_SCOPE=""
      BRANCH_SLUG="${branch#*/}"
      ;;
    *)
      die "branch '$branch' does not match <type>/<scope>/<slug> — create one per git-workflow skill workflow 1"
      ;;
  esac

  case " $VALID_TYPES " in
    *" $BRANCH_TYPE "*) : ;;
    *) die "branch type '$BRANCH_TYPE' is not one of: $VALID_TYPES" ;;
  esac

  printf '%s' "$BRANCH_SLUG" | grep -qE '^[a-z0-9]+(-[a-z0-9]+)*$' \
    || die "branch slug '$BRANCH_SLUG' must be lowercase kebab-case (letters, digits, hyphens)"
  if [ -n "$BRANCH_SCOPE" ]; then
    printf '%s' "$BRANCH_SCOPE" | grep -qE '^[a-z0-9]+(-[a-z0-9]+)*$' \
      || die "branch scope '$BRANCH_SCOPE' must be lowercase kebab-case"
  fi
}

derive_labels() {
  # prefix → type label, mirroring the conventional prefixes against
  # labels.md § Types; keep in sync when either list changes
  local type_label
  case "$BRANCH_TYPE" in
    feat) type_label="feature" ;;
    fix)  type_label="bug" ;;
    perf) type_label="performance" ;;
    ci)   type_label="ci" ;;
    docs) type_label="documentation" ;;
    refactor) type_label="refactor" ;;
    test) type_label="test" ;;
    chore) type_label="chore" ;;
  esac

  label_known "$type_label" "$TYPE_LABELS" \
    || die "type label '$type_label' not in $LABELS_MD — pr.sh's prefix map is stale"

  LABELS="$type_label"

  if [ -n "$BRANCH_SCOPE" ]; then
    if label_known "$BRANCH_SCOPE" "$AREA_LABELS"; then
      LABELS="$LABELS $BRANCH_SCOPE"
    else
      echo "pr: note — scope '$BRANCH_SCOPE' is not an area label, assuming tooling ($LABELS_MD § Areas)" >&2
      LABELS="$LABELS tooling"
    fi
  fi

  for extra in $AREA_EXTRA; do
    label_known "$extra" "$AREA_LABELS" \
      || die "'$extra' is not an area label — valid areas: $(printf '%s' "$AREA_LABELS" | tr '\n' ' ')"
    LABELS="$LABELS $extra"
  done

  LABELS="$(printf '%s\n' $LABELS | sort -u | tr '\n' ' ' | sed 's/ $//')"

  echo "pr: labels: $LABELS"
}

run_checks() {
  if [ "$NO_CHECK" = 1 ]; then
    echo "pr: skipping make check (--no-check)"
    return
  fi
  make check
}

# codegraph_sync (lib-codegraph.sh) refreshes the CodeGraph index so the PR
# reflects the branch's final tree (post make-check) before the push.
# Best-effort and only when an index already exists — a missing or failing
# codegraph must never block the PR.

# Newest-first commit subjects on the branch relative to main; picks the
# first whose type matches the branch (type+scope match wins over
# type-only). Falls back to the branch slug.
derive_title() {
  if [ -n "$TITLE_OVERRIDE" ]; then
    TITLE="$TITLE_OVERRIDE"
    return
  fi

  local subjects chosen summary first rest
  subjects="$(git log --format='%s' main..HEAD || true)"
  chosen="$(printf '%s\n' "$subjects" | awk -v t="$BRANCH_TYPE" -v s="$BRANCH_SCOPE" '
    {
      head=$1; gsub(/^ +| +$/,"",head); n=split(head,a,"("); gsub(/\)/,"",a[2]);
      if (a[1]!=t) next;
      if (s=="" || a[2]==s) { match($0,/^[^:]*: /); print substr($0,RSTART+RLENGTH); exit }
      if (fallback=="") { match($0,/^[^:]*: /); fallback=substr($0,RSTART+RLENGTH) }
    }
    END { if (fallback!="") print fallback }')"

  if [ -n "$chosen" ]; then
    summary="$chosen"
  else
    summary="$(printf '%s' "$BRANCH_SLUG" | tr '-' ' ')"
  fi
  first="$(printf '%s' "$summary" | cut -c1 | tr '[:lower:]' '[:upper:]')"
  rest="$(printf '%s' "$summary" | cut -c2-)"
  summary="$first$rest"

  if [ -n "$BRANCH_SCOPE" ]; then
    TITLE="$BRANCH_TYPE($BRANCH_SCOPE): $summary"
  else
    TITLE="$BRANCH_TYPE: $summary"
  fi
  echo "pr: title: $TITLE"
}

push_and_open() {
  # make check runs fmt → golangci-lint --fix, which rewrites files —
  # pushing uncommitted autofixes would hand CI a different tree
  check_worktree post-check

  git push -u origin HEAD || die "push failed — check remote state"

  local url
  if gh pr view --json number >/dev/null 2>&1; then
    url="$(gh pr view --json url -q .url)"
    echo "pr: PR already exists for this branch: $url"
  else
    # gh's --template only works interactively — pre-fill the template into
    # a temp body file (editor on it when stdin is a TTY) and pass
    # --body-file, which works in both modes. The body itself is left for
    # the agent/human to write per the template (git-workflow skill § 3).
    local body_file
    body_file="$(mktemp -t pr-body.XXXXXX)"
    trap "rm -f '$body_file'" EXIT
    cp .github/pull_request_template.md "$body_file"
    if [ -t 0 ]; then
      "${EDITOR:-vi}" "$body_file" || die "editor failed"
    fi
    local -a args
    args=(--title "$TITLE" --body-file "$body_file")
    for label in $LABELS; do
      args+=(--label "$label")
    done
    gh pr create "${args[@]}"
    url="$(gh pr view --json url -q .url)"
    echo "pr: created $url"
  fi

  echo "pr: next — request a review; ci-pr and final-gate must pass; a session never merges — the human merges (git-workflow skill workflow 6)"
}

main() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --no-check) NO_CHECK=1 ;;
      --title) [ -n "${2:-}" ] || die "--title needs a value"; TITLE_OVERRIDE="$2"; shift ;;
      --area) [ -n "${2:-}" ] || die "--area needs a value"; AREA_EXTRA="$AREA_EXTRA $2"; shift ;;
      -h|--help)
        cat <<'EOF'
pr.sh — one command for the whole Thoth contribute flow.

Usage: ./scripts/pr.sh [--no-check] [--title <title>] [--area <label>]…

  --no-check        skip `make check` locally (CI still runs the gates)
  --title <title>   override the PR title (default: derived from branch + commits)
  --area <label>    add an area label (repeatable; valid areas: references/labels.md)
  -h, --help        show this help and exit

Runs, from an existing branch: preflight → sync with main → branch-name check →
label derivation → make check → push → gh pr create. A session never merges —
the human merges (git-workflow skill workflow 6).
EOF
        exit 0
        ;;
      *) die "unknown argument '$1' — usage: pr.sh [--no-check] [--title <title>] [--area <label>]…" ;;
    esac
    shift
  done

  echo "== 1/5 Preflight ==";   preflight
  echo "== 2/5 Sync ==";        sync_main
  echo "== 3/5 Branch ==";      parse_branch; derive_labels
  echo "== 4/5 make check ==";  run_checks
  derive_title
  codegraph_sync
  echo "== 5/5 Push + PR ==";   push_and_open
}

load_label_sets
main "$@"
