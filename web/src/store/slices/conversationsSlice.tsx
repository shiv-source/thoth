import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { api, type Conversation } from '../../api/client'
import type { RootState } from '../index'

export const fetchConversations = createAsyncThunk('conversations/fetch', async () => api.listConversations())
export const deleteConversation = createAsyncThunk('conversations/delete', async (id: string) => {
    await api.deleteConversation(id)
    return id
})

interface ConversationsState {
    list: Conversation[] | null
    loading: boolean
    error: string | null
}

const initialState: ConversationsState = { list: null, loading: true, error: null }

export const conversationsSlice = createSlice({
    name: 'conversations',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(fetchConversations.pending, (s) => {
                s.loading = true
                s.error = null
            })
            .addCase(fetchConversations.fulfilled, (s, a: PayloadAction<{ conversations: Conversation[] }>) => {
                s.list = a.payload.conversations
                s.loading = false
                s.error = null
            })
            .addCase(fetchConversations.rejected, (s) => {
                s.loading = false
                s.error = 'could not load conversations'
            })
            .addCase(deleteConversation.fulfilled, (s, a: PayloadAction<string>) => {
                if (s.list) s.list = s.list.filter((c) => c.id !== a.payload)
            })
    }
})

export const selectConversations = (s: RootState) => s.conversations
export const selectConversationsLoading = (s: RootState) => s.conversations.loading
