import type { RecentNote } from '../../components/dashboard/ContinueCard'
import type { AttentionItem } from '../../components/dashboard/NeedsAttentionCard'
import type { TodayEvent } from '../../components/dashboard/TodayCard'

// Mock data backing the dashboard tiles until their endpoints land:
//   - inbox count + quick capture → TODO(#17): GET /api/todos-style inbox endpoint
//   - today's meetings + captures  → TODO: index-by-kind endpoint
//   - recent notes                 → TODO: index updated_at endpoint
//   - tags                         → TODO: index tags
//   - stats + activity             → TODO: index stats/counts endpoints
export const mockInbox = {
    count: 3,
    files: ['capture-01.md', 'capture-02.md', 'capture-03.md']
}

// Attention rows for the "Needs attention" widget.
export const mockAttention: AttentionItem[] = [
    {
        id: 'inbox',
        title: '3 captures waiting',
        detail: 'inbox/capture-01.md · inbox/capture-02.md · inbox/capture-03.md',
        tone: 'warning',
        kind: 'capture'
    },
    {
        id: 'todos',
        title: '2 open todos',
        detail: 'Wire the todos tile · Review the app-shell UI',
        tone: 'default',
        kind: 'todo'
    },
    {
        id: 'sync',
        title: '1 unsynced change',
        detail: 'knowledge/renovate-github-action.md not pushed',
        tone: 'danger',
        kind: 'sync'
    }
]

// The "Today" timeline — meetings and captures share the day's schedule.
export const mockToday: TodayEvent[] = [
    { id: 'm-standup', time: '09:30', title: 'Standup', path: 'meetings/2026-08-15-standup.md', kind: 'meeting' },
    {
        id: 'm-review',
        time: '14:00',
        title: 'Sprint review',
        path: 'meetings/2026-08-15-sprint-review.md',
        kind: 'meeting'
    },
    { id: 'c-audit', time: '08:15', title: 'npm audit capture', path: 'inbox/capture-01.md', kind: 'capture' }
]

export const mockTodos = [
    { text: 'Wire the todos tile to GET /api/todos', done: false },
    { text: 'Review the app-shell UI', done: false },
    { text: 'Ship the persistent CLI pool', done: true }
]

// recentNotes builds its timestamps from the current time so the resume
// strip stays anchored to "today" (like the chart day labels) and the
// relative dates read correctly in tests and in real usage.
export const mockRecentNotes = (): RecentNote[] => {
    const hoursAgo = (h: number) => new Date(Date.now() - h * 3_600_000).toISOString()
    return [
        {
            path: 'knowledge/renovate-github-action.md',
            title: 'Renovate GitHub action',
            kind: 'knowledge',
            updatedAt: hoursAgo(2)
        },
        { path: 'links/bookmarks.md', title: 'Bookmarks', kind: 'link', updatedAt: hoursAgo(5) },
        { path: 'meetings/2026-08-15-standup.md', title: 'Standup', kind: 'meeting', updatedAt: hoursAgo(26) }
    ]
}

export const mockTags = ['go', 'react', 'typescript', 'agent', 'renovate', 'github-actions', 'angular']

// Notes created per day for the last 7 days (oldest first). The chart
// computes the day labels from the current date, so the bars stay anchored
// to "today" without the mock going stale.
export const mockActivity = [2, 5, 1, 3, 0, 4, 6]

// Chat messages per day for the last 14 days (oldest first).
export const mockChatActivity = [4, 7, 2, 9, 5, 12, 8, 3, 6, 11, 9, 14, 7, 10]

// Note counts by kind — totals 128, matching mockStats.notes. The slice
// order follows the validated series palette (blue, orange, emerald, yellow).
export const mockNotesByKind = [
    { kind: 'Meetings', count: 24 },
    { kind: 'Captures', count: 18 },
    { kind: 'Knowledge', count: 63 },
    { kind: 'Links', count: 23 }
]

export const mockNotesByFolder = [
    { folder: 'knowledge', count: 63 },
    { folder: 'links', count: 23 },
    { folder: 'meetings', count: 24 },
    { folder: 'capture', count: 18 }
]

export const mockStats = {
    notes: 128,
    captures: mockInbox.count,
    openTodos: mockTodos.filter((t) => !t.done).length,
    lastSync: '2h ago'
}

// A wider recent-notes set for the "Recent notes" widget, distinct from the
// three-note Continue strip so the two widgets don't repeat each other.
export const mockRecentGrid = (): RecentNote[] => {
    const hoursAgo = (h: number) => new Date(Date.now() - h * 3_600_000).toISOString()
    return [
        {
            path: 'knowledge/portkey-gateway.md',
            title: 'Portkey gateway routing',
            kind: 'knowledge',
            updatedAt: hoursAgo(1)
        },
        {
            path: 'meetings/2026-08-24-sprint-review.md',
            title: 'Sprint review',
            kind: 'meeting',
            updatedAt: hoursAgo(3)
        },
        { path: 'captures/quick-thought.md', title: 'Quick thought', kind: 'capture', updatedAt: hoursAgo(6) },
        { path: 'links/graph-rag.md', title: 'Graph RAG papers', kind: 'link', updatedAt: hoursAgo(9) },
        { path: 'knowledge/tailwind-v4.md', title: 'Tailwind v4 tokens', kind: 'knowledge', updatedAt: hoursAgo(20) },
        { path: 'todos/TODO.md', title: 'Project todos', kind: 'capture', updatedAt: hoursAgo(30) }
    ]
}

// Wiki storage numbers for the Storage widget (mock until a size endpoint).
export const mockStorage = {
    sizeMB: 4.2,
    attachments: 24,
    percent: 34
}
