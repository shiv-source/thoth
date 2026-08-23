import { appendFileSync } from "node:fs"
import path from "node:path"
import { pathToFileURL } from "node:url"
import {
  eventPullRequestNumber,
  fetchRunJobs,
  githubContext,
  renderPrBody,
  renderStepSummary,
  reportJobs,
  upsertReportComment,
} from "./lib.mjs"

function writeStepOutput(name, value) {
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `${name}=${value}\n`)
}

export async function main() {
  const ctx = githubContext()
  const token = process.env.GH_TOKEN
  const wrapper = process.env.INPUT_WRAPPER?.trim() ?? ""
  const coverage = process.env.INPUT_COVERAGE?.trim() ?? ""
  const coverageFloor = process.env.INPUT_COVERAGE_FLOOR?.trim() ?? ""
  const webCoverage = process.env.INPUT_WEB_COVERAGE?.trim() ?? ""
  const webCoverageFloor = process.env.INPUT_WEB_COVERAGE_FLOOR?.trim() ?? ""
  const coverageCovered = process.env.INPUT_COVERAGE_COVERED?.trim() ?? ""
  const coverageTotal = process.env.INPUT_COVERAGE_TOTAL?.trim() ?? ""
  const webCoverageCovered = process.env.INPUT_WEB_COVERAGE_COVERED?.trim() ?? ""
  const webCoverageTotal = process.env.INPUT_WEB_COVERAGE_TOTAL?.trim() ?? ""

  if (!ctx.repository || !ctx.runId || !token) {
    console.error("ci-report: GITHUB_REPOSITORY, GITHUB_RUN_ID and GH_TOKEN are required")
    process.exitCode = 1
    return
  }

  const jobs = await fetchRunJobs({ repository: ctx.repository, apiUrl: ctx.apiUrl, runId: ctx.runId, token })
  const report = reportJobs(jobs, { self: ctx.job, wrapper })
  writeStepOutput("result", report.result)
  writeStepOutput("failed", String(report.failed))
  writeStepOutput("passed", String(report.passed))
  writeStepOutput("total", String(report.total))

  const summary = renderStepSummary({ ...report, ...ctx, runId: ctx.runId, runNumber: ctx.runNumber, sha: ctx.sha, coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal })
  if (process.env.GITHUB_STEP_SUMMARY) {
    appendFileSync(process.env.GITHUB_STEP_SUMMARY, `${summary}\n`)
  }

  if (ctx.eventName === "pull_request") {
    const pr = eventPullRequestNumber(ctx.eventPath)
    if (pr) {
      const body = renderPrBody({ ...report, ...ctx, runId: ctx.runId, runNumber: ctx.runNumber, sha: ctx.sha, coverage, coverageFloor, webCoverage, webCoverageFloor, coverageCovered, coverageTotal, webCoverageCovered, webCoverageTotal })
      const action = await upsertReportComment({ repository: ctx.repository, apiUrl: ctx.apiUrl, pr, token, body })
      console.log(`ci-report: ${action} report comment on PR #${pr}`)
    }
  }

  if (report.failed > 0) {
    console.error(`ci-report: ${report.failed} job(s) failed — gate not passed`)
    process.exitCode = 1
  } else {
    console.log(`ci-report: ${report.passed}/${report.total} job(s) succeeded or skipped — gate passed`)
  }
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  main()
}
