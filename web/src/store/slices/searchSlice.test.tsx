import { describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import { searchNotes, selectSearchError, selectSearchLoading, selectSearchResults } from '../index'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const result = { path: 'a.md', title: 'A note', kind: 'note', snippet: 'marked <mark>snippet</mark>' }

describe('searchSlice', () => {
    it('stores the results of the latest query', async () => {
        stubAPI(mocks, {
            '/api/v1/search?q=thoth': () => ({ results: [result] })
        })
        const store = makeStore()
        await store.dispatch(searchNotes('thoth'))
        expect(selectSearchResults(store.getState())).toEqual([result])
        expect(selectSearchLoading(store.getState())).toBe(false)
    })

    it('ignores a stale response that arrives after a newer query', async () => {
        let resolveFirst!: (v: unknown) => void
        mocks.get
            .mockImplementationOnce(() => new Promise((r) => (resolveFirst = r)))
            .mockImplementationOnce(() => Promise.resolve({ data: { results: [] } }))
        const store = makeStore()

        const first = store.dispatch(searchNotes('old'))
        const second = store.dispatch(searchNotes('new'))
        resolveFirst({ data: { results: [result] } })
        await Promise.allSettled([first, second])

        expect(selectSearchResults(store.getState())).toEqual([])
    })

    it('treats an abort as an intentional stop, not an error', async () => {
        mocks.get.mockImplementation((_url: string, opts?: { signal?: AbortSignal }) => {
            return new Promise((_, reject) => {
                opts?.signal?.addEventListener('abort', () => {
                    const err = new Error('aborted')
                    err.name = 'AbortError'
                    reject(err)
                })
            })
        })
        const store = makeStore()
        const controller = new AbortController()

        const pending = store.dispatch(searchNotes('q', { signal: controller.signal }))
        expect(selectSearchLoading(store.getState())).toBe(true)

        controller.abort()
        await pending.catch(() => {})

        expect(selectSearchLoading(store.getState())).toBe(false)
        expect(selectSearchError(store.getState())).toBeNull()
    })

    it('records an error for a genuine failure of the latest query', async () => {
        stubAPI(mocks, {})
        const store = makeStore()
        await store.dispatch(searchNotes('q')).catch(() => {})
        expect(selectSearchError(store.getState())).toBe('search failed')
    })
})
