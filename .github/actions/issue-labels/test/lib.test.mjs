import test from "node:test"
import assert from "node:assert/strict"
import { computeLabels, missingLabels, normalizeLabel, parseFields, sectionValues } from "../src/lib.mjs"

const config = {
  types: ["bug", "feature", "enhancement", "documentation", "chore", "refactor", "test", "performance", "ci"],
  priorities: ["p-critical", "p-high", "p-medium", "p-low"],
  areas: ["api", "chat", "cli", "github", "index", "search", "settings", "store", "sync", "ui", "webui", "wiki", "tooling"],
  fields: { "Change type": "type", Priority: "priority", "Area(s)": "areas" },
  bugSignals: ["Summary", "Steps to reproduce"],
  bugLabel: "bug",
}

const bugBody = `### Environment

v1.2.3, darwin, make dev

### Summary

The X panel shows Y instead of Z when the index rebuilds

### Steps to reproduce

1. Run \`thoth serve --dev\`
2. Search for "logo"
3. Click the result

\`\`\`bash
# not a field heading, but could look like one:
### this is inside a code fence
\`\`\`

### Expected behavior

It should render inline.

### Actual behavior

It stays a link.

### Evidence

internal/index/sync.go:66

### Priority

- p-high

### Area(s)

- [x] api — internal/api (REST + WS)
- [x] index — internal/index (FTS5, watcher)

### Label check

- [x] Applied the type (bug is pre-applied), priority, and area label(s) per the repo rulebook.

### Notes / context

none
`

const featureBody = `### Change type

- enhancement — improvement to an existing capability

### Problem / motivation

We cannot filter the dashboard by date.

### Use cases

As a user, I want to see last week's notes.

### Proposed behavior

A date range filter on the dashboard.

### Acceptance criteria

- [ ] Given a date range, only notes in range render
- [ ] Tests added in web/src/components/DashboardView.test.tsx

### Priority

- p-medium

### Area(s)

- [x] webui — internal/webui embed
- [x] search — search UI/behavior

### Label check

- [x] Applied the type, priority, and area label(s).
`

test("bug template body maps to bug type, priority, and areas", () => {
  const labels = computeLabels(bugBody, config)
  assert.deepEqual(labels.sort(), ["api", "bug", "index", "p-high"])
})

test("feature template body maps to its change type, priority, and areas", () => {
  const labels = computeLabels(featureBody, config)
  assert.deepEqual(labels.sort(), ["enhancement", "p-medium", "search", "webui"])
})

test("blank (non-template) body yields no labels", () => {
  assert.deepEqual(computeLabels("", config), [])
  assert.deepEqual(computeLabels("just some markdown, no form sections\n- [x] foo\n", config), [])
})

test("a ### line inside a code fence is not treated as a field heading", () => {
  const fields = parseFields(bugBody)
  assert.deepEqual(Object.keys(fields).sort(), [
    "Actual behavior",
    "Area(s)",
    "Environment",
    "Evidence",
    "Expected behavior",
    "Label check",
    "Notes / context",
    "Priority",
    "Steps to reproduce",
    "Summary",
  ])
  const steps = fields["Steps to reproduce"]
  assert.ok(steps.includes("### this is inside a code fence"), "fenced content must land in the section, not split it")
})

test("single-select answers come through as list items", () => {
  assert.deepEqual(sectionValues(["- p-critical", "- [x] api", "plain text is ignored"]), ["p-critical", "api"])
})

test("normalization strips descriptions, case, and whitespace", () => {
  assert.equal(normalizeLabel("  API — internal/api (REST + WS) ", config.areas), "api")
  assert.equal(normalizeLabel("P-High", config.priorities), "p-high")
  assert.equal(normalizeLabel("feature - new capability", config.types), "feature")
  assert.equal(normalizeLabel("p-high", config.priorities), "p-high")
})

test("values outside the whitelist are dropped, never applied", () => {
  const body = bugBody.replace("- p-high", "- p-emergency").replace("- [x] api — internal/api (REST + WS)", "- [x] core — not a real area")
  assert.deepEqual(computeLabels(body, config).sort(), ["bug", "index"])
})

test("missingLabels is case-insensitive and never re-lists present or user labels", () => {
  assert.deepEqual(missingLabels(["bug", "p-high", "api"], ["Bug", "ui", "p-high"]), ["api"])
  assert.deepEqual(missingLabels(["bug", "p-high"], ["bug", "p-high", "help wanted"]), [])
})

test("an existing type label suppresses the bug fallback", () => {
  const body = featureBody
  assert.ok(!computeLabels(body, config).includes("bug"))
})

test("type stays exactly one label even if a body lists several", () => {
  const body = featureBody.replace(
    "- enhancement — improvement to an existing capability",
    "- feature — new capability\n- enhancement — improvement to an existing capability",
  )
  assert.deepEqual(computeLabels(body, config).sort(), ["feature", "p-medium", "search", "webui"])
})

test("a malformed config (unknown kind, non-array set) is skipped, not crashed on", () => {
  const broken = { ...config, fields: { ...config.fields, "Change type": "bogus" } }
  assert.deepEqual(computeLabels(featureBody, broken).sort(), ["p-medium", "search", "webui"])
})
