import { beforeEach, describe, expect, it, vi } from 'vitest'
import { axiosError, axiosModuleMock, stubAPI } from '../../test/mockAxios'
import { makeStore } from '../index'
import {
    connectSync,
    createSyncProvider,
    deleteSyncProvider,
    disconnectSync,
    fetchSync,
    fetchSyncSnapshots,
    fetchSyncTargets,
    pushSync,
    restoreSync,
    selectSyncConnections,
    selectSyncError,
    selectSyncLoading,
    selectSyncPushing,
    selectSyncSnapshots,
    selectSyncTargets,
    setActiveSync,
    updateSyncConnection,
    updateSyncProvider
} from './syncSlice'

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }))
vi.mock('axios', () => axiosModuleMock(mocks))

const githubProvider = {
    id: 1,
    slug: 'github',
    name: 'GitHub',
    driver: 'github',
    kind: 'git' as const,
    base_url: 'https://github.com',
    protected: false,
    fields: [{ key: 'token', label: 'GitHub token', secret: true, required: true }],
    connection_count: 1
}
const localProvider = {
    id: 2,
    slug: 'local-backup',
    name: 'Local backup',
    driver: 'local',
    kind: 'local' as const,
    base_url: '',
    protected: true,
    fields: [],
    connection_count: 1
}
const connection = {
    id: 3,
    provider_id: 1,
    provider_slug: 'github',
    provider_name: 'GitHub',
    name: 'home',
    enabled: true,
    protected: false,
    active: true,
    identity: { username: 'octo', display_name: 'Octo Cat' },
    config: { repo_url: 'https://github.com/octo/wiki.git', has_token: true },
    last_synced_at: '2026-08-23T09:00:00Z',
    last_error: '',
    push_history: []
}
const target = {
    full_name: 'octo/wiki',
    url: 'https://github.com/octo/wiki.git',
    private: true,
    description: 'My personal knowledge base'
}

