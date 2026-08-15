import { act, screen } from '@testing-library/react'
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
})
