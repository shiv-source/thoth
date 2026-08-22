import { describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import { fetchNote, selectNoteContent, selectNoteError, selectNoteLoading } from '../index'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

describe('noteSlice', () => {
    it('loads a note by path', async () => {
        stubAPI(mocks, {
            '/api/v1/notes?path=a.md': () => ({ path: 'a.md', content: 'a body' })
        })
        const store = makeStore()
        await store.dispatch(fetchNote('a.md'))
        expect(selectNoteContent(store.getState())).toBe('a body')
        expect(selectNoteLoading(store.getState())).toBe(false)
    })

    it('discards a stale response when the path changed mid-flight', async () => {
        let resolveFirst!: (v: unknown) => void
        mocks.get
            .mockImplementationOnce(() => new Promise((r) => (resolveFirst = r)))
            .mockImplementationOnce(() => Promise.resolve({ data: { path: 'b.md', content: 'b body' } }))
        const store = makeStore()

        const first = store.dispatch(fetchNote('a.md'))
        const second = store.dispatch(fetchNote('b.md'))
        resolveFirst({ data: { path: 'a.md', content: 'a body' } })
        await Promise.allSettled([first, second])

        expect(selectNoteContent(store.getState())).toBe('b body')
    })

    it('records an error when the note fetch fails', async () => {
        stubAPI(mocks, {})
        const store = makeStore()
        await store.dispatch(fetchNote('missing.md')).catch(() => {})
        expect(selectNoteError(store.getState())).toBe('could not load the note')
    })
})
