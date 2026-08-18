import { act, fireEvent, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { notify, selectNotificationsOpen } from '../store'
import { renderWithStore } from '../test/renderWithStore'
import { AppHeader } from './AppHeader'

describe('AppHeader', () => {
    it('renders the title and the notification bell', () => {
        renderWithStore(<AppHeader title="Notes" onOpenSettings={vi.fn()} />)
        expect(screen.getByText('Notes')).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Notifications' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Settings' })).toBeInTheDocument()
    })

    it('omits the settings gear when no handler is given', () => {
        renderWithStore(<AppHeader title="Settings" />)
        expect(screen.queryByRole('button', { name: 'Settings' })).not.toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Notifications' })).toBeInTheDocument()
    })

    it('shows the unread badge and opens the panel', async () => {
        const { store } = renderWithStore(<AppHeader title="Chat" onOpenSettings={vi.fn()} />)
        act(() => {
            store.dispatch(notify({ kind: 'sync', title: 'Git synced' }))
        })
        expect(screen.getByTitle('1 unread')).toBeInTheDocument()

        await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
        expect(await screen.findByRole('dialog', { name: 'Notifications' })).toBeInTheDocument()
        expect(screen.getByText('Git synced')).toBeInTheDocument()
    })

    it('closes the panel on Escape', async () => {
        const { store } = renderWithStore(<AppHeader title="Chat" onOpenSettings={vi.fn()} />)
        await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
        expect(await screen.findByRole('dialog', { name: 'Notifications' })).toBeInTheDocument()

        // jsdom never completes rc-motion's leave, so the popup stays
        // mounted; the close is asserted on the ui state that drives it.
        await userEvent.keyboard('{Escape}')
        expect(selectNotificationsOpen(store.getState())).toBe(false)
    })

    it('closes the panel on a press outside, not inside', async () => {
        const { store } = renderWithStore(<AppHeader title="Chat" onOpenSettings={vi.fn()} />)
        await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
        const panel = await screen.findByRole('dialog', { name: 'Notifications' })

        await userEvent.click(panel)
        expect(selectNotificationsOpen(store.getState())).toBe(true)

        // Real browsers fire pointerdown before mousedown; rc-trigger
        // resets its popup-pointer guard on the pointerdown, then closes
        // on the mousedown outside.
        fireEvent.pointerDown(document.body)
        fireEvent.mouseDown(document.body)
        expect(selectNotificationsOpen(store.getState())).toBe(false)
    })
})
