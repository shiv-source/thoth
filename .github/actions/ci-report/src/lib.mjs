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

// One-line coverage summary for the report: actual % and the area's floor
// side by side, with a pass/fail marker against the floor. Returns null when
// no coverage value was provided (area not touched → the line is omitted).
export function coverageText(coverage, coverageFloor, { emoji = "📊", label = "Backend coverage" } = {}) {
  const actual = String(coverage ?? "").trim().replace(/%$/, "")
  if (!actual) return null
  const floor = String(coverageFloor ?? "").trim().replace(/%$/, "")
  const head = `${emoji} ${label}: **${actual}%**`
  if (!floor) return head
  const ok = Number(actual) >= Number(floor)
  return `${head} — floor **${floor}%** ${ok ? "✅" : "❌"}`
}

// Round a percentage for display: one decimal, no trailing ".0".
function fmtPct(n) {
  const v = Math.round(n * 10) / 10
  return Number.isInteger(v) ? String(v) : String(v.toFixed(1))
}

// Overall coverage across backend + frontend, weighted by each side's total
// statements: (covered_sum / total_sum). The overall floor is the same
// weighted average of the two area floors, so it stays in sync with both
// gates. Returns null until both areas have run — with only one area touched,
// its own line is the whole story.
export function overallCoverageText({ backendCovered, backendTotal, frontendCovered, frontendTotal, backendFloor, frontendFloor }) {
  const bt = Number(backendTotal)
  const ft = Number(frontendTotal)
  const missing = (v) => v === undefined || v === null || v === ""
  if (!bt || !ft || missing(backendCovered) || missing(frontendCovered)) return null
  const pct = ((Number(backendCovered) + Number(frontendCovered)) / (bt + ft)) * 100
  const floor = (Number(backendFloor) * bt + Number(frontendFloor) * ft) / (bt + ft)
  const ok = pct >= floor
  return `🧮 Overall coverage: **${fmtPct(pct)}%** — floor **${fmtPct(floor)}%** ${ok ? "✅" : "❌"}`
}

// Average coverage across backend + frontend: the simple mean of the two area
// percentages (each already gated at its own floor), with the mean of the two
// floors as its own. Returns null until both areas have run.
export function averageCoverageText(coverage, coverageFloor, webCoverage, webCoverageFloor) {
  const b = String(coverage ?? "").trim().replace(/%$/, "")
  const w = String(webCoverage ?? "").trim().replace(/%$/, "")
  if (!b || !w) return null
  const avg = (Number(b) + Number(w)) / 2
  const bf = Number(String(coverageFloor ?? "").trim().replace(/%$/, ""))
  const wf = Number(String(webCoverageFloor ?? "").trim().replace(/%$/, ""))
  if (!bf || !wf) return `📈 Average coverage: **${fmtPct(avg)}%**`
  const floor = (bf + wf) / 2
  const ok = avg >= floor
  return `📈 Average coverage: **${fmtPct(avg)}%** — floor **${fmtPct(floor)}%** ${ok ? "✅" : "❌"}`
}

// The coverage lines for a report: backend, frontend, average, then overall.
// Each is omitted when its inputs are missing, so untouched areas simply
// disappear (average and overall wait until both areas have run).
function coverageLines({
  coverage,
  coverageFloor,
  webCoverage,
  webCoverageFloor,
  coverageCovered,
  coverageTotal,
  webCoverageCovered,
  webCoverageTotal,
}) {
  return [
    coverageText(coverage, coverageFloor),
    coverageText(webCoverage, webCoverageFloor, { emoji: "🖥️", label: "Frontend coverage" }),
    averageCoverageText(coverage, coverageFloor, webCoverage, webCoverageFloor),
    overallCoverageText({
      backendCovered: coverageCovered,
      backendTotal: coverageTotal,
      backendFloor: coverageFloor,
      frontendCovered: webCoverageCovered,
      frontendTotal: webCoverageTotal,
      frontendFloor: webCoverageFloor,
    }),
  ].filter(Boolean)
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
  const covLines = coverageLines({ coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal })
  if (covLines.length > 0) lines.push("", ...covLines)
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
  const covLines = coverageLines({ coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal })
  if (covLines.length > 0) lines.push("", ...covLines)
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
