import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { api } from '../../api/client'
import type { RootState } from '../index'

export const fetchNote = createAsyncThunk('note/fetchNote', async (path: string) => {
    const note = await api.note(path)
    return { path, content: note.content }
})

interface NoteState {
    // The path currently being fetched/shown; fulfilled results for any
    // other path are discarded so switching notes fast never shows a
    // stale note.
    path: string | null
    content: string | null
    loading: boolean
    error: string | null
}

const initialState: NoteState = { path: null, content: null, loading: false, error: null }

export const noteSlice = createSlice({
    name: 'note',
    initialState,
    reducers: {},
    extraReducers: (builder) => {
        builder
            .addCase(fetchNote.pending, (s, a) => {
                s.path = a.meta.arg
                s.loading = true
                s.error = null
            })
            .addCase(fetchNote.fulfilled, (s, a: PayloadAction<{ path: string; content: string }>) => {
                if (a.payload.path !== s.path) return
                s.content = a.payload.content
                s.loading = false
            })
            .addCase(fetchNote.rejected, (s) => {
                s.loading = false
                s.error = 'could not load the note'
            })
    }
})

export const selectNoteContent = (s: RootState) => s.note.content
export const selectNoteLoading = (s: RootState) => s.note.loading
export const selectNoteError = (s: RootState) => s.note.error
