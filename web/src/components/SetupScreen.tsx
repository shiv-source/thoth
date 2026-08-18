import { Alert, Button, Result } from 'antd'
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
            fix: 'npm i -g @anthropic-ai/claude-code && claude login'
        })
    }
    if (!health.wiki.exists) {
        problems.push({
            title: 'Your wiki directory does not exist yet.',
            fix: 'thoth init'
        })
    }
    return problems
}

export function SetupScreen({
    health,
    loading,
    onRecheck
}: {
    health: Health | null
    loading: boolean
    onRecheck: () => void
}) {
    const problems = problemsFromHealth(health)

    return (
        <div className="flex flex-1 items-center justify-center p-6">
            <Result
                className="animate-[fade-in_150ms_ease-out]"
                icon={
                    <span className="text-3xl" aria-hidden="true">
                        🦉
                    </span>
                }
                title="Thoth needs your attention"
                subTitle={
                    <div className="mx-auto max-w-md text-left">
                        {problems.map((p) => (
                            <Alert
                                key={p.title}
                                type="error"
                                showIcon
                                message={p.title}
                                description={
                                    p.fix && (
                                        <code className="inline-block max-w-full overflow-x-auto rounded-lg bg-raised px-2 py-1 font-mono text-xs text-ink">
                                            {p.fix}
                                        </code>
                                    )
                                }
                                className="mb-2"
                            />
                        ))}
                    </div>
                }
                extra={
                    <Button type="primary" loading={loading} onClick={onRecheck}>
                        Re-check
                    </Button>
                }
            />
        </div>
    )
}
