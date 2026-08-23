import test from "node:test"
import assert from "node:assert/strict"
import { writeFileSync, mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import {
  MARKER,
  commitUrl,
  coverageText,
  displayName,
  eventPullRequestNumber,
  fetchRunJobs,
  findMarkerComment,
  githubContext,
  renderPrBody,
  renderStepSummary,
  reportJobs,
  runUrl,
  upsertReportComment,
} from "../src/lib.mjs"

const ctx = {
  repository: "owner/repo",
  apiUrl: "https://api.github.com",
  serverUrl: "https://github.com",
  runId: "123",
  runNumber: "42",
  sha: "0123456789abcdef",
  job: "report",
  eventName: "pull_request",
  eventPath: "/tmp/event.json",
}

const jobs = [
  { name: "report", conclusion: "success" },
  { name: "final-gate", conclusion: "success" },
  { name: "quality / backend-test", conclusion: "failure" },
  { name: "quality / backend-lint", conclusion: "skipped" },
  { name: "frontend-test", conclusion: "success" },
  { name: "docs-build", conclusion: "cancelled" },
]

test("displayName keeps the last segment of a parent / child job name", () => {
  assert.equal(displayName("quality / backend-test"), "backend-test")
  assert.equal(displayName("frontend-test"), "frontend-test")
  assert.equal(displayName(undefined), "")
})

test("reportJobs excludes self and wrapper, counts and sorts the rest", () => {
  const report = reportJobs(jobs, { self: "report", wrapper: "final-gate" })
  assert.deepEqual(report.rows.map((row) => row.name), ["backend-test", "backend-lint", "docs-build", "frontend-test"])
  assert.deepEqual(report.rows.map((row) => row.text), [
    "❌ failure",
    "⏭️ skipped",
    "🚫 cancelled",
    "✅ success",
  ])
  assert.equal(report.failed, 2)
  assert.equal(report.passed, 2)
  assert.equal(report.total, 4)
  assert.equal(report.result, "failed")
})

test("reportJobs passes when nothing failed or was cancelled", () => {
  const ok = [
    { name: "quality / backend-test", conclusion: "success" },
    { name: "quality / backend-lint", conclusion: "skipped" },
  ]
  const report = reportJobs(ok, { self: "report", wrapper: "final-gate" })
  assert.equal(report.failed, 0)
  assert.equal(report.passed, 2)
  assert.equal(report.result, "passed")
})

test("an in-progress job (null conclusion) counts as failed and renders in progress", () => {
  const running = [{ name: "quality / backend-test", conclusion: null }]
  const report = reportJobs(running, { self: "report", wrapper: "final-gate" })
  assert.equal(report.failed, 1)
  assert.equal(report.result, "failed")
  assert.equal(report.rows[0].text, "⏳ in progress")
})

test("renderStepSummary matches the workflow's step-summary format", () => {
  const report = reportJobs(jobs, { self: "report", wrapper: "final-gate" })
  const summary = renderStepSummary({ ...report, ...ctx })
  assert.equal(
    summary,
    [
      "## CI failed ❌ — 2 job(s) failed",
      "",
      "Run: [owner/repo#42](https://github.com/owner/repo/actions/runs/123) · commit [`01234567`](https://github.com/owner/repo/commit/0123456789abcdef)",
      "",
      "| Job | Result |",
      "|---|---|",
      "| backend-test | ❌ failure |",
      "| backend-lint | ⏭️ skipped |",
      "| docs-build | 🚫 cancelled |",
      "| frontend-test | ✅ success |",
    ].join("\n"),
  )
})

test("renderStepSummary renders the passed title when nothing failed", () => {
  const ok = [
    { name: "quality / backend-test", conclusion: "success" },
    { name: "quality / backend-lint", conclusion: "skipped" },
  ]
  const report = reportJobs(ok, { self: "report", wrapper: "final-gate" })
  assert.ok(renderStepSummary({ ...report, ...ctx }).startsWith("## CI passed ✅"))
})

test("renderPrBody carries the marker, tally, table, and footer", () => {
  const report = reportJobs(jobs, { self: "report", wrapper: "final-gate" })
  const body = renderPrBody({ ...report, ...ctx })
  const lines = body.split("\n")
  assert.ok(body.startsWith(`${MARKER}\n\n### CI Report ❌`))
  assert.ok(lines.includes("**2/4 jobs passed** · [Run #42](https://github.com/owner/repo/actions/runs/123) · commit [`01234567`](https://github.com/owner/repo/commit/0123456789abcdef)"))
  assert.ok(lines.includes("| backend-test | ❌ failure |"))
  assert.ok(lines.includes("</details>"))
  assert.ok(lines.includes("🤖 Updated automatically on every run by [final-gate](https://github.com/owner/repo/actions/workflows/final-gate.yml)"))
  assert.ok(lines.includes("⛔ This check must pass before merging"))
})

test("coverageText shows actual and floor with a pass/fail marker", () => {
  assert.equal(coverageText("91.2", "90"), "📊 Backend coverage: **91.2%** — floor **90%** ✅")
  assert.equal(coverageText("87.3", "90"), "📊 Backend coverage: **87.3%** — floor **90%** ❌")
  assert.equal(coverageText("90", "90"), "📊 Backend coverage: **90%** — floor **90%** ✅")
  assert.equal(coverageText("91.2%", "90%"), "📊 Backend coverage: **91.2%** — floor **90%** ✅", "trailing % on inputs is tolerated")
  assert.equal(coverageText("91.2"), "📊 Backend coverage: **91.2%**", "without a floor the marker is omitted")
  assert.equal(coverageText("", "90"), null)
  assert.equal(coverageText(undefined, "90"), null)
  assert.equal(coverageText("  ", "90"), null)
})

test("coverageText labels the frontend line and renders it with its own floor", () => {
  const frontend = (c, f) => coverageText(c, f, { emoji: "🖥️", label: "Frontend coverage" })
  assert.equal(frontend("93.2", "90"), "🖥️ Frontend coverage: **93.2%** — floor **90%** ✅")
  assert.equal(frontend("88.1", "90"), "🖥️ Frontend coverage: **88.1%** — floor **90%** ❌")
  assert.equal(frontend("", "90"), null)
})

test("renderStepSummary inserts the coverage lines between the run and the table", () => {
  const report = reportJobs(jobs, { self: "report", wrapper: "final-gate" })
  const summary = renderStepSummary({ ...report, ...ctx, coverage: "91.2", coverageFloor: "90" })
  assert.ok(summary.includes("\n\n📊 Backend coverage: **91.2%** — floor **90%** ✅\n\n| Job | Result |"))
})

test("renderStepSummary renders backend and frontend coverage lines together", () => {
  const report = reportJobs(jobs, { self: "report", wrapper: "final-gate" })
  const summary = renderStepSummary({ ...report, ...ctx, coverage: "91.2", coverageFloor: "90", webCoverage: "93.2", webCoverageFloor: "90" })
  assert.ok(
    summary.includes("\n\n📊 Backend coverage: **91.2%** — floor **90%** ✅\n🖥️ Frontend coverage: **93.2%** — floor **90%** ✅\n\n| Job | Result |"),
  )
})

test("renderPrBody inserts the coverage line after the tally", () => {
  const report = reportJobs(jobs, { self: "report", wrapper: "final-gate" })
  const body = renderPrBody({ ...report, ...ctx, coverage: "91.2", coverageFloor: "90" })
  assert.ok(body.includes("\n\n📊 Backend coverage: **91.2%** — floor **90%** ✅\n\n<details>"))
})

test("renderPrBody renders backend and frontend coverage lines together", () => {
  const report = reportJobs(jobs, { self: "report", wrapper: "final-gate" })
  const body = renderPrBody({ ...report, ...ctx, coverage: "91.2", coverageFloor: "90", webCoverage: "93.2", webCoverageFloor: "90" })
  assert.ok(
    body.includes("\n\n📊 Backend coverage: **91.2%** — floor **90%** ✅\n🖥️ Frontend coverage: **93.2%** — floor **90%** ✅\n\n<details>"),
  )
})

test("githubContext maps runner env into the github context", () => {
  const env = {
    GITHUB_REPOSITORY: "owner/repo",
    GITHUB_RUN_ID: "9",
    GITHUB_RUN_NUMBER: "7",
    GITHUB_SHA: "abc",
    GITHUB_JOB: "report",
    GITHUB_EVENT_NAME: "push",
    GITHUB_EVENT_PATH: "/tmp/e.json",
  }
  assert.deepEqual(githubContext(env), {
    repository: "owner/repo",
    apiUrl: "https://api.github.com",
    serverUrl: "https://github.com",
    runId: "9",
    runNumber: "7",
    sha: "abc",
    job: "report",
    eventName: "push",
    eventPath: "/tmp/e.json",
  })
})

test("runUrl and commitUrl build GitHub links", () => {
  assert.equal(runUrl(ctx, "123"), "https://github.com/owner/repo/actions/runs/123")
  assert.equal(commitUrl(ctx, "0123456789"), "https://github.com/owner/repo/commit/0123456789")
})

function scriptedFetch(...responses) {
  const queue = [...responses]
  const calls = []
  global.fetch = async (url, init) => {
    calls.push({ url, init })
    const next = queue.shift()
    if (next instanceof Error) throw next
    return next
  }
  return { calls, restore: () => (global.fetch = undefined) }
}

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } })
}

