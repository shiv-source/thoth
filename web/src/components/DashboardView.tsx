import { useEffect } from 'react'
import {
    Clock,
    FileText,
    FolderOpen,
    Inbox,
    ListTodo,
    MessageSquarePlus,
    PenLine,
    RefreshCw,
    Search,
    Tag
} from 'lucide-react'
import { navigate } from '../hooks/useConversationRoute'
import { navigateView } from '../hooks/useView'
import { fetchConversations, selectConversations } from '../store'
import { useAppDispatch, useAppSelector } from '../store/hooks'
import { ActivityChart } from './ActivityChart'
import { Card } from './Card'
import { ChatActivityChart } from './ChatActivityChart'
import { NotesByFolderChart } from './NotesByFolderChart'
import { NotesByKindChart } from './NotesByKindChart'
import {
    mockActivity,
    mockChatActivity,
    mockInbox,
    mockMeetings,
    mockNotesByFolder,
    mockNotesByKind,
    mockRecent,
    mockStats,
    mockTags,
    mockTodos
} from './dashboardMock'
import { TopBar } from './TopBar'

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

function StatTile({ icon: Icon, label, value }: { icon: typeof FileText; label: string; value: string }) {
    return (
        <div className="flex items-center gap-3 rounded-xl border border-line bg-surface px-4 py-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-accent-soft text-accent">
                <Icon className="h-4.5 w-4.5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
                <p className="font-display text-xl font-semibold leading-tight text-heading">{value}</p>
                <p className="truncate text-xs text-subtle">{label}</p>
            </div>
        </div>
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
            <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
                <div className="mx-auto w-full max-w-5xl space-y-4">
                    <header>
                        <h1 className="font-display text-xl font-semibold text-heading">{greeting()}</h1>
                        <p className="text-sm text-subtle">{todayLabel()}</p>
                    </header>

                    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
                        <StatTile icon={FileText} label="Notes" value={String(mockStats.notes)} />
                        <StatTile icon={Inbox} label="Captures" value={String(mockStats.captures)} />
                        <StatTile icon={ListTodo} label="Open todos" value={String(mockStats.openTodos)} />
                        <StatTile icon={RefreshCw} label="Last sync" value={mockStats.lastSync} />
                    </div>

                    <div className="flex flex-wrap gap-2">
                        <button
                            type="button"
                            onClick={() => navigate('/chat')}
                            className="flex items-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-medium text-accent-ink transition hover:bg-accent-hover"
                        >
                            <MessageSquarePlus className="h-4 w-4" aria-hidden="true" />
                            New chat
                        </button>
                        <button
                            type="button"
                            onClick={() => navigateView('search')}
                            className="flex items-center gap-2 rounded-xl border border-line bg-surface px-4 py-2.5 text-sm font-medium text-ink transition hover:bg-raised"
                        >
                            <Search className="h-4 w-4" aria-hidden="true" />
                            Ask the wiki
                        </button>
                        <button
                            type="button"
                            onClick={() => navigateView('notes')}
                            className="flex items-center gap-2 rounded-xl border border-line bg-surface px-4 py-2.5 text-sm font-medium text-ink transition hover:bg-raised"
                        >
                            <PenLine className="h-4 w-4" aria-hidden="true" />
                            New note
                        </button>
                    </div>

                    <h2 className="text-xs font-medium uppercase tracking-wide text-subtle">Overview</h2>
                    <div className="grid gap-4 md:grid-cols-2">
                        <Card title="Inbox needs attention">
                            <p className="flex items-center gap-2 text-sm text-ink">
                                <FolderOpen className="h-4 w-4 shrink-0 text-subtle" aria-hidden="true" />
                                {mockInbox.count} capture{mockInbox.count === 1 ? '' : 's'} waiting
                            </p>
                            <ul className="mt-2 space-y-1">
                                {mockInbox.files.map((f) => (
                                    <li key={f} className="truncate text-xs text-subtle">
                                        inbox/{f}
                                    </li>
                                ))}
                            </ul>
                            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
                        </Card>
                        <Card title="Today's meetings">
                            <ul className="space-y-1">
                                {mockMeetings.map((m) => (
                                    <li
                                        key={m.path}
                                        className="group flex items-center gap-2.5 rounded-lg px-2 py-1.5 transition hover:bg-raised"
                                    >
                                        <span className="shrink-0 rounded-md bg-raised px-1.5 py-0.5 font-mono text-xs text-subtle">
                                            {m.time}
                                        </span>
                                        <span className="min-w-0 flex-1">
                                            <span className="block truncate text-sm text-ink">{m.title}</span>
                                            <span className="block truncate text-xs text-subtle">{m.path}</span>
                                        </span>
                                        <Clock
                                            className="h-4 w-4 shrink-0 text-subtle opacity-0 transition group-hover:opacity-100"
                                            aria-hidden="true"
                                        />
                                    </li>
                                ))}
                            </ul>
                            <p className="mt-3 text-xs text-subtle">mock data — index by kind</p>
                        </Card>
                        <Card title="Open todos">
                            <ul className="space-y-1">
                                {mockTodos.map((t) => (
                                    <li
                                        key={t.text}
                                        className={`flex items-center gap-2.5 rounded-lg px-2 py-1.5 text-sm ${
                                            t.done ? 'text-subtle line-through' : 'text-ink'
                                        }`}
                                    >
                                        <span
                                            aria-hidden="true"
                                            className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border text-[10px] ${
                                                t.done
                                                    ? 'border-accent bg-accent text-accent-ink'
                                                    : 'border-line bg-app'
                                            }`}
                                        >
                                            {t.done ? '✓' : ''}
                                        </span>
                                        {t.text}
                                    </li>
                                ))}
                            </ul>
                            <div className="mt-3 flex items-center gap-2">
                                <span aria-hidden="true" className="h-1 flex-1 overflow-hidden rounded-full bg-raised">
                                    <span
                                        className="block h-full rounded-full bg-accent"
                                        style={{
                                            width: `${(mockTodos.filter((t) => t.done).length / mockTodos.length) * 100}%`
                                        }}
                                    />
                                </span>
                                <span className="shrink-0 text-xs text-subtle">
                                    {mockTodos.filter((t) => t.done).length} of {mockTodos.length} done
                                </span>
                            </div>
                            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
                        </Card>
                        <Card title="Recent notes">
                            <ul className="space-y-1">
                                {mockRecent.map((p) => (
                                    <li key={p} className="truncate rounded-md px-1 py-1 text-sm text-ink">
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
                                <ul className="space-y-1">
                                    {recentChats.map((c) => (
                                        <li key={c.id}>
                                            <button
                                                type="button"
                                                onClick={() => navigate(`/chat/${c.id}`)}
                                                className="w-full truncate rounded-md px-1 py-1 text-left text-sm text-ink transition hover:bg-raised"
                                            >
                                                {c.title}
                                            </button>
                                        </li>
                                    ))}
                                </ul>
                            )}
                        </Card>
                        <Card title="Tags">
                            <div className="flex flex-wrap gap-2">
                                {mockTags.map((t) => (
                                    <button
                                        key={t}
                                        type="button"
                                        onClick={() => navigateView('search')}
                                        className="flex items-center gap-1 rounded-full border border-line bg-app px-3 py-1 text-xs text-ink transition hover:bg-raised"
                                    >
                                        <Tag className="h-3 w-3 text-subtle" aria-hidden="true" />#{t}
                                    </button>
                                ))}
                            </div>
                            <p className="mt-3 text-xs text-subtle">mock data — index tags</p>
                        </Card>
                    </div>

                    <h2 className="text-xs font-medium uppercase tracking-wide text-subtle">Insights</h2>
                    <div className="grid gap-4 md:grid-cols-2">
                        <Card title="Notes this week">
                            <ActivityChart counts={mockActivity} />
                            <p className="mt-3 text-xs text-subtle">mock data — index stats endpoint</p>
                        </Card>
                        <div className="md:col-span-2">
                            <Card title="Chat activity">
                                <ChatActivityChart counts={mockChatActivity} />
                                <p className="mt-3 text-xs text-subtle">mock data — messages endpoint</p>
                            </Card>
                        </div>
                        <Card title="Notes by kind">
                            <NotesByKindChart slices={mockNotesByKind} />
                            <p className="mt-3 text-xs text-subtle">mock data — kind counts</p>
                        </Card>
                        <Card title="Notes by folder">
                            <NotesByFolderChart rows={mockNotesByFolder} />
                            <p className="mt-3 text-xs text-subtle">mock data — folder counts</p>
                        </Card>
                    </div>
                </div>
            </div>
        </div>
    )
}
