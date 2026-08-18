import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { api, type DoctorCheck } from '../../api/client'
import type { RootState } from '../index'

export const runDoctor = createAsyncThunk('doctor/runDoctor', async () => api.doctor())

interface DoctorState {
    checks: DoctorCheck[] | null
    running: boolean
    error: string | null
}

const initialState: DoctorState = { checks: null, running: false, error: null }

export const doctorSlice = createSlice({
    name: 'doctor',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(runDoctor.pending, (s) => {
                s.running = true
                s.error = null
            })
            .addCase(runDoctor.fulfilled, (s, a: PayloadAction<{ checks: DoctorCheck[] }>) => {
                s.checks = a.payload.checks
                s.running = false
            })
            .addCase(runDoctor.rejected, (s) => {
                s.running = false
                s.error = 'could not run the doctor checks'
            })
    }
})

export const selectDoctorChecks = (s: RootState) => s.doctor.checks
export const selectDoctorRunning = (s: RootState) => s.doctor.running
export const selectDoctorError = (s: RootState) => s.doctor.error
