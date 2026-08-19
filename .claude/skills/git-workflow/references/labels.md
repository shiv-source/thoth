# Labels — the three-tier GitHub label set

Usage rules:
- **Types** — exactly one per issue AND PR: the closest fit to the change
- **Areas** — one per area touched: add every area the change touches
- **Priority** — issues only, exactly one: PRs carry no priority

## Types (mirror the conventional-commit prefixes)
| Label | Meaning |
|---|---|
| bug | defect or incorrect behavior |
| feature | new capability |
| enhancement | improvement to an existing capability |
| documentation | docs-only change |
| chore | maintenance, no user-visible behavior |
| refactor | restructure without behavior change |
| test | test-only change |
| performance | speed or resource work |
| ci | CI, workflows, tooling |

## Areas (package-aligned)
| Label | Touches |
|---|---|
| api | internal/api — REST + WS |
| chat | chat UX, WS protocol |
| cli | internal/cli — thoth commands |
| github | internal/github — identity + sync |
| index | internal/index — FTS5, watcher |
| search | search UI/behavior |
| settings | internal/settings + Settings UI |
| store | internal/store — schema, migrations |
| sync | git sync behavior |
| ui | web/src components/hooks |
| webui | internal/webui embed |
| wiki | internal/wiki — file contract |
| tooling | .claude/ skills/settings, .github/, .husky/, scripts/, lint/config files |

## Priority (issues only)
| Label |
|---|
| p-critical |
| p-high |
| p-medium |
| p-low |

## Kept GitHub defaults (outside the three-tier model)
Stock labels intentionally kept on the repo but not part of the
types/areas/priority workflow: duplicate, good first issue, help wanted,
invalid, question, wontfix.

Canonical: CLAUDE.md § Repo rules (usage rule) · live GitHub repo labels

The issue-label automation (`.github/actions/issue-labels/config.json`) applies
this same set to issue-form answers — keep it in sync whenever the label set
changes.

Stale if: the label set on GitHub changes (added, renamed, removed), or
CLAUDE.md's label usage rule changes.