test("fetchRunJobs follows pagination until total_count is reached", async () => {
  const { calls, restore } = scriptedFetch(
    jsonResponse({ total_count: 2, jobs: [{ name: "a", conclusion: "success" }] }),
    jsonResponse({ total_count: 2, jobs: [{ name: "b", conclusion: "skipped" }] }),
  )
  try {
    const fetched = await fetchRunJobs({ repository: "owner/repo", apiUrl: "https://api.github.com", runId: "1", token: "t" })
    assert.deepEqual(fetched.map((job) => job.name), ["a", "b"])
    assert.equal(calls.length, 2)
    assert.ok(calls[0].url.includes("page=1"))
    assert.ok(calls[1].url.includes("page=2"))
  } finally {
    restore()
  }
})

test("findMarkerComment returns the marker-tagged comment even when it is past the first page", async () => {
  const comment = { id: 99, body: `body ${MARKER}` }
  const fullPage = Array.from({ length: 100 }, (_, i) => ({ id: i, body: `plain ${i}` }))
  const { restore } = scriptedFetch(jsonResponse(fullPage), jsonResponse([comment]))
  try {
    const found = await findMarkerComment({ repository: "owner/repo", apiUrl: "https://api.github.com", pr: 7, token: "t" })
    assert.equal(found.id, 99)
  } finally {
    restore()
  }
})

