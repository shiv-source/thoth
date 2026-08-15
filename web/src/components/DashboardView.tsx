import { CheckCircle2, Circle, Clock, RefreshCw } from 'lucide-react'

// DashboardView is the home for quick-glance tiles. The data behind each
// tile is mocked until its backend lands:
//   - open todos      → TODO(#17): GET /api/todos
//   - recent notes    → TODO(#12-ish index endpoint, or /api/search)
//   - sync status     → TODO(#18): GET /api/git/status
//   - context usage   → TODO(#16): tokens on the turn_done frame

const MOCK_TODOS = [
    { text: 'Wire the todos tile to GET /api/todos', done: false },
    { text: 'Review the app-shell UI', done: false },
    { text: 'Ship the persistent CLI pool', done: true }
]

const MOCK_RECENT = ['links/bookmarks.md', 'knowledge/renovate-github-action.md', 'meetings/standup.md']

function Card({ title, children }: { title: string; children: React.ReactNode }) {
    return (
        <section className="rounded-xl border border-line bg-surface p-4">
            <h2 className="mb-3 text-xs font-medium uppercase tracking-wide text-subtle">{title}</h2>
            {children}
        </section>
    )
}

export function DashboardView() {
    return (
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
            <h1 className="font-display text-lg font-semibold text-heading">Dashboard</h1>
            <div className="grid gap-4 md:grid-cols-2">
                <Card title="Open todos">
                    <ul className="space-y-2">
                        {MOCK_TODOS.filter((t) => !t.done).map((t) => (
                            <li key={t.text} className="flex items-start gap-2 text-sm text-ink">
                                <Circle className="mt-0.5 h-4 w-4 shrink-0 text-subtle" aria-hidden="true" />
                                {t.text}
                            </li>
                        ))}
                    </ul>
                    {MOCK_TODOS.filter((t) => !t.done).length === 0 && (
                        <p className="text-sm text-subtle">All clear 🎉</p>
                    )}
                    <p className="mt-3 flex items-center gap-1.5 text-xs text-subtle">
                        <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
                        {MOCK_TODOS.filter((t) => t.done).length} done · mock data (#17)
                    </p>
                </Card>
                <Card title="Recent notes">
                    <ul className="space-y-2">
                        {MOCK_RECENT.map((p) => (
                            <li key={p} className="flex items-center gap-2 text-sm text-ink">
                                <Clock className="h-4 w-4 shrink-0 text-subtle" aria-hidden="true" />
                                {p}
                            </li>
                        ))}
                    </ul>
                    <p className="mt-3 text-xs text-subtle">mock data — wire the index’s updated_at</p>
                </Card>
                <Card title="Git sync">
                    <p className="flex items-center gap-2 text-sm text-ink">
                        <RefreshCw className="h-4 w-4 text-subtle" aria-hidden="true" />
                        Synced just now · mock data (#18)
                    </p>
                </Card>
                <Card title="Context usage">
                    <p className="text-sm text-ink">
                        This conversation: <span className="font-medium">38k</span> tokens · mock data (#16)
                    </p>
                </Card>
            </div>
        </div>
    )
}
