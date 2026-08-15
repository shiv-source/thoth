import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { notify, selectUnreadCount } from '../store'
import { renderWithStore } from '../test/renderWithStore'
import { NotificationPanel } from './NotificationPanel'

describe('NotificationPanel', () => {
    it('shows the empty state when there are no notifications', () => {
        renderWithStore(<NotificationPanel onClose={vi.fn()} />)
        expect(screen.getByText('No notifications yet')).toBeInTheDocument()
    })

    it('renders notifications with title and body', () => {
        const { store } = renderWithStore(<NotificationPanel onClose={vi.fn()} />)
        act(() => {
            store.dispatch(notify({ kind: 'note', title: 'Note saved', body: 'knowledge/notes.md' }))
        })
        expect(screen.getByText('Note saved')).toBeInTheDocument()
        expect(screen.getByText('knowledge/notes.md')).toBeInTheDocument()
    })

    it('marks everything read and clears the badge', async () => {
        const { store } = renderWithStore(<NotificationPanel onClose={vi.fn()} />)
        act(() => {
            store.dispatch(notify({ kind: 'sync', title: 'a' }))
            store.dispatch(notify({ kind: 'doctor', title: 'b' }))
        })
        expect(selectUnreadCount(store.getState())).toBe(2)

        await userEvent.click(screen.getByRole('button', { name: 'Mark all as read' }))
        expect(selectUnreadCount(store.getState())).toBe(0)
    })

    it('dismisses a single notification', async () => {
        const { store } = renderWithStore(<NotificationPanel onClose={vi.fn()} />)
        act(() => {
            store.dispatch(notify({ kind: 'sync', title: 'Synced' }))
        })
        await userEvent.click(screen.getByRole('button', { name: 'Dismiss: Synced' }))
        expect(screen.getByText('No notifications yet')).toBeInTheDocument()
    })

    it('closes via the close button', async () => {
        const onClose = vi.fn()
        renderWithStore(<NotificationPanel onClose={onClose} />)
        await userEvent.click(screen.getByRole('button', { name: 'Close notifications' }))
        expect(onClose).toHaveBeenCalled()
    })
})
