import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { api, type Settings } from '../../api/client'
import type { RootState } from '../index'

export const fetchSettings = createAsyncThunk('settings/fetch', async () => api.settings())
export const saveSettings = createAsyncThunk('settings/save', async (next: Settings) => api.saveSettings(next))

interface SettingsState {
    data: Settings | null
    loading: boolean
    saving: boolean
    error: string | null
}

const initialState: SettingsState = { data: null, loading: true, saving: false, error: null }

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
    }
})

export const selectSettings = (s: RootState) => s.settings
