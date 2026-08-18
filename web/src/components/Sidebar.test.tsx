import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Sidebar } from './Sidebar'
import { renderWithStore } from '../test/renderWithStore'

// The client creates its axios instance via axios.create; the mocks are
// hoisted so the (also hoisted) vi.mock factory can close over them.
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))

vi.mock('axios', () => ({
    default: {
        create: () => ({
            get: mocks.get,
            post: mocks.post,
            put: mocks.put,
            delete: mocks.delete
        }),
        isAxiosError: (e: unknown) =>
            !!(e && typeof e === 'object' && (e as { isAxiosError?: boolean }).isAxiosError === true)
    }
}))

// axiosError builds a rejection value shaped like an axios error response.
function axiosError(status: number, body: unknown) {
    return Object.assign(new Error(`${status}`), {
        isAxiosError: true,
        response: { status, statusText: String(status), data: body }
    })
}

// stubAPI wires the mocks to the handlers, keyed by "METHOD /path" (with a
// plain path fallback). Handlers return the response BODY (axios wraps it as
// `{ data }`), so a handler that used to return a Response now returns the
// parsed object directly.
function stubAPI(handlers: Record<string, () => unknown>) {
    const respond = (method: string, url: string) => {
        const make = handlers[`${method} ${url}`] ?? handlers[url]
        if (!make) {
            return Promise.reject(
                Object.assign(new Error(`unhandled ${method} ${url}`), {
                    isAxiosError: true,
                    response: { status: 500, statusText: 'Internal Server Error' }
                })
            )
        }
        return Promise.resolve({ data: make() })
    }
    mocks.get.mockImplementation((url: string) => respond('GET', url))
    mocks.post.mockImplementation((url: string) => respond('POST', url))
    mocks.put.mockImplementation((url: string) => respond('PUT', url))
    mocks.delete.mockImplementation((url: string) => respond('DELETE', url))
    return mocks
}

const today = new Date()
const yesterday = new Date(Date.now() - 86400000)
const older = new Date(Date.now() - 40 * 86400000)
const iso = (d: Date) => d.toISOString()

const conversations = {
    conversations: [
        { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1', title: 'Today chat', created_at: iso(today) },
        { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2', title: 'Yesterday chat', created_at: iso(yesterday) },
        { id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa3', title: 'Old chat', created_at: iso(older) }
    ]
}

function renderSidebar() {
    return renderWithStore(<Sidebar />)
}

describe('Sidebar chats section', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
    })

    it('groups the conversation history by day with dates on hover', async () => {
        stubAPI({
            '/api/conversations': () => conversations
        })
        renderSidebar()

        expect(await screen.findByText('Today')).toBeInTheDocument()
        expect(screen.getByText('Yesterday')).toBeInTheDocument()
        expect(screen.getByText('Older')).toBeInTheDocument()
        expect(screen.getByText('Today chat')).toBeInTheDocument()
        expect(screen.getByText('Old chat')).toBeInTheDocument()
    })

    it('navigates to a conversation when its row is clicked', async () => {
        stubAPI({
            '/api/conversations': () => conversations
        })
        renderSidebar()
        window.history.pushState(null, '', '/notes')
        await userEvent.click(await screen.findByText('Today chat'))
        expect(window.location.pathname).toBe('/chat/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1')
    })

    it('deletes a conversation via the API and removes it from the list', async () => {
        stubAPI({
            '/api/conversations': () => conversations,
            [`DELETE /api/conversations/${conversations.conversations[0]?.id}`]: () => ({ ok: true })
        })
        renderSidebar()
        await userEvent.click(await screen.findByRole('button', { name: 'Delete Today chat' }))
        expect(await screen.findByText('Conversation deleted')).toBeInTheDocument()
        await waitFor(() => expect(screen.queryByText('Today chat')).not.toBeInTheDocument())
        const deleted = mocks.delete.mock.calls.find(([u]) => String(u).includes('/api/conversations/'))
        expect(deleted).toBeDefined()
    })

    it('keeps the conversation and toasts when the delete fails', async () => {
        stubAPI({
            '/api/conversations': () => conversations
        })
        renderSidebar()
        await screen.findByText('Today chat')
        mocks.delete.mockRejectedValueOnce(axiosError(500, 'boom'))
        await userEvent.click(screen.getByRole('button', { name: 'Delete Today chat' }))
        expect(await screen.findByText('Could not delete the conversation')).toBeInTheDocument()
        expect(screen.getByText('Today chat')).toBeInTheDocument()
    })

    it('navigates to the root when New chat is clicked', async () => {
        stubAPI({
            '/api/conversations': () => conversations
        })
        renderSidebar()
        window.history.pushState(null, '', '/dashboard')
        await userEvent.click(await screen.findByRole('button', { name: /New chat/ }))
        expect(window.location.pathname).toBe('/chat')
    })

    it('shows empty and error states', async () => {
        stubAPI({
            '/api/conversations': () => ({ conversations: [] })
        })
        const { unmount } = renderSidebar()
        expect(await screen.findByText(/No conversations yet/)).toBeInTheDocument()
        unmount()

        // The next conversations GET rejects like an axios 500.
        mocks.get.mockRejectedValueOnce(axiosError(500, 'boom'))
        renderSidebar()
        expect(await screen.findByText('Could not load conversations')).toBeInTheDocument()
    })
})
