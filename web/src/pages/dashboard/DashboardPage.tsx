import { useEffect } from 'react'
import { Button, Card, List, Progress, Statistic } from 'antd'
import {
    ClockCircleOutlined,
    FileTextOutlined,
    FolderOpenOutlined,
    InboxOutlined,
    CheckSquareOutlined,
    PlusSquareOutlined,
    EditOutlined,
    ReloadOutlined,
    SearchOutlined,
    TagOutlined
} from '@ant-design/icons'
import { navigate } from '../../hooks/useConversationRoute'
import { navigateView } from '../../hooks/useView'
import { fetchConversations, selectConversations } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { ActivityChart } from '../../components/dashboard/ActivityChart'
import { ChatActivityChart } from '../../components/dashboard/ChatActivityChart'
import { NotesByFolderChart } from '../../components/dashboard/NotesByFolderChart'
import { NotesByKindChart } from '../../components/dashboard/NotesByKindChart'
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
import { AppHeader } from '../../shared/AppHeader'

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

// StatTile is a KPI tile: an antd Card with a Statistic whose prefix icon
// carries the accent.
function StatTile({ icon: Icon, label, value }: { icon: typeof FileTextOutlined; label: string; value: string }) {
    return (
        <Card size="small">
            <Statistic title={label} value={value} prefix={<Icon aria-hidden="true" className="mr-1 text-accent" />} />
        </Card>
    )
}

