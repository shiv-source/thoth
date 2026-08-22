// Mock data backing the dashboard tiles until their endpoints land:
//   - inbox count      → TODO(#17): GET /api/todos-style inbox endpoint
//   - today's meetings → TODO: index-by-kind endpoint
//   - recent notes     → TODO: index updated_at endpoint
//   - tags             → TODO: index tags
//   - stats + activity → TODO: index stats/counts endpoints
export const mockInbox = {
    count: 3,
    files: ['capture-01.md', 'capture-02.md', 'capture-03.md']
}

export const mockMeetings = [
    { time: '09:30', title: 'Standup', path: 'meetings/2026-08-15-standup.md' },
    { time: '14:00', title: 'Sprint review', path: 'meetings/2026-08-15-sprint-review.md' }
]

export const mockTodos = [
    { text: 'Wire the todos tile to GET /api/todos', done: false },
    { text: 'Review the app-shell UI', done: false },
    { text: 'Ship the persistent CLI pool', done: true }
]

export const mockRecent = [
    'links/bookmarks.md',
    'knowledge/renovate-github-action.md',
    'knowledge/angular-cli-reference.md'
]

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
