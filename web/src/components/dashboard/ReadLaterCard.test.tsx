import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { renderWithStore } from '../../test/renderWithStore'
import { ReadLaterCard } from './ReadLaterCard'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const items = [
    { title: 'Go Channels', url: 'https://go.dev/blog/channels', reason: 'docs' },
    { title: 'A long read', url: 'https://example.com/long', reason: '' }
]

describe('ReadLaterCard', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        stubAPI(mocks, {
            'GET /api/v1/capture/read-later': () => ({ items })
        })
    })
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it('lists the queued links with their reasons', async () => {
        renderWithStore(<ReadLaterCard />)
        expect(await screen.findByText('Go Channels')).toBeInTheDocument()
        expect(screen.getByText('docs')).toBeInTheDocument()
        expect(screen.getByText('A long read')).toBeInTheDocument()
        // Titles open the external URL.
        expect(screen.getByText('Go Channels').closest('a')).toHaveAttribute('href', 'https://go.dev/blog/channels')
    })

    it('shows the empty state when nothing is queued', async () => {
        stubAPI(mocks, { 'GET /api/v1/capture/read-later': () => ({ items: [] }) })
        renderWithStore(<ReadLaterCard />)
        expect(await screen.findByText('Nothing queued')).toBeInTheDocument()
    })

    it('bookmarks an item into links/bookmarks.md and clears it from the queue', async () => {
        stubAPI(mocks, {
            'GET /api/v1/capture/read-later': () => ({ items }),
            'POST /api/v1/capture': () => ({ path: 'links/bookmarks.md', title: 'Go Channels', type: 'bookmark' }),
            'DELETE /api/v1/capture/read-later?url=https%3A%2F%2Fgo.dev%2Fblog%2Fchannels': () => ({ ok: true })
        })
        renderWithStore(<ReadLaterCard />)
        await screen.findByText('Go Channels')
        // One bookmark button per queued row — target the first (Go Channels).
        fireEvent.click(screen.getAllByRole('button', { name: 'Bookmark' })[0]!)
        await waitFor(() =>
            expect(mocks.post).toHaveBeenCalledWith('/api/v1/capture', {
                kind: 'bookmark',
                url: 'https://go.dev/blog/channels',
                title: 'Go Channels',
                reason: 'docs'
            })
        )
        expect(mocks.delete).toHaveBeenCalledWith(
            '/api/v1/capture/read-later?url=https%3A%2F%2Fgo.dev%2Fblog%2Fchannels'
        )
    })

    it('clears a duplicate bookmark from the queue anyway', async () => {
        stubAPI(mocks, {
            'GET /api/v1/capture/read-later': () => ({ items }),
            'POST /api/v1/capture': () => {
                throw Object.assign(new Error('conflict'), { isAxiosError: true, response: { status: 409 } })
            },
            'DELETE /api/v1/capture/read-later?url=https%3A%2F%2Fgo.dev%2Fblog%2Fchannels': () => ({ ok: true })
        })
        renderWithStore(<ReadLaterCard />)
        await screen.findByText('Go Channels')
        fireEvent.click(screen.getAllByRole('button', { name: 'Bookmark' })[0]!)
        await waitFor(() => expect(mocks.delete).toHaveBeenCalled())
    })

    it('marks an item done by removing it from the queue', async () => {
        stubAPI(mocks, {
            'GET /api/v1/capture/read-later': () => ({ items }),
            'DELETE /api/v1/capture/read-later?url=https%3A%2F%2Fgo.dev%2Fblog%2Fchannels': () => ({ ok: true })
        })
        renderWithStore(<ReadLaterCard />)
        await screen.findByText('Go Channels')
        fireEvent.click(screen.getByRole('button', { name: /Done Go Channels/ }))
        await waitFor(() => expect(mocks.delete).toHaveBeenCalled())
    })
})
