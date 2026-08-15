import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../test/mockAxios'
import { renderWithStore } from '../test/renderWithStore'
import { DashboardView } from './DashboardView'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const conversations = {
    conversations: [
        { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1', title: 'Today chat', created_at: new Date().toISOString() },
        { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2', title: 'Old chat', created_at: new Date().toISOString() }
    ]
}

describe('DashboardView', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        mocks.get.mockResolvedValue({ data: conversations })
        // setSystemTime alone (no fake timers) keeps Date deterministic
        // without breaking waitFor/findBy.
        vi.setSystemTime(new Date('2026-08-15T09:30:00'))
    })

    afterEach(() => {
        vi.useRealTimers()
        window.history.pushState(null, '', '/')
    })

    it('renders the time-based greeting and today date', async () => {
        renderWithStore(<DashboardView onOpenSettings={vi.fn()} />)
        expect(screen.getByText('Good morning')).toBeInTheDocument()
        expect(screen.getByText(/Saturday, August 15/)).toBeInTheDocument()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('renders the mock tiles with their dummy data', async () => {
        renderWithStore(<DashboardView onOpenSettings={vi.fn()} />)
        expect(screen.getByText('3 captures waiting')).toBeInTheDocument()
        expect(screen.getByText('Standup')).toBeInTheDocument()
        expect(screen.getByText('Wire the todos tile to GET /api/todos')).toBeInTheDocument()
        expect(screen.getByText('links/bookmarks.md')).toBeInTheDocument()
        await waitFor(() => expect(mocks.get).toHaveBeenCalled()) // flush the conversations fetch
    })

    it('shows real recent chats and navigates to the chat view on click', async () => {
        renderWithStore(<DashboardView onOpenSettings={vi.fn()} />)
        expect(await screen.findByText('Today chat')).toBeInTheDocument()

        window.history.pushState(null, '', '/dashboard')
        fireEvent.click(screen.getByText('Today chat'))
        expect(window.location.pathname).toBe('/chat/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
    })

    it('routes the quick actions to their views', async () => {
        renderWithStore(<DashboardView onOpenSettings={vi.fn()} />)
        await waitFor(() => expect(mocks.get).toHaveBeenCalled())

        fireEvent.click(screen.getByRole('button', { name: /Ask the wiki/ }))
        expect(window.location.pathname).toBe('/search')

        fireEvent.click(screen.getByRole('button', { name: /New note/ }))
        expect(window.location.pathname).toBe('/notes')

        fireEvent.click(screen.getByRole('button', { name: /New chat/ }))
        expect(window.location.pathname).toBe('/chat')
    })
})
