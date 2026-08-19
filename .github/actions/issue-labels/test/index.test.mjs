import test from "node:test"
import assert from "node:assert/strict"
import { mkdtempSync, readFileSync, writeFileSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { main } from "../src/index.mjs"

const repo = "owner/repo"
const token = "test-token"

function withEnv(event, { stubbedFetch } = {}) {
  const dir = mkdtempSync(join(tmpdir(), "issue-labels-"))
  const eventFile = join(dir, "event.json")
  const outputFile = join(dir, "out.txt")
  const summaryFile = join(dir, "summary.md")
  writeFileSync(eventFile, JSON.stringify(event))

  const realFetch = global.fetch
  const calls = []
  if (stubbedFetch) global.fetch = async (url, init) => (calls.push({ url, init }), new Response("[]", { status: 200 }))

  process.env.GITHUB_EVENT_PATH = eventFile
  process.env.GITHUB_REPOSITORY = repo
  process.env.GH_TOKEN = token
  process.env.GITHUB_OUTPUT = outputFile
  process.env.GITHUB_STEP_SUMMARY = summaryFile

  const read = () => {
    const maybe = (file) => {
      try {
        return readFileSync(file, "utf8")
      } catch {
        return ""
      }
    }
    return {
      calls,
      applied: maybe(outputFile).split("\n").filter((line) => line.startsWith("applied=")).at(-1)?.slice("applied=".length),
      summary: maybe(summaryFile),
    }
  }

  return {
    read,
    cleanup: () => {
      global.fetch = realFetch
      delete process.env.GITHUB_EVENT_PATH
      delete process.env.GITHUB_REPOSITORY
      delete process.env.GH_TOKEN
      delete process.env.GITHUB_OUTPUT
      delete process.env.GITHUB_STEP_SUMMARY
      rmSync(dir, { recursive: true, force: true })
    },
  }
}

function bugEvent(labels) {
  return {
    action: "opened",
    issue: {
      number: 43,
      labels: labels ?? [{ name: "bug" }],
      body: "### Summary\n\nx\n\n### Steps to reproduce\n\n1. a\n\n### Priority\n\n- p-high\n\n### Area(s)\n\n- [x] api — internal/api (REST + WS)\n- [x] index — internal/index (FTS5, watcher)\n",
    },
  }
}

test("blank (non-template) issue: no fetch, applied=[]", async () => {
  const env = withEnv({ action: "opened", issue: { number: 1, labels: [], body: "just a note" } })
  try {
    await main()
    const result = env.read()
    assert.deepEqual(result.calls, [])
    assert.equal(result.applied, "[]")
  } finally {
    env.cleanup()
  }
})

test("issue already carrying the bare minimum: no fetch, nothing re-applied", async () => {
  const env = withEnv(bugEvent([{ name: "bug" }, { name: "p-high" }, { name: "api" }, { name: "index" }]))
  try {
    await main()
    const result = env.read()
    assert.deepEqual(result.calls, [])
    assert.equal(result.applied, "[]")
  } finally {
    env.cleanup()
  }
})

test("unhandled action: skipped, output defaults to []", async () => {
  const event = bugEvent()
  event.action = "closed"
  const env = withEnv(event)
  try {
    await main()
    const result = env.read()
    assert.deepEqual(result.calls, [])
    assert.equal(result.applied, "[]")
  } finally {
    env.cleanup()
  }
})

test("POSTs exactly the missing labels; never lists ones already applied or user-added", async () => {
  const env = withEnv(bugEvent([{ name: "bug" }, { name: "help wanted" }]), { stubbedFetch: true })
  try {
    await main()
    const result = env.read()
    assert.equal(result.calls.length, 1)
    const call = result.calls[0]
    assert.equal(call.url, `https://api.github.com/repos/${repo}/issues/43/labels`)
    assert.equal(call.init.method, "POST")
    assert.deepEqual(JSON.parse(call.init.body), { labels: ["p-high", "api", "index"] })
    assert.equal(result.applied, JSON.stringify(["p-high", "api", "index"]))
    assert.match(result.summary, /#43/)
  } finally {
    env.cleanup()
  }
})
