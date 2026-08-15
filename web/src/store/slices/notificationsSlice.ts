import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { RootState } from '../index'

export type NotificationKind = 'sync' | 'note' | 'rulebook' | 'chat' | 'doctor'

export interface Notification {
    id: string
    kind: NotificationKind
    title: string
    body?: string
    read: boolean
}

interface NotificationsState {
    items: Notification[]
}

// Notifications are ephemeral UI state — no persistence, a capped ring so
// the store cannot grow unbounded.
const initialState: NotificationsState = { items: [] }

const MAX_ITEMS = 50

export const notificationsSlice = createSlice({
    name: 'notifications',
    initialState,
    reducers: {
        notify: (s, a: PayloadAction<{ kind: NotificationKind; title: string; body?: string }>) => {
            s.items.push({
                id: crypto.randomUUID(),
                kind: a.payload.kind,
                title: a.payload.title,
                body: a.payload.body,
                read: false
            })
            if (s.items.length > MAX_ITEMS) {
                s.items.splice(0, s.items.length - MAX_ITEMS)
            }
        },
        markNotificationRead: (s, a: PayloadAction<string>) => {
            const n = s.items.find((x) => x.id === a.payload)
            if (n) n.read = true
        },
        markAllRead: (s) => {
            for (const n of s.items) n.read = true
        },
        dismissNotification: (s, a: PayloadAction<string>) => {
            s.items = s.items.filter((x) => x.id !== a.payload)
        }
    }
})

export const { notify, markNotificationRead, markAllRead, dismissNotification } = notificationsSlice.actions

export const selectNotifications = (s: RootState) => s.notifications.items
export const selectUnreadCount = (s: RootState) => s.notifications.items.filter((n) => !n.read).length
