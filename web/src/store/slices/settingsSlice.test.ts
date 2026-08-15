import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Settings } from '../../api/client'
import { axiosError, axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import { fetchSettings, saveSettings, selectSettings } from './settingsSlice'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const saved: Settings = {
    wiki_path: '~/.thoth/wiki',
    model: 'claude-sonnet-5',
    repo_url: 'git@github.com:me/wiki.git',
    sync_enabled: true
}

describe('settingsSlice', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('starts empty and loading', () => {
        const store = makeStore()
        expect(store.getState().settings).toEqual({ data: null, loading: true, saving: false, error: null })
    })

    it('loads settings', async () => {
        stubAPI(mocks, { 'GET /api/settings': () => saved })
        const store = makeStore()
        await store.dispatch(fetchSettings())
        expect(selectSettings(store.getState()).data).toEqual(saved)
        expect(selectSettings(store.getState()).loading).toBe(false)
    })

    it('sets an error message when the load fails', async () => {
        mocks.get.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await store.dispatch(fetchSettings())
        const s = selectSettings(store.getState())
        expect(s.error).toBe('could not load settings')
        expect(s.loading).toBe(false)
    })

    it('saves settings and marks the in-flight state', async () => {
        stubAPI(mocks, { 'PUT /api/settings': () => saved })
        const store = makeStore()
        const pending = store.dispatch(saveSettings(saved))
        expect(selectSettings(store.getState()).saving).toBe(true)
        await pending
        expect(mocks.put).toHaveBeenCalledWith('/api/settings', saved)
        const s = selectSettings(store.getState())
        expect(s.data).toEqual(saved)
        expect(s.saving).toBe(false)
    })

    it('sets an error message when the save fails', async () => {
        mocks.put.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await expect(store.dispatch(saveSettings(saved)).unwrap()).rejects.toThrow()
        const s = selectSettings(store.getState())
        expect(s.saving).toBe(false)
        expect(s.error).toBe('could not save settings')
    })

    it('clears a previous error when saving again', async () => {
        mocks.put.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await expect(store.dispatch(saveSettings(saved)).unwrap()).rejects.toThrow()
        expect(selectSettings(store.getState()).error).not.toBeNull()

        stubAPI(mocks, { 'PUT /api/settings': () => saved })
        const pending = store.dispatch(saveSettings(saved))
        expect(selectSettings(store.getState()).error).toBeNull()
        await pending
    })
})
