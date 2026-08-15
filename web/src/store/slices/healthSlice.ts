import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { api, type Health } from '../../api/client'
import type { RootState } from '../index'

export const fetchHealth = createAsyncThunk('health/fetch', async () => api.health())

interface HealthState {
    data: Health | null
    loading: boolean
    error: string | null
}

const initialState: HealthState = { data: null, loading: true, error: null }

export const healthSlice = createSlice({
    name: 'health',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(fetchHealth.pending, (s) => {
                s.loading = true
                s.error = null
            })
            .addCase(fetchHealth.fulfilled, (s, a: PayloadAction<Health>) => {
                s.data = a.payload
                s.loading = false
            })
            .addCase(fetchHealth.rejected, (s) => {
                s.data = null
                s.loading = false
                s.error = 'failed to reach the server'
            })
    }
})

export const selectHealth = (s: RootState) => s.health.data
export const selectHealthLoading = (s: RootState) => s.health.loading
