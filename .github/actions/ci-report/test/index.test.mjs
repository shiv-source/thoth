import test from "node:test"
import assert from "node:assert/strict"
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { main } from "../src/index.mjs"
import { MARKER } from "../src/lib.mjs"

const JOB_NAMES = ["quality / backend-test", "quality / backend-lint", "frontend-test"]

function jobsWith(conclusions) {
  return conclusions.map((conclusion, i) => ({ name: JOB_NAMES[i], conclusion }))
}

const PASSING_JOBS = jobsWith(["success", "skipped", "success"])
const FAILING_JOBS = jobsWith(["failure", "skipped", "success"])

// Script a fetch that answers the jobs API and (optionally) the comments API,
// recording every call. `comments` is the array the list endpoint returns;
// empty means "no marker-tagged comment yet" (so a POST should follow).
function stubFetch({ conclusions, comments }) {
  const calls = []
  global.fetch = async (url, init) => {
    calls.push({ url, init })
    if (url.includes("/actions/runs/")) {
      return new Response(JSON.stringify({ total_count: conclusions.length, jobs: jobsWith(conclusions) }), {
        status: 200,
        headers: { "content-type": "application/json" },
      })
    }
    if ((init?.method ?? "GET") === "GET") {
      return new Response(JSON.stringify(comments ?? []), { status: 200, headers: { "content-type": "application/json" } })
    }
    return new Response(JSON.stringify({}), { status: 200, headers: { "content-type": "application/json" } })
  }
  return {
    calls,
    restore: () => (global.fetch = undefined),
  }
}

function withEnv(
  event,
  { wrapper = "final-gate", coverage = "", coverageFloor = "", webCoverage = "", webCoverageFloor = "", coverageCovered = "", coverageTotal = "", webCoverageCovered = "", webCoverageTotal = "" } = {},
) {
  const dir = mkdtempSync(join(tmpdir(), "ci-report-"))
  const eventFile = join(dir, "event.json")
  const outputFile = join(dir, "out.txt")
  const summaryFile = join(dir, "summary.md")
  writeFileSync(eventFile, JSON.stringify(event ?? {}))

  const prev = { ...process.env }
  Object.assign(process.env, {
    GITHUB_REPOSITORY: "owner/repo",
    GITHUB_RUN_ID: "123",
    GITHUB_RUN_NUMBER: "42",
    GITHUB_SHA: "0123456789abcdef",
    GITHUB_JOB: "report",
    GITHUB_EVENT_NAME: event?.pull_request ? "pull_request" : "push",
    GITHUB_EVENT_PATH: eventFile,
    GH_TOKEN: "test-token",
    INPUT_WRAPPER: wrapper,
    INPUT_COVERAGE: coverage,
    INPUT_COVERAGE_FLOOR: coverageFloor,
    INPUT_WEB_COVERAGE: webCoverage,
    INPUT_WEB_COVERAGE_FLOOR: webCoverageFloor,
    INPUT_COVERAGE_COVERED: coverageCovered,
    INPUT_COVERAGE_TOTAL: coverageTotal,
    INPUT_WEB_COVERAGE_COVERED: webCoverageCovered,
    INPUT_WEB_COVERAGE_TOTAL: webCoverageTotal,
    GITHUB_OUTPUT: outputFile,
    GITHUB_STEP_SUMMARY: summaryFile,
  })

  const readOutput = (name) => {
    const line = readFileSync(outputFile, "utf8")
      .split("\n")
      .filter((l) => l.startsWith(`${name}=`))
      .at(-1)
    return line ? line.slice(`${name}=`.length) : undefined
  }

  return {
    summary: () => readFileSync(summaryFile, "utf8"),
    output: readOutput,
    cleanup: () => {
      global.fetch = undefined
      process.env = prev
      process.exitCode = undefined
      rmSync(dir, { recursive: true, force: true })
    },
  }
}

test("coverage values flow into the step summary and the PR comment", async () => {
  const env = withEnv({ pull_request: { number: 7 } }, { coverage: "91.2", coverageFloor: "90" })
  const stub = stubFetch({ conclusions: ["success", "skipped", "success"], comments: [] })
  try {
    await main()
    assert.ok(env.summary().includes("📊 Backend coverage: **91.2%** — floor **90%** ✅"))
    const post = stub.calls.find((call) => call.init?.method === "POST")
    assert.ok(post, "a POST comment call must happen on a PR run")
    assert.ok(JSON.parse(post.init.body).body.includes("📊 Backend coverage: **91.2%** — floor **90%** ✅"))
  } finally {
    env.cleanup()
    stub.restore()
  }
})

