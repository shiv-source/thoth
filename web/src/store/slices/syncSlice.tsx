import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import {
    api,
    type ConnectInput,
    type Connection,
    type ConnectionUpdateInput,
    type SyncProvider,
    type SyncProviderInput,
    type SyncSnapshot,
    type SyncTarget
} from '../../api/client'
import type { RootState } from '../index'

export const fetchSync = createAsyncThunk(
    'sync/fetchSync',
    async (): Promise<{ providers: SyncProvider[]; connections: Connection[] }> => {
        const [providers, connections] = await Promise.all([api.syncProviders(), api.syncConnections()])
        return { providers: providers.providers, connections: connections.connections }
    }
)
export const connectSync = createAsyncThunk('sync/connectSync', async (input: ConnectInput) => api.connectSync(input))
export const updateSyncConnection = createAsyncThunk(
    'sync/updateSyncConnection',
    async ({ id, input }: { id: number; input: ConnectionUpdateInput }) => api.updateSyncConnection(id, input)
)
export const disconnectSync = createAsyncThunk('sync/disconnectSync', async (id: number) => {
    await api.disconnectSync(id)
    return id
})
export const pushSync = createAsyncThunk(
    'sync/pushSync',
    async (id: number): Promise<{ ok: boolean; error?: string }> => api.pushSync(id)
)
export const setActiveSync = createAsyncThunk('sync/setActiveSync', async (id: number) => {
    await api.setActiveSync(id)
    return id
})
export const fetchSyncTargets = createAsyncThunk(
    'sync/fetchSyncTargets',
    async (id: number): Promise<{ id: number; targets: SyncTarget[] }> => {
        const res = await api.syncTargets(id)
        return { id, targets: res.targets }
    }
)
export const createSyncProvider = createAsyncThunk('sync/createSyncProvider', async (input: SyncProviderInput) =>
    api.createSyncProvider(input)
)
export const updateSyncProvider = createAsyncThunk(
    'sync/updateSyncProvider',
    async ({ id, input }: { id: number; input: SyncProviderInput }) => api.updateSyncProvider(id, input)
)
export const deleteSyncProvider = createAsyncThunk('sync/deleteSyncProvider', async (id: number) => {
    await api.deleteSyncProvider(id)
    return id
})
export const fetchSyncSnapshots = createAsyncThunk(
    'sync/fetchSyncSnapshots',
    async (id: number): Promise<{ id: number; snapshots: SyncSnapshot[] }> => {
        const res = await api.syncSnapshots(id)
        return { id, snapshots: res.snapshots }
    }
)
export const restoreSync = createAsyncThunk(
    'sync/restoreSync',
    async ({ id, key = '' }: { id: number; key?: string }): Promise<{ id: number; files: number }> => {
        const res = await api.restoreSync(id, key)
        return { id, files: res.files }
    }
)

interface SyncState {
    providers: SyncProvider[]
    connections: Connection[]
    // targets is a per-connection cache of selectable sync destinations
    // (repos for git providers).
    targets: Record<number, SyncTarget[]>
    // snapshots is a per-connection cache of restore points (s3/local).
    snapshots: Record<number, SyncSnapshot[]>
    loading: boolean
    connecting: boolean
    pushing: boolean
    restoring: boolean
    error: string | null
}

const initialState: SyncState = {
    providers: [],
    connections: [],
    targets: {},
    snapshots: {},
    loading: false,
    connecting: false,
    pushing: false,
    restoring: false,
    error: null
}

export const syncSlice = createSlice({
    name: 'sync',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(fetchSync.pending, (s) => {
                s.loading = true
            })
            .addCase(
                fetchSync.fulfilled,
                (s, a: PayloadAction<{ providers: SyncProvider[]; connections: Connection[] }>) => {
                    s.providers = a.payload.providers
                    s.connections = a.payload.connections
                    s.loading = false
                }
            )
            .addCase(fetchSync.rejected, (s) => {
                s.loading = false
                s.error = 'could not load sync connections'
            })
            .addCase(connectSync.pending, (s) => {
                s.connecting = true
                s.error = null
            })
            .addCase(connectSync.fulfilled, (s, a: PayloadAction<Connection>) => {
                s.connecting = false
                s.connections = [...s.connections, a.payload]
            })
            .addCase(connectSync.rejected, (s, a) => {
                s.connecting = false
                s.error = a.error.message ?? 'could not connect'
            })
            .addCase(updateSyncConnection.fulfilled, (s, a: PayloadAction<Connection>) => {
                s.connections = s.connections.map((c) => (c.id === a.payload.id ? a.payload : c))
            })
            .addCase(updateSyncConnection.rejected, (s, a) => {
                s.error = a.error.message ?? 'could not update the connection'
            })
            .addCase(disconnectSync.fulfilled, (s, a: PayloadAction<number>) => {
                s.connections = s.connections.filter((c) => c.id !== a.payload)
                delete s.targets[a.payload]
                delete s.snapshots[a.payload]
            })
            .addCase(pushSync.pending, (s) => {
                s.pushing = true
                s.error = null
            })
            .addCase(pushSync.fulfilled, (s, a: PayloadAction<{ ok: boolean; error?: string }>) => {
                s.pushing = false
                s.error = a.payload.ok ? null : (a.payload.error ?? 'could not push the wiki')
            })
            .addCase(pushSync.rejected, (s) => {
                s.pushing = false
                s.error = 'could not reach the server'
            })
            .addCase(setActiveSync.fulfilled, (s, a: PayloadAction<number>) => {
                s.connections = s.connections.map((c) => ({ ...c, active: c.id === a.payload }))
            })
            .addCase(fetchSyncTargets.fulfilled, (s, a: PayloadAction<{ id: number; targets: SyncTarget[] }>) => {
                s.targets[a.payload.id] = a.payload.targets
            })
            .addCase(fetchSyncSnapshots.fulfilled, (s, a: PayloadAction<{ id: number; snapshots: SyncSnapshot[] }>) => {
                s.snapshots[a.payload.id] = a.payload.snapshots
            })
            .addCase(restoreSync.pending, (s) => {
                s.restoring = true
                s.error = null
            })
            .addCase(restoreSync.fulfilled, (s) => {
                s.restoring = false
            })
            .addCase(restoreSync.rejected, (s, a) => {
                s.restoring = false
                s.error = a.error.message ?? 'could not restore the wiki'
            })
            .addCase(createSyncProvider.fulfilled, (s, a: PayloadAction<SyncProvider>) => {
                s.providers = [...s.providers, a.payload]
            })
            .addCase(updateSyncProvider.fulfilled, (s, a: PayloadAction<SyncProvider>) => {
                s.providers = s.providers.map((p) => (p.id === a.payload.id ? a.payload : p))
            })
            .addCase(deleteSyncProvider.fulfilled, (s, a: PayloadAction<number>) => {
                s.providers = s.providers.filter((p) => p.id !== a.payload)
            })
    }
})

export const selectSyncProviders = (s: RootState) => s.sync.providers
export const selectSyncConnections = (s: RootState) => s.sync.connections
export const selectSyncTargets = (id: number) => (s: RootState) => s.sync.targets[id] ?? null
export const selectSyncSnapshots = (id: number) => (s: RootState) => s.sync.snapshots[id] ?? null
export const selectSyncLoading = (s: RootState) => s.sync.loading
export const selectSyncConnecting = (s: RootState) => s.sync.connecting
export const selectSyncPushing = (s: RootState) => s.sync.pushing
export const selectSyncRestoring = (s: RootState) => s.sync.restoring
export const selectSyncError = (s: RootState) => s.sync.error
