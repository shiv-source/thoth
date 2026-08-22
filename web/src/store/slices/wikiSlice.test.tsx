import { describe, expect, it, vi } from 'vitest'
import { axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import { fetchTree, selectWikiError, selectWikiLoading, selectWikiNodes } from '../index'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const node = { name: 'a.md', path: 'a.md', is_dir: false, children: null }

describe('wikiSlice', () => {
    it('loads the tree and marks loading along the way', async () => {
        let resolve!: (v: unknown) => void
        mocks.get.mockImplementation(() => new Promise((r) => (resolve = r)))
        const store = makeStore()

        const pending = store.dispatch(fetchTree())
        expect(selectWikiLoading(store.getState())).toBe(true)

        resolve({ data: { nodes: [node] } })
        await pending

        expect(selectWikiLoading(store.getState())).toBe(false)
        expect(selectWikiNodes(store.getState())).toEqual([node])
    })

    it('records an error when the tree fetch fails', async () => {
        stubAPI(mocks, {})
        const store = makeStore()
        await store.dispatch(fetchTree()).catch(() => {})
        expect(selectWikiError(store.getState())).toBe('could not load the wiki tree')
        expect(selectWikiNodes(store.getState())).toBeNull()
    })
})
