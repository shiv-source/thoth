# Thoth Project Skills Suite — Design

- Date: 2026-08-17
- Status: Approved (brainstorm 2026-08-17)
- Scope: `.claude/skills/` — two skills, `go` and `react`

## Purpose

Future Claude Code sessions on this repo should be faster and less likely to
break invariants. Each skill delivers two layers:

- **Procedure** — step-by-step workflows for the changes that are easy to get
  wrong (cross-layer sync, migrations, blast walls).
- **Expertise** — distilled domain indexes (components, hooks, packages) that
  load on demand instead of re-reading docs.

Success criteria:

1. A session touching the backend or frontend finds its workflow without
   re-deriving conventions from CLAUDE.md or `docs/`.
2. No rule or invariant is duplicated from CLAUDE.md — skills reference,
   never copy.
3. Staleness is detectable: each reference file states what makes it stale;
   every workflow points at canonical docs and `file:line`.

## Approach: C — hybrid

- **A (self-contained packs)** — rejected: duplicates `docs/` content,
  creates a second source of truth, rots when code changes.
- **B (thin pointers)** — rejected: adds little over the CLAUDE.md layout
  tree; drops the expertise half of the purpose.
- **C (hybrid)** — chosen: SKILL.md = workflows + gotchas; references =
  distilled indexes with canonical pointers.

Division of authority: **CLAUDE.md owns rules, `docs/` owns detail, skills
own procedure + pointers.**

## Decisions log

| Decision | Choice |
|---|---|
| Scope | Core pair: `go` + `react` (not pilot-only, not full suite) |
| Purpose | Procedure + expertise (not expertise-only, not enforcement-only) |
| Approach | C, hybrid (above) |
| Workflow depth | go: 8 workflows; react: 6; routine ones 3–5 lines, invariant-adjacent ones 8–10 |
| Design doc location | `docs/superpowers/specs/` (committed) — not the untracked `docs/specs/` convention, since this governs the committed skills dir |
| Skill placement | `.claude/skills/<name>/SKILL.md` + `references/`, committed |
| CLAUDE.md change | One pointer line under the graphify section; nothing else |
| 2026-08-18 follow-on | Added third skill `git-workflow` (contribution workflow) + CLAUDE.md Repo-rules trim to invariants; supersedes "one pointer line, nothing else" and the "no workflow skill" out-of-scope line for this skill |

## Inventory

```
.claude/skills/
├── go/                          # backend: internal/* + cmd/*
│   ├── SKILL.md                 # when to use, key files, workflows, gotchas
│   └── references/
│       ├── packages.md          # per-package purpose + API surface + canonical docs pointer
│       ├── claude-blast-wall.md # claude CLI flags, persistent pool, process-group kill
│       ├── persistence.md       # store migrations, settings KV, index/FTS contract
│       └── quality.md           # test patterns, coverage gate, race, the check pipeline
└── react/                       # frontend: web/src/
    ├── SKILL.md
    └── references/
        ├── components.md        # component inventory: purpose, props, one-line API
        ├── hooks.md             # hook signatures + what they own
        ├── store.md             # Redux slice conventions, typed hooks, co-located tests
        └── patterns.md          # zod boundary, WS client, test doubles, design system
```

`go` gets four reference files because `internal/` spans ten packages —
scoped by concern, not by package. `react` mirrors the user's proposed
structure plus `store.md` (Redux slices are a large enough domain to
deserve their own index).

## SKILL.md anatomy

```markdown
---
name: go                        # or react
description: >-                  # one line, trigger terms: package/domain names,
  # tasks it fires on            #   so auto-invocation matches real requests
---

# Go backend (or React frontend) — procedures & expertise

## When to use
3-5 bullets: which tasks pull this skill in — and which do NOT
(e.g. note-taking behavior → wiki rulebook, not a repo skill).

## Key files
A 10-15 line scoped map of just this layer (internal/… or web/src/…),
mirroring the CLAUDE.md layout tree without repeating it.

## Workflows
One subsection per error-prone change, numbered steps, each step a verb
+ target file. Cross-layer sync steps are called out in caps
(e.g. "UPDATE BOTH: internal/api/chat.go AND web/src/ws/chat.ts").

## Gotchas
Short bullets of things that will bite, each a pointer to the rule in
CLAUDE.md — not a copy of the rule.

## Canonical docs
Pointer block: authoritative detail lives in docs/; this skill is the
distilled path.

## Maintenance
Derived view — after a behavior change, update this skill + docs/ in
the same commit; go skill also: verify flags against `claude --help`.
```

### Workflow lists

go (8):
1. Add a REST endpoint
2. Extend the WS protocol — server AND `web/src/ws/chat.ts` (shared types)
3. Add a store migration
4. Change claude CLI flags — blast wall: `client.go` only
5. Add a settings key
6. Extend the wiki contract
7. Bump a dependency (CI verifies)
8. Add a doctor install check

react (6):
1. Add a component
2. Add a Redux slice
3. Add a hook
4. Wire an API call (zod boundary)
5. Test a component/slice
6. Touch the WS client

## Reference file anatomy

```markdown
# Components (web/src/components)        ← scope header: which dir this indexes

## ChatPanel
- path: web/src/components/ChatPanel.tsx
- purpose: chat transcript + composer, owns useChat
- props/api: `channels`, `onSend` — see web/src/ws/chat.ts
- canonical: docs/frontend.md § Chat · source file:line

Coverage rule: every component/hook/slice/package gets an entry, or a
deliberate "intentionally skipped" line for leaf utilities — so a
missing entry reads as stale, not as never-indexed.
File ends with "Stale if: …" — the 2-3 signals that mean this index
needs a refresh (new file in dir, signature change, new slice).
```

Three rules keep references honest:
1. **Entry = path + purpose + API + canonical pointer** — never a
   re-telling of the docs.
2. **Complete coverage or explicit skip.**
3. **A "stale if" line** — staleness is detectable, like the blast wall's
   "verify against `claude --help`".

## Wiring

1. Skills committed at `.claude/skills/`; `description:` frontmatter
   carries trigger terms so future sessions auto-invoke them via the
   `Skill` tool, and users can call `/go` / `/react` explicitly.
2. One line added to `CLAUDE.md` under the graphify section, exactly:

   > - **Skills** — `.claude/skills/` holds the go (backend) and react
   >   (frontend) procedure skills. Rules stay in this file; procedures
   >   live there; `docs/` owns detail.
3. Same-commit maintenance contract: behavior change → update skill +
   `docs/` + `graphify update .`.

## Maintenance

- Staleness is treated as a bug, not tolerated: each reference file's
  "stale if" line makes drift findable; the same-commit contract keeps
  drift rare.
- No automation is added — YAGNI; the contract plus detectable staleness
  is enough for two skills.

## Out of scope

- No testing/CI skill, no workflow skill, no wiki skill — CLAUDE.md and
  `docs/` already cover them; revisit only if a real gap shows.
- No user-global skills; project-local only.
- No changes to app code or existing docs in this work item.