export function DashboardPage({ onOpenSettings }: { onOpenSettings: () => void }) {
    const dispatch = useAppDispatch()
    const conversations = useAppSelector(selectConversations)

    useEffect(() => {
        void dispatch(fetchConversations())
    }, [dispatch])

    const recentChats = (conversations.list ?? []).slice(0, 3)
    const doneTodos = mockTodos.filter((t) => t.done).length

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <AppHeader title="Dashboard" onOpenSettings={onOpenSettings} />
            <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
                <div className="mx-auto w-full max-w-5xl space-y-4">
                    <header>
                        <h1 className="font-display text-xl font-semibold text-heading">{greeting()}</h1>
                        <p className="text-sm text-subtle">{todayLabel()}</p>
                    </header>

                    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
                        <StatTile icon={FileTextOutlined} label="Notes" value={String(mockStats.notes)} />
                        <StatTile icon={InboxOutlined} label="Captures" value={String(mockStats.captures)} />
                        <StatTile icon={CheckSquareOutlined} label="Open todos" value={String(mockStats.openTodos)} />
                        <StatTile icon={ReloadOutlined} label="Last sync" value={mockStats.lastSync} />
                    </div>

                    <div className="flex flex-wrap gap-2">
                        <Button
                            type="primary"
                            icon={<PlusSquareOutlined aria-hidden="true" />}
                            onClick={() => navigate('/chat')}
                        >
                            New chat
                        </Button>
                        <Button icon={<SearchOutlined aria-hidden="true" />} onClick={() => navigateView('search')}>
                            Ask the wiki
                        </Button>
                        <Button icon={<EditOutlined aria-hidden="true" />} onClick={() => navigateView('notes')}>
                            New note
                        </Button>
                    </div>

                    <h2 className="text-xs font-medium uppercase tracking-wide text-subtle">Overview</h2>
                    <div className="grid gap-4 md:grid-cols-2">
                        <Card size="small" title="Inbox needs attention">
                            <p className="flex items-center gap-2 text-sm text-ink">
                                <FolderOpenOutlined className="h-4 w-4 shrink-0 text-subtle" aria-hidden="true" />
                                {mockInbox.count} capture{mockInbox.count === 1 ? '' : 's'} waiting
                            </p>
                            <List
                                size="small"
                                dataSource={mockInbox.files}
                                renderItem={(f) => (
                                    <List.Item className="truncate text-xs text-subtle">inbox/{f}</List.Item>
                                )}
                            />
                            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
                        </Card>
                        <Card size="small" title="Today's meetings">
                            <List
                                size="small"
                                dataSource={mockMeetings}
                                renderItem={(m) => (
                                    <List.Item className="group px-2 hover:bg-raised">
                                        <span className="mr-2.5 shrink-0 rounded-md bg-raised px-1.5 py-0.5 font-mono text-xs text-subtle">
                                            {m.time}
                                        </span>
                                        <span className="min-w-0 flex-1">
                                            <span className="block truncate text-sm text-ink">{m.title}</span>
                                            <span className="block truncate text-xs text-subtle">{m.path}</span>
                                        </span>
                                        <ClockCircleOutlined
                                            className="h-4 w-4 shrink-0 text-subtle opacity-0 transition group-hover:opacity-100"
                                            aria-hidden="true"
                                        />
                                    </List.Item>
                                )}
                            />
                            <p className="mt-3 text-xs text-subtle">mock data — index by kind</p>
                        </Card>
                        <Card size="small" title="Open todos">
                            <List
                                size="small"
                                dataSource={mockTodos}
                                renderItem={(t) => (
                                    <List.Item
                                        className={`text-sm ${t.done ? 'text-subtle line-through' : 'text-ink'}`}
                                    >
                                        <span
                                            aria-hidden="true"
                                            className={`mr-2.5 flex h-4 w-4 shrink-0 items-center justify-center rounded border text-[10px] ${
                                                t.done
                                                    ? 'border-accent bg-accent text-accent-ink'
                                                    : 'border-line bg-app'
                                            }`}
                                        >
                                            {t.done ? '✓' : ''}
                                        </span>
                                        {t.text}
                                    </List.Item>
                                )}
                            />
                            <div className="mt-3 flex items-center gap-2">
                                <Progress
                                    percent={Math.round((doneTodos / mockTodos.length) * 100)}
                                    size="small"
                                    className="flex-1"
                                />
                                <span className="shrink-0 text-xs text-subtle">
                                    {doneTodos} of {mockTodos.length} done
                                </span>
                            </div>
                            <p className="mt-3 text-xs text-subtle">mock data (#17)</p>
                        </Card>
                        <Card size="small" title="Recent notes">
                            <List
                                size="small"
                                dataSource={mockRecent}
                                renderItem={(p) => <List.Item className="text-sm text-ink">{p}</List.Item>}
                            />
                            <p className="mt-3 text-xs text-subtle">mock data — wire the index's updated_at</p>
                        </Card>
                        <Card size="small" title="Recent chats">
                            {recentChats.length === 0 ? (
                                <p className="text-sm text-subtle">No conversations yet</p>
                            ) : (
                                <List
                                    size="small"
                                    dataSource={recentChats}
                                    renderItem={(c) => (
                                        <List.Item
                                            onClick={() => navigate(`/chat/${c.id}`)}
                                            className="cursor-pointer truncate text-sm text-ink hover:bg-raised"
                                        >
                                            {c.title}
                                        </List.Item>
                                    )}
                                />
                            )}
                        </Card>
                        <Card size="small" title="Tags">
                            <div className="flex flex-wrap gap-2">
                                {mockTags.map((t) => (
                                    <Button
                                        key={t}
                                        shape="round"
                                        size="small"
                                        icon={<TagOutlined aria-hidden="true" />}
                                        onClick={() => navigateView('search')}
                                    >
                                        #{t}
                                    </Button>
                                ))}
                            </div>
                            <p className="mt-3 text-xs text-subtle">mock data — index tags</p>
                        </Card>
                    </div>

                    <h2 className="text-xs font-medium uppercase tracking-wide text-subtle">Insights</h2>
                    <div className="grid gap-4 md:grid-cols-2">
                        <Card size="small" title="Notes this week">
                            <ActivityChart counts={mockActivity} />
                            <p className="mt-3 text-xs text-subtle">mock data — index stats endpoint</p>
                        </Card>
                        <div className="md:col-span-2">
                            <Card size="small" title="Chat activity">
                                <ChatActivityChart counts={mockChatActivity} />
                                <p className="mt-3 text-xs text-subtle">mock data — messages endpoint</p>
                            </Card>
                        </div>
                        <Card size="small" title="Notes by kind">
                            <NotesByKindChart slices={mockNotesByKind} />
                            <p className="mt-3 text-xs text-subtle">mock data — kind counts</p>
                        </Card>
                        <Card size="small" title="Notes by folder">
                            <NotesByFolderChart rows={mockNotesByFolder} />
                            <p className="mt-3 text-xs text-subtle">mock data — folder counts</p>
                        </Card>
                    </div>
                </div>
            </div>
        </div>
    )
}
