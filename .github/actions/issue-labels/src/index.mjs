import { appendFileSync, readFileSync } from "node:fs"
import path from "node:path"
import { pathToFileURL } from "node:url"
import { computeLabels, loadConfig, missingLabels } from "./lib.mjs"

function writeStepOutput(name, value) {
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `${name}=${value}\n`)
}

export async function main() {
  const eventPath = process.env.GITHUB_EVENT_PATH
  const repository = process.env.GITHUB_REPOSITORY
  const token = process.env.GH_TOKEN
  const configPath = new URL(`../${process.env.LABELS_CONFIG ?? "config.json"}`, import.meta.url)

  if (!eventPath || !repository || !token) {
    console.error("issue-labels: GITHUB_EVENT_PATH, GITHUB_REPOSITORY and GH_TOKEN are required")
    process.exitCode = 1
    return
  }
  writeStepOutput("applied", "[]")

  const config = loadConfig(configPath)
  const event = JSON.parse(readFileSync(eventPath, "utf8"))
  const issue = event.issue

  if (!issue) {
    console.log("issue-labels: no issue in event payload; skipping")
    return
  }
  if (!["opened", "edited"].includes(event.action)) {
    console.log(`issue-labels: action "${event.action}" not handled; skipping`)
    return
  }

  const desired = computeLabels(issue.body ?? "", config)
  if (desired.length === 0) {
    console.log("issue-labels: no three-tier labels derivable from the issue body (blank issue?); skipping")
    return
  }

  const currentNames = (issue.labels ?? []).map((label) => label.name)
  const missing = missingLabels(desired, currentNames)
  if (missing.length === 0) {
    console.log(`issue-labels: bare-minimum labels already present (${desired.join(", ")}); nothing to add`)
    return
  }

  const response = await fetch(`https://api.github.com/repos/${repository}/issues/${issue.number}/labels`, {
    method: "POST",
    headers: {
      authorization: `Bearer ${token}`,
      accept: "application/vnd.github+json",
      "x-github-api-version": "2022-11-28",
      "content-type": "application/json",
    },
    body: JSON.stringify({ labels: missing }),
  })

  if (!response.ok) {
    const detail = await response.text()
    console.error(`issue-labels: failed to add labels [${response.status}] — ${detail}`)
    if (response.status === 422) {
      console.error("issue-labels: a 422/Validation Failed means a label in config.json does not exist on the repo yet")
    }
    process.exitCode = 1
    return
  }

  writeStepOutput("applied", JSON.stringify(missing))
  console.log(`issue-labels: applied ${missing.join(", ")} to #${issue.number}`)
  if (process.env.GITHUB_STEP_SUMMARY) {
    const link = `https://github.com/${repository}/issues/${issue.number}`
    appendFileSync(process.env.GITHUB_STEP_SUMMARY, `Applied bare-minimum labels to [#${issue.number}](${link}): ${missing.join(", ")}\n`)
  }
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  main()
}
