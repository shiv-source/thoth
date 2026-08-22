import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../../test/mockAxios'
import { renderWithStore } from '../../test/renderWithStore'
import { SearchPanel } from './SearchPanel'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

function stubSearch() {
    mocks.get.mockResolvedValue({
        data: {
            results: [{ path: 'meetings/a.md', title: 'Standup', kind: 'meeting', snippet: '…<mark>deploy</mark>…' }]
        }
    })
}

describe('SearchPanel', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        localStorage.clear()
    })

    it('renders results for a query', async () => {
        stubSearch()
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
        await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
    })

    it('opens the highlighted note on Enter', async () => {
        stubSearch()
        const onOpen = vi.fn()
        renderWithStore(<SearchPanel onOpen={onOpen} />)
        await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
        await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
        await userEvent.keyboard('{Enter}')
        expect(onOpen).toHaveBeenCalledWith('meetings/a.md')
    })

    it('clears the query on Escape', async () => {
        stubSearch()
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        const input = screen.getByPlaceholderText(/Search your wiki/)
        await userEvent.type(input, 'deploy')
        await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
        await userEvent.keyboard('{Escape}')
        expect(input).toHaveValue('')
        expect(screen.queryByText('Standup')).not.toBeInTheDocument()
    })
})

describe('SearchPanel routing', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
        localStorage.clear()
    })

    it('restores the query from ?q= on mount', () => {
        window.history.pushState(null, '', '/search?q=goroutines')
        stubSearch()
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        expect(screen.getByPlaceholderText(/Search your wiki/)).toHaveValue('goroutines')
    })

    it('keeps the query in the URL while typing (replaceState, no history spam)', async () => {
        window.history.pushState(null, '', '/search')
        stubSearch()
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
        expect(window.location.pathname).toBe('/search')
        expect(window.location.search).toBe('?q=deploy')
    })
})

describe('SearchPanel history', () => {
    afterEach(() => {
        window.history.pushState(null, '', '/')
        localStorage.clear()
    })

    it('saves a committed search and shows it when the box is empty', async () => {
        stubSearch()
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'deploy')
        await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
        await userEvent.keyboard('{Enter}') // opens the result → clears the query

        expect(screen.getByText('Recent searches')).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'deploy' })).toBeInTheDocument()
        expect(JSON.parse(localStorage.getItem('thoth.searchHistory') ?? '[]')).toEqual(['deploy'])
    })

    it('restores a history item on click', async () => {
        localStorage.setItem('thoth.searchHistory', JSON.stringify(['bookmarks', 'renovate']))
        stubSearch()
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        await userEvent.click(screen.getByRole('button', { name: 'bookmarks' }))

        expect(screen.getByPlaceholderText(/Search your wiki/)).toHaveValue('bookmarks')
        expect(window.location.search).toBe('?q=bookmarks')
    })

    it('moves a repeated search to the front instead of duplicating it', async () => {
        localStorage.setItem('thoth.searchHistory', JSON.stringify(['deploy', 'bookmarks']))
        stubSearch()
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        await userEvent.type(screen.getByPlaceholderText(/Search your wiki/), 'bookmarks')
        await waitFor(() => expect(screen.getByText('Standup')).toBeInTheDocument())
        await userEvent.keyboard('{Enter}')

        expect(JSON.parse(localStorage.getItem('thoth.searchHistory') ?? '[]')).toEqual(['bookmarks', 'deploy'])
    })

    it('clears the history', async () => {
        localStorage.setItem('thoth.searchHistory', JSON.stringify(['bookmarks']))
        renderWithStore(<SearchPanel onOpen={() => {}} />)
        await userEvent.click(screen.getByRole('button', { name: 'Clear' }))

        expect(screen.queryByText('Recent searches')).not.toBeInTheDocument()
        expect(JSON.parse(localStorage.getItem('thoth.searchHistory') ?? 'null')).toEqual([])
    })
})
