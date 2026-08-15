import { act, fireEvent, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { notify } from '../store'
import { renderWithStore } from '../test/renderWithStore'
import { TopBar } from './TopBar'

describe('TopBar', () => {
    it('renders the title and the notification bell', () => {
        renderWithStore(<TopBar title="Notes" onOpenSettings={vi.fn()} />)
        expect(screen.getByText('Notes')).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Notifications' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Settings' })).toBeInTheDocument()
    })

    it('omits the settings gear when no handler is given', () => {
        renderWithStore(<TopBar title="Settings" />)
        expect(screen.queryByRole('button', { name: 'Settings' })).not.toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Notifications' })).toBeInTheDocument()
    })

    it('shows the unread badge and opens the panel', async () => {
        const { store } = renderWithStore(<TopBar title="Chat" onOpenSettings={vi.fn()} />)
        act(() => {
            store.dispatch(notify({ kind: 'sync', title: 'Git synced' }))
        })
        expect(screen.getByLabelText('1 unread')).toBeInTheDocument()

        await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
        expect(screen.getByRole('dialog', { name: 'Notifications' })).toBeInTheDocument()
        expect(screen.getByText('Git synced')).toBeInTheDocument()
    })

    it('closes the panel on Escape', async () => {
        renderWithStore(<TopBar title="Chat" onOpenSettings={vi.fn()} />)
        await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
        expect(screen.getByRole('dialog', { name: 'Notifications' })).toBeInTheDocument()

        await userEvent.keyboard('{Escape}')
        expect(screen.queryByRole('dialog', { name: 'Notifications' })).not.toBeInTheDocument()
    })

    it('closes the panel on a press outside, not inside', async () => {
        renderWithStore(<TopBar title="Chat" onOpenSettings={vi.fn()} />)
        await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
        const panel = screen.getByRole('dialog', { name: 'Notifications' })

        await userEvent.click(panel)
        expect(screen.getByRole('dialog', { name: 'Notifications' })).toBeInTheDocument()

        fireEvent.mouseDown(document.body)
        expect(screen.queryByRole('dialog', { name: 'Notifications' })).not.toBeInTheDocument()
    })
})
