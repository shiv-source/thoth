import { createAsyncThunk, createSelector, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import {
    api,
    type ModelGroup,
    type ModelInput,
    type Provider,
    type ProviderInput,
    type Settings
} from '../../api/client'
import type { RootState } from '../index'

export const fetchSettings = createAsyncThunk('settings/fetch', async () => api.settings())
export const saveSettings = createAsyncThunk('settings/save', async (next: Settings) => api.saveSettings(next))
export const fetchModels = createAsyncThunk('settings/fetchModels', async () => api.models())
export const createModel = createAsyncThunk('settings/createModel', async (input: ModelInput) => api.createModel(input))
export const updateModel = createAsyncThunk('settings/updateModel', async (arg: { id: number; input: ModelInput }) =>
    api.updateModel(arg.id, arg.input)
)
export const deleteModel = createAsyncThunk('settings/deleteModel', async (id: number) => {
    await api.deleteModel(id)
    return id
})
export const fetchProviders = createAsyncThunk('settings/fetchProviders', async () => api.providers())
export const createProvider = createAsyncThunk('settings/createProvider', async (input: ProviderInput) =>
    api.createProvider(input)
)
export const updateProvider = createAsyncThunk(
    'settings/updateProvider',
    async (arg: { id: number; input: ProviderInput }) => api.updateProvider(arg.id, arg.input)
)
export const deleteProvider = createAsyncThunk('settings/deleteProvider', async (id: number) => {
    await api.deleteProvider(id)
    return id
})

interface SettingsState {
    data: Settings | null
    loading: boolean
    saving: boolean
    error: string | null
    // The model registry as the server sends it: groups sorted by provider
    // A→Z. Mutations refetch it (the server re-groups and re-sorts), so the
    // store never edits the list itself.
    groups: ModelGroup[]
    // The providers table, sorted A→Z by the server. Mutations refetch it.
    providers: Provider[]
}

const initialState: SettingsState = { data: null, loading: true, saving: false, error: null, groups: [], providers: [] }

export const settingsSlice = createSlice({
    name: 'settings',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(fetchSettings.pending, (s) => {
                s.loading = true
                s.error = null
            })
            .addCase(fetchSettings.fulfilled, (s, a: PayloadAction<Settings>) => {
                s.data = a.payload
                s.loading = false
            })
            .addCase(fetchSettings.rejected, (s) => {
                s.loading = false
                s.error = 'could not load settings'
            })
            .addCase(saveSettings.pending, (s) => {
                s.saving = true
                s.error = null
            })
            .addCase(saveSettings.fulfilled, (s, a: PayloadAction<Settings>) => {
                s.data = a.payload
                s.saving = false
            })
            .addCase(saveSettings.rejected, (s) => {
                s.saving = false
                s.error = 'could not save settings'
            })
            .addCase(fetchModels.fulfilled, (s, a: PayloadAction<{ groups: ModelGroup[] }>) => {
                s.groups = a.payload.groups
            })
            .addCase(fetchModels.rejected, (s) => {
                s.groups = []
            })
            .addCase(fetchProviders.fulfilled, (s, a: PayloadAction<{ providers: Provider[] }>) => {
                s.providers = a.payload.providers
            })
            .addCase(fetchProviders.rejected, (s) => {
                s.providers = []
            })
    }
})

export const selectSettings = (s: RootState) => s.settings
// The picker consumes the grouped shape; the models table derives a flat
// list from it (createSelector keeps the derived array reference stable).
export const selectModelGroups = (s: RootState) => s.settings.groups
export const selectModelList = createSelector([selectModelGroups], (groups) => groups.flatMap((g) => g.models))
export const selectProviders = (s: RootState) => s.settings.providers