test("backend and frontend coverage both appear when provided", async () => {
  const env = withEnv({ pull_request: { number: 7 } }, { coverage: "91.2", coverageFloor: "90", webCoverage: "93.2", webCoverageFloor: "90" })
  const stub = stubFetch({ conclusions: ["success", "skipped", "success"], comments: [] })
  try {
    await main()
    assert.ok(env.summary().includes("📊 Backend coverage: **91.2%** — floor **90%** ✅"))
    assert.ok(env.summary().includes("🖥️ Frontend coverage: **93.2%** — floor **90%** ✅"))
    assert.ok(env.summary().includes("📈 Average coverage: **92.2%** — floor **90%** ✅"))
    const post = stub.calls.find((call) => call.init?.method === "POST")
    assert.ok(JSON.parse(post.init.body).body.includes("🖥️ Frontend coverage: **93.2%** — floor **90%** ✅"))
  } finally {
    env.cleanup()
    stub.restore()
  }
})

test("overall coverage line appears when both areas carry statement counts", async () => {
  const env = withEnv(
    { pull_request: { number: 7 } },
    {
      coverage: "91.2", coverageFloor: "90", coverageCovered: "1330", coverageTotal: "1458",
      webCoverage: "93.2", webCoverageFloor: "90", webCoverageCovered: "1449", webCoverageTotal: "1554",
    },
  )
  const stub = stubFetch({ conclusions: ["success", "skipped", "success"], comments: [] })
  try {
    await main()
    assert.ok(env.summary().includes("🧮 Overall coverage: **92.3%** — floor **90%** ✅"))
    const post = stub.calls.find((call) => call.init?.method === "POST")
    assert.ok(JSON.parse(post.init.body).body.includes("🧮 Overall coverage: **92.3%** — floor **90%** ✅"))
  } finally {
    env.cleanup()
    stub.restore()
  }
})

test("passing run: writes summary, POSTs a PR comment, exits 0", async () => {
  const env = withEnv({ pull_request: { number: 7 } })
  const stub = stubFetch({ conclusions: ["success", "skipped", "success"], comments: [] })
  try {
    await main()
    const calls = stub.calls
    const post = calls.find((call) => call.init?.method === "POST")
    assert.ok(post, "a POST comment call must happen on a PR run")
    assert.ok(post.url.endsWith("/repos/owner/repo/issues/7/comments"))
    assert.deepEqual(JSON.parse(post.init.body).body.split("\n")[0], MARKER)
    assert.equal(env.output("result"), "passed")
    assert.equal(env.output("failed"), "0")
    assert.equal(env.output("passed"), "3")
    assert.equal(env.output("total"), "3")
    assert.ok(env.summary().startsWith("## CI passed ✅"))
    assert.equal(process.exitCode, undefined)
  } finally {
    env.cleanup()
    stub.restore()
  }
})

test("failing run: still renders summary and comment, exits 1", async () => {
  const env = withEnv({ pull_request: { number: 7 } })
  const stub = stubFetch({ conclusions: ["failure", "skipped", "success"], comments: [{ id: 5, body: `old ${MARKER}` }] })
  try {
    await main()
    const patch = stub.calls.find((call) => call.init?.method === "PATCH")
    assert.ok(patch, "an existing marker-tagged comment must be PATCHed in place")
    assert.ok(patch.url.endsWith("/repos/owner/repo/issues/comments/5"))
    assert.equal(env.output("result"), "failed")
    assert.equal(env.output("failed"), "1")
    assert.ok(env.summary().includes("## CI failed ❌ — 1 job(s) failed"))
    assert.equal(process.exitCode, 1)
  } finally {
    env.cleanup()
    stub.restore()
  }
})

test("push run (not a PR): no comment, just summary and gate", async () => {
  const env = withEnv({})
  const stub = stubFetch({ conclusions: ["success", "skipped", "success"], comments: [] })
  try {
    await main()
    assert.ok(!stub.calls.some((call) => call.url.includes("/issues/")), "no issues/comment API call outside a PR")
    assert.equal(env.output("result"), "passed")
    assert.ok(env.summary().includes("| frontend-test | ✅ success |"))
    assert.equal(process.exitCode, undefined)
  } finally {
    env.cleanup()
    stub.restore()
  }
})

test("self and wrapper jobs are excluded from the report and the gate", async () => {
  const env = withEnv({ pull_request: { number: 7 } })
  const full = [
    { name: "final-gate", conclusion: "success" },
    { name: "report", conclusion: "in_progress" },
    ...jobsWith(["success", "skipped", "success"]),
  ]
  const stub = stubFetch({ conclusions: [], comments: [] })
  stub.calls = []
  global.fetch = async (url) => {
    const json = url.includes("/actions/runs/") ? { total_count: full.length, jobs: full } : []
    return new Response(JSON.stringify(json), { status: 200, headers: { "content-type": "application/json" } })
  }
  try {
    await main()
    assert.equal(env.output("result"), "passed")
    assert.equal(env.output("total"), "3")
    assert.equal(process.exitCode, undefined)
  } finally {
    env.cleanup()
    stub.restore()
  }
})
