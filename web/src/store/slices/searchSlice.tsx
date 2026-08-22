import { createAsyncThunk, createSlice } from '@reduxjs/toolkit'
import { api, type SearchResult } from '../../api/client'
import type { RootState } from '../index'

// The debounce timer lives in SearchPanel; this thunk carries the request
// (and the AbortSignal via the dispatch options) into the store.
export const searchNotes = createAsyncThunk('search/searchNotes', async (q: string, { signal }) =>
    api.search(q, signal)
)

interface SearchState {
    query: string
    results: SearchResult[] | null
    loading: boolean
    error: string | null
}

const initialState: SearchState = { query: '', results: null, loading: false, error: null }

export const searchSlice = createSlice({
    name: 'search',
    initialState,
    reducers: {
        // Clearing the query drops the results immediately; the pending
        // request is aborted by the caller (useSearch).
        clearSearch(s) {
            s.query = ''
            s.results = null
            s.loading = false
            s.error = null
        }
    },
    extraReducers: (builder) => {
        builder
            .addCase(searchNotes.pending, (s, a) => {
                s.query = a.meta.arg
                s.loading = true
                s.error = null
            })
            .addCase(searchNotes.fulfilled, (s, a) => {
                // A response is only shown if it answers the latest query —
                // this is the staleness guard that makes out-of-order
                // responses harmless.
                if (a.meta.arg !== s.query) return
                s.results = a.payload.results
                s.loading = false
            })
            .addCase(searchNotes.rejected, (s, a) => {
                // A superseded request leaves the latest query in charge.
                if (a.meta.arg !== s.query) return
                s.loading = false
                // Aborts are intentional (typing on) and are not errors.
                if (a.error.name !== 'AbortError') s.error = 'search failed'
            })
    }
})

export const { clearSearch } = searchSlice.actions

export const selectSearchResults = (s: RootState) => s.search.results
export const selectSearchLoading = (s: RootState) => s.search.loading
export const selectSearchError = (s: RootState) => s.search.error
