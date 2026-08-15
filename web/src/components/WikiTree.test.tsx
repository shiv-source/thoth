import { useState } from 'react'
import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { assistantStart, turnDone } from '../store'
import { renderWithStore } from '../test/renderWithStore'
import { WikiTree } from './WikiTree'

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

// Harness: WikiTree is controlled from the sidebar now; keep the expanded
// state in the test so interactions behave like production. A fresh store
// per render backs the chat-slice subscription (turn-completion refetch).
function renderWikiTree(openPath: string | null = null) {
    const onOpenNote = vi.fn()
    function Wrapper() {
        const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set())
        return (
            <WikiTree
                openPath={openPath}
                onOpenNote={onOpenNote}
                expandedKeys={expandedKeys}
                onExpandedChange={setExpandedKeys}
            />
        )
    }
    const { store, ...utils } = renderWithStore(<Wrapper />)
    return { onOpenNote, store, ...utils }
}

const treeResponse = {
    nodes: [
        {
            name: 'meetings',
            path: 'meetings',
            is_dir: true,
            children: [{ name: 'standup.md', path: 'meetings/standup.md', is_dir: false, children: null }]
        },
        {
            name: 'todos',
            path: 'todos',
            is_dir: true,
            children: [{ name: 'TODO.md', path: 'todos/TODO.md', is_dir: false, children: null }]
        }
    ]
}

describe('WikiTree', () => {
    it('renders the nested wiki structure', async () => {
        mocks.get.mockResolvedValueOnce({ data: treeResponse })
        renderWikiTree()

        expect(screen.getByText('Loading…')).toBeInTheDocument()
        // Folders start collapsed: top-level names visible, files hidden.
        expect(await screen.findByText('meetings')).toBeInTheDocument()
        expect(screen.getByText('todos')).toBeInTheDocument()
        expect(screen.queryByText('standup.md')).not.toBeInTheDocument()
    })

    it('opens the clicked note and marks it selected', async () => {
        mocks.get.mockResolvedValueOnce({ data: treeResponse })
        const { onOpenNote } = renderWikiTree()

        await userEvent.click(await screen.findByRole('button', { name: 'Expand meetings' }))
        await userEvent.click(screen.getByText('standup.md'))
        expect(onOpenNote).toHaveBeenCalledWith('meetings/standup.md')
    })

    it('collapsing a folder hides its files', async () => {
        mocks.get.mockResolvedValueOnce({ data: treeResponse })
        renderWikiTree()

        await userEvent.click(await screen.findByRole('button', { name: 'Expand meetings' }))
        expect(screen.getByText('standup.md')).toBeInTheDocument()
        await userEvent.click(screen.getByRole('button', { name: 'Collapse meetings' }))
        await waitFor(() => expect(screen.queryByText('standup.md')).not.toBeInTheDocument())
    })

    it('shows the file count in a hover tooltip', async () => {
        mocks.get.mockResolvedValueOnce({ data: treeResponse })
        renderWikiTree()
        await userEvent.hover(await screen.findByText('meetings'))
        expect(await screen.findByRole('tooltip')).toHaveTextContent('1 file')
    })

    it('shows an error state when the tree fetch fails', async () => {
        mocks.get.mockRejectedValueOnce(axiosError(500, 'boom'))
        renderWikiTree()

        expect(await screen.findByText('Could not load the wiki tree')).toBeInTheDocument()
    })

    it('refetches the tree when a chat turn completes', async () => {
        mocks.get.mockClear() // earlier tests' calls would pollute the count
        mocks.get.mockResolvedValue({ data: treeResponse })
        const { store } = renderWikiTree()
        await screen.findByText('meetings')
        expect(mocks.get).toHaveBeenCalledTimes(1)

        // A turn starts and completes — Claude may have saved notes. Separate
        // act() calls so the component observes the streaming true→false edge.
        await act(async () => {
            store.dispatch(assistantStart())
            await Promise.resolve()
        })
        await act(async () => {
            store.dispatch(turnDone('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1'))
            await Promise.resolve()
        })
        await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2))
    })

    it('refetches the tree when the window regains focus', async () => {
        mocks.get.mockClear() // earlier tests' calls would pollute the count
        mocks.get.mockResolvedValue({ data: treeResponse })
        renderWikiTree()
        await screen.findByText('meetings')
        expect(mocks.get).toHaveBeenCalledTimes(1)

        window.dispatchEvent(new Event('focus'))
        await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2))
    })
})
