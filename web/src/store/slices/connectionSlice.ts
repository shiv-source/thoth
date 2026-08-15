import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import type { ConnectionStatus } from '../../ws/chat'
import type { RootState } from '../index'

interface ConnectionState {
    status: ConnectionStatus
}

const initialState: ConnectionState = { status: 'connected' }

export const connectionSlice = createSlice({
    name: 'connection',
    initialState,
    reducers: {
        setStatus: (s, a: PayloadAction<ConnectionStatus>) => {
            s.status = a.payload
        }
    }
})

export const { setStatus } = connectionSlice.actions

export const selectConnectionStatus = (s: RootState) => s.connection.status
