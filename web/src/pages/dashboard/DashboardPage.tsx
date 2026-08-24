import { useEffect } from 'react'
import { App, Button } from 'antd'
import {
    FileTextOutlined,
    InboxOutlined,
    CheckSquareOutlined,
    PlusSquareOutlined,
    EditOutlined,
    ReloadOutlined,
    SearchOutlined
} from '@ant-design/icons'
import { navigate } from '../../hooks/useConversationRoute'
import { navigateNote, navigateView } from '../../hooks/useView'
import { fetchConversations, selectConversations } from '../../store'
import { useAppDispatch, useAppSelector } from '../../store/hooks'
import { ActivityChart } from '../../components/dashboard/ActivityChart'
import { ChatActivityChart } from '../../components/dashboard/ChatActivityChart'
import { NotesByFolderChart } from '../../components/dashboard/NotesByFolderChart'
import { NotesByKindChart } from '../../components/dashboard/NotesByKindChart'
import { ChartCard } from '../../components/dashboard/ChartCard'
import { ContinueCard } from '../../components/dashboard/ContinueCard'
import { NeedsAttentionCard } from '../../components/dashboard/NeedsAttentionCard'
import { TodayCard } from '../../components/dashboard/TodayCard'
import { QuickCaptureCard } from '../../components/dashboard/QuickCaptureCard'
import { StatTile } from '../../components/dashboard/StatTile'
import { TagsCard } from '../../components/dashboard/TagsCard'
import {
    mockActivity,
    mockAttention,
    mockChatActivity,
    mockNotesByFolder,
    mockNotesByKind,
    mockRecentNotes,
    mockStats,
    mockTags,
    mockToday
} from './dashboardMock'
import { AppHeader } from '../../shared/AppHeader'
import { SectionHeader } from '../../shared/SectionHeader'

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

// DashboardPage is the landing view: a greeting header, the KPI tiles, quick
// actions, the Overview widget cards, and the Insights chart cards. Each
// widget is its own component in components/dashboard; the page wires them to
// the mock data (pages/dashboard/dashboardMock.ts) until the index endpoints
// land. The Overview is a priority lane: quick capture, a resume strip, the
// needs-attention and today panels, then the tag cloud.
export function DashboardPage({ onOpenSettings }: { onOpenSettings: () => void }) {
    const dispatch = useAppDispatch()
    const { message } = App.useApp()
    const conversations = useAppSelector(selectConversations)

    useEffect(() => {
        void dispatch(fetchConversations())
    }, [dispatch])

    const recentChats = (conversations.list ?? []).slice(0, 3)

    const openNote = (path: string) => {
        if (path.startsWith('inbox/')) {
            // inbox captures are not viewable notes yet — acknowledge the mock
            message.info('Mock capture — the inbox viewer is on the roadmap')
            return
        }
        navigateNote(path)
    }

    const capture = (text: string) => {
        const slug = text
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '')
            .slice(0, 48)
        message.success(`Captured to inbox/${slug || 'capture'}.md (mock)`)
    }

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <AppHeader title="Dashboard" onOpenSettings={onOpenSettings} />
            <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5">
                <div className="mx-auto w-full max-w-5xl space-y-5">
                    <header>
                        <h1 className="font-display text-2xl font-semibold tracking-tight text-heading">
                            {greeting()}
                        </h1>
                        <p className="mt-1 text-sm text-subtle">{todayLabel()}</p>
                    </header>

                    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
                        <StatTile icon={FileTextOutlined} label="Notes" value={String(mockStats.notes)} delta="+12" />
                        <StatTile icon={InboxOutlined} label="Captures" value={String(mockStats.captures)} delta="+3" />
                        <StatTile
                            icon={CheckSquareOutlined}
                            label="Open todos"
                            value={String(mockStats.openTodos)}
                            delta="-1"
                        />
                        <StatTile icon={ReloadOutlined} label="Last sync" value={mockStats.lastSync} />
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
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

                    <SectionHeader>Overview</SectionHeader>
                    <div className="space-y-5">
                        <QuickCaptureCard onCapture={capture} />
                        <ContinueCard
                            chats={recentChats}
                            notes={mockRecentNotes()}
                            onOpenChat={(id) => navigate(`/chat/${id}`)}
                            onOpenNote={openNote}
                        />
                        <div className="grid gap-4 md:grid-cols-2">
                            <NeedsAttentionCard items={mockAttention} onOpen={() => navigateView('notes')} />
                            <TodayCard events={mockToday} onOpen={openNote} />
                        </div>
                        <TagsCard tags={mockTags} onOpen={() => navigateView('search')} />
                    </div>

                    <SectionHeader>Insights</SectionHeader>
                    <div className="grid gap-4 md:grid-cols-2">
                        <ChartCard title="Notes this week" note="mock data — index stats endpoint">
                            <ActivityChart counts={mockActivity} />
                        </ChartCard>
                        <div className="md:col-span-2">
                            <ChartCard title="Chat activity" note="mock data — messages endpoint">
                                <ChatActivityChart counts={mockChatActivity} />
                            </ChartCard>
                        </div>
                        <ChartCard title="Notes by kind" note="mock data — kind counts">
                            <NotesByKindChart slices={mockNotesByKind} />
                        </ChartCard>
                        <ChartCard title="Notes by folder" note="mock data — folder counts">
                            <NotesByFolderChart rows={mockNotesByFolder} />
                        </ChartCard>
                    </div>
                </div>
            </div>
        </div>
    )
}
