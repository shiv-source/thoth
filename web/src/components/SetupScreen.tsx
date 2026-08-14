import type { Health } from '../api/client'

interface Problem {
  title: string
  fix: string | null
}

// problemsFromHealth lists what is wrong with the installation. A null health
// means the server could not be reached at all.
function problemsFromHealth(health: Health | null): Problem[] {
  const problems: Problem[] = []
  if (!health) {
    problems.push({ title: 'The Thoth server is unreachable.', fix: null })
    return problems
  }
  if (!health.claude.found) {
    problems.push({
      title: 'Claude Code is not installed or not on your PATH.',
      fix: 'npm i -g @anthropic-ai/claude-code && claude login',
    })
  }
  if (!health.wiki.exists) {
    problems.push({
      title: 'Your wiki directory does not exist yet.',
      fix: 'thoth init',
    })
  }
  return problems
}

export function SetupScreen({ health, loading, onRecheck }: {
  health: Health | null
  loading: boolean
  onRecheck: () => void
}) {
  const problems = problemsFromHealth(health)

  return (
    <div className="flex flex-1 items-center justify-center p-6">
      <div className="w-full max-w-md animate-[fade-in_150ms_ease-out] rounded-xl border border-line bg-surface p-6 shadow-md">
        <div className="flex flex-col items-center text-center">
          <span className="text-3xl" aria-hidden="true">🦉</span>
          <h1 className="mt-3 font-display text-2xl font-semibold text-heading">Thoth needs your attention</h1>
        </div>
        <ul className="mt-6 space-y-4">
          {problems.map((p) => (
            <li key={p.title} className="flex items-start gap-2.5">
              <span className="mt-0.5 text-red-600 dark:text-red-400" aria-hidden="true">✗</span>
              <div className="min-w-0">
                <p className="text-sm text-ink">{p.title}</p>
                {p.fix && (
                  <code className="mt-1.5 inline-block max-w-full overflow-x-auto rounded-lg bg-raised px-2 py-1 font-mono text-xs text-ink">
                    {p.fix}
                  </code>
                )}
              </div>
            </li>
          ))}
        </ul>
        <div className="mt-6 flex justify-center">
          <button onClick={onRecheck} disabled={loading}
            className="flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-60">
            {loading && (
              <span aria-hidden="true"
                className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-accent-ink/40 border-t-accent-ink" />
            )}
            {loading ? 'Re-checking…' : 'Re-check'}
          </button>
        </div>
      </div>
    </div>
  )
}
