import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Health } from '../../api/client'
import { axiosError, axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import { fetchHealth, selectHealth, selectHealthLoading } from './healthSlice'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const healthy: Health = {
    status: 'ok',
    claude: { found: true, path: '/usr/local/bin/claude' },
    wiki: { path: '~/.thoth/wiki', exists: true },
    version: '1.2.3'
}

describe('healthSlice', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('starts loading with no data', () => {
        const store = makeStore()
        expect(selectHealth(store.getState())).toBeNull()
        expect(selectHealthLoading(store.getState())).toBe(true)
    })

    it('loads the health payload', async () => {
        stubAPI(mocks, { 'GET /api/health': () => healthy })
        const store = makeStore()
        await store.dispatch(fetchHealth())
        expect(selectHealth(store.getState())).toEqual(healthy)
        expect(selectHealthLoading(store.getState())).toBe(false)
    })

    it('clears data and sets an error when the fetch fails', async () => {
        mocks.get.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await store.dispatch(fetchHealth())
        expect(selectHealth(store.getState())).toBeNull()
        expect(store.getState().health.error).toBe('failed to reach the server')
    })
})
