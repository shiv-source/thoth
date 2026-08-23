import { readFileSync } from "node:fs"

// Marker that tags the bot's PR report comment so each run updates it in
// place instead of stacking a new comment. Keep in sync with git-workflow
// skill "Maintenance" (§ the report marker).
export const MARKER = "<!-- thoth-ci-report -->"

const API_HEADERS = (token) => ({
  authorization: `Bearer ${token}`,
  accept: "application/vnd.github+json",
  "x-github-api-version": "2022-11-28",
})

// The runner env mirrors the ${{ github.* }} context the workflow YAML would
// interpolate; read it through one function so tests can inject any part.
export function githubContext(env = process.env) {
  return {
    repository: env.GITHUB_REPOSITORY ?? "",
    apiUrl: env.GITHUB_API_URL ?? "https://api.github.com",
    serverUrl: env.GITHUB_SERVER_URL ?? "https://github.com",
    runId: env.GITHUB_RUN_ID ?? "",
    runNumber: env.GITHUB_RUN_NUMBER ?? "",
    sha: env.GITHUB_SHA ?? "",
    job: env.GITHUB_JOB ?? "",
    eventName: env.GITHUB_EVENT_NAME ?? "",
    eventPath: env.GITHUB_EVENT_PATH,
  }
}

// A matrix job's API name is "parent / matrix values…" — the workflow's
// report keys jobs by the last segment (the part after the last " / ").
export function displayName(name) {
  return (name ?? "").split(" / ").at(-1)
}

const CONCLUSION_EMOJI = {
  success: "✅",
  failure: "❌",
  cancelled: "🚫",
  skipped: "⏭️",
}

function conclusionText(conclusion) {
  const emoji = CONCLUSION_EMOJI[conclusion]
  return emoji ? `${emoji} ${conclusion}` : `⏳ ${conclusion ?? "in progress"}`
}

// Fetch every job in the run, pages included. Caller supplies the GitHub
// context fields (owner/repo via repository, apiUrl) plus a token.
export async function fetchRunJobs({ repository, apiUrl, runId, token }) {
  const base = `${apiUrl}/repos/${repository}/actions/runs/${runId}/jobs`
  const jobs = []
  for (let page = 1; ; page++) {
    const response = await fetch(`${base}?per_page=100&page=${page}`, { headers: API_HEADERS(token) })
    if (!response.ok) {
      throw new Error(`fetch jobs [${response.status}]: ${await response.text()}`)
    }
    const data = await response.json()
    jobs.push(...(data.jobs ?? []))
    if (data.total_count == null || jobs.length >= data.total_count) break
  }
  return jobs
}

// Summarize the run's jobs into report rows, mirroring the workflow's rules:
//   - jobs whose display name matches the job running this action (self) or
//     the caller's wrapper job are excluded entirely;
//   - remaining jobs sort with non-successes first (so failures/cancellations
//     surface before the success tail);
//   - failed  = any conclusion that is neither success nor skipped;
//   - passed  = success or skipped (skipped gates count as passing);
//   - result  = passed unless something failed (includes in-progress runs,
//     whose jobs have a null conclusion).
export function reportJobs(jobs, { self, wrapper }) {
  const excluded = new Set([self, wrapper].filter(Boolean))
  const rows = jobs
    .filter((job) => !excluded.has(displayName(job.name)))
    .sort((a, b) => Number(a.conclusion === "success") - Number(b.conclusion === "success"))
    .map((job) => ({ name: displayName(job.name), conclusion: job.conclusion, text: conclusionText(job.conclusion) }))
  const failed = rows.filter((row) => row.conclusion !== "success" && row.conclusion !== "skipped").length
  const passed = rows.filter((row) => row.conclusion === "success" || row.conclusion === "skipped").length
  return { rows, failed, passed, total: rows.length, result: failed > 0 ? "failed" : "passed" }
}

export function runUrl({ serverUrl, repository }, runId) {
  return `${serverUrl}/${repository}/actions/runs/${runId}`
}

export function commitUrl({ serverUrl, repository }, sha) {
  return `${serverUrl}/${repository}/commit/${sha}`
}