describe('syncSlice', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('starts empty and loading', () => {
        const store = makeStore()
        expect(store.getState().sync).toEqual({
            providers: [],
            connections: [],
            targets: {},
            snapshots: {},
            loading: false,
            connecting: false,
            pushing: false,
            restoring: false,
            error: null
        })
    })

    it('loads providers and connections together', async () => {
        stubAPI(mocks, {
            'GET /api/v1/sync/providers': () => ({ providers: [githubProvider, localProvider] }),
            'GET /api/v1/sync/connections': () => ({ connections: [connection] })
        })
        const store = makeStore()
        const pending = store.dispatch(fetchSync())
        expect(selectSyncLoading(store.getState())).toBe(true)
        await pending
        expect(selectSyncLoading(store.getState())).toBe(false)
        expect(store.getState().sync.providers).toEqual([githubProvider, localProvider])
        expect(selectSyncConnections(store.getState())).toEqual([connection])
    })

    it('reports a load error', async () => {
        mocks.get.mockRejectedValue(axiosError(500, { error: 'boom' }))
        const store = makeStore()
        await store.dispatch(fetchSync())
        expect(selectSyncError(store.getState())).toBe('could not load sync connections')
        expect(selectSyncLoading(store.getState())).toBe(false)
    })

    it('connects and appends the connection', async () => {
        stubAPI(mocks, { 'POST /api/v1/sync/connections': () => connection })
        const store = makeStore()
        const pending = store.dispatch(connectSync({ provider_id: 1, name: 'home', config: { token: 'ghp_x' } }))
        expect(selectSyncPushing(store.getState())).toBe(false)
        await pending
        expect(mocks.post).toHaveBeenCalledWith('/api/v1/sync/connections', {
            provider_id: 1,
            name: 'home',
            config: { token: 'ghp_x' }
        })
        expect(selectSyncConnections(store.getState())).toEqual([connection])
        expect(store.getState().sync.connecting).toBe(false)
    })

    it('sets the error message when connecting fails', async () => {
        mocks.post.mockRejectedValueOnce(axiosError(400, { error: 'github rejected the token' }))
        const store = makeStore()
        await expect(
            store.dispatch(connectSync({ provider_id: 1, name: 'home', config: { token: 'bad' } })).unwrap()
        ).rejects.toThrow('github rejected the token')
        expect(selectSyncError(store.getState())).toBe('github rejected the token')
        expect(store.getState().sync.connecting).toBe(false)
    })

    it('updates a connection in place', async () => {
        const renamed = { ...connection, name: 'work' }
        mocks.put.mockResolvedValueOnce({ data: renamed })
        const store = makeStore()
        store.dispatch(fetchSync.fulfilled({ providers: [], connections: [connection] }, 'test', undefined))
        await store.dispatch(updateSyncConnection({ id: 3, input: { name: 'work' } }))
        expect(selectSyncConnections(store.getState())[0]!.name).toBe('work')
        expect(mocks.put).toHaveBeenCalledWith('/api/v1/sync/connections/3', { name: 'work' })
    })

    it('disconnects and drops the connection from the list', async () => {
        stubAPI(mocks, { 'DELETE /api/v1/sync/connections/3': () => ({ ok: true }) })
        const store = makeStore()
        store.dispatch(fetchSync.fulfilled({ providers: [], connections: [connection] }, 'test', undefined))
        await store.dispatch(disconnectSync(3))
        expect(selectSyncConnections(store.getState())).toEqual([])
        expect(mocks.delete).toHaveBeenCalledWith('/api/v1/sync/connections/3')
    })

    it('pushes and reports success', async () => {
        stubAPI(mocks, { 'POST /api/v1/sync/connections/3/push': () => ({ ok: true }) })
        const store = makeStore()
        const pending = store.dispatch(pushSync(3))
        expect(selectSyncPushing(store.getState())).toBe(true)
        await pending
        expect(selectSyncPushing(store.getState())).toBe(false)
        expect(selectSyncError(store.getState())).toBeNull()
    })

    it('pushes and surfaces a destination error', async () => {
        stubAPI(mocks, { 'POST /api/v1/sync/connections/3/push': () => ({ ok: false, error: 'remote rejected' }) })
        const store = makeStore()
        await store.dispatch(pushSync(3))
        expect(selectSyncError(store.getState())).toBe('remote rejected')
    })

    it('sets the active connection', async () => {
        stubAPI(mocks, { 'POST /api/v1/sync/connections/3/active': () => ({ ok: true }) })
        const store = makeStore()
        store.dispatch(
            fetchSync.fulfilled(
                {
                    providers: [],
                    connections: [
                        { ...connection, active: false },
                        { ...connection, id: 4, active: false }
                    ]
                },
                'test',
                undefined
            )
        )
        await store.dispatch(setActiveSync(3))
        expect(selectSyncConnections(store.getState()).map((c) => c.active)).toEqual([true, false])
        expect(mocks.post).toHaveBeenCalledWith('/api/v1/sync/connections/3/active')
    })

    it('caches the git connection targets per connection', async () => {
        stubAPI(mocks, { 'GET /api/v1/sync/connections/3/targets': () => ({ targets: [target] }) })
        const store = makeStore()
        await store.dispatch(fetchSyncTargets(3))
        expect(selectSyncTargets(3)(store.getState())).toEqual([target])
    })

    it('caches restore snapshots per connection', async () => {
        const snapshot = { key: 'thoth-wiki-20260102-030405.zip', time: '2026-01-02T03:04:05Z' }
        stubAPI(mocks, { 'GET /api/v1/sync/connections/3/snapshots': () => ({ snapshots: [snapshot] }) })
        const store = makeStore()
        await store.dispatch(fetchSyncSnapshots(3))
        expect(selectSyncSnapshots(3)(store.getState())).toEqual([snapshot])
    })

    it('restores the wiki from a snapshot', async () => {
        stubAPI(mocks, { 'POST /api/v1/sync/connections/3/restore': () => ({ files: 4, backup: '/tmp/x' }) })
        const store = makeStore()
        const pending = store.dispatch(restoreSync({ id: 3, key: 'thoth-wiki-20260102-030405.zip' }))
        expect(store.getState().sync.restoring).toBe(true)
        await pending
        expect(store.getState().sync.restoring).toBe(false)
        expect(mocks.post).toHaveBeenCalledWith('/api/v1/sync/connections/3/restore', {
            key: 'thoth-wiki-20260102-030405.zip'
        })
    })

    it('restores the latest snapshot when no key is given', async () => {
        stubAPI(mocks, { 'POST /api/v1/sync/connections/3/restore': () => ({ files: 2, backup: null }) })
        const store = makeStore()
        await store.dispatch(restoreSync({ id: 3 }))
        expect(mocks.post).toHaveBeenCalledWith('/api/v1/sync/connections/3/restore', { key: '' })
    })

    it('surfaces a restore failure', async () => {
        mocks.post.mockRejectedValueOnce(axiosError(400, { error: 'archive is not a wiki' }))
        const store = makeStore()
        await expect(store.dispatch(restoreSync({ id: 3 })).unwrap()).rejects.toThrow('archive is not a wiki')
        expect(store.getState().sync.restoring).toBe(false)
        expect(selectSyncError(store.getState())).toBe('archive is not a wiki')
    })

    it('creates and updates a sync provider', async () => {
        const enterprise = { ...githubProvider, id: 9, name: 'GitHub Enterprise', connection_count: 0 }
        mocks.post.mockResolvedValueOnce({ data: enterprise })
        const store = makeStore()
        await store.dispatch(createSyncProvider({ name: 'GitHub Enterprise', driver: 'github' }))
        expect(store.getState().sync.providers).toEqual([enterprise])

        const renamed = { ...enterprise, name: 'GHE' }
        mocks.put.mockResolvedValueOnce({ data: renamed })
        await store.dispatch(updateSyncProvider({ id: 9, input: { name: 'GHE', driver: 'github' } }))
        expect(store.getState().sync.providers[0]!.name).toBe('GHE')
    })

    it('deletes a sync provider and returns its id', async () => {
        stubAPI(mocks, { 'DELETE /api/v1/sync/providers/9': () => ({ ok: true }) })
        const store = makeStore()
        store.dispatch(fetchSync.fulfilled({ providers: [githubProvider], connections: [] }, 'test', undefined))
        const id = await store.dispatch(deleteSyncProvider(9)).unwrap()
        expect(id).toBe(9)
        expect(mocks.delete).toHaveBeenCalledWith('/api/v1/sync/providers/9')
    })
})
