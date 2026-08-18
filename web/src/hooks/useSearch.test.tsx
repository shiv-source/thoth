import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { Provider } from 'react-redux'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../test/mockAxios'
import { type SearchResult } from '../api/client'
import { makeStore } from '../store'
import { useSearch } from './useSearch'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

// The hook dispatches into the search slice, so each render gets a fresh
// store behind a Provider.
function renderSearchHook(initial: string) {
    const store = makeStore()
    const wrapper = ({ children }: { children: ReactNode }) => <Provider store={store}>{children}</Provider>
    return renderHook(({ q }: { q: string }) => useSearch(q), { initialProps: { q: initial }, wrapper })
}

describe('useSearch', () => {
    beforeEach(() => {
        // reset, not clear: clearAllMocks keeps once-implementations queued,
        // which would leak between tests here.
        vi.resetAllMocks()
    })

    it('returns results after the debounce', async () => {
        mocks.get.mockResolvedValue({
            data: { results: [{ path: 'a.md', title: 'A', kind: 'note', snippet: '…' }] }
        })
        const { result } = renderSearchHook('a')
        await waitFor(() => expect(result.current.results[0]?.path).toBe('a.md'))
        expect(result.current.loading).toBe(false)
    })

    it('ignores stale responses from an older query', async () => {
        vi.useFakeTimers()
        let resolveFirst!: (r: { data: { results: SearchResult[] } }) => void
        mocks.get
            .mockReturnValueOnce(
                new Promise<{ data: { results: SearchResult[] } }>((resolve) => {
                    resolveFirst = resolve
                })
            )
            .mockResolvedValueOnce({
                data: { results: [{ path: 'new.md', title: 'New', kind: 'note', snippet: '…' }] }
            })

        const { result, rerender } = renderSearchHook('first')
        act(() => {
            vi.advanceTimersByTime(300)
        }) // first query fires its request
        expect(mocks.get).toHaveBeenCalledTimes(1)
        rerender({ q: 'second' })
        act(() => {
            vi.advanceTimersByTime(300)
        }) // second query fires its request
        expect(mocks.get).toHaveBeenCalledTimes(2)
        await act(async () => {
            await Promise.resolve()
        }) // newest response lands
        expect(result.current.results).toEqual([expect.objectContaining({ path: 'new.md' })])

        // The older request resolves last: its payload must be dropped.
        await act(async () => {
            await Promise.resolve()
            resolveFirst({
                data: { results: [{ path: 'stale.md', title: 'Stale', kind: 'note', snippet: '…' }] }
            })
        })
        await act(async () => {
            await Promise.resolve()
        })
        expect(result.current.results).toEqual([expect.objectContaining({ path: 'new.md' })])

        vi.useRealTimers()
    })

    it('aborts the superseded request', () => {
        vi.useFakeTimers()
        mocks.get
            .mockImplementationOnce(() => new Promise(() => {}))
            .mockResolvedValueOnce({
                data: { results: [{ path: 'new.md', title: 'New', kind: 'note', snippet: '…' }] }
            })

        const { rerender } = renderSearchHook('first')
        act(() => {
            vi.advanceTimersByTime(300)
        }) // first query fires its request
        rerender({ q: 'second' })

        // The cleanup of the first effect must abort the in-flight request.
        const [, config] = mocks.get.mock.calls[0] as [string, { signal: AbortSignal } | undefined]
        expect(config?.signal.aborted).toBe(true)
        vi.useRealTimers()
    })

    it('aborts the request on unmount', () => {
        vi.useFakeTimers()
        mocks.get.mockImplementationOnce(() => new Promise(() => {}))

        const { unmount } = renderSearchHook('query')
        act(() => {
            vi.advanceTimersByTime(300)
        }) // request fires
        unmount()

        const [, config] = mocks.get.mock.calls[0] as [string, { signal: AbortSignal } | undefined]
        expect(config?.signal.aborted).toBe(true)
        vi.useRealTimers()
    })

    it('clears results and loading when the query is emptied', async () => {
        vi.useFakeTimers()
        let resolveFirst!: (r: { data: { results: SearchResult[] } }) => void
        mocks.get.mockImplementationOnce(
            () =>
                new Promise<{ data: { results: SearchResult[] } }>((resolve) => {
                    resolveFirst = resolve
                })
        )

        const { result, rerender } = renderSearchHook('first')
        act(() => {
            vi.advanceTimersByTime(300)
        }) // request fires, loading is true
        rerender({ q: '' })
        expect(result.current.loading).toBe(false)
        expect(result.current.results).toEqual([])

        // A late response must not resurrect results: the cleared query
        // makes the slice drop it.
        await act(async () => {
            await Promise.resolve()
            resolveFirst({
                data: { results: [{ path: 'stale.md', title: 'Stale', kind: 'note', snippet: '…' }] }
            })
        })
        await act(async () => {
            await Promise.resolve()
        })
        expect(result.current.results).toEqual([])

        vi.useRealTimers()
    })
})