function shortSha(sha) {
  return sha.slice(0, 8)
}

// Parse a coverage value: trim, tolerate a trailing %, → number (null when
// empty/absent).
function num(v) {
  const s = String(v ?? "").trim().replace(/%$/, "")
  return s === "" ? null : Number(s)
}

// One area's coverage: actual %, its floor, and whether it clears the floor.
// Returns null when the area didn't run (no value provided).
export function areaCoverage(coverage, coverageFloor) {
  const pct = num(coverage)
  if (pct === null) return null
  const floor = num(coverageFloor)
  return { pct, floor, ok: floor === null ? null : pct >= floor }
}

// Round a percentage for display: one decimal, no trailing ".0".
function fmtPct(n) {
  const v = Math.round(n * 10) / 10
  return Number.isInteger(v) ? String(v) : String(v.toFixed(1))
}

// Average coverage across backend + frontend: the simple mean of the two area
// percentages (each already gated at its own floor), with the mean of the two
// floors as its own. Returns null until both areas have run.
export function averageCoverage(coverage, coverageFloor, webCoverage, webCoverageFloor) {
  const b = areaCoverage(coverage, coverageFloor)
  const w = areaCoverage(webCoverage, webCoverageFloor)
  if (!b || !w) return null
  const pct = (b.pct + w.pct) / 2
  const floor = b.floor === null || w.floor === null ? null : (b.floor + w.floor) / 2
  return { pct, floor, ok: floor === null ? null : pct >= floor }
}

// Overall coverage across backend + frontend, weighted by each side's total
// statements: (covered_sum / total_sum). The overall floor is the same
// weighted average of the two area floors, so it stays in sync with both
// gates. Returns null until both areas have run — with only one area touched,
// its own row is the whole story.
export function overallCoverage({ backendCovered, backendTotal, frontendCovered, frontendTotal, backendFloor, frontendFloor }) {
  const bt = num(backendTotal)
  const ft = num(frontendTotal)
  if (!bt || !ft) return null
  const bc = num(backendCovered)
  const fc = num(frontendCovered)
  if (bc === null || fc === null) return null
  const pct = ((bc + fc) / (bt + ft)) * 100
  const bf = num(backendFloor)
  const ff = num(frontendFloor)
  const floor = bf === null || ff === null ? null : (bf * bt + ff * ft) / (bt + ft)
  return { pct, floor, ok: floor === null ? null : pct >= floor }
}

// The coverage table for the report — backend, frontend, average, overall —
// or null when no area ran. Rows whose inputs are missing are omitted
// (average and overall wait until both areas have run).
export function coverageTable({
  coverage,
  coverageFloor,
  webCoverage,
  webCoverageFloor,
  coverageCovered,
  coverageTotal,
  webCoverageCovered,
  webCoverageTotal,
}) {
  const rows = [
    ["📊 Backend (Go)", areaCoverage(coverage, coverageFloor)],
    ["🖥️ Frontend (React)", areaCoverage(webCoverage, webCoverageFloor)],
    ["📈 Average", averageCoverage(coverage, coverageFloor, webCoverage, webCoverageFloor)],
    ["🧮 Overall", overallCoverage({ backendCovered: coverageCovered, backendTotal: coverageTotal, backendFloor: coverageFloor, frontendCovered: webCoverageCovered, frontendTotal: webCoverageTotal, frontendFloor: webCoverageFloor })],
  ].filter(([, value]) => value !== null)
  if (rows.length === 0) return null
  const lines = ["| Area | Coverage | Floor | Gate |", "| --- | --- | --- | --- |"]
  for (const [area, { pct, floor, ok }] of rows) {
    const floorCell = floor === null ? "—" : `**${fmtPct(floor)}%**`
    const gateCell = ok === null ? "" : ok ? "✅" : "❌"
    lines.push(`| ${area} | **${fmtPct(pct)}%** | ${floorCell} | ${gateCell} |`)
  }
  return lines.join("\n")
}

