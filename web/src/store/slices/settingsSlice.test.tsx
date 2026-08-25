import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { LLMModel, Settings } from '../../api/client'
import { axiosError, axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import {
    createModel,
    createProvider,
    deleteModel,
    deleteProvider,
    fetchModels,
    fetchProviders,
    fetchSettings,
    saveSettings,
    selectModelGroups,
    selectModelList,
    selectProviders,
    selectSettings,
    updateModel,
    updateProvider
} from './settingsSlice'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const saved: Settings = {
    wiki_path: '~/.thoth/wiki',
    wiki_folders: [],
    model: 'claude-sonnet-5',
    context_injection: false,
    conversation_retention_days: 7
}

const llmModel: LLMModel = {
    id: 3,
    value: 'my-model',
    name: 'My Model',
    tag: 'test',
    provider: 'Vendor',
    provider_id: 1
}
const provider = {
    id: 1,
    name: 'Vendor',
    base_url: 'https://api.vendor.example',
    custom_headers: {},
    has_api_key: false,
    model_count: 1
}

describe('settingsSlice', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('starts empty and loading', () => {
        const store = makeStore()
        expect(store.getState().settings).toEqual({
            data: null,
            loading: true,
            saving: false,
            error: null,
            groups: [],
            providers: []
        })
    })

    it('loads settings', async () => {
        stubAPI(mocks, { 'GET /api/v1/settings': () => saved })
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
        stubAPI(mocks, { 'PUT /api/v1/settings': () => saved })
        const store = makeStore()
        const pending = store.dispatch(saveSettings(saved))
        expect(selectSettings(store.getState()).saving).toBe(true)
        await pending
        expect(mocks.put).toHaveBeenCalledWith('/api/v1/settings', saved)
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

        stubAPI(mocks, { 'PUT /api/v1/settings': () => saved })
        const pending = store.dispatch(saveSettings(saved))
        expect(selectSettings(store.getState()).error).toBeNull()
        await pending
    })

    it('loads models through the grouped llm_models shape', async () => {
        stubAPI(mocks, { 'GET /api/v1/models': () => ({ groups: [{ provider: 'Vendor', models: [llmModel] }] }) })
        const store = makeStore()
        await store.dispatch(fetchModels())
        expect(selectModelGroups(store.getState())).toEqual([{ provider: 'Vendor', models: [llmModel] }])
        expect(selectModelList(store.getState())).toEqual([llmModel])
    })

    it('createModel calls the API and leaves the list for the refetch', async () => {
        stubAPI(mocks, { 'GET /api/v1/models': () => ({ groups: [] }) })
        const store = makeStore()
        await store.dispatch(fetchModels())
        mocks.post.mockResolvedValueOnce({ data: llmModel })
        const created = await store.dispatch(createModel({ value: 'my-model', name: 'My Model' })).unwrap()
        expect(created).toEqual(llmModel)
        expect(mocks.post).toHaveBeenCalledWith('/api/v1/models', { value: 'my-model', name: 'My Model' })
        expect(selectModelGroups(store.getState())).toEqual([])
    })

    it('updateModel calls the API', async () => {
        const renamed = { ...llmModel, name: 'Renamed' }
        mocks.put.mockResolvedValueOnce({ data: renamed })
        const store = makeStore()
        await store.dispatch(updateModel({ id: 3, input: { value: 'my-model', name: 'Renamed' } }))
        expect(mocks.put).toHaveBeenCalledWith('/api/v1/models/3', { value: 'my-model', name: 'Renamed' })
    })

    it('deleteModel calls the API and returns the id', async () => {
        mocks.delete.mockResolvedValueOnce({ data: { ok: true } })
        const store = makeStore()
        const id = await store.dispatch(deleteModel(3)).unwrap()
        expect(id).toBe(3)
        expect(mocks.delete).toHaveBeenCalledWith('/api/v1/models/3')
    })

    it('loads providers', async () => {
        stubAPI(mocks, { 'GET /api/v1/providers': () => ({ providers: [provider] }) })
        const store = makeStore()
        await store.dispatch(fetchProviders())
        expect(selectProviders(store.getState())).toEqual([provider])
    })

    it('clears providers on a failed fetch', async () => {
        mocks.get.mockRejectedValueOnce(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await store.dispatch(fetchProviders())
        expect(selectProviders(store.getState())).toEqual([])
    })

    it('createProvider calls the API', async () => {
        mocks.post.mockResolvedValueOnce({ data: provider })
        const store = makeStore()
        const created = await store.dispatch(createProvider({ name: 'Vendor' })).unwrap()
        expect(created).toEqual(provider)
        expect(mocks.post).toHaveBeenCalledWith('/api/v1/providers', { name: 'Vendor' })
    })

    it('updateProvider calls the API', async () => {
        const renamed = { ...provider, name: 'Vendor AI' }
        mocks.put.mockResolvedValueOnce({ data: renamed })
        const store = makeStore()
        await store.dispatch(updateProvider({ id: 1, input: { name: 'Vendor AI' } }))
        expect(mocks.put).toHaveBeenCalledWith('/api/v1/providers/1', { name: 'Vendor AI' })
    })

    it('deleteProvider calls the API and returns the id', async () => {
        mocks.delete.mockResolvedValueOnce({ data: { ok: true } })
        const store = makeStore()
        const id = await store.dispatch(deleteProvider(1)).unwrap()
        expect(id).toBe(1)
        expect(mocks.delete).toHaveBeenCalledWith('/api/v1/providers/1')
    })
})