test("upsertReportComment POSTs a new comment when none is tagged, PATCHes otherwise", async () => {
  const { calls, restore } = scriptedFetch(jsonResponse([]), jsonResponse({}), jsonResponse([{ id: 5, body: MARKER }]), jsonResponse({}))
  try {
    assert.equal(
      await upsertReportComment({ repository: "owner/repo", apiUrl: "https://api.github.com", pr: 7, token: "t", body: "hello" }),
      "created",
    )
    const post = calls[1]
    assert.equal(post.init.method, "POST")
    assert.ok(post.url.endsWith("/repos/owner/repo/issues/7/comments"))
    assert.equal(
      await upsertReportComment({ repository: "owner/repo", apiUrl: "https://api.github.com", pr: 7, token: "t", body: "hello" }),
      "updated",
    )
    const patch = calls[3]
    assert.equal(patch.init.method, "PATCH")
    assert.ok(patch.url.endsWith("/repos/owner/repo/issues/comments/5"))
  } finally {
    restore()
  }
})

test("eventPullRequestNumber reads the PR number from the triggering event", () => {
  const dir = mkdtempSync(join(tmpdir(), "ci-report-lib-"))
  try {
    const eventFile = join(dir, "event.json")
    writeFileSync(eventFile, JSON.stringify({ pull_request: { number: 43 } }))
    assert.equal(eventPullRequestNumber(eventFile), 43)
    assert.equal(eventPullRequestNumber(join(dir, "missing.json")), null)
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
})