// The step summary: title line, run/commit links, coverage line when known,
// then the job table. This is what the human opens the "summary + gate" step
// for — rendered even when the gate fails, which is why the exit code is
// decided only after writing it.
export function renderStepSummary({ rows, failed, repository, runNumber, serverUrl, runId, sha, coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal }) {
  const run = runUrl({ serverUrl, repository }, runId)
  const commit = commitUrl({ serverUrl, repository }, sha)
  const lines = [failed > 0 ? `## CI failed ❌ — ${failed} job(s) failed` : "## CI passed ✅"]
  lines.push("")
  lines.push(`Run: [${repository}#${runNumber}](${run}) · commit [\`${shortSha(sha)}\`](${commit})`)
  const table = coverageTable({ coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal })
  if (table) lines.push("", table)
  lines.push("", "| Job | Result |", "|---|---|")
  for (const row of rows) lines.push(`| ${row.name} | ${row.text} |`)
  return lines.join("\n")
}

// The prettier report posted (and updated in place) as a PR comment. Same
// data as the step summary, wrapped for a comment and tagged with the marker.
export function renderPrBody({ rows, failed, passed, total, repository, serverUrl, runNumber, runId, sha, coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal }) {
  const title = failed > 0 ? "### CI Report ❌" : "### CI Report ✅"
  const run = runUrl({ serverUrl, repository }, runId)
  const commit = commitUrl({ serverUrl, repository }, sha)
  const workflow = `${serverUrl}/${repository}/actions/workflows/final-gate.yml`
  const lines = [MARKER, "", title, "", `**${passed}/${total} jobs passed** · [Run #${runNumber}](${run}) · commit [\`${shortSha(sha)}\`](${commit})`]
  const table = coverageTable({ coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal })
  if (table) lines.push("", table)
  lines.push(
    "",
    "<details>",
    "<summary>Job results</summary>",
    "",
    "| Job | Result |",
    "|---|---|",
    ...rows.map((row) => `| ${row.name} | ${row.text} |`),
    "</details>",
    "",
    "---",
    `🤖 Updated automatically on every run by [final-gate](${workflow})`,
    "⛔ This check must pass before merging",
  )
  return lines.join("\n")
}

// Find the existing marker-tagged comment on a PR (any page), so a re-run
// patches it instead of posting a fresh one.
export async function findMarkerComment({ repository, apiUrl, pr, token }) {
  const base = `${apiUrl}/repos/${repository}/issues/${pr}/comments`
  for (let page = 1; ; page++) {
    const response = await fetch(`${base}?per_page=100&page=${page}`, { headers: API_HEADERS(token) })
    if (!response.ok) {
      throw new Error(`fetch comments [${response.status}]: ${await response.text()}`)
    }
    const comments = await response.json()
    if (!Array.isArray(comments)) break
    const match = comments.find((comment) => (comment.body ?? "").includes(MARKER))
    if (match) return match
    if (comments.length < 100) break
  }
  return null
}

// Create the report comment or update the existing marker-tagged one. Returns
// "created" or "updated" for logging.
export async function upsertReportComment({ repository, apiUrl, pr, token, body }) {
  const existing = await findMarkerComment({ repository, apiUrl, pr, token })
  const url = existing
    ? `${apiUrl}/repos/${repository}/issues/comments/${existing.id}`
    : `${apiUrl}/repos/${repository}/issues/${pr}/comments`
  const response = await fetch(url, {
    method: existing ? "PATCH" : "POST",
    headers: { ...API_HEADERS(token), "content-type": "application/json" },
    body: JSON.stringify({ body }),
  })
  if (!response.ok) {
    throw new Error(`update PR comment [${response.status}]: ${await response.text()}`)
  }
  return existing ? "updated" : "created"
}

// The PR number from the triggering event, when the run is pull_request
// triggered (final-gate is workflow_call, so the event payload is the
// caller's event). Returns null for non-PR runs and when the payload has no
// pull_request — the PR comment is a nicety, never a gate.
export function eventPullRequestNumber(eventPath) {
  if (!eventPath) return null
  try {
    const event = JSON.parse(readFileSync(eventPath, "utf8"))
    return event.pull_request?.number ?? null
  } catch {
    return null
  }
}
