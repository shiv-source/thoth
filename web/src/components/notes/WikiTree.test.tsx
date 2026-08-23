import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, axiosError, stubAPI } from '../../test/mockAxios'
import { renderWithStore } from '../../test/renderWithStore'
import { setStatus } from '../../store'
import { WikiTree } from './WikiTree'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

// A fresh store per render backs the wiki and ui slices (tree data,
// expansion, chat-streaming subscription).
function renderWikiTree(openPath: string | null = null) {
    const onOpenNote = vi.fn()
    const utils = renderWithStore(<WikiTree openPath={openPath} onOpenNote={onOpenNote} />)
    return { onOpenNote, ...utils }
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
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('renders the nested wiki structure', async () => {
        stubAPI(mocks, { '/api/v1/wiki/tree': () => treeResponse })
        renderWikiTree()

        expect(screen.getByText('Loading…')).toBeInTheDocument()
        // Folders start collapsed: top-level names visible, files hidden.
        expect(await screen.findByText('meetings')).toBeInTheDocument()
        expect(screen.getByText('todos')).toBeInTheDocument()
        expect(screen.queryByText('standup.md')).not.toBeInTheDocument()
    })

    it('opens the clicked note and marks it selected', async () => {
        stubAPI(mocks, { '/api/v1/wiki/tree': () => treeResponse })
        const { onOpenNote } = renderWikiTree()

        await userEvent.click(await screen.findByText('meetings'))
        await userEvent.click(await screen.findByText('standup.md'))
        expect(onOpenNote).toHaveBeenCalledWith('meetings/standup.md')
    })

    it('collapsing a folder hides its files', async () => {
        stubAPI(mocks, { '/api/v1/wiki/tree': () => treeResponse })
        renderWikiTree()

        await userEvent.click(await screen.findByText('meetings'))
        expect(await screen.findByText('standup.md')).toBeInTheDocument()
        await userEvent.click(screen.getByText('meetings'))
        await waitFor(() => expect(screen.queryByText('standup.md')).not.toBeInTheDocument())
    })

    it('shows the file count in a hover tooltip', async () => {
        stubAPI(mocks, { '/api/v1/wiki/tree': () => treeResponse })
        renderWikiTree()
        await userEvent.hover(await screen.findByText('meetings'))
        expect(await screen.findByRole('tooltip')).toHaveTextContent('1 file')
    })

    it('keeps an unreadable directory visible with a warning tooltip', async () => {
        stubAPI(mocks, {
            '/api/v1/wiki/tree': () => ({
                nodes: [
                    {
                        name: 'locked',
                        path: 'locked',
                        is_dir: true,
                        children: null,
                        error: 'read dir locked: permission denied'
                    },
                    { name: 'open.md', path: 'open.md', is_dir: false, children: null }
                ]
            })
        })
        renderWikiTree()

        // The rest of the tree still renders…
        expect(await screen.findByText('open.md')).toBeInTheDocument()
        // …and the unreadable folder carries the error as a hover tooltip.
        await userEvent.hover(screen.getByText('locked'))
        expect(await screen.findByRole('tooltip')).toHaveTextContent('permission denied')
    })

    it('shows an error state when the tree fetch fails', async () => {
        mocks.get.mockRejectedValueOnce(axiosError(500, 'boom'))
        renderWikiTree()
        expect(await screen.findByText('Could not load the wiki tree')).toBeInTheDocument()
    })

    it('refetches when the connection reconnects', async () => {
        stubAPI(mocks, { '/api/v1/wiki/tree': () => treeResponse })
        const { store } = renderWikiTree()
        await screen.findByText('meetings')
        expect(mocks.get).toHaveBeenCalledTimes(1)

        // The socket dropped and reconnected: wiki_changed frames may have
        // been missed while disconnected, so reseed on the reconnect edge.
        await act(async () => {
            store.dispatch(setStatus('reconnecting'))
            await Promise.resolve()
        })
        await act(async () => {
            store.dispatch(setStatus('connected'))
            await Promise.resolve()
        })
        await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2))
    })

    it('refetches the tree when the window regains focus', async () => {
        stubAPI(mocks, { '/api/v1/wiki/tree': () => treeResponse })
        renderWikiTree()
        await screen.findByText('meetings')
        expect(mocks.get).toHaveBeenCalledTimes(1)

        // The focus handler refetches; keep the event and the response's
        // store update inside act so neither lands after the test body.
        await act(async () => {
            window.dispatchEvent(new Event('focus'))
            await Promise.resolve()
        })
        await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2))
    })
})
