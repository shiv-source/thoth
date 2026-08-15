import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { notify } from '../store'
import { renderWithStore } from '../test/renderWithStore'
import { NavRail } from './NavRail'

describe('NavRail', () => {
    afterEach(() => {
        window.location.hash = ''
    })

    it('renders every view button and highlights the active one', () => {
        window.location.hash = '#/notes'
        renderWithStore(<NavRail />)
        expect(screen.getByRole('button', { name: 'Chat' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Notes' })).toHaveAttribute('aria-current', 'page')
        expect(screen.getByRole('button', { name: 'Dashboard' })).not.toHaveAttribute('aria-current')
        expect(screen.getByRole('button', { name: 'Search' })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'Notifications' })).toBeInTheDocument()
    })

    it('navigates views through the hash on click', async () => {
        renderWithStore(<NavRail />)
        await userEvent.click(screen.getByRole('button', { name: 'Dashboard' }))
        expect(window.location.hash).toBe('#/dashboard')
        expect(screen.getByRole('button', { name: 'Dashboard' })).toHaveAttribute('aria-current', 'page')
    })

    it('shows the unread badge on the bell and opens the panel', async () => {
        const { store } = renderWithStore(<NavRail />)
        act(() => {
            store.dispatch(notify({ kind: 'sync', title: 'Synced' }))
        })
        expect(screen.getByLabelText('1 unread')).toBeInTheDocument()

        await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
        expect(screen.getByRole('dialog', { name: 'Notifications' })).toBeInTheDocument()
        expect(screen.getByText('Synced')).toBeInTheDocument()
    })
})
