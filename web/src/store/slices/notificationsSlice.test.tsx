import { describe, expect, it } from 'vitest'
import { makeStore } from '../index'
import {
    dismissNotification,
    markAllRead,
    markNotificationRead,
    notify,
    selectNotifications,
    selectUnreadCount
} from './notificationsSlice'

describe('notificationsSlice', () => {
    it('starts empty', () => {
        const store = makeStore()
        expect(selectNotifications(store.getState())).toEqual([])
        expect(selectUnreadCount(store.getState())).toBe(0)
    })

    it('notify adds an unread item with an id', () => {
        const store = makeStore()
        store.dispatch(notify({ kind: 'sync', title: 'Synced', body: 'pushed 2 notes' }))
        const items = selectNotifications(store.getState())
        expect(items).toHaveLength(1)
        expect(items[0]?.kind).toBe('sync')
        expect(items[0]?.title).toBe('Synced')
        expect(items[0]?.read).toBe(false)
        expect(items[0]?.id).toBeTruthy()
        expect(selectUnreadCount(store.getState())).toBe(1)
    })

    it('marks one notification read without touching others', () => {
        const store = makeStore()
        store.dispatch(notify({ kind: 'sync', title: 'a' }))
        store.dispatch(notify({ kind: 'note', title: 'b' }))
        const first = selectNotifications(store.getState())[0]!
        store.dispatch(markNotificationRead(first.id))
        const items = selectNotifications(store.getState())
        expect(items.find((n) => n.id === first.id)?.read).toBe(true)
        expect(selectUnreadCount(store.getState())).toBe(1)
    })

    it('markAllRead clears every unread badge', () => {
        const store = makeStore()
        store.dispatch(notify({ kind: 'sync', title: 'a' }))
        store.dispatch(notify({ kind: 'doctor', title: 'b' }))
        store.dispatch(markAllRead())
        expect(selectUnreadCount(store.getState())).toBe(0)
        for (const n of selectNotifications(store.getState())) {
            expect(n.read).toBe(true)
        }
    })

    it('dismiss removes the notification', () => {
        const store = makeStore()
        store.dispatch(notify({ kind: 'sync', title: 'a' }))
        const first = selectNotifications(store.getState())[0]!
        store.dispatch(dismissNotification(first.id))
        expect(selectNotifications(store.getState())).toHaveLength(0)
    })

    it('caps the ring at 50 items, dropping the oldest', () => {
        const store = makeStore()
        for (let i = 0; i < 55; i++) {
            store.dispatch(notify({ kind: 'note', title: `n${i}` }))
        }
        const items = selectNotifications(store.getState())
        expect(items).toHaveLength(50)
        expect(items[0]?.title).toBe('n5')
        expect(items[49]?.title).toBe('n54')
    })
})
