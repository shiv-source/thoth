import { useEffect } from 'react'
import { Clock, FolderOpen, MessageSquarePlus, PenLine, Search } from 'lucide-react'
import { navigate } from '../hooks/useConversationRoute'
import { navigateView } from '../hooks/useView'
import { fetchConversations, selectConversations } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { TopBar } from './TopBar'

// DashboardView is the launcher + resume home. Data behind the mock tiles is
// tagged with its issue until the backend lands:
//   - inbox count      → TODO(#17): GET /api/todos-style inbox endpoint
//   - today's meetings → TODO: index-by-kind endpoint
//   - recent notes     → TODO: index updated_at endpoint
//   - new-note templates → TODO(#6): note templates
const MOCK_INBOX = {
    count: 3,
    files: ['capture-01.md', 'capture-02.md', 'capture-03.md']
}

const MOCK_MEETINGS = [
    { time: '09:30', title: 'Standup', path: 'meetings/2026-08-15-standup.md' },
    { time: '14:00', title: 'Sprint review', path: 'meetings/2026-08-15-sprint-review.md' }
]

const MOCK_TODOS = [
    { text: 'Wire the todos tile to GET /api/todos', done: false },
    { text: 'Review the app-shell UI', done: false },
    { text: 'Ship the persistent CLI pool', done: true }
]

const MOCK_RECENT = ['links/bookmarks.md', 'knowledge/renovate-github-action.md', 'knowledge/angular-cli-reference.md']

function greeting(): string {
    const h = new Date().getHours()
    if (h < 12) return 'Good morning'
    if (h < 17) return 'Good afternoon'
    return 'Good evening'
}

function todayLabel(): string {
    // Explicit locale keeps the format stable across environments (en-US is
    // always available, even without full ICU).
    return new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
    return (
        <section className="rounded-xl border border-line bg-surface p-4">
            <h2 className="mb-3 text-xs font-medium uppercase tracking-wide text-subtle">{title}</h2>
            {children}
        </section>
    )
}

export function DashboardView({ onOpenSettings }: { onOpenSettings: () => void }) {
    const dispatch = useAppDispatch()
    const conversations = useAppSelector(selectConversations)

    useEffect(() => {
        void dispatch(fetchConversations())
    }, [dispatch])

    const recentChats = (conversations.list ?? []).slice(0, 3)

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <TopBar title="Dashboard" onOpenSettings={onOpenSettings} />
            <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4">
                <div className="mx-auto w-full max-w-5xl space-y-4">
                    <header>
                        <h1 className="font-display text-xl font-semibold text-heading">{greeting()}</h1>
                        <p className="text-sm text-subtle">{todayLabel()}</p>
                    </header>

                    <div className="flex flex-wrap gap-2">
                        <button
                            type="button"
                            onClick={() => {
                                navigate('/')
                                navigateView('chat')
                            }}
                            className="flex items-center gap-2 rounded-lg bg-accent px-3 py-2 text-sm font-medium text-accent-ink transition hover:bg-accent-hover"
                        >
                            <MessageSquarePlus className="h-4 w-4" aria-hidden="true" />
                            New chat
                        </button>
                        <button
                            type="button"
                            onClick={() => navigateView('search')}
                            className="flex items-center gap-2 rounded-lg border border-line bg-surface px-3 py-2 text-sm font-medium text-ink transition hover:bg-raised"
                        >
                            <Search className="h-4 w-4" aria-hidden="true" />
                            Ask the wiki
                        </button>
                        <button
                            type="button"
                            onClick={() => navigateView('notes')}
                            className="flex items-center gap-2 rounded-lg border border-line bg-surface px-3 py-2 text-sm font-medium text-ink transition hover:bg-raised"
                        >
                            <PenLine className="h-4 w-4" aria-hidden="true" />
                            New note
                        </button>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                        <Card title="Inbox needs attention">
                            <p className="flex items-center gap-2 text-sm text-ink">
                                <FolderOpen className="h-4 w-4 shrink-0 text-subtle" aria-hidden="true" />
                                {MOCK_INBOX.count} capture{MOCK_INBOX.count === 1 ? '' : 's'} waiting
                            </p>
                            <ul className="mt-2 space-y-1">
                                {MOCK_INBOX.files.map((f) => (
                                    <li key={f} className="truncate text-xs text-subtle">
                                        inbox/{f}
                                    </li>
                                ))}
                            </ul>
                            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
                        </Card>
                        <Card title="Today's meetings">
                            <ul className="space-y-2">
                                {MOCK_MEETINGS.map((m) => (
                                    <li key={m.path} className="flex items-center gap-2 text-sm text-ink">
                                        <Clock className="h-4 w-4 shrink-0 text-subtle" aria-hidden="true" />
                                        <span className="font-mono text-xs text-subtle">{m.time}</span>
                                        {m.title}
                                    </li>
                                ))}
                            </ul>
                            <p className="mt-3 text-xs text-subtle">mock data — index by kind</p>
                        </Card>
                        <Card title="Open todos">
                            <ul className="space-y-2">
                                {MOCK_TODOS.filter((t) => !t.done).map((t) => (
                                    <li key={t.text} className="text-sm text-ink">
                                        {t.text}
                                    </li>
                                ))}
                            </ul>
                            <p className="mt-3 text-xs text-subtle">
                                {MOCK_TODOS.filter((t) => t.done).length} done · mock data (#17)
                            </p>
                        </Card>
                        <Card title="Recent notes">
                            <ul className="space-y-2">
                                {MOCK_RECENT.map((p) => (
                                    <li key={p} className="truncate text-sm text-ink">
                                        {p}
                                    </li>
                                ))}
                            </ul>
                            <p className="mt-3 text-xs text-subtle">mock data — wire the index's updated_at</p>
                        </Card>
                        <Card title="Recent chats">
                            {recentChats.length === 0 ? (
                                <p className="text-sm text-subtle">No conversations yet</p>
                            ) : (
                                <ul className="space-y-2">
                                    {recentChats.map((c) => (
                                        <li key={c.id}>
                                            <button
                                                type="button"
                                                onClick={() => {
                                                    navigate(`/chat/${c.id}`)
                                                    navigateView('chat')
                                                }}
                                                className="w-full truncate rounded-md px-1 py-1 text-left text-sm text-ink transition hover:bg-raised"
                                            >
                                                {c.title}
                                            </button>
                                        </li>
                                    ))}
                                </ul>
                            )}
                        </Card>
                    </div>
                </div>
            </div>
        </div>
    )
}
