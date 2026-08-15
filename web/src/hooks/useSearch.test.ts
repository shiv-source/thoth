import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosModuleMock } from '../test/mockAxios'
import { type SearchResult } from '../api/client'
import { useSearch } from './useSearch'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

describe('useSearch', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('returns results after the debounce', async () => {
        mocks.get.mockResolvedValue({
            data: { results: [{ path: 'a.md', title: 'A', kind: 'note', snippet: '…' }] }
        })
        const { result } = renderHook(() => useSearch('a'))
        await waitFor(() => expect(result.current.loading).toBe(false))
        expect(result.current.results[0]?.path).toBe('a.md')
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

        const { result, rerender } = renderHook(({ q }) => useSearch(q), { initialProps: { q: 'first' } })
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
})
