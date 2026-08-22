import { describe, expect, it } from 'vitest'
import { NOTIFICATION_ICONS } from './notifications'
import type { NotificationKind } from '../store/slices/notificationsSlice'

describe('notifications', () => {
    it('covers every notification kind with an icon', () => {
        const kinds: NotificationKind[] = ['sync', 'note', 'rulebook', 'chat', 'doctor']
        for (const k of kinds) {
            expect(NOTIFICATION_ICONS[k]).toBeTruthy()
        }
    })
})
